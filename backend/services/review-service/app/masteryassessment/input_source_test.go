package masteryassessment

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"inkwords-backend/services/review-service/domain/mastery"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type fixedAssessmentInput struct{ input mastery.AssessmentInput }

func (source fixedAssessmentInput) PrepareAssessmentInput(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (mastery.AssessmentInput, error) {
	return source.input, nil
}

type countingVerificationEvidence struct{ calls int }

func (source *countingVerificationEvidence) LatestAssessmentEvidence(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*sharedtextbook.LearnerVerificationInput, *sharedtextbook.LearnerVerificationReport, error) {
	source.calls++
	return nil, nil, nil
}

func TestVerifiedInputSourceOnlyReadsEvidenceForCodeAndDoesNotStartWork(t *testing.T) {
	input, _ := fixture(t)
	verifications := &countingVerificationEvidence{}
	resolved, err := NewVerifiedInputSource(fixedAssessmentInput{input: input}, verifications).PrepareAssessmentInput(context.Background(), uuid.New(), uuid.New(), uuid.New())
	require.NoError(t, err)
	require.Equal(t, input, resolved)
	require.Zero(t, verifications.calls)

	input.LearnerArtifact = &sharedtextbook.LearnerArtifact{SubmittedAt: time.Now()}
	resolved, err = NewVerifiedInputSource(fixedAssessmentInput{input: input}, verifications).PrepareAssessmentInput(context.Background(), uuid.New(), uuid.New(), uuid.New())
	require.NoError(t, err)
	require.Equal(t, input, resolved)
	require.Equal(t, 1, verifications.calls)
}
