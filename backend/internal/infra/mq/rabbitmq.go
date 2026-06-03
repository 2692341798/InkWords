package mq

import (
	"context"
	"encoding/json"
	"errors"

	amqp "github.com/rabbitmq/amqp091-go"

	taskdomain "inkwords-backend/internal/domain/task"
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

// Publisher 负责把任务领域消息转换为 RabbitMQ 消息并持久化发布出去。
type Publisher struct {
	channel  publishChannel
	exchange string
}

// NewPublisher 通过依赖注入组装 RabbitMQ 发布器。
func NewPublisher(channel *amqp.Channel, exchange string) *Publisher {
	return &Publisher{
		channel:  channel,
		exchange: exchange,
	}
}

// PublishGenerationRequested 把任务领域消息映射为队列契约，避免领域层直接依赖具体 MQ 实现细节。
func (p *Publisher) PublishGenerationRequested(ctx context.Context, message taskdomain.GenerationRequestedMessage) error {
	if p == nil || p.channel == nil {
		return errors.New("rabbitmq publisher channel is nil")
	}

	envelope := GenerationRequestedMessage{
		TaskID:  message.TaskID,
		Kind:    message.Kind,
		UserID:  message.UserID,
		Payload: append(json.RawMessage(nil), message.Payload...),
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	return p.channel.PublishWithContext(
		ctx,
		p.exchange,
		envelope.RoutingKey(),
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

// PublishParseRequested 把解析任务领域消息映射为队列契约，避免领域层直接依赖具体 MQ 实现细节。
func (p *Publisher) PublishParseRequested(ctx context.Context, message taskdomain.ParseRequestedMessage) error {
	if p == nil || p.channel == nil {
		return errors.New("rabbitmq publisher channel is nil")
	}

	envelope := ParseRequestedMessage{
		TaskID:  message.TaskID,
		Kind:    message.Kind,
		UserID:  message.UserID,
		Payload: append(json.RawMessage(nil), message.Payload...),
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	return p.channel.PublishWithContext(
		ctx,
		p.exchange,
		envelope.RoutingKey(),
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

// PublishExportRequested 把导出任务领域消息映射为队列契约，避免领域层直接依赖具体 MQ 实现细节。
func (p *Publisher) PublishExportRequested(ctx context.Context, message taskdomain.ExportRequestedMessage) error {
	if p == nil || p.channel == nil {
		return errors.New("rabbitmq publisher channel is nil")
	}

	envelope := ExportRequestedMessage{
		TaskID:  message.TaskID,
		Kind:    message.Kind,
		UserID:  message.UserID,
		Payload: append(json.RawMessage(nil), message.Payload...),
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	return p.channel.PublishWithContext(
		ctx,
		p.exchange,
		envelope.RoutingKey(),
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

var _ taskdomain.Publisher = (*Publisher)(nil)
