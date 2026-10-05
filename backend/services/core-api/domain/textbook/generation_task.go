package textbook

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// maxGenerationEvidenceRunes is deliberately a source-window limit, not a
// provider context limit. Prompt instructions and generated output need their
// own capacity, and Chinese prose and source code can tokenize more densely
// than ordinary English text. Authors must reduce the reviewed evidence set
// when it exceeds this bound instead of receiving a silently shortened pack.
const maxGenerationEvidenceRunes = 30_000

// PrepareSampleGeneration creates a worker-safe, immutable input using only evidence selected by
// the approved blueprint. It acquires shared locks so a concurrent approval cannot mix revisions.
func (r *GormRepository) PrepareSampleGeneration(ctx context.Context, workspaceID, projectID, chapterID uuid.UUID, target sharedtextbook.SampleGenerationTarget) (sharedtextbook.SampleGenerationTaskPayload, error) {
	var payload sharedtextbook.SampleGenerationTaskPayload
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var project Project
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).Where("id = ? AND workspace_id = ?", projectID, workspaceID).First(&project).Error; err != nil {
			return textbookNotFound(err)
		}
		if project.ApprovedBookContractRevisionID == nil || project.ApprovedStyleSheetRevisionID == nil || project.ApprovedBlueprintRevisionID == nil {
			return ErrInvalidState
		}

		var chapter Chapter
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).Where("id = ? AND project_id = ?", chapterID, project.ID).First(&chapter).Error; err != nil {
			return textbookNotFound(err)
		}

		book, style, blueprint, err := approvedGenerationDocuments(tx, project)
		if err != nil {
			return err
		}
		blueprintChapter, ok := blueprintChapterByID(blueprint, chapter.ID.String())
		if !ok {
			return ErrInvalidState
		}
		if err := blueprintChapter.ValidateCriticalClaimCoverage(); err != nil {
			return fmt.Errorf("%w: %w", ErrClaimCoverageIncomplete, err)
		}

		pack, err := loadBlueprintEvidencePack(tx, project.ID, blueprintChapter.EvidenceIDs)
		if err != nil {
			return err
		}
		if runeCount := generationEvidenceRuneCount(pack); runeCount > maxGenerationEvidenceRunes {
			return fmt.Errorf("%w: %d source characters selected; limit is %d", ErrEvidenceBudgetExceeded, runeCount, maxGenerationEvidenceRunes)
		}
		payload = sharedtextbook.SampleGenerationTaskPayload{
			TaskVersion:            sharedtextbook.SampleGenerationTaskVersion,
			TaskSubtype:            sharedtextbook.TextbookSampleGenerationTaskSubtype,
			PromptSchemaVersion:    sharedtextbook.SamplePromptSchemaVersion,
			QualityContractVersion: sharedtextbook.SampleQualityContractVersion,
			GenerationTarget:       target,
			ProjectID:              project.ID.String(),
			ChapterID:              chapter.ID.String(),
			ExpectedChapterVersion: chapter.RevisionVersion,
			Audience:               project.AudienceLevel,
			BookContract:           book,
			StyleSheet:             style,
			Blueprint:              blueprint,
			EvidencePack:           pack,
		}
		if chapter.CurrentRevisionID != nil {
			payload.ParentRevisionID = chapter.CurrentRevisionID.String()
		}
		payload.InputHash = sharedtextbook.SampleGenerationInputHash(payload)
		return payload.Validate()
	})
	if err != nil {
		return sharedtextbook.SampleGenerationTaskPayload{}, err
	}
	return payload, nil
}

func generationEvidenceRuneCount(pack sharedtextbook.GenerationEvidencePack) int {
	count := 0
	for _, excerpt := range pack.Excerpts {
		count += len([]rune(excerpt))
	}
	return count
}

func textbookNotFound(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func approvedGenerationDocuments(tx *gorm.DB, project Project) (sharedtextbook.BookContract, sharedtextbook.StyleSheet, sharedtextbook.Blueprint, error) {
	var bookRevision BookContractRevision
	if err := tx.Where("id = ? AND project_id = ? AND status = ?", project.ApprovedBookContractRevisionID, project.ID, StatusApproved).First(&bookRevision).Error; err != nil {
		return sharedtextbook.BookContract{}, sharedtextbook.StyleSheet{}, sharedtextbook.Blueprint{}, textbookNotFound(err)
	}
	var styleRevision StyleSheetRevision
	if err := tx.Where("id = ? AND project_id = ? AND status = ?", project.ApprovedStyleSheetRevisionID, project.ID, StatusApproved).First(&styleRevision).Error; err != nil {
		return sharedtextbook.BookContract{}, sharedtextbook.StyleSheet{}, sharedtextbook.Blueprint{}, textbookNotFound(err)
	}
	var blueprintRevision BlueprintRevision
	if err := tx.Where("id = ? AND project_id = ? AND status = ?", project.ApprovedBlueprintRevisionID, project.ID, StatusApproved).First(&blueprintRevision).Error; err != nil {
		return sharedtextbook.BookContract{}, sharedtextbook.StyleSheet{}, sharedtextbook.Blueprint{}, textbookNotFound(err)
	}
	var book sharedtextbook.BookContract
	var style sharedtextbook.StyleSheet
	var blueprint sharedtextbook.Blueprint
	if json.Unmarshal(bookRevision.DocumentJSON, &book) != nil || json.Unmarshal(styleRevision.DocumentJSON, &style) != nil || json.Unmarshal(blueprintRevision.DocumentJSON, &blueprint) != nil || book.Validate() != nil || style.Validate() != nil || blueprint.Validate() != nil {
		return sharedtextbook.BookContract{}, sharedtextbook.StyleSheet{}, sharedtextbook.Blueprint{}, ErrInvalidState
	}
	if book.RevisionID != bookRevision.ID.String() || style.RevisionID != styleRevision.ID.String() || blueprint.RevisionID != blueprintRevision.ID.String() || blueprint.BookContractRevision != book.RevisionID || blueprint.StyleSheetRevision != style.RevisionID {
		return sharedtextbook.BookContract{}, sharedtextbook.StyleSheet{}, sharedtextbook.Blueprint{}, ErrInvalidState
	}
	return book, style, blueprint, nil
}

func blueprintChapterByID(blueprint sharedtextbook.Blueprint, chapterID string) (sharedtextbook.BlueprintChapter, bool) {
	for _, volume := range blueprint.Volumes {
		for _, chapter := range volume.Chapters {
			if chapter.ID == chapterID {
				return chapter, true
			}
		}
	}
	return sharedtextbook.BlueprintChapter{}, false
}

type generationEvidenceRow struct {
	ChunkID            string                    `gorm:"column:chunk_id"`
	DocumentID         string                    `gorm:"column:document_id"`
	ChunkOrdinal       int                       `gorm:"column:chunk_ordinal"`
	ChunkLocator       []byte                    `gorm:"column:chunk_locator"`
	ChunkTextHash      string                    `gorm:"column:chunk_text_hash"`
	ChunkSearchText    string                    `gorm:"column:chunk_search_text"`
	SnapshotID         uuid.UUID                 `gorm:"column:snapshot_id"`
	SourceID           uuid.UUID                 `gorm:"column:source_id"`
	SourceKind         sharedtextbook.SourceKind `gorm:"column:source_kind"`
	SourceRole         sharedtextbook.SourceRole `gorm:"column:source_role"`
	SourceLocator      string                    `gorm:"column:source_locator"`
	ResolvedVersion    string                    `gorm:"column:resolved_version"`
	SnapshotHash       string                    `gorm:"column:snapshot_hash"`
	SnapshotCapturedAt time.Time                 `gorm:"column:snapshot_captured_at"`
}

func loadBlueprintEvidencePack(tx *gorm.DB, projectID uuid.UUID, evidenceIDs []string) (sharedtextbook.GenerationEvidencePack, error) {
	rows, err := loadSourceEvidenceRows(tx, projectID, evidenceIDs)
	if err != nil {
		return sharedtextbook.GenerationEvidencePack{}, err
	}
	return generationEvidencePackFromRows(rows)
}

// loadSourceEvidenceRows scopes every requested chunk to this project's sources.
// Publication citations reuse the same ownership query without requiring a
// chapter's citations to include a primary source used for generation.
func loadSourceEvidenceRows(tx *gorm.DB, projectID uuid.UUID, evidenceIDs []string) ([]generationEvidenceRow, error) {
	if len(evidenceIDs) == 0 {
		return nil, ErrInvalidState
	}
	var rows []generationEvidenceRow
	err := tx.Table("source_chunks").
		Select("source_chunks.id AS chunk_id, source_chunks.document_id, source_chunks.ordinal AS chunk_ordinal, source_chunks.locator AS chunk_locator, source_chunks.text_hash AS chunk_text_hash, source_chunks.search_text AS chunk_search_text, source_snapshots.id AS snapshot_id, source_snapshots.source_id, textbook_sources.kind AS source_kind, textbook_sources.role AS source_role, textbook_sources.locator AS source_locator, source_snapshots.resolved_version, source_snapshots.content_hash AS snapshot_hash, source_snapshots.captured_at AS snapshot_captured_at").
		Joins("JOIN source_documents ON source_documents.id = source_chunks.document_id").
		Joins("JOIN source_snapshots ON source_snapshots.id = source_documents.snapshot_id").
		Joins("JOIN textbook_sources ON textbook_sources.id = source_snapshots.source_id").
		Where("textbook_sources.project_id = ? AND textbook_sources.deleted_at IS NULL AND source_chunks.id IN ?", projectID, evidenceIDs).
		Order("source_chunks.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) != len(evidenceIDs) {
		return nil, ErrInvalidState
	}

	requested := make(map[string]bool, len(evidenceIDs))
	for _, id := range evidenceIDs {
		if requested[id] {
			return nil, ErrInvalidState
		}
		requested[id] = true
	}
	return rows, nil
}

func generationEvidencePackFromRows(rows []generationEvidenceRow) (sharedtextbook.GenerationEvidencePack, error) {
	pack := sharedtextbook.GenerationEvidencePack{Excerpts: make(map[string]string, len(rows))}
	snapshots := make(map[uuid.UUID]sharedtextbook.SourceSnapshot)
	for _, row := range rows {
		var locator sharedtextbook.EvidenceLocator
		if err := json.Unmarshal(row.ChunkLocator, &locator); err != nil || locator.Validate() != nil {
			return sharedtextbook.GenerationEvidencePack{}, ErrInvalidState
		}
		snapshot, exists := snapshots[row.SnapshotID]
		if !exists {
			snapshot = sharedtextbook.SourceSnapshot{ID: row.SnapshotID.String(), SourceID: row.SourceID.String(), Kind: row.SourceKind, Role: row.SourceRole, Locator: row.SourceLocator, ResolvedVersion: row.ResolvedVersion, ContentHash: sourceSnapshotDigest(row.SnapshotHash), CapturedAt: row.SnapshotCapturedAt}
			if err := snapshot.Validate(); err != nil {
				return sharedtextbook.GenerationEvidencePack{}, fmt.Errorf("validate evidence snapshot: %w", err)
			}
			snapshots[row.SnapshotID] = snapshot
		}
		if snapshot.Role == sharedtextbook.SourceRolePrimary {
			if pack.PrimarySnapshot.ID == "" {
				pack.PrimarySnapshot = snapshot
			} else if pack.PrimarySnapshot.ID != snapshot.ID {
				found := false
				for _, current := range pack.PrimarySnapshots {
					if current.ID == snapshot.ID {
						found = true
						break
					}
				}
				if !found {
					pack.PrimarySnapshots = append(pack.PrimarySnapshots, snapshot)
				}
			}
		} else if snapshot.Role == sharedtextbook.SourceRoleOfficial {
			found := false
			for _, current := range pack.OfficialSources {
				if current.ID == snapshot.ID {
					found = true
					break
				}
			}
			if !found {
				pack.OfficialSources = append(pack.OfficialSources, snapshot)
			}
		} else {
			return sharedtextbook.GenerationEvidencePack{}, ErrInvalidState
		}
		evidenceID := "evidence-" + row.ChunkID
		pack.Evidence = append(pack.Evidence, sharedtextbook.EvidenceRef{ID: evidenceID, SnapshotID: snapshot.ID, DocumentID: row.DocumentID, ChunkID: row.ChunkID, Locator: locator, ContentHash: row.ChunkTextHash, Confidence: sharedtextbook.EvidenceConfidenceDocumented, SourceRole: snapshot.Role})
		pack.Excerpts[evidenceID] = row.ChunkSearchText
	}
	if err := pack.Validate(); err != nil {
		return sharedtextbook.GenerationEvidencePack{}, fmt.Errorf("validate generation evidence pack: %w", err)
	}
	return pack, nil
}

// sourceSnapshotDigest bridges the existing CHAR(64) storage column and the explicit shared-contract format.
func sourceSnapshotDigest(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "sha256:") || trimmed == "" {
		return trimmed
	}
	return "sha256:" + trimmed
}
