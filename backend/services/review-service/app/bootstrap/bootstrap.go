package bootstrap

import (
	"errors"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	reviewdomain "inkwords-backend/services/review-service/domain/review"
	"inkwords-backend/services/review-service/infra/wiki"
	reviewroutes "inkwords-backend/services/review-service/transport/http/v1"
	"inkwords-backend/shared/kernel/httpx"
	platformllm "inkwords-backend/shared/platform/llm"
	"inkwords-backend/shared/platform/postgres"
)

// BuildRouter assembles the review-service owned router while keeping shared middleware and health checks reusable.
func BuildRouter() (*gin.Engine, error) {
	dsn := os.Getenv("REVIEW_DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("REVIEW_DATABASE_URL environment variable is not set")
	}

	dbConn, err := postgres.InitReview(dsn)
	if err != nil {
		return nil, err
	}

	r := gin.New()
	r.Use(gin.Recovery(), httpx.RequestID(), httpx.RequestLogger("review-service"))
	httpx.RegisterHealthRoutes(r, httpx.NewHealthAPI("review-service", map[string]httpx.ReadinessCheck{
		"db": httpx.NewGormReadinessCheck(dbConn),
	}))

	// Why: review-service 迁入自有目录后，仍复用共享中间件与健康检查，避免把通用基础设施重新复制一份。
	reviewRepo := reviewdomain.NewGormRepository(dbConn)
	var aiFeedback reviewdomain.AIFeedbackGenerator
	if apiKey := strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")); apiKey != "" {
		aiFeedback = reviewdomain.NewDeepSeekAIFeedbackGenerator(
			platformllm.NewDeepSeekClient(apiKey),
			firstNonEmpty(strings.TrimSpace(os.Getenv("DEEPSEEK_REVIEW_MODEL")), "deepseek-chat"),
		)
	}
	reviewService := reviewdomain.NewService(reviewRepo, wiki.BuildNoteSource(os.Getenv("OBSIDIAN_WIKI_DIR")), aiFeedback)
	reviewHandler := reviewdomain.NewHandler(reviewService)
	reviewroutes.RegisterReviewRoutes(r, httpx.AuthMiddleware(), reviewHandler)

	return r, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
