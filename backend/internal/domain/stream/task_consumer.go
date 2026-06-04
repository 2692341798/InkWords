package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	taskdomain "inkwords-backend/internal/domain/task"
	"inkwords-backend/internal/infra/mq"
	"inkwords-backend/internal/model"
)

const defaultTaskCancellationPollInterval = 500 * time.Millisecond

type taskService interface {
	MarkRunning(ctx context.Context, taskID uuid.UUID) error
	AppendEvent(ctx context.Context, taskID uuid.UUID, input taskdomain.AppendEventInput) error
	MarkSucceeded(ctx context.Context, taskID uuid.UUID, result []byte) error
	MarkFailed(ctx context.Context, taskID uuid.UUID, message string) error
	IsCancelled(ctx context.Context, taskID uuid.UUID) (bool, error)
}

type generationStreamService interface {
	Generate(ctx context.Context, userID uuid.UUID, req GenerateRequest, chunkChan chan<- string, errChan chan<- error)
	BuildGenerateSingleTaskResult(ctx context.Context, req GenerateRequest, content string) ([]byte, error)
	BuildGenerateSeriesTaskResult(ctx context.Context, req GenerateRequest) ([]byte, error)
	BuildContinueTaskResult(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, appendedContent string) ([]byte, error)
	Continue(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, chunkChan chan<- string, errChan chan<- error)
	Polish(ctx context.Context, req PolishRequest, chunkChan chan<- string, errChan chan<- error)
}

// TaskConsumer 把 RabbitMQ 中的 generation task 转换成现有 stream.Service 的执行调用。
type TaskConsumer struct {
	tasks                    taskService
	streams                  generationStreamService
	cancellationPollInterval time.Duration
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
func (c *TaskConsumer) HandleGenerationRequested(ctx context.Context, message mq.GenerationRequestedMessage) error {
	if c == nil || c.tasks == nil || c.streams == nil {
		return errors.New("task consumer dependencies are not configured")
	}

	if !supportsGenerationKind(message.Kind) {
		return c.tasks.MarkFailed(ctx, message.TaskID, fmt.Sprintf("unsupported generation kind: %s", strings.TrimSpace(message.Kind)))
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
	if err := c.startTaskStream(taskCtx, message, chunkChan, errChan); err != nil {
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
			if err := c.tasks.AppendEvent(ctx, message.TaskID, taskdomain.AppendEventInput{
				EventType: "chunk",
				Status:    model.JobTaskStatusStreaming,
				Payload:   buildTaskChunkPayload(chunk),
			}); err != nil {
				return err
			}
		}
	}

	result, err := c.buildFinalTaskResult(ctx, message, fullContent.String())
	if err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, err.Error())
	}

	return c.tasks.MarkSucceeded(ctx, message.TaskID, result)
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
	message mq.GenerationRequestedMessage,
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
		return c.streams.BuildContinueTaskResult(ctx, message.UserID, blogID, fullContent)
	default:
		return []byte(`{"done":true}`), nil
	}
}

func normalizeGenerationMessage(message mq.GenerationRequestedMessage) (mq.GenerationRequestedMessage, error) {
	if strings.TrimSpace(message.Kind) != "generate_series" {
		return message, nil
	}

	var req GenerateRequest
	if err := json.Unmarshal(message.Payload, &req); err != nil {
		return message, errors.New("invalid generation payload")
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
	message mq.GenerationRequestedMessage,
	chunkChan chan<- string,
	errChan chan<- error,
) error {
	switch strings.TrimSpace(message.Kind) {
	case "generate_single", "generate_series":
		var req GenerateRequest
		if err := json.Unmarshal(message.Payload, &req); err != nil {
			return errors.New("invalid generation payload")
		}
		go c.streams.Generate(taskCtx, message.UserID, req, chunkChan, errChan)
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
		go c.streams.Continue(taskCtx, message.UserID, blogID, chunkChan, errChan)
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
