package export

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func TestPublicationPreflightDoesNotTreatAutomationAsHumanReview(t *testing.T) {
	result := EvaluatePublicationPreflight(PublicationPreflightInput{
		BuildID:                "build-1",
		RequiredRightsSubjects: requiredRightsSubjects(),
		RightsItems:            []sharedtextbook.RightsItem{readyRightsItem()},
		AutomatedChecks:        []AutomatedPublicationCheck{{ID: "technical-lint", Detector: "static-policy", Status: sharedtextbook.QualityStatusPass}},
		HumanReviews:           []HumanPublicationReview{{ID: "review-1", BuildID: "build-1", Stage: PublicationReviewTechnical, Reviewer: "automated-review", Notes: "自动记录不能冒充人工审校。", CompletedAt: time.Unix(1, 0), Automated: true}},
	})

	require.False(t, result.Passed)
	require.Contains(t, result.Blockers, "缺少人工或用户委托 AI 技术审校记录。")
}

func TestPublicationPreflightRequiresReadyRightsAndEveryHumanStage(t *testing.T) {
	now := time.Unix(1, 0)
	reviews := make([]HumanPublicationReview, 0, len(requiredPublicationReviewStages))
	for index, stage := range requiredPublicationReviewStages {
		reviews = append(reviews, HumanPublicationReview{ContractVersion: sharedtextbook.HumanPublicationReviewFormat, ReviewerKind: "human", ManifestHash: "sha256:" + strings.Repeat("a", 64), Revision: 1, Verdict: "pass", Score: 3, Scope: "隔离测试中的明确审阅范围", EvidenceRefs: []string{"test:fixture"}, ID: fmt.Sprintf("review-%d", index), BuildID: "build-1", Stage: stage, Reviewer: "隔离测试者", Notes: "已按清单完成该阶段的测试检查。", CompletedAt: now})
	}
	rights := readyRightsItem()
	rights.PublicationStatus = sharedtextbook.RightsStatusPending

	result := EvaluatePublicationPreflight(PublicationPreflightInput{ManifestHash: "sha256:" + strings.Repeat("a", 64), BuildID: "build-1", RequiredRightsSubjects: requiredRightsSubjects(), RightsItems: []sharedtextbook.RightsItem{rights}, HumanReviews: reviews, AutomatedChecks: []AutomatedPublicationCheck{{ID: "rules", Detector: "policy", Status: sharedtextbook.QualityStatusPass}}})

	require.False(t, result.Passed)
	require.Contains(t, result.Blockers, "权利项 rights-1 尚未处于 ready 状态。")
}

func TestPublicationPreflightPassesOnlyWithDistinctHumanEvidence(t *testing.T) {
	now := time.Unix(1, 0)
	reviews := make([]HumanPublicationReview, 0, len(requiredPublicationReviewStages))
	for index, stage := range requiredPublicationReviewStages {
		reviews = append(reviews, HumanPublicationReview{ContractVersion: sharedtextbook.HumanPublicationReviewFormat, ReviewerKind: "human", ManifestHash: "sha256:" + strings.Repeat("a", 64), Revision: 1, Verdict: "pass", Score: 3, Scope: "隔离测试中的明确审阅范围", EvidenceRefs: []string{"test:fixture"}, ID: fmt.Sprintf("review-%d", index), BuildID: "build-1", Stage: stage, Reviewer: "隔离测试者", Notes: "已按清单完成该阶段的测试检查。", CompletedAt: now})
	}

	result := EvaluatePublicationPreflight(PublicationPreflightInput{ManifestHash: "sha256:" + strings.Repeat("a", 64), BuildID: "build-1", RequiredRightsSubjects: requiredRightsSubjects(), RightsItems: []sharedtextbook.RightsItem{readyRightsItem()}, HumanReviews: reviews, AutomatedChecks: []AutomatedPublicationCheck{{ID: "rules", Detector: "policy", Status: sharedtextbook.QualityStatusPass}}})

	require.True(t, result.Passed)
	require.Empty(t, result.Blockers)
}

func TestPublicationPreflightRequiresEveryFrozenRightsSubject(t *testing.T) {
	now := time.Unix(1, 0)
	reviews := make([]HumanPublicationReview, 0, len(requiredPublicationReviewStages))
	for index, stage := range requiredPublicationReviewStages {
		reviews = append(reviews, HumanPublicationReview{ContractVersion: sharedtextbook.HumanPublicationReviewFormat, ReviewerKind: "human", ManifestHash: "sha256:" + strings.Repeat("a", 64), Revision: 1, Verdict: "pass", Score: 3, Scope: "隔离测试中的明确审阅范围", EvidenceRefs: []string{"test:fixture"}, ID: fmt.Sprintf("review-%d", index), BuildID: "build-1", Stage: stage, Reviewer: "隔离测试者", Notes: "已按清单完成该阶段的测试检查。", CompletedAt: now})
	}
	required := append(requiredRightsSubjects(), sharedtextbook.RightsSubject{SubjectRef: "asset:screenshot-1", WorkType: sharedtextbook.RightsWorkTypeScreenshot})
	result := EvaluatePublicationPreflight(PublicationPreflightInput{ManifestHash: "sha256:" + strings.Repeat("a", 64), BuildID: "build-1", RequiredRightsSubjects: required, RightsItems: []sharedtextbook.RightsItem{readyRightsItem()}, HumanReviews: reviews, AutomatedChecks: []AutomatedPublicationCheck{{ID: "rules", Detector: "policy", Status: sharedtextbook.QualityStatusPass}}})
	require.False(t, result.Passed)
	require.Contains(t, result.Blockers, "权利清单尚未覆盖 asset:screenshot-1（screenshot）。")
}

func readyRightsItem() sharedtextbook.RightsItem {
	return sharedtextbook.RightsItem{ID: "rights-1", ProjectID: "project-1", BuildID: "build-1", SubjectRef: "chapter:revision-1", WorkType: sharedtextbook.RightsWorkTypeProse, RightsBasis: "author-owned", AllowedUse: "publication", Attribution: "InkWords", PublicationStatus: sharedtextbook.RightsStatusReady}
}

func requiredRightsSubjects() []sharedtextbook.RightsSubject {
	return []sharedtextbook.RightsSubject{{SubjectRef: "chapter:revision-1", WorkType: sharedtextbook.RightsWorkTypeProse}}
}
