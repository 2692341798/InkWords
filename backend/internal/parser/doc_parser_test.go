package parser

import (
	"bytes"
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
