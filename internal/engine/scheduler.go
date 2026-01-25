package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"dist_task/internal/model"
	"dist_task/internal/repository"
	"dist_task/pkg/logger"
)

type FlowTask struct {
	ID          string          `json:"id"`
	TaskName    string          `json:"task_name"`
	Description string          `json:"description"`
	DependsOn   []string        `json:"depends_on"`
	Config      json.RawMessage `json:"config"`
}

type FlowDefinition struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Tasks       []FlowTask `json:"tasks"`
}

type Engine struct {
	instanceRepo  *repository.InstanceRepository
	taskRepo      *repository.TaskRepository
	exceptionRepo *repository.ExceptionRepository
	logRepo       *repository.LogRepository
}

func NewEngine(
	instanceRepo *repository.InstanceRepository,
	taskRepo *repository.TaskRepository,
	exceptionRepo *repository.ExceptionRepository,
	logRepo *repository.LogRepository,
) *Engine {
	return &Engine{
		instanceRepo:  instanceRepo,
		taskRepo:      taskRepo,
		exceptionRepo: exceptionRepo,
		logRepo:       logRepo,
	}
}

func (e *Engine) Execute(ctx context.Context, instance *model.TaskGroupInstance, flowDef *model.TaskGroupFlow) error {
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
			if err := e.executeTask(ctx, instance.ID, &t, taskMap, instance.FlowID); err != nil {
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
	return e.instanceRepo.Update(instance)
}

func (e *Engine) executeTask(ctx context.Context, groupID string, task *FlowTask, taskMap map[string]*FlowTask, flowID string) error {
	taskDef, err := GetTaskDefinition(task.TaskName)
	if err != nil {
		return err
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

	return nil
}

func resolvePlaceholder(config string, params map[string]interface{}) string {
	for key, value := range params {
		placeholder := fmt.Sprintf("${input.%s}", key)
		config = strings.ReplaceAll(config, placeholder, fmt.Sprintf("%v", value))
	}
	return config
}
