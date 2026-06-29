package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/joho/godotenv"

	authdomain "inkwords-backend/services/core-api/domain/auth"
	blogdomain "inkwords-backend/services/core-api/domain/blog"
	projectdomain "inkwords-backend/services/core-api/domain/project"
	coretask "inkwords-backend/services/core-api/domain/task"
	userdomain "inkwords-backend/services/core-api/domain/user"
	coreapiv1 "inkwords-backend/services/core-api/transport/http/v1"
	"inkwords-backend/services/core-api/app/projectanalysis"
	coremq "inkwords-backend/services/core-api/infra/mq"

	streamdomain "inkwords-backend/services/llm-stream/domain/stream"
	generationapp "inkwords-backend/services/llm-stream/app/generation"
	streamv1 "inkwords-backend/services/llm-stream/transport/http/v1"

	reviewdomain "inkwords-backend/services/review-service/domain/review"
	reviewwiki "inkwords-backend/services/review-service/infra/wiki"
	reviewroutes "inkwords-backend/services/review-service/transport/http/v1"

	exportdomain "inkwords-backend/services/export-service/domain/export"
	exportroutes "inkwords-backend/services/export-service/transport/http/v1"

	parserdomain "inkwords-backend/services/parser-service/domain/parse"
	parserroutes "inkwords-backend/services/parser-service/transport/http/v1"

	"inkwords-backend/shared/kernel/httpx"
	"inkwords-backend/shared/platform/cache"
	platformllm "inkwords-backend/shared/platform/llm"
	"inkwords-backend/shared/platform/obsidian"
	"inkwords-backend/shared/platform/parser"
	"inkwords-backend/shared/platform/postgres"
)

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

	coreDB, err := postgres.InitCore(dsn)
	if err != nil {
		log.Fatalf("Core database initialization failed: %v", err)
	}

	if reviewDSN := os.Getenv("REVIEW_DATABASE_URL"); reviewDSN != "" && reviewDSN != dsn {
		if _, err := postgres.InitReview(reviewDSN); err != nil {
			log.Fatalf("Review database initialization failed: %v", err)
		}
	} else {
		// Why: 聚合模式下复用核心数据库连接，同时 AutoMigrate 审核表。
		if _, err := postgres.InitReview(dsn); err != nil {
			log.Fatalf("Review database initialization failed: %v", err)
		}
	}

	if err := cache.InitRedis(); err != nil {
		log.Printf("Redis initialization failed (cache will be disabled): %v", err)
	}

	taskPublisher, cleanupPublisher, err := initTaskPublisherFromEnv()
	if err != nil {
		log.Fatalf("Task publisher initialization failed: %v", err)
	}
	defer cleanupPublisher()

	r := gin.New()
	r.Use(gin.Recovery(), httpx.RequestID(), httpx.RequestLogger("inkwords-server"))
	r.MaxMultipartMemory = 888 << 20
	r.Static("/uploads", "./uploads")
	httpx.RegisterHealthRoutes(r, httpx.NewHealthAPI("inkwords-server", map[string]httpx.ReadinessCheck{
		"db": httpx.NewGormReadinessCheck(coreDB),
	}))

	userRepo := userdomain.NewGormRepository(coreDB)
	userDomainService := userdomain.NewService(userRepo)
	userDomainHandler := userdomain.NewHandler(userDomainService)

	authRepo := authdomain.NewGormRepository(coreDB)
	authDomainService := authdomain.NewService(authRepo)
	authDomainHandler := authdomain.NewHandler(authDomainService)

	blogRepo := blogdomain.NewGormRepository(coreDB)
	blogDomainService := blogdomain.NewService(blogRepo)
	blogDomainHandler := blogdomain.NewHandler(blogDomainService)

	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	llmClient := platformllm.NewDeepSeekClient(apiKey)
	paService := projectanalysis.NewService(llmClient)
	gitFetcher := parser.NewGitFetcher()
	docParser := parser.NewDocParser()

	projectDomainService := projectdomain.NewService(
		paService,
		gitFetcher,
		docParser,
		userDomainService,
	)
	projectDomainHandler := projectdomain.NewHandler(projectDomainService)

	generationResultRepo := coretask.NewGormGenerationResultRepository(coreDB)
	resultPersister := coretask.NewResultPersister(generationResultRepo, generationResultRepo)

	taskRepo := coretask.NewGormRepository(coreDB)
	taskDomainService := coretask.NewService(taskRepo, taskPublisher, resultPersister)
	taskDomainHandler := coretask.NewHandler(
		taskDomainService,
		envOrDefault("EXPORT_ARTIFACTS_DIR", "/app/export-artifacts"),
	)

	authMiddleware := httpx.AuthMiddleware()

	coreapiv1.RegisterCoreRoutes(r, authMiddleware, coreapiv1.CoreHandlers{
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

	quotaService := generationapp.NewQuotaService(coreDB)
	promptReqService := generationapp.NewPromptRequirements(coreDB)
	generatorService := generationapp.NewGeneratorServiceWithDB(
		coreDB,
		promptReqService,
		streamdomain.NewGeneratedBlogPersistence(coreDB),
	)
	decompositionService := generationapp.NewDecompositionService(
		promptReqService,
		streamdomain.NewSeriesPersistence(coreDB),
		streamdomain.NewContinuePersistence(coreDB),
	)
	streamDomainService := streamdomain.NewService(generatorService, decompositionService, quotaService)
	streamDomainHandler := streamdomain.NewHandler(streamDomainService, streamdomain.NewGormBlogReadable(coreDB))

	streamv1.RegisterStreamRoutes(r, authMiddleware, streamv1.StreamHandlers{
		ContinueBlog: streamDomainHandler.ContinueBlogStreamHandler,
		PolishBlog:   streamDomainHandler.PolishBlogStreamHandler,
		Scan:         streamDomainHandler.ScanStreamHandler,
		Analyze:      streamDomainHandler.AnalyzeStreamHandler,
		Generate:     streamDomainHandler.GenerateBlogStreamHandler,
	})

	reviewRepo := reviewdomain.NewGormRepository(coreDB)
	reviewNoteSource := reviewwiki.BuildNoteSource(os.Getenv("OBSIDIAN_WIKI_DIR"))
	var reviewAIFeedback reviewdomain.AIFeedbackGenerator
	if apiKey := strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")); apiKey != "" {
		reviewAIFeedback = reviewdomain.NewDeepSeekAIFeedbackGenerator(
			platformllm.NewDeepSeekClient(apiKey),
			firstNonEmpty(strings.TrimSpace(os.Getenv("DEEPSEEK_REVIEW_MODEL")), "deepseek-chat"),
		)
	}
	reviewDomainService := reviewdomain.NewService(reviewRepo, reviewNoteSource, reviewAIFeedback)
	reviewDomainHandler := reviewdomain.NewHandler(reviewDomainService)
	reviewroutes.RegisterReviewRoutes(r, authMiddleware, reviewDomainHandler)

	exportRepo := exportdomain.NewGormRepository(coreDB)
	exportDomainService := exportdomain.NewService(
		exportRepo,
		obsidian.NewStoreFromEnv,
		llmClient,
		envOrDefault("DEEPSEEK_MODEL", "deepseek-v4-flash"),
		envOrDefault("OBSIDIAN_WIKI_DIR", "wiki"),
	)
	exportDomainHandler := exportdomain.NewHandler(exportDomainService)
	exportroutes.RegisterExportRoutes(r, authMiddleware, exportDomainHandler)

	parserQuotaChecker := parserdomain.NewGormQuotaChecker(coreDB)
	parserDocParser := parser.NewDocParser()
	parserArchiveParser := parser.NewArchiveParser(parserDocParser)
	parserDomainService := parserdomain.NewService(parserDocParser, parserArchiveParser)
	parserDomainHandler := parserdomain.NewHandler(parserDomainService, parserQuotaChecker)
	parserroutes.RegisterParserRoutes(r, authMiddleware, parserDomainHandler)

	server := httpx.NewServer(r)
	signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpx.ShutdownOnContextDone(signalContext, server, 15*time.Second); err != nil {
			log.Printf("Server shutdown failed: %v", err)
		}
	}()

	log.Printf("InkWords server is running on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		stop()
		log.Printf("Server startup failed: %v", err)
	}
}

func initTaskPublisherFromEnv() (coretask.Publisher, func(), error) {
	rabbitURL := strings.TrimSpace(os.Getenv("RABBITMQ_URL"))
	if rabbitURL == "" {
		return nil, nil, errors.New("RABBITMQ_URL environment variable is not set")
	}

	exchangeName := envOrDefault("RABBITMQ_EXCHANGE", "inkwords.events")

	connection, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, nil, errors.New("dial RabbitMQ failed: " + err.Error())
	}

	channel, err := connection.Channel()
	if err != nil {
		_ = connection.Close()
		return nil, nil, errors.New("open RabbitMQ channel failed: " + err.Error())
	}

	if err := channel.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil); err != nil {
		_ = channel.Close()
		_ = connection.Close()
		return nil, nil, errors.New("declare RabbitMQ exchange failed: " + err.Error())
	}

	publisher := coremq.NewPublisher(channel, exchangeName)
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
