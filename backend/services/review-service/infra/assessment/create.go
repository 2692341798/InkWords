package assessment

import (
	"context"
	"errors"

	"gorm.io/gorm"
	app "inkwords-backend/services/review-service/app/masteryassessment"
	"inkwords-backend/services/review-service/domain/mastery"
)

// Create atomically deduplicates requests and exact successful inputs, validates
// explicit retries, and enforces one running call across the local installation.
func (s *Store) Create(ctx context.Context, job app.Job) (app.Job, bool, error) {
	if err := s.checkRunner(ctx); err != nil {
		return app.Job{}, false, err
	}
	result, created := app.Job{}, false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// This short admission lock serializes only job creation, never model execution.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(734212901)).Error; err != nil {
			return err
		}
		if job.Input.Validate() != nil || mastery.AssessmentInputHash(job.Input) != job.Preview.InputHash || job.Input.AttemptID != job.AttemptID.String() || job.Input.ObjectiveID != job.ObjectiveID.String() {
			return app.ErrConflict
		}
		var count int64
		codeHash := ""
		if job.Input.LearnerArtifact != nil {
			if job.Input.LearnerArtifact.WorkspaceID != job.WorkspaceID.String() {
				return app.ErrConflict
			}
			codeHash = job.Input.LearnerArtifact.SnapshotHash
		}
		if err := tx.Table("mastery_attempts a").Joins("JOIN mastery_objectives o ON o.id=a.objective_id").Where("a.id = ? AND o.id = ? AND o.workspace_id = ? AND o.deleted_at IS NULL AND a.answer = ? AND a.skill = ? AND a.learner_artifact_hash = ?", job.AttemptID, job.ObjectiveID, job.WorkspaceID, job.Input.Answer, string(job.Input.Skill), codeHash).Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			return app.ErrNotFound
		}
		var existing jobRow
		err := tx.Where("workspace_id = ? AND request_id = ?", job.WorkspaceID, job.RequestID).Take(&existing).Error
		if err == nil {
			if existing.ObjectiveID != job.ObjectiveID || existing.AttemptID != job.AttemptID || existing.InputHash != job.Preview.InputHash || existing.RequestHash != job.Preview.RequestHash {
				return app.ErrConflict
			}
			result, err = decodeJob(tx, existing)
			return err
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		err = tx.Where("workspace_id = ? AND objective_id = ? AND attempt_id = ? AND request_hash = ? AND status IN ?", job.WorkspaceID, job.ObjectiveID, job.AttemptID, job.Preview.RequestHash, []string{"succeeded", "running"}).Order("created_at DESC,id DESC").Take(&existing).Error
		if err == nil {
			result, err = decodeJob(tx, existing)
			return err
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if job.RetryOf != nil {
			parent, err := readRow(tx, job.WorkspaceID, job.ObjectiveID, *job.RetryOf)
			if err != nil || parent.AttemptID != job.AttemptID || (parent.Status != "failed" && parent.Status != "cancelled" && parent.Status != "interrupted") {
				return app.ErrConflict
			}
		} else {
			if err := tx.Model(&jobRow{}).Where("workspace_id = ? AND attempt_id = ? AND request_hash = ?", job.WorkspaceID, job.AttemptID, job.Preview.RequestHash).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return app.ErrConflict
			}
		}
		if err := tx.Model(&jobRow{}).Where("status = ?", "running").Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return app.ErrBusy
		}
		input, err := encoded(job.Input)
		if err != nil {
			return err
		}
		preview, err := encoded(job.Preview)
		if err != nil {
			return err
		}
		row := jobRow{ID: job.ID, WorkspaceID: job.WorkspaceID, ObjectiveID: job.ObjectiveID, AttemptID: job.AttemptID, RequestID: job.RequestID, RetryOf: job.RetryOf, Status: "running", InputHash: job.Preview.InputHash, RequestHash: job.Preview.RequestHash, InputJSON: input, PreviewJSON: preview, CreatedAt: job.CreatedAt}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		result, err = decodeJob(tx, row)
		created = err == nil
		return err
	})
	return result, created, err
}
