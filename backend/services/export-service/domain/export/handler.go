package export

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"inkwords-backend/shared/kernel/httpx"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type Handler struct {
	service            *Service
	packageBuilder     *TextbookChapterPackageBuilder
	bookPackageBuilder *BookPackageBuilder
	bookDOCXRenderer   BookDOCXRenderer
	bookPDFRenderer    BookPDFRenderer
	bookImages         BookImageSource
}

// BookDOCXRenderer is the narrow publishing adapter boundary used by the HTTP
// layer. Its input is always the immutable AST captured in a BookBuild.
type BookDOCXRenderer interface {
	RenderDOCX(context.Context, sharedtextbook.CanonicalBookAST) (BookDOCXProjection, error)
}

// BookPDFRenderer is the narrow publishing adapter boundary used by the HTTP
// layer. Its input is always the immutable AST captured in a BookBuild.
type BookPDFRenderer interface {
	RenderPDF(context.Context, sharedtextbook.CanonicalBookAST) (BookPDFProjection, error)
}

func NewHandler(service *Service, packageBuilders ...*TextbookChapterPackageBuilder) *Handler {
	handler := &Handler{service: service, bookPackageBuilder: NewBookPackageBuilder()}
	if len(packageBuilders) > 0 {
		handler.packageBuilder = packageBuilders[0]
	}
	return handler
}

// WithBookProjectionRenderers installs deployment-owned renderers. A missing
// renderer remains unavailable rather than silently producing another format.
func (h *Handler) WithBookProjectionRenderers(docx BookDOCXRenderer, pdf BookPDFRenderer) *Handler {
	if h != nil {
		h.bookDOCXRenderer = docx
		h.bookPDFRenderer = pdf
	}
	return h
}

// WithBookImages configures self-contained Markdown downloads from frozen bytes.
func (h *Handler) WithBookImages(source BookImageSource) *Handler {
	h.bookImages = source
	return h
}

func (h *Handler) ExportSeriesToObsidian(c *gin.Context) {
	workspaceID, ok := currentWorkspaceID(c)
	if !ok {
		return
	}
	blogID, ok := blogIDParam(c)
	if !ok {
		return
	}

	if err := h.service.ExportSeriesToObsidian(c.Request.Context(), blogID, workspaceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "success", "data": nil})
}

func (h *Handler) ExportSeries(c *gin.Context) {
	workspaceID, ok := currentWorkspaceID(c)
	if !ok {
		return
	}
	blogID, ok := blogIDParam(c)
	if !ok {
		return
	}

	blogs, err := h.service.GetSeriesBlogs(c.Request.Context(), blogID, workspaceID)
	if err != nil {
		if errors.Is(err, ErrSeriesNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": http.StatusNotFound, "message": "找不到该系列博客", "data": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": err.Error(), "data": nil})
		return
	}

	c.Writer.Header().Set("Content-Type", "application/zip")
	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", seriesParentTitle(blogs)))

	if err := writeSeriesZip(c.Writer, blogs); err != nil {
		log.Printf("series zip write failed: %v", err)
		return
	}
}

//nolint:gosec
func (h *Handler) ExportSeriesPDF(c *gin.Context) {
	workspaceID, ok := currentWorkspaceID(c)
	if !ok {
		return
	}
	blogID, ok := blogIDParam(c)
	if !ok {
		return
	}

	pdfPath, filename, err := h.service.ExportSeriesToPDF(c.Request.Context(), blogID, workspaceID)
	if err != nil {
		if errors.Is(err, ErrSeriesNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": http.StatusNotFound, "message": "找不到该系列博客", "data": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": err.Error(), "data": nil})
		return
	}

	file, err := os.Open(pdfPath)
	if err != nil {
		_ = os.Remove(pdfPath)
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "读取 PDF 失败", "data": nil})
		return
	}
	defer func() { _ = file.Close() }()

	c.Writer.Header().Set("Content-Type", "application/pdf")
	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Status(http.StatusOK)

	if _, err := io.Copy(c.Writer, file); err != nil {
		log.Printf("pdf copy to response failed for %s: %v", blogID, err)
	}
	if err := os.Remove(pdfPath); err != nil {
		log.Printf("pdf temp file cleanup failed for %s: %v", pdfPath, err)
	}
}

func (h *Handler) ExportToObsidian(c *gin.Context) {
	workspaceID, ok := currentWorkspaceID(c)
	if !ok {
		return
	}
	blogID, ok := blogIDParam(c)
	if !ok {
		return
	}

	if err := h.service.ExportToObsidian(c.Request.Context(), blogID, workspaceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "success", "data": nil})
}

// ExportApprovedTextbookChapterMarkdown returns the exact, approved source
// Markdown. It intentionally does not render publication formats; that belongs
// to the later canonical textbook build rather than this M1 reviewable export.
func (h *Handler) ExportApprovedTextbookChapterMarkdown(c *gin.Context) {
	chapter, ok := h.approvedTextbookChapter(c)
	if !ok {
		return
	}
	c.Header("Content-Type", "text/markdown; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.md\"", sanitizeExportFileName(chapter.Title)))
	c.Status(http.StatusOK)
	_, _ = io.WriteString(c.Writer, chapter.Markdown)
}

// ExportApprovedTextbookChapterZip packages the immutable manuscript and a
// provenance manifest. The fixed filenames make archive paths safe and stable.
func (h *Handler) ExportApprovedTextbookChapterZip(c *gin.Context) {
	chapter, ok := h.approvedTextbookChapter(c)
	if !ok {
		return
	}
	if h.packageBuilder == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "TEXTBOOK_PACKAGE_UNAVAILABLE", "message": "教材工件导出服务尚未配置完成", "data": nil})
		return
	}
	// This legacy chapter projection has no frozen citation ledger. Core's
	// BookBuild is the authority that captures sources for cited manuscripts.
	if len(sharedtextbook.BookEvidenceMarkers(chapter.Markdown)) > 0 {
		c.JSON(http.StatusConflict, gin.H{"code": "TEXTBOOK_CHAPTER_REQUIRES_BOOK_BUILD", "message": "本章包含资料引用。请在工作台冻结待审构建，再下载含冻结来源的整书审校 ZIP。", "data": nil})
		return
	}
	// Complete the archive before committing download headers: a store or AST
	// failure must never return a successful empty or partial ZIP.
	var archive bytes.Buffer
	if err := h.packageBuilder.Build(&archive, chapter); err != nil {
		log.Printf("textbook chapter zip write failed for %s: %v", chapter.ChapterID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": "TEXTBOOK_PACKAGE_FAILED", "message": "教材工件打包失败，未生成可下载文件", "data": nil})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", sanitizeExportFileName(chapter.Title)))
	c.Data(http.StatusOK, "application/zip", archive.Bytes())
}

// ExportFrozenTextbookBookMarkdown writes the Markdown projection from the
// Build's embedded CanonicalBookAST. It intentionally has no fallback to the
// current project, chapters, or revisions.
func (h *Handler) ExportFrozenTextbookBookMarkdown(c *gin.Context) {
	build, ok := h.frozenTextbookBookBuild(c)
	if !ok {
		return
	}
	markdown, err := RenderBookMarkdown(build.Book, h.bookImages)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"code": "TEXTBOOK_BOOK_BUILD_INVALID", "message": "冻结教材构建不完整，不能导出", "data": nil})
		return
	}
	c.Header("Content-Type", "text/markdown; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-build-%s.md\"", sanitizeExportFileName(build.Book.Title), build.BuildID.String()[:8]))
	c.Header("X-InkWords-Book-Build-Manifest", build.ManifestHash)
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write(markdown)
}

// ExportFrozenTextbookBookDOCX renders an editable DOCX directly from the
// canonical AST stored in a frozen Build; it never falls back to current data.
func (h *Handler) ExportFrozenTextbookBookDOCX(c *gin.Context) {
	build, ok := h.frozenTextbookBookBuild(c)
	if !ok {
		return
	}
	docx, ok := h.renderFrozenBookDOCX(c, build)
	if !ok {
		return
	}
	h.writeFrozenBookProjection(c, build, docx.MediaType, "docx", docx.Content)
}

// ExportFrozenTextbookBookPDF renders a PDF from the same frozen AST. The
// deployed Chromium must retain its sandbox; no unsafe fallback is attempted.
func (h *Handler) ExportFrozenTextbookBookPDF(c *gin.Context) {
	build, ok := h.frozenTextbookBookBuild(c)
	if !ok {
		return
	}
	pdf, ok := h.renderFrozenBookPDF(c, build)
	if !ok {
		return
	}
	h.writeFrozenBookProjection(c, build, pdf.MediaType, "pdf", pdf.Content)
}

// ExportFrozenTextbookBookReviewBundle emits an inspectable review bundle.
// Its explicit unavailable records and preflight blockers prevent a ZIP from
// being mistaken for a publication approval.
func (h *Handler) ExportFrozenTextbookBookReviewBundle(c *gin.Context) {
	build, ok := h.frozenTextbookBookBuild(c)
	if !ok {
		return
	}
	docx, ok := h.renderFrozenBookDOCX(c, build)
	if !ok {
		return
	}
	pdf, ok := h.renderFrozenBookPDF(c, build)
	if !ok {
		return
	}
	if h.bookPackageBuilder == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "TEXTBOOK_BOOK_BUNDLE_UNAVAILABLE", "message": "教材整书审校包服务尚未配置完成", "data": nil})
		return
	}
	if h.packageBuilder == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "TEXTBOOK_BOOK_BUNDLE_UNAVAILABLE", "message": "教材整书审校包工件存储尚未配置完成", "data": nil})
		return
	}
	if len(build.ManifestJSON) == 0 {
		c.JSON(http.StatusConflict, gin.H{"code": "TEXTBOOK_BOOK_BUILD_INVALID", "message": "冻结教材构建缺少 manifest，不能生成审校包", "data": nil})
		return
	}
	codeArtifacts, assets, err := h.packageBuilder.FrozenBookPackageFiles(build.ManifestJSON)
	if err != nil {
		log.Printf("textbook review bundle artifacts unavailable for %s: %v", build.BuildID, err)
		c.JSON(http.StatusConflict, gin.H{"code": "TEXTBOOK_BOOK_BUNDLE_ARTIFACTS_UNAVAILABLE", "message": "冻结教材构建的代码或图片工件不可用，不能生成审校包", "data": nil})
		return
	}
	var archive bytes.Buffer
	verificationSummary, err := frozenBookVerificationSummary(build, time.Now().UTC())
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"code": "TEXTBOOK_BOOK_BUILD_INVALID", "message": "冻结运行验证快照无效，不能生成审校包", "data": nil})
		return
	}
	qualityReport, err := frozenBookQualityReport(build, time.Now().UTC())
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"code": "TEXTBOOK_BOOK_BUILD_INVALID", "message": "冻结质量快照无效，不能生成审校包", "data": nil})
		return
	}
	input := BookPackageInput{
		BuildID:             build.BuildID.String(),
		Book:                build.Book,
		DOCX:                &docx,
		PDF:                 &pdf,
		CodeArtifacts:       codeArtifacts,
		Assets:              assets,
		VerificationSummary: verificationSummary,
		RightsItems:         build.RightsItems,
		RightsLedger:        build.RightsLedger,
		ManifestHash:        build.ManifestHash,
		HumanReviews:        build.HumanReviews,
		DelegatedReviews:    build.DelegatedReviews,
		QualityReport:       qualityReport,
		Preflight:           build.Preflight,
		BuildManifest:       build.ManifestJSON,
	}
	if err := h.bookPackageBuilder.Build(&archive, input); err != nil {
		log.Printf("textbook review bundle write failed for %s: %v", build.BuildID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": "TEXTBOOK_BOOK_BUNDLE_FAILED", "message": "生成教材整书审校包失败", "data": nil})
		return
	}
	h.writeFrozenBookProjection(c, build, "application/zip", "review-bundle.zip", archive.Bytes())
}

func (h *Handler) frozenTextbookBookBuild(c *gin.Context) (TextbookBookBuildExport, bool) {
	workspaceID, err := httpx.GetLocalWorkspaceID(c)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "LOCAL_WORKSPACE_UNAVAILABLE", "message": "本地工作区尚未准备完成，请检查数据库迁移后重试", "data": nil})
		return TextbookBookBuildExport{}, false
	}
	buildID, err := uuid.Parse(c.Param("buildID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_TEXTBOOK_BOOK_BUILD_ID", "message": "无效的教材整书构建 ID", "data": nil})
		return TextbookBookBuildExport{}, false
	}
	build, err := h.service.GetFrozenTextbookBookBuild(c.Request.Context(), workspaceID, buildID)
	if err == nil {
		return build, true
	}
	switch {
	case errors.Is(err, ErrTextbookBookBuildNotFound):
		c.JSON(http.StatusNotFound, gin.H{"code": "TEXTBOOK_BOOK_BUILD_NOT_FOUND", "message": "找不到该冻结教材构建", "data": nil})
	case errors.Is(err, ErrTextbookBookBuildInvalid):
		c.JSON(http.StatusConflict, gin.H{"code": "TEXTBOOK_BOOK_BUILD_INVALID", "message": "冻结教材构建不完整，不能导出", "data": nil})
	case errors.Is(err, ErrExportNotConfigured):
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "TEXTBOOK_EXPORT_UNAVAILABLE", "message": "教材导出服务尚未配置完成", "data": nil})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"code": "TEXTBOOK_EXPORT_FAILED", "message": "导出冻结教材构建失败", "data": nil})
	}
	return TextbookBookBuildExport{}, false
}

func (h *Handler) renderFrozenBookDOCX(c *gin.Context, build TextbookBookBuildExport) (BookDOCXProjection, bool) {
	if h.bookDOCXRenderer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "TEXTBOOK_BOOK_DOCX_UNAVAILABLE", "message": "教材 DOCX 渲染器尚未配置参考文档或 Pandoc", "data": nil})
		return BookDOCXProjection{}, false
	}
	projection, err := h.bookDOCXRenderer.RenderDOCX(c.Request.Context(), build.Book)
	if err != nil {
		log.Printf("textbook DOCX render failed for %s: %v", build.BuildID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": "TEXTBOOK_BOOK_DOCX_FAILED", "message": "生成教材 DOCX 失败", "data": nil})
		return BookDOCXProjection{}, false
	}
	return projection, true
}

func (h *Handler) renderFrozenBookPDF(c *gin.Context, build TextbookBookBuildExport) (BookPDFProjection, bool) {
	if h.bookPDFRenderer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "TEXTBOOK_BOOK_PDF_UNAVAILABLE", "message": "教材 PDF 渲染器尚未配置 Chromium", "data": nil})
		return BookPDFProjection{}, false
	}
	projection, err := h.bookPDFRenderer.RenderPDF(c.Request.Context(), build.Book)
	if err != nil {
		log.Printf("textbook PDF render failed for %s: %v", build.BuildID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": "TEXTBOOK_BOOK_PDF_FAILED", "message": "生成教材 PDF 失败", "data": nil})
		return BookPDFProjection{}, false
	}
	return projection, true
}

func (h *Handler) writeFrozenBookProjection(c *gin.Context, build TextbookBookBuildExport, mediaType, extension string, content []byte) {
	c.Header("Content-Type", mediaType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-build-%s.%s\"", sanitizeExportFileName(build.Book.Title), build.BuildID.String()[:8], extension))
	c.Header("X-InkWords-Book-Build-Manifest", build.ManifestHash)
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write(content)
}

func (h *Handler) approvedTextbookChapter(c *gin.Context) (TextbookChapterExport, bool) {
	workspaceID, err := httpx.GetLocalWorkspaceID(c)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "LOCAL_WORKSPACE_UNAVAILABLE", "message": "本地工作区尚未准备完成，请检查数据库迁移后重试", "data": nil})
		return TextbookChapterExport{}, false
	}
	chapterID, err := uuid.Parse(c.Param("chapterID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_TEXTBOOK_CHAPTER_ID", "message": "无效的教材章节 ID", "data": nil})
		return TextbookChapterExport{}, false
	}
	chapter, err := h.service.GetApprovedTextbookChapter(c.Request.Context(), workspaceID, chapterID)
	if err != nil {
		switch {
		case errors.Is(err, ErrTextbookChapterNotFound):
			c.JSON(http.StatusNotFound, gin.H{"code": "TEXTBOOK_CHAPTER_NOT_FOUND", "message": "找不到该教材章节", "data": nil})
		case errors.Is(err, ErrTextbookChapterNotApproved):
			c.JSON(http.StatusConflict, gin.H{"code": "TEXTBOOK_CHAPTER_NOT_APPROVED", "message": "请先人工批准教材稿，再导出", "data": nil})
		case errors.Is(err, ErrExportNotConfigured):
			c.JSON(http.StatusServiceUnavailable, gin.H{"code": "TEXTBOOK_EXPORT_UNAVAILABLE", "message": "教材导出服务尚未配置完成", "data": nil})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": "TEXTBOOK_EXPORT_FAILED", "message": "导出教材章节失败", "data": nil})
		}
		return TextbookChapterExport{}, false
	}
	return chapter, true
}

type textbookChapterManifest struct {
	SchemaVersion string `json:"schema_version"`
	Chapter       struct {
		ID        uuid.UUID `json:"id"`
		Title     string    `json:"title"`
		SortOrder int       `json:"sort_order"`
	} `json:"chapter"`
	Revision struct {
		ID                     uuid.UUID  `json:"id"`
		Number                 int        `json:"number"`
		ContentHash            string     `json:"content_hash"`
		BookContractRevisionID *uuid.UUID `json:"book_contract_revision_id,omitempty"`
		StyleSheetRevisionID   *uuid.UUID `json:"style_sheet_revision_id,omitempty"`
		BlueprintRevisionID    *uuid.UUID `json:"blueprint_revision_id,omitempty"`
		EvidencePackHash       string     `json:"evidence_pack_hash,omitempty"`
		PromptHash             string     `json:"prompt_hash,omitempty"`
		ProviderName           string     `json:"provider_name,omitempty"`
		ModelName              string     `json:"model_name,omitempty"`
	} `json:"revision"`
}

func writeTextbookChapterZip(w io.Writer, chapter TextbookChapterExport) error {
	manifest := textbookChapterManifest{SchemaVersion: "inkwords.textbook-export.v1"}
	manifest.Chapter.ID = chapter.ChapterID
	manifest.Chapter.Title = chapter.Title
	manifest.Chapter.SortOrder = chapter.SortOrder
	manifest.Revision.ID = chapter.RevisionID
	manifest.Revision.Number = chapter.RevisionNumber
	manifest.Revision.ContentHash = chapter.ContentHash
	manifest.Revision.BookContractRevisionID = chapter.BookContractRevisionID
	manifest.Revision.StyleSheetRevisionID = chapter.StyleSheetRevisionID
	manifest.Revision.BlueprintRevisionID = chapter.BlueprintRevisionID
	manifest.Revision.EvidencePackHash = chapter.EvidencePackHash
	manifest.Revision.PromptHash = chapter.PromptHash
	manifest.Revision.ProviderName = chapter.ProviderName
	manifest.Revision.ModelName = chapter.ModelName
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal textbook export manifest: %w", err)
	}

	zipWriter := zip.NewWriter(w)
	markdown, err := zipWriter.Create("chapter.md")
	if err != nil {
		return fmt.Errorf("create textbook chapter entry: %w", err)
	}
	if _, err := io.WriteString(markdown, chapter.Markdown); err != nil {
		return fmt.Errorf("write textbook chapter entry: %w", err)
	}
	manifestFile, err := zipWriter.Create("manifest.json")
	if err != nil {
		return fmt.Errorf("create textbook manifest entry: %w", err)
	}
	if _, err := manifestFile.Write(manifestJSON); err != nil {
		return fmt.Errorf("write textbook manifest entry: %w", err)
	}
	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("close textbook chapter archive: %w", err)
	}
	return nil
}

// writeSeriesZip 将系列博客归档写入 io.Writer，ZIP 创建或写入失败时返回错误。
// 提取为独立可测试函数，避免归档构造逻辑与 HTTP 响应耦合。
func writeSeriesZip(w io.Writer, blogs []Blog) error {
	zipWriter := zip.NewWriter(w)
	defer func() {
		if closeErr := zipWriter.Close(); closeErr != nil {
			log.Printf("series zip close failed: %v", closeErr)
		}
	}()

	for idx, blog := range blogs {
		title := blog.Title
		if title == "" {
			title = fmt.Sprintf("未命名_%d", idx)
		}

		filename := ""
		if blog.ParentID == nil || *blog.ParentID == uuid.Nil {
			filename = fmt.Sprintf("%s.md", title)
		} else {
			filename = fmt.Sprintf("%02d-%s.md", blog.ChapterSort, title)
		}

		file, err := zipWriter.Create(filename)
		if err != nil {
			return fmt.Errorf("zip create entry %q: %w", filename, err)
		}
		if _, err := fmt.Fprintf(file, "# %s\n\n%s", title, blog.Content); err != nil {
			return fmt.Errorf("zip write entry %q: %w", filename, err)
		}
	}
	return nil
}

// seriesParentTitle 从系列博客列表中提取父级标题，用于生成下载文件名。
func seriesParentTitle(blogs []Blog) string {
	if len(blogs) == 0 {
		return "series"
	}
	if blogs[0].Title != "" {
		return blogs[0].Title
	}
	return "series"
}

func currentWorkspaceID(c *gin.Context) (uuid.UUID, bool) {
	workspaceID, err := httpx.GetLocalWorkspaceID(c)
	if err != nil || workspaceID == uuid.Nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": http.StatusServiceUnavailable, "message": "本地工作区尚未准备完成", "data": nil})
		return uuid.Nil, false
	}
	return workspaceID, true
}

func blogIDParam(c *gin.Context) (uuid.UUID, bool) {
	blogID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "无效的博客 ID", "data": nil})
		return uuid.Nil, false
	}
	return blogID, true
}
