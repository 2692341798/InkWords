package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

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
	})

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/auth/login"},
		{http.MethodGet, "/api/v1/user/profile"},
		{http.MethodGet, "/api/v1/blogs"},
		{http.MethodPost, "/api/v1/project/scan"},
		{http.MethodPost, "/api/v1/stream/generate"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s %s expected 200, got %d", tc.method, tc.path, w.Code)
		}
	}
}
