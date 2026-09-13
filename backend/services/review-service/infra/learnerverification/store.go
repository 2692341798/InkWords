package learnerverification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	app "inkwords-backend/services/review-service/app/masteryverification"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type Store struct{ db *gorm.DB }

func NewStore(db *gorm.DB) *Store { return &Store{db: db} }

type row struct {
	ID             uuid.UUID  `gorm:"column:id"`
	WorkspaceID    uuid.UUID  `gorm:"column:workspace_id"`
	ObjectiveID    uuid.UUID  `gorm:"column:objective_id"`
	AttemptID      uuid.UUID  `gorm:"column:attempt_id"`
	RequestID      uuid.UUID  `gorm:"column:request_id"`
	RetryOf        *uuid.UUID `gorm:"column:retry_of"`
	Status         string     `gorm:"column:status"`
	InputHash      string     `gorm:"column:input_hash"`
	ClaimTokenHash string     `gorm:"column:claim_token_hash"`
	InputJSON      []byte     `gorm:"column:input_json;type:jsonb"`
	PreviewJSON    []byte     `gorm:"column:preview_json;type:jsonb"`
	ReportJSON     []byte     `gorm:"column:report_json;type:jsonb"`
	ErrorCode      string     `gorm:"column:error_code"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	StartedAt      *time.Time `gorm:"column:started_at"`
	CompletedAt    *time.Time `gorm:"column:completed_at"`
}

func (row) TableName() string { return "mastery_learner_verification_runs" }

func (store *Store) FindRequest(ctx context.Context, owner, request uuid.UUID) (*app.Job, error) {
	var record row
	err := store.db.WithContext(ctx).Where("workspace_id=? AND request_id=?", owner, request).Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	job, err := decodeRow(record)
	return &job, err
}

func (store *Store) Create(ctx context.Context, job app.Job, claimHash string) (app.Job, bool, error) {
	input, err := json.Marshal(job.Input)
	if err != nil {
		return app.Job{}, false, err
	}
	preview, err := json.Marshal(job.Preview)
	if err != nil {
		return app.Job{}, false, err
	}
	record := row{ID: job.ID, WorkspaceID: job.WorkspaceID, ObjectiveID: job.ObjectiveID, AttemptID: job.AttemptID, RequestID: job.RequestID, RetryOf: job.RetryOf, Status: job.Status, InputHash: job.Preview.InputHash, ClaimTokenHash: claimHash, InputJSON: input, PreviewJSON: preview, CreatedAt: job.CreatedAt}
	if err := store.db.WithContext(ctx).Create(&record).Error; err != nil {
		existing, findErr := store.FindRequest(ctx, job.WorkspaceID, job.RequestID)
		if findErr == nil && existing != nil {
			return *existing, false, nil
		}
		return app.Job{}, false, app.ErrBusy
	}
	return job, true, nil
}

func (store *Store) Resolve(ctx context.Context, reference sharedtextbook.LearnerVerificationReference, claimHash string) (sharedtextbook.LearnerVerificationInput, error) {
	var input sharedtextbook.LearnerVerificationInput
	err := store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record row
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND workspace_id=? AND objective_id=? AND attempt_id=? AND input_hash=? AND claim_token_hash=? AND status='queued'", reference.RunID, reference.WorkspaceID, reference.ObjectiveID, reference.AttemptID, reference.InputHash, claimHash).Take(&record)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return app.ErrNotFound
		}
		if result.Error != nil {
			return result.Error
		}
		if err := json.Unmarshal(record.InputJSON, &input); err != nil || input.ValidateFor(reference) != nil {
			return app.ErrConflict
		}
		now := time.Now().UTC()
		if err := tx.Model(&row{}).Where("id=? AND status='queued'", record.ID).Updates(map[string]any{"status": "running", "started_at": now}).Error; err != nil {
			return err
		}
		return nil
	})
	return input, err
}

func (store *Store) Finish(ctx context.Context, id uuid.UUID, report sharedtextbook.LearnerVerificationReport, status, code string) error {
	encoded, err := json.Marshal(report)
	if err != nil {
		return err
	}
	return store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record row
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=?", id).Take(&record).Error; err != nil {
			return err
		}
		if record.Status == "cancelled" || record.Status == "interrupted" {
			return nil
		}
		if record.Status != "queued" && record.Status != "running" {
			return nil
		}
		var input sharedtextbook.LearnerVerificationInput
		if err := json.Unmarshal(record.InputJSON, &input); err != nil {
			return app.ErrConflict
		}
		reference := sharedtextbook.LearnerVerificationReference{
			Format: sharedtextbook.LearnerVerificationReferenceFormat, RunID: record.ID.String(),
			WorkspaceID: record.WorkspaceID.String(), ObjectiveID: record.ObjectiveID.String(), AttemptID: record.AttemptID.String(),
			InputHash: record.InputHash, ClaimToken: strings.Repeat("A", 43),
		}
		if status != string(report.Status) || report.ValidateFor(reference, input.Plan) != nil {
			return app.ErrConflict
		}
		if report.ExecutionStarted && record.Status != "running" {
			return app.ErrConflict
		}
		now := time.Now().UTC()
		return tx.Model(&row{}).Where("id=?", id).Updates(map[string]any{"status": status, "report_json": encoded, "error_code": code, "completed_at": now}).Error
	})
}

func (store *Store) Read(ctx context.Context, owner, objective, id uuid.UUID) (app.Job, error) {
	var record row
	err := store.db.WithContext(ctx).Where("id=? AND workspace_id=? AND objective_id=?", id, owner, objective).Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return app.Job{}, app.ErrNotFound
	}
	if err != nil {
		return app.Job{}, err
	}
	return decodeRow(record)
}

func (store *Store) Latest(ctx context.Context, owner, objective, attempt uuid.UUID) (*app.Job, error) {
	var record row
	err := store.db.WithContext(ctx).Where("workspace_id=? AND objective_id=? AND attempt_id=?", owner, objective, attempt).Order("created_at DESC,id DESC").Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	job, err := decodeRow(record)
	return &job, err
}

func (store *Store) Cancel(ctx context.Context, owner, objective, id uuid.UUID) (app.Job, error) {
	now := time.Now().UTC()
	result := store.db.WithContext(ctx).Model(&row{}).Where("id=? AND workspace_id=? AND objective_id=? AND status IN ('queued','running')", id, owner, objective).Updates(map[string]any{"status": "cancelled", "completed_at": now, "error_code": "cancelled"})
	if result.Error != nil {
		return app.Job{}, result.Error
	}
	return store.Read(ctx, owner, objective, id)
}

func (store *Store) InterruptRunning(ctx context.Context) error {
	now := time.Now().UTC()
	return store.db.WithContext(ctx).Model(&row{}).Where("status IN ('queued','running')").Updates(map[string]any{"status": "interrupted", "completed_at": now, "error_code": "process_restarted"}).Error
}

func (store *Store) Interrupt(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	return store.db.WithContext(ctx).Model(&row{}).Where("id=? AND status IN ('queued','running')", id).Updates(map[string]any{"status": "interrupted", "completed_at": now, "error_code": "execution_interrupted"}).Error
}

func decodeRow(record row) (app.Job, error) {
	job := app.Job{ID: record.ID, WorkspaceID: record.WorkspaceID, ObjectiveID: record.ObjectiveID, AttemptID: record.AttemptID, RequestID: record.RequestID, RetryOf: record.RetryOf, Status: record.Status, ErrorCode: record.ErrorCode, CreatedAt: record.CreatedAt, StartedAt: record.StartedAt, CompletedAt: record.CompletedAt}
	if err := json.Unmarshal(record.InputJSON, &job.Input); err != nil {
		return app.Job{}, fmt.Errorf("decode learner verification input: %w", err)
	}
	if err := json.Unmarshal(record.PreviewJSON, &job.Preview); err != nil {
		return app.Job{}, fmt.Errorf("decode learner verification preview: %w", err)
	}
	if len(record.ReportJSON) > 0 {
		var report sharedtextbook.LearnerVerificationReport
		if err := json.Unmarshal(record.ReportJSON, &report); err != nil {
			return app.Job{}, fmt.Errorf("decode learner verification report: %w", err)
		}
		job.Report = &report
	}
	return job, nil
}
