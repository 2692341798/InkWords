package task

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	shared "inkwords-backend/shared/kernel/textbook"
	"time"
)

var ErrVerificationAttemptConflict = errors.New("verification previous attempt or request identity changed")
var ErrVerificationExecutionActive = errors.New("previous verification execution has not stopped")
var ErrVerificationTaskTerminal = errors.New("verification task is already terminal")

// VerificationSeries serializes explicit attempts without locking manuscript rows.
type VerificationSeries struct {
	ArtifactID  uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null"`
}

func (VerificationSeries) TableName() string { return "textbook_verification_series" }

// CreateVerificationAttemptInput contains frozen identities only. RequestID is
// also the new task's identity, so an uncertain response is safe to retry.
type CreateVerificationAttemptInput struct {
	RequestID              uuid.UUID
	WorkspaceID            uuid.UUID
	ExpectedPreviousTaskID *uuid.UUID
	Payload                shared.ArtifactVerificationRequest
}

// ListVerificationAttempts returns immutable task identities newest first.
func (r *GormRepository) ListVerificationAttempts(ctx context.Context, workspaceID, artifactID uuid.UUID) ([]JobTask, error) {
	var rows []JobTask
	err := r.db.WithContext(ctx).Where("workspace_id = ? AND verification_artifact_id = ?", workspaceID, artifactID).Order("verification_attempt DESC").Find(&rows).Error
	return rows, err
}

// CreateVerificationAttempt serializes admission per artifact and requires both
// a matching predecessor and an observed executor exit before advancing it.
func (r *GormRepository) CreateVerificationAttempt(ctx context.Context, input CreateVerificationAttemptInput) (JobTask, bool, error) {
	var result JobTask
	created := false
	if input.RequestID == uuid.Nil || input.WorkspaceID == uuid.Nil || input.Payload.Validate() != nil || (input.ExpectedPreviousTaskID != nil && *input.ExpectedPreviousTaskID == uuid.Nil) {
		return result, false, ErrVerificationAttemptConflict
	}
	artifactID, _ := uuid.Parse(input.Payload.ArtifactID)
	raw, _ := json.Marshal(input.Payload)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		series := VerificationSeries{ArtifactID: artifactID, WorkspaceID: input.WorkspaceID}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&series).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("artifact_id = ?", artifactID).First(&series).Error; err != nil {
			return err
		}
		if series.WorkspaceID != input.WorkspaceID {
			return ErrTaskAccessDenied
		}
		var existing JobTask
		err := tx.First(&existing, "id = ?", input.RequestID).Error
		if err == nil {
			var payload shared.ArtifactVerificationRequest
			if existing.WorkspaceID == nil || *existing.WorkspaceID != input.WorkspaceID || existing.VerificationArtifactID == nil || *existing.VerificationArtifactID != artifactID || json.Unmarshal(existing.PayloadJSON, &payload) != nil || payload != input.Payload || !sameVerificationPredecessor(existing.PreviousVerificationTaskID, input.ExpectedPreviousTaskID) {
				return ErrVerificationAttemptConflict
			}
			result = existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var latest JobTask
		err = tx.Where("verification_artifact_id = ?", artifactID).Order("verification_attempt DESC").First(&latest).Error
		attempt := 1
		if err == nil {
			if input.ExpectedPreviousTaskID == nil || *input.ExpectedPreviousTaskID != latest.ID {
				return ErrVerificationAttemptConflict
			}
			var previousPayload shared.ArtifactVerificationRequest
			if json.Unmarshal(latest.PayloadJSON, &previousPayload) != nil || previousPayload != input.Payload {
				return ErrVerificationAttemptConflict
			}
			if !isTerminalStatus(latest.Status) || (latest.VerificationWorkerToken != nil && latest.VerificationWorkerReleasedAt == nil) {
				return ErrVerificationExecutionActive
			}
			attempt = latest.VerificationAttempt + 1
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		} else if input.ExpectedPreviousTaskID != nil {
			return ErrVerificationAttemptConflict
		}
		result = JobTask{ID: input.RequestID, TaskType: taskTypeVerification, TaskSubtype: shared.TextbookTeachingArtifactVerifyTaskSubtype, WorkspaceID: &input.WorkspaceID, Status: JobTaskStatusQueued, IdempotencyKey: "textbook-verification-attempt:" + input.RequestID.String(), PayloadJSON: datatypes.JSON(raw), ResultJSON: datatypes.JSON([]byte(`{}`)), VerificationArtifactID: &artifactID, VerificationAttempt: attempt, PreviousVerificationTaskID: cloneUUID(input.ExpectedPreviousTaskID)}
		if err := tx.Create(&result).Error; err != nil {
			return err
		}
		created = true
		return nil
	})
	return result, created, err
}

func sameVerificationPredecessor(a, b *uuid.UUID) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

// ClaimVerificationWorker atomically prevents a cancelled task or a duplicate
// delivery from acquiring a second execution. No caller supplies the token.
func (r *GormRepository) ClaimVerificationWorker(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	token := uuid.New()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row JobTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", id).Error; err != nil {
			return err
		}
		if isTerminalStatus(row.Status) {
			return ErrVerificationTaskTerminal
		}
		if row.TaskSubtype != shared.TextbookTeachingArtifactVerifyTaskSubtype || row.VerificationAttempt < 1 || row.VerificationArtifactID == nil || row.VerificationWorkerToken != nil || (row.Status != JobTaskStatusQueued && row.Status != JobTaskStatusPending) {
			return ErrVerificationExecutionActive
		}
		return tx.Model(&JobTask{}).Where("id = ?", id).Updates(map[string]any{"status": JobTaskStatusRunning, "started_at": time.Now().UTC(), "updated_at": time.Now().UTC(), "verification_worker_token": token, "verification_worker_released_at": nil, "verification_release_kind": ""}).Error
	})
	return token, err
}

// ReleaseVerificationWorker records an observed executor exit for its own claim.
func (r *GormRepository) ReleaseVerificationWorker(ctx context.Context, id, token uuid.UUID) error {
	if token == uuid.Nil {
		return ErrVerificationAttemptConflict
	}
	result := r.db.WithContext(ctx).Model(&JobTask{}).Where("id = ? AND verification_worker_token = ? AND verification_worker_released_at IS NULL", id, token).Updates(map[string]any{"verification_worker_released_at": time.Now().UTC(), "verification_release_kind": "worker"})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}
	var row JobTask
	if err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		return err
	}
	if row.VerificationWorkerToken != nil && *row.VerificationWorkerToken == token && row.VerificationWorkerReleasedAt != nil {
		return nil
	}
	return ErrVerificationAttemptConflict
}

type verificationAttemptRepository interface {
	CreateVerificationAttempt(context.Context, CreateVerificationAttemptInput) (JobTask, bool, error)
	ListVerificationAttempts(context.Context, uuid.UUID, uuid.UUID) ([]JobTask, error)
}

// ListVerificationAttempts observes history without publishing work.
func (s *Service) ListVerificationAttempts(ctx context.Context, workspace, artifact uuid.UUID) ([]JobTask, error) {
	if workspace == uuid.Nil || artifact == uuid.Nil {
		return nil, ErrTaskAccessDenied
	}
	repo, ok := s.repo.(verificationAttemptRepository)
	if !ok {
		return nil, ErrVerificationAttemptConflict
	}
	return repo.ListVerificationAttempts(ctx, workspace, artifact)
}

// CreateVerificationAttempt admits explicit work and permits exact request
// retries to recover an unclaimed task whose broker response was lost.
func (s *Service) CreateVerificationAttempt(ctx context.Context, input CreateVerificationAttemptInput) (JobTask, error) {
	repo, ok := s.repo.(verificationAttemptRepository)
	if !ok {
		return JobTask{}, ErrVerificationAttemptConflict
	}
	if _, ok := s.publisher.(textbookVerificationPublisher); !ok {
		return JobTask{}, errors.New("verification publisher unavailable")
	}
	task, created, err := repo.CreateVerificationAttempt(ctx, input)
	if err != nil {
		return JobTask{}, err
	}
	// Republishing an unclaimed queued request recovers a crash between DB insert
	// and broker delivery. The worker's claim makes duplicate deliveries harmless.
	if created || (task.Status == JobTaskStatusQueued && task.VerificationWorkerToken == nil) {
		if err := s.publishTextbookVerification(ctx, task, task.PayloadJSON); err != nil {
			return task, err
		}
	}
	return task, nil
}
