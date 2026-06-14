package bootstrap

import (
	"errors"
	"os"

	"github.com/gin-gonic/gin"

	"inkwords-backend/internal/service"
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
		// Why: Redis 在 llm-stream 当前仍是增强项，启动失败不应阻断回滚型流式入口。
	}

	r := gin.New()
	r.Use(gin.Recovery(), httpx.RequestID(), httpx.RequestLogger("llm-stream"))
	httpx.RegisterHealthRoutes(r, httpx.NewHealthAPI("llm-stream", map[string]httpx.ReadinessCheck{
		"db":              httpx.NewGormReadinessCheck(dbConn),
		"rabbitmq_config": httpx.NewRequiredValueCheck(os.Getenv("RABBITMQ_URL"), "RABBITMQ_URL is not configured"),
	}))

	userService := service.NewUserService(dbConn)
	promptReqService := service.NewPromptRequirementsService(dbConn)
	generatorService := service.NewGeneratorServiceWithPersistence(
		promptReqService,
		streamdomain.NewGeneratedBlogPersistence(dbConn),
	)
	decompositionService := service.NewDecompositionServiceWithPersistences(
		promptReqService,
		streamdomain.NewSeriesPersistence(dbConn),
		streamdomain.NewContinuePersistence(dbConn),
	)

	streamDomainService := streamdomain.NewService(generatorService, decompositionService, userService)
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
