package export

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"inkwords-backend/shared/kernel/httpx"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/teachingartifact"
	"inkwords-backend/shared/platform/visualasset"
)

func TestWriteTextbookChapterZipPackagesApprovedManuscriptAndProvenance(t *testing.T) {
	chapterID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	revisionID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	blueprintID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	chapter := TextbookChapterExport{
		ChapterID: chapterID, Title: "请求生命周期", SortOrder: 2,
		RevisionID: revisionID, RevisionNumber: 4, Markdown: "# 请求生命周期\n\n这是已批准的教材稿。",
		ContentHash:         "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		BlueprintRevisionID: &blueprintID, EvidencePackHash: "sha256:evidence", PromptHash: "sha256:prompt",
	}

	var archive bytes.Buffer
	require.NoError(t, writeTextbookChapterZip(&archive, chapter))

	reader, err := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	require.NoError(t, err)
	require.Len(t, reader.File, 2)
	require.Equal(t, "chapter.md", reader.File[0].Name)
	require.Equal(t, "manifest.json", reader.File[1].Name)
	markdown := readZipFile(t, reader.File[0])
	manifest := readZipFile(t, reader.File[1])
	require.Equal(t, chapter.Markdown, markdown)
	require.Contains(t, manifest, `"schema_version": "inkwords.textbook-export.v1"`)
	require.Contains(t, manifest, chapterID.String())
	require.Contains(t, manifest, revisionID.String())
	require.Contains(t, manifest, blueprintID.String())
	require.NotContains(t, manifest, chapter.Markdown)
}

func TestTextbookExportHandlerExportsOnlyApprovedChapter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspaceID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	chapterID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	repo := &textbookExportTestRepository{chapter: TextbookChapterExport{
		ChapterID: chapterID, Title: "Gin 路由", RevisionID: uuid.New(), RevisionNumber: 3,
		Markdown: "# Gin 路由\n\n批准稿", ContentHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}}
	handler := NewHandler(NewService(repo, nil, nil, "", ""), NewTextbookChapterPackageBuilder(teachingartifact.NewStore(t.TempDir()), visualasset.NewStore(t.TempDir())))
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return workspaceID, nil }))
	router.GET("/chapters/:chapterID/export/markdown", handler.ExportApprovedTextbookChapterMarkdown)
	router.GET("/chapters/:chapterID/export/zip", handler.ExportApprovedTextbookChapterZip)

	markdownRequest := httptest.NewRequest(http.MethodGet, "/chapters/"+chapterID.String()+"/export/markdown", nil)
	markdownResponse := httptest.NewRecorder()
	router.ServeHTTP(markdownResponse, markdownRequest)
	require.Equal(t, http.StatusOK, markdownResponse.Code)
	require.Contains(t, markdownResponse.Header().Get("Content-Type"), "text/markdown")
	require.Contains(t, markdownResponse.Header().Get("Content-Disposition"), "Gin 路由.md")
	require.Equal(t, repo.chapter.Markdown, markdownResponse.Body.String())
	require.Equal(t, workspaceID, repo.workspaceID)
	require.Equal(t, chapterID, repo.chapterID)

	zipRequest := httptest.NewRequest(http.MethodGet, "/chapters/"+chapterID.String()+"/export/zip", nil)
	zipResponse := httptest.NewRecorder()
	router.ServeHTTP(zipResponse, zipRequest)
	require.Equal(t, http.StatusOK, zipResponse.Code)
	require.Contains(t, zipResponse.Header().Get("Content-Type"), "application/zip")
	zipReader, err := zip.NewReader(bytes.NewReader(zipResponse.Body.Bytes()), int64(zipResponse.Body.Len()))
	require.NoError(t, err)
	zipFiles := make(map[string]*zip.File, len(zipReader.File))
	for _, file := range zipReader.File {
		zipFiles[file.Name] = file
	}
	require.Equal(t, repo.chapter.Markdown, readZipFile(t, zipFiles["chapter.md"]))
}

func TestTextbookChapterPackageIncludesContentAddressedCodeAndVisualAssets(t *testing.T) {
	teachingStore := teachingartifact.NewStore(t.TempDir())
	code, err := teachingStore.Stage(context.Background(), []teachingartifact.File{{Path: "main.go", Content: []byte("package main\n")}})
	require.NoError(t, err)
	visualStore := visualasset.NewStore(t.TempDir())
	visual, err := visualStore.Stage(context.Background(), "image/png", bytes.NewBufferString("png-bytes"))
	require.NoError(t, err)
	artifactID, assetID := uuid.New(), uuid.New()
	chapter := TextbookChapterExport{
		ChapterID: uuid.New(), Title: "可追溯导出", RevisionID: uuid.New(), RevisionNumber: 1, Markdown: "# 正文", ContentHash: strings.Repeat("a", 64),
		CodeArtifacts: []TextbookCodeArtifactExport{{ID: artifactID, ArtifactHash: code.ArtifactHash, ManifestHash: "sha256:manifest", Status: "unverified"}},
		Assets:        []TextbookManuscriptAssetExport{{ID: assetID, StableRef: "inkwords-asset:test", Kind: "screenshot", ContentHash: visual.ContentHash, AltText: "截图", Source: "本地 IDE", GenerationMethod: "manual_capture", RightsStatus: "pending", Status: "unverified"}},
	}
	var archive bytes.Buffer
	require.NoError(t, NewTextbookChapterPackageBuilder(teachingStore, visualStore).Build(&archive, chapter))

	reader, err := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	require.NoError(t, err)
	files := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		files[file.Name] = file
	}
	require.Equal(t, "# 正文", readZipFile(t, files["chapter.md"]))
	require.Equal(t, "package main\n", readZipFile(t, files["code/"+artifactID.String()+"/main.go"]))
	require.Equal(t, "png-bytes", readZipFile(t, files["assets/"+visual.ContentHash[len("sha256:"):]+".png"]))
	require.Contains(t, readZipFile(t, files["verification-summary.json"]), "[]")
	manifest := readZipFile(t, files["manifest.json"])
	require.Contains(t, manifest, textbookPackageSchemaVersion)
	require.Contains(t, manifest, "publication_blockers")
	require.Contains(t, readZipFile(t, files["manifest.sha256"]), "manifest.json")
}

func TestChapterBookASTCanonicalizesStoredRevisionHash(t *testing.T) {
	chapter := TextbookChapterExport{ChapterID: uuid.New(), Title: "哈希", Markdown: "# 哈希", ContentHash: strings.Repeat("A", 64)}
	document, err := chapterBookAST(chapter, time.Unix(1, 0).UTC())
	require.NoError(t, err)
	require.Equal(t, "sha256:"+strings.Repeat("a", 64), document.Chapters[0].ContentHash)
}

func TestPublicationBlockersRequireCurrentEvidenceAndRights(t *testing.T) {
	artifactID, evidenceID := uuid.New(), uuid.New()
	expiresAt := time.Now().UTC().Add(time.Hour)
	chapter := TextbookChapterExport{
		CodeArtifacts:   []TextbookCodeArtifactExport{{ID: artifactID, ArtifactHash: "sha256:artifact"}},
		RuntimeEvidence: []TextbookRuntimeEvidenceExport{{ID: evidenceID, CodeArtifactID: artifactID, CodeArtifactHash: "sha256:artifact", Status: "verified", ExpiresAt: &expiresAt}},
		Assets:          []TextbookManuscriptAssetExport{{EvidenceID: evidenceID, StableRef: "inkwords-asset:screenshot", RightsStatus: "ready"}},
	}
	blockers := publicationBlockers(chapter)
	require.Len(t, blockers, 1, "only the intentionally unimplemented whole-book and human-review gate remains")

	chapter.RuntimeEvidence[0].ExpiresAt = nil
	chapter.Assets[0].RightsStatus = "pending"
	blockers = publicationBlockers(chapter)
	require.Len(t, blockers, 4)
	require.Contains(t, blockers[1], "代码工件")
	require.Contains(t, blockers[2], "权利状态")
	require.Contains(t, blockers[3], "关联的运行证据")

	chapter.RuntimeEvidence[0].ExpiresAt = &expiresAt
	chapter.RuntimeEvidence[0].StaleReason = "BookContract 已变化，需重新验证。"
	chapter.Assets[0].RightsStatus = "ready"
	blockers = publicationBlockers(chapter)
	require.Len(t, blockers, 3)
	require.Contains(t, blockers[1], "代码工件")
	require.Contains(t, blockers[2], "关联的运行证据")
}

func TestTextbookExportHandlerRejectsUnapprovedAndInvalidChapters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	chapterID := uuid.New()
	repo := &textbookExportTestRepository{err: ErrTextbookChapterNotApproved}
	handler := NewHandler(NewService(repo, nil, nil, "", ""))
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return uuid.New(), nil }))
	router.GET("/chapters/:chapterID/export/markdown", handler.ExportApprovedTextbookChapterMarkdown)

	for _, tc := range []struct {
		path string
		code int
		body string
	}{
		{"/chapters/" + chapterID.String() + "/export/markdown", http.StatusConflict, "TEXTBOOK_CHAPTER_NOT_APPROVED"},
		{"/chapters/not-a-uuid/export/markdown", http.StatusBadRequest, "INVALID_TEXTBOOK_CHAPTER_ID"},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
		require.Equal(t, tc.code, response.Code)
		require.Contains(t, response.Body.String(), tc.body)
	}
}

func TestTextbookExportHandlerUsesFrozenBuildASTForMarkdown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspaceID, buildID := uuid.New(), uuid.New()
	book, err := sharedtextbook.NewCanonicalBookAST("冻结 Gin 教材", time.Unix(1, 0), []sharedtextbook.CanonicalBookChapter{{ID: uuid.NewString(), Order: 1, Title: "冻结章节", Markdown: "# 冻结章节\n\n旧的批准内容", ContentHash: "sha256:" + strings.Repeat("a", 64)}})
	require.NoError(t, err)
	repo := &textbookExportTestRepository{build: TextbookBookBuildExport{BuildID: buildID, ManifestHash: "sha256:manifest", Book: book}}
	handler := NewHandler(NewService(repo, nil, nil, "", ""))
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return workspaceID, nil }))
	router.GET("/book-builds/:buildID/export/markdown", handler.ExportFrozenTextbookBookMarkdown)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/book-builds/"+buildID.String()+"/export/markdown", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "sha256:manifest", response.Header().Get("X-InkWords-Book-Build-Manifest"))
	require.Contains(t, response.Body.String(), "旧的批准内容")
	require.Equal(t, workspaceID, repo.workspaceID)
	require.Equal(t, buildID, repo.buildID)
}

func TestTextbookExportHandlerRendersDOCXAndReviewBundleFromFrozenBuild(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspaceID, buildID := uuid.New(), uuid.New()
	book, err := sharedtextbook.NewCanonicalBookAST("冻结 Gin 教材", time.Unix(1, 0), []sharedtextbook.CanonicalBookChapter{{ID: uuid.NewString(), Order: 1, Title: "冻结章节", Markdown: "# 冻结章节\n\n旧的批准内容", ContentHash: "sha256:" + strings.Repeat("a", 64)}})
	require.NoError(t, err)
	repo := &textbookExportTestRepository{build: TextbookBookBuildExport{BuildID: buildID, ManifestHash: "sha256:manifest", ManifestJSON: json.RawMessage(`{"format":"inkwords.book-build.v1","book":{"title":"must-not-be-read"}}`), Book: book, Preflight: EvaluatePublicationPreflight(PublicationPreflightInput{BuildID: buildID.String()})}}
	handler := NewHandler(NewService(repo, nil, nil, "", ""), NewTextbookChapterPackageBuilder(teachingartifact.NewStore(t.TempDir()), visualasset.NewStore(t.TempDir()))).WithBookProjectionRenderers(fakeBookDOCXRenderer{}, fakeBookPDFRenderer{})
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return workspaceID, nil }))
	router.GET("/book-builds/:buildID/export/docx", handler.ExportFrozenTextbookBookDOCX)
	router.GET("/book-builds/:buildID/export/pdf", handler.ExportFrozenTextbookBookPDF)
	router.GET("/book-builds/:buildID/export/review-bundle", handler.ExportFrozenTextbookBookReviewBundle)

	for _, tc := range []struct {
		path        string
		contentType string
		prefix      string
	}{
		{"/book-builds/" + buildID.String() + "/export/docx", bookDOCXMediaType, "PK\x03\x04"},
		{"/book-builds/" + buildID.String() + "/export/pdf", bookPDFMediaType, "%PDF-"},
		{"/book-builds/" + buildID.String() + "/export/review-bundle", "application/zip", "PK\x03\x04"},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
		require.Equal(t, http.StatusOK, response.Code)
		require.Contains(t, response.Header().Get("Content-Type"), tc.contentType)
		require.Equal(t, "sha256:manifest", response.Header().Get("X-InkWords-Book-Build-Manifest"))
		require.True(t, strings.HasPrefix(response.Body.String(), tc.prefix))
	}

	bundleResponse := httptest.NewRecorder()
	router.ServeHTTP(bundleResponse, httptest.NewRequest(http.MethodGet, "/book-builds/"+buildID.String()+"/export/review-bundle", nil))
	reader, err := zip.NewReader(bytes.NewReader(bundleResponse.Body.Bytes()), int64(bundleResponse.Body.Len()))
	require.NoError(t, err)
	files := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		files[file.Name] = file
	}
	for _, name := range []string{"projections/book.docx", "projections/book.pdf", "verification-summary.json", "rights.json", "human-reviews.json", "quality-report.json", "book-build-manifest.json", "publication-preflight.json"} {
		require.Contains(t, files, name)
	}
	require.Contains(t, readZipFile(t, files["verification-summary.json"]), "unavailable")
	require.Contains(t, readZipFile(t, files["quality-report.json"]), `"chapter_snapshot_status": "unavailable"`)
	require.Contains(t, readZipFile(t, files["quality-report.json"]), "自动门禁通过不代表整书审阅通过")
	require.Contains(t, readZipFile(t, files["publication-preflight.json"]), "缺少人工或用户委托 AI 技术审校记录")
}

func TestTextbookExportHandlerRejectsUnavailableBookRenderer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspaceID, buildID := uuid.New(), uuid.New()
	book := canonicalBookForDOCXTest(t)
	repo := &textbookExportTestRepository{build: TextbookBookBuildExport{BuildID: buildID, ManifestHash: "sha256:manifest", Book: book}}
	handler := NewHandler(NewService(repo, nil, nil, "", ""))
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return workspaceID, nil }))
	router.GET("/book-builds/:buildID/export/docx", handler.ExportFrozenTextbookBookDOCX)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/book-builds/"+buildID.String()+"/export/docx", nil))

	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.Contains(t, response.Body.String(), "TEXTBOOK_BOOK_DOCX_UNAVAILABLE")
}

type fakeBookDOCXRenderer struct{}

func (fakeBookDOCXRenderer) RenderDOCX(_ context.Context, book sharedtextbook.CanonicalBookAST) (BookDOCXProjection, error) {
	if err := book.Validate(); err != nil {
		return BookDOCXProjection{}, err
	}
	return BookDOCXProjection{Content: []byte{'P', 'K', 0x03, 0x04, 'f', 'a', 'k', 'e'}, MediaType: bookDOCXMediaType, ToolVersions: map[string]string{"pandoc": "fake"}}, nil
}

type fakeBookPDFRenderer struct{}

func (fakeBookPDFRenderer) RenderPDF(_ context.Context, book sharedtextbook.CanonicalBookAST) (BookPDFProjection, error) {
	if err := book.Validate(); err != nil {
		return BookPDFProjection{}, err
	}
	return BookPDFProjection{Content: []byte("%PDF-1.7 fake"), MediaType: bookPDFMediaType, RenderLog: "status=success", ToolVersions: map[string]string{"chromium": "fake"}}, nil
}

func readZipFile(t *testing.T, file *zip.File) string {
	t.Helper()
	reader, err := file.Open()
	require.NoError(t, err)
	t.Cleanup(func() { _ = reader.Close() })
	contents, err := io.ReadAll(reader)
	require.NoError(t, err)
	return string(contents)
}

type textbookExportTestRepository struct {
	chapter     TextbookChapterExport
	build       TextbookBookBuildExport
	err         error
	workspaceID uuid.UUID
	chapterID   uuid.UUID
	buildID     uuid.UUID
}

func (r *textbookExportTestRepository) GetByID(context.Context, uuid.UUID, uuid.UUID) (Blog, error) {
	return Blog{}, errors.New("not used")
}

func (r *textbookExportTestRepository) GetSeriesBlogs(context.Context, uuid.UUID, uuid.UUID) ([]Blog, error) {
	return nil, errors.New("not used")
}

func (r *textbookExportTestRepository) GetApprovedTextbookChapter(_ context.Context, workspaceID, chapterID uuid.UUID) (TextbookChapterExport, error) {
	r.workspaceID = workspaceID
	r.chapterID = chapterID
	if r.err != nil {
		return TextbookChapterExport{}, r.err
	}
	return r.chapter, nil
}

func (r *textbookExportTestRepository) GetTextbookBookBuild(_ context.Context, workspaceID, buildID uuid.UUID) (TextbookBookBuildExport, error) {
	r.workspaceID = workspaceID
	r.buildID = buildID
	if r.err != nil {
		return TextbookBookBuildExport{}, r.err
	}
	return r.build, nil
}
