package export

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

// deliveryAcknowledger 将 RabbitMQ amqp.Delivery 的 Ack/Nack 抽象为接口，使无 broker 环境也可测试。
type deliveryAcknowledger interface {
	Ack(multiple bool) error
	Nack(multiple bool, requeue bool) error
}

// ConsumeMessage 消费一条原始消息体，负责反序列化、业务处理、以及投递确认（Ack/Nack）。
func (c *Consumer) ConsumeMessage(ctx context.Context, body []byte, ack deliveryAcknowledger) error {
	var message sharedrabbitmq.ExportRequestedMessage
	if err := json.Unmarshal(body, &message); err != nil {
		log.Printf("invalid export message payload: %v", err)
		if ackErr := ack.Ack(false); ackErr != nil {
			return fmt.Errorf("ack malformed export message: %w", ackErr)
		}
		return nil
	}

	if err := c.HandleExportRequested(ctx, RequestedMessage(message)); err != nil {
		log.Printf("export task handling failed for %s: %v", message.TaskID, err)
		if nackErr := ack.Nack(false, true); nackErr != nil {
			return fmt.Errorf("nack for export task %s: %w (work: %w)", message.TaskID, nackErr, err)
		}
		return nil
	}

	if err := ack.Ack(false); err != nil {
		return fmt.Errorf("ack for export task %s: %w", message.TaskID, err)
	}
	return nil
}

// StartExportConsumer starts the export worker on the service-owned RabbitMQ queue.
func StartExportConsumer(ctx context.Context, consumer *Consumer, queueName string) (func(), error) {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Println("RabbitMQ is not configured, export consumer disabled")
		return func() {}, nil
	}

	conn, channel, err := sharedrabbitmq.Dial(rabbitURL)
	if err != nil {
		return func() {}, err
	}

	exchangeName := envOrDefault("RABBITMQ_EXCHANGE", "inkwords.events")
	routingKey := sharedrabbitmq.ExportRequestedMessage{}.RoutingKey()

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

	deliveries, err := channel.Consume(queue.Name, "export-service-pdf-worker", false, false, false, false, nil)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	// Why: consumer loop和 HTTP server 生命周期必须一致，避免部署重启时残留孤儿 goroutine。
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
					log.Printf("consume export message failed: %v", err)
				}
			}
		}
	}()

	return func() {
		_ = channel.Close()
		_ = conn.Close()
	}, nil
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
