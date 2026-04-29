package service

import "strings"

func applyMarkdownTitle(title string, body string) string {
	title = strings.TrimSpace(title)
	rest := stripLeadingH1(body)
	if rest == "" {
		return "# " + title + "\n"
	}

	return "# " + title + "\n\n" + demoteH1(rest)
}

func buildObsidianBody(body string) string {
	rest := stripLeadingH1(body)
	if rest == "" {
		return ""
	}
	return demoteH1(rest)
}

func stripLeadingH1(body string) string {
	bodyLines := strings.Split(body, "\n")

	lineIndex := 0
	for lineIndex < len(bodyLines) && strings.TrimSpace(bodyLines[lineIndex]) == "" {
		lineIndex++
	}

	if lineIndex < len(bodyLines) && strings.HasPrefix(bodyLines[lineIndex], "# ") {
		lineIndex++
		for lineIndex < len(bodyLines) && strings.TrimSpace(bodyLines[lineIndex]) == "" {
			lineIndex++
		}
	}

	return strings.Join(bodyLines[lineIndex:], "\n")
}

func demoteH1(body string) string {
	lines := strings.Split(body, "\n")
	inFence := false

	for i := range lines {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		left := strings.TrimLeft(lines[i], " \t")
		if !strings.HasPrefix(left, "# ") {
			continue
		}
		indent := lines[i][:len(lines[i])-len(left)]
		lines[i] = indent + "## " + strings.TrimPrefix(left, "# ")
	}

	return strings.Join(lines, "\n")
}
