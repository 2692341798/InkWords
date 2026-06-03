package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	reviewdomain "inkwords-backend/internal/domain/review"
	"inkwords-backend/internal/infra/db"
	"inkwords-backend/internal/service"
	"inkwords-backend/internal/transport/http/middleware"
	transportv1 "inkwords-backend/internal/transport/http/v1"
)

type shutdownableServer interface {
	Shutdown(context.Context) error
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default environment variables")
	}
}

func main() {
	dsn := os.Getenv("REVIEW_DATABASE_URL")
	if dsn == "" {
		log.Fatal("REVIEW_DATABASE_URL environment variable is not set")
	}
	if err := db.InitReviewDB(dsn); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	r := gin.Default()

	r.GET("/api/v1/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"code":    200,
			"message": "pong",
			"data":    nil,
		})
	})

	authMiddleware := middleware.AuthMiddleware()

	reviewRepo := reviewdomain.NewGormRepository(db.DB)
	reviewNoteSource := buildReviewNoteSource()
	reviewDomainService := reviewdomain.NewService(reviewRepo, reviewNoteSource)
	reviewDomainHandler := reviewdomain.NewHandler(reviewDomainService)

	transportv1.RegisterReview(r, authMiddleware, transportv1.ReviewOnlyHandlers{
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

	server := newHTTPServer(r)
	signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go shutdownServerOnContextDone(signalContext, server, 15*time.Second)

	log.Printf("Server is running on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server startup failed: %v", err)
	}
}

func newHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       60 * time.Second,
	}
}

func shutdownServerOnContextDone(signalContext context.Context, server shutdownableServer, timeout time.Duration) {
	<-signalContext.Done()

	shutdownContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		log.Printf("Server shutdown failed: %v", err)
	}
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
