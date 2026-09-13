package mastery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type GormStore struct{ db *gorm.DB }

// ErrMasteryHistoryChanged requires recalculation against newly committed evidence.
var ErrMasteryHistoryChanged = errors.New("mastery history changed; retry with current evidence")

func NewGormStore(db *gorm.DB) (*GormStore, error) {
	if db == nil {
		return nil, fmt.Errorf("mastery database is required")
	}
	return &GormStore{db: db}, nil
}

func (store *GormStore) CreateObjectiveAndInitialSchedule(ctx context.Context, objective *Objective, schedule *Schedule) error {
	return store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(objective).Error; err != nil {
			return err
		}
		schedule.ObjectiveID = objective.ID
		return tx.Create(schedule).Error
	})
}

func (store *GormStore) FindObjectiveByChapter(ctx context.Context, workspaceID uuid.UUID, chapterID string) (Objective, bool, error) {
	var objective Objective
	err := store.db.WithContext(ctx).Where("workspace_id = ? AND chapter_id = ?", workspaceID, chapterID).First(&objective).Error
	if err == nil {
		return objective, true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Objective{}, false, nil
	}
	return Objective{}, false, err
}

func (store *GormStore) GetObjective(ctx context.Context, workspaceID, objectiveID uuid.UUID) (Objective, error) {
	var objective Objective
	err := store.db.WithContext(ctx).Where("id = ? AND workspace_id = ?", objectiveID, workspaceID).First(&objective).Error
	return objective, err
}

func (store *GormStore) ListAttempts(ctx context.Context, objectiveID uuid.UUID) ([]AttemptRecord, error) {
	var records []AttemptRecord
	err := store.db.WithContext(ctx).Where("objective_id = ?", objectiveID).Order("attempted_at ASC, created_at ASC, id ASC").Find(&records).Error
	return records, err
}

func (store *GormStore) AppendAttemptAndSchedule(ctx context.Context, record *AttemptRecord, schedule *Schedule) error {
	if record == nil || schedule == nil || record.ObjectiveID == uuid.Nil || schedule.ObjectiveID != record.ObjectiveID || record.ExpectedPreviousCount < 0 {
		return fmt.Errorf("invalid mastery append identity")
	}
	return store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Serialize evidence appends per objective; a schedule computed from stale
		// history must never replace the projection of a newer committed attempt.
		var objective Objective
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&objective, "id = ?", record.ObjectiveID).Error; err != nil {
			return err
		}
		completed, err := preparePracticeSessionAppend(tx, objective, record)
		if err != nil || completed {
			return err
		}
		var previousCount int64
		if err := tx.Model(&AttemptRecord{}).Where("objective_id = ?", record.ObjectiveID).Count(&previousCount).Error; err != nil {
			return err
		}
		if previousCount != int64(record.ExpectedPreviousCount) {
			return ErrMasteryHistoryChanged
		}
		artifact, err := prepareLearnerArtifact(objective, record)
		if err != nil {
			return err
		}
		if err := tx.Create(record).Error; err != nil {
			return err
		}
		if artifact != nil {
			encoded, err := json.Marshal(artifact)
			if err != nil {
				return err
			}
			// Source contents may contain private text; never include bound file
			// payloads in SQL error logs, even when the transaction is rejected.
			if err = tx.Session(&gorm.Session{Logger: tx.Logger.LogMode(logger.Silent)}).Exec("INSERT INTO mastery_learner_artifacts(attempt_id,objective_id,workspace_id,snapshot_hash,snapshot_json) VALUES(?,?,?,?,?::jsonb)", record.ID, objective.ID, objective.WorkspaceID, artifact.SnapshotHash, string(encoded)).Error; err != nil {
				return err
			}
		}
		// Re-read applications while holding the same objective lock used by
		// explicit grading apply. A later answer cannot restore stale self-ratings.
		projected, err := (&GormStore{db: tx}).ReplayAssessedSchedule(ctx, objective)
		if err != nil {
			return err
		}
		if projected != nil {
			*schedule = *projected
		}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "objective_id"}}, DoUpdates: clause.AssignmentColumns([]string{"next_skill", "due_at", "reason", "algorithm_version", "updated_at"})}).Create(schedule).Error; err != nil {
			return err
		}
		return finishPracticeSession(tx, record, schedule)
	})
}

// GetAttempt reads one saved answer under its objective identity.
func (store *GormStore) GetAttempt(ctx context.Context, objectiveID, attemptID uuid.UUID) (AttemptRecord, error) {
	var record AttemptRecord
	err := store.db.WithContext(ctx).Where("id = ? AND objective_id = ?", attemptID, objectiveID).First(&record).Error
	return record, err
}

func (store *GormStore) ListDue(ctx context.Context, workspaceID uuid.UUID, now time.Time, limit int) ([]DueObjective, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var due []DueObjective
	err := store.db.WithContext(ctx).
		Table("mastery_schedules").
		Select("mastery_schedules.objective_id, mastery_objectives.chapter_id, mastery_objectives.title, mastery_schedules.next_skill AS skill, mastery_schedules.due_at, mastery_schedules.reason").
		Joins("JOIN mastery_objectives ON mastery_objectives.id = mastery_schedules.objective_id").
		Where("mastery_objectives.workspace_id = ? AND mastery_schedules.due_at <= ? AND mastery_objectives.deleted_at IS NULL", workspaceID, now).
		Order("mastery_schedules.due_at ASC, mastery_schedules.objective_id ASC").
		Limit(limit).
		Scan(&due).Error
	return due, err
}
