package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	fileparsedomain "inkwords-backend/internal/domain/fileparse"
	taskdomain "inkwords-backend/internal/domain/task"
	"inkwords-backend/internal/infra/mq"
	"inkwords-backend/internal/infra/parser"
	"inkwords-backend/internal/service"
	"inkwords-backend/internal/transport/http/middleware"
	transportv1 "inkwords-backend/internal/transport/http/v1"
	transportv1api "inkwords-backend/internal/transport/http/v1/api"
	"inkwords-backend/shared/kernel/httpx"
	"inkwords-backend/shared/platform/postgres"
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

// Run boots the parser-service skeleton while Phase 1 keeps business logic in legacy packages.
func Run() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL environment variable is not set")
	}

	database, err := postgres.InitCore(dsn)
	if err != nil {
		return fmt.Errorf("init core database: %w", err)
	}

	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestID(), middleware.RequestLogger("parser-service"))
	r.MaxMultipartMemory = 888 << 20
	transportv1api.RegisterHealthRoutes(r, transportv1api.NewHealthAPI("parser-service", map[string]transportv1api.ReadinessCheck{
		"db": transportv1api.NewGormReadinessCheck(database),
	}))

	userService := service.NewUserService(database)
	docParser := parser.NewDocParser()
	archiveParser := parser.NewArchiveParser(docParser)
	fileParseService := fileparsedomain.NewService(docParser, archiveParser)
	taskRepo := taskdomain.NewGormRepository(database)
	taskDomainService := taskdomain.NewService(taskRepo, nil)
	fileParseHandler := fileparsedomain.NewHandler(fileParseService, userService)

	// Why: Task1 先把 parser-service 的拥有者入口落到 services 目录，解析实现后续再从 internal 迁移。
	transportv1.RegisterParser(r, middleware.AuthMiddleware(), transportv1.ParserHandlers{
		Parse: fileParseHandler.Parse,
	})

	server := httpx.NewServer(r)
	signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	stopConsumer, err := startParseTaskConsumer(signalContext, taskDomainService, fileParseService)
	if err != nil {
		log.Printf("RabbitMQ parse consumer initialization skipped: %v", err)
	}
	defer stopConsumer()

	go func() {
		if err := httpx.ShutdownOnContextDone(signalContext, server, 15*time.Second); err != nil {
			log.Printf("Server shutdown failed: %v", err)
		}
	}()

	log.Printf("Server is running on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server startup failed: %w", err)
	}

	return nil
}

func startParseTaskConsumer(
	signalContext context.Context,
	taskService *taskdomain.Service,
	parseService *fileparsedomain.Service,
) (func(), error) {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Println("RabbitMQ is not configured, parse consumer disabled")
		return func() {}, nil
	}

	conn, channel, err := sharedrabbitmq.Dial(rabbitURL)
	if err != nil {
		return func() {}, err
	}

	exchangeName := envOrDefault("RABBITMQ_EXCHANGE", "inkwords.events")
	queueName := envOrDefault("RABBITMQ_PARSE_QUEUE", "inkwords.parse")
	routingKey := mq.ParseRequestedMessage{}.RoutingKey()

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

	consumer := fileparsedomain.NewTaskConsumer(taskService, parseService)
	go func() {
		for {
			select {
			case <-signalContext.Done():
				return
			case delivery, ok := <-deliveries:
				if !ok {
					return
				}

				var message mq.ParseRequestedMessage
				if err := json.Unmarshal(delivery.Body, &message); err != nil {
					log.Printf("invalid parse message payload: %v", err)
					_ = delivery.Ack(false)
					continue
				}

				if err := consumer.HandleParseRequested(signalContext, message); err != nil {
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

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
