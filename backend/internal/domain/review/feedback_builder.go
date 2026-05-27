package review

import "strings"

func buildStageFeedback(mode string, answer string) string {
	if mode == "detailed_qa" {
		return "这一步已经把主线往前推进了，我们继续补关键细节。"
	}

	if strings.TrimSpace(answer) == "" {
		return "你已经开始回忆了，接下来可以先补主线，再补一个关键例子。"
	}

	return "你已经开始抓主线了，接下来可以再补一个关键概念或具体例子。"
}

func buildFinalFeedback(answer string) FinalFeedback {
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
