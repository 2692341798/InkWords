package crawl

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"time"
)

// ErrRemoteContentChanged tells callers to create a new source snapshot. It
// prevents stale bytes from being reused merely because a conditional request
// was attempted.
var ErrRemoteContentChanged = fmt.Errorf("official documentation changed during revalidation")

// Service owns bounded crawl orchestration and makes partial completion explicit.
type Service struct {
	fetcher     Fetcher
	checkpoints CheckpointStore
}

// NewService wires a policy-agnostic fetcher to the bounded crawl workflow. A
// checkpoint store is optional only for isolated tests; production must provide
// one so cancelled work can be resumed after the process restarts.
func NewService(fetcher Fetcher, checkpoints ...CheckpointStore) *Service {
	service := &Service{fetcher: fetcher}
	if len(checkpoints) > 0 {
		service.checkpoints = checkpoints[0]
	}
	return service
}

type queueItem struct {
	url    *url.URL
	depth  int
	parent string
}

// CapturedPage is one policy-approved page retained only for the duration of
// a structured import. The regular manifest intentionally keeps hashes and
// metadata only, so callers cannot mistake its presence for durable content.
type CapturedPage struct {
	Manifest PageManifest
	Body     []byte
}

type recordingFetcher struct {
	Fetcher
	pages map[string]FetchedPage
}

func (fetcher *recordingFetcher) Fetch(ctx context.Context, candidate *url.URL, policy Policy) (FetchedPage, error) {
	page, err := fetcher.Fetcher.Fetch(ctx, candidate, policy)
	if err == nil && page.StatusCode >= 200 && page.StatusCode < 300 && page.RobotsAllowed && candidate != nil {
		fetcher.pages[candidate.String()] = page
	}
	return page, err
}

// Crawl produces a deterministic manifest. Hitting a budget returns a partial
// manifest, never a misleading claim that the whole documentation stack was imported.
func (service *Service) Crawl(ctx context.Context, policy Policy) (Manifest, error) {
	if service == nil || service.fetcher == nil {
		return Manifest{}, fmt.Errorf("crawl fetcher is not configured")
	}
	if err := policy.Validate(); err != nil {
		return Manifest{}, err
	}
	entry, _ := url.Parse(policy.EntryURL)
	return service.crawl(ctx, policy, Manifest{EntryURL: entry.String(), Policy: policy, Status: "complete"}, []queueItem{{url: entry}}, map[string]struct{}{})
}

// CrawlCaptured runs the same bounded workflow as Crawl, then returns source
// bytes only for manifest pages that were actually kept. The capture is
// in-memory and short-lived; a later caller must still validate and persist it
// as an immutable source snapshot.
func (service *Service) CrawlCaptured(ctx context.Context, policy Policy) (Manifest, []CapturedPage, error) {
	if service == nil || service.fetcher == nil {
		return Manifest{}, nil, fmt.Errorf("crawl fetcher is not configured")
	}
	collector := &recordingFetcher{Fetcher: service.fetcher, pages: make(map[string]FetchedPage)}
	copied := *service
	copied.fetcher = collector
	manifest, err := copied.Crawl(ctx, policy)
	if err != nil {
		return Manifest{}, nil, err
	}
	captured := make([]CapturedPage, 0, len(manifest.Pages))
	for _, page := range manifest.Pages {
		fetched, ok := collector.pages[page.URL]
		if !ok {
			return Manifest{}, nil, fmt.Errorf("captured page body is unavailable for %s", page.URL)
		}
		captured = append(captured, CapturedPage{Manifest: page, Body: append([]byte(nil), fetched.Body...)})
	}
	return manifest, captured, nil
}

// Resume continues an interrupted crawl under exactly the policy that produced
// it. Callers cannot quietly widen host/path or resource limits via this route.
func (service *Service) Resume(ctx context.Context, checkpointID string) (Manifest, error) {
	if service == nil || service.checkpoints == nil {
		return Manifest{}, fmt.Errorf("crawl checkpoint store is not configured")
	}
	state, err := service.checkpoints.Load(ctx, checkpointID)
	if err != nil {
		return Manifest{}, err
	}
	if state.Manifest.Status != "partial" || len(state.Queue) == 0 {
		return Manifest{}, fmt.Errorf("crawl checkpoint cannot be resumed")
	}
	queue := make([]queueItem, 0, len(state.Queue))
	for _, item := range state.Queue {
		parsed, err := url.Parse(item.URL)
		if err != nil {
			return Manifest{}, fmt.Errorf("invalid crawl checkpoint queue: %w", err)
		}
		queue = append(queue, queueItem{url: parsed, depth: item.Depth, parent: item.Parent})
	}
	seen := make(map[string]struct{}, len(state.Seen))
	for _, key := range state.Seen {
		seen[key] = struct{}{}
	}
	manifest := state.Manifest
	manifest.CheckpointID, manifest.CheckpointURL, manifest.StopReason, manifest.Status = "", "", "", "complete"
	return service.crawl(ctx, state.Policy, manifest, queue, seen)
}

// Revalidate performs conditional requests against one completed manifest. If
// every page returns 304, it preserves the exact source input hash so callers
// can avoid scheduling duplicate generation. Any changed page fails closed:
// callers must start a new crawl and persist a new immutable snapshot.
func (service *Service) Revalidate(ctx context.Context, previous Manifest) (Manifest, error) {
	if service == nil || service.fetcher == nil {
		return Manifest{}, fmt.Errorf("crawl fetcher is not configured")
	}
	if previous.Status != "complete" || len(previous.Pages) == 0 || previous.EntryURL != previous.Policy.EntryURL {
		return Manifest{}, fmt.Errorf("only a completed crawl manifest can be revalidated")
	}
	if err := previous.Policy.Validate(); err != nil {
		return Manifest{}, fmt.Errorf("validate previous crawl policy: %w", err)
	}
	if previous.SnapshotInputHash == "" || previous.SnapshotInputHash != SnapshotInputHash(previous) {
		return Manifest{}, fmt.Errorf("crawl manifest snapshot input hash is invalid")
	}
	revalidator, ok := service.fetcher.(RevalidatingFetcher)
	if !ok {
		return Manifest{}, fmt.Errorf("crawl fetcher does not support HTTP revalidation")
	}
	pages := append([]PageManifest(nil), previous.Pages...)
	sort.Slice(pages, func(i, j int) bool { return pages[i].URL < pages[j].URL })
	for _, prior := range pages {
		if prior.ETag == "" && prior.LastModified == "" {
			return Manifest{}, fmt.Errorf("crawl page %q has no HTTP revalidation validator", prior.URL)
		}
		candidate, reason := previous.Policy.NormalizeCandidate(prior.URL)
		if reason != "" {
			return Manifest{}, fmt.Errorf("crawl manifest contains disallowed page %q: %s", prior.URL, reason)
		}
		page, err := revalidator.Revalidate(ctx, candidate, previous.Policy, CacheValidators{ETag: prior.ETag, LastModified: prior.LastModified})
		if err != nil {
			return Manifest{}, fmt.Errorf("revalidate %s: %w", candidate, err)
		}
		if !page.RobotsAllowed || page.StatusCode != 304 {
			return Manifest{}, ErrRemoteContentChanged
		}
		if page.FinalURL == nil {
			return Manifest{}, fmt.Errorf("revalidation returned no final URL for %s", candidate)
		}
		if _, reason := previous.Policy.NormalizeCandidate(page.FinalURL.String()); reason != "" {
			return Manifest{}, fmt.Errorf("revalidation redirect rejected: %s", reason)
		}
	}
	manifest := previous
	manifest.Unchanged = true
	manifest.RevalidatedAt = time.Now().UTC()
	manifest.SnapshotInputHash = SnapshotInputHash(manifest)
	return manifest, nil
}

func (service *Service) crawl(parent context.Context, policy Policy, manifest Manifest, queue []queueItem, seen map[string]struct{}) (Manifest, error) {
	ctx, cancel := context.WithTimeout(parent, policy.TotalTimeout)
	defer cancel()
	var nextRequestAt time.Time
	canonicalHashes := make(map[string]string, len(manifest.Pages))
	for _, page := range manifest.Pages {
		canonicalHashes[page.CanonicalURL] = page.ContentHash
	}
	for len(queue) > 0 {
		if ctx.Err() != nil {
			return service.partial(parent, manifest, policy, queue, seen, stopReason(parent, ctx))
		}
		item := queue[0]
		candidate, reason := policy.NormalizeCandidate(item.url.String())
		if reason != "" {
			queue = queue[1:]
			manifest.Decisions = append(manifest.Decisions, decision(item, reason))
			continue
		}
		// Keep candidate itself for fetching and relative-link resolution. The
		// identity key collapses only trailing-slash navigation aliases.
		key := policy.CandidateKey(candidate)
		if _, exists := seen[key]; exists {
			queue = queue[1:]
			manifest.Decisions = append(manifest.Decisions, decision(item, "duplicate"))
			continue
		}
		if len(manifest.Pages) >= policy.MaxPages {
			return service.partial(parent, manifest, policy, queue, seen, "max_pages")
		}
		if delay := time.Until(nextRequestAt); delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return service.partial(parent, manifest, policy, queue, seen, stopReason(parent, ctx))
			case <-timer.C:
			}
		}

		// This item is now in-flight. If its request is cancelled, delete the
		// mark and keep it at the front of the persisted queue for a safe retry.
		seen[key] = struct{}{}
		page, err := service.fetcher.Fetch(ctx, candidate, policy)
		if err != nil {
			if ctx.Err() != nil {
				delete(seen, key)
				return service.partial(parent, manifest, policy, queue, seen, stopReason(parent, ctx))
			}
			queue = queue[1:]
			manifest.Decisions = append(manifest.Decisions, decision(item, "fetch_failed"))
			continue
		}
		if !page.RobotsAllowed {
			queue = queue[1:]
			manifest.Decisions = append(manifest.Decisions, decision(item, "robots_disallowed"))
			continue
		}
		if page.RetryAfter > 0 {
			nextRequestAt = time.Now().Add(page.RetryAfter)
		}
		if page.FinalURL == nil {
			queue = queue[1:]
			manifest.Decisions = append(manifest.Decisions, decision(item, "invalid_final_url"))
			continue
		}
		finalURL, reason := policy.NormalizeCandidate(page.FinalURL.String())
		if reason != "" {
			queue = queue[1:]
			manifest.Decisions = append(manifest.Decisions, decision(item, "redirect_"+reason))
			continue
		}
		if page.StatusCode < 200 || page.StatusCode >= 300 {
			queue = queue[1:]
			manifest.Decisions = append(manifest.Decisions, decision(item, fmt.Sprintf("http_%d", page.StatusCode)))
			continue
		}
		if int64(len(page.Body)) > policy.MaxPageBytes || manifest.TotalBytes+int64(len(page.Body)) > policy.MaxTotalBytes {
			delete(seen, key)
			return service.partial(parent, manifest, policy, queue, seen, "byte_budget")
		}
		queue = queue[1:]
		manifest.TotalBytes += int64(len(page.Body))
		contentHash := digest(page.Body)
		if previousHash, exists := canonicalHashes[finalURL.String()]; exists {
			if previousHash != contentHash {
				return manifest, fmt.Errorf("official documentation changed between redirect aliases")
			}
			manifest.Decisions = append(manifest.Decisions, decision(item, "duplicate_destination"))
			continue
		}
		canonicalHashes[finalURL.String()] = contentHash
		seen[policy.CandidateKey(finalURL)] = struct{}{}
		fetchedAt := page.FetchedAt
		if fetchedAt.IsZero() {
			fetchedAt = time.Now().UTC()
		}
		manifest.Pages = append(manifest.Pages, PageManifest{URL: candidate.String(), CanonicalURL: finalURL.String(), Title: page.Title, HeadingTree: append([]string(nil), page.HeadingTree...), ContentHash: contentHash, FetchedAt: fetchedAt, StatusCode: page.StatusCode, ContentType: page.ContentType, ETag: page.ETag, LastModified: page.LastModified, ParentURL: item.parent, Depth: item.depth})
		manifest.Decisions = append(manifest.Decisions, EntryDecision{URL: candidate.String(), Status: "kept", Depth: item.depth, ParentURL: item.parent, At: fetchedAt})
		if item.depth >= policy.MaxDepth {
			continue
		}
		links := slices.Clone(page.Links)
		slices.Sort(links)
		for _, link := range links {
			nextURL, err := finalURL.Parse(link)
			if err != nil {
				manifest.Decisions = append(manifest.Decisions, EntryDecision{URL: link, Status: "skipped", Reason: "invalid_url", Depth: item.depth + 1, ParentURL: candidate.String(), At: fetchedAt})
				continue
			}
			queue = append(queue, queueItem{url: nextURL, depth: item.depth + 1, parent: candidate.String()})
		}
	}
	manifest.SnapshotInputHash = SnapshotInputHash(manifest)
	return manifest, nil
}

func (service *Service) partial(parent context.Context, manifest Manifest, policy Policy, queue []queueItem, seen map[string]struct{}, reason string) (Manifest, error) {
	manifest.Status, manifest.StopReason = "partial", reason
	if len(queue) > 0 {
		manifest.CheckpointURL = queue[0].url.String()
	}
	if service.checkpoints == nil {
		manifest.SnapshotInputHash = SnapshotInputHash(manifest)
		return manifest, nil
	}
	state := checkpoint{Policy: policy, Manifest: manifest, Queue: checkpointQueueItems(queue), Seen: sortedKeys(seen)}
	// A cancelled request must still be able to persist the recovery point.
	checkpointID, err := service.checkpoints.Save(context.WithoutCancel(parent), state)
	if err != nil {
		return Manifest{}, fmt.Errorf("save crawl checkpoint: %w", err)
	}
	manifest.CheckpointID = checkpointID
	manifest.SnapshotInputHash = SnapshotInputHash(manifest)
	return manifest, nil
}

func checkpointQueueItems(queue []queueItem) []checkpointQueue {
	items := make([]checkpointQueue, 0, len(queue))
	for _, item := range queue {
		items = append(items, checkpointQueue{URL: item.url.String(), Depth: item.depth, Parent: item.parent})
	}
	return items
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func stopReason(parent, timed context.Context) string {
	if parent.Err() != nil {
		return "cancelled"
	}
	if timed.Err() != nil {
		return "total_timeout"
	}
	return "interrupted"
}

func decision(item queueItem, reason string) EntryDecision {
	return EntryDecision{URL: item.url.String(), Status: "skipped", Reason: reason, Depth: item.depth, ParentURL: item.parent, At: time.Now().UTC()}
}

func digest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}
