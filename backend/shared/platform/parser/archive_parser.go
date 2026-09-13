package parser

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"inkwords-backend/shared/platform/sourceartifact"
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

// ArchiveLimits bounds untrusted ZIP input before any entry is handed to a document parser.
type ArchiveLimits struct {
	MaxCompressedBytes       int64
	MaxEntries               int
	MaxEntryUncompressedSize uint64
	MaxTotalUncompressedSize uint64
	MaxCompressionRatio      float64
}

// DefaultArchiveLimits accepts the same raw ZIP size as a textbook source, while
// keeping decompression and per-entry budgets deliberately conservative.
func DefaultArchiveLimits() ArchiveLimits {
	return ArchiveLimits{
		MaxCompressedBytes:       sourceartifact.MaxTextbookSourceBytes,
		MaxEntries:               1_000,
		MaxEntryUncompressedSize: 16 << 20,
		MaxTotalUncompressedSize: 128 << 20,
		MaxCompressionRatio:      100,
	}
}

// ArchiveEntryReport records an explicit decision for every non-directory entry.
type ArchiveEntryReport struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// ArchiveSummary describes how many files in a ZIP archive were kept, skipped, or deduplicated.
type ArchiveSummary struct {
	TotalFiles     int                  `json:"total_files"`
	SupportedFiles int                  `json:"supported_files"`
	KeptFiles      int                  `json:"kept_files"`
	DuplicateFiles int                  `json:"duplicate_files"`
	IgnoredFiles   int                  `json:"ignored_files"`
	FailedFiles    int                  `json:"failed_files"`
	KeptPaths      []string             `json:"kept_paths,omitempty"`
	Entries        []ArchiveEntryReport `json:"entries,omitempty"`
}

// ParsedSource wraps the merged source content and optional archive parsing metadata.
type ParsedSource struct {
	SourceContent  string          `json:"source_content"`
	ArchiveSummary *ArchiveSummary `json:"archive_summary,omitempty"`
}

// StructuredArchiveResult retains each safe archive entry as independently
// citeable evidence. It must never be reduced to a merged text blob before a
// textbook source snapshot is persisted.
type StructuredArchiveResult struct {
	Documents      []StructuredDocument `json:"documents"`
	ArchiveSummary *ArchiveSummary      `json:"archive_summary"`
}

// ArchiveParser parses ZIP archives into a single source content string for downstream analysis.
type ArchiveParser struct {
	docParser *DocParser
	limits    ArchiveLimits
}

// NewArchiveParser creates an ArchiveParser that reuses DocParser for document extraction.
func NewArchiveParser(docParser *DocParser) *ArchiveParser {
	return NewArchiveParserWithLimits(docParser, DefaultArchiveLimits())
}

// NewArchiveParserWithLimits allows tests and future policy configuration to tighten ZIP budgets.
func NewArchiveParserWithLimits(docParser *DocParser, limits ArchiveLimits) *ArchiveParser {
	return &ArchiveParser{docParser: docParser, limits: limits}
}

// ParseArchive merges supported text-like files inside a ZIP into one source_content payload.
func (p *ArchiveParser) ParseArchive(src io.Reader, filename string) (ParsedSource, error) {
	if strings.ToLower(filepath.Ext(filename)) != ".zip" {
		return ParsedSource{}, fmt.Errorf("unsupported archive extension: %s", filepath.Ext(filename))
	}

	archiveFile, reader, err := p.openArchive(src)
	if err != nil {
		return ParsedSource{}, err
	}
	defer removeTemporaryArchive(archiveFile)

	if len(reader.File) > p.limits.MaxEntries {
		return ParsedSource{}, fmt.Errorf("压缩包文件数超过最大限制")
	}

	entries := slices.Clone(reader.File)
	slices.SortFunc(entries, func(left, right *zip.File) int {
		return strings.Compare(left.Name, right.Name)
	})

	summary := &ArchiveSummary{TotalFiles: len(entries)}
	seenContent := make(map[string]struct{}, len(entries))
	parts := make([]string, 0, len(entries))
	var totalUncompressed uint64

	for _, entry := range entries {
		if entry.FileInfo().IsDir() {
			continue
		}

		archivePath, err := sanitizeArchivePath(entry.Name)
		if err != nil {
			summary.FailedFiles++
			summary.Entries = append(summary.Entries, ArchiveEntryReport{Path: entry.Name, Status: "failed", Reason: "unsafe_path"})
			return ParsedSource{}, err
		}
		if entry.Flags&0x1 != 0 {
			summary.FailedFiles++
			summary.Entries = append(summary.Entries, ArchiveEntryReport{Path: archivePath, Status: "failed", Reason: "encrypted_entry"})
			return ParsedSource{}, fmt.Errorf("压缩包包含加密文件: %s", archivePath)
		}
		if entry.FileInfo().Mode()&os.ModeSymlink != 0 {
			summary.FailedFiles++
			summary.Entries = append(summary.Entries, ArchiveEntryReport{Path: archivePath, Status: "failed", Reason: "symbolic_link"})
			return ParsedSource{}, fmt.Errorf("压缩包包含符号链接: %s", archivePath)
		}
		if err := p.checkEntryLimits(entry, totalUncompressed); err != nil {
			summary.FailedFiles++
			summary.Entries = append(summary.Entries, ArchiveEntryReport{Path: archivePath, Status: "failed", Reason: err.Error()})
			return ParsedSource{}, fmt.Errorf("压缩包条目不安全 %s: %w", archivePath, err)
		}
		totalUncompressed += entry.UncompressedSize64
		if strings.EqualFold(filepath.Ext(archivePath), ".zip") {
			summary.IgnoredFiles++
			summary.Entries = append(summary.Entries, ArchiveEntryReport{Path: archivePath, Status: "skipped", Reason: "nested_archives_not_supported"})
			continue
		}

		if !isSupportedArchiveTextFile(archivePath) {
			summary.IgnoredFiles++
			summary.Entries = append(summary.Entries, ArchiveEntryReport{Path: archivePath, Status: "skipped", Reason: "unsupported_file_type"})
			continue
		}
		summary.SupportedFiles++

		text, err := p.parseArchiveEntry(entry, archivePath)
		if err != nil {
			summary.FailedFiles++
			summary.Entries = append(summary.Entries, ArchiveEntryReport{Path: archivePath, Status: "failed", Reason: "parse_failed"})
			continue
		}

		normalized := normalizeArchiveText(text)
		if normalized == "" {
			summary.IgnoredFiles++
			summary.Entries = append(summary.Entries, ArchiveEntryReport{Path: archivePath, Status: "skipped", Reason: "empty_content"})
			continue
		}

		fingerprint := sha256.Sum256([]byte(normalized))
		digest := hex.EncodeToString(fingerprint[:])
		if _, exists := seenContent[digest]; exists {
			summary.DuplicateFiles++
			summary.Entries = append(summary.Entries, ArchiveEntryReport{Path: archivePath, Status: "skipped", Reason: "duplicate_content"})
			continue
		}

		seenContent[digest] = struct{}{}
		summary.KeptFiles++
		summary.KeptPaths = append(summary.KeptPaths, archivePath)
		summary.Entries = append(summary.Entries, ArchiveEntryReport{Path: archivePath, Status: "kept"})
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

// ParseStructuredArchive preserves the source path and local offsets of every
// supported file in a ZIP. Unsafe entries fail closed; ordinary unsupported
// entries are retained in the summary with an explicit skipped reason.
func (p *ArchiveParser) ParseStructuredArchive(src io.Reader, request StructuredParseRequest) (StructuredArchiveResult, error) {
	if err := request.Validate(); err != nil {
		return StructuredArchiveResult{}, err
	}
	if !strings.EqualFold(filepath.Ext(request.Filename), ".zip") {
		return StructuredArchiveResult{}, fmt.Errorf("unsupported archive extension: %s", filepath.Ext(request.Filename))
	}
	archiveFile, reader, err := p.openArchive(src)
	if err != nil {
		return StructuredArchiveResult{}, err
	}
	defer removeTemporaryArchive(archiveFile)
	if len(reader.File) > p.limits.MaxEntries {
		return StructuredArchiveResult{}, fmt.Errorf("压缩包文件数超过最大限制")
	}

	entries := slices.Clone(reader.File)
	slices.SortFunc(entries, func(left, right *zip.File) int { return strings.Compare(left.Name, right.Name) })
	summary := &ArchiveSummary{TotalFiles: len(entries)}
	result := StructuredArchiveResult{Documents: make([]StructuredDocument, 0, len(entries)), ArchiveSummary: summary}
	var totalUncompressed uint64
	for _, entry := range entries {
		if entry.FileInfo().IsDir() {
			continue
		}
		archivePath, err := sanitizeArchivePath(entry.Name)
		if err != nil {
			summary.FailedFiles++
			summary.Entries = append(summary.Entries, ArchiveEntryReport{Path: entry.Name, Status: "failed", Reason: "unsafe_path"})
			return StructuredArchiveResult{}, err
		}
		if entry.Flags&0x1 != 0 || entry.FileInfo().Mode()&os.ModeSymlink != 0 {
			summary.FailedFiles++
			reason := "encrypted_entry"
			if entry.FileInfo().Mode()&os.ModeSymlink != 0 {
				reason = "symbolic_link"
			}
			summary.Entries = append(summary.Entries, ArchiveEntryReport{Path: archivePath, Status: "failed", Reason: reason})
			return StructuredArchiveResult{}, fmt.Errorf("压缩包包含不安全文件: %s", archivePath)
		}
		if err := p.checkEntryLimits(entry, totalUncompressed); err != nil {
			summary.FailedFiles++
			summary.Entries = append(summary.Entries, ArchiveEntryReport{Path: archivePath, Status: "failed", Reason: err.Error()})
			return StructuredArchiveResult{}, fmt.Errorf("压缩包条目不安全 %s: %w", archivePath, err)
		}
		totalUncompressed += entry.UncompressedSize64
		if strings.EqualFold(filepath.Ext(archivePath), ".zip") {
			summary.IgnoredFiles++
			summary.Entries = append(summary.Entries, ArchiveEntryReport{Path: archivePath, Status: "skipped", Reason: "nested_archives_not_supported"})
			continue
		}
		if !isSupportedArchiveTextFile(archivePath) {
			summary.IgnoredFiles++
			summary.Entries = append(summary.Entries, ArchiveEntryReport{Path: archivePath, Status: "skipped", Reason: "unsupported_file_type"})
			continue
		}
		summary.SupportedFiles++
		parsed, err := p.parseStructuredArchiveEntry(entry, archivePath, request)
		if err != nil {
			summary.FailedFiles++
			summary.Entries = append(summary.Entries, ArchiveEntryReport{Path: archivePath, Status: "failed", Reason: "parse_failed"})
			continue
		}
		summary.KeptFiles++
		summary.KeptPaths = append(summary.KeptPaths, archivePath)
		summary.Entries = append(summary.Entries, ArchiveEntryReport{Path: archivePath, Status: "kept"})
		result.Documents = append(result.Documents, parsed)
	}
	if len(result.Documents) == 0 {
		return StructuredArchiveResult{}, fmt.Errorf("压缩包中没有可可靠引用的文本文件；请移除扫描 PDF 或改用 Markdown、TXT、DOCX")
	}
	return result, nil
}

// openArchive spools the compressed container to disk. A user may retain a
// source close to the product's 888 MiB limit; loading that container into RAM
// before ZIP validation would make the local parser needlessly fragile.
func (p *ArchiveParser) openArchive(src io.Reader) (*os.File, *zip.Reader, error) {
	archiveFile, err := os.CreateTemp("", "inkwords-archive-*.zip")
	if err != nil {
		return nil, nil, fmt.Errorf("创建压缩包临时文件失败: %w", err)
	}
	fail := func(err error) (*os.File, *zip.Reader, error) {
		removeTemporaryArchive(archiveFile)
		return nil, nil, err
	}

	size, err := io.Copy(archiveFile, io.LimitReader(src, p.limits.MaxCompressedBytes+1))
	if err != nil {
		return fail(fmt.Errorf("读取压缩包失败: %w", err))
	}
	if size > p.limits.MaxCompressedBytes {
		return fail(fmt.Errorf("压缩包超过最大压缩大小限制"))
	}
	if err := archiveFile.Sync(); err != nil {
		return fail(fmt.Errorf("同步压缩包临时文件失败: %w", err))
	}
	reader, err := zip.NewReader(archiveFile, size)
	if err != nil {
		return fail(fmt.Errorf("读取压缩包失败: %w", err))
	}
	return archiveFile, reader, nil
}

func removeTemporaryArchive(file *os.File) {
	if file == nil {
		return
	}
	_ = file.Close()
	_ = os.Remove(file.Name())
}

func (p *ArchiveParser) parseStructuredArchiveEntry(entry *zip.File, archivePath string, parent StructuredParseRequest) (StructuredDocument, error) {
	reader, err := entry.Open()
	if err != nil {
		return StructuredDocument{}, err
	}
	defer func() { _ = reader.Close() }()
	content, err := io.ReadAll(io.LimitReader(reader, int64(p.limits.MaxEntryUncompressedSize)+1))
	if err != nil {
		return StructuredDocument{}, err
	}
	if uint64(len(content)) > p.limits.MaxEntryUncompressedSize {
		return StructuredDocument{}, fmt.Errorf("文件超过解压大小限制")
	}
	request := parent
	request.ArtifactPath = archivePath
	request.CanonicalLocator = parent.CanonicalLocator + "#" + archivePath
	request.Filename = archivePath
	if archiveCodeTextExtensions[strings.ToLower(filepath.Ext(archivePath))] {
		request.Filename = archivePath + ".txt"
		parsed, err := NewStructuredParser().Parse(bytes.NewReader(content), request)
		if err != nil {
			return StructuredDocument{}, err
		}
		parsed.Document.MediaType = codeMediaType(archivePath)
		for index := range parsed.Chunks {
			parsed.Chunks[index].CodeLanguage = codeLanguage(archivePath)
		}
		return parsed, nil
	}
	return p.docParser.ParseStructured(bytes.NewReader(content), request)
}

func codeMediaType(artifactPath string) string {
	if language := codeLanguage(artifactPath); language != "" {
		return "text/x-" + language
	}
	return "text/plain"
}

func codeLanguage(artifactPath string) string {
	switch strings.ToLower(filepath.Ext(artifactPath)) {
	case ".c", ".h":
		return "c"
	case ".cpp", ".hpp":
		return "cpp"
	case ".go":
		return "go"
	case ".java":
		return "java"
	case ".js", ".jsx":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".sh":
		return "shell"
	case ".sql":
		return "sql"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	default:
		return ""
	}
}

func (p *ArchiveParser) parseArchiveEntry(entry *zip.File, archivePath string) (string, error) {
	reader, err := entry.Open()
	if err != nil {
		return "", fmt.Errorf("打开压缩包文件失败: %w", err)
	}
	defer func() { _ = reader.Close() }()

	content, err := io.ReadAll(io.LimitReader(reader, int64(p.limits.MaxEntryUncompressedSize)+1))
	if err != nil {
		return "", fmt.Errorf("读取压缩包文件失败: %w", err)
	}
	if uint64(len(content)) > p.limits.MaxEntryUncompressedSize {
		return "", fmt.Errorf("文件超过解压大小限制")
	}
	if requiresDocParser(archivePath) {
		return p.docParser.Parse(bytes.NewReader(content), archivePath)
	}
	return string(content), nil
}

func (p *ArchiveParser) checkEntryLimits(entry *zip.File, totalUncompressed uint64) error {
	if entry.UncompressedSize64 > p.limits.MaxEntryUncompressedSize {
		return fmt.Errorf("entry_too_large")
	}
	if entry.UncompressedSize64 > p.limits.MaxTotalUncompressedSize-totalUncompressed {
		return fmt.Errorf("total_uncompressed_size_exceeded")
	}
	if entry.UncompressedSize64 > 0 {
		if entry.CompressedSize64 == 0 || float64(entry.UncompressedSize64)/float64(entry.CompressedSize64) > p.limits.MaxCompressionRatio {
			return fmt.Errorf("compression_ratio_exceeded")
		}
	}
	return nil
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
	normalized := path.Clean(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"))
	if normalized == "." || normalized == "" {
		return "", fmt.Errorf("非法压缩包路径: %s", name)
	}
	isWindowsVolume := len(normalized) >= 2 && normalized[1] == ':'
	if strings.HasPrefix(normalized, "../") || normalized == ".." || path.IsAbs(normalized) || filepath.IsAbs(normalized) || filepath.VolumeName(normalized) != "" || isWindowsVolume || strings.Contains(normalized, "\x00") {
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
