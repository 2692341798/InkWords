package export

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	shared "inkwords-backend/shared/kernel/textbook"
)

func TestPublicationNoticesProjectFromFrozenASTWithoutChangingChapter(t *testing.T) {
	book, err := shared.NewCanonicalBookAST("教材", time.Unix(1, 0), []shared.CanonicalBookChapter{{ID: "chapter", Order: 1, Title: "章", Markdown: "# 章\n\n原文", ContentHash: "sha256:" + strings.Repeat("a", 64)}})
	require.NoError(t, err)
	text := "Copyright fixture\n~~~\n<script>alert('untrusted')</script>\n"
	notices, err := shared.FreezePublicationNotices([]shared.PublicationNoticeDraft{{Title: "许可", Text: text, PreparedBy: "委托 AI", SourceURL: "https://example.org/LICENSE", SubjectRefs: []string{"code-artifact:00000000-0000-4000-8000-000000000001"}}})
	require.NoError(t, err)
	book.Notices = notices
	require.Error(t, book.Validate(), "old AST formats must not silently accept new publication text")
	book.Format = shared.CanonicalBookASTWithNoticesFormat
	markdown, err := RenderBookMarkdown(book)
	require.NoError(t, err)
	require.Contains(t, string(markdown), text)
	require.Equal(t, "# 章\n\n原文", book.Chapters[0].Markdown)
	html, err := RenderBookHTML(book)
	require.NoError(t, err)
	require.NotContains(t, string(html), "<script>alert")
	require.Contains(t, string(html), "&lt;script&gt;")
	files := bookNoticeFiles(book)
	require.Len(t, files, 2)
	require.Equal(t, "code/00000000-0000-4000-8000-000000000001/THIRD-PARTY-NOTICES.txt", files[0].Path)
	require.Contains(t, string(files[0].Content), text)
	book.Notices[0].Text += "changed"
	_, err = RenderBookMarkdown(book)
	require.Error(t, err)
}
