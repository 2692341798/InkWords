package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	masteryverification "inkwords-backend/services/review-service/app/masteryverification"
	masterydomain "inkwords-backend/services/review-service/domain/mastery"
	reviewdomain "inkwords-backend/services/review-service/domain/review"
	learnerverificationinfra "inkwords-backend/services/review-service/infra/learnerverification"
	textbooksource "inkwords-backend/services/review-service/infra/textbook"
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
	workspaceDB, err := postgres.OpenExisting(os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, fmt.Errorf("open core workspace database: %w", err)
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
	masteryStore, err := masterydomain.NewGormStore(dbConn)
	if err != nil {
		return nil, err
	}
	workspaceMiddleware := httpx.LocalWorkspaceContext(postgres.NewLocalWorkspaceResolver(workspaceDB))
	practiceSource, err := textbooksource.NewClient(firstNonEmpty(os.Getenv("CORE_API_URL"), "http://core-api:8080"))
	if err != nil {
		return nil, err
	}
	masteryService := masterydomain.NewService(masteryStore).WithPracticeSource(practiceSource)
	learnerRunner, err := learnerverificationinfra.NewClient(firstNonEmpty(os.Getenv("COURSE_RUNNER_URL"), "http://course-runner:8080"))
	if err != nil {
		return nil, err
	}
	learnerVerification := masteryverification.NewService(learnerRunner, masteryService, learnerverificationinfra.NewStore(dbConn))
	if err := learnerVerification.Recover(context.Background()); err != nil {
		return nil, err
	}
	assessments, err := BuildAssessmentService(dbConn, masteryService, learnerVerification)
	if err != nil {
		return nil, err
	}
	reviewService := reviewdomain.NewService(reviewRepo, wiki.BuildNoteSource(os.Getenv("OBSIDIAN_WIKI_DIR")), aiFeedback).WithLegacyMasteryAdapter(legacyNoteMasteryAdapter{mastery: masteryService})
	reviewHandler := reviewdomain.NewHandler(reviewService)
	reviewroutes.RegisterReviewRoutes(r, workspaceMiddleware, reviewHandler)
	reviewroutes.RegisterLegacyReviewMigrationRoutes(r, workspaceMiddleware, reviewHandler)
	reviewroutes.RegisterMasteryRoutes(r, workspaceMiddleware, masterydomain.NewHandler(masteryService))
	reviewroutes.RegisterAssessmentRoutes(r, workspaceMiddleware, assessments)
	reviewroutes.RegisterLearnerVerificationRoutes(r, workspaceMiddleware, learnerVerification)

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
