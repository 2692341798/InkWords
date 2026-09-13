package textbook

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
)

// TextbookSourceImportTaskSubtype is handled by parser-service, which only
// returns parsed structures. core-api validates and persists the final facts.
const TextbookSourceImportTaskSubtype = "textbook_source_import"

// SourceImportTaskVersion binds new imports to declaration-aware Go parsing.
const SourceImportTaskVersion = 3

// SourceImportTaskPayload freezes one local parse job. ArtifactToken resolves
// to immutable bytes in a dedicated local volume; raw source data is never
// placed in the task database, broker message, or model context.
type SourceImportTaskPayload struct {
	TaskVersion   int        `json:"task_version"`
	TaskSubtype   string     `json:"task_subtype"`
	ProjectID     string     `json:"project_id"`
	SourceID      string     `json:"source_id"`
	SnapshotID    string     `json:"snapshot_id"`
	SourceKind    SourceKind `json:"source_kind"`
	SourceRole    SourceRole `json:"source_role"`
	Locator       string     `json:"locator"`
	Filename      string     `json:"filename"`
	ArtifactToken string     `json:"artifact_token"`
	ContentHash   string     `json:"content_hash"`
	// ResolvedVersion records an immutable upstream identifier. Git imports
	// require the exact commit SHA rather than a floating branch or tag.
	ResolvedVersion string `json:"resolved_version"`
	ByteSize        int64  `json:"byte_size"`
	InputHash       string `json:"input_hash"`
}

func (payload SourceImportTaskPayload) Validate() error {
	if (payload.TaskVersion != 2 && payload.TaskVersion != SourceImportTaskVersion) || payload.TaskSubtype != TextbookSourceImportTaskSubtype || strings.TrimSpace(payload.ProjectID) == "" || strings.TrimSpace(payload.SourceID) == "" || strings.TrimSpace(payload.SnapshotID) == "" || strings.TrimSpace(payload.Locator) == "" || strings.TrimSpace(payload.Filename) == "" || strings.TrimSpace(payload.ArtifactToken) == "" || payload.ByteSize < 1 {
		return fmt.Errorf("source import task identity and content are required")
	}
	if err := payload.SourceKind.Validate(); err != nil {
		return err
	}
	if !isLocalFileSourceKind(payload.SourceKind) || !filenameMatchesSourceKind(payload.Filename, payload.SourceKind) {
		return fmt.Errorf("source import requires a matching local file source")
	}
	if payload.SourceKind == SourceKindGitRepository && !isGitCommitSHA(payload.ResolvedVersion) {
		return fmt.Errorf("git source import requires a 40-character commit SHA")
	}
	if payload.SourceRole != SourceRolePrimary && payload.SourceRole != SourceRoleOfficial {
		return fmt.Errorf("source import only accepts primary or confirmed official sources")
	}
	if !isSHA256Digest(payload.ContentHash) || !isSHA256Digest(payload.InputHash) || payload.ArtifactToken != payload.ContentHash {
		return fmt.Errorf("source import hashes are required")
	}
	return nil
}

// SourceImportTaskResult is the untrusted parser-service output. It is valid
// only when it exactly matches the frozen payload and all document provenance.
type SourceImportTaskResult struct {
	ResultVersion int              `json:"result_version"`
	TaskSubtype   string           `json:"task_subtype"`
	ProjectID     string           `json:"project_id"`
	SourceID      string           `json:"source_id"`
	SnapshotID    string           `json:"snapshot_id"`
	InputHash     string           `json:"input_hash"`
	Documents     []SourceDocument `json:"documents"`
	Chunks        []SourceChunk    `json:"chunks"`
}

// ValidateAgainst proves the worker neither changes source ownership nor
// smuggles chunks from another snapshot into a local textbook project.
func (result SourceImportTaskResult) ValidateAgainst(payload SourceImportTaskPayload) error {
	if err := payload.Validate(); err != nil {
		return err
	}
	if result.ResultVersion != 1 || result.TaskSubtype != TextbookSourceImportTaskSubtype || result.ProjectID != payload.ProjectID || result.SourceID != payload.SourceID || result.SnapshotID != payload.SnapshotID || result.InputHash != payload.InputHash {
		return fmt.Errorf("source import result does not match frozen task")
	}
	if len(result.Documents) == 0 || len(result.Chunks) == 0 {
		return fmt.Errorf("source import result requires documents and chunks")
	}
	documents := make(map[string]bool, len(result.Documents))
	for _, document := range result.Documents {
		if document.SnapshotID != result.SnapshotID || document.Validate() != nil || documents[document.ID] {
			return fmt.Errorf("invalid or duplicate imported document")
		}
		documents[document.ID] = true
	}
	chunks := make(map[string]bool, len(result.Chunks))
	for _, chunk := range result.Chunks {
		if !documents[chunk.DocumentID] || chunk.Validate() != nil || chunks[chunk.ID] {
			return fmt.Errorf("invalid or duplicate imported chunk")
		}
		chunks[chunk.ID] = true
	}
	return nil
}

// SourceImportInputHash is stable across retries and intentionally includes no
// generated prose, keeping source extraction separate from LLM token usage.
func SourceImportInputHash(projectID, sourceID, _ string, filename, contentHash string, resolvedVersions ...string) string {
	resolvedVersion := ""
	if len(resolvedVersions) > 0 {
		resolvedVersion = strings.TrimSpace(resolvedVersions[0])
	}
	canonical := fmt.Sprintf("parser_version=%d\n", SourceImportTaskVersion) + "project_id=" + projectID + "\nsource_id=" + sourceID + "\nfilename=" + filename + "\ncontent_hash=" + contentHash + "\nresolved_version=" + resolvedVersion
	sum := sha256.Sum256([]byte(canonical))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func isLocalFileSourceKind(kind SourceKind) bool {
	return kind == SourceKindPDF || kind == SourceKindDOCX || kind == SourceKindMarkdown || kind == SourceKindText || kind == SourceKindZIP || kind == SourceKindGitRepository
}

func filenameMatchesSourceKind(filename string, kind SourceKind) bool {
	extension := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	switch kind {
	case SourceKindPDF:
		return extension == ".pdf"
	case SourceKindDOCX:
		return extension == ".docx"
	case SourceKindMarkdown:
		return extension == ".md" || extension == ".markdown"
	case SourceKindText:
		return extension == ".txt"
	case SourceKindZIP:
		return extension == ".zip"
	case SourceKindGitRepository:
		return extension == ".md" || extension == ".markdown" || extension == ".txt" || isGitCodeExtension(extension)
	default:
		return false
	}
}

func isGitCodeExtension(extension string) bool {
	switch extension {
	case ".c", ".cpp", ".go", ".h", ".hpp", ".java", ".js", ".json", ".py", ".rs", ".sh", ".sql", ".ts", ".tsx", ".jsx", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func isGitCommitSHA(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 40 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
