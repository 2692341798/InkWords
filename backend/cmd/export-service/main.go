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

	blogdomain "inkwords-backend/internal/domain/blog"
	taskdomain "inkwords-backend/internal/domain/task"
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

	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestID(), middleware.RequestLogger("export-service"))
	api.RegisterHealthRoutes(r, api.NewHealthAPI("export-service", map[string]api.ReadinessCheck{
		"db": api.NewGormReadinessCheck(db.DB),
	}))

	blogService := service.NewBlogService()
	blogRepo := blogdomain.NewGormRepository(db.DB)
	blogDomainService := blogdomain.NewService(blogRepo)
	blogDomainHandler := blogdomain.NewHandlerWithLegacy(blogDomainService, blogService)
	taskRepo := taskdomain.NewGormRepository(db.DB)
	taskDomainService := taskdomain.NewService(taskRepo, nil, nil)
	blogAPI := api.NewBlogAPIWithDeps(blogService, blogDomainHandler)

	authMiddleware := middleware.AuthMiddleware()
	transportv1.RegisterExport(r, authMiddleware, transportv1.ExportHandlers{
		ExportSeries:           blogAPI.ExportSeries,
		ExportSeriesPDF:        blogAPI.ExportSeriesPDF,
		ExportToObsidian:       blogAPI.ExportToObsidian,
		ExportSeriesToObsidian: blogAPI.ExportSeriesToObsidian,
	})

	server := newHTTPServer(r)
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
