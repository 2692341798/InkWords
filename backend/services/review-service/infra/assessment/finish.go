package assessment

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	app "inkwords-backend/services/review-service/app/masteryassessment"
)

// Finish saves a terminal response without promoting a cancelled or abandoned job.
func (s *Store) Finish(ctx context.Context, id uuid.UUID, result app.Result, status, code string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row jobRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).Take(&row).Error; err != nil {
			return err
		}
		if row.Status != "running" && row.Status != "cancelled" {
			return nil
		}
		if row.Status == "cancelled" && len(row.ResultJSON) > 0 && string(row.ResultJSON) != "null" {
			return nil
		}
		job, err := decodeJob(tx, row)
		if err != nil {
			return err
		}
		if status != "succeeded" && status != "failed" && status != "cancelled" {
			return app.ErrConflict
		}
		if result.ProviderCalls == 0 && result.InputHash == "" && result.Feedback == nil {
			result.InputHash, result.RequestHash, result.Provider, result.Model = row.InputHash, row.RequestHash, job.Preview.Provider, job.Preview.Model
			result.Contract, result.Origin = job.Input.ContractVersion(), "automated"
		}
		if result.InputHash != row.InputHash || result.RequestHash != row.RequestHash || result.Provider != job.Preview.Provider || result.Model != job.Preview.Model || result.Contract != job.Input.ContractVersion() || result.Origin != "automated" || result.ProviderCalls < 0 || result.ProviderCalls > 1 {
			return app.ErrConflict
		}
		if row.Status == "cancelled" {
			status, code = "cancelled", "cancelled"
		}
		if status == "succeeded" {
			if result.ValidateFeedback(job.Input) != nil {
				return app.ErrConflict
			}
		} else {
			result.Feedback = nil
			result.Decisions = nil
			result.Judgments = nil
		}
		data, err := encoded(result)
		if err != nil {
			return err
		}
		return tx.Model(&jobRow{}).Where("id = ?", id).Updates(map[string]any{"status": status, "result_json": data, "error_code": code, "completed_at": time.Now().UTC()}).Error
	})
}

// Cancel is idempotent and leaves any already completed result intact.
func (s *Store) Cancel(ctx context.Context, owner, objective, id uuid.UUID) (app.Job, error) {
	var job app.Job
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := readRow(tx.Clauses(clause.Locking{Strength: "UPDATE"}), owner, objective, id)
		if err != nil {
			return err
		}
		if row.Status == "running" {
			now := time.Now().UTC()
			if err := tx.Model(&row).Updates(map[string]any{"status": "cancelled", "error_code": "cancelled", "completed_at": now}).Error; err != nil {
				return err
			}
			row.Status, row.ErrorCode, row.CompletedAt = "cancelled", "cancelled", &now
		}
		job, err = decodeJob(tx, row)
		return err
	})
	return job, err
}
