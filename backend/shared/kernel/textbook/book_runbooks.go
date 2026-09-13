package textbook

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
)

// BookVideoRunbooksFormat identifies companion guides frozen with a book.
const BookVideoRunbooksFormat = "inkwords.book-video-runbooks.v1"

// FrozenChapterVideoRunbook binds a guide, including explicit absence, to the
// approved revision and manuscript bytes. It never represents capture evidence.
type FrozenChapterVideoRunbook struct {
	ChapterID    string                  `json:"chapter_id"`
	RevisionID   string                  `json:"revision_id"`
	ContentHash  string                  `json:"content_hash"`
	RunbookHash  string                  `json:"video_runbook_hash"`
	VideoRunbook *VideoRunbookProjection `json:"video_runbook"`
}

// BookVideoRunbooks is a versioned companion projection, not editable prose.
type BookVideoRunbooks struct {
	Format   string                      `json:"format"`
	Chapters []FrozenChapterVideoRunbook `json:"chapters"`
}

// VideoRunbookHash uses the canonical typed JSON value, including null, so
// PostgreSQL JSONB formatting cannot invalidate an unchanged projection.
func VideoRunbookHash(runbook *VideoRunbookProjection) (string, error) {
	if runbook != nil {
		if err := runbook.Validate(); err != nil {
			return "", err
		}
	}
	raw, err := json.Marshal(runbook)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(raw)), nil
}

// ValidateAgainstBook rejects duplicate, altered or cross-book companions.
func (s BookVideoRunbooks) ValidateAgainstBook(book CanonicalBookAST) error {
	if s.Format != BookVideoRunbooksFormat || book.Validate() != nil || len(s.Chapters) != len(book.Chapters) {
		return fmt.Errorf("invalid frozen book video runbooks")
	}
	chapters := make(map[string]string, len(book.Chapters))
	for _, chapter := range book.Chapters {
		chapters[chapter.ID] = chapter.ContentHash
	}
	seen, revisions := map[string]bool{}, map[string]bool{}
	for _, chapter := range s.Chapters {
		hash, err := VideoRunbookHash(chapter.VideoRunbook)
		if chapter.ChapterID == "" || chapter.RevisionID == "" || seen[chapter.ChapterID] || revisions[chapter.RevisionID] || !isFullSHA256Digest(chapter.ContentHash) || chapters[chapter.ChapterID] != chapter.ContentHash || err != nil || hash != chapter.RunbookHash {
			return fmt.Errorf("frozen video runbook identity or hash mismatch")
		}
		seen[chapter.ChapterID], revisions[chapter.RevisionID] = true, true
	}
	return nil
}

// ReadBookVideoRunbooks reads only frozen manifest fields. Legacy builds return
// nil rather than resolving today's chapter and rewriting history on export.
func ReadBookVideoRunbooks(raw json.RawMessage) (*BookVideoRunbooks, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var manifest struct {
		Book         CanonicalBookAST            `json:"book"`
		Chapters     []FrozenChapterVideoRunbook `json:"chapters"`
		ToolVersions map[string]string           `json:"tool_versions"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, err
	}
	version := manifest.ToolVersions["video_runbooks"]
	if version == "" {
		for _, chapter := range manifest.Chapters {
			if chapter.VideoRunbook != nil || chapter.RunbookHash != "" {
				return nil, fmt.Errorf("undeclared book video runbooks")
			}
		}
		return nil, nil
	}
	snapshot := BookVideoRunbooks{Format: version, Chapters: manifest.Chapters}
	for index := range snapshot.Chapters {
		chapter := &snapshot.Chapters[index]
		chapter.ContentHash = "sha256:" + strings.TrimPrefix(chapter.ContentHash, "sha256:")
	}
	if err := snapshot.ValidateAgainstBook(manifest.Book); err != nil {
		return nil, err
	}
	return &snapshot, nil
}
