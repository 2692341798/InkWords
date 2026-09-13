package crawl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"
)

// EntryDecision records every discovered URL, including policy skips and fetch failures.
type EntryDecision struct {
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason,omitempty"`
	Depth     int       `json:"depth"`
	ParentURL string    `json:"parent_url,omitempty"`
	At        time.Time `json:"at"`
}

// PageManifest is the immutable per-page evidence record produced by a crawl.
type PageManifest struct {
	URL          string    `json:"url"`
	CanonicalURL string    `json:"canonical_url"`
	Title        string    `json:"title,omitempty"`
	HeadingTree  []string  `json:"heading_tree,omitempty"`
	ContentHash  string    `json:"content_hash"`
	FetchedAt    time.Time `json:"fetched_at"`
	StatusCode   int       `json:"status_code"`
	ContentType  string    `json:"content_type"`
	ETag         string    `json:"etag,omitempty"`
	LastModified string    `json:"last_modified,omitempty"`
	ParentURL    string    `json:"parent_url,omitempty"`
	Depth        int       `json:"depth"`
}

// Manifest is a resumable, explainable description of an official-document import.
type Manifest struct {
	EntryURL   string          `json:"entry_url"`
	Policy     Policy          `json:"policy"`
	Status     string          `json:"status"`
	StopReason string          `json:"stop_reason,omitempty"`
	TotalBytes int64           `json:"total_bytes"`
	Pages      []PageManifest  `json:"pages"`
	Decisions  []EntryDecision `json:"decisions"`
	// CheckpointID identifies an on-disk, resumable crawl state. It is present
	// only while a crawl stopped before the policy-complete manifest was built.
	CheckpointID  string `json:"checkpoint_id,omitempty"`
	CheckpointURL string `json:"checkpoint_url,omitempty"`
	// SnapshotInputHash identifies the policy and immutable document bytes used
	// downstream. It deliberately excludes retrieval time and HTTP validators.
	SnapshotInputHash string    `json:"snapshot_input_hash,omitempty"`
	Unchanged         bool      `json:"unchanged,omitempty"`
	RevalidatedAt     time.Time `json:"revalidated_at,omitempty"`
}

// SnapshotInputHash is stable when the allowed crawl policy and every captured
// source byte remain unchanged. Callers can use it as the source input in an
// idempotency key before enqueueing any generation work.
func SnapshotInputHash(manifest Manifest) string {
	type pageIdentity struct {
		URL, CanonicalURL, ContentHash string
	}
	pages := make([]pageIdentity, 0, len(manifest.Pages))
	for _, page := range manifest.Pages {
		pages = append(pages, pageIdentity{URL: page.URL, CanonicalURL: page.CanonicalURL, ContentHash: page.ContentHash})
	}
	sort.Slice(pages, func(i, j int) bool {
		if pages[i].CanonicalURL == pages[j].CanonicalURL {
			return pages[i].URL < pages[j].URL
		}
		return pages[i].CanonicalURL < pages[j].CanonicalURL
	})
	identity := struct {
		EntryURL string         `json:"entry_url"`
		Policy   Policy         `json:"policy"`
		Pages    []pageIdentity `json:"pages"`
	}{EntryURL: manifest.EntryURL, Policy: manifest.Policy, Pages: pages}
	encoded, _ := json.Marshal(identity)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}
