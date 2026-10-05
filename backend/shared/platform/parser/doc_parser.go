package parser

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/ledongthuc/pdf"
	"inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/sourceartifact"
)

var plainTextExtensions = map[string]bool{
	".md":       true,
	".markdown": true,
	".txt":      true,
}

const maxStructuredPDFPages = 1_000

const maxStructuredSourceBytes = sourceartifact.MaxTextbookSourceBytes

func isPlainTextExtension(ext string) bool {
	return plainTextExtensions[strings.ToLower(ext)]
}

func isStructuredTextExtension(ext string) bool {
	ext = strings.ToLower(ext)
	return isPlainTextExtension(ext) || archiveCodeTextExtensions[ext]
}

// Parser defines the interface for all document parsers
type Parser interface {
	Parse(src io.Reader, filename string) (string, error)
}

// DocParser implements Parser interface for PDF and Markdown files
type DocParser struct{}

// NewDocParser creates a new instance of DocParser
func NewDocParser() *DocParser {
	return &DocParser{}
}

// Parse extracts text from the given io.Reader and filename.
// It uses a temporary file for processing and guarantees "Burn After Reading" (阅后即焚)
// by deleting the temporary file immediately after parsing.
func (p *DocParser) Parse(src io.Reader, filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	// Write source to a temporary file
	tempFile, err := os.CreateTemp("", "inkwords-parse-*"+ext)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	// 核心策略：阅后即焚
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()

	// Copy data to temp file
	size, err := io.Copy(tempFile, io.LimitReader(src, maxStructuredSourceBytes+1))
	if err != nil {
		return "", fmt.Errorf("failed to write to temp file: %w", err)
	}
	if size > maxStructuredSourceBytes {
		return "", fmt.Errorf("source file exceeds the maximum size limit")
	}

	// Ensure the temp file contents are fully flushed to disk
	if err := tempFile.Sync(); err != nil {
		return "", fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Seek to beginning before passing it to specific parsers
	if _, err := tempFile.Seek(0, 0); err != nil {
		return "", fmt.Errorf("failed to seek temp file: %w", err)
	}

	// Route to specific parser based on extension
	switch {
	case ext == ".pdf":
		return p.parsePDF(tempFile, size)
	case isPlainTextExtension(ext):
		return p.parsePlainText(tempFile)
	case ext == ".docx":
		return p.parseDocx(tempFile)
	default:
		return "", fmt.Errorf("unsupported file extension: %s", ext)
	}
}

// ParseStructured returns citeable chunks for formats whose structure can be preserved.
// PDF is deliberately fail-closed here: the legacy text extractor has no dependable page
// boundaries, and pretending otherwise would create evidence that cannot be audited.
func (p *DocParser) ParseStructured(src io.Reader, request StructuredParseRequest) (StructuredDocument, error) {
	ext := strings.ToLower(filepath.Ext(request.Filename))
	if isStructuredTextExtension(ext) {
		return NewStructuredParser().Parse(src, request)
	}
	if ext == ".pdf" {
		return p.parseStructuredPDF(src, request)
	}
	if ext != ".docx" {
		return StructuredDocument{}, fmt.Errorf("无法保留 %s 的可靠结构：请改用 Markdown、TXT、DOCX 或带文本层的 PDF", ext)
	}

	content, err := p.Parse(src, request.Filename)
	if err != nil {
		return StructuredDocument{}, err
	}
	markdownRequest := request
	markdownRequest.Filename = request.Filename + ".md"
	result, err := NewStructuredParser().Parse(strings.NewReader(content), markdownRequest)
	if err != nil {
		return StructuredDocument{}, err
	}
	result.Document.MediaType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	return result, nil
}

// ParseOfficialWebHTML parses an already policy-approved page into URL-scoped
// textbook evidence. Fetching and URL authorization stay outside DocParser.
func (p *DocParser) ParseOfficialWebHTML(src io.Reader, request StructuredParseRequest) (StructuredDocument, error) {
	return NewStructuredParser().ParseOfficialWebHTML(src, request)
}

// parseStructuredPDF makes every page an independent citeable chunk. PDF text
// has no dependable source-line concept, so page numbers are the only locator
// emitted; a scanned or garbled page is never silently promoted to evidence.
func (p *DocParser) parseStructuredPDF(src io.Reader, request StructuredParseRequest) (StructuredDocument, error) {
	tempFile, err := os.CreateTemp("", "inkwords-structured-pdf-*.pdf")
	if err != nil {
		return StructuredDocument{}, fmt.Errorf("create temporary PDF: %w", err)
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()
	size, err := io.Copy(tempFile, io.LimitReader(src, maxStructuredSourceBytes+1))
	if err != nil {
		return StructuredDocument{}, fmt.Errorf("read PDF: %w", err)
	}
	if size > maxStructuredSourceBytes {
		return StructuredDocument{}, fmt.Errorf("PDF 文件超过最大大小限制")
	}
	if _, err := tempFile.Seek(0, io.SeekStart); err != nil {
		return StructuredDocument{}, fmt.Errorf("seek PDF: %w", err)
	}
	reader, err := pdf.NewReader(tempFile, size)
	if err != nil {
		return StructuredDocument{}, fmt.Errorf("解析 PDF 失败: %w", err)
	}
	pageCount := reader.NumPage()
	if pageCount < 1 || pageCount > maxStructuredPDFPages {
		return StructuredDocument{}, fmt.Errorf("PDF 页数不在可安全解析范围内")
	}
	fonts := make(map[string]*pdf.Font)
	pages := make([]string, 0, pageCount)
	pageNumbers := make([]int, 0, pageCount)
	var extractedBytes int64
	for pageNumber := 1; pageNumber <= pageCount; pageNumber++ {
		page := reader.Page(pageNumber)
		for _, name := range page.Fonts() {
			if _, exists := fonts[name]; !exists {
				font := page.Font(name)
				fonts[name] = &font
			}
		}
		text, err := page.GetPlainText(fonts)
		if err != nil {
			return StructuredDocument{}, fmt.Errorf("解析 PDF 第 %d 页失败: %w", pageNumber, err)
		}
		text = strings.TrimSpace(normalizeStructuredText(text))
		if text == "" {
			continue
		}
		if isLowQualityPDFExtraction(text) {
			return StructuredDocument{}, fmt.Errorf("无法可靠解析 PDF 第 %d 页：检测到严重乱码或扫描文本；请 OCR 后重试", pageNumber)
		}
		extractedBytes += int64(len(text))
		if extractedBytes > maxStructuredTextBytes {
			return StructuredDocument{}, fmt.Errorf("PDF 可引用文本超过可安全解析大小限制；请拆分后重试")
		}
		pages = append(pages, text)
		pageNumbers = append(pageNumbers, pageNumber)
	}
	if len(pages) == 0 {
		return StructuredDocument{}, fmt.Errorf("无法可靠解析 PDF：没有可引用文本；请 OCR 后重试")
	}
	content := strings.Join(pages, "\n\f\n")
	document := newStructuredDocument(request, filepath.Base(request.Filename), "application/pdf", content)
	chunks := make([]textbook.SourceChunk, 0, len(pages))
	offset := 0
	path := request.ArtifactPath
	if path == "" {
		path = request.CanonicalLocator
	}
	for index, text := range pages {
		end := offset + len(text)
		chunks = append(chunks, newStructuredChunk(document.ID, index+1, index+1, offset, end, 0, 0, path, request.CanonicalLocator, "", nil, text))
		chunks[len(chunks)-1].Locator.Page = pageNumbers[index]
		offset = end + len("\n\f\n")
	}
	return StructuredDocument{SourceID: request.SourceID, Document: document, Chunks: chunks}, nil
}

// parsePDF extracts text from a PDF file using github.com/ledongthuc/pdf
func (p *DocParser) parsePDF(file *os.File, size int64) (string, error) {
	// The file pointer is already at 0,0 from the caller

	reader, err := pdf.NewReader(file, size)
	if err != nil {
		if strings.Contains(err.Error(), "missing %%EOF") {
			return "", fmt.Errorf("解析失败：该文件似乎已损坏或不是标准的PDF格式")
		}
		return "", fmt.Errorf("解析 PDF 失败: %w", err)
	}

	var buf bytes.Buffer
	b, err := reader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("提取 PDF 文本失败: %w", err)
	}
	_, _ = buf.ReadFrom(b)

	return resolveReadablePDFText(strings.TrimSpace(buf.String()), file.Name())
}

func isLowQualityPDFExtraction(text string) bool {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) < 500 {
		return false
	}

	meaningfulRunes := 0
	replacementRunes := 0
	controlRunes := 0
	for _, r := range runes {
		if unicode.IsSpace(r) {
			continue
		}
		meaningfulRunes++
		if r == '\uFFFD' {
			replacementRunes++
			continue
		}
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			controlRunes++
		}
	}

	if meaningfulRunes == 0 {
		return true
	}

	replacementRatio := float64(replacementRunes) / float64(meaningfulRunes)
	controlRatio := float64(controlRunes) / float64(meaningfulRunes)

	return replacementRunes >= 50 || replacementRatio >= 0.005 || controlRatio >= 0.01
}

func resolveReadablePDFText(primaryText, filePath string) (string, error) {
	if !isLowQualityPDFExtraction(primaryText) {
		return primaryText, nil
	}

	fallbackText, fallbackErr := extractPDFTextWithPdftotext(filePath)
	if fallbackErr == nil && !isLowQualityPDFExtraction(fallbackText) {
		return fallbackText, nil
	}

	return "", fmt.Errorf("无法可靠解析该 PDF 文本：检测到严重乱码，可能是扫描版、嵌入字体或当前解析库不兼容。请尝试导出为可复制文本的 PDF，或改用 DOCX/Markdown")
}

//nolint:gosec,noctx
func extractPDFTextWithPdftotext(filePath string) (string, error) {
	cmd := exec.Command("pdftotext", "-enc", "UTF-8", "-layout", filePath, "-")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// parsePlainText extracts text from plain text files like Markdown or TXT
func (p *DocParser) parsePlainText(file *os.File) (string, error) {
	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to read plain text metadata: %w", err)
	}
	if info.Size() > maxStructuredTextBytes {
		return "", fmt.Errorf("plain text file exceeds the safe parsing size limit")
	}
	// Need to seek to the beginning before reading
	if _, err := file.Seek(0, 0); err != nil {
		return "", fmt.Errorf("failed to seek file: %w", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		return "", fmt.Errorf("failed to read plain text file: %w", err)
	}

	return strings.TrimSpace(buf.String()), nil
}
