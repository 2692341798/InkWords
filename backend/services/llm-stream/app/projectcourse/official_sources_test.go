package projectcourse

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type staticRoundTripper struct {
	status      int
	contentType string
	body        string
}

func (r staticRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: r.status, Header: http.Header{"Content-Type": []string{r.contentType}}, Body: io.NopCloser(strings.NewReader(r.body))}, nil
}

func TestOfficialRegistryResolvesAllowlistedSourceAndHashesContent(t *testing.T) {
	registry := NewDefaultOfficialRegistry()
	source, err := registry.Resolve("Go", "1.25")
	require.NoError(t, err)
	source, err = FinalizeOfficialSource(source, "official semantics", time.Unix(1, 0))
	require.NoError(t, err)
	require.Equal(t, "1.25", source.VersionConstraint)
	require.NotEmpty(t, source.ContentHash)
}

func TestValidateOfficialURLRejectsSSRFAndUntrustedDomains(t *testing.T) {
	require.Error(t, ValidateOfficialURL("http://go.dev/doc", []string{"go.dev"}))
	require.Error(t, ValidateOfficialURL("https://127.0.0.1/admin", []string{"127.0.0.1"}))
	require.Error(t, ValidateOfficialURL("https://evil.example/", []string{"go.dev"}))
	require.NoError(t, ValidateOfficialURL("https://pkg.go.dev/std", []string{"go.dev"}))
}

func TestOfficialRegistryRejectsUnknownTechnology(t *testing.T) {
	_, err := NewDefaultOfficialRegistry().Resolve("Unknown", "")
	require.ErrorContains(t, err, "no official source")
}

func TestHTTPOfficialSourceProviderEnforcesResponseLimitsAndTypes(t *testing.T) {
	provider := HTTPOfficialSourceProvider{Client: &http.Client{Transport: staticRoundTripper{status: http.StatusOK, contentType: "text/plain", body: "official"}}, AllowedDomains: []string{"example.com"}, MaxBytes: 32}
	content, err := provider.Fetch("https://example.com/docs")
	require.NoError(t, err)
	require.Equal(t, "official", content)
	provider.Client = &http.Client{Transport: staticRoundTripper{status: http.StatusOK, contentType: "application/octet-stream", body: "binary"}}
	_, err = provider.Fetch("https://example.com/docs")
	require.ErrorContains(t, err, "content type")
	provider.Client = &http.Client{Transport: staticRoundTripper{status: http.StatusOK, contentType: "text/plain", body: "too long"}}
	provider.MaxBytes = 2
	_, err = provider.Fetch("https://example.com/docs")
	require.ErrorContains(t, err, "size limit")
}
