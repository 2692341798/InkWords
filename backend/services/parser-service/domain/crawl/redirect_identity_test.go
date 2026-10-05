package crawl

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedirectAliasesKeepOneCanonicalPageAndUseFinalLinkBase(t *testing.T) {
	policy, err := DefaultPolicy("https://go.dev/doc/tutorial/")
	require.NoError(t, err)
	policy.AllowedPathPrefixes = []string{"/doc/tutorial"}
	page := func(target, body string, links ...string) FetchedPage {
		u, err := url.Parse(target)
		require.NoError(t, err)
		return FetchedPage{FinalURL: u, StatusCode: 200, RobotsAllowed: true, ContentType: "text/html", Body: []byte(body), Links: links}
	}
	service := NewService(fixtureFetcher{pages: map[string]FetchedPage{
		policy.EntryURL:                            page(policy.EntryURL, "index", "start.html", "index.html", "start"),
		"https://go.dev/doc/tutorial/index.html":   page(policy.EntryURL, "index"),
		"https://go.dev/doc/tutorial/start":        page("https://go.dev/doc/tutorial/nested/", "start", "child"),
		"https://go.dev/doc/tutorial/start.html":   page("https://go.dev/doc/tutorial/nested/", "start", "child"),
		"https://go.dev/doc/tutorial/nested/child": page("https://go.dev/doc/tutorial/nested/child", "child"),
	}})
	manifest, captured, err := service.CrawlCaptured(context.Background(), policy)
	require.NoError(t, err)
	require.Equal(t, "complete", manifest.Status)
	require.Len(t, manifest.Pages, 3)
	require.Len(t, captured, 3)
	require.Equal(t, "https://go.dev/doc/tutorial/nested/child", manifest.Pages[2].CanonicalURL)
	var aliases int
	for _, decision := range manifest.Decisions {
		if decision.Reason == "duplicate_destination" {
			aliases++
		}
	}
	require.Equal(t, 2, aliases)
	require.Equal(t, int64(25), manifest.TotalBytes, "duplicate response bytes still count against network budget")
}

func TestRedirectAliasesRejectChangingBytesForOneCanonicalPage(t *testing.T) {
	policy, err := DefaultPolicy("https://go.dev/doc/tutorial/")
	require.NoError(t, err)
	finalURL, err := url.Parse(policy.EntryURL)
	require.NoError(t, err)
	service := NewService(functionFetcher(func(_ context.Context, target *url.URL, _ Policy) (FetchedPage, error) {
		body := "first version"
		if target.Path != "/doc/tutorial/" {
			body = "different version"
		}
		return FetchedPage{FinalURL: finalURL, StatusCode: 200, RobotsAllowed: true, ContentType: "text/html", Body: []byte(body), Links: []string{"index.html"}}, nil
	}))
	_, _, err = service.CrawlCaptured(context.Background(), policy)
	require.ErrorContains(t, err, "changed between redirect aliases")
}
