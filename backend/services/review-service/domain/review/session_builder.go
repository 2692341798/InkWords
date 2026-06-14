package review

import (
	"strings"
	"unicode"
)

func buildSessionSnapshot(note ReviewNote) (string, SessionOutline, string) {
	body := truncateRunes(strings.TrimSpace(note.Body), 800)
	outline := buildSessionOutline(note)
	sourcePreview := truncateRunes(strings.TrimSpace(note.Body), 2400)
	return body, outline, sourcePreview
}

func buildSessionOutline(note ReviewNote) SessionOutline {
	fragments := splitSentenceFragments(note.Body)
	checkpoints := pickDistinctFragments(fragments, 3)
	if len(checkpoints) == 0 {
		checkpoints = []string{
			"先讲清楚这篇内容主要在解决什么问题",
			"再补关键概念、步骤或因果关系",
			"最后补一个适用场景或例子",
		}
	}

	coreConcepts := extractCoreConcepts(checkpoints)
	processSteps := pickStepFragments(fragments, checkpoints)
	applicationCases := pickApplicationFragments(fragments, checkpoints)

	return SessionOutline{
		Summary:          firstNonEmpty(truncateRunes(strings.TrimSpace(note.Body), 180), note.Title),
		MainQuestion:     buildMainQuestion(note.Title, checkpoints),
		CoreConcepts:     coreConcepts,
		ProcessSteps:     processSteps,
		ApplicationCases: applicationCases,
		Checkpoints:      checkpoints,
	}
}

func openingPrompt(mode string, outline SessionOutline) string {
	if mode == ReviewModeDetailedQA {
		return "先别看原文，我们先围绕这篇文章的主线来回答：" + firstNonEmpty(outline.MainQuestion, "这篇文章最核心在讲什么？")
	}

	return "先别看原文，试着用自己的话讲讲这篇内容。你不需要一字不差，只要先抓住主线：" + firstNonEmpty(outline.MainQuestion, "这篇内容主要在解决什么问题？")
}

func initialHints(mode string, outline SessionOutline) []string {
	if mode == ReviewModeDetailedQA {
		return []string{}
	}
	if len(outline.Checkpoints) > 0 {
		limit := len(outline.Checkpoints)
		if limit > 3 {
			limit = 3
		}
		return append([]string(nil), outline.Checkpoints[:limit]...)
	}
	return []string{
		"先讲它主要在解决什么问题",
		"再补关键概念或步骤",
		"最后给一个例子或适用场景",
	}
}

func currentRoundGoal(mode string, answerCount int, outline SessionOutline) string {
	if mode == ReviewModeDetailedQA {
		switch answerCount {
		case 0:
			return "先讲清楚这篇文章的主线问题和核心结论。"
		case 1:
			return "继续补上关键概念、步骤关系或为什么这样设计。"
		default:
			return "尝试把它迁移到场景、例子或给新手的解释中。"
		}
	}

	if answerCount == 0 {
		return "先把你记得的主线讲出来，不必追求一次讲全。"
	}
	return "继续对照遗漏点，把关键概念、步骤或例子补完整。"
}

func nextDetailedQuestion(answerCount int, outline SessionOutline) string {
	switch answerCount {
	case 0:
		return firstNonEmpty(outline.MainQuestion, "这篇文章最核心在讲什么？")
	case 1:
		if len(outline.CoreConcepts) > 0 {
			return "这篇文章里最关键的概念或关系是什么？请至少讲到：" + strings.Join(outline.CoreConcepts, "、") + "。"
		}
		if len(outline.ProcessSteps) > 0 {
			return "把它的关键步骤或关系顺着讲一遍，重点提到：" + strings.Join(outline.ProcessSteps, "；") + "。"
		}
		return "它的关键概念、步骤或关系是什么？"
	default:
		if len(outline.ApplicationCases) > 0 {
			return "如果让你把它讲给一个新手，或者放到实际场景里，你会怎么解释？可以结合：" + strings.Join(outline.ApplicationCases, "；") + "。"
		}
		return "如果让你把它讲给一个新手，你会怎么解释？"
	}
}

func buildHintText(session ReviewSession, turns []ReviewTurn, outline SessionOutline) string {
	if session.Mode == ReviewModeDetailedQA {
		answerCount := countUserAnswers(turns)
		currentQuestion := nextDetailedQuestion(answerCount, outline)
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

	keyPoints := outline.Checkpoints
	if len(keyPoints) == 0 {
		keyPoints = []string{
			"先用自己的话概括主线",
			"再补关键概念或步骤",
			"最后给一个迁移例子",
		}
	}

	switch session.HintUsedCount + 1 {
	case 1:
		return "先抓主线，不必追求完整。可以先回答：" + keyPoints[0] + "。"
	case 2:
		return "试着按“问题 -> 关键概念/步骤 -> 一个例子”的顺序来讲，会更容易组织内容。"
	default:
		return "把重点落在这几个点上：" + strings.Join(keyPoints, "；") + "。"
	}
}

func truncateRunes(text string, limit int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit])
}

func splitSentenceFragments(body string) []string {
	parts := strings.FieldsFunc(strings.TrimSpace(body), func(r rune) bool {
		return r == '。' || r == '！' || r == '？' || r == '\n' || r == ';' || r == '；'
	})

	items := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if len([]rune(trimmed)) < 8 {
			continue
		}
		items = append(items, trimmed)
	}
	return items
}

func pickDistinctFragments(fragments []string, limit int) []string {
	items := make([]string, 0, limit)
	seen := make(map[string]struct{}, limit)
	for _, fragment := range fragments {
		normalized := normalizeForMatch(fragment)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		items = append(items, fragment)
		if len(items) == limit {
			break
		}
	}
	return items
}

func extractCoreConcepts(checkpoints []string) []string {
	items := make([]string, 0, len(checkpoints))
	for _, checkpoint := range checkpoints {
		items = append(items, buildCheckpointTopic(checkpoint))
		if len(items) == 2 {
			break
		}
	}
	return compactNonEmpty(items, 2)
}

func pickStepFragments(fragments []string, checkpoints []string) []string {
	items := make([]string, 0, 2)
	for _, fragment := range fragments {
		if strings.Contains(fragment, "先") || strings.Contains(fragment, "再") || strings.Contains(fragment, "然后") || strings.Contains(fragment, "最后") || strings.Contains(fragment, "通常") {
			items = append(items, fragment)
		}
		if len(items) == 2 {
			return items
		}
	}
	return compactNonEmpty(checkpoints, 2)
}

func pickApplicationFragments(fragments []string, checkpoints []string) []string {
	items := make([]string, 0, 2)
	for _, fragment := range fragments {
		if strings.Contains(fragment, "例如") || strings.Contains(fragment, "比如") || strings.Contains(fragment, "场景") || strings.Contains(fragment, "落地") || strings.Contains(fragment, "实战") || strings.Contains(fragment, "适用") {
			items = append(items, fragment)
		}
	}
	if len(items) > 0 {
		return compactNonEmpty(items, 2)
	}
	return compactNonEmpty(checkpoints, 1)
}

func buildMainQuestion(title string, checkpoints []string) string {
	firstCheckpoint := ""
	if len(checkpoints) > 0 {
		firstCheckpoint = checkpoints[0]
	}
	title = strings.TrimSpace(title)
	switch {
	case title != "" && firstCheckpoint != "":
		return "围绕“" + title + "”这篇文章，先讲清楚它主要在解决什么问题，并带到这条主线：" + firstCheckpoint + "。"
	case title != "":
		return "围绕“" + title + "”这篇文章，先讲清楚它主要在解决什么问题。"
	case firstCheckpoint != "":
		return "先讲清楚这篇文章的主线，并带到这个关键点：" + firstCheckpoint + "。"
	default:
		return "这篇文章最核心在讲什么？"
	}
}

func buildCheckpointTopic(checkpoint string) string {
	for _, marker := range []string{"负责", "用于", "用来", "限制", "控制", "避免", "保证", "协调"} {
		if idx := strings.Index(checkpoint, marker); idx > 0 {
			return strings.TrimSpace(checkpoint[:idx])
		}
	}
	return checkpoint
}

func compactNonEmpty(items []string, limit int) []string {
	result := make([]string, 0, limit)
	seen := map[string]struct{}{}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
		if len(result) == limit {
			break
		}
	}
	return result
}

func normalizeForMatch(text string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(text) {
		if unicode.IsSpace(r) {
			continue
		}
		switch r {
		case '，', '。', '、', '；', '：', '！', '？', ',', '.', ';', ':', '(', ')', '（', '）', '"', '\'', '“', '”':
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}
