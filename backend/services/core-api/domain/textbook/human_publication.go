package textbook

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	shared "inkwords-backend/shared/kernel/textbook"
)

func (input CompletePublicationReviewInput) document(at time.Time) shared.HumanPublicationReview {
	score := 0
	if input.Score != nil {
		score = *input.Score
	}
	return shared.HumanPublicationReview{
		ContractVersion: shared.HumanPublicationReviewFormat, ReviewerKind: "human", ID: input.ID.String(), BuildID: input.BuildID.String(),
		ManifestHash: input.ManifestHash, Revision: input.ExpectedRevision + 1, Stage: input.Stage,
		Reviewer: input.Reviewer, Verdict: input.Verdict, Score: score, Scope: input.Scope, Notes: input.Notes,
		EvidenceRefs: input.EvidenceRefs, HardFailures: input.HardFailures, CompletedAt: at,
	}
}

func (input CompletePublicationReviewInput) validate() error {
	if input.ID == uuid.Nil || input.BuildID == uuid.Nil || input.ExpectedRevision < 0 || (input.Score == nil && input.Verdict != "not_assessed") {
		return ErrInvalidState
	}
	return input.document(time.Now().UTC()).Validate()
}

// CompletePublicationReview appends an explicit human statement, retaining the
// original completion notes and every later decision. A caller must acknowledge
// the latest stage revision; stable IDs make an uncertain response safe to retry.
func (r *GormRepository) CompletePublicationReview(ctx context.Context, workspaceID uuid.UUID, input CompletePublicationReviewInput) (*PublicationReviewRow, error) {
	if workspaceID == uuid.Nil || input.validate() != nil {
		return nil, ErrInvalidState
	}
	inputHash := digestJSON(input)
	var result *PublicationReviewRow
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		build, err := loadWorkspaceBookBuild(tx, workspaceID, input.BuildID, true)
		if err != nil {
			return err
		}
		if build.ManifestHash != input.ManifestHash {
			return ErrVersionConflict
		}
		var existing PublicationReviewRow
		err = tx.Where("id = ?", input.ID).First(&existing).Error
		if err == nil {
			if existing.BuildID != build.ID || existing.InputHash != inputHash || existing.ToContract().Validate() != nil {
				return ErrVersionConflict
			}
			result = &existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if build.Status != shared.BookBuildReadyForReview {
			return ErrInvalidState
		}
		var latest int
		if err := tx.Model(&PublicationReviewRow{}).Where("build_id = ? AND stage = ?", build.ID, input.Stage).Select("COALESCE(MAX(revision), 0)").Scan(&latest).Error; err != nil {
			return err
		}
		if latest != input.ExpectedRevision {
			return ErrVersionConflict
		}
		document := input.document(time.Now().UTC())
		refs, _ := json.Marshal(input.EvidenceRefs)
		failures, _ := json.Marshal(input.HardFailures)
		if input.EvidenceRefs == nil {
			refs = []byte("[]")
		}
		if input.HardFailures == nil {
			failures = []byte("[]")
		}
		result = &PublicationReviewRow{
			ID: input.ID, BuildID: build.ID, Stage: input.Stage, Reviewer: input.Reviewer, Notes: input.Notes, CompletedAt: document.CompletedAt,
			ContractVersion: document.ContractVersion, ManifestHash: build.ManifestHash, Revision: document.Revision, ReviewerKind: "human",
			Verdict: input.Verdict, Score: document.Score, Scope: input.Scope, InputHash: inputHash, EvidenceRefs: datatypes.JSON(refs), HardFailures: datatypes.JSON(failures),
		}
		return tx.Create(result).Error
	})
	return result, err
}
