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

// Apply appends consent for an exact feedback snapshot and rebuilds the schedule
// in the same transaction. The objective lock also serializes later answers.
func (s *Store) Apply(ctx context.Context, owner, objectiveID, jobID uuid.UUID, input app.ApplyInput) (app.Job, error) {
	var job app.Job
	if input.ID == uuid.Nil || input.ExpectedFeedbackHash == "" || input.PreviousID != nil && *input.PreviousID == uuid.Nil {
		return job, app.ErrConflict
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var objective mastery.Objective
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND workspace_id = ?", objectiveID, owner).Take(&objective).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return app.ErrNotFound
			}
			return err
		}
		row, err := readRow(tx.Clauses(clause.Locking{Strength: "UPDATE"}), owner, objectiveID, jobID)
		if err != nil {
			return err
		}
		job, err = decodeJob(tx, row)
		if err != nil {
			return err
		}
		var existing mastery.AssessmentApplicationRecord
		err = tx.Where("id = ?", input.ID).Take(&existing).Error
		if err == nil {
			saved, decodeErr := existing.Decode()
			if decodeErr != nil || saved.ObjectiveID != objectiveID || saved.JobID != jobID || saved.AppliedBy != owner || saved.FeedbackHash != input.ExpectedFeedbackHash || !reflect.DeepEqual(saved.PreviousID, input.PreviousID) {
				return app.ErrConflict
			}
			// A lost response can be retried after another correction; never reapply
			// the old decision over a newer one. Return the current persisted view.
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if job.Status != "succeeded" || job.EffectiveFeedback == nil || job.EffectiveHash != input.ExpectedFeedbackHash {
			return app.ErrConflict
		}
		if job.AppliedAssessment == nil && input.PreviousID != nil || job.AppliedAssessment != nil && (input.PreviousID == nil || *input.PreviousID != job.AppliedAssessment.ID) {
			return app.ErrConflict
		}
		if job.AppliedAssessment != nil && job.AppliedAssessment.JobID == jobID && job.AppliedAssessment.FeedbackHash == job.EffectiveHash {
			return nil
		}
		historyStore, err := mastery.NewGormStore(tx)
		if err != nil {
			return err
		}
		history, err := historyStore.ListAssessmentApplications(ctx, objectiveID)
		if err != nil {
			return err
		}
		event := mastery.AssessmentApplication{ID: input.ID, ObjectiveID: objectiveID, AttemptID: job.AttemptID, JobID: jobID, PreviousID: input.PreviousID, SequenceNo: len(history) + 1, AppliedBy: owner, AppliedAt: time.Now().UTC(), AlgorithmVersion: mastery.AssessmentAlgorithmVersion, InputHash: job.Preview.InputHash, FeedbackHash: job.EffectiveHash, Input: job.Input, Feedback: *job.EffectiveFeedback}
		records, err := historyStore.ListAttempts(ctx, objectiveID)
		if err != nil {
			return err
		}
		if _, err = mastery.ReplayAssessmentApplications(objective, records, append(history, event)); err != nil {
			return app.ErrConflict
		}
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}
		if err = tx.Create(&mastery.AssessmentApplicationRecord{ID: event.ID, ObjectiveID: objectiveID, AttemptID: event.AttemptID, JobID: jobID, SequenceNo: event.SequenceNo, ApplicationJSON: data}).Error; err != nil {
			return err
		}
		schedule, err := historyStore.ReplayAssessedSchedule(ctx, objective)
		if err != nil {
			return err
		}
		if schedule == nil {
			return app.ErrConflict
		}
		if err = tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "objective_id"}}, DoUpdates: clause.AssignmentColumns([]string{"next_skill", "due_at", "reason", "algorithm_version", "updated_at"})}).Create(schedule).Error; err != nil {
			return err
		}
		return loadAppliedAssessment(tx, &job)
	})
	return job, err
}

func loadAppliedAssessment(tx *gorm.DB, job *app.Job) error {
	var row mastery.AssessmentApplicationRecord
	err := tx.Where("objective_id = ? AND attempt_id = ?", job.ObjectiveID, job.AttemptID).Order("sequence_no DESC").Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	event, err := row.Decode()
	if err != nil || event.AppliedBy != job.WorkspaceID {
		return app.ErrConflict
	}
	job.AppliedAssessment = &event
	var schedule mastery.Schedule
	if err = tx.Where("objective_id = ?", job.ObjectiveID).Take(&schedule).Error; err != nil {
		return err
	}
	job.Schedule = &mastery.DueTask{Skill: mastery.Skill(schedule.NextSkill), DueAt: schedule.DueAt, Reason: schedule.Reason}
	return nil
}
