package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterParser_OnlyRegistersParseRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	RegisterParser(r, func(c *gin.Context) { c.Next() }, ParserHandlers{
		Parse: func(c *gin.Context) { c.Status(http.StatusOK) },
	})

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/project/parse", nil)
	r.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)

	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/project/scan", nil)
	r.ServeHTTP(resp, req)
	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestRegisterExport_OnlyRegistersExportRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	RegisterExport(r, func(c *gin.Context) { c.Next() }, ExportHandlers{
		ExportSeries:           func(c *gin.Context) { c.Status(http.StatusOK) },
		ExportSeriesPDF:        func(c *gin.Context) { c.Status(http.StatusOK) },
		ExportToObsidian:       func(c *gin.Context) { c.Status(http.StatusOK) },
		ExportSeriesToObsidian: func(c *gin.Context) { c.Status(http.StatusOK) },
	})

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/blogs/1/export", nil)
	r.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)

	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/blogs", nil)
	r.ServeHTTP(resp, req)
	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestRegisterReview_RegistersReviewRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	RegisterReview(r, func(c *gin.Context) { c.Next() }, ReviewOnlyHandlers{
		Review: ReviewHandlers{
			GetTodayCard:  ok,
			GetHistory:    ok,
			PickRandom:    ok,
			ListNotes:     ok,
			CreateSession: ok,
			GetSession:    ok,
			Respond:       ok,
			RequestHint:   ok,
			Finish:        ok,
		},
	})

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/review/today", nil)
	r.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)

	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/blogs/1/export", nil)
	r.ServeHTTP(resp, req)
	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestRegisterCore_TaskRoutesAreReachable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	authMiddleware := func(c *gin.Context) { c.Next() }
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }

	RegisterCore(r, authMiddleware, CoreHandlers{
		Auth: AuthHandlers{
			Register:      ok,
			Login:         ok,
			BindGithub:    ok,
			GetCaptcha:    ok,
			OAuthRedirect: ok,
			OAuthCallback: ok,
		},
		User: UserHandlers{
			GetProfile:           ok,
			UpdateProfile:        ok,
			UploadAvatar:         ok,
			GetUserStats:         ok,
			GetPromptSettings:    ok,
			UpdatePromptSettings: ok,
		},
		Blog: CoreBlogHandlers{
			GetUserBlogs:     ok,
			CreateDraftBlog:  ok,
			BatchDeleteBlogs: ok,
			UpdateBlog:       ok,
		},
		Project: CoreProjectHandlers{
			ScanGithubRepo: ok,
			Analyze:        ok,
		},
		Task: TaskHandlers{
			CreateGeneration: ok,
			CreateParse:      ok,
			CreateExport:     ok,
			GetTask:          ok,
			CancelTask:       ok,
			StreamTask:       ok,
			DownloadTask:     ok,
		},
	})

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/tasks/generation"},
		{method: http.MethodPost, path: "/api/v1/tasks/parse"},
		{method: http.MethodPost, path: "/api/v1/tasks/export"},
		{method: http.MethodGet, path: "/api/v1/tasks/123e4567-e89b-12d3-a456-426614174000"},
		{method: http.MethodPost, path: "/api/v1/tasks/123e4567-e89b-12d3-a456-426614174000/cancel"},
		{method: http.MethodGet, path: "/api/v1/tasks/123e4567-e89b-12d3-a456-426614174000/stream"},
		{method: http.MethodGet, path: "/api/v1/tasks/123e4567-e89b-12d3-a456-426614174000/download"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code)
	}
}

func TestRegisterStream_LegacyRoutesRemainReachableForRollback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	authMiddleware := func(c *gin.Context) { c.Next() }
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }

	RegisterStream(r, authMiddleware, StreamOnlyHandlers{
		Blog: StreamBlogHandlers{
			ContinueBlog: ok,
			PolishBlog:   ok,
		},
		Stream: StreamHandlers{
			ScanStreamHandler:     ok,
			AnalyzeStreamHandler:  ok,
			GenerateStreamHandler: ok,
		},
	})

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/stream/scan"},
		{method: http.MethodPost, path: "/api/v1/stream/analyze"},
		{method: http.MethodPost, path: "/api/v1/stream/generate"},
		{method: http.MethodPost, path: "/api/v1/blogs/123e4567-e89b-12d3-a456-426614174000/continue"},
		{method: http.MethodPost, path: "/api/v1/blogs/123e4567-e89b-12d3-a456-426614174000/polish"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code)
	}
}
