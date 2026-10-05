package textbook

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"

	sharedgeneration "inkwords-backend/shared/kernel/generation"
)

// GenerationCacheKey makes reuse auditable and invalidates it when any teaching input changes.
type GenerationCacheKey struct {
	SnapshotHash     string
	Audience         string
	BookContractHash string
	StyleSheetHash   string
	Stage            string
	EvidenceHash     string
	PromptSchema     string
	QualityContract  string
	RequestHash      string
	Provider         string
	Model            string
}

func (key GenerationCacheKey) Digest() string {
	parts := []string{key.SnapshotHash, key.Audience, key.BookContractHash, key.StyleSheetHash, key.Stage, key.EvidenceHash, key.PromptSchema, key.QualityContract, key.RequestHash, key.Provider, key.Model}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// GenerationResultCache keeps only a response accepted by candidate validation.
// Callers must still run the current quality gates on every cache hit.
type GenerationResultCache interface {
	Load(string) (sharedgeneration.Result, bool)
	Store(string, sharedgeneration.Result)
}

// MemoryGenerationResultCache is a process-local implementation for the local
// single-user runtime. It is intentionally injectable so a durable cache can
// later preserve the same invalidation contract without leaking providers.
type MemoryGenerationResultCache struct {
	mu      sync.RWMutex
	results map[string]sharedgeneration.Result
}

func NewMemoryGenerationResultCache() *MemoryGenerationResultCache {
	return &MemoryGenerationResultCache{results: make(map[string]sharedgeneration.Result)}
}

func (cache *MemoryGenerationResultCache) Load(key string) (sharedgeneration.Result, bool) {
	if cache == nil {
		return sharedgeneration.Result{}, false
	}
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	result, ok := cache.results[key]
	return result, ok
}

func (cache *MemoryGenerationResultCache) Store(key string, result sharedgeneration.Result) {
	if cache == nil {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.results[key] = result
}
