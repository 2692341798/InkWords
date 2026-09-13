package export

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	shared "inkwords-backend/shared/kernel/textbook"
	"strings"
	"testing"
	"time"
)

func TestFrozenBookQualityReportDoesNotPromoteChapterChecksToBookApproval(t *testing.T) {
	now := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	raw := json.RawMessage(`{"contract_version":"` + shared.SampleQualityContractVersion + `","passed":true,"manual_review_required":true,"advisories":[{"code":"long_paragraph"}]}`)
	hash, err := shared.BookQualityReportHash(raw)
	require.NoError(t, err)
	snapshot := shared.BookQualitySnapshot{Format: shared.BookQualitySnapshotFormat, CapturedAt: now, BookContractRevisionID: "book-r1", StyleSheetRevisionID: "style-r1", Chapters: []shared.FrozenChapterQuality{{ChapterID: "chapter-1", RevisionID: "revision-1", ContentHash: "sha256:" + strings.Repeat("a", 64), BookContractRevisionID: "book-r1", StyleSheetRevisionID: "style-r1", ReportHash: hash, Report: raw}}}
	manifest, err := json.Marshal(map[string]any{"quality_snapshot": snapshot, "chapters": snapshot.Chapters, "book_contract_revision_id": "book-r1", "style_sheet_revision_id": "style-r1"})
	require.NoError(t, err)
	build := TextbookBookBuildExport{BuildID: uuid.New(), ManifestHash: hash, ManifestJSON: manifest, HumanReviews: []shared.HumanPublicationReview{}, DelegatedReviews: []shared.DelegatedPublicationReview{}, Preflight: shared.PublicationPreflightResult{Passed: false, Blockers: []string{"缺少权利清单。"}}}
	out, err := frozenBookQualityReport(build, now)
	require.NoError(t, err)
	var report struct {
		ChapterSnapshot   shared.BookQualitySnapshot        `json:"chapter_snapshot"`
		ChapterAssessment shared.BookQualityAssessment      `json:"chapter_assessment"`
		Preflight         shared.PublicationPreflightResult `json:"publication_preflight"`
		HumanReviews      []shared.HumanPublicationReview   `json:"human_reviews"`
	}
	require.NoError(t, json.Unmarshal(out, &report))
	require.True(t, report.ChapterAssessment.Passed)
	require.False(t, report.Preflight.Passed)
	require.Empty(t, report.HumanReviews)
	require.JSONEq(t, string(raw), string(report.ChapterSnapshot.Chapters[0].Report))
	require.Contains(t, string(out), "manual_review_required")
	require.Contains(t, string(out), "long_paragraph")
	require.Contains(t, string(out), build.BuildID.String())
	build.ManifestJSON = json.RawMessage(`{"format":"inkwords.book-build.v1"}`)
	out, err = frozenBookQualityReport(build, now)
	require.NoError(t, err)
	require.Contains(t, string(out), `"chapter_snapshot_status":"unavailable"`)
	require.Contains(t, string(out), `"chapter_assessment":null`)
	require.Contains(t, string(out), "缺少权利清单。")
	build.ManifestJSON = json.RawMessage(`{"quality_snapshot":null}`)
	_, err = frozenBookQualityReport(build, now)
	require.Error(t, err)
}
