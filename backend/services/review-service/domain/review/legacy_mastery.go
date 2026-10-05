package review

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// LegacyMasteryAdapter is an application-owned bridge. The review domain owns
// note discovery, while the adapter owns the target workspace/objective model.
type LegacyMasteryAdapter interface {
	MigrateLegacyNote(context.Context, uuid.UUID, ReviewNote) (objectiveID uuid.UUID, created bool, err error)
}

type LegacyNoteMigrationResponse struct {
	ObjectiveID uuid.UUID `json:"objective_id"`
	Created     bool      `json:"created"`
	NotePath    string    `json:"note_path"`
}

// MigrateLegacyNote reads the note from the configured source rather than
// trusting client-supplied prose, then delegates the mapping to the target
// workspace adapter. It neither edits the source note nor changes old review
// session ownership.
func (service *Service) MigrateLegacyNote(ctx context.Context, workspaceID uuid.UUID, notePath string) (LegacyNoteMigrationResponse, error) {
	if service == nil || service.legacyMastery == nil || workspaceID == uuid.Nil {
		return LegacyNoteMigrationResponse{}, fmt.Errorf("legacy note migration is unavailable")
	}
	note, err := service.findNoteByPath(ctx, strings.TrimSpace(notePath))
	if err != nil {
		return LegacyNoteMigrationResponse{}, err
	}
	objectiveID, created, err := service.legacyMastery.MigrateLegacyNote(ctx, workspaceID, note)
	if err != nil {
		return LegacyNoteMigrationResponse{}, fmt.Errorf("migrate legacy review note: %w", err)
	}
	return LegacyNoteMigrationResponse{ObjectiveID: objectiveID, Created: created, NotePath: note.NotePath}, nil
}
