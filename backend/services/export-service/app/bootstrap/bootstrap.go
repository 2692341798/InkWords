package bootstrap

import (
	"errors"
	"os"

	"github.com/gin-gonic/gin"

	blogdomain "inkwords-backend/internal/domain/blog"
	taskdomain "inkwords-backend/internal/domain/task"
	legacyservice "inkwords-backend/internal/service"
	"inkwords-backend/internal/transport/http/middleware"
	transportv1api "inkwords-backend/internal/transport/http/v1/api"
	exportdomain "inkwords-backend/services/export-service/domain/export"
	artifact "inkwords-backend/services/export-service/infra/artifact"
	exportroutes "inkwords-backend/services/export-service/transport/http/v1"
	"inkwords-backend/shared/platform/postgres"
)

// BuildRouter assembles the export-service router and worker dependencies behind service-owned entrypoints.
func BuildRouter() (*gin.Engine, *taskdomain.ExportConsumer, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, nil, errors.New("DATABASE_URL environment variable is not set")
	}

	dbConn, err := postgres.InitCore(dsn)
	if err != nil {
		return nil, nil, err
	}

	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestID(), middleware.RequestLogger("export-service"))
	transportv1api.RegisterHealthRoutes(r, transportv1api.NewHealthAPI("export-service", map[string]transportv1api.ReadinessCheck{
		"db": transportv1api.NewGormReadinessCheck(dbConn),
	}))

	// Why: export-service 先收口自己的装配边界，再逐步替换掉 legacy export implementation。
	blogService := legacyservice.NewBlogServiceWithDB(dbConn)
	exportService := exportdomain.NewService(blogService)
	blogRepo := blogdomain.NewGormRepository(dbConn)
	blogDomainService := blogdomain.NewService(blogRepo)
	blogDomainHandler := blogdomain.NewHandlerWithLegacy(blogDomainService, exportService)
	taskService := taskdomain.NewService(taskdomain.NewGormRepository(dbConn), nil, nil)
	blogAPI := transportv1api.NewBlogAPIWithDeps(blogService, blogDomainHandler)
	exportroutes.RegisterExportRoutes(r, middleware.AuthMiddleware(), blogAPI)

	artifactStore := artifact.NewStore(envOrDefault("EXPORT_ARTIFACTS_DIR", "/app/export-artifacts"))
	consumer := taskdomain.NewExportConsumer(taskService, exportService, artifactStore)

	return r, consumer, nil
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
