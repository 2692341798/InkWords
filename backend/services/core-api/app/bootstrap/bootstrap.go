package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"

	"inkwords-backend/internal/service"
	authdomain "inkwords-backend/services/core-api/domain/auth"
	blogdomain "inkwords-backend/services/core-api/domain/blog"
	projectdomain "inkwords-backend/services/core-api/domain/project"
	coretask "inkwords-backend/services/core-api/domain/task"
	userdomain "inkwords-backend/services/core-api/domain/user"
	coremq "inkwords-backend/services/core-api/infra/mq"
	corev1 "inkwords-backend/services/core-api/transport/http/v1"
	"inkwords-backend/shared/kernel/httpx"
	sharedprompt "inkwords-backend/shared/kernel/prompt"
	"inkwords-backend/shared/platform/cache"
	"inkwords-backend/shared/platform/parser"
	"inkwords-backend/shared/platform/postgres"
)

type taskPublisherFactory func(rabbitURL string, exchange string) (coretask.Publisher, func(), error)

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
	r.Use(gin.Recovery(), httpx.RequestID(), httpx.RequestLogger("core-api"))
	r.MaxMultipartMemory = 888 << 20
	r.Static("/uploads", "./uploads")
	httpx.RegisterHealthRoutes(r, httpx.NewHealthAPI("core-api", map[string]httpx.ReadinessCheck{
		"db": httpx.NewGormReadinessCheck(dbConn),
	}))

	userService := service.NewUserService(dbConn)
	promptReqService := service.NewPromptRequirementsService(dbConn)
	decompositionService := service.NewDecompositionServiceWithPersistences(
		promptReqService,
		blogdomain.NewSeriesPersistence(dbConn),
		blogdomain.NewContinuePersistence(dbConn),
	)
	gitFetcher := parser.NewGitFetcher()
	docParser := parser.NewDocParser()

	authRepo := authdomain.NewGormRepository(dbConn)
	authDomainService := authdomain.NewService(authRepo)
	authDomainHandler := authdomain.NewHandler(authDomainService)

	blogRepo := blogdomain.NewGormRepository(dbConn)
	blogDomainService := blogdomain.NewService(blogRepo)
	blogDomainHandler := blogdomain.NewHandler(blogDomainService)

	userRepo := userdomain.NewGormRepository(dbConn)
	userDomainService := userdomain.NewService(userRepo)
	userDomainHandler := userdomain.NewHandler(userDomainService)

	projectDomainService := projectdomain.NewService(
		projectAnalyzerAdapter{decomposition: decompositionService},
		gitFetcher,
		docParser,
		userService,
	)
	projectDomainHandler := projectdomain.NewHandler(projectDomainService)
	generationResultRepo := coretask.NewGormGenerationResultRepository(dbConn)
	resultPersister := coretask.NewResultPersister(generationResultRepo, generationResultRepo)

	taskRepo := coretask.NewGormRepository(dbConn)
	taskDomainService := coretask.NewService(taskRepo, taskPublisher, resultPersister)
	taskDomainHandler := coretask.NewHandler(
		taskDomainService,
		envOrDefault("EXPORT_ARTIFACTS_DIR", "/app/export-artifacts"),
	)

	corev1.RegisterCoreRoutes(r, httpx.AuthMiddleware(), corev1.CoreHandlers{
		AuthRegister:         authDomainHandler.Register,
		AuthLogin:            authDomainHandler.Login,
		AuthBindGithub:       authDomainHandler.BindGithub,
		AuthGetCaptcha:       authDomainHandler.GetCaptcha,
		AuthOAuthRedirect:    authDomainHandler.OAuthRedirect,
		AuthOAuthCallback:    authDomainHandler.OAuthCallback,
		UserProfile:          userDomainHandler.GetProfile,
		UserUpdateProfile:    userDomainHandler.UpdateProfile,
		UserUploadAvatar:     userDomainHandler.UploadAvatar,
		UserStats:            userDomainHandler.GetUserStats,
		UserGetPromptSetting: userDomainHandler.GetPromptSettings,
		UserPutPromptSetting: userDomainHandler.UpdatePromptSettings,
		BlogList:             blogDomainHandler.GetUserBlogs,
		BlogCreateDraft:      blogDomainHandler.CreateDraftBlog,
		BlogBatchDelete:      blogDomainHandler.BatchDeleteBlogs,
		BlogUpdate:           blogDomainHandler.UpdateBlog,
		ProjectScan:          projectDomainHandler.ScanGithubRepo,
		ProjectAnalyze:       projectDomainHandler.Analyze,
		TaskCreateGeneration: taskDomainHandler.CreateGenerationTask,
		TaskCreateParse:      taskDomainHandler.CreateParseTask,
		TaskCreateExport:     taskDomainHandler.CreateExportTask,
		TaskGet:              taskDomainHandler.GetTask,
		TaskCancel:           taskDomainHandler.CancelTask,
		TaskStream:           taskDomainHandler.StreamTask,
		TaskDownload:         taskDomainHandler.DownloadTask,
	})

	return r, cleanupTaskPublisher, nil
}

// InitTaskPublisherFromEnv builds the core task publisher from the service runtime environment.
func InitTaskPublisherFromEnv(factory taskPublisherFactory) (coretask.Publisher, func(), error) {
	rabbitURL := strings.TrimSpace(os.Getenv("RABBITMQ_URL"))
	if rabbitURL == "" {
		return nil, nil, errors.New("RABBITMQ_URL environment variable is not set")
	}

	exchangeName := envOrDefault("RABBITMQ_EXCHANGE", "inkwords.events")
	return factory(rabbitURL, exchangeName)
}

func newRabbitMQTaskPublisher(rabbitURL string, exchange string) (coretask.Publisher, func(), error) {
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

	publisher := coremq.NewPublisher(channel, exchange)
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

type projectAnalyzerAdapter struct {
	decomposition *service.DecompositionService
}

func (a projectAnalyzerAdapter) ScanProjectModules(ctx context.Context, gitURL string) ([]projectdomain.ModuleCard, error) {
	modules, err := a.decomposition.ScanProjectModules(ctx, gitURL)
	if err != nil {
		return nil, err
	}

	result := make([]projectdomain.ModuleCard, 0, len(modules))
	for _, module := range modules {
		result = append(result, projectdomain.ModuleCard{
			Path:        module.Path,
			Name:        module.Name,
			Description: module.Description,
		})
	}
	return result, nil
}

func (a projectAnalyzerAdapter) GenerateOutline(ctx context.Context, sourceContent string, scenarioMode sharedprompt.ScenarioMode) (projectdomain.OutlineResult, error) {
	outline, err := a.decomposition.GenerateOutline(ctx, sourceContent, scenarioMode, nil, nil)
	if err != nil {
		return projectdomain.OutlineResult{}, err
	}

	chapters := make([]projectdomain.Chapter, 0, len(outline.Chapters))
	for _, chapter := range outline.Chapters {
		chapters = append(chapters, projectdomain.Chapter{
			ID:      chapter.ID,
			Title:   chapter.Title,
			Summary: chapter.Summary,
			Sort:    chapter.Sort,
			Files:   chapter.Files,
			Action:  chapter.Action,
		})
	}
	return projectdomain.OutlineResult{
		SeriesTitle: outline.SeriesTitle,
		Chapters:    chapters,
		ParentID:    outline.ParentID,
	}, nil
}
