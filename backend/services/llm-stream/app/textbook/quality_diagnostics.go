package textbook

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var diagnosticCredentialMarker = regexp.MustCompile(`(?i)(api[_ -]?key|authorization|bearer|password|secret|token\s*[:=]|sk-[a-z0-9]|密码|密钥)`)

// QualityFailureDiagnostic locates a hard failure without retaining a rejected
// manuscript. Excerpts are plain text, bounded, and never executable artifacts.
type QualityFailureDiagnostic struct {
	Failure     string `json:"failure"`
	Line        int    `json:"line"`
	Area        string `json:"area"`
	Occurrences int    `json:"occurrences"`
	Excerpt     string `json:"excerpt"`
}

// DiagnoseUndefinedAcronyms retains at most ten short first-occurrence probes.
// Code fences are masked at identical byte offsets, preserving manuscript line
// numbers while excluding their contents from both counts and excerpts.
func DiagnoseUndefinedAcronyms(markdown string) []QualityFailureDiagnostic {
	prose := fencedCodeBlock.ReplaceAllStringFunc(markdown, func(block string) string {
		masked := []byte(block)
		for index, value := range masked {
			if value != '\n' && value != '\r' {
				masked[index] = ' '
			}
		}
		return string(masked)
	})
	diagnostics := make([]QualityFailureDiagnostic, 0)
	matches := acronymPattern.FindAllStringIndex(prose, -1)
	for _, acronym := range UndefinedRepeatedAcronyms(markdown) {
		first, count := -1, 0
		for _, match := range matches {
			if prose[match[0]:match[1]] == acronym {
				count++
				if first < 0 {
					first = match[0]
				}
			}
		}
		if first < 0 {
			continue
		}
		start := strings.LastIndex(prose[:first], "\n") + 1
		end := strings.Index(prose[first:], "\n")
		if end < 0 {
			end = len(prose)
		} else {
			end += first
		}
		line := []rune(prose[start:end])
		left := max(0, utf8.RuneCountInString(prose[start:first])-40)
		right := min(len(line), left+160)
		excerpt := strings.TrimSpace(string(line[left:right]))
		if diagnosticCredentialMarker.MatchString(string(line)) {
			excerpt = "[含凭据标记，片段已隐藏]"
		}
		area := "manuscript"
		if strings.Contains(prose[:first], "<!-- inkwords.practice-set.v1:") {
			area = "practice_set"
		}
		diagnostics = append(diagnostics, QualityFailureDiagnostic{
			Failure: "undefined_repeated_acronym: " + acronym,
			Line:    strings.Count(prose[:first], "\n") + 1, Area: area,
			Occurrences: count, Excerpt: excerpt,
		})
		if len(diagnostics) == 10 {
			break
		}
	}
	return diagnostics
}
