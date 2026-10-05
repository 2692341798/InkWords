package textbook

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// ApprovedRevisionProjections contains read-only views derived from exactly
// one approved manuscript revision. It is never persisted as another content
// source, so a projection failure cannot alter the manuscript.
type ApprovedRevisionProjections struct {
	WorkspaceID  uuid.UUID                             `json:"workspace_id"`
	Blog         sharedtextbook.BlogProjection         `json:"blog"`
	VideoRunbook sharedtextbook.VideoRunbookProjection `json:"video_runbook"`
	Learning     sharedtextbook.LearningProjection     `json:"learning"`
}

// BuildApprovedRevisionProjections is the only conversion from a textbook
// manuscript to its blog, filming, and learning views. It rejects candidate,
// draft, malformed, or unsupported documents instead of guessing from prose.
func BuildApprovedRevisionProjections(chapter Chapter, revision ChapterRevision) (ApprovedRevisionProjections, error) {
	if chapter.ID == uuid.Nil || revision.ID == uuid.Nil || revision.ChapterID != chapter.ID || revision.Kind != sharedtextbook.RevisionKindApproved {
		return ApprovedRevisionProjections{}, ErrInvalidState
	}
	var document struct {
		Format string `json:"format"`
		Sample struct {
			LearningArc sharedtextbook.LearningArc  `json:"learning_arc"`
			PracticeSet *sharedtextbook.PracticeSet `json:"practice_set,omitempty"`
			EvidenceIDs []string                    `json:"evidence_ids"`
		} `json:"sample"`
		VideoRunbook sharedtextbook.VideoRunbookProjection `json:"video_runbook"`
	}
	if err := json.Unmarshal(revision.DocumentJSON, &document); err != nil || document.Format != "inkwords.textbook.sample.v1" {
		return ApprovedRevisionProjections{}, fmt.Errorf("%w: approved manuscript projection data is invalid", ErrInvalidState)
	}
	contentHash := revision.ContentHash
	if !strings.HasPrefix(contentHash, "sha256:") {
		contentHash = "sha256:" + contentHash
	}
	blog := sharedtextbook.BlogProjection{Format: "inkwords.blog-projection.v1", RevisionID: revision.ID.String(), ChapterID: chapter.ID.String(), Title: chapter.Title, Markdown: revision.Markdown, ContentHash: contentHash}
	if err := blog.Validate(); err != nil {
		return ApprovedRevisionProjections{}, fmt.Errorf("%w: blog projection: %w", ErrInvalidState, err)
	}
	if err := document.VideoRunbook.Validate(); err != nil {
		return ApprovedRevisionProjections{}, fmt.Errorf("%w: video runbook projection: %w", ErrInvalidState, err)
	}
	learning := sharedtextbook.LearningProjection{
		Format:      "inkwords.learning-projection.v1",
		RevisionID:  revision.ID.String(),
		ChapterID:   chapter.ID.String(),
		ContentHash: contentHash,
		LearningArc: document.Sample.LearningArc,
		Objectives: []sharedtextbook.LearningObjective{{
			ID:            revision.ID.String() + ":mastery",
			ChapterID:     chapter.ID.String(),
			Text:          "完成《" + chapter.Title + "》的解释、补全、复现、迁移、诊断和延迟保持练习。",
			RequiredModes: []sharedtextbook.LearningTaskMode{sharedtextbook.LearningTaskExplain, sharedtextbook.LearningTaskComplete, sharedtextbook.LearningTaskReproduce, sharedtextbook.LearningTaskTransfer, sharedtextbook.LearningTaskDiagnose, sharedtextbook.LearningTaskRetain},
		}},
	}
	if document.Sample.PracticeSet != nil {
		if err := document.Sample.PracticeSet.Validate(document.Sample.EvidenceIDs); err != nil || !strings.Contains(revision.Markdown, sharedtextbook.RenderPracticeSet(*document.Sample.PracticeSet)) {
			return ApprovedRevisionProjections{}, fmt.Errorf("%w: practice tasks do not match approved manuscript", ErrInvalidState)
		}
		learning.Format = "inkwords.learning-projection.v2"
		learning.PracticeSet = document.Sample.PracticeSet
		learning.EvidenceIDs = append([]string(nil), document.Sample.EvidenceIDs...)
	}
	if err := learning.Validate(); err != nil {
		return ApprovedRevisionProjections{}, fmt.Errorf("%w: learning projection: %w", ErrInvalidState, err)
	}
	return ApprovedRevisionProjections{Blog: blog, VideoRunbook: document.VideoRunbook, Learning: learning}, nil
}
