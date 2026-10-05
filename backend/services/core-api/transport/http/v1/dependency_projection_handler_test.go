package v1

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"inkwords-backend/services/core-api/app/textbookartifact"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	"inkwords-backend/shared/kernel/httpx"
)

type dependencyHTTPFake struct {
	calls              int
	workspace, chapter uuid.UUID
	confirmed          string
	err                error
}

func (f *dependencyHTTPFake) Options(_ context.Context, w, ch uuid.UUID) ([]textbookartifact.DependencyOption, error) {
	f.workspace, f.chapter = w, ch
	return []textbookartifact.DependencyOption{}, f.err
}
func (f *dependencyHTTPFake) Preview(_ context.Context, w, ch uuid.UUID, _ textbookartifact.DependencyProjectionRequest) (textbookartifact.DependencyProjectionPreview, error) {
	f.calls++
	f.workspace, f.chapter = w, ch
	return textbookartifact.DependencyProjectionPreview{}, f.err
}
func (f *dependencyHTTPFake) Apply(_ context.Context, w, ch uuid.UUID, _ textbookartifact.DependencyProjectionRequest, confirmed string) (*textbookdomain.CodeArtifactRow, error) {
	f.calls++
	f.workspace, f.chapter = w, ch
	f.confirmed = confirmed
	return nil, f.err
}

func TestDependencyHTTPRejectsInjectedPathsAndPreservesExplicitConfirmation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w, ch := uuid.New(), uuid.New()
	fake := &dependencyHTTPFake{}
	r := gin.New()
	RegisterDependencyProjectionRoutes(r, httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return w, nil }), NewDependencyProjectionHandler(fake, fake))
	url := "/api/v1/textbook-projects/chapters/" + ch.String() + "/dependency-projection"
	call := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, url+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(rec, req)
		return rec
	}
	for _, body := range []string{`{"root":"/private"}`, `{"command":"go test"}`, `{} {}`, `{"option_id":"` + strings.Repeat("x", 4096) + `"}`} {
		require.Equal(t, http.StatusBadRequest, call(http.MethodPost, "/apply", body).Code)
	}
	require.Zero(t, fake.calls)
	require.Equal(t, http.StatusOK, call(http.MethodGet, "/options", "").Code)
	require.Equal(t, w, fake.workspace)
	require.Equal(t, ch, fake.chapter)
	require.Equal(t, http.StatusOK, call(http.MethodPost, "/apply", `{"option_id":"prepared","confirmed_preview_hash":"sha256:explicit"}`).Code)
	require.Equal(t, "sha256:explicit", fake.confirmed)
	fake.err = textbookartifact.ErrDependencyConfirmationChanged
	require.Equal(t, http.StatusConflict, call(http.MethodPost, "/apply", `{}`).Code)
	fake.err = errors.New("/private/credentials: secret-value")
	response := call(http.MethodPost, "/preview", `{}`)
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.NotContains(t, response.Body.String(), "private")
	require.NotContains(t, response.Body.String(), "secret-value")
}
