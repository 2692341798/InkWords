package rabbitmq

import (
	"encoding/json"

	"github.com/google/uuid"
)

// GenerationRequestedMessage is the stable RabbitMQ envelope for generation tasks.
type GenerationRequestedMessage struct {
	TaskID      uuid.UUID       `json:"task_id"`
	Kind        string          `json:"kind"`
	WorkspaceID *uuid.UUID      `json:"workspace_id,omitempty"`
	Payload     json.RawMessage `json:"payload"`
}

// RoutingKey returns the stable routing key shared by generation producers and consumers.
func (GenerationRequestedMessage) RoutingKey() string {
	return "generation.requested"
}

// TextbookGenerationRequestedMessage shares the generation worker queue but
// authorizes exclusively through WorkspaceID and omits the legacy owner.
type TextbookGenerationRequestedMessage struct {
	TaskID      uuid.UUID       `json:"task_id"`
	Kind        string          `json:"kind"`
	WorkspaceID *uuid.UUID      `json:"workspace_id"`
	Payload     json.RawMessage `json:"payload"`
}

func (TextbookGenerationRequestedMessage) RoutingKey() string {
	return "generation.requested"
}

// ParseRequestedMessage is the stable RabbitMQ envelope for parse tasks.
type ParseRequestedMessage struct {
	TaskID      uuid.UUID       `json:"task_id"`
	Kind        string          `json:"kind"`
	WorkspaceID *uuid.UUID      `json:"workspace_id,omitempty"`
	Payload     json.RawMessage `json:"payload"`
}

// RoutingKey returns the stable routing key shared by parse producers and consumers.
func (ParseRequestedMessage) RoutingKey() string {
	return "parse.requested"
}

// TextbookParseRequestedMessage shares the parser worker queue but authorizes
// exclusively through WorkspaceID and omits the legacy owner.
type TextbookParseRequestedMessage struct {
	TaskID      uuid.UUID       `json:"task_id"`
	Kind        string          `json:"kind"`
	WorkspaceID *uuid.UUID      `json:"workspace_id"`
	Payload     json.RawMessage `json:"payload"`
}

func (TextbookParseRequestedMessage) RoutingKey() string {
	return "parse.requested"
}

// ExportRequestedMessage is the stable RabbitMQ envelope for export tasks.
type ExportRequestedMessage struct {
	TaskID      uuid.UUID       `json:"task_id"`
	Kind        string          `json:"kind"`
	WorkspaceID *uuid.UUID      `json:"workspace_id,omitempty"`
	Payload     json.RawMessage `json:"payload"`
}

// RoutingKey returns the stable routing key shared by export producers and consumers.
func (ExportRequestedMessage) RoutingKey() string {
	return "export.requested"
}

// TextbookVerificationRequestedMessage carries only immutable textbook
// artifact identities and authorizes exclusively through WorkspaceID.
type TextbookVerificationRequestedMessage struct {
	TaskID      uuid.UUID       `json:"task_id"`
	Kind        string          `json:"kind"`
	WorkspaceID *uuid.UUID      `json:"workspace_id,omitempty"`
	Payload     json.RawMessage `json:"payload"`
}

func (TextbookVerificationRequestedMessage) RoutingKey() string {
	return "textbook.verification.requested"
}
