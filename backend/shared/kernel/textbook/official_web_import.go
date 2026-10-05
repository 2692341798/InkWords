package textbook

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
)

// TextbookOfficialWebImportTaskSubtype identifies a bounded crawler task. It
// intentionally differs from TextbookSourceImportTaskSubtype: no browser
// upload artifact exists for a remote official-document stack.
const TextbookOfficialWebImportTaskSubtype = "textbook_official_web_import"

// OfficialWebParserVersion identifies the immutable HTML extraction rules.
const OfficialWebParserVersion = "inkwords.official-html.v3"

// OfficialWebVisibleContentParserVersion retains deterministic replay of v2
// tasks, which intentionally omitted inactive tab panels.
const OfficialWebVisibleContentParserVersion = "inkwords.official-html.v2"

// OfficialWebImportTaskPayload freezes the project-owned URL boundary before
// parser-service accesses a remote server. Source bytes and the manifest hash
// are produced by the worker and validated by core-api before persistence.
type OfficialWebImportTaskPayload struct {
	TaskVersion         int        `json:"task_version"`
	ParserVersion       string     `json:"parser_version,omitempty"`
	TaskSubtype         string     `json:"task_subtype"`
	ProjectID           string     `json:"project_id"`
	SourceID            string     `json:"source_id"`
	SnapshotID          string     `json:"snapshot_id"`
	SourceKind          SourceKind `json:"source_kind"`
	SourceRole          SourceRole `json:"source_role"`
	EntryURL            string     `json:"entry_url"`
	AllowedPathPrefixes []string   `json:"allowed_path_prefixes"`
	InputHash           string     `json:"input_hash"`
}

// Validate rejects a request before it reaches a remote host. The detailed
// SSRF, redirect and DNS checks remain the crawler's responsibility.
func (payload OfficialWebImportTaskPayload) Validate() error {
	if (payload.TaskVersion != 1 && payload.TaskVersion != 2) || payload.TaskSubtype != TextbookOfficialWebImportTaskSubtype || strings.TrimSpace(payload.ProjectID) == "" || strings.TrimSpace(payload.SourceID) == "" || strings.TrimSpace(payload.SnapshotID) == "" || payload.SourceKind != SourceKindOfficialWeb || payload.SourceRole != SourceRoleOfficial || !isSHA256Digest(payload.InputHash) {
		return fmt.Errorf("official web import identity is invalid")
	}
	if payload.TaskVersion == 1 && payload.ParserVersion != "" || payload.TaskVersion == 2 && ((payload.ParserVersion != OfficialWebParserVersion && payload.ParserVersion != OfficialWebVisibleContentParserVersion) || payload.InputHash != OfficialWebImportInputHash(payload.ProjectID, payload.SourceID, payload.SnapshotID, payload.EntryURL, payload.AllowedPathPrefixes, payload.ParserVersion)) {
		return fmt.Errorf("official web import parser does not match frozen task")
	}
	entry, err := url.Parse(strings.TrimSpace(payload.EntryURL))
	if err != nil || entry.Scheme != "https" || entry.Hostname() == "" || entry.User != nil {
		return fmt.Errorf("official web import requires a public HTTPS entry URL")
	}
	if len(payload.AllowedPathPrefixes) == 0 {
		return fmt.Errorf("official web import requires an allowed path boundary")
	}
	for _, prefix := range payload.AllowedPathPrefixes {
		prefix = strings.TrimSpace(prefix)
		if !strings.HasPrefix(prefix, "/") || strings.Contains(prefix, "?") || strings.Contains(prefix, "#") {
			return fmt.Errorf("official web import path boundary is invalid")
		}
	}
	return nil
}

// OfficialWebImportInputHash is stable across retries of the same bounded
// task. It deliberately includes no remote bytes: those become the result's
// immutable manifest hash only after the crawler has completed.
func OfficialWebImportInputHash(projectID, sourceID, snapshotID, entryURL string, prefixes []string, parserVersion ...string) string {
	normalized := append([]string(nil), prefixes...)
	for index := range normalized {
		normalized[index] = strings.TrimSpace(normalized[index])
	}
	slices.Sort(normalized)
	identity := "project_id=" + projectID + "\nsource_id=" + sourceID + "\nsnapshot_id=" + snapshotID + "\nentry_url=" + strings.TrimSpace(entryURL) + "\npaths=" + strings.Join(normalized, ",")
	if len(parserVersion) > 0 && parserVersion[0] != "" {
		identity += "\nparser_version=" + parserVersion[0]
	}
	sum := sha256.Sum256([]byte(identity))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// OfficialWebImportTaskResult is the untrusted parser-service output for a
// bounded documentation stack. ContentHash is the crawler manifest's stable
// source-input hash, never an HTTP cache validator or fetch timestamp.
type OfficialWebImportTaskResult struct {
	ResultVersion   int              `json:"result_version"`
	ParserVersion   string           `json:"parser_version,omitempty"`
	ManifestHash    string           `json:"manifest_hash,omitempty"`
	CrawlManifest   json.RawMessage  `json:"crawl_manifest,omitempty"`
	TaskSubtype     string           `json:"task_subtype"`
	ProjectID       string           `json:"project_id"`
	SourceID        string           `json:"source_id"`
	SnapshotID      string           `json:"snapshot_id"`
	InputHash       string           `json:"input_hash"`
	ResolvedVersion string           `json:"resolved_version"`
	ContentHash     string           `json:"content_hash"`
	CapturedAt      time.Time        `json:"captured_at"`
	Documents       []SourceDocument `json:"documents"`
	Chunks          []SourceChunk    `json:"chunks"`
}

// ValidateAgainst verifies the worker could not repoint a project, source or
// snapshot. A partial/failed crawler result must never be converted to an
// apparently complete source snapshot.
func (result OfficialWebImportTaskResult) ValidateAgainst(payload OfficialWebImportTaskPayload) error {
	if err := payload.Validate(); err != nil {
		return err
	}
	if result.ResultVersion != payload.TaskVersion || result.TaskSubtype != TextbookOfficialWebImportTaskSubtype || result.ProjectID != payload.ProjectID || result.SourceID != payload.SourceID || result.SnapshotID != payload.SnapshotID || result.InputHash != payload.InputHash || result.ResolvedVersion != result.ContentHash || !isSHA256Digest(result.ContentHash) || result.CapturedAt.IsZero() {
		return fmt.Errorf("official web import result does not match frozen task")
	}
	if len(result.Documents) == 0 || len(result.Chunks) == 0 {
		return fmt.Errorf("official web import result requires documents and chunks")
	}
	if payload.TaskVersion == 2 {
		if err := result.validateCrawlAudit(payload); err != nil {
			return err
		}
	}
	documents := make(map[string]bool, len(result.Documents))
	for _, document := range result.Documents {
		if document.SnapshotID != result.SnapshotID || document.Validate() != nil || documents[document.ID] {
			return fmt.Errorf("invalid or duplicate official web document")
		}
		documents[document.ID] = true
	}
	chunks := make(map[string]bool, len(result.Chunks))
	for _, chunk := range result.Chunks {
		if !documents[chunk.DocumentID] || chunk.Validate() != nil || chunks[chunk.ID] {
			return fmt.Errorf("invalid or duplicate official web chunk")
		}
		chunks[chunk.ID] = true
	}
	return nil
}
