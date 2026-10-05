package export

import (
	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"strings"
	"testing"
	"time"
)

func TestRenderBookHTMLUsesTheCanonicalBookAST(t *testing.T) {
	hash := "sha256:" + strings.Repeat("a", 64)
	book, err := sharedtextbook.NewCanonicalBookAST("Gin 教材", time.Unix(1, 0), []sharedtextbook.CanonicalBookChapter{
		{ID: "chapter-2", Order: 2, Title: "实践", Markdown: "# 实践\n\n第二章内容。", ContentHash: hash},
		{ID: "chapter-1", Order: 1, Title: "概念", Markdown: "# 概念\n\n第一章内容。", ContentHash: hash},
	})
	require.NoError(t, err)
	result, err := RenderBookHTML(book)
	require.NoError(t, err)
	text := string(result)
	require.Less(t, strings.Index(text, "第一章内容"), strings.Index(text, "第二章内容"))
	require.Contains(t, text, "Content-Security-Policy")
	require.Contains(t, text, "font-src 'none'")
	require.Contains(t, text, "Gin 教材")
}
