package bootstrap

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	domain "inkwords-backend/services/course-runner/domain/learnerverification"
	infra "inkwords-backend/services/course-runner/infra/learnerverification"
	v1 "inkwords-backend/services/course-runner/transport/http/v1"
)

// TestServeApprovedPracticeAcceptanceRunner exposes the unchanged production
// sandbox for a bounded browser acceptance session with a dedicated resolver.
func TestServeApprovedPracticeAcceptanceRunner(t *testing.T) {
	if os.Getenv("INKWORDS_APPROVED_PRACTICE_ACCEPTANCE") != "approved" {
		t.Skip("explicit approved-practice acceptance opt-in required")
	}
	capability, executor := configureLearnerRuntime(true, os.Getenv("INKWORDS_EXPECTED_RUNNER_IMAGE"), "/evaluation/seccomp.json")
	require.True(t, capability.Available, capability.Reason)
	resolver, err := infra.NewClient(os.Getenv("INKWORDS_ACCEPTANCE_REVIEW_ORIGIN"))
	require.NoError(t, err)
	router := gin.New()
	registerLearnerCapabilityRoute(router, capability)
	v1.RegisterLearnerVerificationRoutes(router, domain.Runner{Resolver: resolver, Stager: infra.Stager{}, Executor: executor, Identity: *capability.Runner})
	server := &http.Server{Addr: ":8080", Handler: router, ReadHeaderTimeout: 5 * time.Second}
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	select {
	case err := <-done:
		require.ErrorIs(t, err, http.ErrServerClosed)
	case <-time.After(20 * time.Minute):
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, server.Shutdown(ctx))
	}
}
