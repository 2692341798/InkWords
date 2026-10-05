package textbook

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHumanReviewNeedsExplicitDecisionAndLatestRevision(t *testing.T) {
	legacy := HumanPublicationReview{ID: "legacy", BuildID: "build", Stage: PublicationReviewRights, Reviewer: "测试审阅者", Notes: "历史记录只有说明，没有结论与评分。", CompletedAt: time.Unix(1, 0)}
	input := PublicationPreflightInput{BuildID: "build", ManifestHash: "sha256:" + strings.Repeat("a", 64), HumanReviews: []HumanPublicationReview{legacy}}
	completed, blockers := evaluateHumanPublicationReviews(input)
	require.False(t, completed[PublicationReviewRights], "a legacy completion note is not an explicit passing decision")
	require.Empty(t, blockers)
	review := legacy
	review.ID = "review-v2"
	review.ContractVersion, review.ReviewerKind = HumanPublicationReviewFormat, "human"
	review.ManifestHash, review.Revision = input.ManifestHash, 2
	review.Verdict, review.Score = "pass", 3
	review.Scope = "隔离测试中复核同一冻结稿的权利清单。"
	review.EvidenceRefs = []string{"test:fixture"}
	review.CompletedAt = time.Unix(2, 0)
	require.NoError(t, review.Validate())
	input.HumanReviews = append(input.HumanReviews, review)
	completed, blockers = evaluateHumanPublicationReviews(input)
	require.True(t, completed[PublicationReviewRights])
	require.Empty(t, blockers)
	failed := review
	failed.ID, failed.Revision, failed.Verdict, failed.Score = "review-v3", 3, "needs_revision", 1
	failed.HardFailures = []string{"测试中新证据仍缺署名"}
	failed.CompletedAt = time.Unix(3, 0)
	input.HumanReviews = []HumanPublicationReview{failed, legacy, review}
	completed, blockers = evaluateHumanPublicationReviews(input)
	require.False(t, completed[PublicationReviewRights])
	require.Contains(t, strings.Join(blockers, " "), failed.HardFailures[0])
	failed.Revision = 5
	input.HumanReviews[0] = failed
	_, blockers = evaluateHumanPublicationReviews(input)
	require.NotEmpty(t, blockers, "missing history cannot be treated as a valid latest decision")
}

func TestHumanReviewRejectsAmbiguousOrMisattributedDecisions(t *testing.T) {
	review := HumanPublicationReview{ContractVersion: HumanPublicationReviewFormat, ReviewerKind: "human", ID: "r", BuildID: "b", ManifestHash: "sha256:" + strings.Repeat("a", 64), Revision: 1, Stage: PublicationReviewTechnical, Reviewer: "隔离测试者", Scope: "隔离测试中的明确审阅范围", Notes: "隔离测试中的明确审阅结论", Verdict: "pass", Score: 3, EvidenceRefs: []string{"test:fixture"}, CompletedAt: time.Unix(1, 0)}
	require.NoError(t, review.Validate())
	for _, change := range []func(*HumanPublicationReview){
		func(r *HumanPublicationReview) { r.ReviewerKind = "delegated_ai" },
		func(r *HumanPublicationReview) { r.Score = 2 },
		func(r *HumanPublicationReview) { r.HardFailures = []string{"未解决失败"} },
		func(r *HumanPublicationReview) { r.ManifestHash = "" },
		func(r *HumanPublicationReview) { r.EvidenceRefs = nil },
	} {
		copy := review
		change(&copy)
		require.Error(t, copy.Validate())
	}
}

func TestHumanReviewFindingCannotBeWaivedByDelegatedPass(t *testing.T) {
	ai := delegatedPublicationFixture()
	human := HumanPublicationReview{ContractVersion: HumanPublicationReviewFormat, ReviewerKind: "human", ID: "human", BuildID: ai.BuildID, ManifestHash: ai.ManifestHash, Revision: 1, Stage: ai.Stage, Reviewer: "隔离测试者", Scope: ai.Scope, Notes: ai.Notes, Verdict: "needs_revision", Score: 1, HardFailures: []string{"测试发现尚未解决的代码错误"}, CompletedAt: ai.CompletedAt}
	result := EvaluatePublicationPreflight(PublicationPreflightInput{BuildID: ai.BuildID, ManifestHash: ai.ManifestHash, HumanReviews: []HumanPublicationReview{human}, DelegatedReviews: []DelegatedPublicationReview{ai}})
	require.False(t, result.Passed)
	require.Contains(t, strings.Join(result.Blockers, " "), human.HardFailures[0])
}
