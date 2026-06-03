package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRequestID_GeneratesIDWhenHeaderMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/health", func(c *gin.Context) {
		requestID := GetRequestID(c)
		require.NotEmpty(t, requestID)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.NotEmpty(t, resp.Header().Get(requestIDHeader))
}

func TestRequestID_PropagatesIncomingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/ready", func(c *gin.Context) {
		require.Equal(t, "req-from-proxy", GetRequestID(c))
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	req.Header.Set(requestIDHeader, "req-from-proxy")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "req-from-proxy", resp.Header().Get(requestIDHeader))
}
