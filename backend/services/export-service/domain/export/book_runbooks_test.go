package export

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	shared "inkwords-backend/shared/kernel/textbook"
)

func TestBookPackageIncludesFrozenGuidesAndHashesWithoutChangingProse(t *testing.T) {
	book, err := shared.NewCanonicalBookAST("教材", time.Unix(1, 0), []shared.CanonicalBookChapter{{ID: "chapter", Title: "章节", Order: 1, Markdown: "只有教材正文", ContentHash: "sha256:" + strings.Repeat("a", 64)}})
	require.NoError(t, err)
	guide := shared.VideoRunbookProjection{
		Format: "inkwords.video-runbook.v1", Stack: "go", ObservationGoal: "同源定位",
		Recommendation:   shared.DemonstrationToolRecommendation{Primary: "VS Code", Alternative: "编辑器", Reason: "静态查看", ManualCaptureRequired: true},
		Steps:            []shared.VideoRunbookStep{{Tool: "VS Code", ToolVersion: "1.137.0", StartState: "原字节", Action: "定位真实测试", ShortcutOrMenu: "转到行", Input: "TestMethodSpecific:49", ExpectedView: "测试断言", Narration: "源码不代表运行成功", CapturePoint: "tests", Recovery: "核对哈希", CompletionSignal: "静态定位完成"}},
		CaptureChecklist: []string{"记录修订和文件摘要"}, ManualCapturePending: true, VerificationStatus: shared.ArtifactStatusUnverified,
	}
	hash, err := shared.VideoRunbookHash(&guide)
	require.NoError(t, err)
	manifest, err := json.Marshal(map[string]any{"book": book, "tool_versions": map[string]string{"video_runbooks": shared.BookVideoRunbooksFormat}, "chapters": []shared.FrozenChapterVideoRunbook{{ChapterID: "chapter", RevisionID: "approved-r15", ContentHash: book.Chapters[0].ContentHash, VideoRunbook: &guide, RunbookHash: hash}}})
	require.NoError(t, err)
	var archive bytes.Buffer
	require.NoError(t, NewBookPackageBuilder().Build(&archive, BookPackageInput{Book: book, BuildManifest: manifest}))
	reader, err := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	require.NoError(t, err)
	files := map[string]*zip.File{}
	for _, f := range reader.File {
		files[f.Name] = f
	}
	md := readZipFile(t, files["projections/video-runbooks.md"])
	require.Contains(t, md, "TestMethodSpecific:49")
	require.Contains(t, md, "approved-r15")
	require.Contains(t, md, "unverified")
	require.Contains(t, md, "采集待完成：true")
	require.NotContains(t, readZipFile(t, files["projections/book.md"]), "TestMethodSpecific")
	require.Contains(t, readZipFile(t, files["manifest.json"]), "projections/video-runbooks.json")
	require.Contains(t, readZipFile(t, files["manifest.json"]), "projections/video-runbooks.md")
	// A changed guide cannot pass with the old frozen digest.
	corrupt := bytes.Replace(manifest, []byte("TestMethodSpecific:49"), []byte("ImaginaryTest:1"), 1)
	require.Error(t, NewBookPackageBuilder().Build(&bytes.Buffer{}, BookPackageInput{Book: book, BuildManifest: corrupt}))
}

func TestLegacyBookRunbookExportStaysExplicitlyUnavailable(t *testing.T) {
	files, err := frozenBookRunbookFiles([]byte(`{"format":"inkwords.book-build.v1"}`), shared.CanonicalBookAST{})
	require.NoError(t, err)
	require.Equal(t, "null", string(files[0].Content))
	require.Contains(t, string(files[1].Content), "该历史构建未冻结视频教案")
	require.NotContains(t, string(files[1].Content), "TestMethodSpecific")
}
