package generation

import (
	"context"

	"github.com/google/uuid"
)

// TaskWriter captures the task-table write surface that llm-stream will keep owning.
type TaskWriter interface {
	AppendEvent(ctx context.Context, taskID uuid.UUID, eventType string, payload any) error
	UpdateResult(ctx context.Context, taskID uuid.UUID, result any) error
}

// BlogWriter captures the future hand-off boundary where final business facts move back to core-api.
type BlogWriter interface {
	PersistFinalResult(ctx context.Context, taskID uuid.UUID) error
}

// Service is the service-owned home for generation use cases during the deep split migration.
type Service struct {
	blogWriter BlogWriter
	taskWriter TaskWriter
}

// NewService creates a generation service owned by llm-stream.
func NewService(blogWriter BlogWriter, taskWriter TaskWriter) *Service {
	return &Service{
		blogWriter: blogWriter,
		taskWriter: taskWriter,
	}
}
