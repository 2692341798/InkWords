package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// JobTaskStatus 表示后台任务当前所处的生命周期状态。
type JobTaskStatus string

const (
	JobTaskStatusPending   JobTaskStatus = "pending"
	JobTaskStatusQueued    JobTaskStatus = "queued"
	JobTaskStatusRunning   JobTaskStatus = "running"
	JobTaskStatusStreaming JobTaskStatus = "streaming"
	JobTaskStatusSucceeded JobTaskStatus = "succeeded"
	JobTaskStatusFailed    JobTaskStatus = "failed"
	JobTaskStatusCancelled JobTaskStatus = "cancelled"
)

// JobTask 记录一条生成链路的异步任务主状态。
type JobTask struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	TaskType       string         `gorm:"type:varchar(32);not null;index" json:"task_type"`
	TaskSubtype    string         `gorm:"type:varchar(64);not null;index" json:"task_subtype"`
	Status         JobTaskStatus  `gorm:"type:varchar(16);not null;index" json:"status"`
	RequestedBy    uuid.UUID      `gorm:"type:uuid;not null;index" json:"requested_by"`
	IdempotencyKey string         `gorm:"type:varchar(255);index" json:"idempotency_key"`
	PayloadJSON    datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"payload_json"`
	ResultJSON     datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"result_json"`
	ErrorMessage   string         `gorm:"type:text" json:"error_message"`
	RetryCount     int            `gorm:"type:integer;not null;default:0" json:"retry_count"`
	StartedAt      *time.Time     `json:"started_at"`
	FinishedAt     *time.Time     `json:"finished_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// BeforeCreate 在插入数据库前自动生成 UUID，避免依赖数据库扩展来生成主键。
func (t *JobTask) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}

	return nil
}

// JobTaskEvent 记录任务流式执行过程中的状态变化与事件载荷。
type JobTaskEvent struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"task_id"`
	EventType string         `gorm:"type:varchar(32);not null;index" json:"event_type"`
	Status    JobTaskStatus  `gorm:"type:varchar(16);not null;index" json:"status"`
	Payload   datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}
