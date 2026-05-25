package llm

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var thinkBlockPattern = regexp.MustCompile(`(?s)<think>.*?</think>`)

func sanitizeLeadingGeneratedText(content string) string {
	cleaned := strings.TrimSpace(thinkBlockPattern.ReplaceAllString(content, ""))
	if cleaned == "" {
		return ""
	}

	paragraphs := splitMarkdownParagraphs(cleaned)
	dropCount := 0
	for dropCount < len(paragraphs) && isLeadingMetaParagraph(paragraphs[dropCount]) {
		dropCount++
	}
	if dropCount == 0 {
		return cleaned
	}

	return strings.TrimSpace(strings.Join(paragraphs[dropCount:], "\n\n"))
}

func shouldHoldLeadingSanitization(buffer string) bool {
	trimmed := strings.TrimSpace(buffer)
	if trimmed == "" {
		return true
	}
	if strings.Contains(trimmed, "\n\n") {
		return false
	}
	if strings.Contains(trimmed, "\n#") || strings.HasPrefix(trimmed, "#") {
		return false
	}
	return utf8.RuneCountInString(trimmed) < 240
}

func splitMarkdownParagraphs(content string) []string {
	return regexp.MustCompile(`\n\s*\n`).Split(content, -1)
}

func isLeadingMetaParagraph(paragraph string) bool {
	trimmed := strings.TrimSpace(paragraph)
	if trimmed == "" {
		return false
	}
	if startsWithMarkdownContent(trimmed) {
		return false
	}

	hasRoleIntro := strings.Contains(trimmed, "作为高级") ||
		strings.Contains(trimmed, "作为一名") ||
		(strings.Contains(trimmed, "作为") &&
			(strings.Contains(trimmed, "架构师") ||
				strings.Contains(trimmed, "博主") ||
				strings.Contains(trimmed, "助手") ||
				strings.Contains(trimmed, "AI")))

	hasTaskTalk := strings.Contains(trimmed, "收到你的需求") ||
		strings.Contains(trimmed, "根据你提供") ||
		strings.Contains(trimmed, "我将根据") ||
		strings.Contains(trimmed, "我会根据") ||
		strings.Contains(trimmed, "以下是根据") ||
		strings.Contains(trimmed, "我将为你") ||
		strings.Contains(trimmed, "接下来我将") ||
		strings.Contains(trimmed, "下面我将")

	hasGenerationIntent := strings.Contains(trimmed, "撰写") ||
		strings.Contains(trimmed, "生成") ||
		strings.Contains(trimmed, "整理") ||
		strings.Contains(trimmed, "输出") ||
		strings.Contains(trimmed, "博客") ||
		strings.Contains(trimmed, "文章")

	return hasRoleIntro || (hasTaskTalk && hasGenerationIntent)
}

func startsWithMarkdownContent(trimmed string) bool {
	return strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, ">") ||
		strings.HasPrefix(trimmed, "- ") ||
		strings.HasPrefix(trimmed, "* ") ||
		strings.HasPrefix(trimmed, "```") ||
		strings.HasPrefix(trimmed, "|") ||
		regexp.MustCompile(`^\d+\.\s`).MatchString(trimmed)
}
