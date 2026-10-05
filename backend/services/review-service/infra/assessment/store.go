package assessment

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	app "inkwords-backend/services/review-service/app/masteryassessment"
	"inkwords-backend/services/review-service/domain/mastery"
)

type jobRow struct {
	ID          uuid.UUID `gorm:"primaryKey"`
	WorkspaceID uuid.UUID
	ObjectiveID uuid.UUID
	AttemptID   uuid.UUID
	RequestID   uuid.UUID
	RetryOf     *uuid.UUID
	Status      string
	InputHash   string
	RequestHash string
	InputJSON   datatypes.JSON
	PreviewJSON datatypes.JSON
	ResultJSON  datatypes.JSON
	ErrorCode   string
	CreatedAt   time.Time
	CompletedAt *time.Time
}

func (jobRow) TableName() string { return "mastery_assessment_jobs" }

type correctionRow struct {
	ID             uuid.UUID `gorm:"primaryKey"`
	JobID          uuid.UUID
	SequenceNo     int
	CorrectionJSON datatypes.JSON
}

func (correctionRow) TableName() string { return "mastery_assessment_corrections" }

// Store owns PostgreSQL persistence for local assessment execution and corrections.
type Store struct {
	db       *gorm.DB
	leaderMu sync.Mutex
	leader   *sql.Conn
}

// NewStore uses the versioned review schema, never AutoMigrate.
func NewStore(db *gorm.DB) *Store {
	// Job/application JSON and query bindings include private answers and files.
	// SQL failures must not echo their contents; callers return stable errors.
	return &Store{db: db.Session(&gorm.Session{Logger: db.Logger.LogMode(logger.Silent)})}
}

func decodeJob(tx *gorm.DB, row jobRow) (app.Job, error) {
	job := app.Job{ID: row.ID, WorkspaceID: row.WorkspaceID, ObjectiveID: row.ObjectiveID, AttemptID: row.AttemptID, RequestID: row.RequestID, RetryOf: row.RetryOf, Status: row.Status, ErrorCode: row.ErrorCode, CreatedAt: row.CreatedAt, CompletedAt: row.CompletedAt, Corrections: []mastery.AssessmentCorrection{}}
	if json.Unmarshal(row.InputJSON, &job.Input) != nil || json.Unmarshal(row.PreviewJSON, &job.Preview) != nil || job.Input.Validate() != nil || mastery.AssessmentInputHash(job.Input) != row.InputHash || job.Preview.InputHash != row.InputHash || job.Preview.RequestHash != row.RequestHash {
		return job, app.ErrConflict
	}
	if len(row.ResultJSON) > 0 && string(row.ResultJSON) != "null" {
		job.Result = &app.Result{}
		if json.Unmarshal(row.ResultJSON, job.Result) != nil {
			return job, app.ErrConflict
		}
	}
	var rows []correctionRow
	if err := tx.Where("job_id = ?", job.ID).Order("sequence_no ASC").Find(&rows).Error; err != nil {
		return job, err
	}
	for index, row := range rows {
		var correction mastery.AssessmentCorrection
		if row.SequenceNo != index+1 || json.Unmarshal(row.CorrectionJSON, &correction) != nil {
			return job, app.ErrConflict
		}
		job.Corrections = append(job.Corrections, correction)
	}
	if job.Status == "succeeded" {
		if job.Result == nil || job.Result.Feedback == nil {
			return job, app.ErrConflict
		}
		if job.Result.ValidateFeedback(job.Input) != nil {
			return job, app.ErrConflict
		}
		effective, err := mastery.ReplayAssessmentCorrections(job.Input, *job.Result.Feedback, job.Corrections)
		if err != nil {
			return job, app.ErrConflict
		}
		job.EffectiveFeedback = &effective
		job.EffectiveHash = mastery.AssessmentFeedbackHash(effective)
	}
	if err := loadAppliedAssessment(tx, &job); err != nil {
		return job, err
	}
	return job, nil
}

func readRow(tx *gorm.DB, owner, objective, id uuid.UUID) (jobRow, error) {
	var row jobRow
	err := tx.Where("id = ? AND workspace_id = ? AND objective_id = ?", id, owner, objective).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = app.ErrNotFound
	}
	return row, err
}

// Read exposes only the workspace-owned job and its append-only effective view.
func (s *Store) Read(ctx context.Context, owner, objective, id uuid.UUID) (app.Job, error) {
	var result app.Job
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := readRow(tx, owner, objective, id)
		if err != nil {
			return err
		}
		result, err = decodeJob(tx, row)
		return err
	})
	return result, err
}

// FindRequest deduplicates a retried HTTP request before any source/model work.
func (s *Store) FindRequest(ctx context.Context, owner, request uuid.UUID) (*app.Job, error) {
	var row jobRow
	err := s.db.WithContext(ctx).Where("workspace_id = ? AND request_id = ?", owner, request).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	job, err := s.Read(ctx, owner, row.ObjectiveID, row.ID)
	return &job, err
}

// Latest reads past work for this saved attempt, without starting a call.
func (s *Store) Latest(ctx context.Context, owner, objective, attempt uuid.UUID) (*app.Job, error) {
	var row jobRow
	err := s.db.WithContext(ctx).Where("workspace_id = ? AND objective_id = ? AND attempt_id = ?", owner, objective, attempt).Order("created_at DESC,id DESC").Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	job, err := s.Read(ctx, owner, objective, row.ID)
	return &job, err
}

// InterruptRunning records the uncertainty of calls abandoned by a stopped process.
func (s *Store) InterruptRunning(ctx context.Context) error {
	if err := s.acquireRunner(ctx); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&jobRow{}).Where("status = ?", "running").Updates(map[string]any{"status": "interrupted", "error_code": "execution_unknown", "completed_at": time.Now().UTC()}).Error
}

// Interrupt closes an orphaned local execution without pretending to know its cost.
func (s *Store) Interrupt(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&jobRow{}).Where("id = ? AND status = ?", id, "running").Updates(map[string]any{"status": "interrupted", "error_code": "execution_unknown", "completed_at": time.Now().UTC()}).Error
}

func encoded(value any) (datatypes.JSON, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode assessment record: %w", err)
	}
	return b, nil
}

var _ app.Store = (*Store)(nil)
