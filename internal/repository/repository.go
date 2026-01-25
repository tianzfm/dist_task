package repository

import (
	"dist_task/internal/model"
	"dist_task/pkg/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var db *gorm.DB

func Init(cfg string) error {
	var err error
	db, err = gorm.Open(mysql.Open(cfg), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Info),
	})
	if err != nil {
		logger.Error().Err(err).Msg("connect database failed")
		return err
	}
	logger.Info().Msg("database connected successfully")
	return nil
}

func GetDB() *gorm.DB {
	return db
}

type FlowRepository struct{}

func (r *FlowRepository) Create(flow *model.TaskGroupFlow) error {
	return db.Create(flow).Error
}

func (r *FlowRepository) GetByID(id string) (*model.TaskGroupFlow, error) {
	var flow model.TaskGroupFlow
	if err := db.First(&flow, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &flow, nil
}

func (r *FlowRepository) List(offset, limit int) ([]model.TaskGroupFlow, int64) {
	var flows []model.TaskGroupFlow
	var total int64

	db.Model(&model.TaskGroupFlow{}).Count(&total)

	db.Offset(offset).Limit(limit).Find(&flows)

	return flows, total
}

func (r *FlowRepository) Update(flow *model.TaskGroupFlow) error {
	return db.Save(flow).Error
}

func (r *FlowRepository) Delete(id string) error {
	return db.Delete(&model.TaskGroupFlow{}, "id = ?", id).Error
}

type InstanceRepository struct{}

func (r *InstanceRepository) Create(instance *model.TaskGroupInstance) error {
	return db.Create(instance).Error
}

func (r *InstanceRepository) GetByID(id string) (*model.TaskGroupInstance, error) {
	var instance model.TaskGroupInstance
	if err := db.First(&instance, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *InstanceRepository) List(offset, limit int) ([]model.TaskGroupInstance, int64) {
	var instances []model.TaskGroupInstance
	var total int64

	db.Model(&model.TaskGroupInstance{}).Count(&total)

	db.Offset(offset).Limit(limit).Order("created_at DESC").Find(&instances)

	return instances, total
}

func (r *InstanceRepository) Update(instance *model.TaskGroupInstance) error {
	return db.Save(instance).Error
}

type TaskRepository struct{}

func (r *TaskRepository) Create(task *model.DistTask) error {
	return db.Create(task).Error
}

func (r *TaskRepository) GetByID(id string) (*model.DistTask, error) {
	var task model.DistTask
	if err := db.First(&task, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *TaskRepository) ListByGroupID(groupID string) ([]model.DistTask, error) {
	var tasks []model.DistTask
	if err := db.Where("group_id = ?", groupID).Order("created_at ASC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *TaskRepository) Update(task *model.DistTask) error {
	return db.Save(task).Error
}

type ExceptionRepository struct{}

func (r *ExceptionRepository) Create(exception *model.ExceptionRecord) error {
	return db.Create(exception).Error
}

func (r *ExceptionRepository) List(offset, limit int, handled *bool) ([]model.ExceptionRecord, int64) {
	var exceptions []model.ExceptionRecord
	var total int64

	query := db.Model(&model.ExceptionRecord{})
	if handled != nil {
		query = query.Where("handled = ?", *handled)
	}

	query.Count(&total)

	query.Offset(offset).Limit(limit).Order("occurred_at DESC").Find(&exceptions)

	return exceptions, total
}

func (r *ExceptionRepository) Update(exception *model.ExceptionRecord) error {
	return db.Save(exception).Error
}

type LogRepository struct{}

func (r *LogRepository) Create(log *model.ExecutionLog) error {
	return db.Create(log).Error
}

func (r *LogRepository) ListByTaskID(taskID string) ([]model.ExecutionLog, error) {
	var logs []model.ExecutionLog
	if err := db.Where("task_id = ?", taskID).Order("created_at ASC").Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

func (r *LogRepository) ListByGroupID(groupID string) ([]model.ExecutionLog, error) {
	var logs []model.ExecutionLog
	if err := db.Where("group_id = ?", groupID).Order("created_at ASC").Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
