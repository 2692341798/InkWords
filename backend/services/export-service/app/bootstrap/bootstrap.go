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

	blogdomain "inkwords-backend/internal/domain/blog"
	taskdomain "inkwords-backend/internal/domain/task"
	"inkwords-backend/internal/infra/mq"
	"inkwords-backend/internal/service"
	"inkwords-backend/internal/transport/http/middleware"
	transportv1 "inkwords-backend/internal/transport/http/v1"
	transportv1api "inkwords-backend/internal/transport/http/v1/api"
	"inkwords-backend/shared/kernel/httpx"
	"inkwords-backend/shared/platform/postgres"
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

// Run boots the export-service skeleton while keeping export business logic on the legacy implementation.
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
	r.Use(gin.Recovery(), middleware.RequestID(), middleware.RequestLogger("export-service"))
	transportv1api.RegisterHealthRoutes(r, transportv1api.NewHealthAPI("export-service", map[string]transportv1api.ReadinessCheck{
		"db": transportv1api.NewGormReadinessCheck(database),
	}))

	blogService := service.NewBlogService()
	blogRepo := blogdomain.NewGormRepository(database)
	blogDomainService := blogdomain.NewService(blogRepo)
	blogDomainHandler := blogdomain.NewHandlerWithLegacy(blogDomainService, blogService)
	taskRepo := taskdomain.NewGormRepository(database)
	taskDomainService := taskdomain.NewService(taskRepo, nil)
	blogAPI := transportv1api.NewBlogAPIWithDeps(blogService, blogDomainHandler)

	// Why: Task1 的目标是先把服务拥有者入口转移到 services 目录，导出实现本轮仍沿用已验证逻辑。
	transportv1.RegisterExport(r, middleware.AuthMiddleware(), transportv1.ExportHandlers{
		ExportSeries:           blogAPI.ExportSeries,
		ExportSeriesPDF:        blogAPI.ExportSeriesPDF,
		ExportToObsidian:       blogAPI.ExportToObsidian,
		ExportSeriesToObsidian: blogAPI.ExportSeriesToObsidian,
	})

	server := httpx.NewServer(r)
	signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	artifactStore := taskdomain.NewExportArtifactStore(
		envOrDefault("EXPORT_ARTIFACTS_DIR", "/app/export-artifacts"),
		15*time.Minute,
		time.Now,
	)
	exportConsumer := taskdomain.NewExportConsumer(taskDomainService, blogService, artifactStore)
	stopConsumer, err := startExportTaskConsumer(
		signalContext,
		exportConsumer,
		envOrDefault("RABBITMQ_EXPORT_QUEUE", "inkwords.export"),
	)
	if err != nil {
		log.Printf("RabbitMQ export consumer initialization skipped: %v", err)
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

func startExportTaskConsumer(
	signalContext context.Context,
	consumer *taskdomain.ExportConsumer,
	queueName string,
) (func(), error) {
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

	go func() {
		for {
			select {
			case <-signalContext.Done():
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

				if err := consumer.HandleExportRequested(signalContext, message); err != nil {
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
