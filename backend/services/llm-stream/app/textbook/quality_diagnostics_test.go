package textbook

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestAcronymDiagnosticsLocateFirstProseOccurrenceAndExcludeCode(t *testing.T) {
	markdown := "# 教材\n\n```go\n// POST POST private-code-marker\n```\n\n首次 POST 请求没有解释。\n之后 POST 仍没有解释。"
	diagnostics := DiagnoseUndefinedAcronyms(markdown)
	require.Len(t, diagnostics, 1)
	require.Equal(t, 7, diagnostics[0].Line)
	require.Equal(t, 2, diagnostics[0].Occurrences)
	require.Contains(t, diagnostics[0].Excerpt, "首次 POST")
	require.NotContains(t, diagnostics[0].Excerpt, "private-code-marker")
	require.Equal(t, "manuscript", diagnostics[0].Area)
	require.Empty(t, DiagnoseUndefinedAcronyms("`POST`（提交方法）说明。POST 再次出现。"))
}

func TestAcronymDiagnosticsBoundOutputAndLocatePracticeText(t *testing.T) {
	markdown := "## 六维练习与答案\n\n<!-- inkwords.practice-set.v1:hash -->\n\n" + strings.Repeat("中", 300) + "POST 请求。POST 请求。" + strings.Repeat("文", 300)
	diagnostics := DiagnoseUndefinedAcronyms(markdown)
	require.Len(t, diagnostics, 1)
	require.Equal(t, "practice_set", diagnostics[0].Area)
	require.True(t, utf8.ValidString(diagnostics[0].Excerpt))
	require.LessOrEqual(t, utf8.RuneCountInString(diagnostics[0].Excerpt), 160)
	require.Contains(t, diagnostics[0].Excerpt, "POST")
}

func TestAcronymDiagnosticHidesCredentialMarkedLine(t *testing.T) {
	diagnostics := DiagnoseUndefinedAcronyms("POST 请求 password=private-marker\nPOST 再出现")
	require.Len(t, diagnostics, 1)
	require.Equal(t, "[含凭据标记，片段已隐藏]", diagnostics[0].Excerpt)
}
