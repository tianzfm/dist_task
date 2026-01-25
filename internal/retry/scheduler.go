package retry

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"

	"dist_task/internal/engine"
	"dist_task/internal/model"
	"dist_task/internal/repository"
	"dist_task/pkg/logger"
)

type RetryScheduler struct {
	exceptionRepo *repository.ExceptionRepository
	engine        *engine.Engine
	interval      time.Duration
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

func NewRetryScheduler(exceptionRepo *repository.ExceptionRepository, eng *engine.Engine, intervalSeconds int) *RetryScheduler {
	return &RetryScheduler{
		exceptionRepo: exceptionRepo,
		engine:        eng,
		interval:      time.Duration(intervalSeconds) * time.Second,
		stopCh:        make(chan struct{}),
	}
}

func (s *RetryScheduler) Start() {
	s.wg.Add(1)
	go s.run()
	log.Printf("Retry scheduler started with interval: %v", s.interval)
}

func (s *RetryScheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	log.Println("Retry scheduler stopped")
}

func (s *RetryScheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.processRetries()
		}
	}
}

func (s *RetryScheduler) processRetries() {
	exceptions, err := s.exceptionRepo.GetPendingRetry()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get pending retries")
		return
	}

	for _, ex := range exceptions {
		s.processRetry(&ex)
	}
}

func (s *RetryScheduler) processRetry(ex *model.ExceptionRecord) {
	logger.Info().
		Str("exception_id", strconv.FormatInt(ex.ID, 10)).
		Str("task_id", ex.TaskID).
		Int("retry_times", int(ex.RetryTimes)).
		Msg("Processing retry")

	ctx := context.Background()

	instance, err := s.getInstanceByGroupID(ex.GroupID)
	if err != nil {
		logger.Error().Err(err).Str("group_id", ex.GroupID).Msg("Failed to get instance for retry")
		return
	}

	err = s.retryTask(ctx, instance, ex)
	if err != nil {
		logger.Error().Err(err).Str("exception_id", strconv.FormatInt(ex.ID, 10)).Msg("Retry failed")

		if ex.RetryTimes >= ex.RetryMax-1 {
			s.exceptionRepo.MarkRetryComplete(strconv.FormatInt(ex.ID, 10))
			logger.Warn().Str("exception_id", strconv.FormatInt(ex.ID, 10)).Msg("Retry exhausted, marked complete")
		} else {
			s.exceptionRepo.IncrementRetry(strconv.FormatInt(ex.ID, 10))
		}
		return
	}

	s.exceptionRepo.MarkRetryComplete(strconv.FormatInt(ex.ID, 10))
	logger.Info().Str("exception_id", strconv.FormatInt(ex.ID, 10)).Msg("Retry succeeded")
}

func (s *RetryScheduler) getInstanceByGroupID(groupID string) (*model.TaskGroupInstance, error) {
	repo := &repository.InstanceRepository{}
	return repo.GetByID(groupID)
}

func (s *RetryScheduler) retryTask(ctx context.Context, instance *model.TaskGroupInstance, ex *model.ExceptionRecord) error {
	flowRepo := &repository.FlowRepository{}
	flow, err := flowRepo.GetByID(instance.FlowID)
	if err != nil {
		return err
	}

	tasks, err := s.getTasksByGroupID(ex.GroupID)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		if task.ID == ex.TaskID {
			return s.engine.RetryTask(ctx, instance, flow, task.Name, task.Config)
		}
	}

	return nil
}

func (s *RetryScheduler) getTasksByGroupID(groupID string) ([]model.DistTask, error) {
	repo := &repository.TaskRepository{}
	return repo.ListByGroupID(groupID)
}
