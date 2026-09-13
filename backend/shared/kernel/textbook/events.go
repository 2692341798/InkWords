package textbook

import (
	"fmt"
	"strings"
)

// TextbookEvent is a versioned, idempotent worker-to-core-api contract.
type TextbookEvent struct {
	EventVersion int       `json:"event_version"`
	EventID      string    `json:"event_id"`
	EventType    EventType `json:"event_type"`
	ProjectID    string    `json:"project_id"`
	ChapterID    string    `json:"chapter_id,omitempty"`
	Stage        string    `json:"stage"`
	InputHash    string    `json:"input_hash"`
	OutputHash   string    `json:"output_hash,omitempty"`
}

func (event TextbookEvent) Validate() error {
	if event.EventVersion != 1 || strings.TrimSpace(event.EventID) == "" || strings.TrimSpace(event.ProjectID) == "" || strings.TrimSpace(event.Stage) == "" {
		return fmt.Errorf("textbook event identity is incomplete")
	}
	if err := event.EventType.Validate(); err != nil {
		return err
	}
	if !isSHA256Digest(event.InputHash) {
		return fmt.Errorf("textbook event input hash is required")
	}
	if event.OutputHash != "" && !isSHA256Digest(event.OutputHash) {
		return fmt.Errorf("textbook event output hash is invalid")
	}
	return nil
}
