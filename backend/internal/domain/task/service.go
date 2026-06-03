package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"inkwords-backend/internal/model"
)

// Service 封装生成任务的创建、取消与事件查询逻辑。
type Service struct {
	repo      Repository
	publisher Publisher
}

// NewService 通过依赖注入组装任务领域服务。
func NewService(repo Repository, publisher Publisher) *Service {
	return &Service{
		repo:      repo,
		publisher: publisher,
	}
}

// CreateGenerationTask 负责创建一条可幂等复用的生成任务，并在可用时发布创建事件。
func (s *Service) CreateGenerationTask(ctx context.Context, input CreateGenerationTaskInput) (model.JobTask, error) {
	if err := validateCreateGenerationTaskInput(input); err != nil {
		return model.JobTask{}, err
	}

	if input.IdempotencyKey != "" {
		existing, err := s.repo.FindByIdempotencyKey(ctx, input.RequestedBy, taskTypeGeneration, input.IdempotencyKey)
		if err != nil {
			return model.JobTask{}, fmt.Errorf("查找幂等任务失败: %w", err)
		}
		if existing != nil {
			return *existing, nil
		}
	}

	task := model.JobTask{
		TaskType:       taskTypeGeneration,
		TaskSubtype:    strings.TrimSpace(input.TaskSubtype),
		Status:         model.JobTaskStatusQueued,
		RequestedBy:    input.RequestedBy,
		IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
		PayloadJSON:    normalizeJSON(input.Payload),
		ResultJSON:     datatypes.JSON([]byte(`{}`)),
	}

	if err := s.repo.Create(ctx, &task); err != nil {
		return model.JobTask{}, fmt.Errorf("创建任务失败: %w", err)
	}

	if s.publisher != nil {
		if err := s.publisher.PublishGenerationRequested(ctx, GenerationRequestedMessage{
			TaskID:  task.ID,
			Kind:    task.TaskSubtype,
			UserID:  task.RequestedBy,
			Payload: append([]byte(nil), input.Payload...),
		}); err != nil {
			return model.JobTask{}, fmt.Errorf("发布任务消息失败: %w", err)
		}
	}

	return task, nil
}

// AppendEvent 负责把任务运行过程中的状态事件持久化，并同步更新任务主状态。
func (s *Service) AppendEvent(ctx context.Context, taskID uuid.UUID, input AppendEventInput) error {
	if _, err := s.repo.GetByID(ctx, taskID); err != nil {
		return wrapTaskLookupError(err)
	}

	event := model.JobTaskEvent{
		TaskID:    taskID,
		EventType: strings.TrimSpace(input.EventType),
		Status:    input.Status,
		Payload:   normalizeJSON(input.Payload),
	}
	if err := s.repo.AppendEvent(ctx, &event); err != nil {
		return fmt.Errorf("追加任务事件失败: %w", err)
	}

	if input.Status != "" {
		if err := s.repo.UpdateStatus(ctx, taskID, input.Status, ""); err != nil {
			return fmt.Errorf("更新任务状态失败: %w", err)
		}
	}
	return nil
}

// CancelTask 仅允许任务创建者取消自己的非终态任务。
func (s *Service) CancelTask(ctx context.Context, taskID uuid.UUID, requestedBy uuid.UUID) error {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return wrapTaskLookupError(err)
	}
	if task.RequestedBy != requestedBy {
		return ErrTaskAccessDenied
	}
	if isTerminalStatus(task.Status) {
		return nil
	}
	if err := s.repo.UpdateStatus(ctx, taskID, model.JobTaskStatusCancelled, ""); err != nil {
		return fmt.Errorf("取消任务失败: %w", err)
	}

	// 取消动作也要落一条事件，后续 SSE 与审计才能观察到终态来源。
	if err := s.repo.AppendEvent(ctx, &model.JobTaskEvent{
		TaskID:    taskID,
		EventType: "cancelled",
		Status:    model.JobTaskStatusCancelled,
		Payload:   datatypes.JSON([]byte(`{"status":"cancelled"}`)),
	}); err != nil {
		return fmt.Errorf("写入取消事件失败: %w", err)
	}

	return nil
}

// GetTask 返回任务快照，并确保调用方只能读取自己的任务。
func (s *Service) GetTask(ctx context.Context, taskID uuid.UUID, requestedBy uuid.UUID) (model.JobTask, error) {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return model.JobTask{}, wrapTaskLookupError(err)
	}
	if task.RequestedBy != requestedBy {
		return model.JobTask{}, ErrTaskAccessDenied
	}
	return *task, nil
}

// ListStreamEvents 返回 afterID 之后的事件，并告知调用方任务是否已进入终态。
func (s *Service) ListStreamEvents(ctx context.Context, taskID uuid.UUID, afterID uint64) ([]model.JobTaskEvent, bool, error) {
	events, err := s.repo.ListEventsAfter(ctx, taskID, afterID, defaultStreamEventLimit)
	if err != nil {
		return nil, false, fmt.Errorf("查询任务事件失败: %w", err)
	}

	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, false, wrapTaskLookupError(err)
	}

	return events, isTerminalStatus(task.Status), nil
}

// MarkRunning 把任务切到运行态，供异步 worker 在真正开始消费前回写可观测状态。
func (s *Service) MarkRunning(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return wrapTaskLookupError(err)
	}
	if task.Status == model.JobTaskStatusCancelled {
		return nil
	}
	if err := s.repo.UpdateStatus(ctx, taskID, model.JobTaskStatusRunning, ""); err != nil {
		return fmt.Errorf("更新任务运行状态失败: %w", err)
	}
	return nil
}

// MarkSucceeded 在 worker 正常结束后写回终态与结果快照，供任务查询接口复用。
func (s *Service) MarkSucceeded(ctx context.Context, taskID uuid.UUID, result []byte) error {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return wrapTaskLookupError(err)
	}
	if task.Status == model.JobTaskStatusCancelled {
		return nil
	}
	if err := s.repo.UpdateResult(ctx, taskID, normalizeJSON(result)); err != nil {
		return fmt.Errorf("更新任务结果失败: %w", err)
	}
	if err := s.repo.UpdateStatus(ctx, taskID, model.JobTaskStatusSucceeded, ""); err != nil {
		return fmt.Errorf("更新任务成功状态失败: %w", err)
	}
	return nil
}

// MarkFailed 把后台执行错误写回任务主状态，并追加一条 error 事件供 SSE 订阅端消费。
func (s *Service) MarkFailed(ctx context.Context, taskID uuid.UUID, message string) error {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return wrapTaskLookupError(err)
	}
	if task.Status == model.JobTaskStatusCancelled {
		return nil
	}

	trimmedMessage := strings.TrimSpace(message)
	payload, err := json.Marshal(map[string]string{
		"status":  string(model.JobTaskStatusFailed),
		"message": trimmedMessage,
	})
	if err != nil {
		return fmt.Errorf("序列化任务失败事件失败: %w", err)
	}
	if err := s.repo.AppendEvent(ctx, &model.JobTaskEvent{
		TaskID:    taskID,
		EventType: "error",
		Status:    model.JobTaskStatusFailed,
		Payload:   datatypes.JSON(payload),
	}); err != nil {
		return fmt.Errorf("追加失败事件失败: %w", err)
	}
	if err := s.repo.UpdateStatus(ctx, taskID, model.JobTaskStatusFailed, trimmedMessage); err != nil {
		return fmt.Errorf("更新任务失败状态失败: %w", err)
	}
	return nil
}

// IsCancelled 供 worker 轮询取消态，避免用户主动取消后仍继续占用生成资源。
func (s *Service) IsCancelled(ctx context.Context, taskID uuid.UUID) (bool, error) {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return false, wrapTaskLookupError(err)
	}
	return task.Status == model.JobTaskStatusCancelled, nil
}

func validateCreateGenerationTaskInput(input CreateGenerationTaskInput) error {
	if input.RequestedBy == uuid.Nil {
		return ErrEmptyRequestedBy
	}
	if strings.TrimSpace(input.TaskSubtype) == "" {
		return ErrEmptyTaskSubtype
	}
	return nil
}

func normalizeJSON(payload []byte) datatypes.JSON {
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

func wrapTaskLookupError(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), ErrTaskNotFound.Error()) {
		return ErrTaskNotFound
	}
	return fmt.Errorf("查询任务失败: %w", err)
}

func isTerminalStatus(status model.JobTaskStatus) bool {
	switch status {
	case model.JobTaskStatusSucceeded, model.JobTaskStatusFailed, model.JobTaskStatusCancelled:
		return true
	default:
		return false
	}
}
