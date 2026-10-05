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

// RecordDelegatedPublicationReviewInput binds an append to the frozen manifest
// and the stage revision actually read by the caller. ID makes retries safe.
type RecordDelegatedPublicationReviewInput struct {
	ID               uuid.UUID                             `json:"id"`
	BuildID          uuid.UUID                             `json:"-"`
	ManifestHash     string                                `json:"manifest_hash"`
	ExpectedRevision int                                   `json:"expected_revision"`
	Stage            sharedtextbook.PublicationReviewStage `json:"stage"`
	Reviewer         string                                `json:"reviewer"`
	DelegationNote   string                                `json:"delegation_note"`
	Verdict          string                                `json:"verdict"`
	Score            int                                   `json:"score"`
	Scope            string                                `json:"scope"`
	Notes            string                                `json:"notes"`
	EvidenceRefs     []string                              `json:"evidence_refs"`
	HardFailures     []string                              `json:"hard_failures"`
}

func (input RecordDelegatedPublicationReviewInput) document(at time.Time) sharedtextbook.DelegatedPublicationReview {
	return sharedtextbook.DelegatedPublicationReview{
		ContractVersion: sharedtextbook.DelegatedPublicationReviewFormat, ID: input.ID.String(), BuildID: input.BuildID.String(),
		ManifestHash: input.ManifestHash, Stage: input.Stage, Revision: input.ExpectedRevision + 1,
		ReviewerKind: "delegated_ai", Reviewer: input.Reviewer, DelegationNote: input.DelegationNote,
		Verdict: input.Verdict, Score: input.Score, Scope: input.Scope, Notes: input.Notes,
		EvidenceRefs: input.EvidenceRefs, HardFailures: input.HardFailures, CompletedAt: at,
	}
}

// DelegatedPublicationReviewRow is append-only; the old human evidence table
// remains reserved for actual people. DocumentJSON stores the versioned rubric.
type DelegatedPublicationReviewRow struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	BuildID      uuid.UUID `gorm:"type:uuid;not null"`
	Stage        sharedtextbook.PublicationReviewStage
	Revision     int
	Verdict      string
	ManifestHash string
	InputHash    string
	DocumentJSON datatypes.JSON
	CompletedAt  time.Time
}

func (DelegatedPublicationReviewRow) TableName() string {
	return "textbook_delegated_publication_reviews"
}

func (row DelegatedPublicationReviewRow) document() (*sharedtextbook.DelegatedPublicationReview, error) {
	var document sharedtextbook.DelegatedPublicationReview
	if err := json.Unmarshal(row.DocumentJSON, &document); err != nil {
		return nil, fmt.Errorf("decode delegated review: %w", err)
	}
	if document.Validate() != nil || document.ID != row.ID.String() || document.BuildID != row.BuildID.String() || document.ManifestHash != row.ManifestHash || document.Revision != row.Revision || document.Stage != row.Stage || document.Verdict != row.Verdict {
		return nil, ErrInvalidState
	}
	return &document, nil
}

// RecordDelegatedPublicationReview records the user's explicit delegation,
// including failed or unassessed outcomes, without fabricating human evidence.
func (s *Service) RecordDelegatedPublicationReview(ctx context.Context, workspaceID uuid.UUID, input RecordDelegatedPublicationReviewInput) (*sharedtextbook.DelegatedPublicationReview, error) {
	if workspaceID == uuid.Nil || input.BuildID == uuid.Nil || input.ID == uuid.Nil || input.ExpectedRevision < 0 || input.document(time.Now().UTC()).Validate() != nil {
		return nil, ErrInvalidState
	}
	return s.repository.RecordDelegatedPublicationReview(ctx, workspaceID, input)
}

func (r *GormRepository) RecordDelegatedPublicationReview(ctx context.Context, workspaceID uuid.UUID, input RecordDelegatedPublicationReviewInput) (*sharedtextbook.DelegatedPublicationReview, error) {
	document := input.document(time.Now().UTC())
	if workspaceID == uuid.Nil || input.ID == uuid.Nil || input.BuildID == uuid.Nil || document.Validate() != nil {
		return nil, ErrInvalidState
	}
	encodedInput, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	inputHash := fmt.Sprintf("sha256:%x", sha256.Sum256(encodedInput))
	var result *sharedtextbook.DelegatedPublicationReview
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		build, err := loadWorkspaceBookBuild(tx, workspaceID, input.BuildID, true)
		if err != nil {
			return err
		}
		if build.ManifestHash != input.ManifestHash {
			return ErrVersionConflict
		}
		var prior DelegatedPublicationReviewRow
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
		var latest int
		if err := tx.Model(&DelegatedPublicationReviewRow{}).Where("build_id = ? AND stage = ?", build.ID, input.Stage).Select("COALESCE(MAX(revision), 0)").Scan(&latest).Error; err != nil {
			return err
		}
		if latest != input.ExpectedRevision {
			return ErrVersionConflict
		}
		raw, err := json.Marshal(document)
		if err != nil {
			return err
		}
		row := DelegatedPublicationReviewRow{ID: input.ID, BuildID: build.ID, Stage: document.Stage, Revision: document.Revision, Verdict: document.Verdict, ManifestHash: build.ManifestHash, InputHash: inputHash, DocumentJSON: raw, CompletedAt: document.CompletedAt}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		result = &document
		return nil
	})
	return result, err
}

func loadDelegatedPublicationReviews(db *gorm.DB, buildID uuid.UUID) ([]sharedtextbook.DelegatedPublicationReview, error) {
	var rows []DelegatedPublicationReviewRow
	if err := db.Where("build_id = ?", buildID).Order("stage ASC, revision ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list delegated reviews: %w", err)
	}
	reviews := make([]sharedtextbook.DelegatedPublicationReview, 0, len(rows))
	for _, row := range rows {
		document, err := row.document()
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, *document)
	}
	return reviews, nil
}
