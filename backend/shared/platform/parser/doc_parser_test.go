package parser

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jung-kurt/gofpdf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocParser_Parse_PlainText(t *testing.T) {
	parser := NewDocParser()

	// Markdown text
	mdContent := "# Hello World\nThis is a markdown file.\n"
	reader := strings.NewReader(mdContent)

	text, err := parser.Parse(reader, "test.md")
	require.NoError(t, err)
	assert.Equal(t, "Hello World\nThis is a markdown file.", strings.ReplaceAll(text, "# ", "")) // Simple assertion because Parse might keep `#`
	assert.Contains(t, text, "# Hello World")
	assert.Contains(t, text, "This is a markdown file.")
}

func TestDocParser_Parse_PDF(t *testing.T) {
	// Create a simple PDF for testing
	pdfDoc := gofpdf.New("P", "mm", "A4", "")
	pdfDoc.AddPage()
	pdfDoc.SetFont("Arial", "B", 16)
	pdfDoc.Cell(40, 10, "Hello PDF World")

	var buf bytes.Buffer
	err := pdfDoc.Output(&buf)
	require.NoError(t, err)

	parser := NewDocParser()
	text, err := parser.Parse(&buf, "test.pdf")
	require.NoError(t, err)

	// Note: pdf text extraction might strip some spaces or add some, so we just check Contains
	assert.Contains(t, text, "Hello PDF World")
}

func TestDocParser_Parse_UnsupportedExtension(t *testing.T) {
	parser := NewDocParser()
	reader := strings.NewReader("some data")

	_, err := parser.Parse(reader, "test.exe")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported file extension: .exe")
}

func TestIsLowQualityPDFExtraction_DetectsSevereGarbledText(t *testing.T) {
	garbled := strings.Repeat("¼úð��关系与泛泛之�有什么区�；", 120) +
		strings.Repeat("Y+XÄ!FÈ!ÿ010>,´B'M63jË¶4ÿ¨,", 120)

	assert.True(t, isLowQualityPDFExtraction(garbled))
}

func TestIsLowQualityPDFExtraction_AllowsReadableText(t *testing.T) {
	readable := strings.Repeat("亲密关系是人际经验的核心，科学认识亲密关系能帮助我们理解吸引、承诺、沟通与婚姻。", 120)

	assert.False(t, isLowQualityPDFExtraction(readable))
}

//nolint:gosec
func TestResolveReadablePDFText_UsesPdftotextFallbackForGarbledPrimary(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "pdftotext")
	script := "#!/bin/sh\nprintf '亲密关系 可读回退文本\\n第二行'\n"
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0o755))
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	got, err := resolveReadablePDFText(strings.Repeat("��乱码", 200), "/tmp/fake.pdf")
	require.NoError(t, err)
	assert.Contains(t, got, "亲密关系 可读回退文本")
}

func TestResolveReadablePDFText_ReturnsGuidanceWhenFallbackUnavailable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := resolveReadablePDFText(strings.Repeat("��乱码", 200), "/tmp/fake.pdf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "无法可靠解析该 PDF 文本")
}
