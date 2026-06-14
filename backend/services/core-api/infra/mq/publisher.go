package mq

import (
	"context"
	"encoding/json"
	"errors"

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
		TaskID:  message.TaskID,
		Kind:    message.Kind,
		UserID:  message.UserID,
		Payload: append(json.RawMessage(nil), message.Payload...),
	}

	return p.publish(ctx, envelope.RoutingKey(), envelope)
}

// PublishParseRequested publishes a parse task request.
func (p *Publisher) PublishParseRequested(ctx context.Context, message sharedrabbitmq.ParseRequestedMessage) error {
	if p == nil || p.channel == nil {
		return errors.New("rabbitmq publisher channel is nil")
	}

	envelope := sharedrabbitmq.ParseRequestedMessage{
		TaskID:  message.TaskID,
		Kind:    message.Kind,
		UserID:  message.UserID,
		Payload: append(json.RawMessage(nil), message.Payload...),
	}

	return p.publish(ctx, envelope.RoutingKey(), envelope)
}

// PublishExportRequested publishes an export task request.
func (p *Publisher) PublishExportRequested(ctx context.Context, message sharedrabbitmq.ExportRequestedMessage) error {
	if p == nil || p.channel == nil {
		return errors.New("rabbitmq publisher channel is nil")
	}

	envelope := sharedrabbitmq.ExportRequestedMessage{
		TaskID:  message.TaskID,
		Kind:    message.Kind,
		UserID:  message.UserID,
		Payload: append(json.RawMessage(nil), message.Payload...),
	}

	return p.publish(ctx, envelope.RoutingKey(), envelope)
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
