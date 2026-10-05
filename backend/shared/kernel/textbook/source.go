package textbook

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// SourceSnapshot identifies one immutable imported source version.
type SourceSnapshot struct {
	ID              string     `json:"id"`
	SourceID        string     `json:"source_id"`
	Kind            SourceKind `json:"kind"`
	Role            SourceRole `json:"role"`
	Locator         string     `json:"locator"`
	ResolvedVersion string     `json:"resolved_version"`
	ContentHash     string     `json:"content_hash"`
	CapturedAt      time.Time  `json:"captured_at"`
}

func (snapshot SourceSnapshot) Validate() error {
	if strings.TrimSpace(snapshot.ID) == "" || strings.TrimSpace(snapshot.SourceID) == "" || strings.TrimSpace(snapshot.Locator) == "" {
		return fmt.Errorf("snapshot id, source id and locator are required")
	}
	if err := snapshot.Kind.Validate(); err != nil {
		return err
	}
	if err := snapshot.Role.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(snapshot.ResolvedVersion) == "" || isFloatingVersion(snapshot.ResolvedVersion) {
		return fmt.Errorf("snapshot resolved version must be immutable")
	}
	if snapshot.Kind == SourceKindGitRepository {
		if len(snapshot.ResolvedVersion) != 40 {
			return fmt.Errorf("git snapshot resolved version must be a 40-character SHA")
		}
		if _, err := hex.DecodeString(snapshot.ResolvedVersion); err != nil {
			return fmt.Errorf("git snapshot resolved version must be hexadecimal: %w", err)
		}
	}
	if !isSHA256Digest(snapshot.ContentHash) || snapshot.CapturedAt.IsZero() {
		return fmt.Errorf("snapshot content hash and capture time are required")
	}
	return nil
}

// EvidenceLocator preserves a stable location without requiring one file type.
type EvidenceLocator struct {
	Path        string   `json:"path,omitempty"`
	URL         string   `json:"url,omitempty"`
	Symbol      string   `json:"symbol,omitempty"`
	StartLine   int      `json:"start_line,omitempty"`
	EndLine     int      `json:"end_line,omitempty"`
	Page        int      `json:"page,omitempty"`
	HeadingPath []string `json:"heading_path,omitempty"`
}

func (locator EvidenceLocator) Validate() error {
	if strings.TrimSpace(locator.Path) == "" && strings.TrimSpace(locator.URL) == "" {
		return fmt.Errorf("evidence locator requires a path or URL")
	}
	if locator.StartLine != 0 || locator.EndLine != 0 {
		if locator.StartLine < 1 || locator.EndLine < locator.StartLine {
			return fmt.Errorf("invalid evidence line range")
		}
	}
	if locator.Page < 0 {
		return fmt.Errorf("evidence page cannot be negative")
	}
	if locator.StartLine == 0 && locator.Page == 0 && len(locator.HeadingPath) == 0 && strings.TrimSpace(locator.Symbol) == "" {
		return fmt.Errorf("evidence locator requires a symbol, line, page or heading")
	}
	return nil
}

// EvidenceRef is an immutable, source-scoped reference used by claims.
type EvidenceRef struct {
	ID          string             `json:"id"`
	SnapshotID  string             `json:"snapshot_id"`
	DocumentID  string             `json:"document_id"`
	ChunkID     string             `json:"chunk_id,omitempty"`
	Locator     EvidenceLocator    `json:"locator"`
	ContentHash string             `json:"content_hash"`
	Confidence  EvidenceConfidence `json:"confidence"`
	SourceRole  SourceRole         `json:"source_role"`
}

func (reference EvidenceRef) Validate() error {
	if strings.TrimSpace(reference.ID) == "" || strings.TrimSpace(reference.SnapshotID) == "" || strings.TrimSpace(reference.DocumentID) == "" {
		return fmt.Errorf("evidence id, snapshot id and document id are required")
	}
	if err := reference.Locator.Validate(); err != nil {
		return err
	}
	if !isSHA256Digest(reference.ContentHash) {
		return fmt.Errorf("evidence content hash is required")
	}
	if err := reference.Confidence.Validate(); err != nil {
		return err
	}
	return reference.SourceRole.Validate()
}

// Claim is a textbook assertion whose cited evidence is validated by the owning application.
type Claim struct {
	ID          string      `json:"id"`
	Text        string      `json:"text"`
	Kind        string      `json:"kind"`
	Critical    bool        `json:"critical"`
	EvidenceIDs []string    `json:"evidence_ids"`
	Status      ClaimStatus `json:"status"`
}

func (claim Claim) Validate() error {
	if strings.TrimSpace(claim.ID) == "" || strings.TrimSpace(claim.Text) == "" || strings.TrimSpace(claim.Kind) == "" || len(claim.EvidenceIDs) == 0 {
		return fmt.Errorf("claim identity and evidence are required")
	}
	return claim.Status.Validate()
}

// SourceDocument is one parsed artifact within an immutable snapshot.
type SourceDocument struct {
	ID               string `json:"id"`
	SnapshotID       string `json:"snapshot_id"`
	CanonicalLocator string `json:"canonical_locator"`
	Title            string `json:"title"`
	MediaType        string `json:"media_type"`
	ContentHash      string `json:"content_hash"`
	ParentID         string `json:"parent_id,omitempty"`
	ArtifactPath     string `json:"artifact_path,omitempty"`
}

func (document SourceDocument) Validate() error {
	if strings.TrimSpace(document.ID) == "" || strings.TrimSpace(document.SnapshotID) == "" || strings.TrimSpace(document.CanonicalLocator) == "" || strings.TrimSpace(document.Title) == "" || strings.TrimSpace(document.MediaType) == "" {
		return fmt.Errorf("source document identity is incomplete")
	}
	if !isSHA256Digest(document.ContentHash) {
		return fmt.Errorf("source document content hash is required")
	}
	return nil
}

// SourceChunk is a searchable, location-preserving fragment of one source document.
type SourceChunk struct {
	ID           string          `json:"id"`
	DocumentID   string          `json:"document_id"`
	Ordinal      int             `json:"ordinal"`
	HeadingPath  []string        `json:"heading_path,omitempty"`
	Locator      EvidenceLocator `json:"locator"`
	Paragraph    int             `json:"paragraph,omitempty"`
	CodeLanguage string          `json:"code_language,omitempty"`
	StartByte    int             `json:"start_byte,omitempty"`
	EndByte      int             `json:"end_byte,omitempty"`
	TextHash     string          `json:"text_hash"`
	SearchText   string          `json:"search_text"`
}

func (chunk SourceChunk) Validate() error {
	if strings.TrimSpace(chunk.ID) == "" || strings.TrimSpace(chunk.DocumentID) == "" || chunk.Ordinal < 1 || strings.TrimSpace(chunk.SearchText) == "" {
		return fmt.Errorf("source chunk identity is incomplete")
	}
	if err := chunk.Locator.Validate(); err != nil {
		return err
	}
	if !isSHA256Digest(chunk.TextHash) {
		return fmt.Errorf("source chunk text hash is required")
	}
	if chunk.Paragraph < 0 || chunk.StartByte < 0 || chunk.EndByte < 0 {
		return fmt.Errorf("source chunk offsets cannot be negative")
	}
	if chunk.EndByte != 0 && chunk.EndByte <= chunk.StartByte {
		return fmt.Errorf("source chunk byte range is invalid")
	}
	return nil
}

func isFloatingVersion(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "head", "main", "master", "latest":
		return true
	default:
		return false
	}
}

func isSHA256Digest(value string) bool {
	return strings.HasPrefix(value, "sha256:") && len(strings.TrimPrefix(value, "sha256:")) > 0
}
