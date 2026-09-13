package bootstrap

import (
	"context"
	"errors"
	"os"

	"github.com/gin-gonic/gin"

	exportdomain "inkwords-backend/services/export-service/domain/export"
	artifact "inkwords-backend/services/export-service/infra/artifact"
	"inkwords-backend/services/export-service/infra/bookrender"
	exportroutes "inkwords-backend/services/export-service/transport/http/v1"
	"inkwords-backend/shared/kernel/httpx"
	platformllm "inkwords-backend/shared/platform/llm"
	"inkwords-backend/shared/platform/obsidian"
	"inkwords-backend/shared/platform/postgres"
	"inkwords-backend/shared/platform/teachingartifact"
	"inkwords-backend/shared/platform/visualasset"
)

// BuildRouter assembles the export-service router and worker dependencies behind service-owned entrypoints.
func BuildRouter() (*gin.Engine, *exportdomain.Consumer, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, nil, errors.New("DATABASE_URL environment variable is not set")
	}

	dbConn, err := postgres.InitCore(dsn)
	if err != nil {
		return nil, nil, err
	}

	r := gin.New()
	r.Use(gin.Recovery(), httpx.RequestID(), httpx.RequestLogger("export-service"))
	httpx.RegisterHealthRoutes(r, httpx.NewHealthAPI("export-service", map[string]httpx.ReadinessCheck{
		"db": httpx.NewGormReadinessCheck(dbConn),
	}))

	llmClient := platformllm.NewDeepSeekClient(os.Getenv("DEEPSEEK_API_KEY"))
	exportRepo := exportdomain.NewGormRepository(dbConn)
	exportService := exportdomain.NewService(
		exportRepo,
		obsidian.NewStoreFromEnv,
		llmClient,
		envOrDefault("DEEPSEEK_MODEL", "deepseek-v4-flash"),
		envOrDefault("OBSIDIAN_WIKI_DIR", "wiki"),
	)
	visualStore := visualasset.NewStore(envOrDefault("TEXTBOOK_VISUAL_ASSETS_DIR", "/app/visual-assets"))
	images := bookrender.NewImageSource(visualStore)
	exportHandler := exportdomain.NewHandler(exportService, exportdomain.NewTextbookChapterPackageBuilder(
		teachingartifact.NewStore(envOrDefault("TEXTBOOK_TEACHING_ARTIFACTS_DIR", "/app/teaching-artifacts")),
		visualStore,
	)).WithBookProjectionRenderers(configuredBookDOCXRenderer(images), configuredBookPDFRenderer(images)).WithBookImages(images)
	workspaceID, err := postgres.EnsureLocalWorkspaceIdentity(context.Background(), dbConn)
	if err != nil {
		return nil, nil, err
	}
	exportroutes.RegisterExportRoutes(r, httpx.LocalWorkspaceContext(postgres.NewLocalWorkspaceResolver(dbConn)), exportHandler)

	artifactStore := artifact.NewStore(envOrDefault("EXPORT_ARTIFACTS_DIR", "/app/export-artifacts"))
	taskStore := exportdomain.NewGormTaskStore(dbConn)
	consumer := exportdomain.NewConsumer(taskStore, exportService, artifactStore, workspaceID)

	return r, consumer, nil
}

func configuredBookDOCXRenderer(images exportdomain.BookImageSource) exportdomain.BookDOCXRenderer {
	executable, reference := os.Getenv("TEXTBOOK_PANDOC_BIN"), os.Getenv("TEXTBOOK_REFERENCE_DOCX")
	if executable == "" || reference == "" {
		return nil
	}
	return exportdomain.NewPandocBookRenderer(executable, reference).WithBookImages(images)
}

func configuredBookPDFRenderer(images exportdomain.BookImageSource) exportdomain.BookPDFRenderer {
	executable := os.Getenv("TEXTBOOK_CHROMIUM_BIN")
	if executable == "" {
		return nil
	}
	return bookrender.NewChromium(executable).WithBookImages(images)
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
