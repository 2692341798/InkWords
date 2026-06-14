package export

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type taskStatus string

const (
	taskStatusRunning   taskStatus = "running"
	taskStatusSucceeded taskStatus = "succeeded"
	taskStatusFailed    taskStatus = "failed"
	taskStatusCancelled taskStatus = "cancelled"
)

var errTaskNotFound = errors.New("task not found")

type jobTask struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey"`
	TaskType     string         `gorm:"type:varchar(32);not null;index"`
	Status       taskStatus     `gorm:"type:varchar(16);not null;index"`
	ResultJSON   datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	ErrorMessage string         `gorm:"type:text"`
	StartedAt    *time.Time
	FinishedAt   *time.Time
	UpdatedAt    time.Time
}

func (jobTask) TableName() string {
	return "job_tasks"
}

type jobTaskEvent struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement"`
	TaskID    uuid.UUID      `gorm:"type:uuid;not null;index"`
	EventType string         `gorm:"type:varchar(32);not null;index"`
	Status    taskStatus     `gorm:"type:varchar(16);not null;index"`
	Payload   datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt time.Time
}

func (jobTaskEvent) TableName() string {
	return "job_task_events"
}

type GormTaskStore struct {
	db *gorm.DB
}

func NewGormTaskStore(db *gorm.DB) *GormTaskStore {
	return &GormTaskStore{db: db}
}

func (s *GormTaskStore) MarkRunning(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.getByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task.Status == taskStatusCancelled {
		return nil
	}

	now := time.Now().UTC()
	return s.updateTask(ctx, taskID, map[string]any{
		"status":        taskStatusRunning,
		"error_message": "",
		"started_at":    now,
		"updated_at":    now,
	})
}

func (s *GormTaskStore) MarkSucceeded(ctx context.Context, taskID uuid.UUID, result []byte) error {
	task, err := s.getByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task.Status == taskStatusCancelled {
		return nil
	}

	now := time.Now().UTC()
	if err := s.updateTask(ctx, taskID, map[string]any{
		"result_json": append(datatypes.JSON(nil), normalizeTaskJSON(result)...),
		"updated_at":  now,
	}); err != nil {
		return fmt.Errorf("更新任务结果失败: %w", err)
	}

	return s.updateTask(ctx, taskID, map[string]any{
		"status":        taskStatusSucceeded,
		"error_message": "",
		"finished_at":   now,
		"updated_at":    now,
	})
}

func (s *GormTaskStore) MarkFailed(ctx context.Context, taskID uuid.UUID, message string) error {
	task, err := s.getByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task.Status == taskStatusCancelled {
		return nil
	}

	trimmedMessage := strings.TrimSpace(message)
	payload, err := json.Marshal(map[string]string{
		"status":  string(taskStatusFailed),
		"message": trimmedMessage,
	})
	if err != nil {
		return fmt.Errorf("序列化任务失败事件失败: %w", err)
	}

	if err := s.db.WithContext(ctx).Create(&jobTaskEvent{
		TaskID:    taskID,
		EventType: "error",
		Status:    taskStatusFailed,
		Payload:   datatypes.JSON(payload),
	}).Error; err != nil {
		return fmt.Errorf("追加失败事件失败: %w", err)
	}

	now := time.Now().UTC()
	return s.updateTask(ctx, taskID, map[string]any{
		"status":        taskStatusFailed,
		"error_message": trimmedMessage,
		"finished_at":   now,
		"updated_at":    now,
	})
}

func (s *GormTaskStore) IsCancelled(ctx context.Context, taskID uuid.UUID) (bool, error) {
	task, err := s.getByID(ctx, taskID)
	if err != nil {
		return false, err
	}
	return task.Status == taskStatusCancelled, nil
}

func (s *GormTaskStore) getByID(ctx context.Context, taskID uuid.UUID) (jobTask, error) {
	var task jobTask
	if err := s.db.WithContext(ctx).Where("id = ?", taskID).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return jobTask{}, errTaskNotFound
		}
		return jobTask{}, err
	}
	return task, nil
}

func (s *GormTaskStore) updateTask(ctx context.Context, taskID uuid.UUID, updates map[string]any) error {
	result := s.db.WithContext(ctx).Model(&jobTask{}).Where("id = ?", taskID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errTaskNotFound
	}
	return nil
}

func normalizeTaskJSON(payload []byte) datatypes.JSON {
	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" {
		return datatypes.JSON([]byte(`{}`))
	}

	var raw json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return datatypes.JSON([]byte(`{}`))
	}
	return datatypes.JSON(raw)
}
