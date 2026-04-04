package parser

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

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
	defer tempFile.Close()

	// 核心策略：阅后即焚
	defer os.Remove(tempFile.Name())

	// Copy data to temp file
	size, err := io.Copy(tempFile, src)
	if err != nil {
		return "", fmt.Errorf("failed to write to temp file: %w", err)
	}

	// Route to specific parser based on extension
	switch ext {
	case ".pdf":
		return p.parsePDF(tempFile, size)
	case ".md", ".markdown", ".txt":
		return p.parsePlainText(tempFile)
	default:
		return "", fmt.Errorf("unsupported file extension: %s", ext)
	}
}

// parsePDF extracts text from a PDF file using github.com/ledongthuc/pdf
func (p *DocParser) parsePDF(file *os.File, size int64) (string, error) {
	// Need to seek to the beginning before reading
	if _, err := file.Seek(0, 0); err != nil {
		return "", fmt.Errorf("failed to seek file: %w", err)
	}

	reader, err := pdf.NewReader(file, size)
	if err != nil {
		return "", fmt.Errorf("failed to create PDF reader: %w", err)
	}

	var buf bytes.Buffer
	b, err := reader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("failed to get plain text from PDF: %w", err)
	}

	buf.ReadFrom(b)
	return strings.TrimSpace(buf.String()), nil
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
