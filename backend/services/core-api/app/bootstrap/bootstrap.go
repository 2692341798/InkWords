package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"

	"inkwords-backend/services/core-api/app/projectanalysis"
	"inkwords-backend/services/core-api/app/textbookartifact"
	"inkwords-backend/services/core-api/app/textbookgeneration"
	"inkwords-backend/services/core-api/app/textbookimport"
	blogdomain "inkwords-backend/services/core-api/domain/blog"
	projectdomain "inkwords-backend/services/core-api/domain/project"
	coretask "inkwords-backend/services/core-api/domain/task"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	coremq "inkwords-backend/services/core-api/infra/mq"
	corev1 "inkwords-backend/services/core-api/transport/http/v1"
	"inkwords-backend/shared/kernel/httpx"
	"inkwords-backend/shared/platform/cache"
	llm "inkwords-backend/shared/platform/llm"
	"inkwords-backend/shared/platform/parser"
	"inkwords-backend/shared/platform/postgres"
	"inkwords-backend/shared/platform/sourceartifact"
	"inkwords-backend/shared/platform/teachingartifact"
	"inkwords-backend/shared/platform/visualasset"
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
		_ = err // Redis 是增强项而不是启动硬依赖
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

	blogRepo := blogdomain.NewGormRepository(dbConn)
	blogDomainService := blogdomain.NewService(blogRepo)
	blogDomainHandler := blogdomain.NewHandler(blogDomainService)

	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	llmClient := llm.NewDeepSeekClient(apiKey)
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
		cleanupTaskPublisher()
		return nil, nil, fmt.Errorf("configure textbook sample target: %w", err)
	}
	textbookRepo := textbookdomain.NewGormRepository(dbConn, textbookTarget)
	generationResultRepo := coretask.NewGormGenerationResultRepository(dbConn)
	teachingArtifacts := teachingartifact.NewStore(envOrDefault("TEXTBOOK_TEACHING_ARTIFACTS_DIR", "/app/teaching-artifacts")).WithReadGroup(teachingartifact.ReaderGroupID)
	textbookArtifactService := textbookartifact.NewService(teachingArtifacts, textbookRepo)
	textbookSamplePersister := textbookartifact.NewSampleProjectionPersister(textbookRepo, textbookRepo, textbookArtifactService).WithGoToolchainVersion(os.Getenv("TEXTBOOK_TEACHING_GO_VERSION"))
	resultPersister := coretask.NewResultPersister(generationResultRepo).WithTextbookSampleRepository(textbookSamplePersister).WithTextbookSourceImportRepository(textbookRepo)

	taskRepo := coretask.NewGormRepository(dbConn)
	taskDomainService := coretask.NewService(taskRepo, taskPublisher, resultPersister)
	taskDomainHandler := coretask.NewHandler(
		taskDomainService,
		envOrDefault("EXPORT_ARTIFACTS_DIR", "/app/export-artifacts"),
	)
	textbookService := textbookdomain.NewService(textbookRepo)
	textbookTaskCreator := textbookgeneration.NewService(textbookService, taskDomainService, textbookTarget)
	textbookVerificationTaskCreator := textbookartifact.NewVerificationTaskService(textbookService, taskDomainService, os.Getenv("TEXTBOOK_TEACHING_ARTIFACT_VERIFICATION_ENABLED") == "true")
	sourceArtifacts := sourceartifact.NewStore(envOrDefault("TEXTBOOK_SOURCE_ARTIFACTS_DIR", "/app/source-artifacts"))
	textbookImportService := textbookimport.NewService(textbookService, taskDomainService, sourceArtifacts)
	textbookHandler := textbookdomain.NewHandler(textbookService, textbookTaskCreator).WithSourceImportCreator(textbookImportService).WithOfficialWebImportCreator(textbookImportService).WithVerificationTaskCreator(textbookVerificationTaskCreator).WithVisualAssetCreator(textbookartifact.NewVisualAssetService(visualasset.NewStore(envOrDefault("TEXTBOOK_VISUAL_ASSETS_DIR", "/app/visual-assets")).WithReadGroup(teachingartifact.ReaderGroupID), textbookService))
	textbookTaskHandler := corev1.NewTextbookTaskHandler(textbookgeneration.NewTaskAccessService(taskDomainService))

	workspaceMiddleware := httpx.LocalWorkspaceContext(postgres.NewLocalWorkspaceResolver(dbConn))
	dependencyCatalog := textbookartifact.NewLocalDependencyCatalog(os.Getenv("TEXTBOOK_DEPENDENCY_CATALOG_FILE"), textbookRepo)
	corev1.RegisterDependencyProjectionRoutes(r, workspaceMiddleware, corev1.NewDependencyProjectionHandler(dependencyCatalog, textbookartifact.NewDependencyProjectionService(textbookRepo, dependencyCatalog, textbookArtifactService)))
	corev1.RegisterBlogRoutes(r, workspaceMiddleware, corev1.BlogHandlers{
		BlogList:        blogDomainHandler.GetUserBlogs,
		BlogCreateDraft: blogDomainHandler.CreateDraftBlog,
		BlogBatchDelete: blogDomainHandler.BatchDeleteBlogs,
		BlogUpdate:      blogDomainHandler.UpdateBlog,
	})
	corev1.RegisterProjectRoutes(r, workspaceMiddleware, corev1.ProjectHandlers{
		ProjectScan:    projectDomainHandler.ScanGithubRepo,
		ProjectAnalyze: projectDomainHandler.Analyze,
	})
	corev1.RegisterTaskRoutes(r, workspaceMiddleware, corev1.TaskHandlers{
		TaskCreateGeneration: taskDomainHandler.CreateGenerationTask,
		TaskCreateParse:      taskDomainHandler.CreateParseTask,
		TaskCreateExport:     taskDomainHandler.CreateExportTask,
		TaskGet:              taskDomainHandler.GetTask,
		TaskRetry:            taskDomainHandler.RetryGenerationTask,
		TaskCancel:           taskDomainHandler.CancelTask,
		TaskStream:           taskDomainHandler.StreamTask,
		TaskDownload:         taskDomainHandler.DownloadTask,
	})
	corev1.RegisterTextbookRoutes(r, workspaceMiddleware, corev1.TextbookHandlers{
		TextbookGetTask:                          textbookTaskHandler.GetTask,
		TextbookRetryTask:                        textbookTaskHandler.RetryTask,
		TextbookCreateProject:                    textbookHandler.CreateProject,
		TextbookListProjects:                     textbookHandler.ListProjects,
		TextbookGetProject:                       textbookHandler.GetProject,
		TextbookGetProjectWorkspace:              textbookHandler.GetProjectWorkspace,
		TextbookGetProjectProgress:               textbookHandler.GetProjectProgress,
		TextbookCreateBookBuild:                  textbookHandler.CreateBookBuild,
		TextbookGetEditorialWorkspace:            textbookHandler.GetEditorialWorkspace,
		TextbookAddRightsItem:                    textbookHandler.AddRightsItem,
		TextbookAppendRightsAmendment:            textbookHandler.AppendRightsAmendment,
		TextbookCompletePublicationReview:        textbookHandler.CompletePublicationReview,
		TextbookRecordDelegatedPublicationReview: textbookHandler.RecordDelegatedPublicationReview,
		TextbookPromoteBookBuild:                 textbookHandler.PromoteBookBuild,
		TextbookListSourceLibrary:                textbookHandler.ListSourceLibrary,
		TextbookListSourceEvidence:               textbookHandler.ListSourceEvidence,
		TextbookRetrieveSourceEvidence:           textbookHandler.RetrieveSourceEvidence,
		TextbookGetChapterWorkspace:              textbookHandler.GetChapterWorkspace,
		TextbookGetApprovedProjections:           textbookHandler.GetApprovedChapterProjections,
		TextbookGetPracticeEvidence:              textbookHandler.GetPracticeEvidence,
		TextbookAddSource:                        textbookHandler.AddSource,
		TextbookLoadGinFixture:                   textbookHandler.LoadGinFixture,
		TextbookCreateSourceImport:               textbookHandler.CreateSourceImport,
		TextbookCreateOfficialWebImport:          textbookHandler.CreateOfficialWebImport,
		TextbookCreateChapter:                    textbookHandler.CreateChapter,
		TextbookCreateBookContract:               textbookHandler.CreateBookContract,
		TextbookCreateStyleSheet:                 textbookHandler.CreateStyleSheet,
		TextbookCreateBlueprint:                  textbookHandler.CreateBlueprint,
		TextbookApproveBookContract:              textbookHandler.ApproveBookContract,
		TextbookApproveStyleSheet:                textbookHandler.ApproveStyleSheet,
		TextbookApproveBlueprint:                 textbookHandler.ApproveBlueprint,
		TextbookAcquireLock:                      textbookHandler.AcquireLock,
		TextbookAppendRevision:                   textbookHandler.AppendRevision,
		TextbookApplyCandidate:                   textbookHandler.ApplyCandidate,
		TextbookRejectCandidate:                  textbookHandler.RejectCandidate,
		TextbookGetSampleGenerationPreflight:     textbookHandler.GetSampleGenerationPreflight,
		TextbookGenerateSample:                   textbookHandler.GenerateSample,
		TextbookCorrectSample:                    textbookHandler.CorrectSample,
		TextbookCreateArtifactVerification:       textbookHandler.CreateArtifactVerification,
		TextbookGetArtifactVerification:          textbookHandler.GetArtifactVerification,
		TextbookUploadVisualAsset:                textbookHandler.UploadVisualAsset,
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
