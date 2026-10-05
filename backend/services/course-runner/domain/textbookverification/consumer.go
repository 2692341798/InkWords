package textbookverification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

const TaskSubtype = sharedtextbook.TextbookTeachingArtifactVerifyTaskSubtype

type VerificationPayload = sharedtextbook.ArtifactVerificationRequest

type ArtifactResolver interface {
	Resolve(context.Context, VerificationPayload) (RunRequest, error)
}

type TaskStore interface {
	ClaimVerificationWorker(context.Context, uuid.UUID) (uuid.UUID, error)
	ReleaseVerificationWorker(context.Context, uuid.UUID, uuid.UUID) error
	MarkSucceeded(context.Context, uuid.UUID, []byte) error
	MarkFailed(context.Context, uuid.UUID, string) error
	IsCancelled(context.Context, uuid.UUID) (bool, error)
	TextbookWorkspaceMatches(context.Context, uuid.UUID, uuid.UUID, string) (bool, error)
}

// EvidencePersister is separate from task state: a completed worker can
// truthfully report `unverified` while still leaving an immutable observation
// record explaining why no passing claim exists.
type EvidencePersister interface {
	PersistVerificationReport(context.Context, uuid.UUID, VerificationPayload, Report) error
}

type Consumer struct {
	tasks    TaskStore
	resolver ArtifactResolver
	verifier Runner
	evidence EvidencePersister
}

func NewConsumer(tasks TaskStore, resolver ArtifactResolver, verifier Runner, evidence EvidencePersister) *Consumer {
	return &Consumer{tasks: tasks, resolver: resolver, verifier: verifier, evidence: evidence}
}

func (consumer *Consumer) HandleVerificationRequested(ctx context.Context, message sharedrabbitmq.TextbookVerificationRequestedMessage) (resultErr error) {
	if consumer == nil || consumer.tasks == nil || consumer.resolver == nil || consumer.evidence == nil {
		return errors.New("textbook verification consumer dependencies are not configured")
	}
	if message.Kind != TaskSubtype {
		return consumer.tasks.MarkFailed(ctx, message.TaskID, "unsupported textbook verification kind")
	}
	if message.WorkspaceID == nil || *message.WorkspaceID == uuid.Nil {
		return consumer.tasks.MarkFailed(ctx, message.TaskID, "textbook task workspace identity is missing")
	}
	matches, err := consumer.tasks.TextbookWorkspaceMatches(ctx, message.TaskID, *message.WorkspaceID, message.Kind)
	if err != nil {
		return err
	}
	if !matches {
		return consumer.tasks.MarkFailed(ctx, message.TaskID, "textbook task workspace identity does not match")
	}
	var payload VerificationPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil || payload.Validate() != nil {
		return consumer.tasks.MarkFailed(ctx, message.TaskID, "invalid textbook verification payload")
	}
	cancelled, err := consumer.tasks.IsCancelled(ctx, message.TaskID)
	if err != nil {
		return err
	}
	if cancelled {
		return nil
	}
	token, err := consumer.tasks.ClaimVerificationWorker(ctx, message.TaskID)
	if err != nil {
		if errors.Is(err, ErrVerificationTaskTerminal) {
			return nil
		}
		return err
	}
	// Installed before the watcher cleanup so release cannot precede executor
	// and observer exit. Cancellation of the work must not cancel this receipt.
	defer func() {
		releaseCtx, cancelRelease := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancelRelease()
		resultErr = errors.Join(resultErr, consumer.tasks.ReleaseVerificationWorker(releaseCtx, message.TaskID, token))
	}()
	workCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	watchCtx, stopWatching := context.WithCancel(workCtx)
	done := make(chan struct{})
	go func() { defer close(done); consumer.observeCancellation(watchCtx, cancel, message.TaskID) }()
	defer func() { stopWatching(); <-done }()
	request, err := consumer.resolver.Resolve(workCtx, payload)
	if err != nil {
		if cause := context.Cause(workCtx); cause != nil {
			if errors.Is(cause, ErrVerificationCancelled) {
				return nil
			}
			return cause
		}
		return consumer.tasks.MarkFailed(ctx, message.TaskID, err.Error())
	}
	report := consumer.verifier.Verify(workCtx, request)
	stopWatching()
	<-done
	if cause := context.Cause(workCtx); cause != nil {
		if errors.Is(cause, ErrVerificationCancelled) {
			return nil
		}
		return cause
	}
	// Cover a fast completion before the first polling tick. The persistence
	// adapter must also serialize final evidence against task cancellation.
	cancelled, err = consumer.tasks.IsCancelled(ctx, message.TaskID)
	if err != nil {
		return err
	}
	if cancelled {
		return nil
	}
	if err := consumer.evidence.PersistVerificationReport(ctx, message.TaskID, payload, report); err != nil {
		if errors.Is(err, ErrVerificationCancelled) || errors.Is(err, ErrVerificationTaskTerminal) {
			return nil
		}
		return err
	}
	body, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal textbook verification report: %w", err)
	}
	return consumer.tasks.MarkSucceeded(ctx, message.TaskID, body)
}

type deliveryAcknowledger interface {
	Ack(multiple bool) error
	Nack(multiple bool, requeue bool) error
}

func (consumer *Consumer) ConsumeMessage(ctx context.Context, body []byte, ack deliveryAcknowledger) error {
	var message sharedrabbitmq.TextbookVerificationRequestedMessage
	if err := json.Unmarshal(body, &message); err != nil {
		if ackErr := ack.Ack(false); ackErr != nil {
			return fmt.Errorf("ack malformed textbook verification message: %w", ackErr)
		}
		return nil
	}
	if err := consumer.HandleVerificationRequested(ctx, message); err != nil {
		if nackErr := ack.Nack(false, true); nackErr != nil {
			return fmt.Errorf("nack textbook verification task: %w (work: %w)", nackErr, err)
		}
		return nil
	}
	return ack.Ack(false)
}

// StartConsumer intentionally stays disabled until the platform has an
// explicitly enabled Linux bwrap runner. The caller can still start legacy
// course verification independently; their queues never overlap.
func StartConsumer(ctx context.Context, consumer *Consumer, queueName string) (func(), error) {
	if os.Getenv("TEXTBOOK_TEACHING_ARTIFACT_VERIFICATION_ENABLED") != "true" {
		log.Println("textbook teaching-artifact verification disabled")
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
	if err := channel.QueueBind(queue.Name, sharedrabbitmq.TextbookVerificationRequestedMessage{}.RoutingKey(), exchange, false, nil); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}
	deliveries, err := channel.Consume(queue.Name, "inkwords-textbook-runner", false, false, false, false, nil)
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
					log.Printf("textbook verification consume failed: %v", err)
				}
			}
		}
	}()
	return func() { _ = channel.Close(); _ = conn.Close() }, nil
}
