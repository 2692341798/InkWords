package rabbitmq

import (
	"encoding/json"

	"github.com/google/uuid"
)

// GenerationRequestedMessage is the stable RabbitMQ envelope for generation tasks.
type GenerationRequestedMessage struct {
	TaskID  uuid.UUID       `json:"task_id"`
	Kind    string          `json:"kind"`
	UserID  uuid.UUID       `json:"user_id"`
	Payload json.RawMessage `json:"payload"`
}

// RoutingKey returns the stable routing key shared by generation producers and consumers.
func (GenerationRequestedMessage) RoutingKey() string {
	return "generation.requested"
}

// ParseRequestedMessage is the stable RabbitMQ envelope for parse tasks.
type ParseRequestedMessage struct {
	TaskID  uuid.UUID       `json:"task_id"`
	Kind    string          `json:"kind"`
	UserID  uuid.UUID       `json:"user_id"`
	Payload json.RawMessage `json:"payload"`
}

// RoutingKey returns the stable routing key shared by parse producers and consumers.
func (ParseRequestedMessage) RoutingKey() string {
	return "parse.requested"
}

// ExportRequestedMessage is the stable RabbitMQ envelope for export tasks.
type ExportRequestedMessage struct {
	TaskID  uuid.UUID       `json:"task_id"`
	Kind    string          `json:"kind"`
	UserID  uuid.UUID       `json:"user_id"`
	Payload json.RawMessage `json:"payload"`
}

// RoutingKey returns the stable routing key shared by export producers and consumers.
func (ExportRequestedMessage) RoutingKey() string {
	return "export.requested"
}
