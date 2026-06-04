package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	reviewdomain "inkwords-backend/internal/domain/review"
	"inkwords-backend/internal/service"
	"inkwords-backend/internal/transport/http/middleware"
	transportv1 "inkwords-backend/internal/transport/http/v1"
	transportv1api "inkwords-backend/internal/transport/http/v1/api"
	"inkwords-backend/shared/kernel/httpx"
	"inkwords-backend/shared/platform/postgres"
)

// Run boots the review-service skeleton while it still delegates business logic to legacy packages.
func Run() error {
	dsn := os.Getenv("REVIEW_DATABASE_URL")
	if dsn == "" {
		return errors.New("REVIEW_DATABASE_URL environment variable is not set")
	}

	database, err := postgres.InitReview(dsn)
	if err != nil {
		return fmt.Errorf("init review database: %w", err)
	}

	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestID(), middleware.RequestLogger("review-service"))
	transportv1api.RegisterHealthRoutes(r, transportv1api.NewHealthAPI("review-service", map[string]transportv1api.ReadinessCheck{
		"db": transportv1api.NewGormReadinessCheck(database),
	}))

	reviewRepo := reviewdomain.NewGormRepository(database)
	reviewNoteSource := buildReviewNoteSource()
	reviewDomainService := reviewdomain.NewService(reviewRepo, reviewNoteSource)
	reviewDomainHandler := reviewdomain.NewHandler(reviewDomainService)

	// Why: Task1 只建立服务归属入口，先复用已验证的 review 领域实现，避免骨架提交混入行为迁移。
	transportv1.RegisterReview(r, middleware.AuthMiddleware(), transportv1.ReviewOnlyHandlers{
		Review: transportv1.ReviewHandlers{
			GetTodayCard:  reviewDomainHandler.GetTodayCard,
			GetHistory:    reviewDomainHandler.GetHistory,
			PickRandom:    reviewDomainHandler.PickRandom,
			ListNotes:     reviewDomainHandler.ListNotes,
			CreateSession: reviewDomainHandler.CreateSession,
			GetSession:    reviewDomainHandler.GetSession,
			Respond:       reviewDomainHandler.Respond,
			RequestHint:   reviewDomainHandler.RequestHint,
			Finish:        reviewDomainHandler.Finish,
		},
	})

	server := httpx.NewServer(r)
	signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpx.ShutdownOnContextDone(signalContext, server, 15*time.Second); err != nil {
			log.Printf("Server shutdown failed: %v", err)
		}
	}()

	log.Printf("Server is running on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server startup failed: %w", err)
	}

	return nil
}

type unavailableReviewNoteSource struct {
	err error
}

func (s unavailableReviewNoteSource) ListEligibleNotes(context.Context) ([]reviewdomain.ReviewNote, error) {
	return nil, s.err
}

func buildReviewNoteSource() reviewdomain.NoteSource {
	store, err := service.NewObsidianStoreFromEnv()
	if err != nil {
		log.Printf("Review note source initialization failed: %v", err)
		return unavailableReviewNoteSource{err: err}
	}

	rootDir := strings.TrimSpace(os.Getenv("OBSIDIAN_WIKI_DIR"))
	if rootDir == "" {
		rootDir = "wiki"
	}

	return reviewdomain.NewReviewNoteSource(store, rootDir)
}
