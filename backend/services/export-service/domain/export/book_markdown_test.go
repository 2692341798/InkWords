package export

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func TestBookASTMarkdownUsesCanonicalOrderAndStableProvenanceIDs(t *testing.T) {
	hash := "sha256:" + strings.Repeat("a", 64)
	book, err := sharedtextbook.NewCanonicalBookAST("Go 教材", time.Unix(1, 0), []sharedtextbook.CanonicalBookChapter{
		{ID: "chapter-2", Order: 2, Title: "实践", Markdown: "# 实践", ContentHash: hash},
		{ID: "chapter-1", Order: 1, Title: "概念", Markdown: "# 概念", ContentHash: hash, Assets: []sharedtextbook.CanonicalBookAsset{{ID: "asset-1", StableRef: "inkwords-asset:route", ContentHash: hash, AltText: "路由图"}}},
	})
	require.NoError(t, err)

	markdown, err := RenderBookMarkdown(book)
	require.NoError(t, err)
	text := string(markdown)
	require.Less(t, strings.Index(text, "chapter-1"), strings.Index(text, "chapter-2"))
	require.Contains(t, text, "inkwords:asset:asset-1")
	require.Contains(t, text, "stable-ref:inkwords-asset:route")
}
