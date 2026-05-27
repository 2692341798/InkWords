package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegister_PanicsWhenHandlerMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	authMiddleware := func(c *gin.Context) { c.Next() }
	ok := func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) }

	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic when handlers are missing")
		}
	}()

	Register(r, authMiddleware, Handlers{
		Auth: AuthHandlers{
			Register:      ok,
			Login:         ok,
			BindGithub:    ok,
			GetCaptcha:    ok,
			OAuthRedirect: ok,
			OAuthCallback: ok,
		},
		User: UserHandlers{
			GetProfile:    ok,
			UpdateProfile: ok,
			UploadAvatar:  ok,
			GetUserStats:  ok,
		},
		Blog: BlogHandlers{
			GetUserBlogs:           ok,
			CreateDraftBlog:        ok,
			BatchDeleteBlogs:       ok,
			UpdateBlog:             ok,
			ExportSeries:           ok,
			ExportSeriesPDF:        ok,
			ExportToObsidian:       ok,
			ExportSeriesToObsidian: ok,
			ContinueBlog:           ok,
			PolishBlog:             ok,
		},
		Project: ProjectHandlers{
			ScanGithubRepo: ok,
			Analyze:        ok,
			Parse:          ok,
		},
		Stream: StreamHandlers{
			ScanStreamHandler:     ok,
			AnalyzeStreamHandler:  ok,
			GenerateStreamHandler: ok,
		},
		Review: ReviewHandlers{},
	})
}

func TestRegister_RoutesAreReachable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()

	authMiddleware := func(c *gin.Context) { c.Next() }
	ok := func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) }

	Register(r, authMiddleware, Handlers{
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
		Blog: BlogHandlers{
			GetUserBlogs:           ok,
			CreateDraftBlog:        ok,
			BatchDeleteBlogs:       ok,
			UpdateBlog:             ok,
			ExportSeries:           ok,
			ExportSeriesPDF:        ok,
			ExportToObsidian:       ok,
			ExportSeriesToObsidian: ok,
			ContinueBlog:           ok,
			PolishBlog:             ok,
		},
		Project: ProjectHandlers{
			ScanGithubRepo: ok,
			Analyze:        ok,
			Parse:          ok,
		},
		Stream: StreamHandlers{
			ScanStreamHandler:     ok,
			AnalyzeStreamHandler:  ok,
			GenerateStreamHandler: ok,
		},
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

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/auth/login"},
		{http.MethodGet, "/api/v1/user/profile"},
		{http.MethodGet, "/api/v1/user/prompt-settings"},
		{http.MethodPut, "/api/v1/user/prompt-settings"},
		{http.MethodGet, "/api/v1/blogs"},
		{http.MethodPost, "/api/v1/project/scan"},
		{http.MethodPost, "/api/v1/stream/generate"},
		{http.MethodGet, "/api/v1/review/today"},
		{http.MethodGet, "/api/v1/review/history"},
		{http.MethodPost, "/api/v1/review/sessions"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s %s expected 200, got %d", tc.method, tc.path, w.Code)
		}
	}
}
