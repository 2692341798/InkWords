package textbook

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func delegatedPublicationFixture() DelegatedPublicationReview {
	return DelegatedPublicationReview{ContractVersion: DelegatedPublicationReviewFormat, ID: "review", BuildID: "build", ManifestHash: "sha256:" + strings.Repeat("a", 64), Stage: PublicationReviewTechnical, Revision: 1, ReviewerKind: "delegated_ai", Reviewer: "Codex", DelegationNote: "用户明确委托 AI 完成审阅与决策。", Verdict: "pass", Score: 3, Scope: "核对固定母稿、工件哈希与实际隔离测试报告。", Notes: "固定工件对应的隔离测试通过；未声称运行整个上游仓库。", EvidenceRefs: []string{"verification-task:run-1"}, CompletedAt: time.Unix(100, 0)}
}

func TestDelegatedPublicationReviewRequiresAuthorityAndHonestVerdict(t *testing.T) {
	valid := delegatedPublicationFixture()
	require.NoError(t, valid.Validate())
	for _, mutate := range []func(*DelegatedPublicationReview){
		func(r *DelegatedPublicationReview) { r.DelegationNote = "" },
		func(r *DelegatedPublicationReview) { r.ReviewerKind = "human" },
		func(r *DelegatedPublicationReview) { r.Score = 2 },
		func(r *DelegatedPublicationReview) { r.HardFailures = []string{"实际代码测试失败"} },
		func(r *DelegatedPublicationReview) { r.EvidenceRefs = nil },
		func(r *DelegatedPublicationReview) { r.ManifestHash = "floating" },
		func(r *DelegatedPublicationReview) {
			r.Verdict = "not_assessed"
			r.Score = 0
			r.HardFailures = []string{"已知代码失败不能藏入未评估状态"}
		},
	} {
		copy := valid
		mutate(&copy)
		require.Error(t, copy.Validate())
	}
}

func TestDelegatedPublicationPreflightCanCompleteInternalReviewButCannotWaiveRights(t *testing.T) {
	fixture := delegatedPublicationFixture()
	input := PublicationPreflightInput{
		BuildID: fixture.BuildID, ManifestHash: fixture.ManifestHash,
		RequiredRightsSubjects: []RightsSubject{{SubjectRef: "chapter:r1", WorkType: RightsWorkTypeProse}},
		RightsItems:            []RightsItem{{ID: "rights", ProjectID: "project", BuildID: fixture.BuildID, SubjectRef: "chapter:r1", WorkType: RightsWorkTypeProse, RightsBasis: "test fixture evidence", AllowedUse: "local draft", Attribution: "test author", PublicationStatus: RightsStatusReady}},
		AutomatedChecks:        []AutomatedPublicationCheck{{ID: "verified", Detector: "test fixture", Status: QualityStatusPass}},
	}
	for _, stage := range RequiredPublicationReviewStages() {
		review := fixture
		review.ID, review.Stage = string(stage), stage
		input.DelegatedReviews = append(input.DelegatedReviews, review)
	}
	require.True(t, EvaluatePublicationPreflight(input).Passed)
	require.Empty(t, input.HumanReviews, "AI delegation must not synthesize a human record")
	input.RightsItems[0].PublicationStatus = RightsStatusPending
	require.False(t, EvaluatePublicationPreflight(input).Passed)
}

func TestDelegatedPublicationPreflightUsesLatestBoundReviewWithoutInventingHumanEvidence(t *testing.T) {
	review := delegatedPublicationFixture()
	input := PublicationPreflightInput{BuildID: review.BuildID, ManifestHash: review.ManifestHash, DelegatedReviews: []DelegatedPublicationReview{review}}
	result := EvaluatePublicationPreflight(input)
	require.NotContains(t, result.Blockers, "缺少人工或用户委托 AI 技术审校记录。")
	require.False(t, result.Passed, "one delegated stage must not waive rights or other stages")
	newer := review
	newer.ID, newer.Revision, newer.Verdict, newer.Score = "review-2", 2, "needs_revision", 2
	newer.HardFailures = []string{"新校样发现代码被截断"}
	input.DelegatedReviews = append(input.DelegatedReviews, newer)
	result = EvaluatePublicationPreflight(input)
	require.Contains(t, result.Blockers, "用户委托 AI 技术审校需要修改：新校样发现代码被截断")
	require.Contains(t, result.Blockers, "技术审校已有记录，但尚未通过。")
	require.NotContains(t, result.Blockers, "缺少人工或用户委托 AI 技术审校记录。", "failed review exists and must not be mislabeled missing")
	input.DelegatedReviews = []DelegatedPublicationReview{review}
	input.ManifestHash = "sha256:" + strings.Repeat("b", 64)
	result = EvaluatePublicationPreflight(input)
	require.Contains(t, result.Blockers, "委托审阅 review 不完整或不属于当前冻结构建。")
}
