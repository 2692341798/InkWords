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

	"inkwords-backend/services/core-api/app/textbookartifact"
	"inkwords-backend/shared/platform/teachingartifact"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"

	"inkwords-backend/services/core-api/app/projectanalysis"
	"inkwords-backend/services/core-api/app/textbookgeneration"
	"inkwords-backend/services/core-api/app/textbookimport"
	blogdomain "inkwords-backend/services/core-api/domain/blog"
	projectdomain "inkwords-backend/services/core-api/domain/project"
	coretask "inkwords-backend/services/core-api/domain/task"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	coremq "inkwords-backend/services/core-api/infra/mq"
	coreapiv1 "inkwords-backend/services/core-api/transport/http/v1"

	generationapp "inkwords-backend/services/llm-stream/app/generation"
	streamdomain "inkwords-backend/services/llm-stream/domain/stream"
	streamv1 "inkwords-backend/services/llm-stream/transport/http/v1"

	reviewbootstrap "inkwords-backend/services/review-service/app/bootstrap"
	masterydomain "inkwords-backend/services/review-service/domain/mastery"
	reviewdomain "inkwords-backend/services/review-service/domain/review"
	textbookpractice "inkwords-backend/services/review-service/infra/textbook"
	reviewwiki "inkwords-backend/services/review-service/infra/wiki"
	reviewroutes "inkwords-backend/services/review-service/transport/http/v1"

	exportdomain "inkwords-backend/services/export-service/domain/export"
	exportroutes "inkwords-backend/services/export-service/transport/http/v1"

	crawldomain "inkwords-backend/services/parser-service/domain/crawl"
	parserdomain "inkwords-backend/services/parser-service/domain/parse"
	parserroutes "inkwords-backend/services/parser-service/transport/http/v1"

	"inkwords-backend/shared/kernel/httpx"
	"inkwords-backend/shared/platform/cache"
	platformllm "inkwords-backend/shared/platform/llm"
	"inkwords-backend/shared/platform/obsidian"
	"inkwords-backend/shared/platform/parser"
	"inkwords-backend/shared/platform/postgres"
	"inkwords-backend/shared/platform/sourceartifact"
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

	var reviewDB *gorm.DB
	if reviewDSN := os.Getenv("REVIEW_DATABASE_URL"); reviewDSN != "" && reviewDSN != dsn {
		reviewDB, err = postgres.InitReview(reviewDSN)
		if err != nil {
			log.Fatalf("Review database initialization failed: %v", err)
		}
	} else {
		// Why: 聚合模式下复用核心数据库连接，同时 AutoMigrate 审核表。
		reviewDB, err = postgres.InitReview(dsn)
		if err != nil {
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
	)
	projectDomainHandler := projectdomain.NewHandler(projectDomainService)

	textbookTarget, err := textbookgeneration.SampleGenerationTargetFromConfig(os.Getenv("TEXTBOOK_GENERATION_PROVIDER"), os.Getenv("TEXTBOOK_STANDARD_MODEL"))
	if err != nil {
		log.Fatalf("configure textbook sample target: %v", err)
	}
	textbookRepo := textbookdomain.NewGormRepository(coreDB, textbookTarget)
	generationResultRepo := coretask.NewGormGenerationResultRepository(coreDB)
	resultPersister := coretask.NewResultPersister(generationResultRepo).WithTextbookSampleRepository(textbookRepo).WithTextbookSourceImportRepository(textbookRepo)

	taskRepo := coretask.NewGormRepository(coreDB)
	taskDomainService := coretask.NewService(taskRepo, taskPublisher, resultPersister)
	taskDomainHandler := coretask.NewHandler(
		taskDomainService,
		envOrDefault("EXPORT_ARTIFACTS_DIR", "/app/export-artifacts"),
	)
	textbookService := textbookdomain.NewService(textbookRepo)
	textbookTaskCreator := textbookgeneration.NewService(textbookService, taskDomainService, textbookTarget)
	sourceArtifacts := sourceartifact.NewStore(envOrDefault("TEXTBOOK_SOURCE_ARTIFACTS_DIR", "/app/source-artifacts"))
	textbookHandler := textbookdomain.NewHandler(textbookService, textbookTaskCreator).WithSourceImportCreator(textbookimport.NewService(textbookService, taskDomainService, sourceArtifacts))
	textbookTaskHandler := coreapiv1.NewTextbookTaskHandler(textbookgeneration.NewTaskAccessService(taskDomainService))

	workspaceMiddleware := httpx.LocalWorkspaceContext(postgres.NewLocalWorkspaceResolver(coreDB))
	dependencyCatalog := textbookartifact.NewLocalDependencyCatalog(os.Getenv("TEXTBOOK_DEPENDENCY_CATALOG_FILE"), textbookRepo)
	dependencyArtifacts := textbookartifact.NewService(teachingartifact.NewStore(envOrDefault("TEXTBOOK_TEACHING_ARTIFACTS_DIR", "/app/teaching-artifacts")).WithReadGroup(teachingartifact.ReaderGroupID), textbookRepo)
	coreapiv1.RegisterDependencyProjectionRoutes(r, workspaceMiddleware, coreapiv1.NewDependencyProjectionHandler(dependencyCatalog, textbookartifact.NewDependencyProjectionService(textbookRepo, dependencyCatalog, dependencyArtifacts)))

	coreapiv1.RegisterBlogRoutes(r, workspaceMiddleware, coreapiv1.BlogHandlers{
		BlogList:        blogDomainHandler.GetUserBlogs,
		BlogCreateDraft: blogDomainHandler.CreateDraftBlog,
		BlogBatchDelete: blogDomainHandler.BatchDeleteBlogs,
		BlogUpdate:      blogDomainHandler.UpdateBlog,
	})
	coreapiv1.RegisterProjectRoutes(r, workspaceMiddleware, coreapiv1.ProjectHandlers{
		ProjectScan:    projectDomainHandler.ScanGithubRepo,
		ProjectAnalyze: projectDomainHandler.Analyze,
	})
	coreapiv1.RegisterTaskRoutes(r, workspaceMiddleware, coreapiv1.TaskHandlers{
		TaskCreateGeneration: taskDomainHandler.CreateGenerationTask,
		TaskCreateParse:      taskDomainHandler.CreateParseTask,
		TaskCreateExport:     taskDomainHandler.CreateExportTask,
		TaskGet:              taskDomainHandler.GetTask,
		TaskRetry:            taskDomainHandler.RetryGenerationTask,
		TaskCancel:           taskDomainHandler.CancelTask,
		TaskStream:           taskDomainHandler.StreamTask,
		TaskDownload:         taskDomainHandler.DownloadTask,
	})
	coreapiv1.RegisterTextbookRoutes(r, workspaceMiddleware, coreapiv1.TextbookHandlers{
		TextbookGetTask:                      textbookTaskHandler.GetTask,
		TextbookRetryTask:                    textbookTaskHandler.RetryTask,
		TextbookCreateProject:                textbookHandler.CreateProject,
		TextbookListProjects:                 textbookHandler.ListProjects,
		TextbookGetProject:                   textbookHandler.GetProject,
		TextbookGetProjectWorkspace:          textbookHandler.GetProjectWorkspace,
		TextbookGetProjectProgress:           textbookHandler.GetProjectProgress,
		TextbookCreateBookBuild:              textbookHandler.CreateBookBuild,
		TextbookListSourceLibrary:            textbookHandler.ListSourceLibrary,
		TextbookListSourceEvidence:           textbookHandler.ListSourceEvidence,
		TextbookRetrieveSourceEvidence:       textbookHandler.RetrieveSourceEvidence,
		TextbookGetChapterWorkspace:          textbookHandler.GetChapterWorkspace,
		TextbookGetApprovedProjections:       textbookHandler.GetApprovedChapterProjections,
		TextbookAddSource:                    textbookHandler.AddSource,
		TextbookLoadGinFixture:               textbookHandler.LoadGinFixture,
		TextbookCreateSourceImport:           textbookHandler.CreateSourceImport,
		TextbookCreateOfficialWebImport:      textbookHandler.CreateOfficialWebImport,
		TextbookCreateChapter:                textbookHandler.CreateChapter,
		TextbookCreateBookContract:           textbookHandler.CreateBookContract,
		TextbookCreateStyleSheet:             textbookHandler.CreateStyleSheet,
		TextbookCreateBlueprint:              textbookHandler.CreateBlueprint,
		TextbookApproveBookContract:          textbookHandler.ApproveBookContract,
		TextbookApproveStyleSheet:            textbookHandler.ApproveStyleSheet,
		TextbookApproveBlueprint:             textbookHandler.ApproveBlueprint,
		TextbookAcquireLock:                  textbookHandler.AcquireLock,
		TextbookAppendRevision:               textbookHandler.AppendRevision,
		TextbookApplyCandidate:               textbookHandler.ApplyCandidate,
		TextbookRejectCandidate:              textbookHandler.RejectCandidate,
		TextbookGetSampleGenerationPreflight: textbookHandler.GetSampleGenerationPreflight,
		TextbookGenerateSample:               textbookHandler.GenerateSample,
		TextbookCreateArtifactVerification:   textbookHandler.CreateArtifactVerification,
		TextbookGetArtifactVerification:      textbookHandler.GetArtifactVerification,
		TextbookUploadVisualAsset:            textbookHandler.UploadVisualAsset,
	})

	promptReqService := generationapp.NewPromptRequirements()
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
	streamDomainService := streamdomain.NewService(generatorService, decompositionService)
	streamDomainHandler := streamdomain.NewHandler(streamDomainService, streamdomain.NewGormBlogReadable(coreDB))

	streamv1.RegisterStreamRoutes(r, workspaceMiddleware, streamv1.StreamHandlers{
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
	reviewroutes.RegisterReviewRoutes(r, workspaceMiddleware, reviewDomainHandler)
	masteryStore, err := masterydomain.NewGormStore(reviewDB)
	if err != nil {
		log.Fatalf("Mastery storage initialization failed: %v", err)
	}
	practiceSource, err := textbookpractice.NewClient(envOrDefault("CORE_API_URL", "http://127.0.0.1:8080"))
	if err != nil {
		log.Fatalf("Mastery practice source initialization failed: %v", err)
	}
	masteryService := masterydomain.NewService(masteryStore).WithPracticeSource(practiceSource)
	reviewroutes.RegisterMasteryRoutes(r, workspaceMiddleware, masterydomain.NewHandler(masteryService))
	assessments, err := reviewbootstrap.BuildAssessmentService(reviewDB, masteryService)
	if err != nil {
		log.Fatalf("Failed to initialize mastery assessments: %v", err)
	}
	reviewroutes.RegisterAssessmentRoutes(r, workspaceMiddleware, assessments)

	exportRepo := exportdomain.NewGormRepository(coreDB)
	exportDomainService := exportdomain.NewService(
		exportRepo,
		obsidian.NewStoreFromEnv,
		llmClient,
		envOrDefault("DEEPSEEK_MODEL", "deepseek-v4-flash"),
		envOrDefault("OBSIDIAN_WIKI_DIR", "wiki"),
	)
	exportDomainHandler := exportdomain.NewHandler(exportDomainService)
	exportroutes.RegisterExportRoutes(r, workspaceMiddleware, exportDomainHandler)

	parserDocParser := parser.NewDocParser()
	parserArchiveParser := parser.NewArchiveParser(parserDocParser)
	parserDomainService := parserdomain.NewService(parserDocParser, parserArchiveParser)
	parserDomainHandler := parserdomain.NewHandler(parserDomainService)
	parserCrawlHandler := crawldomain.NewHandler(crawldomain.NewService(crawldomain.NewDefaultHTTPFetcher()))
	parserroutes.RegisterParserRoutes(r, workspaceMiddleware, parserDomainHandler, parserCrawlHandler)

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
