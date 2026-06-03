package task

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
)

// Repository 定义任务领域访问持久化层所需的最小接口。
type Repository interface {
	FindByIdempotencyKey(ctx context.Context, requestedBy uuid.UUID, taskType, key string) (*model.JobTask, error)
	Create(ctx context.Context, task *model.JobTask) error
	GetByID(ctx context.Context, taskID uuid.UUID) (*model.JobTask, error)
	UpdateStatus(ctx context.Context, taskID uuid.UUID, status model.JobTaskStatus, errorMessage string) error
	UpdateResult(ctx context.Context, taskID uuid.UUID, result datatypes.JSON) error
	AppendEvent(ctx context.Context, event *model.JobTaskEvent) error
	ListEventsAfter(ctx context.Context, taskID uuid.UUID, afterID uint64, limit int) ([]model.JobTaskEvent, error)
}

// Publisher 定义任务创建后向外部消息系统发布事件的能力边界。
type Publisher interface {
	PublishGenerationRequested(ctx context.Context, payload GenerationRequestedMessage) error
}

// GormRepository 使用 GORM 实现任务领域的数据访问。
type GormRepository struct {
	db *gorm.DB
}

// NewGormRepository 创建任务领域的 GORM 仓储实现。
func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) FindByIdempotencyKey(ctx context.Context, requestedBy uuid.UUID, taskType, key string) (*model.JobTask, error) {
	if strings.TrimSpace(key) == "" {
		return nil, nil
	}

	var task model.JobTask
	err := r.db.WithContext(ctx).
		Where("requested_by = ? AND task_type = ? AND idempotency_key = ?", requestedBy, taskType, strings.TrimSpace(key)).
		First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *GormRepository) Create(ctx context.Context, task *model.JobTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *GormRepository) GetByID(ctx context.Context, taskID uuid.UUID) (*model.JobTask, error) {
	var task model.JobTask
	err := r.db.WithContext(ctx).Where("id = ?", taskID).First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *GormRepository) UpdateStatus(ctx context.Context, taskID uuid.UUID, status model.JobTaskStatus, errorMessage string) error {
	updates := map[string]any{
		"status":        status,
		"error_message": errorMessage,
		"updated_at":    time.Now().UTC(),
	}
	if status == model.JobTaskStatusRunning {
		updates["started_at"] = time.Now().UTC()
	}
	if isTerminalStatus(status) {
		updates["finished_at"] = time.Now().UTC()
	}

	result := r.db.WithContext(ctx).Model(&model.JobTask{}).Where("id = ?", taskID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTaskNotFound
	}
	return nil
}

func (r *GormRepository) UpdateResult(ctx context.Context, taskID uuid.UUID, result datatypes.JSON) error {
	updates := map[string]any{
		"result_json": append(datatypes.JSON(nil), result...),
		"updated_at":  time.Now().UTC(),
	}

	stored := r.db.WithContext(ctx).Model(&model.JobTask{}).Where("id = ?", taskID).Updates(updates)
	if stored.Error != nil {
		return stored.Error
	}
	if stored.RowsAffected == 0 {
		return ErrTaskNotFound
	}
	return nil
}

func (r *GormRepository) AppendEvent(ctx context.Context, event *model.JobTaskEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *GormRepository) ListEventsAfter(ctx context.Context, taskID uuid.UUID, afterID uint64, limit int) ([]model.JobTaskEvent, error) {
	var events []model.JobTaskEvent

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
