package textbook

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type practiceEvidenceRepository interface {
	LoadPracticeEvidence(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (sharedtextbook.PracticeEvidenceProjection, error)
}

// GetPracticeEvidence reads only sources cited by a task in an explicit approved revision.
func (s *Service) GetPracticeEvidence(ctx context.Context, workspaceID, chapterID, revisionID uuid.UUID, taskID string) (sharedtextbook.PracticeEvidenceProjection, error) {
	repository, ok := s.repository.(practiceEvidenceRepository)
	if !ok || workspaceID == uuid.Nil || chapterID == uuid.Nil || revisionID == uuid.Nil || strings.TrimSpace(taskID) == "" || len(taskID) > 400 {
		return sharedtextbook.PracticeEvidenceProjection{}, ErrInvalidState
	}
	return repository.LoadPracticeEvidence(ctx, workspaceID, chapterID, revisionID, taskID)
}

// LoadPracticeEvidence resolves the approved task's citation aliases through
// project-owned imported sources. No browser excerpts are accepted.
func (r *GormRepository) LoadPracticeEvidence(ctx context.Context, workspaceID, chapterID, revisionID uuid.UUID, taskID string) (sharedtextbook.PracticeEvidenceProjection, error) {
	var result sharedtextbook.PracticeEvidenceProjection
	var chapter Chapter
	if err := r.db.WithContext(ctx).Where("id = ?", chapterID).First(&chapter).Error; err != nil {
		return result, ErrNotFound
	}
	if _, err := r.GetProject(ctx, workspaceID, chapter.ProjectID); err != nil {
		return result, err
	}
	var revision ChapterRevision
	if err := r.db.WithContext(ctx).Where("id = ? AND chapter_id = ?", revisionID, chapterID).First(&revision).Error; err != nil {
		return result, ErrNotFound
	}
	projection, err := BuildApprovedRevisionProjections(chapter, revision)
	if err != nil || projection.Learning.PracticeSet == nil {
		return result, ErrInvalidState
	}
	var task *sharedtextbook.PracticeTask
	for index := range projection.Learning.PracticeSet.Tasks {
		if projection.Learning.PracticeSet.Tasks[index].ID == taskID {
			task = &projection.Learning.PracticeSet.Tasks[index]
			break
		}
	}
	if task == nil || len(task.EvidenceIDs) > 40 {
		return result, ErrInvalidState
	}
	var document struct {
		EvidenceAliases map[string]string `json:"evidence_aliases"`
	}
	if json.Unmarshal(revision.DocumentJSON, &document) != nil {
		return result, ErrInvalidState
	}
	chunks := make([]string, 0, len(task.EvidenceIDs))
	resolved, seen := map[string]string{}, map[string]bool{}
	for _, id := range task.EvidenceIDs {
		canonical, aliased := document.EvidenceAliases[id]
		if !aliased {
			canonical = id // Historical approved tasks used canonical IDs directly.
		}
		if !strings.HasPrefix(canonical, "evidence-") || canonical == "evidence-" {
			return result, ErrInvalidState
		}
		chunkID := strings.TrimPrefix(canonical, "evidence-")
		resolved[id] = chunkID
		if !seen[chunkID] {
			chunks = append(chunks, chunkID)
			seen[chunkID] = true
		}
	}
	// A task may cite official supporting sources only; generation's separate
	// primary-source requirement does not apply to this narrow scoring view.
	rows, err := loadSourceEvidenceRows(r.db.WithContext(ctx), chapter.ProjectID, chunks)
	if err != nil {
		return result, err
	}
	byChunk := map[string]generationEvidenceRow{}
	for _, row := range rows {
		byChunk[row.ChunkID] = row
	}
	result = sharedtextbook.PracticeEvidenceProjection{Format: "inkwords.practice-evidence.v1", WorkspaceID: workspaceID.String(), ChapterID: chapterID.String(), RevisionID: revisionID.String(), ContentHash: projection.Learning.ContentHash, TaskID: taskID}
	for _, id := range task.EvidenceIDs {
		row, exists := byChunk[resolved[id]]
		if !exists {
			return sharedtextbook.PracticeEvidenceProjection{}, ErrInvalidState
		}
		var locator sharedtextbook.EvidenceLocator
		if json.Unmarshal(row.ChunkLocator, &locator) != nil {
			return sharedtextbook.PracticeEvidenceProjection{}, ErrInvalidState
		}
		snapshot := sharedtextbook.SourceSnapshot{ID: row.SnapshotID.String(), SourceID: row.SourceID.String(), Kind: row.SourceKind, Role: row.SourceRole, Locator: row.SourceLocator, ResolvedVersion: row.ResolvedVersion, ContentHash: sourceSnapshotDigest(row.SnapshotHash), CapturedAt: row.SnapshotCapturedAt}
		ref := sharedtextbook.EvidenceRef{ID: id, SnapshotID: snapshot.ID, DocumentID: row.DocumentID, ChunkID: row.ChunkID, Locator: locator, ContentHash: row.ChunkTextHash, Confidence: sharedtextbook.EvidenceConfidenceDocumented, SourceRole: snapshot.Role}
		result.Sources = append(result.Sources, sharedtextbook.PracticeSourceExcerpt{Reference: ref, Snapshot: snapshot, Excerpt: row.ChunkSearchText, ExcerptHash: sharedtextbook.PracticeExcerptHash(row.ChunkSearchText)})
	}
	if result.Validate() != nil {
		return sharedtextbook.PracticeEvidenceProjection{}, ErrInvalidState
	}
	return result, nil
}
