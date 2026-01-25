package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"dist_task/internal/engine/executor"
	"dist_task/internal/model"
	"dist_task/internal/repository"
	"dist_task/pkg/logger"
	"dist_task/pkg/taskdef"
)

type FlowTask struct {
	ID          string          `json:"id"`
	TaskName    string          `json:"task_name"`
	Description string          `json:"description"`
	DependsOn   []string        `json:"depends_on"`
	Config      json.RawMessage `json:"config"`
	Retry       *RetryConfig    `json:"retry,omitempty"`
}

type RetryConfig struct {
	Strategy    string `json:"strategy"`               // manual / auto / no_retry
	MaxAttempts int    `json:"max_attempts,omitempty"` // 最大重试次数
	Interval    int    `json:"interval,omitempty"`     // 重试间隔（秒）
}

type FlowDefinition struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Tasks       []FlowTask `json:"tasks"`
}

type Engine struct {
	instanceRepo    *repository.InstanceRepository
	taskRepo        *repository.TaskRepository
	exceptionRepo   *repository.ExceptionRepository
	logRepo         *repository.LogRepository
	executorFactory *executor.ExecutorFactory
}

func NewEngine(
	instanceRepo *repository.InstanceRepository,
	taskRepo *repository.TaskRepository,
	exceptionRepo *repository.ExceptionRepository,
	logRepo *repository.LogRepository,
	executorFactory *executor.ExecutorFactory,
) *Engine {
	return &Engine{
		instanceRepo:    instanceRepo,
		taskRepo:        taskRepo,
		exceptionRepo:   exceptionRepo,
		logRepo:         logRepo,
		executorFactory: executorFactory,
	}
}

func (e *Engine) Execute(ctx context.Context, instance *model.TaskGroupInstance, flowDef *model.TaskGroupFlow, globalParams map[string]interface{}) error {
	var flowDefinition FlowDefinition
	if err := json.Unmarshal([]byte(flowDef.Definition), &flowDefinition); err != nil {
		return fmt.Errorf("parse flow definition failed: %w", err)
	}

	instance.Status = "running"
	if err := e.instanceRepo.Update(instance); err != nil {
		return err
	}

	taskMap := make(map[string]*FlowTask)
	for i := range flowDefinition.Tasks {
		taskMap[flowDefinition.Tasks[i].ID] = &flowDefinition.Tasks[i]
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(flowDefinition.Tasks))

	for i := range flowDefinition.Tasks {
		wg.Add(1)
		go func(t FlowTask) {
			defer wg.Done()
			if err := e.executeTask(ctx, instance.ID, &t, taskMap, globalParams); err != nil {
				errCh <- err
			}
		}(flowDefinition.Tasks[i])
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		instance.Status = "failed"
		e.instanceRepo.Update(instance)
		return err
	}

	instance.Status = "success"
	now := time.Now()
	instance.CompletedAt = &now
	return e.instanceRepo.Update(instance)
}

func (e *Engine) executeTask(ctx context.Context, groupID string, task *FlowTask, taskMap map[string]*FlowTask, globalParams map[string]interface{}) error {
	taskDef, err := taskdef.GetTaskDefinition(task.TaskName)
	if err != nil {
		return err
	}

	if taskDef == nil {
		return fmt.Errorf("task definition not found: %s", task.TaskName)
	}

	now := time.Now()
	taskRecord := &model.DistTask{
		ID:        fmt.Sprintf("%s_%s", groupID, task.ID),
		GroupID:   groupID,
		Name:      task.Description,
		Type:      taskDef.Type,
		Status:    "running",
		MaxRetry:  3,
		StartedAt: &now,
		Config:    string(task.Config),
	}

	if err := e.taskRepo.Create(taskRecord); err != nil {
		return err
	}

	e.logRepo.Create(&model.ExecutionLog{
		TaskID:  taskRecord.ID,
		GroupID: groupID,
		Action:  "start",
		Message: fmt.Sprintf("task %s started", task.TaskName),
	})

	logger.Info().Str("task_id", taskRecord.ID).Str("task_name", task.TaskName).Msg("task started")

	taskParams, err := e.extractTaskParams(task.TaskName, globalParams)
	if err != nil {
		taskRecord.Status = "failed"
		taskRecord.ErrorMessage = err.Error()
		e.taskRepo.Update(taskRecord)

		e.logRepo.Create(&model.ExecutionLog{
			TaskID:  taskRecord.ID,
			GroupID: groupID,
			Action:  "failed",
			Message: err.Error(),
		})

		return err
	}

	taskConfig, _ := json.Marshal(taskDef.Config)
	mergedConfig := e.mergeConfig(taskConfig, task.Config)

	taskExecutor, err := e.executorFactory.Create(taskDef.Type)
	if err != nil {
		taskRecord.Status = "failed"
		taskRecord.ErrorMessage = err.Error()
		e.taskRepo.Update(taskRecord)
		return err
	}

	if err := taskExecutor.Execute(ctx, mergedConfig, taskParams); err != nil {
		taskRecord.Status = "failed"
		taskRecord.ErrorMessage = err.Error()
		e.taskRepo.Update(taskRecord)

		retryStrategy := "manual"
		maxAttempts := 3
		interval := 60

		if task.Retry != nil {
			retryStrategy = task.Retry.Strategy
			if task.Retry.MaxAttempts > 0 {
				maxAttempts = task.Retry.MaxAttempts
			}
			if task.Retry.Interval > 0 {
				interval = task.Retry.Interval
			}
		}

		nextAt := time.Now().Add(time.Duration(interval) * time.Second)
		e.exceptionRepo.Create(&model.ExceptionRecord{
			GroupID:       groupID,
			GroupName:     task.Description,
			TaskID:        taskRecord.ID,
			TaskName:      task.TaskName,
			ErrorType:     1,
			ErrorMessage:  err.Error(),
			RetryStrategy: retryStrategy,
			RetryMax:      maxAttempts,
			RetryInterval: interval,
			RetryNextAt:   &nextAt,
			OccurredAt:    time.Now(),
		})

		e.logRepo.Create(&model.ExecutionLog{
			TaskID:  taskRecord.ID,
			GroupID: groupID,
			Action:  "failed",
			Message: err.Error(),
		})

		return err
	}

	completedAt := time.Now()
	taskRecord.Status = "success"
	taskRecord.CompletedAt = &completedAt
	e.taskRepo.Update(taskRecord)

	e.logRepo.Create(&model.ExecutionLog{
		TaskID:  taskRecord.ID,
		GroupID: groupID,
		Action:  "success",
		Message: "task completed",
	})

	logger.Info().Str("task_id", taskRecord.ID).Str("task_name", task.TaskName).Msg("task completed")

	return nil
}

func (e *Engine) extractTaskParams(taskName string, globalParams map[string]interface{}) (map[string]interface{}, error) {
	validator := taskdef.NewValidator()

	taskDef, err := taskdef.GetTaskDefinition(taskName)
	if err != nil {
		return nil, err
	}

	if taskDef == nil {
		return globalParams, nil
	}

	if len(taskDef.InputFields) == 0 {
		return globalParams, nil
	}

	taskParams, ok := globalParams[taskName].(map[string]interface{})
	if !ok {
		taskParams = make(map[string]interface{})
	}

	validatedParams, err := validator.Validate(taskDef.InputFields, taskParams)
	if err != nil {
		return nil, err
	}

	return validatedParams, nil
}

func (e *Engine) mergeConfig(baseConfig []byte, taskConfig json.RawMessage) []byte {
	if len(taskConfig) == 0 {
		return baseConfig
	}

	var base map[string]interface{}
	var task map[string]interface{}

	json.Unmarshal(baseConfig, &base)
	json.Unmarshal(taskConfig, &task)

	if base == nil {
		base = make(map[string]interface{})
	}

	for k, v := range task {
		base[k] = v
	}

	result, _ := json.Marshal(base)
	return result
}

func resolvePlaceholder(config string, params map[string]interface{}) string {
	for key, value := range params {
		placeholder := fmt.Sprintf("${input.%s}", key)
		config = strings.ReplaceAll(config, placeholder, fmt.Sprintf("%v", value))
	}
	return config
}

func (e *Engine) RetryTask(ctx context.Context, instance *model.TaskGroupInstance, flow *model.TaskGroupFlow, taskName string, taskConfig string) error {
	taskDef, err := taskdef.GetTaskDefinition(taskName)
	if err != nil {
		return err
	}

	if taskDef == nil {
		return fmt.Errorf("task definition not found: %s", taskName)
	}

	taskID := fmt.Sprintf("%s_retry_%d", instance.ID, time.Now().UnixNano())
	now := time.Now()

	taskRecord := &model.DistTask{
		ID:         taskID,
		GroupID:    instance.ID,
		Name:       taskDef.Name,
		Type:       taskDef.Type,
		Status:     "running",
		MaxRetry:   3,
		RetryCount: 1,
		StartedAt:  &now,
		Config:     taskConfig,
	}

	if err := e.taskRepo.Create(taskRecord); err != nil {
		return err
	}

	e.logRepo.Create(&model.ExecutionLog{
		TaskID:  taskRecord.ID,
		GroupID: instance.ID,
		Action:  "retry",
		Message: fmt.Sprintf("retrying task %s", taskName),
	})

	logger.Info().Str("task_id", taskRecord.ID).Str("task_name", taskName).Msg("task retry started")

	taskExecutor, err := e.executorFactory.Create(taskDef.Type)
	if err != nil {
		taskRecord.Status = "failed"
		taskRecord.ErrorMessage = err.Error()
		e.taskRepo.Update(taskRecord)
		return err
	}

	taskConfigBytes, _ := json.Marshal(taskDef.Config)

	if err := taskExecutor.Execute(ctx, taskConfigBytes, map[string]interface{}{}); err != nil {
		taskRecord.Status = "failed"
		taskRecord.ErrorMessage = err.Error()
		e.taskRepo.Update(taskRecord)

		e.logRepo.Create(&model.ExecutionLog{
			TaskID:  taskRecord.ID,
			GroupID: instance.ID,
			Action:  "failed",
			Message: err.Error(),
		})

		return err
	}

	completedAt := time.Now()
	taskRecord.Status = "success"
	taskRecord.CompletedAt = &completedAt
	e.taskRepo.Update(taskRecord)

	e.logRepo.Create(&model.ExecutionLog{
		TaskID:  taskRecord.ID,
		GroupID: instance.ID,
		Action:  "success",
		Message: "retry completed",
	})

	logger.Info().Str("task_id", taskRecord.ID).Str("task_name", taskName).Msg("task retry completed")

	return nil
}
