package textbook

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// lookupImportedSnapshot preserves old reconciliation while allowing a new
// parser identity to retain a separate interpretation of the same raw bytes.
// Task creation already deduplicates current imports by their versioned input
// hash; retries retain the original frozen SnapshotID.
func lookupImportedSnapshot(tx *gorm.DB, sourceID uuid.UUID, payload sharedtextbook.SourceImportTaskPayload) (SourceSnapshot, error) {
	var snapshot SourceSnapshot
	query := tx.Where("id = ? AND source_id = ?", payload.SnapshotID, sourceID)
	if payload.TaskVersion == 2 {
		query = tx.Where("source_id = ? AND resolved_version = ? AND content_hash = ?", sourceID, payload.ResolvedVersion, strings.TrimPrefix(payload.ContentHash, "sha256:"))
	}
	err := query.First(&snapshot).Error
	return snapshot, err
}
