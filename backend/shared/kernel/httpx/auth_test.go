package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddlewareUsesConfiguredDevelopmentUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.MustParse("c488417d-81fa-4639-bdff-ae2acc9ca205")
	t.Setenv("DEV_AUTH_USER_ID", userID.String())

	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/private", func(c *gin.Context) {
		value, exists := c.Get(authorizationPayloadKey)
		require.True(t, exists)
		require.Equal(t, userID, value)
		c.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/private", nil))

	require.Equal(t, http.StatusNoContent, response.Code)
}

func TestAuthMiddlewareRejectsMissingTokenWithoutDevelopmentUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DEV_AUTH_USER_ID", "")

	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/private", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/private", nil))

	require.Equal(t, http.StatusUnauthorized, response.Code)
}
