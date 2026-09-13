package export

import sharedtextbook "inkwords-backend/shared/kernel/textbook"

// PublicationReviewStage names an editorial review stage. Automated detectors use
// AutomatedPublicationCheck and are intentionally a different type.
type PublicationReviewStage = sharedtextbook.PublicationReviewStage

const (
	PublicationReviewDevelopmental = sharedtextbook.PublicationReviewDevelopmental
	PublicationReviewTechnical     = sharedtextbook.PublicationReviewTechnical
	PublicationReviewSelfStudy     = sharedtextbook.PublicationReviewSelfStudy
	PublicationReviewConsistency   = sharedtextbook.PublicationReviewConsistency
	PublicationReviewCopyEditing   = sharedtextbook.PublicationReviewCopyEditing
	PublicationReviewLayout        = sharedtextbook.PublicationReviewLayout
	PublicationReviewRights        = sharedtextbook.PublicationReviewRights
	PublicationReviewReaderTrial   = sharedtextbook.PublicationReviewReaderTrial
)

var requiredPublicationReviewStages = sharedtextbook.RequiredPublicationReviewStages()

// HumanPublicationReview records review evidence provided by a person. The
// Automated flag is retained to fail closed when an import attempts to label a
// machine-generated result as a human review.
type HumanPublicationReview = sharedtextbook.HumanPublicationReview

// AutomatedPublicationCheck identifies its detector so UI and exported
// manifests never present it as a human or publisher review.
type AutomatedPublicationCheck = sharedtextbook.AutomatedPublicationCheck

type PublicationPreflightInput = sharedtextbook.PublicationPreflightInput

type PublicationPreflightResult = sharedtextbook.PublicationPreflightResult

// EvaluatePublicationPreflight is deliberately fail-closed: a publication
// candidate needs rights clearance and all eight explicitly attributed stages.
// It returns blockers rather than changing BookBuild status; core-api owns
// state transitions after a caller explicitly applies this result.
func EvaluatePublicationPreflight(input PublicationPreflightInput) PublicationPreflightResult {
	return sharedtextbook.EvaluatePublicationPreflight(input)
}
