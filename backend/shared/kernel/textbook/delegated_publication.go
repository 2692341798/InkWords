package textbook

import (
	"fmt"
	"strings"
	"time"
)

// DelegatedPublicationReviewFormat identifies the separately attributed AI review contract.
const DelegatedPublicationReviewFormat = "inkwords.delegated-publication-review.v1"

// DelegatedPublicationReview is an explicitly authorized AI editorial decision,
// never a statement that a human peer, learner or publisher performed a review.
// Revisions append evidence; they do not overwrite a previous conclusion.
type DelegatedPublicationReview struct {
	ContractVersion string                 `json:"contract_version"`
	ID              string                 `json:"id"`
	BuildID         string                 `json:"build_id"`
	ManifestHash    string                 `json:"manifest_hash"`
	Stage           PublicationReviewStage `json:"stage"`
	Revision        int                    `json:"revision"`
	ReviewerKind    string                 `json:"reviewer_kind"`
	Reviewer        string                 `json:"reviewer"`
	DelegationNote  string                 `json:"delegation_note"`
	Verdict         string                 `json:"verdict"`
	Score           int                    `json:"score"`
	Scope           string                 `json:"scope"`
	Notes           string                 `json:"notes"`
	EvidenceRefs    []string               `json:"evidence_refs"`
	HardFailures    []string               `json:"hard_failures"`
	CompletedAt     time.Time              `json:"completed_at"`
}

func (review DelegatedPublicationReview) Validate() error {
	bounded := func(value string, min, max int) bool {
		n := len([]rune(strings.TrimSpace(value)))
		return n >= min && n <= max
	}
	if review.ContractVersion != DelegatedPublicationReviewFormat || review.ReviewerKind != "delegated_ai" || !bounded(review.ID, 1, 128) || !bounded(review.BuildID, 1, 128) || !isSHA256Digest(review.ManifestHash) || review.Stage.Validate() != nil || review.Revision < 1 || review.CompletedAt.IsZero() {
		return fmt.Errorf("delegated publication review identity is invalid")
	}
	if !bounded(review.Reviewer, 1, 128) || !bounded(review.DelegationNote, 8, 1000) || !bounded(review.Scope, 8, 1000) || !bounded(review.Notes, 8, 4000) || review.Score < 0 || review.Score > 4 || len(review.EvidenceRefs) > 32 || len(review.HardFailures) > 32 {
		return fmt.Errorf("delegated publication review evidence is incomplete")
	}
	for _, reference := range review.EvidenceRefs {
		if !bounded(reference, 1, 500) {
			return fmt.Errorf("invalid review evidence reference")
		}
	}
	for _, failure := range review.HardFailures {
		if !bounded(failure, 1, 500) {
			return fmt.Errorf("invalid review hard failure")
		}
	}
	switch review.Verdict {
	case "pass":
		if review.Score < 3 || len(review.HardFailures) != 0 || len(review.EvidenceRefs) == 0 {
			return fmt.Errorf("passing review needs evidence, score at least 3 and no hard failure")
		}
	case "needs_revision":
		if len(review.HardFailures) == 0 {
			return fmt.Errorf("revision decision needs an explicit blocking finding")
		}
	case "not_assessed":
		if review.Score != 0 || len(review.HardFailures) != 0 {
			return fmt.Errorf("unassessed review must not claim a score or hide a blocking finding")
		}
	default:
		return fmt.Errorf("invalid delegated review verdict")
	}
	return nil
}

func applyDelegatedPublicationReviews(input PublicationPreflightInput, completed map[PublicationReviewStage]bool, blockers []string) []string {
	latest := map[PublicationReviewStage]DelegatedPublicationReview{}
	for _, review := range input.DelegatedReviews {
		if review.Validate() != nil || review.BuildID != input.BuildID || review.ManifestHash != input.ManifestHash {
			blockers = append(blockers, "委托审阅 "+review.ID+" 不完整或不属于当前冻结构建。")
			continue
		}
		prior, exists := latest[review.Stage]
		if exists && review.Revision == prior.Revision && review.ID != prior.ID {
			blockers = append(blockers, "同一阶段存在冲突的委托审阅版本。")
			continue
		}
		if !exists || review.Revision > prior.Revision {
			latest[review.Stage] = review
		}
	}
	for _, stage := range RequiredPublicationReviewStages() {
		review, exists := latest[stage]
		if !exists {
			continue
		}
		switch review.Verdict {
		case "pass":
			completed[stage] = true
		case "needs_revision":
			completed[stage] = false
			blockers = append(blockers, "用户委托 AI "+stage.Label()+"审校需要修改："+strings.Join(review.HardFailures, "；"))
		case "not_assessed":
			if !completed[stage] {
				completed[stage] = false
			}
		}
	}
	return blockers
}
