package textbook

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	coretask "inkwords-backend/services/core-api/domain/task"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// GetDependencySourceSnapshot checks an explicit chapter selection against the
// current live project and primary source. It never selects a latest snapshot.
func (r *GormRepository) GetDependencySourceSnapshot(ctx context.Context, workspaceID, projectID, chapterID, snapshotID uuid.UUID) (sharedtextbook.SourceSnapshot, error) {
	if workspaceID == uuid.Nil || projectID == uuid.Nil || chapterID == uuid.Nil || snapshotID == uuid.Nil {
		return sharedtextbook.SourceSnapshot{}, ErrInvalidState
	}
	var row struct {
		SourceSnapshot
		Kind    sharedtextbook.SourceKind
		Role    sharedtextbook.SourceRole
		Locator string
	}
	err := r.db.WithContext(ctx).Table("source_snapshots").
		Select("source_snapshots.*, textbook_sources.kind, textbook_sources.role, textbook_sources.locator").
		Joins("JOIN textbook_sources ON textbook_sources.id = source_snapshots.source_id AND textbook_sources.deleted_at IS NULL").
		Joins("JOIN textbook_projects ON textbook_projects.id = textbook_sources.project_id AND textbook_projects.primary_source_id = textbook_sources.id AND textbook_projects.deleted_at IS NULL").
		Joins("JOIN textbook_chapters ON textbook_chapters.project_id = textbook_projects.id AND textbook_chapters.deleted_at IS NULL").
		Where("source_snapshots.id = ? AND source_snapshots.status = ? AND textbook_projects.workspace_id = ? AND textbook_projects.id = ? AND textbook_chapters.id = ? AND textbook_sources.role = ? AND textbook_sources.kind = ?", snapshotID, "captured", workspaceID, projectID, chapterID, sharedtextbook.SourceRolePrimary, sharedtextbook.SourceKindGitRepository).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return sharedtextbook.SourceSnapshot{}, ErrNotFound
	}
	if err != nil {
		return sharedtextbook.SourceSnapshot{}, err
	}
	snapshot := sharedtextbook.SourceSnapshot{ID: row.ID.String(), SourceID: row.SourceID.String(), Kind: row.Kind, Role: row.Role, Locator: row.Locator, ResolvedVersion: row.ResolvedVersion, ContentHash: canonicalContractHash(row.ContentHash), CapturedAt: row.CapturedAt}
	return snapshot, snapshot.Validate()
}

// ValidateDependencyCandidateSource proves the selected snapshot occurred in
// the candidate's original frozen evidence, not merely somewhere in its project.
func (r *GormRepository) ValidateDependencyCandidateSource(ctx context.Context, revision ChapterRevision, selection sharedtextbook.TeachingDependencySelection) error {
	if selection.Validate() != nil || revision.GenerationTaskID == nil || revision.ChapterID.String() != selection.ChapterID {
		return ErrInvalidState
	}
	var task coretask.JobTask
	if err := r.db.WithContext(ctx).Where("id = ? AND workspace_id = ? AND textbook_chapter_id = ? AND task_subtype = ?", *revision.GenerationTaskID, selection.WorkspaceID, selection.ChapterID, sharedtextbook.TextbookSampleGenerationTaskSubtype).First(&task).Error; err != nil {
		return ErrNotFound
	}
	var payload sharedtextbook.SampleGenerationTaskPayload
	if json.Unmarshal(task.PayloadJSON, &payload) != nil || payload.ProjectID != selection.ProjectID || payload.ChapterID != selection.ChapterID || payload.EvidencePack.Validate() != nil || sharedtextbook.GenerationEvidencePackHash(payload.EvidencePack) != revision.EvidencePackHash {
		return ErrInvalidState
	}
	for _, snapshot := range append([]sharedtextbook.SourceSnapshot{payload.EvidencePack.PrimarySnapshot}, payload.EvidencePack.PrimarySnapshots...) {
		if selection.MatchesSnapshot(snapshot) {
			return nil
		}
	}
	return ErrInvalidState
}
