package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRequestLogger_EmitsStructuredAccessLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer

	router := gin.New()
	router.Use(RequestID())
	router.Use(RequestLoggerWithWriter("core-api", &output))
	router.GET("/api/v1/ping", func(c *gin.Context) {
		c.Status(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	req.Header.Set(requestIDHeader, "test-request-id")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusCreated, resp.Code)
	require.NotEmpty(t, output.String())

	var entry map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &entry))
	require.Equal(t, "request_completed", entry["msg"])
	require.Equal(t, "core-api", entry["service"])
	require.Equal(t, "test-request-id", entry["request_id"])
	require.Equal(t, "/api/v1/ping", entry["path"])
	require.Equal(t, http.MethodGet, entry["method"])
	require.Equal(t, float64(http.StatusCreated), entry["status"])
	require.Contains(t, entry, "latency_ms")
}
