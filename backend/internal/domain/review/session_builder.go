package review

import (
	"strings"

	"inkwords-backend/internal/model"
)

func buildSessionSnapshot(note ReviewNote) (string, []string) {
	body := strings.TrimSpace(note.Body)
	if len([]rune(body)) > 800 {
		body = string([]rune(body)[:800])
	}

	keyPoints := []string{
		"这篇内容主要在解决什么问题",
		"有哪些关键概念或步骤",
		"有没有一个可以迁移到别处的例子",
	}

	return body, keyPoints
}

func openingPrompt(mode string) string {
	if mode == model.ReviewModeDetailedQA {
		return "先别看原文，我们从主线开始，一步一步把它讲清楚。"
	}

	return "先别看原文，试着用自己的话讲讲这篇内容。你不需要一字不差，只要抓住主线。"
}

func initialHints(mode string, keyPoints []string) []string {
	if mode == model.ReviewModeDetailedQA {
		return []string{}
	}

	return append([]string(nil), keyPoints...)
}

func nextDetailedQuestion(answerCount int) string {
	switch answerCount {
	case 0:
		return "这篇文章最核心在讲什么？"
	case 1:
		return "它的关键概念、步骤或关系是什么？"
	default:
		return "如果让你把它讲给一个新手，你会怎么解释？"
	}
}

func buildHintText(session model.ReviewSession, turns []model.ReviewTurn, keyPoints []string) string {
	if session.Mode == model.ReviewModeDetailedQA {
		answerCount := countUserAnswers(turns)
		currentQuestion := nextDetailedQuestion(answerCount)
		if currentQuestion == "" {
			currentQuestion = "把当前问题先拆成结论、依据和例子三层来回答。"
		}

		switch session.HintUsedCount + 1 {
		case 1:
			return "先只回答当前追问，不用一次讲完整篇。当前追问是：" + currentQuestion
		case 2:
			return "把回答聚焦在当前追问，并尽量补上一个关键概念、步骤关系或因果依据。当前追问是：" + currentQuestion
		default:
			return "如果卡住了，就围绕当前追问拆成三句：先给结论，再讲依据，最后补一个例子。当前追问是：" + currentQuestion
		}
	}

	level := session.HintUsedCount + 1
	if len(keyPoints) == 0 {
		keyPoints = []string{
			"先用自己的话概括主线",
			"再补关键概念或步骤",
			"最后给一个迁移例子",
		}
	}

	switch level {
	case 1:
		return "先抓主线，不必追求完整。可以先回答：" + keyPoints[0] + "。"
	case 2:
		return "试着按“问题 -> 关键概念/步骤 -> 一个例子”的顺序来讲，会更容易组织内容。"
	default:
		return "把重点落在这几个点上：" + strings.Join(keyPoints, "；") + "。"
	}
}
