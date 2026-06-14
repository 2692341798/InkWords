package task

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// JobTaskStatus represents the lifecycle state of an async task.
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

// JobTask is core-api's projection of the job_tasks table.
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

func (JobTask) TableName() string {
	return "job_tasks"
}

// BeforeCreate keeps task UUID generation independent from database extensions.
func (t *JobTask) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// JobTaskEvent records stream events and status changes for a task.
type JobTaskEvent struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"task_id"`
	EventType string         `gorm:"type:varchar(32);not null;index" json:"event_type"`
	Status    JobTaskStatus  `gorm:"type:varchar(16);not null;index" json:"status"`
	Payload   datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}

func (JobTaskEvent) TableName() string {
	return "job_task_events"
}

type blogRecord struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey"`
	UserID      uuid.UUID      `gorm:"type:uuid;index:idx_user_parent_chapter;not null"`
	ParentID    *uuid.UUID     `gorm:"type:uuid;index:idx_user_parent_chapter"`
	ChapterSort int            `gorm:"type:integer;index:idx_user_parent_chapter"`
	Title       string         `gorm:"type:varchar(255);not null"`
	Content     string         `gorm:"type:text;not null"`
	SourceType  string         `gorm:"type:varchar(50);not null"`
	SourceURL   string         `gorm:"type:varchar(512)"`
	IsSeries    bool           `gorm:"type:boolean;default:false"`
	Status      int16          `gorm:"type:smallint;default:0"`
	WordCount   int            `gorm:"type:integer;default:0"`
	TechStacks  datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (blogRecord) TableName() string {
	return "blogs"
}

type userRecord struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	Username   string    `gorm:"type:varchar(255);not null"`
	Email      string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	TokensUsed int       `gorm:"type:integer;default:0"`
	TokenLimit int       `gorm:"type:integer;default:1000000000"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (userRecord) TableName() string {
	return "users"
}
