package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterLegacyAndTextbookRoutes_RegisterCompleteCoreServiceSurface(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	localOwnerMiddleware := func(c *gin.Context) { c.Next() }
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }

	RegisterBlogRoutes(r, localOwnerMiddleware, BlogHandlers{
		BlogList:        ok,
		BlogCreateDraft: ok,
		BlogBatchDelete: ok,
		BlogUpdate:      ok,
	})
	RegisterProjectRoutes(r, localOwnerMiddleware, ProjectHandlers{
		ProjectScan:    ok,
		ProjectAnalyze: ok,
	})
	RegisterTaskRoutes(r, localOwnerMiddleware, TaskHandlers{
		TaskCreateGeneration: ok,
		TaskCreateParse:      ok,
		TaskCreateExport:     ok,
		TaskGet:              ok,
		TaskRetry:            ok,
		TaskCancel:           ok,
		TaskStream:           ok,
		TaskDownload:         ok,
	})
	RegisterTextbookRoutes(r, localOwnerMiddleware, TextbookHandlers{
		TextbookGetTask:                          ok,
		TextbookRetryTask:                        ok,
		TextbookCreateProject:                    ok,
		TextbookListProjects:                     ok,
		TextbookGetProject:                       ok,
		TextbookGetProjectWorkspace:              ok,
		TextbookGetProjectProgress:               ok,
		TextbookCreateBookBuild:                  ok,
		TextbookGetEditorialWorkspace:            ok,
		TextbookAddRightsItem:                    ok,
		TextbookAppendRightsAmendment:            ok,
		TextbookCompletePublicationReview:        ok,
		TextbookRecordDelegatedPublicationReview: ok,
		TextbookPromoteBookBuild:                 ok,
		TextbookListSourceLibrary:                ok,
		TextbookListSourceEvidence:               ok,
		TextbookRetrieveSourceEvidence:           ok,
		TextbookGetChapterWorkspace:              ok,
		TextbookGetApprovedProjections:           ok,
		TextbookGetPracticeEvidence:              ok,
		TextbookAddSource:                        ok,
		TextbookLoadGinFixture:                   ok,
		TextbookCreateSourceImport:               ok,
		TextbookCreateOfficialWebImport:          ok,
		TextbookCreateChapter:                    ok,
		TextbookCreateBookContract:               ok,
		TextbookCreateStyleSheet:                 ok,
		TextbookCreateBlueprint:                  ok,
		TextbookApproveBookContract:              ok,
		TextbookApproveStyleSheet:                ok,
		TextbookApproveBlueprint:                 ok,
		TextbookAcquireLock:                      ok,
		TextbookAppendRevision:                   ok,
		TextbookApplyCandidate:                   ok,
		TextbookRejectCandidate:                  ok,
		TextbookGenerateSample:                   ok,
		TextbookCorrectSample:                    ok,
		TextbookGetSampleGenerationPreflight:     ok,
		TextbookCreateArtifactVerification:       ok,
		TextbookGetArtifactVerification:          ok,
		TextbookUploadVisualAsset:                ok,
	})

	for _, tc := range []struct {
		method string
		path   string
	}{
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
		{method: http.MethodPost, path: "/api/v1/tasks/task-1/retry"},
		{method: http.MethodPost, path: "/api/v1/tasks/task-1/cancel"},
		{method: http.MethodGet, path: "/api/v1/tasks/task-1/stream"},
		{method: http.MethodGet, path: "/api/v1/tasks/task-1/download"},
		{method: http.MethodGet, path: "/api/v1/textbook-projects"},
		{method: http.MethodGet, path: "/api/v1/textbook-projects/tasks/00000000-0000-0000-0000-000000000003"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/tasks/00000000-0000-0000-0000-000000000003/retry"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects"},
		{method: http.MethodGet, path: "/api/v1/textbook-projects/00000000-0000-0000-0000-000000000001"},
		{method: http.MethodGet, path: "/api/v1/textbook-projects/00000000-0000-0000-0000-000000000001/workspace"},
		{method: http.MethodGet, path: "/api/v1/textbook-projects/00000000-0000-0000-0000-000000000001/progress"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/00000000-0000-0000-0000-000000000001/book-builds"},
		{method: http.MethodGet, path: "/api/v1/textbook-projects/book-builds/00000000-0000-0000-0000-000000000004/editorial"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/book-builds/00000000-0000-0000-0000-000000000004/rights"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/book-builds/00000000-0000-0000-0000-000000000004/reviews"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/book-builds/00000000-0000-0000-0000-000000000004/publication-candidate"},
		{method: http.MethodGet, path: "/api/v1/textbook-projects/00000000-0000-0000-0000-000000000001/source-library"},
		{method: http.MethodGet, path: "/api/v1/textbook-projects/00000000-0000-0000-0000-000000000001/source-evidence"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/00000000-0000-0000-0000-000000000001/source-retrieval"},
		{method: http.MethodGet, path: "/api/v1/textbook-projects/chapters/00000000-0000-0000-0000-000000000001/workspace"},
		{method: http.MethodGet, path: "/api/v1/textbook-projects/chapters/00000000-0000-0000-0000-000000000001/projections"},
		{method: http.MethodGet, path: "/api/v1/textbook-projects/chapters/00000000-0000-0000-0000-000000000001/practice-evidence"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/00000000-0000-0000-0000-000000000001/sources"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/00000000-0000-0000-0000-000000000001/load-gin-fixture"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/00000000-0000-0000-0000-000000000001/source-imports"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/00000000-0000-0000-0000-000000000001/official-web-imports"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/00000000-0000-0000-0000-000000000001/chapters"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/00000000-0000-0000-0000-000000000001/book-contracts"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/00000000-0000-0000-0000-000000000001/style-sheets"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/00000000-0000-0000-0000-000000000001/blueprints"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/00000000-0000-0000-0000-000000000001/book-contracts/00000000-0000-0000-0000-000000000002/approve"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/00000000-0000-0000-0000-000000000001/style-sheets/00000000-0000-0000-0000-000000000002/approve"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/00000000-0000-0000-0000-000000000001/blueprints/00000000-0000-0000-0000-000000000002/approve"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/chapters/00000000-0000-0000-0000-000000000001/lock"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/chapters/00000000-0000-0000-0000-000000000001/revisions"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/chapters/00000000-0000-0000-0000-000000000001/revisions/00000000-0000-0000-0000-000000000002/apply"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/chapters/00000000-0000-0000-0000-000000000001/revisions/00000000-0000-0000-0000-000000000002/reject"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/00000000-0000-0000-0000-000000000001/chapters/00000000-0000-0000-0000-000000000002/generate-sample"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/chapters/00000000-0000-0000-0000-000000000001/code-artifacts/00000000-0000-0000-0000-000000000002/verify"},
		{method: http.MethodGet, path: "/api/v1/textbook-projects/chapters/00000000-0000-0000-0000-000000000001/code-artifacts/00000000-0000-0000-0000-000000000002/verify"},
		{method: http.MethodPost, path: "/api/v1/textbook-projects/chapters/00000000-0000-0000-0000-000000000001/visual-assets"},
	} {
		req := httptest.NewRequestWithContext(context.Background(), tc.method, tc.path, nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code, tc.path)
	}

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
		{method: http.MethodPost, path: "/api/v1/project-courses"},
		{method: http.MethodGet, path: "/api/v1/project-courses/course-1"},
		{method: http.MethodGet, path: "/api/v1/project-courses/course-1/coverage"},
		{method: http.MethodGet, path: "/api/v1/project-courses/course-1/quality-report"},
		{method: http.MethodPost, path: "/api/v1/project-courses/course-1/blueprint/preview"},
		{method: http.MethodPut, path: "/api/v1/project-courses/course-1/blueprint"},
		{method: http.MethodPost, path: "/api/v1/project-courses/course-1/approve"},
		{method: http.MethodPost, path: "/api/v1/project-courses/course-1/package"},
	} {
		req := httptest.NewRequestWithContext(context.Background(), tc.method, tc.path, nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		require.Equal(t, http.StatusNotFound, resp.Code, tc.path)
	}
}

func TestProjectRoutesUseWorkspaceWithoutLegacyOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	workspaceCalls := 0
	workspace := func(c *gin.Context) {
		workspaceCalls++
		c.Next()
	}
	RegisterProjectRoutes(r, workspace, ProjectHandlers{
		ProjectScan:    func(c *gin.Context) { c.Status(http.StatusOK) },
		ProjectAnalyze: func(c *gin.Context) { c.Status(http.StatusOK) },
	})

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/project/scan", nil))
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, 1, workspaceCalls)
}

func TestRegisterTextbookRoutes_DoesNotRequireLegacyAuthenticationSurface(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	workspaceMiddlewareCalled := false
	workspaceMiddleware := func(c *gin.Context) {
		workspaceMiddlewareCalled = true
		c.Next()
	}
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }

	RegisterTextbookRoutes(r, workspaceMiddleware, TextbookHandlers{
		TextbookGetTask:                          ok,
		TextbookRetryTask:                        ok,
		TextbookCreateProject:                    ok,
		TextbookListProjects:                     ok,
		TextbookGetProject:                       ok,
		TextbookGetProjectWorkspace:              ok,
		TextbookGetProjectProgress:               ok,
		TextbookCreateBookBuild:                  ok,
		TextbookGetEditorialWorkspace:            ok,
		TextbookAddRightsItem:                    ok,
		TextbookAppendRightsAmendment:            ok,
		TextbookCompletePublicationReview:        ok,
		TextbookRecordDelegatedPublicationReview: ok,
		TextbookPromoteBookBuild:                 ok,
		TextbookListSourceLibrary:                ok,
		TextbookListSourceEvidence:               ok,
		TextbookRetrieveSourceEvidence:           ok,
		TextbookGetChapterWorkspace:              ok,
		TextbookGetApprovedProjections:           ok,
		TextbookGetPracticeEvidence:              ok,
		TextbookAddSource:                        ok,
		TextbookLoadGinFixture:                   ok,
		TextbookCreateSourceImport:               ok,
		TextbookCreateOfficialWebImport:          ok,
		TextbookCreateChapter:                    ok,
		TextbookCreateBookContract:               ok,
		TextbookCreateStyleSheet:                 ok,
		TextbookCreateBlueprint:                  ok,
		TextbookApproveBookContract:              ok,
		TextbookApproveStyleSheet:                ok,
		TextbookApproveBlueprint:                 ok,
		TextbookAcquireLock:                      ok,
		TextbookAppendRevision:                   ok,
		TextbookApplyCandidate:                   ok,
		TextbookRejectCandidate:                  ok,
		TextbookGenerateSample:                   ok,
		TextbookCorrectSample:                    ok,
		TextbookGetSampleGenerationPreflight:     ok,
		TextbookCreateArtifactVerification:       ok,
		TextbookGetArtifactVerification:          ok,
		TextbookUploadVisualAsset:                ok,
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/textbook-projects", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)
	require.True(t, workspaceMiddlewareCalled)

	legacyReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/auth/login", nil)
	legacyResp := httptest.NewRecorder()
	r.ServeHTTP(legacyResp, legacyReq)
	require.Equal(t, http.StatusNotFound, legacyResp.Code)
}
