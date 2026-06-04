package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"

	authdomain "inkwords-backend/internal/domain/auth"
	blogdomain "inkwords-backend/internal/domain/blog"
	projectdomain "inkwords-backend/internal/domain/project"
	taskdomain "inkwords-backend/internal/domain/task"
	userdomain "inkwords-backend/internal/domain/user"
	"inkwords-backend/internal/infra/cache"
	"inkwords-backend/internal/infra/mq"
	"inkwords-backend/internal/infra/parser"
	"inkwords-backend/internal/service"
	"inkwords-backend/internal/transport/http/middleware"
	transportv1api "inkwords-backend/internal/transport/http/v1/api"
	coretask "inkwords-backend/services/core-api/domain/task"
	corev1 "inkwords-backend/services/core-api/transport/http/v1"
	"inkwords-backend/shared/platform/postgres"
)

type taskPublisherFactory func(rabbitURL string, exchange string) (taskdomain.Publisher, func(), error)

// BuildRouter assembles the core-api owned router and returns a cleanup hook for runtime resources.
func BuildRouter() (*gin.Engine, func(), error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, nil, errors.New("DATABASE_URL environment variable is not set")
	}

	dbConn, err := postgres.InitCore(dsn)
	if err != nil {
		return nil, nil, err
	}
	if err := cache.InitRedis(); err != nil {
		// Why: cache 是增强项而不是启动硬依赖，保持与现有 core-api 行为一致。
	}

	taskPublisher, cleanupTaskPublisher, err := InitTaskPublisherFromEnv(newRabbitMQTaskPublisher)
	if err != nil {
		return nil, nil, err
	}

	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestID(), middleware.RequestLogger("core-api"))
	r.MaxMultipartMemory = 888 << 20
	r.Static("/uploads", "./uploads")
	transportv1api.RegisterHealthRoutes(r, transportv1api.NewHealthAPI("core-api", map[string]transportv1api.ReadinessCheck{
		"db": transportv1api.NewGormReadinessCheck(dbConn),
	}))

	userService := service.NewUserService(dbConn)
	blogService := service.NewBlogService()
	promptReqService := service.NewPromptRequirementsService(dbConn)
	decompositionService := service.NewDecompositionService(promptReqService)
	gitFetcher := parser.NewGitFetcher()
	docParser := parser.NewDocParser()

	authRepo := authdomain.NewGormRepository(dbConn)
	authDomainService := authdomain.NewService(authRepo)
	authDomainHandler := authdomain.NewHandler(authDomainService)

	blogRepo := blogdomain.NewGormRepository(dbConn)
	blogDomainService := blogdomain.NewService(blogRepo)
	blogDomainHandler := blogdomain.NewHandlerWithLegacy(blogDomainService, blogService)

	userRepo := userdomain.NewGormRepository(dbConn)
	userDomainService := userdomain.NewService(userRepo)
	userDomainHandler := userdomain.NewHandler(userDomainService)

	projectDomainService := projectdomain.NewService(decompositionService, gitFetcher, docParser, userService)
	projectDomainHandler := projectdomain.NewHandler(projectDomainService)
	resultPersister := coretask.NewResultPersister(nil, nil)

	taskRepo := taskdomain.NewGormRepository(dbConn)
	taskDomainService := taskdomain.NewService(taskRepo, taskPublisher)
	taskDomainHandler := taskdomain.NewHandler(
		taskDomainService,
		envOrDefault("EXPORT_ARTIFACTS_DIR", "/app/export-artifacts"),
	)

	authAPI := transportv1api.NewAuthAPIWithDeps(authDomainHandler)
	userAPI := transportv1api.NewUserAPIWithDeps(userService, userDomainHandler)
	projectAPI := transportv1api.NewProjectAPIWithDeps(decompositionService, gitFetcher, docParser, userService, projectDomainHandler)
	blogAPI := transportv1api.NewBlogAPIWithDeps(blogService, blogDomainHandler)
	taskAPI := transportv1api.NewTaskAPIWithDeps(taskDomainHandler)
	_ = resultPersister

	corev1.RegisterCoreRoutes(r, middleware.AuthMiddleware(), corev1.CoreHandlers{
		AuthRegister:         authAPI.Register,
		AuthLogin:            authAPI.Login,
		AuthBindGithub:       authAPI.BindGithub,
		AuthGetCaptcha:       authAPI.GetCaptcha,
		AuthOAuthRedirect:    authAPI.OAuthRedirect,
		AuthOAuthCallback:    authAPI.OAuthCallback,
		UserProfile:          userAPI.GetProfile,
		UserUpdateProfile:    userAPI.UpdateProfile,
		UserUploadAvatar:     userAPI.UploadAvatar,
		UserStats:            userAPI.GetUserStats,
		UserGetPromptSetting: userAPI.GetPromptSettings,
		UserPutPromptSetting: userAPI.UpdatePromptSettings,
		BlogList:             blogAPI.GetUserBlogs,
		BlogCreateDraft:      blogAPI.CreateDraftBlog,
		BlogBatchDelete:      blogAPI.BatchDeleteBlogs,
		BlogUpdate:           blogAPI.UpdateBlog,
		ProjectScan:          projectAPI.ScanGithubRepo,
		ProjectAnalyze:       projectAPI.Analyze,
		TaskCreateGeneration: taskAPI.CreateGenerationTask,
		TaskCreateParse:      taskAPI.CreateParseTask,
		TaskCreateExport:     taskAPI.CreateExportTask,
		TaskGet:              taskAPI.GetTask,
		TaskCancel:           taskAPI.CancelTask,
		TaskStream:           taskAPI.StreamTask,
		TaskDownload:         taskAPI.DownloadTask,
	})

	return r, cleanupTaskPublisher, nil
}

// InitTaskPublisherFromEnv builds the core task publisher from the service runtime environment.
func InitTaskPublisherFromEnv(factory taskPublisherFactory) (taskdomain.Publisher, func(), error) {
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
