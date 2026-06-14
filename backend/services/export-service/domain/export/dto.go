package export

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RequestedMessage is the export-service view of the shared export RabbitMQ envelope.
type RequestedMessage struct {
	TaskID  uuid.UUID       `json:"task_id"`
	Kind    string          `json:"kind"`
	UserID  uuid.UUID       `json:"user_id"`
	Payload json.RawMessage `json:"payload"`
}

// TaskResult describes controlled download metadata stored in task result_json.
type TaskResult struct {
	FileToken   string    `json:"file_token"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	ExpiresAt   time.Time `json:"expires_at"`
}
