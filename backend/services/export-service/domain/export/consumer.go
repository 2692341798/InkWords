package export

import (
	"context"
	"encoding/json"
	"log"
	"os"

	taskdomain "inkwords-backend/internal/domain/task"
	"inkwords-backend/internal/infra/mq"
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

// StartExportConsumer starts the export worker on the service-owned RabbitMQ queue.
func StartExportConsumer(ctx context.Context, consumer *taskdomain.ExportConsumer, queueName string) (func(), error) {
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
	routingKey := mq.ExportRequestedMessage{}.RoutingKey()

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

				var message taskdomain.ExportRequestedMessage
				if err := json.Unmarshal(delivery.Body, &message); err != nil {
					log.Printf("invalid export message payload: %v", err)
					_ = delivery.Ack(false)
					continue
				}

				if err := consumer.HandleExportRequested(ctx, message); err != nil {
					log.Printf("export task handling failed for %s: %v", message.TaskID, err)
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

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
