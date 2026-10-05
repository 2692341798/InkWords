package crawl

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fixtureFetcher struct{ pages map[string]FetchedPage }

func (fetcher fixtureFetcher) Fetch(_ context.Context, target *url.URL, _ Policy) (FetchedPage, error) {
	page, ok := fetcher.pages[target.String()]
	if !ok {
		return FetchedPage{}, fmt.Errorf("missing fixture for %s", target)
	}
	page.RequestedURL = target
	return page, nil
}

type functionFetcher func(context.Context, *url.URL, Policy) (FetchedPage, error)

func (fetcher functionFetcher) Fetch(ctx context.Context, target *url.URL, policy Policy) (FetchedPage, error) {
	return fetcher(ctx, target, policy)
}

type revalidatingFixtureFetcher struct {
	pages      map[string]FetchedPage
	validators map[string]CacheValidators
}

func (fetcher *revalidatingFixtureFetcher) Fetch(_ context.Context, target *url.URL, _ Policy) (FetchedPage, error) {
	page, ok := fetcher.pages[target.String()]
	if !ok {
		return FetchedPage{}, fmt.Errorf("missing fixture for %s", target)
	}
	page.RequestedURL = target
	return page, nil
}

func (fetcher *revalidatingFixtureFetcher) Revalidate(_ context.Context, target *url.URL, _ Policy, validators CacheValidators) (FetchedPage, error) {
	if fetcher.validators == nil {
		fetcher.validators = map[string]CacheValidators{}
	}
	fetcher.validators[target.String()] = validators
	page, ok := fetcher.pages[target.String()]
	if !ok {
		return FetchedPage{}, fmt.Errorf("missing revalidation fixture for %s", target)
	}
	page.RequestedURL = target
	return page, nil
}

func TestPolicyRejectsUnsafeURLsAndDynamicLoops(t *testing.T) {
	policy, err := DefaultPolicy("https://docs.example.test/guide/")
	require.NoError(t, err)
	policy.AllowedPathPrefixes = []string{"/guide"}
	require.NoError(t, policy.Validate())
	for _, raw := range []string{"file:///etc/passwd", "http://127.0.0.1/admin", "https://evil.example/guide", "https://docs.example.test/search?q=gin", "https://docs.example.test/other"} {
		candidate, reason := policy.NormalizeCandidate(raw)
		require.Nil(t, candidate, raw)
		require.NotEmpty(t, reason, raw)
	}
	candidate, reason := policy.NormalizeCandidate("https://docs.example.test/guide/install#linux")
	require.Empty(t, reason)
	require.Equal(t, "https://docs.example.test/guide/install", candidate.String())
}

func TestServiceCrawlBuildsDeterministicManifestAndReturnsPartialBudget(t *testing.T) {
	policy, err := DefaultPolicy("https://docs.example.test/guide/")
	require.NoError(t, err)
	policy.MaxPages = 1
	policy.MaxDepth = 2
	policy.TotalTimeout = time.Second
	pageURL, _ := url.Parse("https://docs.example.test/guide/")
	service := NewService(fixtureFetcher{pages: map[string]FetchedPage{
		pageURL.String(): {FinalURL: pageURL, StatusCode: 200, ContentType: "text/html", Body: []byte("gin docs"), Title: "Guide", HeadingTree: []string{"Guide"}, Links: []string{"/guide/install", "https://evil.example/payload"}, RobotsAllowed: true, FetchedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)},
	}})
	manifest, err := service.Crawl(context.Background(), policy)
	require.NoError(t, err)
	require.Equal(t, "partial", manifest.Status)
	require.Equal(t, "max_pages", manifest.StopReason)
	require.Len(t, manifest.Pages, 1)
	require.Equal(t, "sha256:f2acc2c9088fa5089a4a8e0549d5cf0b33934c1621c9ebe316db8dc578054cc7", manifest.Pages[0].ContentHash)
	require.Equal(t, "https://docs.example.test/guide/install", manifest.CheckpointURL)
	require.Equal(t, "kept", manifest.Decisions[0].Status)
}

func TestServiceCrawlCapturedReturnsOnlyKeptPageBodies(t *testing.T) {
	policy, err := DefaultPolicy("https://docs.example.test/guide/")
	require.NoError(t, err)
	policy.AllowedPathPrefixes = []string{"/guide"}
	policy.TotalTimeout = time.Second
	entry := mustURL(t, policy.EntryURL)
	service := NewService(fixtureFetcher{pages: map[string]FetchedPage{
		entry.String(): {FinalURL: entry, StatusCode: http.StatusOK, ContentType: "text/html", Body: []byte("<h1>Guide</h1>"), RobotsAllowed: true, Links: []string{"https://other.example/skip"}},
	}})
	manifest, captured, err := service.CrawlCaptured(context.Background(), policy)
	require.NoError(t, err)
	require.Equal(t, "complete", manifest.Status)
	require.Len(t, captured, 1)
	require.Equal(t, entry.String(), captured[0].Manifest.URL)
	require.Equal(t, []byte("<h1>Guide</h1>"), captured[0].Body)
	require.True(t, hasDecision(manifest.Decisions, "https://other.example/skip", "cross_origin"))
}

func TestServiceCrawlRecordsRobotsAndCrossOriginDecisions(t *testing.T) {
	policy, err := DefaultPolicy("https://docs.example.test/guide/")
	require.NoError(t, err)
	policy.AllowedPathPrefixes = []string{"/guide"}
	policy.MaxDepth = 1
	policy.TotalTimeout = time.Second
	entry, _ := url.Parse(policy.EntryURL)
	service := NewService(fixtureFetcher{pages: map[string]FetchedPage{
		entry.String(): {FinalURL: entry, StatusCode: 200, ContentType: "text/html", Body: []byte("root"), RobotsAllowed: true, Links: []string{"/guide/blocked", "https://elsewhere.test/x"}},
		"https://docs.example.test/guide/blocked": {FinalURL: mustURL(t, "https://docs.example.test/guide/blocked"), StatusCode: 200, Body: []byte("blocked"), RobotsAllowed: false},
	}})
	manifest, err := service.Crawl(context.Background(), policy)
	require.NoError(t, err)
	require.Len(t, manifest.Pages, 1)
	require.True(t, hasDecision(manifest.Decisions, "https://docs.example.test/guide/blocked", "robots_disallowed"))
	require.True(t, hasDecision(manifest.Decisions, "https://elsewhere.test/x", "cross_origin"))
}

func TestServiceCrawlKeepsBoundedCPlusPlusOfficialSiteManifest(t *testing.T) {
	// These are structural fixtures, not a claim about the current live pages.
	// They cover an official-site entry page whose navigation reaches a deeper
	// page, while proving that a discovered third-party link stays outside the
	// immutable source snapshot.
	policy, err := DefaultPolicy("https://isocpp.org/get-started")
	require.NoError(t, err)
	policy.MaxDepth = 1
	policy.MaxPages = 4
	policy.TotalTimeout = time.Second
	entry := mustURL(t, policy.EntryURL)
	tour := mustURL(t, "https://isocpp.org/tour")
	service := NewService(fixtureFetcher{pages: map[string]FetchedPage{
		entry.String(): {FinalURL: entry, StatusCode: http.StatusOK, ContentType: "text/html", Body: []byte("cpp-entry"), Title: "C++ 入门", HeadingTree: []string{"开始使用 C++"}, Links: []string{"/get-started/", "/tour", "/tour/", "https://third-party.example/cpp"}, RobotsAllowed: true, FetchedAt: time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)},
		tour.String():  {FinalURL: tour, StatusCode: http.StatusOK, ContentType: "text/html", Body: []byte("cpp-tour"), Title: "C++ 导览", HeadingTree: []string{"语言与标准库导览", "下一步"}, RobotsAllowed: true, FetchedAt: time.Date(2026, 9, 3, 0, 0, 1, 0, time.UTC)},
	}})

	manifest, err := service.Crawl(context.Background(), policy)
	require.NoError(t, err)
	require.Equal(t, "complete", manifest.Status)
	require.Len(t, manifest.Pages, 2)
	require.Equal(t, entry.String(), manifest.Pages[0].URL)
	require.Equal(t, []string{"开始使用 C++"}, manifest.Pages[0].HeadingTree)
	require.Equal(t, tour.String(), manifest.Pages[1].URL)
	require.Equal(t, entry.String(), manifest.Pages[1].ParentURL)
	require.Equal(t, []string{"语言与标准库导览", "下一步"}, manifest.Pages[1].HeadingTree)
	require.True(t, hasDecision(manifest.Decisions, "https://isocpp.org/get-started/", "duplicate"))
	require.True(t, hasDecision(manifest.Decisions, "https://isocpp.org/tour/", "duplicate"))
	require.True(t, hasDecision(manifest.Decisions, "https://third-party.example/cpp", "cross_origin"))
	require.NotEmpty(t, manifest.SnapshotInputHash)
}

func TestServiceResumeReplaysPersistedQueueAfterCancellation(t *testing.T) {
	policy, err := DefaultPolicy("https://docs.example.test/guide/")
	require.NoError(t, err)
	policy.MaxDepth = 1
	policy.TotalTimeout = time.Second
	entry := mustURL(t, policy.EntryURL)
	next := mustURL(t, "https://docs.example.test/guide/install")
	checkpointRoot := t.TempDir()
	checkpointStore, err := NewFilesystemCheckpointStore(checkpointRoot)
	require.NoError(t, err)
	parent, cancel := context.WithCancel(context.Background())
	service := NewService(functionFetcher(func(_ context.Context, target *url.URL, _ Policy) (FetchedPage, error) {
		switch target.String() {
		case entry.String():
			cancel()
			return FetchedPage{FinalURL: entry, StatusCode: http.StatusOK, Body: []byte("root"), RobotsAllowed: true, Links: []string{"/guide/install"}}, nil
		case next.String():
			return FetchedPage{FinalURL: next, StatusCode: http.StatusOK, Body: []byte("install"), RobotsAllowed: true}, nil
		default:
			return FetchedPage{}, fmt.Errorf("unexpected URL %s", target)
		}
	}), checkpointStore)

	partial, err := service.Crawl(parent, policy)
	require.NoError(t, err)
	require.Equal(t, "partial", partial.Status)
	require.Equal(t, "cancelled", partial.StopReason)
	require.NotEmpty(t, partial.CheckpointID)
	require.Equal(t, next.String(), partial.CheckpointURL)

	// Reopen the store and service to prove this is a process-restart recovery,
	// not an accidental in-memory queue continuation.
	reopenedStore, err := NewFilesystemCheckpointStore(checkpointRoot)
	require.NoError(t, err)
	resumedService := NewService(fixtureFetcher{pages: map[string]FetchedPage{
		next.String(): {FinalURL: next, StatusCode: http.StatusOK, Body: []byte("install"), RobotsAllowed: true},
	}}, reopenedStore)
	resumed, err := resumedService.Resume(context.Background(), partial.CheckpointID)
	require.NoError(t, err)
	require.Equal(t, "complete", resumed.Status)
	require.Empty(t, resumed.CheckpointID)
	require.Len(t, resumed.Pages, 2)
	require.Equal(t, next.String(), resumed.Pages[1].URL)
}

func TestServiceRevalidateReusesUnchangedSnapshotIdentity(t *testing.T) {
	policy, err := DefaultPolicy("https://docs.example.test/guide/")
	require.NoError(t, err)
	policy.TotalTimeout = time.Second
	entry := mustURL(t, policy.EntryURL)
	previous := Manifest{
		EntryURL: policy.EntryURL,
		Policy:   policy,
		Status:   "complete",
		Pages: []PageManifest{{
			URL: entry.String(), CanonicalURL: entry.String(), ContentHash: digest([]byte("stable document")),
			ETag: `"document-v1"`, LastModified: "Tue, 02 Sep 2026 00:00:00 GMT",
		}},
	}
	previous.SnapshotInputHash = SnapshotInputHash(previous)
	fetcher := &revalidatingFixtureFetcher{pages: map[string]FetchedPage{
		entry.String(): {FinalURL: entry, StatusCode: http.StatusNotModified, RobotsAllowed: true},
	}}

	manifest, err := NewService(fetcher).Revalidate(context.Background(), previous)
	require.NoError(t, err)
	require.True(t, manifest.Unchanged)
	require.Equal(t, previous.SnapshotInputHash, manifest.SnapshotInputHash)
	require.Equal(t, CacheValidators{ETag: `"document-v1"`, LastModified: "Tue, 02 Sep 2026 00:00:00 GMT"}, fetcher.validators[entry.String()])
}

func TestServiceRevalidateRejectsChangedContentInsteadOfReusingSnapshot(t *testing.T) {
	policy, err := DefaultPolicy("https://docs.example.test/guide/")
	require.NoError(t, err)
	policy.TotalTimeout = time.Second
	entry := mustURL(t, policy.EntryURL)
	previous := Manifest{EntryURL: policy.EntryURL, Policy: policy, Status: "complete", Pages: []PageManifest{{URL: entry.String(), CanonicalURL: entry.String(), ContentHash: digest([]byte("old document")), ETag: `"document-v1"`}}}
	previous.SnapshotInputHash = SnapshotInputHash(previous)
	fetcher := &revalidatingFixtureFetcher{pages: map[string]FetchedPage{
		entry.String(): {FinalURL: entry, StatusCode: http.StatusOK, Body: []byte("new document"), RobotsAllowed: true},
	}}

	_, err = NewService(fetcher).Revalidate(context.Background(), previous)
	require.ErrorIs(t, err, ErrRemoteContentChanged)
}

func hasDecision(decisions []EntryDecision, rawURL, reason string) bool {
	for _, entry := range decisions {
		if entry.URL == rawURL && entry.Status == "skipped" && entry.Reason == reason {
			return true
		}
	}
	return false
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	return parsed
}
