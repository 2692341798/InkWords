package textbook

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewCanonicalBookASTSortsAndValidatesFrozenChapters(t *testing.T) {
	hash := "sha256:" + strings.Repeat("a", 64)
	document, err := NewCanonicalBookAST("Go 入门", time.Unix(1, 0), []CanonicalBookChapter{
		{ID: "chapter-2", Order: 2, Title: "实践", Markdown: "# 实践", ContentHash: hash},
		{ID: "chapter-1", Order: 1, Title: "概念", Markdown: "# 概念", ContentHash: hash},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"chapter-1", "chapter-2"}, []string{document.Chapters[0].ID, document.Chapters[1].ID})

	_, err = NewCanonicalBookAST("", time.Unix(1, 0), []CanonicalBookChapter{{ID: "chapter", Order: 1, Title: "章节", Markdown: "# 章节", ContentHash: hash}})
	require.Error(t, err)
}
