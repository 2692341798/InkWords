package textbook

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const sourceEvidencePageSize = 200

func validateSourceEvidenceIDs(ids []string) error {
	if len(ids) > sourceEvidencePageSize {
		return fmt.Errorf("%w: too many evidence identifiers", ErrInvalidState)
	}
	for _, id := range ids {
		if strings.TrimSpace(id) == "" || len(id) > 200 {
			return fmt.Errorf("%w: invalid evidence identifier", ErrInvalidState)
		}
	}
	return nil
}

// ListSourceEvidence returns metadata only. Explicit IDs resolve saved blueprint
// selections beyond the initial page without searching or loading source text.
func (r *GormRepository) ListSourceEvidence(ctx context.Context, workspaceID, projectID uuid.UUID, ids ...string) ([]SourceLibraryEvidence, error) {
	if err := validateSourceEvidenceIDs(ids); err != nil {
		return nil, err
	}
	if _, err := r.GetProject(ctx, workspaceID, projectID); err != nil {
		return nil, err
	}
	rows := []SourceLibraryEvidence{}
	query := r.db.WithContext(ctx).Table("source_chunks").
		Select("source_chunks.id, source_chunks.document_id, source_documents.title AS document_title, source_documents.canonical_locator, source_chunks.ordinal, source_chunks.locator").
		Joins("JOIN source_documents ON source_documents.id = source_chunks.document_id").
		Joins("JOIN source_snapshots ON source_snapshots.id = source_documents.snapshot_id").
		Joins("JOIN textbook_sources ON textbook_sources.id = source_snapshots.source_id").
		Where("textbook_sources.project_id = ? AND textbook_sources.deleted_at IS NULL", projectID)
	if len(ids) > 0 {
		query = query.Where("source_chunks.id IN ?", ids)
	}
	err := query.Order("source_documents.created_at ASC, source_chunks.ordinal ASC, source_chunks.id ASC").
		Limit(sourceEvidencePageSize).Scan(&rows).Error
	return rows, err
}
