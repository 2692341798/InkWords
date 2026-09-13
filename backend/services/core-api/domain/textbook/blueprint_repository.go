package textbook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// CreateBlueprint saves a draft outline bound to the currently approved generation contracts.
func (r *GormRepository) CreateBlueprint(ctx context.Context, workspaceID uuid.UUID, input CreateBlueprintInput) (*BlueprintRevision, error) {
	var revision *BlueprintRevision
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		project, err := lockWorkspaceProject(tx, workspaceID, input.ProjectID)
		if err != nil {
			return err
		}
		if project.ApprovedBookContractRevisionID == nil || project.ApprovedStyleSheetRevisionID == nil {
			return ErrInvalidState
		}
		if err := validateBlueprintChapters(tx, project.ID, input.Volumes); err != nil {
			return err
		}
		if err := validateBlueprintEvidence(tx, project.ID, input.Volumes); err != nil {
			return err
		}
		var latest BlueprintRevision
		next := 1
		if err := tx.Where("project_id = ?", project.ID).Order("revision_number DESC").First(&latest).Error; err == nil {
			next = latest.RevisionNumber + 1
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		id := uuid.New()
		blueprint := sharedtextbook.Blueprint{
			RevisionID: id.String(), ProjectID: project.ID.String(), RevisionNumber: next,
			ContentHash:          digestJSON(blueprintPayload(*project.ApprovedBookContractRevisionID, *project.ApprovedStyleSheetRevisionID, input.Volumes)),
			BookContractRevision: project.ApprovedBookContractRevisionID.String(),
			StyleSheetRevision:   project.ApprovedStyleSheetRevisionID.String(), Volumes: input.Volumes,
		}
		if err := blueprint.Validate(); err != nil {
			return fmt.Errorf("validate blueprint: %w", err)
		}
		document, err := json.Marshal(blueprint)
		if err != nil {
			return err
		}
		revision = &BlueprintRevision{ID: id, ProjectID: project.ID, RevisionNumber: next, DocumentJSON: datatypes.JSON(document), ContentHash: strings.TrimPrefix(blueprint.ContentHash, "sha256:"), Status: StatusDraft}
		return tx.Create(revision).Error
	})
	return revision, err
}

// ApproveBlueprint makes a reviewed outline eligible only while it references the current contracts.
func (r *GormRepository) ApproveBlueprint(ctx context.Context, workspaceID, projectID, revisionID uuid.UUID) (*BlueprintRevision, error) {
	var revision BlueprintRevision
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		project, err := lockWorkspaceProject(tx, workspaceID, projectID)
		if err != nil {
			return err
		}
		if project.ApprovedBookContractRevisionID == nil || project.ApprovedStyleSheetRevisionID == nil {
			return ErrInvalidState
		}
		if err := tx.Where("id = ? AND project_id = ?", revisionID, projectID).First(&revision).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		var blueprint sharedtextbook.Blueprint
		if err := json.Unmarshal(revision.DocumentJSON, &blueprint); err != nil || blueprint.Validate() != nil {
			return ErrInvalidState
		}
		if blueprint.RevisionID != revision.ID.String() || blueprint.ProjectID != project.ID.String() || blueprint.RevisionNumber != revision.RevisionNumber || blueprint.ContentHash != "sha256:"+revision.ContentHash || blueprint.BookContractRevision != project.ApprovedBookContractRevisionID.String() || blueprint.StyleSheetRevision != project.ApprovedStyleSheetRevisionID.String() {
			return ErrInvalidState
		}
		if err := validateBlueprintChapters(tx, project.ID, blueprint.Volumes); err != nil {
			return err
		}
		if err := validateBlueprintEvidence(tx, project.ID, blueprint.Volumes); err != nil {
			return err
		}
		if err := blueprint.ValidateGenerationReadiness(); err != nil {
			return fmt.Errorf("%w: %w", ErrClaimCoverageIncomplete, err)
		}
		if err := tx.Model(&BlueprintRevision{}).Where("id = ?", revision.ID).Update("status", StatusApproved).Error; err != nil {
			return err
		}
		return tx.Model(&Project{}).Where("id = ?", projectID).Update("approved_blueprint_revision_id", revision.ID).Error
	})
	if err == nil {
		revision.Status = StatusApproved
	}
	return &revision, err
}

func blueprintPayload(bookContractID, styleSheetID uuid.UUID, volumes []sharedtextbook.BlueprintVolume) any {
	return struct {
		BookContractRevision string                           `json:"book_contract_revision"`
		StyleSheetRevision   string                           `json:"style_sheet_revision"`
		Volumes              []sharedtextbook.BlueprintVolume `json:"volumes"`
	}{bookContractID.String(), styleSheetID.String(), volumes}
}

// The workbench renders blueprint entries through project chapter rows. Check
// both writes: old drafts may predate this rule or refer to a deleted chapter.
func validateBlueprintChapters(tx *gorm.DB, projectID uuid.UUID, volumes []sharedtextbook.BlueprintVolume) error {
	ids := make([]uuid.UUID, 0)
	for _, volume := range volumes {
		for _, chapter := range volume.Chapters {
			id, err := uuid.Parse(chapter.ID)
			// Alternate UUID spellings resolve in PostgreSQL but not in UI string joins.
			if err != nil || id == uuid.Nil || id.String() != chapter.ID {
				return ErrInvalidState
			}
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return ErrInvalidState
	}
	var count int64
	if err := tx.Model(&Chapter{}).Where("project_id = ? AND id IN ?", projectID, ids).Count(&count).Error; err != nil {
		return fmt.Errorf("validate blueprint chapter membership: %w", err)
	}
	if count != int64(len(ids)) {
		return ErrInvalidState
	}
	return nil
}

func validateBlueprintEvidence(tx *gorm.DB, projectID uuid.UUID, volumes []sharedtextbook.BlueprintVolume) error {
	evidenceIDs := make(map[string]struct{})
	for _, volume := range volumes {
		for _, chapter := range volume.Chapters {
			for _, evidenceID := range chapter.EvidenceIDs {
				evidenceIDs[evidenceID] = struct{}{}
			}
		}
	}
	if len(evidenceIDs) == 0 {
		return ErrInvalidState
	}
	ids := make([]string, 0, len(evidenceIDs))
	for id := range evidenceIDs {
		ids = append(ids, id)
	}
	var count int64
	err := tx.Table("source_chunks").
		Joins("JOIN source_documents ON source_documents.id = source_chunks.document_id").
		Joins("JOIN source_snapshots ON source_snapshots.id = source_documents.snapshot_id").
		Joins("JOIN textbook_sources ON textbook_sources.id = source_snapshots.source_id").
		Where("textbook_sources.project_id = ? AND textbook_sources.deleted_at IS NULL AND source_chunks.id IN ?", projectID, ids).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return ErrInvalidState
	}
	return nil
}
