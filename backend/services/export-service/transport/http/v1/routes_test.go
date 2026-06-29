package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterExportRoutes_OnlyRegistersExportRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	registerExportRoutes(r, func(c *gin.Context) { c.Next() }, exportRouteHandlers{
		ExportSeries:           ok,
		ExportSeriesPDF:        ok,
		ExportToObsidian:       ok,
		ExportSeriesToObsidian: ok,
	})

	for _, tc := range []struct {
		method string
		path   string
		code   int
	}{
		{method: http.MethodGet, path: "/api/v1/blogs/1/export", code: http.StatusOK},
		{method: http.MethodGet, path: "/api/v1/blogs/1/export/pdf", code: http.StatusOK},
		{method: http.MethodPost, path: "/api/v1/blogs/1/export/obsidian", code: http.StatusOK},
		{method: http.MethodPost, path: "/api/v1/blogs/1/export/obsidian/series", code: http.StatusOK},
		{method: http.MethodGet, path: "/api/v1/blogs", code: http.StatusNotFound},
	} {
		req := httptest.NewRequestWithContext(context.Background(), tc.method, tc.path, nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		require.Equal(t, tc.code, resp.Code)
	}
}
