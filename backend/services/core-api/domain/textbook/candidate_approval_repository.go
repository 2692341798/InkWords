package textbook

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"time"
)

func (r *GormRepository) ApplyCandidate(ctx context.Context, workspaceID uuid.UUID, input ApplyCandidateInput) (*ChapterRevision, error) {
	var approved *ChapterRevision
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var chapter Chapter
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", input.ChapterID).First(&chapter).Error; err != nil {
			return ErrNotFound
		}
		var project Project
		if err := tx.Where("id = ? AND workspace_id = ?", chapter.ProjectID, workspaceID).First(&project).Error; err != nil {
			return ErrNotFound
		}
		if chapter.RevisionVersion != input.ExpectedVersion {
			return ErrVersionConflict
		}
		var lock ChapterLock
		if err := tx.Where("chapter_id = ?", chapter.ID).First(&lock).Error; err != nil || lock.OwnerID != input.LockOwnerID || lock.Version != input.LockVersion || !lock.LeaseExpiresAt.After(time.Now().UTC()) {
			return ErrRevisionLocked
		}
		var candidate ChapterRevision
		if err := tx.Where("id = ? AND chapter_id = ? AND kind = ?", input.CandidateRevisionID, chapter.ID, sharedtextbook.RevisionKindCandidate).First(&candidate).Error; err != nil {
			return ErrInvalidState
		}
		// Applying a candidate is a compare-and-swap over the chapter head, not
		// merely a lookup by candidate ID. A manual draft created after this
		// candidate becomes the current revision and must never be overwritten.
		if chapter.CurrentRevisionID == nil || *chapter.CurrentRevisionID != candidate.ID {
			return ErrVersionConflict
		}
		var decisionCount int64
		if err := tx.Model(&CandidateReview{}).Where("candidate_revision_id = ?", candidate.ID).Count(&decisionCount).Error; err != nil {
			return err
		}
		if decisionCount > 0 {
			return ErrInvalidState
		}
		if err := sharedtextbook.ValidateCurrentSampleQualityReport(json.RawMessage(candidate.QualityReportJSON)); err != nil {
			return ErrInvalidState
		}
		if !r.candidateMatchesGenerationTarget(candidate) {
			return ErrInvalidState
		}
		review := sharedtextbook.NewSampleReview(input.DimensionScores, input.ReviewerKind, input.DelegationNote)
		if review.Validate() != nil {
			return ErrInvalidState
		}
		humanReviewJSON, err := json.Marshal(review)
		if err != nil {
			return err
		}
		approvalReview := CandidateReview{
			ProjectID:              project.ID,
			ChapterID:              chapter.ID,
			CandidateRevisionID:    candidate.ID,
			ReviewerWorkspaceID:    workspaceID,
			Decision:               CandidateDecisionApproved,
			Reason:                 input.ReviewNote,
			CandidateContentHash:   candidate.ContentHash,
			QualityContractVersion: sharedtextbook.SampleQualityContractVersion,
			HumanReviewJSON:        datatypes.JSON(humanReviewJSON),
		}
		if err := tx.Create(&approvalReview).Error; err != nil {
			return err
		}
		approved = &ChapterRevision{ChapterID: chapter.ID, RevisionNumber: chapter.RevisionVersion + 1, Kind: sharedtextbook.RevisionKindApproved, Markdown: candidate.Markdown, DocumentJSON: candidate.DocumentJSON, ContentHash: candidate.ContentHash, CreatedBy: RevisionCreatorManual, ParentRevisionID: &candidate.ID, BookContractRevisionID: candidate.BookContractRevisionID, StyleSheetRevisionID: candidate.StyleSheetRevisionID, BlueprintRevisionID: candidate.BlueprintRevisionID, EvidencePackHash: candidate.EvidencePackHash, PromptHash: candidate.PromptHash, ProviderName: candidate.ProviderName, ModelName: candidate.ModelName, ProviderUsageJSON: candidate.ProviderUsageJSON, QualityReportJSON: candidate.QualityReportJSON}
		if review.ReviewerKind == "delegated_ai" {
			approved.CreatedBy = RevisionCreatorGeneration
		}
		if err := tx.Create(approved).Error; err != nil {
			return err
		}
		result := tx.Model(&Chapter{}).Where("id = ? AND revision_version = ?", chapter.ID, input.ExpectedVersion).Updates(map[string]any{"current_revision_id": approved.ID, "approved_revision_id": approved.ID, "revision_version": approved.RevisionNumber, "status": StatusApproved, "updated_at": gorm.Expr("CURRENT_TIMESTAMP")})
		if result.Error != nil || result.RowsAffected != 1 {
			return ErrVersionConflict
		}
		return nil
	})
	return approved, err
}
