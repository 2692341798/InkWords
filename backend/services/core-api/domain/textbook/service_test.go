package textbook

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func TestAppendRevisionRejectsGeneratedCandidateWithoutAuditableProvenance(t *testing.T) {
	service := NewService(nil)
	_, err := service.AppendRevision(context.Background(), uuid.New(), AppendRevisionInput{
		ChapterID:    uuid.New(),
		Kind:         sharedtextbook.RevisionKindCandidate,
		Markdown:     "# candidate",
		DocumentJSON: []byte(`{}`),
		ContentHash:  strings.Repeat("a", 64),
		CreatedBy:    RevisionCreatorGeneration,
	})

	require.ErrorIs(t, err, ErrInvalidState)
	require.ErrorContains(t, err, "generated candidates require complete provenance")
}

func TestAppendRevisionRejectsMalformedGeneratedCandidateProvenanceJSON(t *testing.T) {
	parentID, bookID, styleID := uuid.New(), uuid.New(), uuid.New()
	service := NewService(nil)
	_, err := service.AppendRevision(context.Background(), uuid.New(), AppendRevisionInput{
		ChapterID:              uuid.New(),
		Kind:                   sharedtextbook.RevisionKindCandidate,
		Markdown:               "# candidate",
		DocumentJSON:           []byte(`{}`),
		ContentHash:            strings.Repeat("a", 64),
		CreatedBy:              RevisionCreatorGeneration,
		ParentRevisionID:       &parentID,
		BookContractRevisionID: &bookID,
		StyleSheetRevisionID:   &styleID,
		EvidencePackHash:       "sha256:evidence",
		PromptHash:             "sha256:prompt",
		ProviderName:           "fake",
		ModelName:              "deterministic-gin-fixture",
		ProviderUsageJSON:      []byte(`not-json`),
		QualityReportJSON:      []byte(`{"passed":true}`),
	})

	require.ErrorIs(t, err, ErrInvalidState)
	require.ErrorContains(t, err, "generated candidates require complete provenance")
}

func TestRejectCandidateRequiresBoundedReasonAndLeaseIdentity(t *testing.T) {
	service := NewService(nil)
	valid := RejectCandidateInput{
		ChapterID:           uuid.New(),
		CandidateRevisionID: uuid.New(),
		ExpectedVersion:     1,
		LockOwnerID:         uuid.New(),
		LockVersion:         1,
		Reason:              "太短",
	}

	_, err := service.RejectCandidate(context.Background(), uuid.New(), valid)
	require.ErrorIs(t, err, ErrInvalidState)
	require.ErrorContains(t, err, "invalid candidate rejection")

	valid.Reason = strings.Repeat("审", 2001)
	_, err = service.RejectCandidate(context.Background(), uuid.New(), valid)
	require.ErrorIs(t, err, ErrInvalidState)

	valid.Reason = "请求查找证据不足，不能通过人工审阅。"
	valid.LockOwnerID = uuid.Nil
	_, err = service.RejectCandidate(context.Background(), uuid.New(), valid)
	require.ErrorIs(t, err, ErrInvalidState)
}

func TestApplyCandidateRequiresACompletePassingHumanReview(t *testing.T) {
	valid := ApplyCandidateInput{
		ChapterID:           uuid.New(),
		CandidateRevisionID: uuid.New(),
		ExpectedVersion:     1,
		LockOwnerID:         uuid.New(),
		LockVersion:         1,
		ReviewNote:          "已逐项人工审阅并确认可批准。",
		DimensionScores:     passingSampleReviewScores(),
	}

	valid.DimensionScores[0].Score = 2
	_, err := NewService(nil).ApplyCandidate(context.Background(), uuid.New(), valid)
	require.ErrorIs(t, err, ErrInvalidState)

	valid.DimensionScores = passingSampleReviewScores()
	valid.ReviewNote = "太短"
	_, err = NewService(nil).ApplyCandidate(context.Background(), uuid.New(), valid)
	require.ErrorIs(t, err, ErrInvalidState)
}
