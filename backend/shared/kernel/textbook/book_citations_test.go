package textbook

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBookEvidenceMarkersRespectMarkdownCodeAndEscapes(t *testing.T) {
	text := "正文 [evidence:route] 与 [evidence:tree]. `" + "[evidence:inline]`\n\n```go\n[evidence:fenced]\n```\n\n\\[evidence:escaped]\n"
	markers := BookEvidenceMarkers(text)
	require.Len(t, markers, 2)
	require.Equal(t, "route", markers[0].ID)
	require.Equal(t, "tree", markers[1].ID)
	for _, marker := range markers {
		require.Equal(t, "[evidence:"+marker.ID+"]", text[marker.Start:marker.End])
	}
}

func TestFrozenBookCitationValidationAndLegacyCompatibility(t *testing.T) {
	hash := "sha256:" + strings.Repeat("a", 64)
	snapshot := SourceSnapshot{ID: "snapshot", SourceID: "source", Kind: SourceKindGitRepository, Role: SourceRolePrimary, Locator: "https://github.com/gin-gonic/gin", ResolvedVersion: strings.Repeat("b", 40), ContentHash: hash, CapturedAt: time.Unix(1, 0)}
	reference := EvidenceRef{ID: "evidence", SnapshotID: snapshot.ID, DocumentID: "doc", Locator: EvidenceLocator{Path: "tree.go", StartLine: 1, EndLine: 2}, ContentHash: hash, Confidence: EvidenceConfidenceDocumented, SourceRole: snapshot.Role}
	chapter := CanonicalBookChapter{ID: "chapter", Order: 1, Title: "标题", Markdown: "[evidence:route]", ContentHash: hash, Citations: []CanonicalBookCitation{{ID: "route", Evidence: reference, Snapshot: snapshot}}}
	book, err := NewCanonicalBookAST("书", time.Unix(1, 0), []CanonicalBookChapter{chapter})
	require.NoError(t, err)
	book.Chapters[0].Citations = append(book.Chapters[0].Citations, chapter.Citations[0])
	require.Error(t, book.Validate())
	book.Chapters[0].Citations = append([]CanonicalBookCitation(nil), chapter.Citations...)
	book.Chapters[0].Citations[0].Snapshot.ID = "different-snapshot"
	require.Error(t, book.Validate())
	book.Format = "inkwords.book-ast.v1"
	book.Chapters[0].Citations = nil
	require.ErrorContains(t, book.Validate(), "no frozen source")
	book.Chapters[0].Markdown = "旧构建未使用证据标记。"
	require.NoError(t, book.Validate())
}
