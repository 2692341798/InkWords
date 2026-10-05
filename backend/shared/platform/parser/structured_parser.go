package parser

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"inkwords-backend/shared/kernel/textbook"
)

// StructuredParseRequest identifies the immutable source context for one parsed artifact.
// The parser never invents a source or snapshot identity, because evidence must remain
// traceable to the import that supplied it.
type StructuredParseRequest struct {
	SourceID         string
	SnapshotID       string
	CanonicalLocator string
	ArtifactPath     string
	Filename         string
	// LegacyCodeParagraphs replays imports frozen before declaration parsing.
	LegacyCodeParagraphs bool
	// LegacyOfficialWeb replays the tokenizer used by frozen v1 web tasks.
	LegacyOfficialWeb bool
	// PreserveOfficialWebTabs extracts explicitly associated inactive tab panels
	// for v3 tasks; false retains the frozen v2 visible-content projection.
	PreserveOfficialWebTabs bool
}

// Validate rejects a structure request that could not later be cited back to its source.
func (request StructuredParseRequest) Validate() error {
	if strings.TrimSpace(request.SourceID) == "" || strings.TrimSpace(request.SnapshotID) == "" {
		return fmt.Errorf("structured parse requires source id and snapshot id")
	}
	if strings.TrimSpace(request.CanonicalLocator) == "" || strings.TrimSpace(request.Filename) == "" {
		return fmt.Errorf("structured parse requires a canonical locator and filename")
	}
	return nil
}

// DocumentLink preserves a Markdown link without treating linked content as trusted input.
type DocumentLink struct {
	Text      string `json:"text"`
	Target    string `json:"target"`
	StartByte int    `json:"start_byte"`
	EndByte   int    `json:"end_byte"`
}

// StructuredDocument is a parsed artifact and its independently citeable chunks.
type StructuredDocument struct {
	SourceID string                  `json:"source_id"`
	Document textbook.SourceDocument `json:"document"`
	Chunks   []textbook.SourceChunk  `json:"chunks"`
	Links    []DocumentLink          `json:"links,omitempty"`
}

// StructuredParser keeps source structure intact for Markdown and TXT artifacts.
type StructuredParser struct{}

// maxStructuredTextBytes bounds the in-memory representation needed to build
// byte-accurate chunks. Larger raw files remain in the local artifact store,
// but must be split before import rather than exhausting the parser worker.
const maxStructuredTextBytes int64 = 64 << 20

// NewStructuredParser creates the deterministic, dependency-free structured parser.
func NewStructuredParser() *StructuredParser {
	return &StructuredParser{}
}

// Parse parses a Markdown or TXT artifact into stable, location-preserving chunks.
func (p *StructuredParser) Parse(src io.Reader, request StructuredParseRequest) (StructuredDocument, error) {
	if err := request.Validate(); err != nil {
		return StructuredDocument{}, err
	}

	content, err := io.ReadAll(io.LimitReader(src, maxStructuredTextBytes+1))
	if err != nil {
		return StructuredDocument{}, fmt.Errorf("read structured source: %w", err)
	}
	if int64(len(content)) > maxStructuredTextBytes {
		return StructuredDocument{}, fmt.Errorf("文本文件超过可安全解析大小限制；请拆分后重试")
	}
	if !utf8.Valid(content) {
		return StructuredDocument{}, fmt.Errorf("无法可靠解析该文件：文本不是有效 UTF-8。请转换为 UTF-8 后重试")
	}

	normalized := normalizeStructuredText(string(content))
	if strings.TrimSpace(normalized) == "" {
		return StructuredDocument{}, fmt.Errorf("无法可靠解析该文件：没有可用文本")
	}

	extension := strings.ToLower(filepath.Ext(request.Filename))
	switch extension {
	case ".md", ".markdown":
		return p.parseMarkdown(normalized, request)
	case ".txt":
		return p.parseText(normalized, request)
	default:
		if codeLanguage(request.Filename) != "" {
			return p.parseCode(normalized, request)
		}
		return StructuredDocument{}, fmt.Errorf("structured parser does not support %s", extension)
	}
}

func (p *StructuredParser) parseMarkdown(content string, request StructuredParseRequest) (StructuredDocument, error) {
	document := newStructuredDocument(request, markdownTitle(content, request.Filename), "text/markdown", content)
	chunks, links := buildMarkdownChunks(content, document.ID, request.ArtifactPath, request.CanonicalLocator)
	if len(chunks) == 0 {
		return StructuredDocument{}, fmt.Errorf("无法可靠解析 Markdown：没有可引用的段落或代码块")
	}
	return StructuredDocument{SourceID: request.SourceID, Document: document, Chunks: chunks, Links: links}, nil
}

func (p *StructuredParser) parseText(content string, request StructuredParseRequest) (StructuredDocument, error) {
	document := newStructuredDocument(request, filepath.Base(request.Filename), "text/plain", content)
	chunks := buildTextChunks(content, document.ID, request.ArtifactPath, request.CanonicalLocator)
	if len(chunks) == 0 {
		return StructuredDocument{}, fmt.Errorf("无法可靠解析文本：没有可引用的段落")
	}
	return StructuredDocument{SourceID: request.SourceID, Document: document, Chunks: chunks}, nil
}

// parseCode preserves fixed-source line ranges without pretending that source
// code has Markdown headings or prose paragraphs.
func (p *StructuredParser) parseCode(content string, request StructuredParseRequest) (StructuredDocument, error) {
	language := codeLanguage(request.Filename)
	document := newStructuredDocument(request, filepath.Base(request.Filename), codeMediaType(request.Filename), content)
	if language == "go" && !request.LegacyCodeParagraphs {
		chunks, err := buildGoChunks(content, document.ID, request.ArtifactPath, request.CanonicalLocator)
		if err != nil {
			return StructuredDocument{}, err
		}
		return StructuredDocument{SourceID: request.SourceID, Document: document, Chunks: chunks}, nil
	}
	chunks := buildTextChunks(content, document.ID, request.ArtifactPath, request.CanonicalLocator)
	if len(chunks) == 0 {
		return StructuredDocument{}, fmt.Errorf("无法可靠解析代码文件：没有可引用的文本")
	}
	for index := range chunks {
		chunks[index].CodeLanguage = language
	}
	return StructuredDocument{SourceID: request.SourceID, Document: document, Chunks: chunks}, nil
}

func newStructuredDocument(request StructuredParseRequest, title, mediaType, content string) textbook.SourceDocument {
	contentHash := sha256Digest(content)
	return textbook.SourceDocument{
		ID:               stableID("document", request.SnapshotID, request.CanonicalLocator, contentHash),
		SnapshotID:       request.SnapshotID,
		CanonicalLocator: request.CanonicalLocator,
		Title:            title,
		MediaType:        mediaType,
		ContentHash:      contentHash,
		ArtifactPath:     request.ArtifactPath,
	}
}

func newStructuredChunk(documentID string, ordinal, paragraph, startByte, endByte, startLine, endLine int, artifactPath, canonicalLocator, codeLanguage string, headingPath []string, text string) textbook.SourceChunk {
	textHash := sha256Digest(text)
	path := strings.TrimSpace(artifactPath)
	if path == "" {
		path = canonicalLocator
	}
	return textbook.SourceChunk{
		ID:           stableID("chunk", documentID, fmt.Sprint(ordinal), textHash),
		DocumentID:   documentID,
		Ordinal:      ordinal,
		HeadingPath:  append([]string(nil), headingPath...),
		Locator:      textbook.EvidenceLocator{Path: path, StartLine: startLine, EndLine: endLine, HeadingPath: append([]string(nil), headingPath...)},
		Paragraph:    paragraph,
		CodeLanguage: codeLanguage,
		StartByte:    startByte,
		EndByte:      endByte,
		TextHash:     textHash,
		SearchText:   text,
	}
}

func stableID(prefix string, parts ...string) string {
	return prefix + "-" + strings.TrimPrefix(sha256Digest(strings.Join(parts, "\x00")), "sha256:")[:24]
}

func sha256Digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func normalizeStructuredText(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return content
}
