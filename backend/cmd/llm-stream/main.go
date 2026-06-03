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

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"

	streamdomain "inkwords-backend/internal/domain/stream"
	taskdomain "inkwords-backend/internal/domain/task"
	"inkwords-backend/internal/infra/cache"
	"inkwords-backend/internal/infra/db"
	"inkwords-backend/internal/infra/mq"
	"inkwords-backend/internal/service"
	"inkwords-backend/internal/transport/http/middleware"
	transportv1 "inkwords-backend/internal/transport/http/v1"
	"inkwords-backend/internal/transport/http/v1/api"
)

type shutdownableServer interface {
	Shutdown(context.Context) error
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default environment variables")
	}
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}
	if err := db.InitCoreDB(dsn); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	if err := cache.InitRedis(); err != nil {
		log.Printf("Redis initialization failed (cache will be disabled): %v", err)
	}

	r := gin.Default()

	r.GET("/api/v1/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"code":    200,
			"message": "pong",
			"data":    nil,
		})
	})

	userService := service.NewUserService(db.DB)
	promptReqService := service.NewPromptRequirementsService(db.DB)
	generatorService := service.NewGeneratorService(promptReqService)
	decompositionService := service.NewDecompositionService(promptReqService)

	streamDomainService := streamdomain.NewService(generatorService, decompositionService, userService)
	taskRepo := taskdomain.NewGormRepository(db.DB)
	taskDomainService := taskdomain.NewService(taskRepo, nil)
	streamDomainHandler := streamdomain.NewHandler(streamDomainService, streamdomain.NewGormBlogReadable())
	streamAPI := api.NewStreamAPIWithDeps(generatorService, decompositionService, userService, streamDomainHandler)

	transportv1.RegisterStream(r, middleware.AuthMiddleware(), transportv1.StreamOnlyHandlers{
		Blog: transportv1.StreamBlogHandlers{
			ContinueBlog: streamAPI.ContinueBlogStreamHandler,
			PolishBlog:   streamAPI.PolishBlogStreamHandler,
		},
		Stream: transportv1.StreamHandlers{
			ScanStreamHandler:     streamAPI.ScanStreamHandler,
			AnalyzeStreamHandler:  streamAPI.AnalyzeStreamHandler,
			GenerateStreamHandler: streamAPI.GenerateBlogStreamHandler,
		},
	})

	signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	stopConsumer, err := startGenerationTaskConsumer(signalContext, taskDomainService, streamDomainService)
	if err != nil {
		log.Printf("RabbitMQ generation consumer initialization skipped: %v", err)
	}
	defer stopConsumer()

	server := newHTTPServer(r)
	go shutdownServerOnContextDone(signalContext, server, 15*time.Second)

	log.Printf("Server is running on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server startup failed: %v", err)
	}
}

func newHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       60 * time.Second,
	}
}

func shutdownServerOnContextDone(signalContext context.Context, server shutdownableServer, timeout time.Duration) {
	<-signalContext.Done()

	shutdownContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		log.Printf("Server shutdown failed: %v", err)
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
