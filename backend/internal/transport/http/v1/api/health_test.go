package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterHealthRoutes_ExposesPingHealthAndReady(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterHealthRoutes(router, NewHealthAPI("core-api", map[string]ReadinessCheck{
		"db": func(context.Context) error { return nil },
	}))

	for _, path := range []string{"/api/v1/ping", "/health", "/ready"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code, path)
	}
}

func TestReady_ReturnsServiceUnavailableWhenDependencyCheckFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterHealthRoutes(router, NewHealthAPI("llm-stream", map[string]ReadinessCheck{
		"db":       func(context.Context) error { return nil },
		"rabbitmq": func(context.Context) error { return errors.New("dial failed") },
	}))

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusServiceUnavailable, resp.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	require.Equal(t, "llm-stream", payload["service"])

	checks := payload["checks"].(map[string]any)
	rabbitmq := checks["rabbitmq"].(map[string]any)
	require.Equal(t, "error", rabbitmq["status"])
	require.Equal(t, "dial failed", rabbitmq["error"])
}
