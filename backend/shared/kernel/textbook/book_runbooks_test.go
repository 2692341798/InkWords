package textbook

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFrozenVideoRunbooksKeepMissingGuidesAndRejectBrokenBindings(t *testing.T) {
	book, err := NewCanonicalBookAST("测试", time.Unix(1, 0), []CanonicalBookChapter{{ID: "chapter", Order: 1, Title: "标题", Markdown: "正文", ContentHash: "sha256:" + strings.Repeat("a", 64)}})
	require.NoError(t, err)
	hash, err := VideoRunbookHash(nil)
	require.NoError(t, err)
	chapter := FrozenChapterVideoRunbook{ChapterID: "chapter", RevisionID: "approved-revision", ContentHash: book.Chapters[0].ContentHash, RunbookHash: hash}
	manifest := func(chapters []FrozenChapterVideoRunbook, version string) []byte {
		raw, marshalErr := json.Marshal(map[string]any{"book": book, "chapters": chapters, "tool_versions": map[string]string{"video_runbooks": version}})
		require.NoError(t, marshalErr)
		return raw
	}
	snapshot, err := ReadBookVideoRunbooks(manifest([]FrozenChapterVideoRunbook{chapter}, BookVideoRunbooksFormat))
	require.NoError(t, err)
	require.Nil(t, snapshot.Chapters[0].VideoRunbook, "absence must not become a generated guide")
	for name, mutate := range map[string]func(*FrozenChapterVideoRunbook){
		"cross chapter":    func(c *FrozenChapterVideoRunbook) { c.ChapterID = "other" },
		"missing revision": func(c *FrozenChapterVideoRunbook) { c.RevisionID = "" },
		"changed bytes":    func(c *FrozenChapterVideoRunbook) { c.ContentHash = "sha256:" + strings.Repeat("b", 64) },
		"changed hash":     func(c *FrozenChapterVideoRunbook) { c.RunbookHash = "sha256:" + strings.Repeat("c", 64) },
		"malformed guide":  func(c *FrozenChapterVideoRunbook) { c.VideoRunbook = &VideoRunbookProjection{Format: "unknown"} },
	} {
		t.Run(name, func(t *testing.T) {
			changed := chapter
			mutate(&changed)
			_, err := ReadBookVideoRunbooks(manifest([]FrozenChapterVideoRunbook{changed}, BookVideoRunbooksFormat))
			require.Error(t, err)
		})
	}
	for _, version := range []string{"", "future-format"} {
		_, err := ReadBookVideoRunbooks(manifest([]FrozenChapterVideoRunbook{chapter}, version))
		require.Error(t, err)
	}
	for _, chapters := range [][]FrozenChapterVideoRunbook{nil, {chapter, chapter}} {
		_, err := ReadBookVideoRunbooks(manifest(chapters, BookVideoRunbooksFormat))
		require.Error(t, err)
	}
	legacy, err := ReadBookVideoRunbooks([]byte(`{"chapters":[{"chapter_id":"chapter","revision_id":"old"}]}`))
	require.NoError(t, err)
	require.Nil(t, legacy)
}
