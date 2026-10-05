package textbook

import (
	"fmt"
	"sort"
	"strings"
)

// HumanPublicationReviewFormat requires an explicit decision for an immutable
// build. Legacy completion notes remain readable, but cannot establish a pass.
const HumanPublicationReviewFormat = "inkwords.human-publication-review.v2"

func (review HumanPublicationReview) validateDecision() error {
	if review.ContractVersion == "" {
		if review.ManifestHash != "" || review.Revision != 0 || review.ReviewerKind != "" || review.Verdict != "" || review.Score != 0 || review.Scope != "" || len(review.EvidenceRefs)+len(review.HardFailures) != 0 {
			return fmt.Errorf("legacy review contains unversioned decision fields")
		}
		return nil
	}
	if review.ContractVersion != HumanPublicationReviewFormat || review.ReviewerKind != "human" || !isSHA256Digest(review.ManifestHash) || review.Revision < 1 || len([]rune(review.Reviewer)) > 128 {
		return fmt.Errorf("human review decision identity is invalid")
	}
	return validatePublicationDecision(review.Verdict, review.Score, review.Scope, review.Notes, review.EvidenceRefs, review.HardFailures)
}

func validatePublicationDecision(verdict string, score int, scope, notes string, evidence, failures []string) error {
	bounded := func(value string, min, max int) bool {
		n := len([]rune(strings.TrimSpace(value)))
		return n >= min && n <= max
	}
	if !bounded(scope, 8, 1000) || !bounded(notes, 8, 4000) || score < 0 || score > 4 || len(evidence) > 32 || len(failures) > 32 {
		return fmt.Errorf("publication review decision is incomplete")
	}
	for _, items := range [][]string{evidence, failures} {
		for _, item := range items {
			if !bounded(item, 1, 500) {
				return fmt.Errorf("invalid publication review evidence or finding")
			}
		}
	}
	switch verdict {
	case "pass":
		if score < 3 || len(evidence) == 0 || len(failures) != 0 {
			return fmt.Errorf("passing review needs evidence, score at least 3 and no hard failure")
		}
	case "needs_revision":
		if len(failures) == 0 {
			return fmt.Errorf("revision decision needs an explicit blocking finding")
		}
	case "not_assessed":
		if score != 0 || len(failures) != 0 {
			return fmt.Errorf("unassessed review must not claim a score or findings")
		}
	default:
		return fmt.Errorf("invalid publication review verdict")
	}
	return nil
}

func evaluateHumanPublicationReviews(input PublicationPreflightInput) (map[PublicationReviewStage]bool, []string) {
	completed := map[PublicationReviewStage]bool{}
	var blockers []string
	byStage := map[PublicationReviewStage][]HumanPublicationReview{}
	for _, review := range input.HumanReviews {
		if review.Validate() != nil || review.BuildID != input.BuildID || (review.ContractVersion != "" && review.ManifestHash != input.ManifestHash) {
			blockers = append(blockers, "真人审阅 "+review.ID+" 不完整或不属于当前冻结构建。")
			continue
		}
		byStage[review.Stage] = append(byStage[review.Stage], review)
	}
	version := func(review HumanPublicationReview) int {
		if review.ContractVersion == "" {
			return 1
		}
		return review.Revision
	}
	for _, stage := range RequiredPublicationReviewStages() {
		history := byStage[stage]
		if len(history) == 0 {
			continue
		}
		completed[stage] = false
		sort.Slice(history, func(i, j int) bool { return version(history[i]) < version(history[j]) })
		valid := true
		seen := map[string]bool{}
		for i, review := range history {
			if version(review) != i+1 || seen[review.ID] || (i > 0 && review.CompletedAt.Before(history[i-1].CompletedAt)) {
				valid = false
			}
			seen[review.ID] = true
		}
		if !valid {
			blockers = append(blockers, "真人"+stage.Label()+"审阅历史缺失或版本冲突。")
			continue
		}
		latest := history[len(history)-1]
		if latest.ContractVersion == "" {
			continue
		}
		completed[stage] = latest.Verdict == "pass"
		if latest.Verdict == "needs_revision" {
			blockers = append(blockers, "真人"+stage.Label()+"审校需要修改："+strings.Join(latest.HardFailures, "；"))
		}
	}
	return completed, blockers
}
