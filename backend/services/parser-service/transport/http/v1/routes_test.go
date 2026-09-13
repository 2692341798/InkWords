package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterParserRoutes_RegistersOnlyParserOwnedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	RegisterParserRoutes(r, func(c *gin.Context) { c.Next() }, &HandlerAdapter{
		ParseFunc: func(c *gin.Context) { c.Status(http.StatusOK) },
	}, &HandlerAdapter{CrawlFunc: func(c *gin.Context) { c.Status(http.StatusCreated) }})

	resp := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/project/parse", nil)
	r.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)

	resp = httptest.NewRecorder()
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/project/crawl", nil)
	r.ServeHTTP(resp, req)
	require.Equal(t, http.StatusCreated, resp.Code)

	resp = httptest.NewRecorder()
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/project/scan", nil)
	r.ServeHTTP(resp, req)
	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestRegisterParserRoutesUsesWorkspaceMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	workspaceCalls := 0
	workspace := func(c *gin.Context) {
		workspaceCalls++
		c.Next()
	}
	RegisterParserRoutes(r, workspace, &HandlerAdapter{
		ParseFunc: func(c *gin.Context) { c.Status(http.StatusOK) },
	}, &HandlerAdapter{CrawlFunc: func(c *gin.Context) { c.Status(http.StatusCreated) }})

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/project/parse", nil))
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, 1, workspaceCalls)
}

type HandlerAdapter struct {
	ParseFunc gin.HandlerFunc
	CrawlFunc gin.HandlerFunc
}

func (h *HandlerAdapter) Parse(c *gin.Context) {
	h.ParseFunc(c)
}

func (h *HandlerAdapter) Crawl(c *gin.Context) {
	h.CrawlFunc(c)
}
