package task

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"inkwords-backend/internal/model"
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

const (
	taskTypeGeneration      = "generation"
	taskTypeParse           = "parse"
	defaultStreamEventLimit = 200
)

var (
	ErrTaskNotFound     = errors.New("task not found")
	ErrTaskAccessDenied = errors.New("task access denied")
	ErrInvalidTaskInput = errors.New("invalid task input")
	ErrEmptyTaskSubtype = errors.New("task subtype is required")
	ErrEmptyRequestedBy = errors.New("requested_by is required")
)

// CreateGenerationTaskInput 描述创建生成任务时服务层需要的输入。
type CreateGenerationTaskInput struct {
	RequestedBy    uuid.UUID
	TaskSubtype    string
	IdempotencyKey string
	Payload        []byte
}

// CreateParseTaskInput 描述创建解析任务时服务层需要的输入。
type CreateParseTaskInput struct {
	RequestedBy    uuid.UUID
	TaskSubtype    string
	IdempotencyKey string
	Payload        []byte
}

// AppendEventInput 描述写入一条任务事件时的输入。
type AppendEventInput struct {
	EventType string
	Status    model.JobTaskStatus
	Payload   []byte
}

// GenerationRequestedMessage 是任务创建成功后发往消息队列的标准载荷。
type GenerationRequestedMessage = sharedrabbitmq.GenerationRequestedMessage

// ParseRequestedMessage 是解析任务创建成功后发往消息队列的标准载荷。
type ParseRequestedMessage = sharedrabbitmq.ParseRequestedMessage

// ExportRequestedMessage 是导出任务创建成功后发往消息队列的标准载荷。
type ExportRequestedMessage = sharedrabbitmq.ExportRequestedMessage

// CreateGenerationTaskRequest 描述创建生成任务的 HTTP 请求体。
type CreateGenerationTaskRequest struct {
	Kind           string          `json:"kind" binding:"required"`
	Payload        json.RawMessage `json:"payload"`
	IdempotencyKey string          `json:"idempotency_key"`
}

// CreateGenerationTaskResponse 描述创建任务后的 HTTP 响应。
type CreateGenerationTaskResponse struct {
	TaskID    uuid.UUID           `json:"task_id"`
	Status    model.JobTaskStatus `json:"status"`
	StreamURL string              `json:"stream_url"`
}

// TaskResponse 描述任务查询接口返回的任务快照。
type TaskResponse struct {
	ID           uuid.UUID           `json:"id"`
	TaskType     string              `json:"task_type"`
	TaskSubtype  string              `json:"task_subtype"`
	Status       model.JobTaskStatus `json:"status"`
	Result       json.RawMessage     `json:"result,omitempty"`
	ErrorMessage string              `json:"error_message,omitempty"`
	RetryCount   int                 `json:"retry_count"`
	StartedAt    *time.Time          `json:"started_at,omitempty"`
	FinishedAt   *time.Time          `json:"finished_at,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}
