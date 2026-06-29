package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterStreamRoutes_RegistersLegacyStreamEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	authMiddleware := func(c *gin.Context) { c.Next() }
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }

	RegisterStreamRoutes(r, authMiddleware, StreamHandlers{
		ContinueBlog: ok,
		PolishBlog:   ok,
		Scan:         ok,
		Analyze:      ok,
		Generate:     ok,
	})

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/blogs/task-1/continue"},
		{method: http.MethodPost, path: "/api/v1/blogs/task-1/polish"},
		{method: http.MethodPost, path: "/api/v1/stream/scan"},
		{method: http.MethodPost, path: "/api/v1/stream/analyze"},
		{method: http.MethodPost, path: "/api/v1/stream/generate"},
	} {
		req := httptest.NewRequestWithContext(context.Background(), tc.method, tc.path, nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code, tc.path)
	}
}
