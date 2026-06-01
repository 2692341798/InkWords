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
	"github.com/nguyenthenguyen/docx"
)

var plainTextExtensions = map[string]bool{
	".md":       true,
	".markdown": true,
	".txt":      true,
}

func isPlainTextExtension(ext string) bool {
	return plainTextExtensions[strings.ToLower(ext)]
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
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()

	// Copy data to temp file
	size, err := io.Copy(tempFile, src)
	if err != nil {
		return "", fmt.Errorf("failed to write to temp file: %w", err)
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

// parseDocx extracts text from a .docx file using github.com/nguyenthenguyen/docx
func (p *DocParser) parseDocx(file *os.File) (string, error) {
	// Need to close the file to let the library open it by path
	// The library opens by filename
	// But it requires file path
	doc, err := docx.ReadDocxFile(file.Name())
	if err != nil {
		return "", fmt.Errorf("failed to open docx file: %w", err)
	}
	defer doc.Close()

	text := doc.Editable().GetContent()
	// Usually text contains raw xml or plain text, wait, GetContent() returns a string
	// The docx lib provides GetContent() which returns string with xml stripped or full content?
	// Actually, `docx.ReadDocxFile` provides `Editable().GetContent()`, which might return the XML content.
	// Wait, let's just return the raw text if possible, or use a better text extraction if needed.
	// The simplest is to just use it.
	// Let's strip XML tags just in case
	text = stripXMLTags(text)
	return strings.TrimSpace(text), nil
}

// stripXMLTags is a simple helper to strip XML tags from docx content
func stripXMLTags(content string) string {
	var buf bytes.Buffer
	inTag := false
	for _, r := range content {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			buf.WriteRune(r)
		}
	}
	return buf.String()
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
	buf.ReadFrom(b)

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
