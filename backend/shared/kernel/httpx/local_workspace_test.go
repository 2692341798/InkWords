package httpx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestLocalWorkspaceContextUsesResolverRatherThanClientInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspaceID := uuid.MustParse("791e60ae-f3c8-4e9e-8718-0aa7bdcef6f9")
	resolverCalls := 0

	router := gin.New()
	router.Use(LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) {
		resolverCalls++
		return workspaceID, nil
	}))
	router.GET("/textbooks", func(c *gin.Context) {
		resolvedID, err := GetLocalWorkspaceID(c)
		require.NoError(t, err)
		require.Equal(t, workspaceID, resolvedID)
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/textbooks?workspace_id="+uuid.NewString(), nil)
	request.Header.Set("X-Workspace-ID", uuid.NewString())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
	require.Equal(t, 1, resolverCalls)
}

func TestLocalWorkspaceContextFailsClosedWhenResolverCannotResolveWorkspace(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) {
		return uuid.Nil, errors.New("migration has not created local workspace")
	}))
	router.GET("/textbooks", func(c *gin.Context) {
		t.Fatal("request must not reach a route without a local workspace")
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/textbooks", nil))

	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.JSONEq(t, `{"code":"LOCAL_WORKSPACE_UNAVAILABLE","message":"本地工作区尚未准备完成，请检查数据库迁移后重试","data":null}`, response.Body.String())
}

func TestGetLocalWorkspaceIDRejectsMissingOrWrongValue(t *testing.T) {
	context := &gin.Context{}
	_, err := GetLocalWorkspaceID(context)
	require.Error(t, err)

	context.Set(localWorkspaceContextKey, "not-a-uuid")
	_, err = GetLocalWorkspaceID(context)
	require.Error(t, err)
}
