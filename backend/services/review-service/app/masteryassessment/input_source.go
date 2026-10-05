package masteryassessment

import (
	"context"

	"github.com/google/uuid"
	"inkwords-backend/services/review-service/domain/mastery"
)

type verifiedInputSource struct {
	answers       InputSource
	verifications VerificationSource
}

// NewVerifiedInputSource decorates the read-only assessment input assembly.
// Preparing or previewing an assessment never creates a verification run.
func NewVerifiedInputSource(answers InputSource, verifications VerificationSource) InputSource {
	return verifiedInputSource{answers: answers, verifications: verifications}
}

func (source verifiedInputSource) PrepareAssessmentInput(ctx context.Context, owner, objective, attempt uuid.UUID) (mastery.AssessmentInput, error) {
	input, err := source.answers.PrepareAssessmentInput(ctx, owner, objective, attempt)
	if err != nil || input.LearnerArtifact == nil || source.verifications == nil {
		return input, err
	}
	resolved, report, err := source.verifications.LatestAssessmentEvidence(ctx, owner, objective, attempt)
	if err != nil || resolved == nil || report == nil {
		return input, err
	}
	return mastery.AttachLearnerVerification(input, *resolved, *report)
}
