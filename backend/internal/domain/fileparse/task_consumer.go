package fileparse

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"

	taskdomain "inkwords-backend/internal/domain/task"
	"inkwords-backend/internal/infra/mq"
)

type parseTaskService interface {
	MarkRunning(ctx context.Context, taskID uuid.UUID) error
	AppendEvent(ctx context.Context, taskID uuid.UUID, input taskdomain.AppendEventInput) error
	MarkSucceeded(ctx context.Context, taskID uuid.UUID, result []byte) error
	MarkFailed(ctx context.Context, taskID uuid.UUID, message string) error
	IsCancelled(ctx context.Context, taskID uuid.UUID) (bool, error)
}

type parseExecutor interface {
	Parse(src io.Reader, filename string) (ParseResult, error)
}

// TaskConsumer 把 RabbitMQ 中的 parse task 转换成现有 fileparse.Service 的执行调用。
type TaskConsumer struct {
	tasks  parseTaskService
	parser parseExecutor
}

// NewTaskConsumer 通过依赖注入组装 parser-service 使用的 parse worker consumer。
func NewTaskConsumer(tasks parseTaskService, parser parseExecutor) *TaskConsumer {
	return &TaskConsumer{
		tasks:  tasks,
		parser: parser,
	}
}

type parsePayload struct {
	Filename string
	Content  []byte
}

// HandleParseRequested 消费一条 parse.requested 消息并把解析结果回写到任务表。
func (c *TaskConsumer) HandleParseRequested(ctx context.Context, message mq.ParseRequestedMessage) error {
	if c == nil || c.tasks == nil || c.parser == nil {
		return errors.New("parse task consumer dependencies are not configured")
	}

	payload, err := decodeParsePayload(message.Payload)
	if err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, "invalid parse payload")
	}

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

	result, err := c.parser.Parse(bytes.NewReader(payload.Content), payload.Filename)
	if err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, err.Error())
	}

	resultJSON, err := buildParseResultJSON(result)
	if err != nil {
		return fmt.Errorf("marshal parse result failed: %w", err)
	}
	return c.tasks.MarkSucceeded(ctx, message.TaskID, resultJSON)
}

func decodeParsePayload(raw []byte) (parsePayload, error) {
	var payload struct {
		Filename      string `json:"filename"`
		ContentBase64 string `json:"content_base64"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return parsePayload{}, errors.New("invalid parse payload")
	}
	if strings.TrimSpace(payload.Filename) == "" {
		return parsePayload{}, errors.New("invalid parse payload")
	}

	content, err := base64.StdEncoding.DecodeString(payload.ContentBase64)
	if err != nil {
		return parsePayload{}, errors.New("invalid parse payload")
	}
	return parsePayload{
		Filename: strings.TrimSpace(payload.Filename),
		Content:  content,
	}, nil
}

func buildParseResultJSON(result ParseResult) ([]byte, error) {
	payload := map[string]any{
		"source_content": result.SourceContent,
	}
	if result.ArchiveSummary != nil {
		payload["archive_summary"] = result.ArchiveSummary
	}
	return json.Marshal(payload)
}
