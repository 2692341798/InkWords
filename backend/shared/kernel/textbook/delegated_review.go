package textbook

import (
	"fmt"
	"strings"
)

// SampleDelegatedReviewContractVersion prevents AI approval from being recorded
// as the legacy human-review contract.
const SampleDelegatedReviewContractVersion = "inkwords.sample-delegated-review.v1"

// NewSampleReview records who performed the review. A delegated decision is not
// human peer review, cold-reader evidence, or publisher certification.
func NewSampleReview(scores []DimensionScore, kind, authority string) SampleHumanReview {
	review := SampleHumanReview{ContractVersion: SampleHumanReviewContractVersion, DimensionScores: scores, ReviewerKind: strings.TrimSpace(kind), DelegationNote: strings.TrimSpace(authority)}
	if review.ReviewerKind == "delegated_ai" {
		review.ContractVersion = SampleDelegatedReviewContractVersion
	}
	return review
}

func (review SampleHumanReview) validateReviewer() error {
	switch review.ReviewerKind {
	case "", "human":
		if review.ContractVersion != SampleHumanReviewContractVersion || review.DelegationNote != "" {
			return fmt.Errorf("invalid human review identity")
		}
	case "delegated_ai":
		n := len([]rune(strings.TrimSpace(review.DelegationNote)))
		if review.ContractVersion != SampleDelegatedReviewContractVersion || n < 8 || n > 1000 {
			return fmt.Errorf("delegated AI review requires explicit authority")
		}
	default:
		return fmt.Errorf("unsupported reviewer kind")
	}
	return nil
}
