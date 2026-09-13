package mq

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

type publishChannel interface {
	PublishWithContext(
		ctx context.Context,
		exchange string,
		key string,
		mandatory bool,
		immediate bool,
		msg amqp.Publishing,
	) error
}

// Publisher maps core-api task domain messages onto RabbitMQ envelopes.
type Publisher struct {
	channel  publishChannel
	exchange string
}

// NewPublisher builds a RabbitMQ publisher for core-api task creation.
func NewPublisher(channel *amqp.Channel, exchange string) *Publisher {
	return &Publisher{
		channel:  channel,
		exchange: exchange,
	}
}

// PublishGenerationRequested publishes a generation task request.
func (p *Publisher) PublishGenerationRequested(ctx context.Context, message sharedrabbitmq.GenerationRequestedMessage) error {
	if p == nil || p.channel == nil {
		return errors.New("rabbitmq publisher channel is nil")
	}

	envelope := sharedrabbitmq.GenerationRequestedMessage{
		TaskID:      message.TaskID,
		Kind:        message.Kind,
		WorkspaceID: cloneWorkspaceID(message.WorkspaceID),
		Payload:     append(json.RawMessage(nil), message.Payload...),
	}

	return p.publish(ctx, envelope.RoutingKey(), envelope)
}

// PublishTextbookGenerationRequested uses the existing generation routing key
// while keeping legacy user identity out of the textbook envelope.
func (p *Publisher) PublishTextbookGenerationRequested(ctx context.Context, message sharedrabbitmq.TextbookGenerationRequestedMessage) error {
	if p == nil || p.channel == nil {
		return errors.New("rabbitmq publisher channel is nil")
	}
	envelope := sharedrabbitmq.TextbookGenerationRequestedMessage{
		TaskID: message.TaskID, Kind: message.Kind, WorkspaceID: cloneWorkspaceID(message.WorkspaceID), Payload: append(json.RawMessage(nil), message.Payload...),
	}
	return p.publish(ctx, envelope.RoutingKey(), envelope)
}

// PublishParseRequested publishes a parse task request.
func (p *Publisher) PublishParseRequested(ctx context.Context, message sharedrabbitmq.ParseRequestedMessage) error {
	if p == nil || p.channel == nil {
		return errors.New("rabbitmq publisher channel is nil")
	}

	envelope := sharedrabbitmq.ParseRequestedMessage{
		TaskID:      message.TaskID,
		Kind:        message.Kind,
		WorkspaceID: cloneWorkspaceID(message.WorkspaceID),
		Payload:     append(json.RawMessage(nil), message.Payload...),
	}

	return p.publish(ctx, envelope.RoutingKey(), envelope)
}

// PublishTextbookParseRequested uses the existing parser routing key while
// keeping legacy user identity out of the textbook envelope.
func (p *Publisher) PublishTextbookParseRequested(ctx context.Context, message sharedrabbitmq.TextbookParseRequestedMessage) error {
	if p == nil || p.channel == nil {
		return errors.New("rabbitmq publisher channel is nil")
	}
	envelope := sharedrabbitmq.TextbookParseRequestedMessage{
		TaskID: message.TaskID, Kind: message.Kind, WorkspaceID: cloneWorkspaceID(message.WorkspaceID), Payload: append(json.RawMessage(nil), message.Payload...),
	}
	return p.publish(ctx, envelope.RoutingKey(), envelope)
}

// PublishExportRequested publishes an export task request.
func (p *Publisher) PublishExportRequested(ctx context.Context, message sharedrabbitmq.ExportRequestedMessage) error {
	if p == nil || p.channel == nil {
		return errors.New("rabbitmq publisher channel is nil")
	}

	envelope := sharedrabbitmq.ExportRequestedMessage{
		TaskID: message.TaskID, Kind: message.Kind,
		WorkspaceID: cloneWorkspaceID(message.WorkspaceID), Payload: append(json.RawMessage(nil), message.Payload...),
	}

	return p.publish(ctx, envelope.RoutingKey(), envelope)
}

// PublishTextbookVerificationRequested publishes a separate, immutable
// textbook-artifact verification request. It must not share the legacy course
// verification queue because their manifest and trust contracts differ.
func (p *Publisher) PublishTextbookVerificationRequested(ctx context.Context, message sharedrabbitmq.TextbookVerificationRequestedMessage) error {
	if p == nil || p.channel == nil {
		return errors.New("rabbitmq publisher channel is nil")
	}
	envelope := sharedrabbitmq.TextbookVerificationRequestedMessage{
		TaskID:      message.TaskID,
		Kind:        message.Kind,
		WorkspaceID: cloneWorkspaceID(message.WorkspaceID),
		Payload:     append(json.RawMessage(nil), message.Payload...),
	}
	return p.publish(ctx, envelope.RoutingKey(), envelope)
}

func cloneWorkspaceID(workspaceID *uuid.UUID) *uuid.UUID {
	if workspaceID == nil {
		return nil
	}
	copy := *workspaceID
	return &copy
}

func (p *Publisher) publish(ctx context.Context, routingKey string, envelope any) error {
	body, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	return p.channel.PublishWithContext(
		ctx,
		p.exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}
