package bootstrap

import (
	"errors"
	"os"

	"github.com/gin-gonic/gin"

	"inkwords-backend/internal/transport/http/middleware"
	transportv1api "inkwords-backend/internal/transport/http/v1/api"
	reviewdomain "inkwords-backend/services/review-service/domain/review"
	"inkwords-backend/services/review-service/infra/wiki"
	reviewroutes "inkwords-backend/services/review-service/transport/http/v1"
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
	r.Use(gin.Recovery(), middleware.RequestID(), middleware.RequestLogger("review-service"))
	transportv1api.RegisterHealthRoutes(r, transportv1api.NewHealthAPI("review-service", map[string]transportv1api.ReadinessCheck{
		"db": transportv1api.NewGormReadinessCheck(dbConn),
	}))

	// Why: review-service 迁入自有目录后，仍复用共享中间件与健康检查，避免把通用基础设施重新复制一份。
	reviewRepo := reviewdomain.NewGormRepository(dbConn)
	reviewService := reviewdomain.NewService(reviewRepo, wiki.BuildNoteSource(os.Getenv("OBSIDIAN_WIKI_DIR")))
	reviewHandler := reviewdomain.NewHandler(reviewService)
	reviewroutes.RegisterReviewRoutes(r, middleware.AuthMiddleware(), reviewHandler)

	return r, nil
}
