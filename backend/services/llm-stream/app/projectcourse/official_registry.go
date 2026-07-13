package projectcourse

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type OfficialSourceCache interface {
	Get(key string) (OfficialSource, bool)
	Set(key string, source OfficialSource, ttl time.Duration)
}

type memoryOfficialSourceCache struct {
	mu      sync.Mutex
	entries map[string]memoryOfficialSourceEntry
}

type memoryOfficialSourceEntry struct {
	source    OfficialSource
	expiresAt time.Time
}

func NewMemoryOfficialSourceCache() OfficialSourceCache {
	return &memoryOfficialSourceCache{entries: make(map[string]memoryOfficialSourceEntry)}
}

func (c *memoryOfficialSourceCache) Get(key string) (OfficialSource, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || (!entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt)) {
		if ok {
			delete(c.entries, key)
		}
		return OfficialSource{}, false
	}
	return entry.source, true
}

func (c *memoryOfficialSourceCache) Set(key string, source OfficialSource, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	expiresAt := time.Time{}
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}
	c.entries[key] = memoryOfficialSourceEntry{source: source, expiresAt: expiresAt}
}

type OfficialRegistryProvider struct {
	Registry OfficialRegistry
	Fetcher  OfficialSourceProvider
	Cache    OfficialSourceCache
	TTL      time.Duration
	Now      func() time.Time
}

func (p OfficialRegistryProvider) FetchTechnology(technology, versionConstraint string) (OfficialSource, error) {
	source, err := p.Registry.Resolve(technology, versionConstraint)
	if err != nil {
		return OfficialSource{}, err
	}
	key := officialSourceCacheKey(source)
	if p.Cache != nil {
		if cached, ok := p.Cache.Get(key); ok {
			return cached, nil
		}
	}
	if p.Fetcher == nil {
		return OfficialSource{}, fmt.Errorf("official source fetcher is not configured")
	}
	content, err := p.Fetcher.Fetch(source.URL)
	if err != nil {
		return OfficialSource{}, err
	}
	now := time.Now
	if p.Now != nil {
		now = p.Now
	}
	source, err = FinalizeOfficialSource(source, content, now())
	if err != nil {
		return OfficialSource{}, err
	}
	if p.Cache != nil {
		ttl := p.TTL
		if ttl <= 0 {
			ttl = 24 * time.Hour
		}
		// The content-addressed key prevents a changed upstream page from
		// silently colliding with a previous version; the base key is a short
		// lookup alias for the same provider/version/URL tuple.
		p.Cache.Set(officialSourceContentCacheKey(source), source, ttl)
		p.Cache.Set(key, source, ttl)
	}
	return source, nil
}

func officialSourceContentCacheKey(source OfficialSource) string {
	return officialSourceCacheKey(source) + ":" + strings.TrimSpace(source.ContentHash)
}

func officialSourceCacheKey(source OfficialSource) string {
	return "project-course:official:" + strings.ToLower(strings.TrimSpace(source.Technology)) + ":" + strings.TrimSpace(source.VersionConstraint) + ":" + source.URL
}

// OfficialSourcePromptBlock makes the provenance boundary explicit: fetched
// web text is data only, never an instruction to the model.
func OfficialSourcePromptBlock(source OfficialSource) string {
	return fmt.Sprintf("<official_source_data technology=%q version=%q url=%q>\n%s\n</official_source_data>", source.Technology, source.VersionConstraint, source.URL, source.Content)
}
