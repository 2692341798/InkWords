package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"

	streamdomain "inkwords-backend/internal/domain/stream"
	taskdomain "inkwords-backend/internal/domain/task"
	"inkwords-backend/internal/infra/mq"
	"inkwords-backend/services/llm-stream/app/bootstrap"
	"inkwords-backend/shared/kernel/httpx"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default environment variables")
	}
}

func main() {
	router, taskDomainService, streamDomainService, err := bootstrap.BuildRouter()
	if err != nil {
		log.Fatalf("bootstrap llm-stream failed: %v", err)
	}

	signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	stopConsumer, err := startGenerationTaskConsumer(signalContext, taskDomainService, streamDomainService)
	if err != nil {
		log.Printf("RabbitMQ generation consumer initialization skipped: %v", err)
	}
	defer stopConsumer()

	server := httpx.NewServer(router)
	go func() {
		if err := httpx.ShutdownOnContextDone(signalContext, server, 15*time.Second); err != nil {
			log.Printf("Server shutdown failed: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server startup failed: %v", err)
	}
}

func startGenerationTaskConsumer(
	signalContext context.Context,
	taskService *taskdomain.Service,
	streamService *streamdomain.Service,
) (func(), error) {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Println("RabbitMQ is not configured, generation consumer disabled")
		return func() {}, nil
	}

	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return func() {}, err
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return func() {}, err
	}

	exchangeName := envOrDefault("RABBITMQ_EXCHANGE", "inkwords.events")
	queueName := envOrDefault("RABBITMQ_GENERATION_QUEUE", "inkwords.generation")
	routingKey := mq.GenerationRequestedMessage{}.RoutingKey()

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

	deliveries, err := channel.Consume(queue.Name, "llm-stream-generation-worker", false, false, false, false, nil)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	consumer := streamdomain.NewTaskConsumer(taskService, streamService)
	go func() {
		for {
			select {
			case <-signalContext.Done():
				return
			case delivery, ok := <-deliveries:
				if !ok {
					return
				}

				var message mq.GenerationRequestedMessage
				if err := json.Unmarshal(delivery.Body, &message); err != nil {
					log.Printf("invalid generation message payload: %v", err)
					_ = delivery.Ack(false)
					continue
				}

				if err := consumer.HandleGenerationRequested(signalContext, message); err != nil {
					log.Printf("generation task handling failed for %s: %v", message.TaskID, err)
					_ = delivery.Nack(false, true)
					continue
				}

				_ = delivery.Ack(false)
			}
		}
	}()

	stop := func() {
		_ = channel.Close()
		_ = conn.Close()
	}

	return stop, nil
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
