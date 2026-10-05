package crawl

import (
	"fmt"
	"net"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"
)

// Policy defines the explicit boundary of a documentation-stack import.
type Policy struct {
	EntryURL            string
	AllowedHosts        []string
	AllowedPathPrefixes []string
	MaxDepth            int
	MaxPages            int
	MaxTotalBytes       int64
	MaxPageBytes        int64
	RequestTimeout      time.Duration
	TotalTimeout        time.Duration
	MaxConcurrent       int
}

// DefaultPolicy creates conservative defaults around one public HTTPS entry URL.
func DefaultPolicy(entryURL string) (Policy, error) {
	entry, err := url.Parse(entryURL)
	if err != nil {
		return Policy{}, fmt.Errorf("parse entry URL: %w", err)
	}
	return Policy{
		EntryURL:            entry.String(),
		AllowedHosts:        []string{strings.ToLower(entry.Hostname())},
		AllowedPathPrefixes: []string{"/"},
		MaxDepth:            4,
		MaxPages:            200,
		MaxTotalBytes:       32 << 20,
		MaxPageBytes:        1 << 20,
		RequestTimeout:      10 * time.Second,
		TotalTimeout:        2 * time.Minute,
		// The first implementation is deliberately serial. It is a strict bound
		// of one request, which makes ordering, Retry-After, and manifests stable.
		MaxConcurrent: 1,
	}, nil
}

// Validate proves that a policy has finite budgets and cannot start from an internal URL.
func (policy Policy) Validate() error {
	entry, err := url.Parse(policy.EntryURL)
	if err != nil {
		return fmt.Errorf("parse entry URL: %w", err)
	}
	if err := validatePublicHTTPURL(entry); err != nil {
		return fmt.Errorf("invalid entry URL: %w", err)
	}
	if len(policy.AllowedHosts) == 0 || len(policy.AllowedPathPrefixes) == 0 {
		return fmt.Errorf("allowed hosts and path prefixes are required")
	}
	if policy.MaxDepth < 0 || policy.MaxPages < 1 || policy.MaxTotalBytes < 1 || policy.MaxPageBytes < 1 || policy.RequestTimeout <= 0 || policy.TotalTimeout <= 0 || policy.MaxConcurrent < 1 {
		return fmt.Errorf("crawl policy requires positive finite budgets")
	}
	if policy.MaxPageBytes > policy.MaxTotalBytes {
		return fmt.Errorf("page byte budget cannot exceed total byte budget")
	}
	if !policy.allowsHost(entry.Hostname()) || !policy.allowsPath(entry.EscapedPath()) {
		return fmt.Errorf("entry URL is outside its own allow-list")
	}
	return nil
}

// NormalizeCandidate validates and canonicalizes a discovered URL before it enters the queue.
func (policy Policy) NormalizeCandidate(raw string) (*url.URL, string) {
	candidate, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, "invalid_url"
	}
	if err := validatePublicHTTPURL(candidate); err != nil {
		return nil, "unsafe_url"
	}
	if !policy.allowsHost(candidate.Hostname()) {
		return nil, "cross_origin"
	}
	originalPath := candidate.EscapedPath()
	candidate.Path = path.Clean("/" + strings.TrimPrefix(originalPath, "/"))
	// A trailing slash changes relative-link resolution (guide/ + install versus guide + install),
	// so canonicalization must not silently turn a documentation directory into a file URL.
	if strings.HasSuffix(originalPath, "/") && candidate.Path != "/" {
		candidate.Path += "/"
	}
	candidate.RawPath = ""
	if !policy.allowsPath(candidate.Path) {
		return nil, "path_outside_boundary"
	}
	if hasTraversalQuery(candidate.Query()) {
		return nil, "dynamic_or_session_url"
	}
	candidate.Fragment = ""
	return candidate, ""
}

// CandidateKey is the stable crawl-queue identity for an allowed URL. The
// request URL deliberately retains a trailing slash because it changes how
// relative links resolve. The queue, however, treats /guide and /guide/ as
// one source identity so navigation aliases cannot create duplicate snapshots.
func (policy Policy) CandidateKey(candidate *url.URL) string {
	if candidate == nil {
		return ""
	}
	key := *candidate
	if key.Path != "/" {
		key.Path = strings.TrimRight(key.Path, "/")
	}
	key.RawPath = ""
	return key.String()
}

func (policy Policy) allowsHost(host string) bool {
	return slices.ContainsFunc(policy.AllowedHosts, func(allowed string) bool {
		return strings.EqualFold(strings.TrimSpace(allowed), host)
	})
}

func (policy Policy) allowsPath(candidate string) bool {
	for _, prefix := range policy.AllowedPathPrefixes {
		cleanPrefix := path.Clean("/" + strings.TrimPrefix(strings.TrimSpace(prefix), "/"))
		if cleanPrefix == "/" || candidate == cleanPrefix || strings.HasPrefix(candidate, cleanPrefix+"/") {
			return true
		}
	}
	return false
}

func validatePublicHTTPURL(candidate *url.URL) error {
	if candidate == nil || (candidate.Scheme != "https" && candidate.Scheme != "http") || candidate.User != nil || candidate.Hostname() == "" {
		return fmt.Errorf("only unauthenticated HTTP(S) URLs are allowed")
	}
	host := strings.ToLower(candidate.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return fmt.Errorf("localhost is not allowed")
	}
	if ip := net.ParseIP(host); ip != nil && !isPublicIP(ip) {
		return fmt.Errorf("private or special-use IP is not allowed")
	}
	return nil
}

func hasTraversalQuery(values url.Values) bool {
	for key := range values {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if normalized == "q" || normalized == "query" || normalized == "search" || normalized == "page" || normalized == "p" || strings.Contains(normalized, "session") || strings.Contains(normalized, "token") || normalized == "sid" {
			return true
		}
	}
	return false
}

// isPublicIP rejects loopback, private, link-local, multicast, unspecified and ULA ranges.
func isPublicIP(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsMulticast() && !ip.IsUnspecified() && !ip.IsInterfaceLocalMulticast() && !ip.IsLinkLocalMulticast()
}
