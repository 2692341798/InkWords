package bootstrap

import (
	"errors"
	"os"

	"github.com/gin-gonic/gin"

	streamdomain "inkwords-backend/internal/domain/stream"
	taskdomain "inkwords-backend/internal/domain/task"
	"inkwords-backend/internal/infra/cache"
	"inkwords-backend/internal/service"
	"inkwords-backend/internal/transport/http/middleware"
	transportv1api "inkwords-backend/internal/transport/http/v1/api"
	generation "inkwords-backend/services/llm-stream/domain/generation"
	streamv1 "inkwords-backend/services/llm-stream/transport/http/v1"
	"inkwords-backend/shared/platform/postgres"
)

// BuildRouter assembles the llm-stream owned router plus the services required by its consumer worker.
func BuildRouter() (*gin.Engine, *taskdomain.Service, *streamdomain.Service, error) {
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
	r.Use(gin.Recovery(), middleware.RequestID(), middleware.RequestLogger("llm-stream"))
	transportv1api.RegisterHealthRoutes(r, transportv1api.NewHealthAPI("llm-stream", map[string]transportv1api.ReadinessCheck{
		"db":              transportv1api.NewGormReadinessCheck(dbConn),
		"rabbitmq_config": transportv1api.NewRequiredValueCheck(os.Getenv("RABBITMQ_URL"), "RABBITMQ_URL is not configured"),
	}))

	userService := service.NewUserService(dbConn)
	promptReqService := service.NewPromptRequirementsService(dbConn)
	generatorService := service.NewGeneratorService(promptReqService)
	decompositionService := service.NewDecompositionService(promptReqService)
	generationService := generation.NewService(nil, nil)

	streamDomainService := streamdomain.NewService(generatorService, decompositionService, userService)
	taskRepo := taskdomain.NewGormRepository(dbConn)
	taskDomainService := taskdomain.NewService(taskRepo, nil, nil)
	streamDomainHandler := streamdomain.NewHandler(streamDomainService, streamdomain.NewGormBlogReadable())
	streamAPI := transportv1api.NewStreamAPIWithDeps(generatorService, decompositionService, userService, streamDomainHandler)
	_ = generationService

	streamv1.RegisterStreamRoutes(r, middleware.AuthMiddleware(), streamv1.StreamHandlers{
		ContinueBlog: streamAPI.ContinueBlogStreamHandler,
		PolishBlog:   streamAPI.PolishBlogStreamHandler,
		Scan:         streamAPI.ScanStreamHandler,
		Analyze:      streamAPI.AnalyzeStreamHandler,
		Generate:     streamAPI.GenerateBlogStreamHandler,
	})

	return r, taskDomainService, streamDomainService, nil
}
