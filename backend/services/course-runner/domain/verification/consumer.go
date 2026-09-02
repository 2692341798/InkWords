package verification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

const TaskSubtype = "project_course_lab_verify"

type VerificationPayload struct {
	CourseID         uuid.UUID `json:"course_id"`
	BlueprintVersion int       `json:"blueprint_version"`
	ChapterID        string    `json:"chapter_id"`
	ArtifactToken    string    `json:"artifact_token"`
}

type ArtifactResolver interface {
	Resolve(ctx context.Context, payload VerificationPayload) (RunRequest, error)
}

type TaskStore interface {
	MarkRunning(ctx context.Context, taskID uuid.UUID) error
	MarkSucceeded(ctx context.Context, taskID uuid.UUID, result []byte) error
	MarkFailed(ctx context.Context, taskID uuid.UUID, message string) error
	IsCancelled(ctx context.Context, taskID uuid.UUID) (bool, error)
}

type Verifier interface {
	Verify(ctx context.Context, request RunRequest) Report
}

type Consumer struct {
	tasks    TaskStore
	resolver ArtifactResolver
	verifier Verifier
}

func NewConsumer(tasks TaskStore, resolver ArtifactResolver, verifier Verifier) *Consumer {
	return &Consumer{tasks: tasks, resolver: resolver, verifier: verifier}
}

func (c *Consumer) HandleVerificationRequested(ctx context.Context, message sharedrabbitmq.VerificationRequestedMessage) error {
	if c == nil || c.tasks == nil || c.resolver == nil || c.verifier == nil {
		return errors.New("verification consumer dependencies are not configured")
	}
	if message.Kind != TaskSubtype {
		return c.tasks.MarkFailed(ctx, message.TaskID, "unsupported verification kind")
	}
	var payload VerificationPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil || payload.CourseID == uuid.Nil || payload.BlueprintVersion < 1 || payload.ChapterID == "" || payload.ArtifactToken == "" {
		return c.tasks.MarkFailed(ctx, message.TaskID, "invalid verification payload")
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
	request, err := c.resolver.Resolve(ctx, payload)
	if err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, err.Error())
	}
	report := c.verifier.Verify(ctx, request)
	body, err := json.Marshal(report)
	if err != nil {
		return err
	}
	return c.tasks.MarkSucceeded(ctx, message.TaskID, body)
}

type deliveryAcknowledger interface {
	Ack(multiple bool) error
	Nack(multiple bool, requeue bool) error
}

func (c *Consumer) ConsumeMessage(ctx context.Context, body []byte, ack deliveryAcknowledger) error {
	var message sharedrabbitmq.VerificationRequestedMessage
	if err := json.Unmarshal(body, &message); err != nil {
		if ackErr := ack.Ack(false); ackErr != nil {
			return fmt.Errorf("ack malformed verification message: %w", ackErr)
		}
		return nil
	}
	if err := c.HandleVerificationRequested(ctx, message); err != nil {
		if nackErr := ack.Nack(false, true); nackErr != nil {
			return fmt.Errorf("nack verification task: %w (work: %w)", nackErr, err)
		}
		return nil
	}
	return ack.Ack(false)
}

// StartVerificationConsumer is disabled unless explicitly enabled. A missing
// sandbox must fail closed rather than silently running labs in this service.
func StartVerificationConsumer(ctx context.Context, consumer *Consumer, queueName string) (func(), error) {
	if os.Getenv("PROJECT_COURSE_LAB_VERIFICATION_ENABLED") != "true" {
		log.Println("project course lab verification disabled")
		return func() {}, nil
	}
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		return func() {}, errors.New("RABBITMQ_URL is not configured")
	}
	conn, channel, err := sharedrabbitmq.Dial(url)
	if err != nil {
		return func() {}, err
	}
	exchange := os.Getenv("RABBITMQ_EXCHANGE")
	if exchange == "" {
		exchange = "inkwords.events"
	}
	if err := channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
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
	if err := channel.QueueBind(queue.Name, sharedrabbitmq.VerificationRequestedMessage{}.RoutingKey(), exchange, false, nil); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}
	deliveries, err := channel.Consume(queue.Name, "inkwords-course-runner", false, false, false, false, nil)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case delivery, ok := <-deliveries:
				if !ok {
					return
				}
				if err := consumer.ConsumeMessage(ctx, delivery.Body, delivery); err != nil {
					log.Printf("course verification consume failed: %v", err)
				}
			}
		}
	}()
	return func() { _ = channel.Close(); _ = conn.Close() }, nil
}
