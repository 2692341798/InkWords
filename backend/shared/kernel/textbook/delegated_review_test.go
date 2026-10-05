package textbook

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDelegatedReviewPreservesActorAndRequiresAuthorityAndPassingScores(t *testing.T) {
	var scores []DimensionScore
	for _, dimension := range SampleManualReviewDimensions() {
		scores = append(scores, DimensionScore{Dimension: dimension, Score: 3})
	}
	review := NewSampleReview(scores, "delegated_ai", "用户明确委托 Codex 审阅当前样章并决定是否批准。")
	require.NoError(t, review.Validate())
	require.NotEqual(t, SampleHumanReviewContractVersion, review.ContractVersion)
	require.Equal(t, "delegated_ai", review.ReviewerKind)
	review.DelegationNote = ""
	require.Error(t, review.Validate())
	review = NewSampleReview(scores, "publisher", "不能把助手冒充出版社。")
	require.Error(t, review.Validate())
	review = NewSampleReview(scores, "", "")
	require.NoError(t, review.Validate(), "legacy human reviews stay compatible")
	require.Equal(t, SampleHumanReviewContractVersion, review.ContractVersion)
	review = NewSampleReview(scores, "delegated_ai", "用户明确委托 Codex 审阅当前样章并决定是否批准。")
	review.DimensionScores[0].Score = 2
	require.Error(t, review.Validate(), "delegation does not lower the rubric")
}
