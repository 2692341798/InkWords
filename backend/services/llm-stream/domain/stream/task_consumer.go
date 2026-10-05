package stream

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

const defaultTaskCancellationPollInterval = 500 * time.Millisecond

type taskService interface {
	MarkRunning(ctx context.Context, taskID uuid.UUID) error
	AppendEvent(ctx context.Context, taskID uuid.UUID, input AppendEventInput) error
	MarkSucceeded(ctx context.Context, taskID uuid.UUID, result []byte) error
	MarkFailed(ctx context.Context, taskID uuid.UUID, message string) error
	IsCancelled(ctx context.Context, taskID uuid.UUID) (bool, error)
}

type textbookWorkspaceTaskStore interface {
	TextbookWorkspaceMatches(context.Context, uuid.UUID, uuid.UUID, string) (bool, error)
}

type generationStreamService interface {
	Generate(ctx context.Context, workspaceID uuid.UUID, req GenerateRequest, chunkChan chan<- string, errChan chan<- error)
	BuildGenerateSingleTaskResult(ctx context.Context, req GenerateRequest, content string) ([]byte, error)
	BuildGenerateSeriesTaskResult(ctx context.Context, req GenerateRequest) ([]byte, error)
	BuildContinueTaskResult(ctx context.Context, workspaceID uuid.UUID, blogID uuid.UUID, appendedContent string) ([]byte, error)
	Continue(ctx context.Context, workspaceID uuid.UUID, blogID uuid.UUID, chunkChan chan<- string, errChan chan<- error)
	Polish(ctx context.Context, req PolishRequest, chunkChan chan<- string, errChan chan<- error)
}

// textbookSampleRunner is deliberately result-only: core-api remains the sole writer of textbook revisions.
type textbookSampleRunner interface {
	Run(context.Context, sharedtextbook.SampleGenerationTaskPayload) (sharedtextbook.SampleGenerationTaskResult, error)
}

type textbookSampleFailureEvidence interface {
	FailureResult() sharedtextbook.SampleGenerationTaskFailureResult
}

type taskFailureResultStore interface {
	MarkFailedWithResult(context.Context, uuid.UUID, string, []byte) error
}

// TaskConsumer 把 RabbitMQ 中的 generation task 转换成现有 stream.Service 的执行调用。
type TaskConsumer struct {
	tasks                    taskService
	streams                  generationStreamService
	textbookSample           textbookSampleRunner
	localWorkspaceID         uuid.UUID
	cancellationPollInterval time.Duration
}

// WithLocalWorkspace pins legacy blog task execution to the server-resolved
// installation workspace. Message workspace values may confirm but never
// override this identity.
func (c *TaskConsumer) WithLocalWorkspace(workspaceID uuid.UUID) *TaskConsumer {
	if c != nil {
		c.localWorkspaceID = workspaceID
	}
	return c
}

// WithTextbookSampleRunner enables the isolated textbook candidate path without changing legacy callers.
func (c *TaskConsumer) WithTextbookSampleRunner(runner textbookSampleRunner) *TaskConsumer {
	if c != nil {
		c.textbookSample = runner
	}
	return c
}

// NewTaskConsumer 通过依赖注入组装 llm-stream 使用的 generation worker consumer。

func NewTaskConsumer(tasks taskService, streams generationStreamService) *TaskConsumer {
	return &TaskConsumer{
		tasks:                    tasks,
		streams:                  streams,
		cancellationPollInterval: defaultTaskCancellationPollInterval,
	}
}

// HandleGenerationRequested 消费一条 generation.requested 消息并把执行结果回写到任务表。
//
//nolint:gocyclo
func (c *TaskConsumer) HandleGenerationRequested(ctx context.Context, message sharedrabbitmq.GenerationRequestedMessage) error {
	if c == nil || c.tasks == nil || c.streams == nil {
		return errors.New("task consumer dependencies are not configured")
	}
	if strings.TrimSpace(message.Kind) == sharedtextbook.TextbookSampleGenerationTaskSubtype {
		return c.handleTextbookSample(ctx, message)
	}

	if !supportsGenerationKind(message.Kind) {
		return c.tasks.MarkFailed(ctx, message.TaskID, fmt.Sprintf("unsupported generation kind: %s", strings.TrimSpace(message.Kind)))
	}
	workspaceID, err := c.resolveBlogWorkspace(message.WorkspaceID)
	if err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, err.Error())
	}

	normalizedMessage, err := normalizeGenerationMessage(message)
	if err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, err.Error())
	}
	message = normalizedMessage

	cancelled, err := c.tasks.IsCancelled(ctx, message.TaskID)
	if err != nil {
		return err
	}
	if cancelled {
		return nil
	}

	if err := c.tasks.MarkRunning(ctx, message.TaskID); err != nil {
		return err
	}

	taskCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go c.watchCancellation(taskCtx, cancel, message.TaskID)

	chunkChan, errChan := newGenerateStreamChannels()
	if err := c.startTaskStream(taskCtx, workspaceID, message, chunkChan, errChan); err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, err.Error())
	}

	chunkOpen, errOpen := true, true
	var fullContent strings.Builder
	for chunkOpen || errOpen {
		select {
		case <-taskCtx.Done():
			cancelled, cancelErr := c.tasks.IsCancelled(ctx, message.TaskID)
			if cancelErr != nil {
				return cancelErr
			}
			if cancelled {
				return nil
			}
			return taskCtx.Err()
		case err, ok := <-errChan:
			if !ok {
				errOpen = false
				errChan = nil
				continue
			}
			if err == nil {
				continue
			}
			cancelled, cancelErr := c.tasks.IsCancelled(ctx, message.TaskID)
			if cancelErr != nil {
				return cancelErr
			}
			if cancelled {
				return nil
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			return c.tasks.MarkFailed(ctx, message.TaskID, err.Error())
		case chunk, ok := <-chunkChan:
			if !ok {
				chunkOpen = false
				chunkChan = nil
				continue
			}
			fullContent.WriteString(chunk)
			if err := c.tasks.AppendEvent(ctx, message.TaskID, AppendEventInput{
				EventType: "chunk",
				Status:    TaskStatusStreaming,
				Payload:   buildTaskChunkPayload(chunk),
			}); err != nil {
				return err
			}
		}
	}

	result, err := c.buildFinalTaskResult(ctx, workspaceID, message, fullContent.String())
	if err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, err.Error())
	}

	return c.tasks.MarkSucceeded(ctx, message.TaskID, result)
}

func (c *TaskConsumer) handleTextbookSample(ctx context.Context, message sharedrabbitmq.GenerationRequestedMessage) error {
	valid, err := c.validateTextbookWorkspace(ctx, message.TaskID, message.WorkspaceID, message.Kind)
	if err != nil || !valid {
		return err
	}
	if c.textbookSample == nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, "textbook sample worker is not configured")
	}
	var payload sharedtextbook.SampleGenerationTaskPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, "invalid textbook sample payload")
	}
	if err := payload.Validate(); err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, "invalid textbook sample payload: "+err.Error())
	}
	cancelled, err := c.tasks.IsCancelled(ctx, message.TaskID)
	if err != nil {
		return err
	}
	if cancelled {
		return nil
	}
	stageKey := sharedtextbook.TaskStageInputHash(message.TaskID.String(), sharedtextbook.SampleGenerationStage, payload.InputHash)
	if reuse, ok := c.tasks.(textbookStageResultReuseStore); ok {
		cachedResult, found, reuseErr := reuse.FindCompletedTextbookStageResult(ctx, message.TaskID, stageKey)
		if reuseErr != nil {
			return reuseErr
		}
		if found {
			// A failed terminal-state write must not cause the already completed
			// model stage to run again. The checkpoint is scoped to this exact
			// task, stage, and frozen input; write its original result back so the
			// core-api reconciler can continue the normal candidate-persistence flow.
			return c.tasks.MarkSucceeded(ctx, message.TaskID, cachedResult)
		}
	}
	if err := c.tasks.MarkRunning(ctx, message.TaskID); err != nil {
		return err
	}
	if err := c.appendTextbookSampleEvent(ctx, message.TaskID, payload, "started", false, nil); err != nil {
		return err
	}
	taskCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go c.watchCancellation(taskCtx, cancel, message.TaskID)
	result, err := c.textbookSample.Run(taskCtx, payload)
	if err != nil {
		cancelled, cancelErr := c.tasks.IsCancelled(ctx, message.TaskID)
		if cancelErr != nil {
			return cancelErr
		}
		if cancelled || errors.Is(err, context.Canceled) {
			return nil
		}
		var rejected textbookSampleFailureEvidence
		if errors.As(err, &rejected) {
			failure := rejected.FailureResult()
			if validateErr := failure.ValidateAgainst(payload); validateErr != nil {
				return c.tasks.MarkFailed(ctx, message.TaskID, "invalid textbook sample failure telemetry")
			}
			encoded, marshalErr := json.Marshal(failure)
			if marshalErr != nil {
				return c.tasks.MarkFailed(ctx, message.TaskID, "marshal textbook sample failure telemetry")
			}
			if store, ok := c.tasks.(taskFailureResultStore); ok {
				return store.MarkFailedWithResult(ctx, message.TaskID, err.Error(), encoded)
			}
			if appendErr := c.appendTextbookSampleFailureEvent(ctx, message.TaskID, payload, encoded); appendErr != nil {
				return c.tasks.MarkFailed(ctx, message.TaskID, err.Error())
			}
		}
		return c.tasks.MarkFailed(ctx, message.TaskID, err.Error())
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, "marshal textbook sample result: "+err.Error())
	}
	if err := c.appendTextbookSampleEvent(ctx, message.TaskID, payload, "result_ready", true, encoded); err != nil {
		return err
	}
	return c.tasks.MarkSucceeded(ctx, message.TaskID, encoded)
}

func (c *TaskConsumer) appendTextbookSampleFailureEvent(ctx context.Context, taskID uuid.UUID, payload sharedtextbook.SampleGenerationTaskPayload, failure []byte) error {
	stageKey := sharedtextbook.TaskStageInputHash(taskID.String(), sharedtextbook.SampleGenerationStage, payload.InputHash)
	var failureResult json.RawMessage
	if err := json.Unmarshal(failure, &failureResult); err != nil {
		return fmt.Errorf("decode textbook sample failure evidence: %w", err)
	}
	eventPayload, err := json.Marshal(map[string]any{
		"project_id": payload.ProjectID, "chapter_id": payload.ChapterID, "stage": sharedtextbook.SampleGenerationStage,
		"checkpoint": "quality_rejected", "input_hash": payload.InputHash, "stage_execution_key": stageKey,
		"completed": false, "failure": failureResult,
	})
	if err != nil {
		return fmt.Errorf("marshal textbook sample failure event: %w", err)
	}
	return c.tasks.AppendEvent(ctx, taskID, AppendEventInput{EventType: "textbook_sample_phase", Status: TaskStatusRunning, Payload: eventPayload})
}

func (c *TaskConsumer) validateTextbookWorkspace(ctx context.Context, taskID uuid.UUID, workspaceID *uuid.UUID, taskSubtype string) (bool, error) {
	if workspaceID == nil || *workspaceID == uuid.Nil {
		return false, c.tasks.MarkFailed(ctx, taskID, "textbook task workspace identity is missing")
	}
	store, ok := c.tasks.(textbookWorkspaceTaskStore)
	if !ok {
		return false, errors.New("textbook task workspace store is not configured")
	}
	matches, err := store.TextbookWorkspaceMatches(ctx, taskID, *workspaceID, taskSubtype)
	if err != nil {
		return false, err
	}
	if !matches {
		return false, c.tasks.MarkFailed(ctx, taskID, "textbook task workspace identity does not match")
	}
	return true, nil
}

func (c *TaskConsumer) appendTextbookSampleEvent(ctx context.Context, taskID uuid.UUID, payload sharedtextbook.SampleGenerationTaskPayload, checkpoint string, completed bool, result []byte) error {
	stageKey := sharedtextbook.TaskStageInputHash(taskID.String(), sharedtextbook.SampleGenerationStage, payload.InputHash)
	event := map[string]any{
		"project_id": payload.ProjectID, "chapter_id": payload.ChapterID, "stage": sharedtextbook.SampleGenerationStage, "checkpoint": checkpoint,
		"input_hash": payload.InputHash, "stage_execution_key": stageKey, "output_hash": hashCourseContent(result), "completed": completed,
	}
	if completed {
		var stageResult json.RawMessage
		if err := json.Unmarshal(result, &stageResult); err != nil {
			return fmt.Errorf("decode textbook sample stage result: %w", err)
		}
		event["result"] = stageResult
	}
	eventPayload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal textbook sample event: %w", err)
	}
	return c.tasks.AppendEvent(ctx, taskID, AppendEventInput{EventType: "textbook_sample_phase", Status: TaskStatusRunning, Payload: eventPayload})
}

func hashCourseContent(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func (c *TaskConsumer) watchCancellation(taskCtx context.Context, cancel context.CancelFunc, taskID uuid.UUID) {
	ticker := time.NewTicker(c.cancellationPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-taskCtx.Done():
			return
		case <-ticker.C:
			cancelled, err := c.tasks.IsCancelled(taskCtx, taskID)
			if err != nil {
				continue
			}
			if cancelled {
				cancel()
				return
			}
		}
	}
}

func supportsGenerationKind(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "generate_single", "generate_series", "continue", "polish":
		return true
	default:
		return false
	}
}

func (c *TaskConsumer) buildFinalTaskResult(
	ctx context.Context,
	workspaceID uuid.UUID,
	message sharedrabbitmq.GenerationRequestedMessage,
	fullContent string,
) ([]byte, error) {
	switch strings.TrimSpace(message.Kind) {
	case "generate_single":
		var req GenerateRequest
		if err := json.Unmarshal(message.Payload, &req); err != nil {
			return nil, errors.New("invalid generation payload")
		}
		return c.streams.BuildGenerateSingleTaskResult(ctx, req, fullContent)
	case "generate_series":
		var req GenerateRequest
		if err := json.Unmarshal(message.Payload, &req); err != nil {
			return nil, errors.New("invalid generation payload")
		}
		return c.streams.BuildGenerateSeriesTaskResult(ctx, req)
	case "continue":
		var payload struct {
			BlogID string `json:"blog_id"`
		}
		if err := json.Unmarshal(message.Payload, &payload); err != nil {
			return nil, errors.New("invalid generation payload")
		}
		blogID, err := uuid.Parse(strings.TrimSpace(payload.BlogID))
		if err != nil {
			return nil, errors.New("invalid generation payload")
		}
		return c.streams.BuildContinueTaskResult(ctx, workspaceID, blogID, fullContent)
	default:
		return []byte(`{"done":true}`), nil
	}
}

func normalizeGenerationMessage(message sharedrabbitmq.GenerationRequestedMessage) (sharedrabbitmq.GenerationRequestedMessage, error) {
	kind := strings.TrimSpace(message.Kind)
	if kind != "generate_single" && kind != "generate_series" {
		return message, nil
	}

	var req GenerateRequest
	if err := json.Unmarshal(message.Payload, &req); err != nil {
		return message, errors.New("invalid generation payload")
	}
	req = req.Normalize()

	if kind != "generate_series" {
		normalizedPayload, err := json.Marshal(req)
		if err != nil {
			return message, errors.New("invalid generation payload")
		}
		message.Payload = normalizedPayload
		return message, nil
	}
	if strings.TrimSpace(req.ParentID) == "" {
		req.ParentID = uuid.NewString()
	}

	normalizedPayload, err := json.Marshal(req)
	if err != nil {
		return message, errors.New("invalid generation payload")
	}
	message.Payload = normalizedPayload
	return message, nil
}

func (c *TaskConsumer) startTaskStream(
	taskCtx context.Context,
	workspaceID uuid.UUID,
	message sharedrabbitmq.GenerationRequestedMessage,
	chunkChan chan<- string,
	errChan chan<- error,
) error {
	switch strings.TrimSpace(message.Kind) {
	case "generate_single", "generate_series":
		var req GenerateRequest
		if err := json.Unmarshal(message.Payload, &req); err != nil {
			return errors.New("invalid generation payload")
		}
		go c.streams.Generate(taskCtx, workspaceID, req, chunkChan, errChan)
		return nil
	case "continue":
		var payload struct {
			BlogID string `json:"blog_id"`
		}
		if err := json.Unmarshal(message.Payload, &payload); err != nil {
			return errors.New("invalid generation payload")
		}
		blogID, err := uuid.Parse(strings.TrimSpace(payload.BlogID))
		if err != nil {
			return errors.New("invalid generation payload")
		}
		go c.streams.Continue(taskCtx, workspaceID, blogID, chunkChan, errChan)
		return nil
	case "polish":
		var payload struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(message.Payload, &payload); err != nil {
			return errors.New("invalid generation payload")
		}
		if strings.TrimSpace(payload.Content) == "" {
			return errors.New("invalid generation payload")
		}
		go c.streams.Polish(taskCtx, PolishRequest{
			Title:   payload.Title,
			Content: payload.Content,
		}, chunkChan, errChan)
		return nil
	default:
		return fmt.Errorf("unsupported generation kind: %s", strings.TrimSpace(message.Kind))
	}
}

func (c *TaskConsumer) resolveBlogWorkspace(messageWorkspaceID *uuid.UUID) (uuid.UUID, error) {
	if c.localWorkspaceID == uuid.Nil {
		return uuid.Nil, errors.New("local workspace is not configured for blog task")
	}
	if messageWorkspaceID != nil && *messageWorkspaceID != c.localWorkspaceID {
		return uuid.Nil, errors.New("blog task workspace does not match the local installation")
	}
	return c.localWorkspaceID, nil
}

// Why: 任务事件表使用 jsonb 存储 payload，若直接写入纯文本 chunk 会在持久化层被吞成空对象；
// 这里对结构化 chunk 原样透传，对纯文本 chunk 包一层 content，兼顾现有系列流与最小持久化兼容。
func buildTaskChunkPayload(chunk string) []byte {
	trimmed := strings.TrimSpace(chunk)
	if trimmed != "" && json.Valid([]byte(trimmed)) {
		return []byte(trimmed)
	}

	payload, err := json.Marshal(map[string]string{"content": chunk})
	if err != nil {
		return []byte(`{"content":""}`)
	}
	return payload
}
