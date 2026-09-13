package export

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	shared "inkwords-backend/shared/kernel/textbook"
)

func citedBook(t *testing.T) shared.CanonicalBookAST {
	t.Helper()
	hash := "sha256:" + strings.Repeat("a", 64)
	snapshot := shared.SourceSnapshot{ID: "snapshot", SourceID: "source", Kind: shared.SourceKindGitRepository, Role: shared.SourceRolePrimary, Locator: "https://github.com/gin-gonic/gin.git", ResolvedVersion: strings.Repeat("b", 40), ContentHash: hash, CapturedAt: time.Unix(1, 0)}
	reference := shared.EvidenceRef{ID: "evidence-chunk", SnapshotID: snapshot.ID, DocumentID: "document", ChunkID: "chunk", Locator: shared.EvidenceLocator{Path: "tree.go", Symbol: "node.getValue", StartLine: 10, EndLine: 20}, ContentHash: hash, Confidence: shared.EvidenceConfidenceDocumented, SourceRole: snapshot.Role}
	book, err := shared.NewCanonicalBookAST("引用测试", time.Unix(1, 0), []shared.CanonicalBookChapter{{ID: "chapter", Order: 1, Title: "路由", Markdown: "正文 [evidence:route]，再次 [evidence:route]。\n\n`[evidence:inline]`\n\n```go\n[evidence:fenced]\n```\n\n\\[evidence:escaped]", ContentHash: hash, Citations: []shared.CanonicalBookCitation{{ID: "route", Evidence: reference, Snapshot: snapshot}}}})
	require.NoError(t, err)
	return book
}

func TestBookCitationsRenderFixedSourceFootnotesWithoutChangingManuscript(t *testing.T) {
	book := citedBook(t)
	original := book.Chapters[0].Markdown
	markdown, err := RenderBookMarkdown(book)
	require.NoError(t, err)
	text := string(markdown)
	require.NotContains(t, text, "[evidence:route]")
	require.Equal(t, 3, strings.Count(text, "[^c1-s1]"))
	require.Contains(t, text, "https://github.com/gin-gonic/gin/blob/"+strings.Repeat("b", 40)+"/tree.go#L10-L20")
	require.Contains(t, text, "node.getValue")
	for _, literal := range []string{"`[evidence:inline]`", "```go\n[evidence:fenced]\n```", "\\[evidence:escaped]"} {
		require.Contains(t, text, literal)
	}
	require.Equal(t, original, book.Chapters[0].Markdown)
	html, err := RenderBookHTML(book)
	require.NoError(t, err)
	require.Contains(t, string(html), `class="footnote-ref"`)
	require.Contains(t, string(html), "tree.go#L10-L20")
}

func TestBookCitationsRejectMissingFrozenSourcesAndEscapeUnsafeMetadata(t *testing.T) {
	book := citedBook(t)
	book.Chapters[0].Citations = nil
	_, err := RenderBookMarkdown(book)
	require.ErrorContains(t, err, "no frozen source")
	book = citedBook(t)
	book.Chapters[0].Citations[0].Snapshot.Locator = "javascript:alert(1)"
	book.Chapters[0].Citations[0].Evidence.Locator.Symbol = "<script>alert(1)</script>"
	markdown, err := RenderBookMarkdown(book)
	require.NoError(t, err)
	require.NotContains(t, string(markdown), "[查看来源]")
	require.Contains(t, string(markdown), "来源快照 snapshot")
	html, err := RenderBookHTML(book)
	require.NoError(t, err)
	require.NotContains(t, string(html), "<script>")
	require.NotContains(t, string(html), "javascript:")
}
