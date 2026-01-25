package model

import (
	"time"
)

type TaskGroupFlow struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(64)"`
	Name        string    `json:"name" gorm:"type:varchar(255);not null"`
	Description string    `json:"description" gorm:"type:text"`
	FlowType    string    `json:"flow_type" gorm:"type:varchar(50);not null"`
	Version     int       `json:"version" gorm:"not null;default:1"`
	Definition  string    `json:"definition" gorm:"type:json"`
	IsActive    bool      `json:"is_active" gorm:"not null;default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreateUser  string    `json:"create_user" gorm:"type:varchar(100);not null"`
	UpdatedUser string    `json:"updated_user" gorm:"type:varchar(100);not null"`
}

func (TaskGroupFlow) TableName() string {
	return "task_group_flow"
}

type TaskGroupInstance struct {
	ID          string     `json:"id" gorm:"primaryKey;type:varchar(64)"`
	FlowID      string     `json:"flow_id" gorm:"type:varchar(64);not null"`
	Status      string     `json:"status" gorm:"type:varchar(20);not null;default:pending"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

func (TaskGroupInstance) TableName() string {
	return "task_group_instance"
}

type DistTask struct {
	ID           string     `json:"id" gorm:"primaryKey;type:varchar(64)"`
	GroupID      string     `json:"group_id" gorm:"type:varchar(64);not null"`
	Name         string     `json:"name" gorm:"type:varchar(255);not null"`
	Type         string     `json:"type" gorm:"type:varchar(20);not null"`
	Status       string     `json:"status" gorm:"type:varchar(20);not null;default:pending"`
	MaxRetry     int        `json:"max_retry" gorm:"default:3"`
	RetryCount   int        `json:"retry_count" gorm:"default:0"`
	Config       string     `json:"config" gorm:"type:json"`
	InputData    string     `json:"input_data" gorm:"type:json"`
	OutputData   string     `json:"output_data" gorm:"type:json"`
	ErrorMessage string     `json:"error_message" gorm:"type:text"`
	ErrorStack   string     `json:"error_stack" gorm:"type:text"`
	StartedAt    *time.Time `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
}

func (DistTask) TableName() string {
	return "dist_task"
}

type ExceptionRecord struct {
	ID            int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	GroupID       string     `json:"group_id" gorm:"type:varchar(64);not null"`
	GroupName     string     `json:"group_name" gorm:"type:varchar(255);not null"`
	TaskID        string     `json:"task_id" gorm:"type:varchar(64);not null"`
	TaskName      string     `json:"task_name" gorm:"type:varchar(255);not null"`
	ErrorType     int        `json:"error_type" gorm:"not null"`
	ErrorCode     string     `json:"error_code" gorm:"type:varchar(100)"`
	ErrorMessage  string     `json:"error_message" gorm:"type:text"`
	StackTrace    string     `json:"stack_trace" gorm:"type:text"`
	RetryStrategy string     `json:"retry_strategy" gorm:"type:varchar(50);default:manual"`
	RetryTimes    int        `json:"retry_times" gorm:"default:0"`
	RetryMax      int        `json:"retry_max" gorm:"default:3"`
	RetryInterval int        `json:"retry_interval" gorm:"default:60"`
	RetryNextAt   *time.Time `json:"retry_next_at"`
	Handled       bool       `json:"handled" gorm:"default:false"`
	HandledBy     string     `json:"handled_by" gorm:"type:varchar(100)"`
	HandledAt     *time.Time `json:"handled_at"`
	HandledRemark string     `json:"handled_remark" gorm:"type:text"`
	OccurredAt    time.Time  `json:"occurred_at"`
}

func (ExceptionRecord) TableName() string {
	return "exception_record"
}

type ExecutionLog struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	TaskID    string    `json:"task_id" gorm:"type:varchar(64);not null"`
	GroupID   string    `json:"group_id" gorm:"type:varchar(64);not null"`
	Action    string    `json:"action" gorm:"type:varchar(20);not null"`
	Message   string    `json:"message" gorm:"type:text"`
	Details   string    `json:"details" gorm:"type:json"`
	CreatedAt time.Time `json:"created_at"`
}

func (ExecutionLog) TableName() string {
	return "execution_log"
}
