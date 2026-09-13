package textbook

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func bookQualityFixture(t *testing.T) BookQualitySnapshot {
	t.Helper()
	raw := json.RawMessage(`{"contract_version":"` + SampleQualityContractVersion + `","passed":true,"manual_review_required":true,"advisories":[{"code":"long_paragraph"}]}`)
	hash, err := BookQualityReportHash(raw)
	require.NoError(t, err)
	book, style := uuid.NewString(), uuid.NewString()
	return BookQualitySnapshot{Format: BookQualitySnapshotFormat, CapturedAt: time.Unix(100, 0).UTC(), BookContractRevisionID: book, StyleSheetRevisionID: style, Chapters: []FrozenChapterQuality{{ChapterID: uuid.NewString(), RevisionID: uuid.NewString(), ContentHash: "sha256:" + strings.Repeat("a", 64), BookContractRevisionID: book, StyleSheetRevisionID: style, ReportHash: hash, Report: raw}}}
}

func TestBookQualitySnapshotKeepsAutomaticReportAndRejectsStaleOrFalseGates(t *testing.T) {
	s := bookQualityFixture(t)
	require.NoError(t, s.Validate())
	require.True(t, s.Assess().Passed)
	before, _ := json.Marshal(s)
	s.Assess()
	after, _ := json.Marshal(s)
	require.Equal(t, string(before), string(after))
	for _, raw := range []string{`null`, `{"passed":true}`, `{"contract_version":"old","passed":true}`, `{"contract_version":"` + SampleQualityContractVersion + `","passed":false}`, `{"contract_version":"` + SampleQualityContractVersion + `","passed":true,"failures":["broken code"]}`} {
		changed := bookQualityFixture(t)
		changed.Chapters[0].Report = json.RawMessage(raw)
		hash, err := BookQualityReportHash(changed.Chapters[0].Report)
		require.NoError(t, err)
		changed.Chapters[0].ReportHash = hash
		require.False(t, changed.Assess().Passed)
	}
	changed := bookQualityFixture(t)
	changed.Chapters[0].BookContractRevisionID = uuid.NewString()
	require.False(t, changed.Assess().Passed)
	changed = bookQualityFixture(t)
	changed.Chapters[0].Report = json.RawMessage(`{"changed":true}`)
	require.Error(t, changed.Validate())
}

func TestReadBookQualitySnapshotBindsEveryChapterAndContract(t *testing.T) {
	s := bookQualityFixture(t)
	envelope := map[string]any{"quality_snapshot": s, "chapters": s.Chapters, "book_contract_revision_id": s.BookContractRevisionID, "style_sheet_revision_id": s.StyleSheetRevisionID}
	raw, _ := json.Marshal(envelope)
	got, err := ReadBookQualitySnapshot(raw)
	require.NoError(t, err)
	require.Equal(t, s.CapturedAt, got.CapturedAt)
	for _, key := range []string{"revision_id", "content_hash", "chapter_id"} {
		var changed map[string]any
		require.NoError(t, json.Unmarshal(raw, &changed))
		changed["chapters"].([]any)[0].(map[string]any)[key] = "foreign"
		bad, _ := json.Marshal(changed)
		_, err := ReadBookQualitySnapshot(bad)
		require.Error(t, err, key)
	}
	envelope["tool_versions"] = map[string]string{"quality_snapshot": "unknown"}
	bad, _ := json.Marshal(envelope)
	_, err = ReadBookQualitySnapshot(bad)
	require.Error(t, err)
	delete(envelope, "tool_versions")
	// JSONB can reorder object keys and whitespace without altering the report.
	hash, err := BookQualityReportHash(json.RawMessage(`{ "b": 2, "a": 1 }`))
	require.NoError(t, err)
	other, err := BookQualityReportHash(json.RawMessage(`{"a":1,"b":2}`))
	require.NoError(t, err)
	require.Equal(t, hash, other)
	largeA, err := BookQualityReportHash(json.RawMessage(`{"count":9007199254740992}`))
	require.NoError(t, err)
	largeB, err := BookQualityReportHash(json.RawMessage(`{"count":9007199254740993}`))
	require.NoError(t, err)
	require.NotEqual(t, largeA, largeB, "hashing cannot round report numbers")
	_, err = BookQualityReportHash(json.RawMessage(`{} {}`))
	require.Error(t, err)
	envelope["chapters"] = []any{}
	raw, _ = json.Marshal(envelope)
	_, err = ReadBookQualitySnapshot(raw)
	require.Error(t, err)
	for _, raw := range []string{`{"quality_snapshot":null}`, `{"quality_snapshot":{}}`, `{"tool_versions":{"quality_snapshot":"required"}}`} {
		_, err := ReadBookQualitySnapshot([]byte(raw))
		require.Error(t, err)
	}
	legacy, err := ReadBookQualitySnapshot([]byte(`{"format":"inkwords.book-build.v1"}`))
	require.NoError(t, err)
	require.Nil(t, legacy)
}
