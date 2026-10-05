package textbook

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func assertDelegatedApproval(t *testing.T, db *gorm.DB, workspaceID, ownerID uuid.UUID, candidate *ChapterRevision, lockVersion int, target sharedtextbook.SampleGenerationTarget) {
	t.Helper()
	tx := db.Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	service := NewService(NewGormRepository(tx, target))
	input := ApplyCandidateInput{ChapterID: candidate.ChapterID, CandidateRevisionID: candidate.ID, ExpectedVersion: candidate.RevisionNumber, LockOwnerID: ownerID, LockVersion: lockVersion, ReviewNote: "逐项复核后，按用户明确委托批准当前候选。", DimensionScores: passingSampleReviewScores(), ReviewerKind: "delegated_ai"}
	_, err := service.ApplyCandidate(t.Context(), workspaceID, input)
	require.ErrorIs(t, err, ErrInvalidState, "missing authority must not create an approval")
	input.DelegationNote = "用户明确将此章审阅与批准决定交给 Codex 执行。"
	approved, err := service.ApplyCandidate(t.Context(), workspaceID, input)
	require.NoError(t, err)
	require.Equal(t, candidate.Markdown, approved.Markdown)
	require.Equal(t, candidate.ContentHash, approved.ContentHash)
	require.Equal(t, RevisionCreatorGeneration, approved.CreatedBy)
	var stored CandidateReview
	require.NoError(t, tx.Where("candidate_revision_id = ?", candidate.ID).First(&stored).Error)
	var review sharedtextbook.SampleHumanReview
	require.NoError(t, json.Unmarshal(stored.HumanReviewJSON, &review))
	require.NoError(t, review.Validate())
	require.Equal(t, sharedtextbook.SampleDelegatedReviewContractVersion, review.ContractVersion)
	require.Equal(t, input.DelegationNote, review.DelegationNote)
	require.Equal(t, "delegated_ai", review.ReviewerKind)
	_, err = service.ApplyCandidate(t.Context(), workspaceID, input)
	require.ErrorIs(t, err, ErrVersionConflict)
}
