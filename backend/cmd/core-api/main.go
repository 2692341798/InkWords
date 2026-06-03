package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"

	authdomain "inkwords-backend/internal/domain/auth"
	blogdomain "inkwords-backend/internal/domain/blog"
	projectdomain "inkwords-backend/internal/domain/project"
	taskdomain "inkwords-backend/internal/domain/task"
	userdomain "inkwords-backend/internal/domain/user"
	"inkwords-backend/internal/infra/cache"
	"inkwords-backend/internal/infra/db"
	"inkwords-backend/internal/infra/mq"
	"inkwords-backend/internal/infra/parser"
	"inkwords-backend/internal/service"
	"inkwords-backend/internal/transport/http/middleware"
	transportv1 "inkwords-backend/internal/transport/http/v1"
	"inkwords-backend/internal/transport/http/v1/api"
)

type shutdownableServer interface {
	Shutdown(context.Context) error
}

type taskPublisherFactory func(rabbitURL string, exchange string) (taskdomain.Publisher, func(), error)

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

	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestID(), middleware.RequestLogger("core-api"))
	r.MaxMultipartMemory = 888 << 20
	r.Static("/uploads", "./uploads")
	api.RegisterHealthRoutes(r, api.NewHealthAPI("core-api", map[string]api.ReadinessCheck{
		"db": api.NewGormReadinessCheck(db.DB),
	}))

	signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	userService := service.NewUserService(db.DB)
	blogService := service.NewBlogService()
	promptReqService := service.NewPromptRequirementsService(db.DB)
	decompositionService := service.NewDecompositionService(promptReqService)
	gitFetcher := parser.NewGitFetcher()
	docParser := parser.NewDocParser()

	authRepo := authdomain.NewGormRepository(db.DB)
	authDomainService := authdomain.NewService(authRepo)
	authDomainHandler := authdomain.NewHandler(authDomainService)

	blogRepo := blogdomain.NewGormRepository(db.DB)
	blogDomainService := blogdomain.NewService(blogRepo)
	blogDomainHandler := blogdomain.NewHandlerWithLegacy(blogDomainService, blogService)

	userRepo := userdomain.NewGormRepository(db.DB)
	userDomainService := userdomain.NewService(userRepo)
	userDomainHandler := userdomain.NewHandler(userDomainService)

	projectDomainService := projectdomain.NewService(decompositionService, gitFetcher, docParser, userService)
	projectDomainHandler := projectdomain.NewHandler(projectDomainService)

	taskRepo := taskdomain.NewGormRepository(db.DB)
	// Why: core-api 不能再静默接受“创建成功但未投递”的假成功，因此启动时必须显式接入 RabbitMQ publisher。
	taskPublisher, closeTaskPublisher, err := initTaskPublisherFromEnv(newRabbitMQTaskPublisher)
	if err != nil {
		log.Fatalf("RabbitMQ publisher initialization failed: %v", err)
	}
	defer closeTaskPublisher()
	go func() {
		<-signalContext.Done()
		closeTaskPublisher()
	}()
	taskDomainService := taskdomain.NewService(taskRepo, taskPublisher)
	taskDomainHandler := taskdomain.NewHandler(
		taskDomainService,
		envOrDefault("EXPORT_ARTIFACTS_DIR", "/app/export-artifacts"),
	)

	authAPI := api.NewAuthAPIWithDeps(authDomainHandler)
	userAPI := api.NewUserAPIWithDeps(userService, userDomainHandler)
	projectAPI := api.NewProjectAPIWithDeps(decompositionService, gitFetcher, docParser, userService, projectDomainHandler)
	blogAPI := api.NewBlogAPIWithDeps(blogService, blogDomainHandler)
	taskAPI := api.NewTaskAPIWithDeps(taskDomainHandler)

	authMiddleware := middleware.AuthMiddleware()

	transportv1.RegisterCore(r, authMiddleware, transportv1.CoreHandlers{
		Auth: transportv1.AuthHandlers{
			Register:      authAPI.Register,
			Login:         authAPI.Login,
			BindGithub:    authAPI.BindGithub,
			GetCaptcha:    authAPI.GetCaptcha,
			OAuthRedirect: authAPI.OAuthRedirect,
			OAuthCallback: authAPI.OAuthCallback,
		},
		User: transportv1.UserHandlers{
			GetProfile:           userAPI.GetProfile,
			UpdateProfile:        userAPI.UpdateProfile,
			UploadAvatar:         userAPI.UploadAvatar,
			GetUserStats:         userAPI.GetUserStats,
			GetPromptSettings:    userAPI.GetPromptSettings,
			UpdatePromptSettings: userAPI.UpdatePromptSettings,
		},
		Blog: transportv1.CoreBlogHandlers{
			GetUserBlogs:     blogAPI.GetUserBlogs,
			CreateDraftBlog:  blogAPI.CreateDraftBlog,
			BatchDeleteBlogs: blogAPI.BatchDeleteBlogs,
			UpdateBlog:       blogAPI.UpdateBlog,
		},
		Project: transportv1.CoreProjectHandlers{
			ScanGithubRepo: projectAPI.ScanGithubRepo,
			Analyze:        projectAPI.Analyze,
		},
		Task: transportv1.TaskHandlers{
			CreateGeneration: taskAPI.CreateGenerationTask,
			CreateParse:      taskAPI.CreateParseTask,
			CreateExport:     taskAPI.CreateExportTask,
			GetTask:          taskAPI.GetTask,
			CancelTask:       taskAPI.CancelTask,
			StreamTask:       taskAPI.StreamTask,
			DownloadTask:     taskAPI.DownloadTask,
		},
	})

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

func initTaskPublisherFromEnv(factory taskPublisherFactory) (taskdomain.Publisher, func(), error) {
	rabbitURL := strings.TrimSpace(os.Getenv("RABBITMQ_URL"))
	if rabbitURL == "" {
		return nil, nil, errors.New("RABBITMQ_URL environment variable is not set")
	}

	exchangeName := envOrDefault("RABBITMQ_EXCHANGE", "inkwords.events")
	return factory(rabbitURL, exchangeName)
}

func newRabbitMQTaskPublisher(rabbitURL string, exchange string) (taskdomain.Publisher, func(), error) {
	connection, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, nil, fmt.Errorf("dial RabbitMQ failed: %w", err)
	}

	channel, err := connection.Channel()
	if err != nil {
		_ = connection.Close()
		return nil, nil, fmt.Errorf("open RabbitMQ channel failed: %w", err)
	}

	if err := channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		_ = channel.Close()
		_ = connection.Close()
		return nil, nil, fmt.Errorf("declare RabbitMQ exchange failed: %w", err)
	}

	publisher := mq.NewPublisher(channel, exchange)
	var closeOnce sync.Once
	cleanup := func() {
		closeOnce.Do(func() {
			_ = channel.Close()
			_ = connection.Close()
		})
	}

	return publisher, cleanup, nil
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
