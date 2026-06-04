package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterCoreRoutes_RegistersCoreServiceSurface(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	authMiddleware := func(c *gin.Context) { c.Next() }
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }

	RegisterCoreRoutes(r, authMiddleware, CoreHandlers{
		AuthRegister:         ok,
		AuthLogin:            ok,
		AuthBindGithub:       ok,
		AuthGetCaptcha:       ok,
		AuthOAuthRedirect:    ok,
		AuthOAuthCallback:    ok,
		UserProfile:          ok,
		UserUpdateProfile:    ok,
		UserUploadAvatar:     ok,
		UserStats:            ok,
		UserGetPromptSetting: ok,
		UserPutPromptSetting: ok,
		BlogList:             ok,
		BlogCreateDraft:      ok,
		BlogBatchDelete:      ok,
		BlogUpdate:           ok,
		ProjectScan:          ok,
		ProjectAnalyze:       ok,
		TaskCreateGeneration: ok,
		TaskCreateParse:      ok,
		TaskCreateExport:     ok,
		TaskGet:              ok,
		TaskCancel:           ok,
		TaskStream:           ok,
		TaskDownload:         ok,
	})

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/auth/register"},
		{method: http.MethodPost, path: "/api/v1/auth/login"},
		{method: http.MethodPost, path: "/api/v1/auth/bind-github"},
		{method: http.MethodGet, path: "/api/v1/auth/captcha"},
		{method: http.MethodGet, path: "/api/v1/auth/oauth/github"},
		{method: http.MethodGet, path: "/api/v1/auth/callback/github"},
		{method: http.MethodGet, path: "/api/v1/user/profile"},
		{method: http.MethodPut, path: "/api/v1/user/profile"},
		{method: http.MethodPost, path: "/api/v1/user/avatar"},
		{method: http.MethodGet, path: "/api/v1/user/stats"},
		{method: http.MethodGet, path: "/api/v1/user/prompt-settings"},
		{method: http.MethodPut, path: "/api/v1/user/prompt-settings"},
		{method: http.MethodGet, path: "/api/v1/blogs"},
		{method: http.MethodPost, path: "/api/v1/blogs/draft"},
		{method: http.MethodDelete, path: "/api/v1/blogs"},
		{method: http.MethodPut, path: "/api/v1/blogs/task-1"},
		{method: http.MethodPost, path: "/api/v1/project/scan"},
		{method: http.MethodPost, path: "/api/v1/project/analyze"},
		{method: http.MethodPost, path: "/api/v1/tasks/generation"},
		{method: http.MethodPost, path: "/api/v1/tasks/parse"},
		{method: http.MethodPost, path: "/api/v1/tasks/export"},
		{method: http.MethodGet, path: "/api/v1/tasks/task-1"},
		{method: http.MethodPost, path: "/api/v1/tasks/task-1/cancel"},
		{method: http.MethodGet, path: "/api/v1/tasks/task-1/stream"},
		{method: http.MethodGet, path: "/api/v1/tasks/task-1/download"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code, tc.path)
	}
}
