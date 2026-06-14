package bootstrap

import (
	"errors"
	"os"

	"github.com/gin-gonic/gin"

	exportdomain "inkwords-backend/services/export-service/domain/export"
	artifact "inkwords-backend/services/export-service/infra/artifact"
	exportroutes "inkwords-backend/services/export-service/transport/http/v1"
	"inkwords-backend/shared/kernel/httpx"
	platformllm "inkwords-backend/shared/platform/llm"
	"inkwords-backend/shared/platform/obsidian"
	"inkwords-backend/shared/platform/postgres"
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
	exportHandler := exportdomain.NewHandler(exportService)
	exportroutes.RegisterExportRoutes(r, httpx.AuthMiddleware(), exportHandler)

	artifactStore := artifact.NewStore(envOrDefault("EXPORT_ARTIFACTS_DIR", "/app/export-artifacts"))
	taskStore := exportdomain.NewGormTaskStore(dbConn)
	consumer := exportdomain.NewConsumer(taskStore, exportService, artifactStore)

	return r, consumer, nil
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
