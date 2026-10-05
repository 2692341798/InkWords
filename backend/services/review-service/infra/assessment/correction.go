package assessment

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	app "inkwords-backend/services/review-service/app/masteryassessment"
	"inkwords-backend/services/review-service/domain/mastery"
)

// Correct verifies ownership and the current feedback hash in the append transaction.
func (s *Store) Correct(ctx context.Context, owner, objective, id uuid.UUID, input app.CorrectionInput) (app.Job, error) {
	var job app.Job
	if input.ID == uuid.Nil {
		return job, app.ErrConflict
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := readRow(tx.Clauses(clause.Locking{Strength: "UPDATE"}), owner, objective, id)
		if err != nil {
			return err
		}
		job, err = decodeJob(tx, row)
		if err != nil {
			return err
		}
		if job.Status != "succeeded" {
			return app.ErrConflict
		}
		var existing correctionRow
		err = tx.Where("id = ?", input.ID).Take(&existing).Error
		if err == nil {
			var saved mastery.AssessmentCorrection
			if existing.JobID != id || json.Unmarshal(existing.CorrectionJSON, &saved) != nil || saved.ReviewerID != owner.String() || saved.PreviousHash != input.PreviousHash || saved.Reason != input.Reason || !reflect.DeepEqual(saved.Changes, input.Changes) || !reflect.DeepEqual(saved.Findings, input.Findings) {
				return app.ErrConflict
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if input.PreviousHash != job.EffectiveHash {
			return app.ErrConflict
		}
		correction := mastery.AssessmentCorrection{ID: input.ID.String(), ReviewerID: owner.String(), CorrectedAt: time.Now().UTC(), Reason: input.Reason, InputHash: job.Preview.InputHash, PreviousHash: input.PreviousHash, Changes: input.Changes, Findings: input.Findings}
		corrections := append(job.Corrections, correction)
		effective, err := mastery.ReplayAssessmentCorrections(job.Input, *job.Result.Feedback, corrections)
		if err != nil {
			return app.ErrConflict
		}
		data, err := encoded(correction)
		if err != nil {
			return err
		}
		if err := tx.Create(&correctionRow{ID: input.ID, JobID: id, SequenceNo: len(corrections), CorrectionJSON: data}).Error; err != nil {
			return err
		}
		job.Corrections, job.EffectiveFeedback, job.EffectiveHash = corrections, &effective, mastery.AssessmentFeedbackHash(effective)
		return nil
	})
	return job, err
}
