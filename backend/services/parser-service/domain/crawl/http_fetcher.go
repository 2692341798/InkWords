package crawl

import (
	"context"
	"fmt"
	"net/url"

	platformcrawler "inkwords-backend/shared/platform/crawler"
)

// HTTPFetcher adapts the shared DNS-safe HTTP transport to the crawl domain.
// It deliberately performs no persistence: a returned body is only an
// untrusted candidate for later structure-preserving parsing.
type HTTPFetcher struct{ fetcher *platformcrawler.HTTPFetcher }

func NewHTTPFetcher(fetcher *platformcrawler.HTTPFetcher) *HTTPFetcher {
	return &HTTPFetcher{fetcher: fetcher}
}

func NewDefaultHTTPFetcher() *HTTPFetcher {
	return NewHTTPFetcher(platformcrawler.NewHTTPFetcher())
}

func (fetcher *HTTPFetcher) Fetch(ctx context.Context, target *url.URL, policy Policy) (FetchedPage, error) {
	page, err := fetcher.fetcher.Fetch(ctx, target, policy.RequestTimeout, func(candidate *url.URL) (*url.URL, error) {
		normalized, reason := policy.NormalizeCandidate(candidate.String())
		if reason != "" {
			return nil, fmt.Errorf("redirect rejected: %s", reason)
		}
		return normalized, nil
	})
	if err != nil {
		return FetchedPage{}, err
	}
	return FetchedPage{
		RequestedURL: page.RequestedURL, FinalURL: page.FinalURL, StatusCode: page.StatusCode, ContentType: page.ContentType,
		Body: page.Body, Title: page.Title, HeadingTree: page.HeadingTree, Links: page.Links, RobotsAllowed: page.RobotsAllowed,
		RetryAfter: page.RetryAfter, FetchedAt: page.FetchedAt,
	}, nil
}

// Revalidate maps the persisted validators into a policy-safe conditional request.
func (fetcher *HTTPFetcher) Revalidate(ctx context.Context, target *url.URL, policy Policy, validators CacheValidators) (FetchedPage, error) {
	page, err := fetcher.fetcher.Revalidate(ctx, target, policy.RequestTimeout, platformcrawler.CacheValidators{ETag: validators.ETag, LastModified: validators.LastModified}, func(candidate *url.URL) (*url.URL, error) {
		normalized, reason := policy.NormalizeCandidate(candidate.String())
		if reason != "" {
			return nil, fmt.Errorf("redirect rejected: %s", reason)
		}
		return normalized, nil
	})
	if err != nil {
		return FetchedPage{}, err
	}
	return FetchedPage{RequestedURL: page.RequestedURL, FinalURL: page.FinalURL, StatusCode: page.StatusCode, ContentType: page.ContentType, Body: page.Body, Title: page.Title, HeadingTree: page.HeadingTree, Links: page.Links, RobotsAllowed: page.RobotsAllowed, RetryAfter: page.RetryAfter, FetchedAt: page.FetchedAt, ETag: page.ETag, LastModified: page.LastModified}, nil
}
