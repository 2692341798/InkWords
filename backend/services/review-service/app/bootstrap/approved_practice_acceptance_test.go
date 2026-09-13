package bootstrap

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	verificationapp "inkwords-backend/services/review-service/app/masteryverification"
	"inkwords-backend/services/review-service/domain/mastery"
	verificationinfra "inkwords-backend/services/review-service/infra/learnerverification"
	textbooksource "inkwords-backend/services/review-service/infra/textbook"
	v1 "inkwords-backend/services/review-service/transport/http/v1"
	"inkwords-backend/shared/kernel/httpx"
	platformpostgres "inkwords-backend/shared/platform/postgres"
)

// TestServeApprovedPracticeAcceptance runs the production learning components
// against an isolated database. Core is read-only; all answers are operator
// acceptance inputs, never evidence of the user's personal mastery.
func TestServeApprovedPracticeAcceptance(t *testing.T) {
	if os.Getenv("INKWORDS_APPROVED_PRACTICE_ACCEPTANCE") != "approved" {
		t.Skip("explicit real browser/provider/sandbox acceptance opt-in required")
	}
	dir, dist := os.Getenv("INKWORDS_ACCEPTANCE_DIR"), os.Getenv("INKWORDS_ACCEPTANCE_DIST")
	require.True(t, filepath.IsAbs(dir) && filepath.IsAbs(dist))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	container, err := postgrescontainer.Run(ctx, "postgres:14-alpine", postgrescontainer.WithDatabase("approved_practice_acceptance"), postgrescontainer.WithUsername("evaluation"), postgrescontainer.WithPassword("test-only-password"), postgrescontainer.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	restoreAcceptanceDatabase(t, ctx, container.GetContainerID())
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := platformpostgres.InitReview(dsn)
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	store, err := mastery.NewGormStore(db)
	require.NoError(t, err)
	source, err := textbooksource.NewClient("http://127.0.0.1")
	require.NoError(t, err)
	service := mastery.NewService(store).WithPracticeSource(source)
	runner, err := verificationinfra.NewClient(os.Getenv("INKWORDS_ACCEPTANCE_RUNNER_ORIGIN"))
	require.NoError(t, err)
	verification := verificationapp.NewService(runner, service, verificationinfra.NewStore(db))
	require.NoError(t, verification.Recover(ctx))
	assessment, err := acceptanceAssessmentService(db, service, verification, dir)
	require.NoError(t, err)
	owner, err := uuid.Parse(os.Getenv("INKWORDS_ACCEPTANCE_WORKSPACE"))
	require.NoError(t, err)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())
	middleware := httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return owner, nil })
	v1.RegisterMasteryRoutes(router, middleware, mastery.NewHandler(service))
	v1.RegisterAssessmentRoutes(router, middleware, assessment)
	v1.RegisterLearnerVerificationRoutes(router, middleware, verification)
	origin, err := url.Parse("http://127.0.0.1")
	require.NoError(t, err)
	proxy := httputil.NewSingleHostReverseProxy(origin)
	index, err := os.ReadFile(filepath.Join(dist, "index.html"))
	require.NoError(t, err)
	index = []byte(strings.Replace(string(index), "<body>", `<body><div style="position:sticky;top:0;z-index:9999;background:#fff2b2;color:#202020;padding:8px">AI 操作验收 · 独立数据库 · 不计个人掌握 · 真实模型调用最多两次</div>`, 1))
	static := http.FileServer(http.Dir(dist))
	var modelRequests atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-InkWords-Acceptance", "operator-isolated-database")
		if strings.HasPrefix(r.URL.Path, "/api/v1/mastery/") || strings.HasPrefix(r.URL.Path, "/internal/v1/learner-verification/") {
			if r.Method == http.MethodPost && (strings.HasSuffix(r.URL.Path, "/assessments") || strings.HasSuffix(r.URL.Path, "/retry")) && modelRequests.Add(1) > 2 {
				http.Error(w, "acceptance model request budget exhausted", http.StatusConflict)
				return
			}
			router.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if r.Method != http.MethodGet {
				http.Error(w, "core writes disabled in isolated acceptance", http.StatusMethodNotAllowed)
				return
			}
			proxy.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(index)
			return
		}
		static.ServeHTTP(w, r)
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	t.Cleanup(func() { _ = server.Close() })
	go func() { _ = server.Serve(listener) }()
	ready, _ := json.Marshal(map[string]any{"port": listener.Addr().(*net.TCPAddr).Port, "database_container": container.GetContainerID(), "operator_only": true, "core_writes_disabled": true})
	require.NoError(t, os.WriteFile(filepath.Join(dir, "review-ready.json"), ready, 0600))
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			t.Fatal("acceptance server deadline reached")
		case <-ticker.C:
			if _, err := os.Stat(filepath.Join(dir, "stop")); err == nil {
				return
			}
		}
	}
}
