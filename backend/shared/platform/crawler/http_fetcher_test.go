package crawler

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fixtureResolver map[string][]net.IPAddr

func (resolver fixtureResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	return resolver[host], nil
}

type fixtureRoundTripper func(*http.Request) (*http.Response, error)

func (roundTrip fixtureRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func TestHTTPFetcherRejectsPrivateDNSAnswersBeforeRequest(t *testing.T) {
	fetcher := newHTTPFetcher(fixtureResolver{"docs.example.test": {{IP: net.ParseIP("127.0.0.1")}}}, &http.Client{Transport: fixtureRoundTripper(func(*http.Request) (*http.Response, error) {
		t.Fatal("private destination must not be requested")
		return nil, nil
	})})
	target, err := url.Parse("https://docs.example.test/guide")
	require.NoError(t, err)
	_, err = fetcher.Fetch(context.Background(), target, time.Second, nil)
	require.ErrorContains(t, err, "private or special-use")
}

func TestHTTPFetcherHonorsRobotsAndExtractsPageFacts(t *testing.T) {
	fetcher := newHTTPFetcher(fixtureResolver{"docs.example.test": {{IP: net.ParseIP("203.0.113.10")}}}, &http.Client{Transport: fixtureRoundTripper(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path {
		case "/robots.txt":
			return fixtureResponse(http.StatusOK, "text/plain", "User-agent: *\nDisallow: /private\n", nil), nil
		case "/guide":
			return fixtureResponse(http.StatusOK, "text/html", "<title>Guide</title><h1>Start</h1><h2>Install</h2><a href=\"/guide/install\">next</a>", http.Header{"Retry-After": []string{"2"}}), nil
		default:
			t.Fatalf("unexpected request: %s", request.URL)
			return nil, nil
		}
	})})
	target, err := url.Parse("https://docs.example.test/guide")
	require.NoError(t, err)
	page, err := fetcher.Fetch(context.Background(), target, time.Second, nil)
	require.NoError(t, err)
	require.True(t, page.RobotsAllowed)
	require.Equal(t, "Guide", page.Title)
	require.Equal(t, []string{"Start", "Install"}, page.HeadingTree)
	require.Equal(t, []string{"/guide/install"}, page.Links)
	require.Equal(t, 2*time.Second, page.RetryAfter)

	blocked, err := url.Parse("https://docs.example.test/private")
	require.NoError(t, err)
	page, err = fetcher.Fetch(context.Background(), blocked, time.Second, nil)
	require.NoError(t, err)
	require.False(t, page.RobotsAllowed)
}

func TestExtractDocumentFactsHandlesNavigationAndNestedCPlusPlusHeadings(t *testing.T) {
	title, headings, links := extractDocumentFacts([]byte(`
		<!doctype html><html><head><title>C++ getting started</title></head>
		<body><nav><a href="/tour">Tour</a></nav><main><h1>Start with C++</h1>
		<section><h2>Compile a program</h2><a href="https://example.invalid/third-party">reference</a></section>
		</main></body></html>`), "text/html; charset=utf-8")

	require.Equal(t, "C++ getting started", title)
	require.Equal(t, []string{"Start with C++", "Compile a program"}, headings)
	require.Equal(t, []string{"/tour", "https://example.invalid/third-party"}, links)
}

func TestHTTPFetcherRechecksRedirectDestination(t *testing.T) {
	fetcher := newHTTPFetcher(fixtureResolver{
		"docs.example.test":     {{IP: net.ParseIP("203.0.113.10")}},
		"internal.example.test": {{IP: net.ParseIP("10.0.0.8")}},
	}, &http.Client{Transport: fixtureRoundTripper(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path == "/robots.txt" {
			return fixtureResponse(http.StatusOK, "text/plain", "", nil), nil
		}
		return fixtureResponse(http.StatusFound, "text/html", "", http.Header{"Location": []string{"https://internal.example.test/private"}}), nil
	})})
	target, err := url.Parse("https://docs.example.test/guide")
	require.NoError(t, err)
	_, err = fetcher.Fetch(context.Background(), target, time.Second, nil)
	require.ErrorContains(t, err, "private or special-use")
}

func TestHTTPFetcherRevalidateSendsValidatorsAndAcceptsNotModified(t *testing.T) {
	fetcher := newHTTPFetcher(fixtureResolver{"docs.example.test": {{IP: net.ParseIP("203.0.113.10")}}}, &http.Client{Transport: fixtureRoundTripper(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path {
		case "/robots.txt":
			return fixtureResponse(http.StatusOK, "text/plain", "", nil), nil
		case "/guide":
			require.Equal(t, `"guide-v1"`, request.Header.Get("If-None-Match"))
			require.Equal(t, "Tue, 02 Sep 2026 00:00:00 GMT", request.Header.Get("If-Modified-Since"))
			return fixtureResponse(http.StatusNotModified, "text/html", "", http.Header{"ETag": []string{`"guide-v1"`}}), nil
		default:
			t.Fatalf("unexpected request: %s", request.URL)
			return nil, nil
		}
	})})
	target, err := url.Parse("https://docs.example.test/guide")
	require.NoError(t, err)
	page, err := fetcher.Revalidate(context.Background(), target, time.Second, CacheValidators{ETag: `"guide-v1"`, LastModified: "Tue, 02 Sep 2026 00:00:00 GMT"}, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotModified, page.StatusCode)
	require.True(t, page.RobotsAllowed)
	require.Empty(t, page.Body)
}

func fixtureResponse(status int, contentType, content string, header http.Header) *http.Response {
	if header == nil {
		header = make(http.Header)
	}
	header.Set("Content-Type", contentType)
	return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(content))}
}
