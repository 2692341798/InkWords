package textbook

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// AppendRightsAmendmentInput supplies a client-generated retry ID and the exact
// predecessor read by the caller; the server derives all work identity fields.
type AppendRightsAmendmentInput struct {
	AssetFontReview     *sharedtextbook.AssetFontReview `json:"asset_font_review,omitempty"`
	ID                  uuid.UUID                       `json:"id"`
	BuildID             uuid.UUID                       `json:"-"`
	BaseItemID          uuid.UUID                       `json:"base_item_id"`
	PreviousAmendmentID *uuid.UUID                      `json:"previous_amendment_id"`
	ManifestHash        string                          `json:"manifest_hash"`
	ReviewerKind        string                          `json:"reviewer_kind"`
	Reviewer            string                          `json:"reviewer"`
	DelegationNote      string                          `json:"delegation_note"`
	Reason              string                          `json:"reason"`
	EvidenceRefs        []string                        `json:"evidence_refs"`
	RightsBasis         string                          `json:"rights_basis"`
	AllowedUse          string                          `json:"allowed_use"`
	Attribution         string                          `json:"attribution"`
	PublicationStatus   sharedtextbook.RightsStatus     `json:"publication_status"`
}

// RightsAmendmentRow stores append-only decisions separately from their originals.
type RightsAmendmentRow struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey"`
	BuildID             uuid.UUID
	BaseItemID          uuid.UUID
	PreviousAmendmentID *uuid.UUID
	Revision            int
	ManifestHash        string
	InputHash           string
	DocumentJSON        datatypes.JSON
	CompletedAt         time.Time
}

func (RightsAmendmentRow) TableName() string { return "textbook_rights_amendments" }

func (row RightsAmendmentRow) document() (*sharedtextbook.RightsAmendment, error) {
	var a sharedtextbook.RightsAmendment
	previous := ""
	if row.PreviousAmendmentID != nil {
		previous = row.PreviousAmendmentID.String()
	}
	if json.Unmarshal(row.DocumentJSON, &a) != nil || a.Validate() != nil || a.ID != row.ID.String() || a.BuildID != row.BuildID.String() || a.BaseItemID != row.BaseItemID.String() || a.ManifestHash != row.ManifestHash || a.Revision != row.Revision || a.PreviousAmendmentID != previous || !a.CompletedAt.Equal(row.CompletedAt) {
		return nil, ErrInvalidState
	}
	return &a, nil
}

// AppendRightsAmendment never mutates the original row or manuscript.
func (s *Service) AppendRightsAmendment(ctx context.Context, workspaceID uuid.UUID, input AppendRightsAmendmentInput) (*sharedtextbook.RightsAmendment, error) {
	if workspaceID == uuid.Nil || input.BuildID == uuid.Nil || input.ID == uuid.Nil || input.BaseItemID == uuid.Nil || (input.PreviousAmendmentID != nil && *input.PreviousAmendmentID == uuid.Nil) {
		return nil, ErrInvalidState
	}
	return s.repository.AppendRightsAmendment(ctx, workspaceID, input)
}

func (r *GormRepository) AppendRightsAmendment(ctx context.Context, workspaceID uuid.UUID, input AppendRightsAmendmentInput) (*sharedtextbook.RightsAmendment, error) {
	encoded, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	inputHash := fmt.Sprintf("sha256:%x", sha256.Sum256(encoded))
	var result *sharedtextbook.RightsAmendment
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		build, err := loadWorkspaceBookBuild(tx, workspaceID, input.BuildID, true)
		if err != nil {
			return err
		}
		if build.ManifestHash != input.ManifestHash {
			return ErrVersionConflict
		}
		var prior RightsAmendmentRow
		err = tx.Where("id = ?", input.ID).First(&prior).Error
		if err == nil {
			if prior.BuildID != build.ID || prior.InputHash != inputHash {
				return ErrVersionConflict
			}
			result, err = prior.document()
			return err
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if build.Status != sharedtextbook.BookBuildReadyForReview {
			return ErrInvalidState
		}
		if input.AssetFontReview != nil && !assetFontReviewMatchesBuild(*input.AssetFontReview, *build) {
			return ErrInvalidState
		}
		var original RightsItemRow
		if err = tx.Where("id = ? AND build_id = ? AND project_id = ?", input.BaseItemID, build.ID, build.ProjectID).First(&original).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		history, err := loadRightsAmendments(tx, build.ID)
		if err != nil {
			return err
		}
		previous, revision := "", 1
		for _, a := range history {
			if a.BaseItemID == original.ID.String() && a.Revision >= revision {
				previous, revision = a.ID, a.Revision+1
			}
		}
		expected := ""
		if input.PreviousAmendmentID != nil {
			expected = input.PreviousAmendmentID.String()
		}
		if expected != previous {
			return ErrVersionConflict
		}
		effective := original.ToContract()
		effective.RightsBasis, effective.AllowedUse, effective.Attribution, effective.PublicationStatus = input.RightsBasis, input.AllowedUse, input.Attribution, input.PublicationStatus
		document := sharedtextbook.RightsAmendment{ContractVersion: sharedtextbook.RightsAmendmentFormat, ID: input.ID.String(), BuildID: build.ID.String(), ManifestHash: build.ManifestHash, BaseItemID: original.ID.String(), PreviousAmendmentID: previous, Revision: revision, ReviewerKind: input.ReviewerKind, Reviewer: input.Reviewer, DelegationNote: input.DelegationNote, Reason: input.Reason, EvidenceRefs: input.EvidenceRefs, EffectiveItem: effective, CompletedAt: time.Now().UTC().Truncate(time.Microsecond)}
		document.AssetFontReview = input.AssetFontReview
		var originals []RightsItemRow
		if err := tx.Where("build_id = ?", build.ID).Order("created_at ASC, id ASC").Find(&originals).Error; err != nil {
			return err
		}
		items := make([]sharedtextbook.RightsItem, 0, len(originals))
		for _, item := range originals {
			items = append(items, item.ToContract())
		}
		if _, err := sharedtextbook.ResolveRightsLedger(build.ID.String(), build.ManifestHash, items, append(history, document)); err != nil {
			return ErrInvalidState
		}
		raw, err := json.Marshal(document)
		if err != nil {
			return err
		}
		row := RightsAmendmentRow{ID: input.ID, BuildID: build.ID, BaseItemID: original.ID, PreviousAmendmentID: input.PreviousAmendmentID, Revision: revision, ManifestHash: build.ManifestHash, InputHash: inputHash, DocumentJSON: raw, CompletedAt: document.CompletedAt}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		result = &document
		return nil
	})
	return result, err
}

func loadRightsAmendments(db *gorm.DB, buildID uuid.UUID) ([]sharedtextbook.RightsAmendment, error) {
	var rows []RightsAmendmentRow
	if err := db.Where("build_id = ?", buildID).Order("base_item_id ASC, revision ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	history := make([]sharedtextbook.RightsAmendment, 0, len(rows))
	for _, row := range rows {
		a, err := row.document()
		if err != nil {
			return nil, err
		}
		history = append(history, *a)
	}
	return history, nil
}
