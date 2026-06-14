package review

import (
	"strings"
)

func buildStageFeedback(mode string, feedback ReviewFeedback) string {
	if mode == ReviewModeDetailedQA {
		switch feedback.Judgement {
		case "偏题":
			return "这一轮先不用展开整篇，只要先回答当前追问即可，下一轮我们再继续往下拆。"
		case "部分答对":
			return "这一轮已经抓到一部分主线了，接下来我会沿着还没讲到的关键概念和关系继续追问。"
		default:
			return "这一轮已经抓住当前追问的重点了，下一轮我会继续往关键细节和迁移场景推进。"
		}
	}

	switch feedback.Judgement {
	case "偏题":
		return "你已经开始回忆了，接下来可以先补主线，再补一个关键例子。"
	case "部分答对":
		return "你已经抓到部分主线了，接下来优先把还没提到的关键概念补完整。"
	default:
		return "你已经抓住主线了，接下来可以再补一个关键概念、步骤关系或具体例子。"
	}
}

func buildFinalFeedback(mode string, turns []ReviewTurn) FinalFeedback {
	if mode == ReviewModeDetailedQA {
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

func buildReviewFeedback(outline SessionOutline, answer string) ReviewFeedback {
	hitPoints := make([]string, 0, len(outline.Checkpoints))
	missedPoints := make([]string, 0, len(outline.Checkpoints))
	for _, checkpoint := range outline.Checkpoints {
		if matchesCheckpoint(checkpoint, answer) {
			hitPoints = append(hitPoints, checkpoint)
			continue
		}
		missedPoints = append(missedPoints, checkpoint)
	}

	if len(hitPoints) == 0 && len(missedPoints) == 0 {
		missedPoints = []string{"还没有覆盖这篇文章的核心主线"}
	}

	judgement := classifyReviewAnswer(len(hitPoints), len(outline.Checkpoints))
	return ReviewFeedback{
		Judgement:    judgement,
		HitPoints:    ensureFeedbackItems(hitPoints, "已经开始围绕主题作答，但还需要更贴近文章主线"),
		MissedPoints: ensureFeedbackItems(missedPoints, "可以再补一个更具体的关键点"),
		Suggestion:   buildReviewSuggestion(judgement, missedPoints),
	}
}

func matchesCheckpoint(checkpoint string, answer string) bool {
	normalizedCheckpoint := normalizeForMatch(checkpoint)
	normalizedAnswer := normalizeForMatch(answer)
	if normalizedCheckpoint == "" || normalizedAnswer == "" {
		return false
	}
	if strings.Contains(normalizedAnswer, normalizedCheckpoint) || strings.Contains(normalizedCheckpoint, normalizedAnswer) {
		return true
	}

	topic := normalizeForMatch(buildCheckpointTopic(checkpoint))
	if topic != "" && strings.Contains(normalizedAnswer, topic) {
		return true
	}

	return false
}

func classifyReviewAnswer(hitCount int, total int) string {
	if total <= 0 {
		total = 1
	}
	switch {
	case hitCount >= total-1:
		return "答对较多"
	case hitCount > 0:
		return "部分答对"
	default:
		return "偏题"
	}
}

func ensureFeedbackItems(items []string, fallback string) []string {
	if len(items) > 0 {
		return items
	}
	return []string{fallback}
}

func buildReviewSuggestion(judgement string, missedPoints []string) string {
	switch judgement {
	case "答对较多":
		if len(missedPoints) > 0 {
			return "下一轮重点把这几点补齐：" + strings.Join(missedPoints, "；") + "。"
		}
		return "下一轮试着补一个适用场景或迁移例子，让解释更完整。"
	case "部分答对":
		if len(missedPoints) > 0 {
			return "你已经抓到一部分主线，下一轮优先补这几点：" + strings.Join(missedPoints, "；") + "。"
		}
		return "你已经抓到一部分主线，下一轮再把关键概念和因果说得更清楚。"
	default:
		return "先回到文章主线，优先回答“它主要在解决什么问题，以及核心做法是什么”。"
	}
}
