package export

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"inkwords-backend/shared/kernel/httpx"
	"inkwords-backend/shared/platform/teachingartifact"
	"inkwords-backend/shared/platform/visualasset"
)

func TestChapterZipFailureNeverDownloadsEmptyOrPartialArchive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, markdown, code          string
		status                        int
		missingArtifact, unconfigured bool
	}{
		{name: "cited manuscript needs frozen build", markdown: "# 章节\n\n来源 [evidence:gin-routergroup-get]", status: http.StatusConflict, code: "TEXTBOOK_CHAPTER_REQUIRES_BOOK_BUILD"},
		{name: "missing artifact after manuscript entries", markdown: "# 章节", missingArtifact: true, status: http.StatusInternalServerError, code: "TEXTBOOK_PACKAGE_FAILED"},
		{name: "unconfigured", markdown: "# 章节", unconfigured: true, status: http.StatusServiceUnavailable, code: "TEXTBOOK_PACKAGE_UNAVAILABLE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chapter := TextbookChapterExport{ChapterID: uuid.New(), RevisionID: uuid.New(), Title: "章节", Markdown: tc.markdown, ContentHash: strings.Repeat("a", 64)}
			if tc.missingArtifact {
				chapter.CodeArtifacts = []TextbookCodeArtifactExport{{ID: uuid.New(), ArtifactHash: "sha256:" + strings.Repeat("b", 64)}}
			}
			handler := NewHandler(NewService(&textbookExportTestRepository{chapter: chapter}, nil, nil, "", ""))
			if !tc.unconfigured {
				handler.packageBuilder = NewTextbookChapterPackageBuilder(teachingartifact.NewStore(t.TempDir()), visualasset.NewStore(t.TempDir()))
			}
			router := gin.New()
			router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return uuid.New(), nil }))
			router.GET("/chapters/:chapterID/export/zip", handler.ExportApprovedTextbookChapterZip)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/chapters/"+chapter.ChapterID.String()+"/export/zip", nil))
			require.Equal(t, tc.status, response.Code)
			require.Contains(t, response.Header().Get("Content-Type"), "application/json")
			require.Empty(t, response.Header().Get("Content-Disposition"))
			require.Contains(t, response.Body.String(), tc.code)
			require.NotContains(t, response.Body.String(), "PK")
		})
	}
}
