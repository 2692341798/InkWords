// Package crawler contains transport safeguards shared by documentation importers.
// It intentionally returns untrusted page material; deciding whether a URL is in a
// textbook import boundary remains parser-service domain work.
package crawler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

const (
	userAgent    = "InkWordsDocCrawler/1.0"
	maxRedirects = 5
)

// Page is the transport-level result for one page. Its Body is source material,
// never executable instructions or trusted application input.
type Page struct {
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

// CacheValidators are sent only on explicit revalidation requests.
type CacheValidators struct {
	ETag         string
	LastModified string
}

// Resolver permits deterministic DNS-rebinding tests without weakening the
// production constructor's connection-time address validation.
type Resolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

// RedirectValidator lets a service domain apply its own allow-list before a
// redirect becomes a network request. Returning a normalized URL makes URL
// canonicalization and the transport destination identical.
type RedirectValidator func(*url.URL) (*url.URL, error)

// HTTPFetcher makes one bounded, robots-aware HTTP(S) request. The default
// transport disables proxy use and validates DNS answers immediately before it
// dials, so a hostname cannot pivot the crawler into a private network.
type HTTPFetcher struct {
	client   *http.Client
	resolver Resolver
	now      func() time.Time

	robotsMu sync.Mutex
	robots   map[string]robotsRules
}

// NewHTTPFetcher creates the production-safe fetcher used by documentation
// crawls. Tests that need a controlled transport stay package-local.
func NewHTTPFetcher() *HTTPFetcher {
	return newHTTPFetcher(net.DefaultResolver, nil)
}

func newHTTPFetcher(resolver Resolver, client *http.Client) *HTTPFetcher {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	fetcher := &HTTPFetcher{resolver: resolver, now: time.Now, robots: make(map[string]robotsRules)}
	if client != nil {
		// A caller-provided transport is used only by package tests. Redirect policy
		// remains owned here so test transports cannot accidentally bypass it.
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		fetcher.client = client
		return fetcher
	}
	dialer := &net.Dialer{}
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           fetcher.dialContext(dialer),
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		IdleConnTimeout:       30 * time.Second,
	}
	fetcher.client = &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return fetcher
}

// Fetch returns a page only after its destination is public and allowed by
// robots.txt. Redirects are followed manually so each destination gets the
// same DNS-rebinding protection as the entry URL.
func (fetcher *HTTPFetcher) Fetch(ctx context.Context, target *url.URL, timeout time.Duration, validateRedirect RedirectValidator) (Page, error) {
	return fetcher.fetch(ctx, target, timeout, CacheValidators{}, validateRedirect)
}

// Revalidate issues a conditional request while preserving all destination,
// robots, redirect and body-size controls used for a normal crawl.
func (fetcher *HTTPFetcher) Revalidate(ctx context.Context, target *url.URL, timeout time.Duration, validators CacheValidators, validateRedirect RedirectValidator) (Page, error) {
	return fetcher.fetch(ctx, target, timeout, validators, validateRedirect)
}

func (fetcher *HTTPFetcher) fetch(ctx context.Context, target *url.URL, timeout time.Duration, validators CacheValidators, validateRedirect RedirectValidator) (Page, error) {
	if fetcher == nil || fetcher.client == nil || target == nil {
		return Page{}, fmt.Errorf("HTTP fetcher is not configured")
	}
	if timeout <= 0 {
		return Page{}, fmt.Errorf("request timeout must be positive")
	}
	current := cloneURL(target)
	for redirects := 0; redirects <= maxRedirects; redirects++ {
		if err := fetcher.validateDestination(ctx, current); err != nil {
			return Page{}, err
		}
		allowed, err := fetcher.allowedByRobots(ctx, current, timeout)
		if err != nil {
			return Page{}, err
		}
		if !allowed {
			return Page{RequestedURL: cloneURL(target), FinalURL: current, RobotsAllowed: false, FetchedAt: fetcher.now().UTC()}, nil
		}

		requestCtx, cancel := context.WithTimeout(ctx, timeout)
		request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, current.String(), nil)
		if err != nil {
			cancel()
			return Page{}, fmt.Errorf("build fetch request: %w", err)
		}
		request.Header.Set("User-Agent", userAgent)
		request.Header.Set("Accept", "text/html, text/plain;q=0.9, application/xhtml+xml;q=0.8")
		if validators.ETag != "" {
			request.Header.Set("If-None-Match", validators.ETag)
		}
		if validators.LastModified != "" {
			request.Header.Set("If-Modified-Since", validators.LastModified)
		}
		response, err := fetcher.client.Do(request)
		if err != nil {
			cancel()
			return Page{}, fmt.Errorf("fetch %s: %w", current, err)
		}

		if response.StatusCode == http.StatusNotModified {
			_ = response.Body.Close()
			cancel()
			return Page{RequestedURL: cloneURL(target), FinalURL: cloneURL(current), StatusCode: response.StatusCode, ContentType: response.Header.Get("Content-Type"), RobotsAllowed: true, RetryAfter: retryAfter(response.Header.Get("Retry-After"), fetcher.now()), FetchedAt: fetcher.now().UTC(), ETag: response.Header.Get("ETag"), LastModified: response.Header.Get("Last-Modified")}, nil
		}
		if isRedirect(response.StatusCode) {
			location := response.Header.Get("Location")
			_ = response.Body.Close()
			cancel()
			if location == "" {
				return Page{}, fmt.Errorf("redirect from %s has no Location header", current)
			}
			next, err := current.Parse(location)
			if err != nil {
				return Page{}, fmt.Errorf("parse redirect location: %w", err)
			}
			if validateRedirect != nil {
				next, err = validateRedirect(next)
				if err != nil {
					return Page{}, err
				}
			}
			current = next
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 8<<20+1))
		_ = response.Body.Close()
		cancel()
		if readErr != nil {
			return Page{}, fmt.Errorf("read %s: %w", current, readErr)
		}
		if len(body) > 8<<20 {
			return Page{}, fmt.Errorf("response body exceeds transport safety limit")
		}
		title, headings, links := extractDocumentFacts(body, response.Header.Get("Content-Type"))
		return Page{
			RequestedURL: cloneURL(target), FinalURL: cloneURL(current), StatusCode: response.StatusCode,
			ContentType: response.Header.Get("Content-Type"), Body: body, Title: title, HeadingTree: headings,
			Links: links, RobotsAllowed: true, RetryAfter: retryAfter(response.Header.Get("Retry-After"), fetcher.now()), FetchedAt: fetcher.now().UTC(), ETag: response.Header.Get("ETag"), LastModified: response.Header.Get("Last-Modified"),
		}, nil
	}
	return Page{}, fmt.Errorf("too many redirects")
}

func (fetcher *HTTPFetcher) dialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := fetcher.resolvePublicIPs(ctx, host)
		if err != nil {
			return nil, err
		}
		var lastErr error
		for _, ip := range ips {
			connection, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return connection, nil
			}
			lastErr = err
		}
		return nil, fmt.Errorf("dial resolved public address: %w", lastErr)
	}
}

func (fetcher *HTTPFetcher) validateDestination(ctx context.Context, target *url.URL) error {
	if target.Scheme != "http" && target.Scheme != "https" {
		return fmt.Errorf("only HTTP(S) destinations are supported")
	}
	if target.User != nil || strings.TrimSpace(target.Hostname()) == "" {
		return fmt.Errorf("destination must not include credentials and must include a host")
	}
	_, err := fetcher.resolvePublicIPs(ctx, target.Hostname())
	return err
}

func (fetcher *HTTPFetcher) resolvePublicIPs(ctx context.Context, host string) ([]net.IP, error) {
	if parsed := net.ParseIP(host); parsed != nil {
		if !isPublicIP(parsed) {
			return nil, fmt.Errorf("destination resolves to a private or special-use IP")
		}
		return []net.IP{parsed}, nil
	}
	addresses, err := fetcher.resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve destination host: %w", err)
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("destination host has no IP addresses")
	}
	ips := make([]net.IP, 0, len(addresses))
	for _, address := range addresses {
		if !isPublicIP(address.IP) {
			return nil, fmt.Errorf("destination resolves to a private or special-use IP")
		}
		ips = append(ips, address.IP)
	}
	return ips, nil
}

func (fetcher *HTTPFetcher) allowedByRobots(ctx context.Context, target *url.URL, timeout time.Duration) (bool, error) {
	key := target.Scheme + "://" + target.Host
	fetcher.robotsMu.Lock()
	rules, ok := fetcher.robots[key]
	fetcher.robotsMu.Unlock()
	if !ok {
		var err error
		rules, err = fetcher.fetchRobots(ctx, target, timeout)
		if err != nil {
			return false, err
		}
		fetcher.robotsMu.Lock()
		fetcher.robots[key] = rules
		fetcher.robotsMu.Unlock()
	}
	return rules.allows(target.EscapedPath()), nil
}

func (fetcher *HTTPFetcher) fetchRobots(ctx context.Context, target *url.URL, timeout time.Duration) (robotsRules, error) {
	robotsURL := &url.URL{Scheme: target.Scheme, Host: target.Host, Path: "/robots.txt"}
	if err := fetcher.validateDestination(ctx, robotsURL); err != nil {
		return robotsRules{}, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, robotsURL.String(), nil)
	if err != nil {
		return robotsRules{}, err
	}
	request.Header.Set("User-Agent", userAgent)
	response, err := fetcher.client.Do(request)
	if err != nil {
		return robotsRules{}, fmt.Errorf("fetch robots.txt: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone {
		return robotsRules{}, nil
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return robotsRules{}, fmt.Errorf("robots.txt returned HTTP %d", response.StatusCode)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, 1<<20+1))
	if err != nil {
		return robotsRules{}, err
	}
	if len(content) > 1<<20 {
		return robotsRules{}, fmt.Errorf("robots.txt exceeds safety limit")
	}
	return parseRobots(content), nil
}

type robotsRule struct {
	path  string
	allow bool
}

type robotsRules struct{ rules []robotsRule }

func parseRobots(content []byte) robotsRules {
	type group struct {
		agents []string
		rules  []robotsRule
	}
	groups := []group{}
	current := group{}
	flush := func() {
		if len(current.agents) > 0 {
			groups = append(groups, current)
		}
		current = group{}
	}
	for _, rawLine := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(strings.SplitN(rawLine, "#", 2)[0])
		if line == "" {
			flush()
			continue
		}
		pair := strings.SplitN(line, ":", 2)
		if len(pair) != 2 {
			continue
		}
		key, value := strings.ToLower(strings.TrimSpace(pair[0])), strings.TrimSpace(pair[1])
		switch key {
		case "user-agent":
			if len(current.rules) > 0 {
				flush()
			}
			current.agents = append(current.agents, strings.ToLower(value))
		case "allow", "disallow":
			if len(current.agents) > 0 && value != "" {
				current.rules = append(current.rules, robotsRule{path: value, allow: key == "allow"})
			}
		}
	}
	flush()
	for _, preferred := range []string{strings.ToLower(userAgent), "inkwordsdoccrawler", "*"} {
		for _, candidate := range groups {
			for _, agent := range candidate.agents {
				if agent == preferred {
					return robotsRules{rules: candidate.rules}
				}
			}
		}
	}
	return robotsRules{}
}

func (rules robotsRules) allows(escapedPath string) bool {
	if escapedPath == "" {
		escapedPath = "/"
	}
	bestLength, allowed := -1, true
	for _, rule := range rules.rules {
		if strings.HasPrefix(escapedPath, rule.path) && (len(rule.path) > bestLength || (len(rule.path) == bestLength && rule.allow)) {
			bestLength, allowed = len(rule.path), rule.allow
		}
	}
	return allowed
}

func extractDocumentFacts(content []byte, contentType string) (string, []string, []string) {
	if !strings.Contains(strings.ToLower(contentType), "html") {
		return "", nil, nil
	}
	tokenizer := html.NewTokenizer(bytes.NewReader(content))
	var title string
	headings, links := make([]string, 0), make([]string, 0)
	var capture string
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}
		switch tokenType {
		case html.StartTagToken:
			tag, hasAttributes := tokenizer.TagName()
			switch string(tag) {
			case "title", "h1", "h2", "h3", "h4", "h5", "h6":
				capture = string(tag)
			case "a":
				for hasAttributes {
					key, value, more := tokenizer.TagAttr()
					if strings.EqualFold(string(key), "href") && strings.TrimSpace(string(value)) != "" {
						links = append(links, strings.TrimSpace(string(value)))
					}
					hasAttributes = more
				}
			}
		case html.TextToken:
			text := strings.TrimSpace(string(tokenizer.Text()))
			if text == "" || capture == "" {
				continue
			}
			if capture == "title" {
				title += text
			} else {
				headings = append(headings, text)
			}
		case html.EndTagToken:
			tag, _ := tokenizer.TagName()
			if string(tag) == capture {
				capture = ""
			}
		}
	}
	return strings.TrimSpace(title), headings, links
}

func retryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil && when.After(now) {
		return when.Sub(now)
	}
	return 0
}

func isRedirect(status int) bool { return status >= 300 && status < 400 }

func cloneURL(value *url.URL) *url.URL {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func isPublicIP(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsMulticast() && !ip.IsUnspecified() && !ip.IsInterfaceLocalMulticast() && !ip.IsLinkLocalMulticast()
}
