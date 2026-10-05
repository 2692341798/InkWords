package textbook

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

// OfficialWebParsedContentHash separates extraction rules from immutable web
// bytes, so reprocessing cannot silently reuse a snapshot from an older parser.
func OfficialWebParsedContentHash(manifestHash, parserVersion string) string {
	sum := sha256.Sum256([]byte(manifestHash + "\nparser_version=" + parserVersion))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Only the stable cross-process audit fields are interpreted here. The full
// manifest, including HTTP metadata and discovery decisions, remains intact.
func (result OfficialWebImportTaskResult) validateCrawlAudit(payload OfficialWebImportTaskPayload) error {
	if result.ParserVersion != payload.ParserVersion || !fullOfficialDigest(result.ManifestHash) || result.ContentHash != OfficialWebParsedContentHash(result.ManifestHash, result.ParserVersion) || len(result.CrawlManifest) == 0 || len(result.CrawlManifest) > 8<<20 {
		return fmt.Errorf("official web extraction provenance is invalid")
	}
	var audit struct {
		EntryURL          string `json:"entry_url"`
		Status            string `json:"status"`
		SnapshotInputHash string `json:"snapshot_input_hash"`
		TotalBytes        int64  `json:"total_bytes"`
		Policy            struct {
			EntryURL                                                  string
			AllowedHosts                                              []string
			AllowedPathPrefixes                                       []string
			MaxPages, MaxDepth, MaxConcurrent                         int
			MaxTotalBytes, MaxPageBytes, RequestTimeout, TotalTimeout int64
		} `json:"policy"`
		Pages []struct {
			CanonicalURL string `json:"canonical_url"`
			ContentHash  string `json:"content_hash"`
			StatusCode   int    `json:"status_code"`
			ContentType  string `json:"content_type"`
			Depth        int    `json:"depth"`
		} `json:"pages"`
	}
	if json.Unmarshal(result.CrawlManifest, &audit) != nil || audit.EntryURL != payload.EntryURL || audit.Policy.EntryURL != payload.EntryURL || audit.Status != "complete" || audit.SnapshotInputHash != result.ManifestHash {
		return fmt.Errorf("official web crawl audit does not match task")
	}
	entry, _ := url.Parse(payload.EntryURL)
	p := audit.Policy
	if !slices.Equal(p.AllowedHosts, []string{strings.ToLower(entry.Hostname())}) || !slices.Equal(p.AllowedPathPrefixes, payload.AllowedPathPrefixes) || p.MaxPages != 200 || p.MaxDepth != 4 || p.MaxConcurrent != 1 || p.MaxTotalBytes != 32<<20 || p.MaxPageBytes != 1<<20 || p.RequestTimeout != 10_000_000_000 || p.TotalTimeout != 120_000_000_000 || audit.TotalBytes <= 0 || audit.TotalBytes > p.MaxTotalBytes || len(audit.Pages) != len(result.Documents) || len(audit.Pages) > p.MaxPages {
		return fmt.Errorf("official web crawl exceeded frozen policy")
	}
	pages := make(map[string]bool, len(audit.Pages))
	for _, page := range audit.Pages {
		if page.CanonicalURL == "" || pages[page.CanonicalURL] || !fullOfficialDigest(page.ContentHash) || page.StatusCode < 200 || page.StatusCode >= 300 || !strings.Contains(strings.ToLower(page.ContentType), "html") || page.Depth < 0 || page.Depth > p.MaxDepth {
			return fmt.Errorf("official web crawl page audit is invalid")
		}
		pages[page.CanonicalURL] = true
	}
	for _, document := range result.Documents {
		if !pages[document.CanonicalLocator] {
			return fmt.Errorf("official web document has no captured page")
		}
		delete(pages, document.CanonicalLocator)
	}
	return nil
}

func fullOfficialDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != 71 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
