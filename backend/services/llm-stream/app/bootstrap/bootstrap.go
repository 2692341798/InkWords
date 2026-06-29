package bootstrap

import (
	"errors"
	"os"

	"github.com/gin-gonic/gin"

	generationapp "inkwords-backend/services/llm-stream/app/generation"
	streamdomain "inkwords-backend/services/llm-stream/domain/stream"
	streamv1 "inkwords-backend/services/llm-stream/transport/http/v1"
	"inkwords-backend/shared/kernel/httpx"
	"inkwords-backend/shared/platform/cache"
	"inkwords-backend/shared/platform/postgres"
)

// BuildRouter assembles the llm-stream owned router plus the services required by its consumer worker.
func BuildRouter() (*gin.Engine, *streamdomain.GormTaskStore, *streamdomain.Service, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, nil, nil, errors.New("DATABASE_URL environment variable is not set")
	}

	dbConn, err := postgres.InitCore(dsn)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := cache.InitRedis(); err != nil {
	}

	r := gin.New()
	r.Use(gin.Recovery(), httpx.RequestID(), httpx.RequestLogger("llm-stream"))
	httpx.RegisterHealthRoutes(r, httpx.NewHealthAPI("llm-stream", map[string]httpx.ReadinessCheck{
		"db":              httpx.NewGormReadinessCheck(dbConn),
		"rabbitmq_config": httpx.NewRequiredValueCheck(os.Getenv("RABBITMQ_URL"), "RABBITMQ_URL is not configured"),
	}))

	quotaService := generationapp.NewQuotaService(dbConn)
	promptReqService := generationapp.NewPromptRequirements(dbConn)
	generatorService := generationapp.NewGeneratorServiceWithDB(
		dbConn,
		promptReqService,
		streamdomain.NewGeneratedBlogPersistence(dbConn),
	)
	decompositionService := generationapp.NewDecompositionService(
		promptReqService,
		streamdomain.NewSeriesPersistence(dbConn),
		streamdomain.NewContinuePersistence(dbConn),
	)

	streamDomainService := streamdomain.NewService(generatorService, decompositionService, quotaService)
	taskDomainService := streamdomain.NewGormTaskStore(dbConn)
	streamDomainHandler := streamdomain.NewHandler(streamDomainService, streamdomain.NewGormBlogReadable(dbConn))

	streamv1.RegisterStreamRoutes(r, httpx.AuthMiddleware(), streamv1.StreamHandlers{
		ContinueBlog: streamDomainHandler.ContinueBlogStreamHandler,
		PolishBlog:   streamDomainHandler.PolishBlogStreamHandler,
		Scan:         streamDomainHandler.ScanStreamHandler,
		Analyze:      streamDomainHandler.AnalyzeStreamHandler,
		Generate:     streamDomainHandler.GenerateBlogStreamHandler,
	})

	return r, taskDomainService, streamDomainService, nil
}
