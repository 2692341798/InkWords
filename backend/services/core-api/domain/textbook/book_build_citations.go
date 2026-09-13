package textbook

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// freezeBookCitations resolves only aliases from the approved document and
// captures immutable workspace-owned source metadata before the manifest hash.
func freezeBookCitations(tx *gorm.DB, projectID uuid.UUID, chapter bookBuildChapterSource) ([]sharedtextbook.CanonicalBookCitation, error) {
	markers := sharedtextbook.BookEvidenceMarkers(chapter.Markdown)
	if len(markers) == 0 {
		return nil, nil
	}
	var document struct {
		EvidenceAliases map[string]string `json:"evidence_aliases"`
	}
	if json.Unmarshal(chapter.DocumentJSON, &document) != nil {
		return nil, fmt.Errorf("%w: approved chapter has no citation aliases", ErrInvalidState)
	}
	seen, chunks := map[string]bool{}, []string{}
	for _, marker := range markers {
		id := document.EvidenceAliases[marker.ID]
		if !strings.HasPrefix(id, "evidence-") {
			return nil, fmt.Errorf("%w: approved citation alias %q is unresolved", ErrInvalidState, marker.ID)
		}
		chunkID := strings.TrimPrefix(id, "evidence-")
		if !seen[chunkID] {
			chunks = append(chunks, chunkID)
			seen[chunkID] = true
		}
	}
	rows, err := loadSourceEvidenceRows(tx, projectID, chunks)
	if err != nil {
		return nil, err
	}
	byEvidenceID := map[string]sharedtextbook.CanonicalBookCitation{}
	for _, row := range rows {
		var locator sharedtextbook.EvidenceLocator
		if json.Unmarshal(row.ChunkLocator, &locator) != nil {
			return nil, ErrInvalidState
		}
		snapshot := sharedtextbook.SourceSnapshot{ID: row.SnapshotID.String(), SourceID: row.SourceID.String(), Kind: row.SourceKind, Role: row.SourceRole, Locator: row.SourceLocator, ResolvedVersion: row.ResolvedVersion, ContentHash: sourceSnapshotDigest(row.SnapshotHash), CapturedAt: row.SnapshotCapturedAt}
		reference := sharedtextbook.EvidenceRef{ID: "evidence-" + row.ChunkID, SnapshotID: snapshot.ID, DocumentID: row.DocumentID, ChunkID: row.ChunkID, Locator: locator, ContentHash: row.ChunkTextHash, Confidence: sharedtextbook.EvidenceConfidenceDocumented, SourceRole: snapshot.Role}
		if snapshot.Validate() != nil || reference.Validate() != nil {
			return nil, fmt.Errorf("%w: invalid frozen citation source", ErrInvalidState)
		}
		byEvidenceID[reference.ID] = sharedtextbook.CanonicalBookCitation{Evidence: reference, Snapshot: snapshot}
	}
	var citations []sharedtextbook.CanonicalBookCitation
	seen = map[string]bool{}
	for _, marker := range markers {
		if seen[marker.ID] {
			continue
		}
		seen[marker.ID] = true
		citation := byEvidenceID[document.EvidenceAliases[marker.ID]]
		citation.ID = marker.ID
		citations = append(citations, citation)
	}
	return citations, nil
}
