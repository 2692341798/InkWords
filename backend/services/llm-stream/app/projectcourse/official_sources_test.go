package projectcourse

import (
	"io"
	"net"
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

type redirectRoundTripper struct{}

func (redirectRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": []string{"https://evil.example/docs"}}, Body: io.NopCloser(strings.NewReader("")), Request: request}, nil
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

func TestDefaultOfficialRegistryCoversCoreInkWordsTechnologies(t *testing.T) {
	registry := NewDefaultOfficialRegistry()
	for _, technology := range []string{"Go", "Gin", "React", "Zustand", "PostgreSQL", "RabbitMQ", "Redis", "Docker Compose", "Nginx", "TypeScript"} {
		source, err := registry.Resolve(technology, "")
		require.NoError(t, err, technology)
		require.Equal(t, technology, source.Technology)
	}
}

func TestHTTPOfficialSourceProviderEnforcesResponseLimitsAndTypes(t *testing.T) {
	publicResolver := func(string) ([]net.IP, error) { return []net.IP{net.ParseIP("93.184.216.34")}, nil }
	provider := HTTPOfficialSourceProvider{Client: &http.Client{Transport: staticRoundTripper{status: http.StatusOK, contentType: "text/plain", body: "official"}}, AllowedDomains: []string{"example.com"}, MaxBytes: 32, ResolveHost: publicResolver}
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

func TestHTTPOfficialSourceProviderRejectsUntrustedRedirect(t *testing.T) {
	provider := HTTPOfficialSourceProvider{Client: &http.Client{Transport: redirectRoundTripper{}}, AllowedDomains: []string{"example.com"}, ResolveHost: func(host string) ([]net.IP, error) {
		if host == "example.com" {
			return []net.IP{net.ParseIP("93.184.216.34")}, nil
		}
		return []net.IP{net.ParseIP("192.168.1.10")}, nil
	}}
	_, err := provider.Fetch("https://example.com/docs")
	require.ErrorContains(t, err, "redirect rejected")
}

func TestHTTPOfficialSourceProviderRejectsDNSRebindingToPrivateAddress(t *testing.T) {
	provider := HTTPOfficialSourceProvider{AllowedDomains: []string{"example.com"}, ResolveHost: func(string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("169.254.169.254")}, nil
	}}
	_, err := provider.Fetch("https://example.com/docs")
	require.ErrorContains(t, err, "private address")
}
