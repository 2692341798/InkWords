package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterParserRoutes_OnlyRegistersParseRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	RegisterParserRoutes(r, func(c *gin.Context) { c.Next() }, &HandlerAdapter{
		ParseFunc: func(c *gin.Context) { c.Status(http.StatusOK) },
	})

	resp := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/project/parse", nil)
	r.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)

	resp = httptest.NewRecorder()
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/project/scan", nil)
	r.ServeHTTP(resp, req)
	require.Equal(t, http.StatusNotFound, resp.Code)
}

type HandlerAdapter struct {
	ParseFunc gin.HandlerFunc
}

func (h *HandlerAdapter) Parse(c *gin.Context) {
	h.ParseFunc(c)
}
