package review

import (
	"testing"

	"github.com/stretchr/testify/require"

	"inkwords-backend/internal/model"
)

func TestBuildHintText_DifferentiatesReviewModes(t *testing.T) {
	t.Parallel()

	outline := SessionOutline{
		Checkpoints: []string{
			"这篇内容主要在解决什么问题",
			"有哪些关键概念或步骤",
			"有没有一个可以迁移到别处的例子",
		},
	}

	lightHint := buildHintText(model.ReviewSession{
		Mode:          model.ReviewModeLightRecall,
		HintUsedCount: 0,
	}, nil, outline)
	detailedHint := buildHintText(model.ReviewSession{
		Mode:          model.ReviewModeDetailedQA,
		HintUsedCount: 0,
	}, nil, outline)

	require.NotEqual(t, lightHint, detailedHint)
	require.Contains(t, lightHint, "主线")
	require.Contains(t, detailedHint, "追问")
}

func TestBuildFinalFeedback_DifferentiatesReviewModes(t *testing.T) {
	t.Parallel()

	lightFeedback := buildFinalFeedback(model.ReviewModeLightRecall, []model.ReviewTurn{{
		Role:     model.ReviewTurnRoleUser,
		TurnType: model.ReviewTurnTypeAnswer,
		Content:  "我先讲主线",
	}})
	detailedFeedback := buildFinalFeedback(model.ReviewModeDetailedQA, []model.ReviewTurn{{
		Role:     model.ReviewTurnRoleUser,
		TurnType: model.ReviewTurnTypeAnswer,
		Content:  "我回答当前追问",
	}})

	require.NotEqual(t, lightFeedback.Summary, detailedFeedback.Summary)
	require.Contains(t, lightFeedback.Summary, "有效复习")
	require.Contains(t, detailedFeedback.Summary, "逐步追问")
	require.NotEqual(t, lightFeedback.NextFocus, detailedFeedback.NextFocus)
}

func TestBuildReviewFeedback_ReportsHitsAndMisses(t *testing.T) {
	t.Parallel()

	outline := SessionOutline{
		Checkpoints: []string{
			"并发控制限制同时执行的任务数量",
			"速率限制控制单位时间内的请求频率",
			"信号量负责发放和回收执行许可",
		},
	}

	feedback := buildReviewFeedback(outline, "并发控制是为了限制同时执行的任务数量，通常会配合信号量。")
	require.Equal(t, "答对较多", feedback.Judgement)
	require.Contains(t, feedback.HitPoints, "并发控制限制同时执行的任务数量")
	require.Contains(t, feedback.HitPoints, "信号量负责发放和回收执行许可")
	require.Contains(t, feedback.MissedPoints, "速率限制控制单位时间内的请求频率")
	require.NotEmpty(t, feedback.Suggestion)
}
