package parse

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"

	sharedmq "inkwords-backend/shared/platform/rabbitmq"
)

type parseTaskService interface {
	MarkRunning(ctx context.Context, taskID uuid.UUID) error
	MarkSucceeded(ctx context.Context, taskID uuid.UUID, result []byte) error
	MarkFailed(ctx context.Context, taskID uuid.UUID, message string) error
	IsCancelled(ctx context.Context, taskID uuid.UUID) (bool, error)
}

type parseExecutor interface {
	Parse(src io.Reader, filename string) (ParseResult, error)
}

// TaskConsumer converts RabbitMQ parse tasks into service-owned parse executions.
type TaskConsumer struct {
	tasks  parseTaskService
	parser parseExecutor
}

// NewTaskConsumer wires the parser-service parse worker with task persistence and parse execution.
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

// StartParseConsumer boots the parser-service RabbitMQ worker for parse.requested messages.
func StartParseConsumer(ctx context.Context, taskService parseTaskService, parseService *Service) (func(), error) {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Println("RabbitMQ is not configured, parse consumer disabled")
		return func() {}, nil
	}

	conn, channel, err := sharedmq.Dial(rabbitURL)
	if err != nil {
		return func() {}, err
	}

	exchangeName := envOrDefault("RABBITMQ_EXCHANGE", "inkwords.events")
	queueName := envOrDefault("RABBITMQ_PARSE_QUEUE", "inkwords.parse")
	routingKey := sharedmq.ParseRequestedMessage{}.RoutingKey()

	if err := channel.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	queue, err := channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	if err := channel.QueueBind(queue.Name, routingKey, exchangeName, false, nil); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	deliveries, err := channel.Consume(queue.Name, "parser-service-parse-worker", false, false, false, false, nil)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	consumer := NewTaskConsumer(taskService, parseService)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case delivery, ok := <-deliveries:
				if !ok {
					return
				}

				var message sharedmq.ParseRequestedMessage
				if err := json.Unmarshal(delivery.Body, &message); err != nil {
					log.Printf("invalid parse message payload: %v", err)
					_ = delivery.Ack(false)
					continue
				}

				if err := consumer.HandleParseRequested(ctx, message); err != nil {
					log.Printf("parse task handling failed for %s: %v", message.TaskID, err)
					_ = delivery.Nack(false, true)
					continue
				}

				_ = delivery.Ack(false)
			}
		}
	}()

	return func() {
		_ = channel.Close()
		_ = conn.Close()
	}, nil
}

// HandleParseRequested consumes one parse.requested message and persists the parse result to the task store.
func (c *TaskConsumer) HandleParseRequested(ctx context.Context, message sharedmq.ParseRequestedMessage) error {
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

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
