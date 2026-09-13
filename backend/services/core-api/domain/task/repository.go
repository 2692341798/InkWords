package task

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Repository 定义任务领域访问持久化层所需的最小接口。
type Repository interface {
	FindByWorkspaceIdempotencyKey(ctx context.Context, workspaceID uuid.UUID, taskType, key string) (*JobTask, error)
	Create(ctx context.Context, task *JobTask) error
	GetByID(ctx context.Context, taskID uuid.UUID) (*JobTask, error)
	UpdateStatus(ctx context.Context, taskID uuid.UUID, status JobTaskStatus, errorMessage string) error
	UpdateResult(ctx context.Context, taskID uuid.UUID, result datatypes.JSON) error
	RetryFailed(ctx context.Context, taskID uuid.UUID) (*JobTask, error)
	AppendEvent(ctx context.Context, event *JobTaskEvent) error
	ListEventsAfter(ctx context.Context, taskID uuid.UUID, afterID uint64, limit int) ([]JobTaskEvent, error)
}

type resultPersistenceRepository interface {
	ClaimResultPersistence(ctx context.Context, taskID uuid.UUID, staleBefore time.Time) (bool, error)
	CompleteResultPersistence(ctx context.Context, taskID uuid.UUID) error
	ReleaseResultPersistence(ctx context.Context, taskID uuid.UUID) error
}

// Publisher 定义任务创建后向外部消息系统发布事件的能力边界。
type Publisher interface {
	PublishGenerationRequested(ctx context.Context, payload GenerationRequestedMessage) error
	PublishParseRequested(ctx context.Context, payload ParseRequestedMessage) error
	PublishExportRequested(ctx context.Context, payload ExportRequestedMessage) error
}

// textbookVerificationPublisher is opt-in to keep existing task publishers
// and tests compatible while the separate textbook runner is introduced.
type textbookVerificationPublisher interface {
	PublishTextbookVerificationRequested(context.Context, TextbookVerificationRequestedMessage) error
}

type textbookGenerationPublisher interface {
	PublishTextbookGenerationRequested(context.Context, TextbookGenerationRequestedMessage) error
}

type textbookParsePublisher interface {
	PublishTextbookParseRequested(context.Context, TextbookParseRequestedMessage) error
}

// GormRepository 使用 GORM 实现任务领域的数据访问。
type GormRepository struct {
	db *gorm.DB
}

// NewGormRepository 创建任务领域的 GORM 仓储实现。
func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

// FindByWorkspaceIdempotencyKey scopes task reuse to the stable installation workspace.
func (r *GormRepository) FindByWorkspaceIdempotencyKey(ctx context.Context, workspaceID uuid.UUID, taskType, key string) (*JobTask, error) {
	if workspaceID == uuid.Nil || strings.TrimSpace(key) == "" {
		return nil, nil
	}

	var task JobTask
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND task_type = ? AND idempotency_key = ?", workspaceID, taskType, strings.TrimSpace(key)).
		First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *GormRepository) Create(ctx context.Context, task *JobTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *GormRepository) GetByID(ctx context.Context, taskID uuid.UUID) (*JobTask, error) {
	var task JobTask
	err := r.db.WithContext(ctx).Where("id = ?", taskID).First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *GormRepository) UpdateStatus(ctx context.Context, taskID uuid.UUID, status JobTaskStatus, errorMessage string) error {
	updates := map[string]any{
		"status":        status,
		"error_message": errorMessage,
		"updated_at":    time.Now().UTC(),
	}
	if status == JobTaskStatusRunning {
		updates["started_at"] = time.Now().UTC()
	}
	if isTerminalStatus(status) {
		updates["finished_at"] = time.Now().UTC()
	}

	query := r.db.WithContext(ctx).Model(&JobTask{}).Where("id = ?", taskID)
	if status == JobTaskStatusCancelled {
		query = query.Where("status NOT IN ?", []JobTaskStatus{JobTaskStatusSucceeded, JobTaskStatusFailed, JobTaskStatusCancelled})
	} else {
		// A late worker update must not resurrect an acknowledged cancellation.
		query = query.Where("status <> ?", JobTaskStatusCancelled)
	}
	result := query.Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		_, err := r.GetByID(ctx, taskID)
		return err
	}
	return nil
}

func (r *GormRepository) UpdateResult(ctx context.Context, taskID uuid.UUID, result datatypes.JSON) error {
	updates := map[string]any{
		"result_json": append(datatypes.JSON(nil), result...),
		"updated_at":  time.Now().UTC(),
	}

	stored := r.db.WithContext(ctx).Model(&JobTask{}).Where("id = ? AND status <> ?", taskID, JobTaskStatusCancelled).Updates(updates)
	if stored.Error != nil {
		return stored.Error
	}
	if stored.RowsAffected == 0 {
		_, err := r.GetByID(ctx, taskID)
		return err
	}
	return nil
}

// RetryFailed moves only a failed task back to queued. Its task id and frozen
// payload remain unchanged, so a worker retry cannot silently use new inputs.
func (r *GormRepository) RetryFailed(ctx context.Context, taskID uuid.UUID) (*JobTask, error) {
	var task JobTask
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", taskID).First(&task).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTaskNotFound
			}
			return err
		}
		if task.Status != JobTaskStatusFailed {
			return ErrTaskNotRetryable
		}
		now := time.Now().UTC()
		if err := tx.Model(&JobTask{}).Where("id = ? AND status = ?", taskID, JobTaskStatusFailed).Updates(map[string]any{
			"status": JobTaskStatusQueued, "retry_count": gorm.Expr("retry_count + 1"), "error_message": "", "result_json": datatypes.JSON([]byte(`{}`)),
			"result_persistence_started_at": nil, "result_persisted_at": nil, "started_at": nil, "finished_at": nil, "updated_at": now,
		}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", taskID).First(&task).Error
	})
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *GormRepository) ClaimResultPersistence(ctx context.Context, taskID uuid.UUID, staleBefore time.Time) (bool, error) {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).Model(&JobTask{}).
		Where("id = ? AND result_persisted_at IS NULL AND (result_persistence_started_at IS NULL OR result_persistence_started_at < ?)", taskID, staleBefore).
		Updates(map[string]any{"result_persistence_started_at": now, "updated_at": now})
	return result.RowsAffected == 1, result.Error
}

func (r *GormRepository) CompleteResultPersistence(ctx context.Context, taskID uuid.UUID) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&JobTask{}).Where("id = ?", taskID).Updates(map[string]any{
		"result_persisted_at": now, "result_persistence_started_at": nil, "updated_at": now,
	}).Error
}

func (r *GormRepository) ReleaseResultPersistence(ctx context.Context, taskID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&JobTask{}).Where("id = ? AND result_persisted_at IS NULL", taskID).
		Updates(map[string]any{"result_persistence_started_at": nil, "updated_at": time.Now().UTC()}).Error
}

func (r *GormRepository) AppendEvent(ctx context.Context, event *JobTaskEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *GormRepository) ListEventsAfter(ctx context.Context, taskID uuid.UUID, afterID uint64, limit int) ([]JobTaskEvent, error) {
	var events []JobTaskEvent

	query := r.db.WithContext(ctx).
		Where("task_id = ? AND id > ?", taskID, afterID).
		Order("id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

var _ Repository = (*GormRepository)(nil)
var _ resultPersistenceRepository = (*GormRepository)(nil)
