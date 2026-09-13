package stream

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

type TaskStatus string

const (
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusStreaming TaskStatus = "streaming"
	TaskStatusSucceeded TaskStatus = "succeeded"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

var errTaskNotFound = errors.New("task not found")

type AppendEventInput struct {
	EventType string
	Status    TaskStatus
	Payload   []byte
}

// textbookStageResultReuseStore recognizes a completed stage for the exact
// delivery identity. The result comes from the durable stage checkpoint, not
// the mutable terminal task row, so a retry can resume after a terminal-state
// write failed. Keeping it optional preserves legacy task-store fakes.
type textbookStageResultReuseStore interface {
	FindCompletedTextbookStageResult(ctx context.Context, taskID uuid.UUID, stageExecutionKey string) ([]byte, bool, error)
}

type jobTask struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Status       TaskStatus     `gorm:"type:varchar(16);not null;index"`
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
	Status    TaskStatus     `gorm:"type:varchar(16);not null;index"`
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
	if task.Status == TaskStatusCancelled {
		return nil
	}

	now := time.Now().UTC()
	return s.updateTask(ctx, taskID, map[string]any{
		"status":        TaskStatusRunning,
		"error_message": "",
		"started_at":    now,
		"updated_at":    now,
	})
}

func (s *GormTaskStore) AppendEvent(ctx context.Context, taskID uuid.UUID, input AppendEventInput) error {
	if _, err := s.getByID(ctx, taskID); err != nil {
		return err
	}

	if err := s.db.WithContext(ctx).Create(&jobTaskEvent{
		TaskID:    taskID,
		EventType: strings.TrimSpace(input.EventType),
		Status:    input.Status,
		Payload:   normalizeTaskJSON(input.Payload),
	}).Error; err != nil {
		return fmt.Errorf("追加任务事件失败: %w", err)
	}

	if input.Status != "" {
		if err := s.updateTask(ctx, taskID, map[string]any{
			"status":     input.Status,
			"updated_at": time.Now().UTC(),
		}); err != nil {
			return fmt.Errorf("更新任务状态失败: %w", err)
		}
	}
	return nil
}

func (s *GormTaskStore) MarkSucceeded(ctx context.Context, taskID uuid.UUID, result []byte) error {
	task, err := s.getByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task.Status == TaskStatusCancelled {
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
		"status":        TaskStatusSucceeded,
		"error_message": "",
		"finished_at":   now,
		"updated_at":    now,
	})
}

func (s *GormTaskStore) MarkFailed(ctx context.Context, taskID uuid.UUID, message string) error {
	return s.markFailed(ctx, taskID, message, nil)
}

// MarkFailedWithResult persists safe structured failure evidence alongside the
// terminal task state. Callers must omit rejected model prose and raw provider
// diagnostics; the task store still applies its credential redaction guard.
func (s *GormTaskStore) MarkFailedWithResult(ctx context.Context, taskID uuid.UUID, message string, result []byte) error {
	if len(result) == 0 || !json.Valid(result) {
		return fmt.Errorf("失败任务结果必须是有效 JSON")
	}
	return s.markFailed(ctx, taskID, message, result)
}

func (s *GormTaskStore) markFailed(ctx context.Context, taskID uuid.UUID, message string, result []byte) error {
	task, err := s.getByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task.Status == TaskStatusCancelled {
		return nil
	}

	trimmedMessage := sanitizeTaskFailureMessage(message)
	payloadFields := map[string]any{
		"status":  string(TaskStatusFailed),
		"message": trimmedMessage,
	}
	if len(result) > 0 {
		payloadFields["failure"] = json.RawMessage(result)
	}
	payload, err := json.Marshal(payloadFields)
	if err != nil {
		return fmt.Errorf("序列化任务失败事件失败: %w", err)
	}

	if err := s.db.WithContext(ctx).Create(&jobTaskEvent{
		TaskID:    taskID,
		EventType: "error",
		Status:    TaskStatusFailed,
		Payload:   datatypes.JSON(payload),
	}).Error; err != nil {
		return fmt.Errorf("追加失败事件失败: %w", err)
	}

	updates := map[string]any{
		"status":        TaskStatusFailed,
		"error_message": trimmedMessage,
		"finished_at":   time.Now().UTC(),
		"updated_at":    time.Now().UTC(),
	}
	if len(result) > 0 {
		updates["result_json"] = append(datatypes.JSON(nil), result...)
	}
	return s.updateTask(ctx, taskID, updates)
}

// sanitizeTaskFailureMessage is the final persistence guard for diagnostics
// received from providers or transport libraries. Provider adapters already
// return structured errors, but task storage must remain safe if a future
// adapter accidentally includes a credential in its error text.
func sanitizeTaskFailureMessage(message string) string {
	trimmed := strings.TrimSpace(message)
	lower := strings.ToLower(trimmed)
	for _, marker := range []string{"api_key", "apikey", "authorization", "bearer ", "sk-"} {
		if strings.Contains(lower, marker) {
			return "generation task failed; provider diagnostic redacted"
		}
	}
	return trimmed
}

func (s *GormTaskStore) IsCancelled(ctx context.Context, taskID uuid.UUID) (bool, error) {
	task, err := s.getByID(ctx, taskID)
	if err != nil {
		return false, err
	}
	return task.Status == TaskStatusCancelled, nil
}

func (s *GormTaskStore) TextbookWorkspaceMatches(ctx context.Context, taskID, workspaceID uuid.UUID, taskSubtype string) (bool, error) {
	if taskID == uuid.Nil || workspaceID == uuid.Nil || strings.TrimSpace(taskSubtype) == "" {
		return false, nil
	}
	var count int64
	err := s.db.WithContext(ctx).Model(&jobTask{}).
		Where("id = ? AND workspace_id = ? AND task_subtype = ?", taskID, workspaceID, strings.TrimSpace(taskSubtype)).
		Count(&count).Error
	return count == 1, err
}

// FindCompletedTextbookStageResult returns only a result recorded in a completed
// stage checkpoint for this task. The stage key binds task ID, stage, and frozen
// input hash, so a redelivery cannot accidentally reuse another task. It does
// not depend on task.status: terminal state may be the write that failed after
// a stage was already completed.
func (s *GormTaskStore) FindCompletedTextbookStageResult(ctx context.Context, taskID uuid.UUID, stageExecutionKey string) ([]byte, bool, error) {
	if taskID == uuid.Nil || strings.TrimSpace(stageExecutionKey) == "" {
		return nil, false, nil
	}
	var resultJSON datatypes.JSON
	query := s.db.WithContext(ctx).Table("job_task_events AS event").
		Select("event.payload->'result' AS result_json").
		Where("event.task_id = ? AND event.event_type = ? AND event.payload->>'stage_execution_key' = ? AND event.payload->>'checkpoint' = ? AND (event.payload->>'completed')::boolean = true AND event.payload ? 'result'", taskID, "textbook_sample_phase", stageExecutionKey, "result_ready").
		Order("event.created_at DESC").Limit(1).Scan(&resultJSON)
	if query.Error != nil {
		return nil, false, query.Error
	}
	if query.RowsAffected == 0 || len(resultJSON) == 0 || string(resultJSON) == "{}" {
		return nil, false, nil
	}
	return append([]byte(nil), resultJSON...), true, nil
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
