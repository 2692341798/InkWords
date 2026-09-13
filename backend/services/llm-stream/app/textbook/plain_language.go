package textbook

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var acronymPattern = regexp.MustCompile(`\b[A-Z][A-Z0-9]{1,}\b`)
var fencedCodeBlock = regexp.MustCompile("(?s)```.*?```")
var acronymDefinition = regexp.MustCompile(`^\s*[（(][^）)\n]{2,80}[）)]`)
var codeBlockStart = regexp.MustCompile("(?m)^```[A-Za-z0-9_-]+[^\\n]*\\n")
var leadingDefinition = regexp.MustCompile(`(?m)^\s*(?:[-*+]\s+|\d+[.)]\s+)?(?:\*\*)?([\p{Han}]{2,16}?|[A-Za-z][A-Za-z0-9_-]{1,})(?:\*\*)?\s*(?:[（(][^）)\n]{1,80}[）)])?\s*(?:是|就是|指)\s*([^。！？!?\n]+)`)

// PlainLanguageRisks flags repeated unexplained acronyms for human editorial review.
func PlainLanguageRisks(markdown string) []string {
	seen := map[string]int{}
	for _, item := range acronymPattern.FindAllString(proseOnly(markdown), -1) {
		seen[item]++
	}
	var risks []string
	for item, count := range seen {
		if count >= 2 {
			risks = append(risks, "repeated_acronym:"+item)
		}
	}
	sort.Strings(risks)
	return risks
}

// UndefinedRepeatedAcronyms returns repeated technical abbreviations whose
// first prose occurrence does not immediately give a reader-facing definition.
// Short request-method names such as GET are deliberately left to the chapter's
// domain-specific teaching checks; this generic guard only covers acronyms with
// four or more characters, where unexplained repetition is a common jargon wall.
func UndefinedRepeatedAcronyms(markdown string) []string {
	prose := proseOnly(markdown)
	matches := acronymPattern.FindAllStringIndex(prose, -1)
	counts := make(map[string]int)
	firstEnds := make(map[string]int)
	firstStarts := make(map[string]int)
	for _, match := range matches {
		acronym := prose[match[0]:match[1]]
		if len(acronym) < 4 {
			continue
		}
		counts[acronym]++
		if _, exists := firstEnds[acronym]; !exists {
			firstEnds[acronym] = match[1]
			firstStarts[acronym] = match[0]
		}
	}

	undefined := make([]string, 0)
	for acronym, count := range counts {
		if count >= 2 && !hasImmediateAcronymDefinition(prose, firstStarts[acronym], firstEnds[acronym]) {
			undefined = append(undefined, acronym)
		}
	}
	sort.Strings(undefined)
	return undefined
}

func hasImmediateAcronymDefinition(prose string, start, end int) bool {
	before, after := prose[:start], prose[end:]
	// Inline code/emphasis delimiters are presentation, not a missing
	// definition. Strip only a matched pair around the first occurrence.
	for _, marker := range []string{"**", "`", "*"} {
		if strings.HasSuffix(before, marker) && strings.HasPrefix(after, marker) {
			after = strings.TrimPrefix(after, marker)
			break
		}
	}
	return acronymDefinition.MatchString(after)
}

// CircularTermDefinitions returns terms whose definition begins by repeating
// the same term, such as "路由是路由的规则". It is intentionally narrow: a
// quality gate must not reject a concrete explanation merely because it later
// mentions the term again in a legitimate comparison or example.
func CircularTermDefinitions(markdown string) []string {
	definitions := leadingDefinition.FindAllStringSubmatch(proseOnly(markdown), -1)
	seen := make(map[string]struct{}, len(definitions))
	for _, definition := range definitions {
		term := strings.TrimSpace(definition[1])
		explanation := strings.TrimLeft(strings.TrimSpace(definition[2]), "\"'“‘")
		if strings.HasPrefix(explanation, term) {
			seen[term] = struct{}{}
		}
	}
	terms := make([]string, 0, len(seen))
	for term := range seen {
		terms = append(terms, term)
	}
	sort.Strings(terms)
	return terms
}

func proseOnly(markdown string) string {
	return strings.TrimSpace(fencedCodeBlock.ReplaceAllString(markdown, ""))
}

// MissingCodeBlockOrigins reports code blocks that are not introduced as a
// teaching implementation, integration example, or upstream source walkthrough.
// The label must be in the immediately preceding paragraph so a later generic
// attribution cannot make an unrelated code block look sourced.
func MissingCodeBlockOrigins(markdown string) []string {
	starts := codeBlockStart.FindAllStringIndex(markdown, -1)
	missing := make([]string, 0)
	for index, start := range starts {
		intro := strings.TrimSpace(markdown[:start[0]])
		if paragraph := strings.LastIndex(intro, "\n\n"); paragraph >= 0 {
			intro = intro[paragraph+2:]
		}
		if !containsCodeOriginLabel(intro) {
			missing = append(missing, fmt.Sprintf("code_block_%d", index+1))
		}
	}
	return missing
}

func containsCodeOriginLabel(intro string) bool {
	return strings.Contains(intro, "代码来源：教学实现") ||
		strings.Contains(intro, "代码来源：集成示例") ||
		strings.Contains(intro, "代码来源：上游源码讲解")
}
