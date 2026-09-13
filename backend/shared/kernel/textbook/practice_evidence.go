package textbook

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"
)

// PracticeSourceExcerpt retains source provenance separately from the exact
// excerpt digest used by an assessment. Imported chunk hashes may use another encoding.
type PracticeSourceExcerpt struct {
	Reference   EvidenceRef    `json:"reference"`
	Snapshot    SourceSnapshot `json:"snapshot"`
	Excerpt     string         `json:"excerpt"`
	ExcerptHash string         `json:"excerpt_hash"`
}

// PracticeEvidenceProjection is a narrow, read-only view of one approved task's sources.
type PracticeEvidenceProjection struct {
	Format      string                  `json:"format"`
	WorkspaceID string                  `json:"workspace_id"`
	ChapterID   string                  `json:"chapter_id"`
	RevisionID  string                  `json:"revision_id"`
	ContentHash string                  `json:"content_hash"`
	TaskID      string                  `json:"task_id"`
	Sources     []PracticeSourceExcerpt `json:"sources"`
}

// PracticeExcerptHash hashes the exact source bytes, without rewriting or truncation.
func PracticeExcerptHash(excerpt string) string {
	digest := sha256.Sum256([]byte(excerpt))
	return "sha256:" + hex.EncodeToString(digest[:])
}

// Validate rejects missing provenance, duplicate sources and altered excerpts.
func (projection PracticeEvidenceProjection) Validate() error {
	if projection.Format != "inkwords.practice-evidence.v1" || !practiceText(projection.WorkspaceID, 200) || !practiceText(projection.ChapterID, 200) || !practiceText(projection.RevisionID, 200) || !practiceText(projection.TaskID, 100) || !isSHA256Digest(projection.ContentHash) || len(projection.Sources) == 0 || len(projection.Sources) > 40 {
		return fmt.Errorf("invalid approved practice evidence projection")
	}
	seen := map[string]bool{}
	for _, source := range projection.Sources {
		ref, snapshot := source.Reference, source.Snapshot
		if ref.Validate() != nil || snapshot.Validate() != nil || ref.SnapshotID != snapshot.ID || ref.SourceRole != snapshot.Role || (snapshot.Role != SourceRolePrimary && snapshot.Role != SourceRoleOfficial) || seen[ref.ID] || !utf8.ValidString(source.Excerpt) || strings.TrimSpace(source.Excerpt) == "" || utf8.RuneCountInString(source.Excerpt) > 20000 || strings.ContainsRune(source.Excerpt, 0) || source.ExcerptHash != PracticeExcerptHash(source.Excerpt) {
			return fmt.Errorf("invalid approved practice source identity")
		}
		seen[ref.ID] = true
	}
	return nil
}
