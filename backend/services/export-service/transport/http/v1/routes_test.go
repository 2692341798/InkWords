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
		ExportSeries:                          ok,
		ExportSeriesPDF:                       ok,
		ExportToObsidian:                      ok,
		ExportSeriesToObsidian:                ok,
		ExportApprovedTextbookChapterMarkdown: ok,
		ExportApprovedTextbookChapterZip:      ok,
		ExportFrozenTextbookBookMarkdown:      ok,
		ExportFrozenTextbookBookDOCX:          ok,
		ExportFrozenTextbookBookPDF:           ok,
		ExportFrozenTextbookBookReviewBundle:  ok,
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
		{method: http.MethodGet, path: "/api/v1/textbook-projects/chapters/1/export/markdown", code: http.StatusOK},
		{method: http.MethodGet, path: "/api/v1/textbook-projects/chapters/1/export/zip", code: http.StatusOK},
		{method: http.MethodGet, path: "/api/v1/textbook-projects/book-builds/1/export/markdown", code: http.StatusOK},
		{method: http.MethodGet, path: "/api/v1/textbook-projects/book-builds/1/export/docx", code: http.StatusOK},
		{method: http.MethodGet, path: "/api/v1/textbook-projects/book-builds/1/export/pdf", code: http.StatusOK},
		{method: http.MethodGet, path: "/api/v1/textbook-projects/book-builds/1/export/review-bundle", code: http.StatusOK},
		{method: http.MethodGet, path: "/api/v1/blogs", code: http.StatusNotFound},
	} {
		req := httptest.NewRequestWithContext(context.Background(), tc.method, tc.path, nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		require.Equal(t, tc.code, resp.Code)
	}
}

func TestBlogAndTextbookExportRoutesUseWorkspace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	workspaceCalls := 0
	workspace := func(c *gin.Context) {
		workspaceCalls++
		c.Next()
	}
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	registerExportRoutes(r, workspace, exportRouteHandlers{
		ExportSeries: ok, ExportSeriesPDF: ok, ExportToObsidian: ok, ExportSeriesToObsidian: ok,
		ExportApprovedTextbookChapterMarkdown: ok, ExportApprovedTextbookChapterZip: ok,
		ExportFrozenTextbookBookMarkdown: ok, ExportFrozenTextbookBookDOCX: ok,
		ExportFrozenTextbookBookPDF: ok, ExportFrozenTextbookBookReviewBundle: ok,
	})

	blogResponse := httptest.NewRecorder()
	r.ServeHTTP(blogResponse, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/blogs/1/export", nil))
	require.Equal(t, http.StatusOK, blogResponse.Code)
	require.Equal(t, 1, workspaceCalls)

	textbookResponse := httptest.NewRecorder()
	r.ServeHTTP(textbookResponse, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/textbook-projects/chapters/1/export/markdown", nil))
	require.Equal(t, http.StatusOK, textbookResponse.Code)
	require.Equal(t, 2, workspaceCalls)
}
