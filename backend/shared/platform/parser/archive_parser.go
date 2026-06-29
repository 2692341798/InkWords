package parser

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
)

var archiveCodeTextExtensions = map[string]bool{
	".c":    true,
	".cpp":  true,
	".go":   true,
	".h":    true,
	".hpp":  true,
	".java": true,
	".js":   true,
	".json": true,
	".py":   true,
	".rs":   true,
	".sh":   true,
	".sql":  true,
	".ts":   true,
	".tsx":  true,
	".jsx":  true,
	".yaml": true,
	".yml":  true,
}

// ArchiveSummary describes how many files in a ZIP archive were kept, skipped, or deduplicated.
type ArchiveSummary struct {
	TotalFiles     int      `json:"total_files"`
	SupportedFiles int      `json:"supported_files"`
	KeptFiles      int      `json:"kept_files"`
	DuplicateFiles int      `json:"duplicate_files"`
	IgnoredFiles   int      `json:"ignored_files"`
	FailedFiles    int      `json:"failed_files"`
	KeptPaths      []string `json:"kept_paths,omitempty"`
}

// ParsedSource wraps the merged source content and optional archive parsing metadata.
type ParsedSource struct {
	SourceContent  string          `json:"source_content"`
	ArchiveSummary *ArchiveSummary `json:"archive_summary,omitempty"`
}

// ArchiveParser parses ZIP archives into a single source content string for downstream analysis.
type ArchiveParser struct {
	docParser *DocParser
}

// NewArchiveParser creates an ArchiveParser that reuses DocParser for document extraction.
func NewArchiveParser(docParser *DocParser) *ArchiveParser {
	return &ArchiveParser{docParser: docParser}
}

// ParseArchive merges supported text-like files inside a ZIP into one source_content payload.
func (p *ArchiveParser) ParseArchive(src io.Reader, filename string) (ParsedSource, error) {
	if strings.ToLower(filepath.Ext(filename)) != ".zip" {
		return ParsedSource{}, fmt.Errorf("unsupported archive extension: %s", filepath.Ext(filename))
	}

	archiveBytes, err := io.ReadAll(src)
	if err != nil {
		return ParsedSource{}, fmt.Errorf("读取压缩包失败: %w", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(archiveBytes), int64(len(archiveBytes)))
	if err != nil {
		return ParsedSource{}, fmt.Errorf("读取压缩包失败: %w", err)
	}

	entries := slices.Clone(reader.File)
	slices.SortFunc(entries, func(left, right *zip.File) int {
		return strings.Compare(left.Name, right.Name)
	})

	summary := &ArchiveSummary{TotalFiles: len(entries)}
	seenContent := make(map[string]struct{}, len(entries))
	parts := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.FileInfo().IsDir() {
			continue
		}

		archivePath, err := sanitizeArchivePath(entry.Name)
		if err != nil {
			return ParsedSource{}, err
		}

		if !isSupportedArchiveTextFile(archivePath) {
			summary.IgnoredFiles++
			continue
		}
		summary.SupportedFiles++

		text, err := p.parseArchiveEntry(entry, archivePath)
		if err != nil {
			summary.FailedFiles++
			continue
		}

		normalized := normalizeArchiveText(text)
		if normalized == "" {
			summary.IgnoredFiles++
			continue
		}

		fingerprint := sha256.Sum256([]byte(normalized))
		digest := hex.EncodeToString(fingerprint[:])
		if _, exists := seenContent[digest]; exists {
			summary.DuplicateFiles++
			continue
		}

		seenContent[digest] = struct{}{}
		summary.KeptFiles++
		summary.KeptPaths = append(summary.KeptPaths, archivePath)
		parts = append(parts, fmt.Sprintf("--- 文件: %s ---\n%s", archivePath, normalized))
	}

	if len(parts) == 0 {
		return ParsedSource{}, fmt.Errorf("压缩包中没有可解析的文本文件")
	}

	return ParsedSource{
		SourceContent:  strings.Join(parts, "\n\n"),
		ArchiveSummary: summary,
	}, nil
}

func (p *ArchiveParser) parseArchiveEntry(entry *zip.File, archivePath string) (string, error) {
	reader, err := entry.Open()
	if err != nil {
		return "", fmt.Errorf("打开压缩包文件失败: %w", err)
	}
	defer func() { _ = reader.Close() }()

	if requiresDocParser(archivePath) {
		return p.docParser.Parse(reader, archivePath)
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("读取压缩包文件失败: %w", err)
	}
	return string(content), nil
}

func isSupportedArchiveTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return isPlainTextExtension(ext) || archiveCodeTextExtensions[ext] || ext == ".pdf" || ext == ".docx"
}

func requiresDocParser(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".pdf" || ext == ".docx" || isPlainTextExtension(ext)
}

func sanitizeArchivePath(name string) (string, error) {
	normalized := filepath.ToSlash(filepath.Clean(strings.TrimSpace(name)))
	if normalized == "." || normalized == "" {
		return "", fmt.Errorf("非法压缩包路径: %s", name)
	}
	if strings.HasPrefix(normalized, "../") || normalized == ".." || filepath.IsAbs(normalized) {
		return "", fmt.Errorf("非法压缩包路径: %s", name)
	}
	return normalized, nil
}

func normalizeArchiveText(text string) string {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	lines := strings.Split(normalized, "\n")
	var builder strings.Builder
	previousBlank := false

	for _, line := range lines {
		trimmedRight := strings.TrimRight(line, " \t")
		if strings.TrimSpace(trimmedRight) == "" {
			if previousBlank {
				continue
			}
			previousBlank = true
		} else {
			previousBlank = false
		}

		if builder.Len() > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(trimmedRight)
	}

	return strings.TrimSpace(builder.String())
}
