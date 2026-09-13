package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// Service 封装生成任务的创建、取消与事件查询逻辑。
type Service struct {
	repo                 Repository
	publisher            Publisher
	resultPersister      GenerationResultPersister
	parseResultPersister ParseResultPersister
}

// NewService 通过依赖注入组装任务领域服务。
func NewService(repo Repository, publisher Publisher, resultPersister GenerationResultPersister) *Service {
	parseResultPersister, _ := resultPersister.(ParseResultPersister)
	return &Service{
		repo:                 repo,
		publisher:            publisher,
		resultPersister:      resultPersister,
		parseResultPersister: parseResultPersister,
	}
}

// GenerationResultPersister defines the core-api owned success-path side effect that
// materializes structured task results into business facts.
type GenerationResultPersister interface {
	PersistGenerationResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error
}

// ParseResultPersister is intentionally opt-in: only typed parser results may
// create durable business facts during task reconciliation.
type ParseResultPersister interface {
	PersistParseResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error
}

// CreateGenerationTask 负责创建一条可幂等复用的生成任务，并在可用时发布创建事件。
func (s *Service) CreateGenerationTask(ctx context.Context, input CreateGenerationTaskInput) (JobTask, error) {
	return s.createTask(ctx, createTaskParams{
		taskType:          taskTypeGeneration,
		taskSubtype:       input.TaskSubtype,
		workspaceID:       input.WorkspaceID,
		textbookChapterID: input.TextbookChapterID,
		idempotencyKey:    input.IdempotencyKey,
		payload:           input.Payload,
		publish: func(task JobTask) error {
			if s.publisher == nil {
				return nil
			}
			return s.publishGenerationRequested(ctx, task, input.Payload)
		},
	})
}

// RetryGenerationTask republishes the exact frozen payload of a failed
// generation or parse task. The name is retained for the existing HTTP contract;
// it does not create a new task or re-run task preparation.
func (s *Service) RetryGenerationTask(ctx context.Context, taskID, workspaceID uuid.UUID) (JobTask, error) {
	task, err := s.GetTask(ctx, taskID, workspaceID)
	if err != nil {
		return JobTask{}, err
	}
	return s.retryResolvedTask(ctx, task)
}

// RetryTextbookTask requeues only a task owned by the server-resolved local workspace.
func (s *Service) RetryTextbookTask(ctx context.Context, taskID, workspaceID uuid.UUID) (JobTask, error) {
	task, err := s.GetTextbookTask(ctx, taskID, workspaceID)
	if err != nil {
		return JobTask{}, err
	}
	return s.retryResolvedTask(ctx, task)
}

func (s *Service) retryResolvedTask(ctx context.Context, task JobTask) (JobTask, error) {
	verification := task.TaskType == taskTypeVerification && task.TaskSubtype == sharedtextbook.TextbookTeachingArtifactVerifyTaskSubtype
	if verification && task.VerificationAttempt>0{return JobTask{},ErrTaskNotRetryable}
	if (task.TaskType != taskTypeGeneration && task.TaskType != taskTypeParse && !verification) || task.Status != JobTaskStatusFailed {
		return JobTask{}, ErrTaskNotRetryable
	}
	if verification {
		var frozen sharedtextbook.ArtifactVerificationRequest
		if json.Unmarshal(task.PayloadJSON, &frozen) != nil || frozen.Validate() != nil {
			return JobTask{}, ErrTaskNotRetryable
		}
	}
	retried, err := s.repo.RetryFailed(ctx, task.ID)
	if err != nil {
		return JobTask{}, err
	}
	if s.publisher == nil {
		return *retried, nil
	}
	var publishErr error
	switch retried.TaskType {
	case taskTypeGeneration:
		publishErr = s.publishGenerationRequested(ctx, *retried, retried.PayloadJSON)
	case taskTypeParse:
		publishErr = s.publishParseRequested(ctx, *retried, retried.PayloadJSON)
	case taskTypeVerification:
		publishErr = s.publishTextbookVerification(ctx, *retried, retried.PayloadJSON)
	}
	if publishErr != nil {
		_ = s.repo.UpdateStatus(ctx, retried.ID, JobTaskStatusFailed, "发布重试任务消息失败")
		return JobTask{}, fmt.Errorf("发布重试任务消息失败: %w", publishErr)
	}
	return *retried, nil
}

// CreateParseTask 负责创建一条解析任务，并在可用时发布给 parser-service worker。
func (s *Service) CreateParseTask(ctx context.Context, input CreateParseTaskInput) (JobTask, error) {
	return s.createTask(ctx, createTaskParams{
		taskType:       taskTypeParse,
		taskSubtype:    input.TaskSubtype,
		workspaceID:    input.WorkspaceID,
		idempotencyKey: input.IdempotencyKey,
		payload:        input.Payload,
		publish: func(task JobTask) error {
			if s.publisher == nil {
				return nil
			}
			return s.publishParseRequested(ctx, task, input.Payload)
		},
	})
}

// CreateTextbookVerificationTask publishes only immutable artifact metadata
// to the isolated textbook verifier. It never reuses a legacy lab
// routing or accepts a host path/command from a browser.
func (s *Service) CreateTextbookVerificationTask(ctx context.Context, input CreateTextbookVerificationTaskInput) (JobTask, error) {
	return s.createTask(ctx, createTaskParams{
		taskType:       taskTypeVerification,
		taskSubtype:    input.TaskSubtype,
		workspaceID:    input.WorkspaceID,
		idempotencyKey: input.IdempotencyKey,
		payload:        input.Payload,
		publish: func(task JobTask) error {
			if s.publisher == nil {
				return nil
			}
			return s.publishTextbookVerification(ctx, task, input.Payload)
		},
	})
}

func (s *Service) publishTextbookVerification(ctx context.Context, task JobTask, payload []byte) error {
	publisher, ok := s.publisher.(textbookVerificationPublisher)
	if !ok {
		return fmt.Errorf("textbook verification publisher is not configured")
	}
	return publisher.PublishTextbookVerificationRequested(ctx, TextbookVerificationRequestedMessage{TaskID: task.ID, Kind: task.TaskSubtype, WorkspaceID: cloneUUID(task.WorkspaceID), Payload: append([]byte(nil), payload...)})
}

// CreateExportTask 负责创建一条导出任务，并在可用时发布给 export-service worker。
func (s *Service) CreateExportTask(ctx context.Context, input CreateExportTaskInput) (JobTask, error) {
	return s.createTask(ctx, createTaskParams{
		taskType:       taskTypeExport,
		taskSubtype:    input.TaskSubtype,
		workspaceID:    input.WorkspaceID,
		idempotencyKey: input.IdempotencyKey,
		payload:        input.Payload,
		publish: func(task JobTask) error {
			if s.publisher == nil {
				return nil
			}
			return s.publisher.PublishExportRequested(ctx, ExportRequestedMessage{
				TaskID: task.ID, Kind: task.TaskSubtype,
				WorkspaceID: cloneUUID(task.WorkspaceID), Payload: append([]byte(nil), input.Payload...),
			})
		},
	})
}

func (s *Service) publishGenerationRequested(ctx context.Context, task JobTask, payload []byte) error {
	if task.TaskSubtype == sharedtextbook.TextbookSampleGenerationTaskSubtype {
		publisher, ok := s.publisher.(textbookGenerationPublisher)
		if !ok {
			return fmt.Errorf("textbook generation publisher is not configured")
		}
		return publisher.PublishTextbookGenerationRequested(ctx, TextbookGenerationRequestedMessage{
			TaskID: task.ID, Kind: task.TaskSubtype, WorkspaceID: cloneUUID(task.WorkspaceID), Payload: append([]byte(nil), payload...),
		})
	}
	return s.publisher.PublishGenerationRequested(ctx, GenerationRequestedMessage{
		TaskID: task.ID, Kind: task.TaskSubtype, WorkspaceID: cloneUUID(task.WorkspaceID), Payload: append([]byte(nil), payload...),
	})
}

func (s *Service) publishParseRequested(ctx context.Context, task JobTask, payload []byte) error {
	if task.TaskSubtype == sharedtextbook.TextbookSourceImportTaskSubtype || task.TaskSubtype == sharedtextbook.TextbookOfficialWebImportTaskSubtype {
		publisher, ok := s.publisher.(textbookParsePublisher)
		if !ok {
			return fmt.Errorf("textbook parse publisher is not configured")
		}
		return publisher.PublishTextbookParseRequested(ctx, TextbookParseRequestedMessage{
			TaskID: task.ID, Kind: task.TaskSubtype, WorkspaceID: cloneUUID(task.WorkspaceID), Payload: append([]byte(nil), payload...),
		})
	}
	return s.publisher.PublishParseRequested(ctx, ParseRequestedMessage{
		TaskID: task.ID, Kind: task.TaskSubtype, WorkspaceID: cloneUUID(task.WorkspaceID), Payload: append([]byte(nil), payload...),
	})
}

type createTaskParams struct {
	taskType          string
	taskSubtype       string
	workspaceID       uuid.UUID
	textbookChapterID *uuid.UUID
	idempotencyKey    string
	payload           []byte
	publish           func(task JobTask) error
}

func (s *Service) createTask(ctx context.Context, params createTaskParams) (JobTask, error) {
	if err := validateCreateTaskInput(params.workspaceID, params.taskSubtype); err != nil {
		return JobTask{}, err
	}

	if params.idempotencyKey != "" {
		existing, err := s.repo.FindByWorkspaceIdempotencyKey(ctx, params.workspaceID, params.taskType, params.idempotencyKey)
		if err != nil {
			return JobTask{}, fmt.Errorf("查找幂等任务失败: %w", err)
		}
		if existing != nil {
			return *existing, nil
		}
	}

	task := JobTask{
		TaskType:          params.taskType,
		TaskSubtype:       strings.TrimSpace(params.taskSubtype),
		Status:            JobTaskStatusQueued,
		TextbookChapterID: cloneUUID(params.textbookChapterID),
		IdempotencyKey:    strings.TrimSpace(params.idempotencyKey),
		PayloadJSON:       normalizeJSON(params.payload),
		ResultJSON:        datatypes.JSON([]byte(`{}`)),
	}
	workspaceID := params.workspaceID
	task.WorkspaceID = &workspaceID

	if err := s.repo.Create(ctx, &task); err != nil {
		return JobTask{}, fmt.Errorf("创建任务失败: %w", err)
	}

	if params.publish != nil {
		if err := params.publish(task); err != nil {
			// The task already exists at this point. Leaving it queued makes every
			// idempotent retry reuse an item that was never delivered to a worker,
			// so SSE waits forever. Persist a terminal failure instead.
			_ = s.repo.UpdateStatus(ctx, task.ID, JobTaskStatusFailed, "发布任务消息失败")
			return JobTask{}, fmt.Errorf("发布任务消息失败: %w", err)
		}
	}

	return task, nil
}

// AppendEvent 负责把任务运行过程中的状态事件持久化，并同步更新任务主状态。
func (s *Service) AppendEvent(ctx context.Context, taskID uuid.UUID, input AppendEventInput) error {
	if _, err := s.repo.GetByID(ctx, taskID); err != nil {
		return wrapTaskLookupError(err)
	}

	event := JobTaskEvent{
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
func (s *Service) CancelTask(ctx context.Context, taskID uuid.UUID, workspaceID uuid.UUID) error {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return wrapTaskLookupError(err)
	}
	if workspaceID == uuid.Nil || task.WorkspaceID == nil || *task.WorkspaceID != workspaceID {
		return ErrTaskAccessDenied
	}
	if isTerminalStatus(task.Status) {
		return nil
	}
	if err := s.repo.UpdateStatus(ctx, taskID, JobTaskStatusCancelled, ""); err != nil {
		return fmt.Errorf("取消任务失败: %w", err)
	}
	current, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return wrapTaskLookupError(err)
	}
	if current.Status != JobTaskStatusCancelled {
		return nil // Completion committed before cancellation acquired the row.
	}

	// 取消动作也要落一条事件，后续 SSE 与审计才能观察到终态来源。
	if err := s.repo.AppendEvent(ctx, &JobTaskEvent{
		TaskID:    taskID,
		EventType: "cancelled",
		Status:    JobTaskStatusCancelled,
		Payload:   datatypes.JSON([]byte(`{"status":"cancelled"}`)),
	}); err != nil {
		return fmt.Errorf("写入取消事件失败: %w", err)
	}

	return nil
}

// GetTask 返回任务快照，并确保调用方只能读取自己的任务。
func (s *Service) GetTask(ctx context.Context, taskID uuid.UUID, workspaceID uuid.UUID) (JobTask, error) {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return JobTask{}, wrapTaskLookupError(err)
	}
	if workspaceID == uuid.Nil || task.WorkspaceID == nil || *task.WorkspaceID != workspaceID {
		return JobTask{}, ErrTaskAccessDenied
	}
	if err := s.reconcileGenerationResult(ctx, task); err != nil {
		return JobTask{}, err
	}
	return *task, nil
}

// GetTextbookTask authorizes through the local workspace column.
func (s *Service) GetTextbookTask(ctx context.Context, taskID, workspaceID uuid.UUID) (JobTask, error) {
	if workspaceID == uuid.Nil {
		return JobTask{}, ErrTaskAccessDenied
	}
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return JobTask{}, wrapTaskLookupError(err)
	}
	if task.WorkspaceID == nil || *task.WorkspaceID != workspaceID || !isTextbookTaskSubtype(task.TaskSubtype) {
		return JobTask{}, ErrTaskAccessDenied
	}
	if err := s.reconcileGenerationResult(ctx, task); err != nil {
		return JobTask{}, err
	}
	return *task, nil
}

// ListStreamEvents 返回 afterID 之后的事件，并告知调用方任务是否已进入终态。
func (s *Service) ListStreamEvents(ctx context.Context, taskID uuid.UUID, afterID uint64) ([]JobTaskEvent, bool, error) {
	events, err := s.repo.ListEventsAfter(ctx, taskID, afterID, defaultStreamEventLimit)
	if err != nil {
		return nil, false, fmt.Errorf("查询任务事件失败: %w", err)
	}

	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, false, wrapTaskLookupError(err)
	}

	if err := s.reconcileGenerationResult(ctx, task); err != nil {
		return nil, false, err
	}
	return events, isTerminalStatus(task.Status), nil
}

func (s *Service) reconcileGenerationResult(ctx context.Context, task *JobTask) error {
	if task == nil || task.Status != JobTaskStatusSucceeded || task.ResultPersistedAt != nil {
		return nil
	}
	if task.TaskType == taskTypeGeneration && s.resultPersister == nil {
		return nil
	}
	if task.TaskType == taskTypeParse && ((task.TaskSubtype != sharedtextbook.TextbookSourceImportTaskSubtype && task.TaskSubtype != sharedtextbook.TextbookOfficialWebImportTaskSubtype) || s.parseResultPersister == nil) {
		return nil
	}
	if task.TaskType != taskTypeGeneration && task.TaskType != taskTypeParse {
		return nil
	}
	repo, ok := s.repo.(resultPersistenceRepository)
	if !ok {
		return nil
	}
	claimed, err := repo.ClaimResultPersistence(ctx, task.ID, time.Now().UTC().Add(-5*time.Minute))
	if err != nil || !claimed {
		return err
	}

	var decoded map[string]any
	if err := json.Unmarshal(task.ResultJSON, &decoded); err != nil {
		_ = repo.ReleaseResultPersistence(ctx, task.ID)
		return fmt.Errorf("解析任务结果失败: %w", err)
	}
	var persistErr error
	if task.TaskType == taskTypeParse {
		persistErr = s.parseResultPersister.PersistParseResult(ctx, task.ID, decoded)
	} else {
		persistErr = s.resultPersister.PersistGenerationResult(ctx, task.ID, decoded)
	}
	if persistErr != nil {
		_ = repo.ReleaseResultPersistence(ctx, task.ID)
		return fmt.Errorf("持久化任务结果失败: %w", persistErr)
	}
	if err := repo.CompleteResultPersistence(ctx, task.ID); err != nil {
		return fmt.Errorf("标记 generation 结果已持久化失败: %w", err)
	}
	now := time.Now().UTC()
	task.ResultPersistedAt = &now
	task.ResultPersistenceStartedAt = nil
	return nil
}

// MarkRunning 把任务切到运行态，供异步 worker 在真正开始消费前回写可观测状态。
func (s *Service) MarkRunning(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return wrapTaskLookupError(err)
	}
	if task.Status == JobTaskStatusCancelled {
		return nil
	}
	if err := s.repo.UpdateStatus(ctx, taskID, JobTaskStatusRunning, ""); err != nil {
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
	if task.Status == JobTaskStatusCancelled {
		return nil
	}
	normalizedResult := normalizeJSON(result)
	if err := s.repo.UpdateResult(ctx, taskID, normalizedResult); err != nil {
		return fmt.Errorf("更新任务结果失败: %w", err)
	}

	if s.resultPersister != nil && task.TaskType == taskTypeGeneration {
		var decoded map[string]any
		if err := json.Unmarshal(normalizedResult, &decoded); err != nil {
			return fmt.Errorf("解析任务结果失败: %w", err)
		}
		if err := s.resultPersister.PersistGenerationResult(ctx, taskID, decoded); err != nil {
			return fmt.Errorf("持久化 generation 结果失败: %w", err)
		}
	}
	if err := s.repo.UpdateStatus(ctx, taskID, JobTaskStatusSucceeded, ""); err != nil {
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
	if task.Status == JobTaskStatusCancelled {
		return nil
	}

	trimmedMessage := strings.TrimSpace(message)
	payload, err := json.Marshal(map[string]string{
		"status":  string(JobTaskStatusFailed),
		"message": trimmedMessage,
	})
	if err != nil {
		return fmt.Errorf("序列化任务失败事件失败: %w", err)
	}
	if err := s.repo.AppendEvent(ctx, &JobTaskEvent{
		TaskID:    taskID,
		EventType: "error",
		Status:    JobTaskStatusFailed,
		Payload:   datatypes.JSON(payload),
	}); err != nil {
		return fmt.Errorf("追加失败事件失败: %w", err)
	}
	if err := s.repo.UpdateStatus(ctx, taskID, JobTaskStatusFailed, trimmedMessage); err != nil {
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
	return task.Status == JobTaskStatusCancelled, nil
}

func validateCreateTaskInput(workspaceID uuid.UUID, taskSubtype string) error {
	taskSubtype = strings.TrimSpace(taskSubtype)
	if taskSubtype == "" {
		return ErrEmptyTaskSubtype
	}
	if workspaceID == uuid.Nil {
		return ErrEmptyWorkspaceID
	}
	return nil
}

func isTextbookTaskSubtype(taskSubtype string) bool {
	switch strings.TrimSpace(taskSubtype) {
	case sharedtextbook.TextbookSampleGenerationTaskSubtype,
		sharedtextbook.TextbookSourceImportTaskSubtype,
		sharedtextbook.TextbookOfficialWebImportTaskSubtype,
		sharedtextbook.TextbookTeachingArtifactVerifyTaskSubtype:
		return true
	default:
		return false
	}
}

func cloneUUID(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
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

func isTerminalStatus(status JobTaskStatus) bool {
	switch status {
	case JobTaskStatusSucceeded, JobTaskStatusFailed, JobTaskStatusCancelled:
		return true
	default:
		return false
	}
}
