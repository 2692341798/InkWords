package crawl

import (
	"context"
	"net/url"
	"time"
)

// Fetcher is the infrastructure boundary for one policy-approved HTTP request.
// Implementations must re-check every redirect target and resolved IP address.
type Fetcher interface {
	Fetch(context.Context, *url.URL, Policy) (FetchedPage, error)
}

// CacheValidators are the HTTP validators captured with an immutable page
// snapshot. They are transport metadata only: page bytes remain the source of
// truth for the snapshot content hash.
type CacheValidators struct {
	ETag         string
	LastModified string
}

// RevalidatingFetcher adds conditional HTTP revalidation without weakening the
// baseline Fetcher contract used by deterministic parser tests.
type RevalidatingFetcher interface {
	Revalidate(context.Context, *url.URL, Policy, CacheValidators) (FetchedPage, error)
}

// FetchedPage contains only transport and extracted-document facts; its body remains
// untrusted source material and is never eligible to alter application instructions.
type FetchedPage struct {
	RequestedURL  *url.URL
	FinalURL      *url.URL
	StatusCode    int
	ContentType   string
	Body          []byte
	Title         string
	HeadingTree   []string
	Links         []string
	RobotsAllowed bool
	RetryAfter    time.Duration
	FetchedAt     time.Time
	ETag          string
	LastModified  string
}
