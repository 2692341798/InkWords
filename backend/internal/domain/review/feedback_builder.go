package review

import (
	"strings"

	"inkwords-backend/internal/model"
)

func buildStageFeedback(mode string, answer string) string {
	if mode == model.ReviewModeDetailedQA {
		if strings.TrimSpace(answer) == "" {
			return "这一轮先不用展开整篇，只要先回答当前追问即可，下一轮我们再继续往下拆。"
		}
		return "这一轮已经回答了当前追问，接下来我会继续沿着关键概念、步骤关系或因果细节往下追问。"
	}

	if strings.TrimSpace(answer) == "" {
		return "你已经开始回忆了，接下来可以先补主线，再补一个关键例子。"
	}

	return "你已经开始抓主线了，接下来可以再补一个关键概念或具体例子。"
}

func buildFinalFeedback(mode string, turns []model.ReviewTurn) FinalFeedback {
	if mode == model.ReviewModeDetailedQA {
		answerCount := countUserAnswers(turns)
		summary := "这次已经完成一轮逐步追问式复习。"
		if answerCount == 0 {
			summary = "这次逐步追问先告一段落，下一次可以先对第一问给出更明确的结论。"
		} else if answerCount < 3 {
			summary = "这次已经完成一部分逐步追问，下一次可以继续把剩余问题答完。"
		}

		return FinalFeedback{
			Summary:   summary,
			Strengths: []string{"已经能跟随追问逐层展开，而不是只停留在笼统概括"},
			Gaps:      []string{"某些关键概念、关系或例子还可以回答得更具体"},
			NextFocus: []string{"下次优先练习“结论 -> 依据 -> 例子”的逐问回答结构"},
		}
	}

	answer := lastUserAnswer(turns)
	summary := "这次已经完成一次有效复习。"
	if strings.TrimSpace(answer) == "" {
		summary = "这次复习已经完成，下一次可以尝试更主动地先说出主线。"
	}

	return FinalFeedback{
		Summary:   summary,
		Strengths: []string{"已经尝试主动回忆并输出主线"},
		Gaps:      []string{"还可以补一个更具体的例子或迁移场景"},
		NextFocus: []string{"下次优先讲清楚为什么这样设计"},
	}
}
