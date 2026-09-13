package bootstrap

import (
	"context"
	"net"
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

// TestServeRealLearnerHTTP is an opt-in evaluation server using production
// HTTP routes, resolver, stager and executor. Its process is bounded to 3 minutes.
func TestServeRealLearnerHTTP(t *testing.T) {
	if os.Getenv("INKWORDS_REAL_LEARNER_HTTP") != "approved" {
		t.Skip("explicit isolated HTTP evaluation opt-in required")
	}
	capability, executor := configureLearnerRuntime(true, os.Getenv("INKWORDS_EXPECTED_RUNNER_IMAGE"), "/evaluation/seccomp.json")
	require.True(t, capability.Available, capability.Reason)
	require.NotNil(t, executor)
	resolver, err := infra.NewClient(os.Getenv("INKWORDS_EVALUATION_REVIEW_ORIGIN"))
	require.NoError(t, err)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerLearnerCapabilityRoute(router, capability)
	v1.RegisterLearnerVerificationRoutes(router, domain.Runner{Resolver: resolver, Stager: infra.Stager{}, Executor: executor, Identity: *capability.Runner})
	listener, err := net.Listen("tcp", ":8080")
	require.NoError(t, err)
	server := &http.Server{Handler: router, ReadHeaderTimeout: 5 * time.Second}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Log("evaluation runner listening; learner execution still requires an explicit run request")
	select {
	case err := <-done:
		require.ErrorIs(t, err, http.ErrServerClosed)
	case <-time.After(180 * time.Second):
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, server.Shutdown(ctx))
	}
}
