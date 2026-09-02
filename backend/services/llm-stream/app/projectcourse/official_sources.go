package projectcourse

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type OfficialSource struct {
	Technology        string    `json:"technology"`
	VersionConstraint string    `json:"version_constraint"`
	URL               string    `json:"url"`
	RetrievedAt       time.Time `json:"retrieved_at"`
	ContentHash       string    `json:"content_hash"`
	Content           string    `json:"content,omitempty"`
}

type OfficialSourceProvider interface {
	Fetch(url string) (content string, err error)
}

type HTTPOfficialSourceProvider struct {
	Client         *http.Client
	AllowedDomains []string
	MaxBytes       int64
	ResolveHost    func(host string) ([]net.IP, error)
}

func (p HTTPOfficialSourceProvider) Fetch(rawURL string) (string, error) {
	if err := p.validateFetchURL(rawURL); err != nil {
		return "", err
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	clientCopy := *client
	previousRedirect := clientCopy.CheckRedirect
	clientCopy.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many official source redirects")
		}
		if err := p.validateFetchURL(req.URL.String()); err != nil {
			return fmt.Errorf("redirect rejected: %w", err)
		}
		if previousRedirect != nil {
			return previousRedirect(req, via)
		}
		return nil
	}
	maxBytes := p.MaxBytes
	if maxBytes <= 0 {
		maxBytes = 1 << 20
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("create official source request: %w", err)
	}
	response, err := clientCopy.Do(request)
	if err != nil {
		return "", fmt.Errorf("fetch official source: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("official source returned status %d", response.StatusCode)
	}
	contentType := strings.ToLower(strings.Split(response.Header.Get("Content-Type"), ";")[0])
	if contentType != "" && contentType != "text/html" && contentType != "text/plain" && contentType != "application/json" {
		return "", fmt.Errorf("unsupported official source content type %q", contentType)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return "", fmt.Errorf("read official source: %w", err)
	}
	if int64(len(content)) > maxBytes {
		return "", fmt.Errorf("official source exceeds size limit")
	}
	if len(content) == 0 {
		return "", fmt.Errorf("official source content is empty")
	}
	return string(content), nil
}

func (p HTTPOfficialSourceProvider) validateFetchURL(rawURL string) error {
	if err := ValidateOfficialURL(rawURL, p.AllowedDomains); err != nil {
		return err
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse official source URL: %w", err)
	}
	resolver := p.ResolveHost
	if resolver == nil {
		resolver = net.LookupIP
	}
	ips, err := resolver(parsed.Hostname())
	if err != nil {
		return fmt.Errorf("resolve official source host: %w", err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("official source host has no resolved address")
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			return fmt.Errorf("official source host resolves to a private address")
		}
	}
	return nil
}

type OfficialRegistry struct {
	Domains map[string][]string
	URLs    map[string]string
}

func NewDefaultOfficialRegistry() OfficialRegistry {
	return OfficialRegistry{
		Domains: map[string][]string{
			"Go":             {"go.dev", "pkg.go.dev"},
			"Gin":            {"gin-gonic.com"},
			"React":          {"react.dev"},
			"Zustand":        {"zustand.docs.pmnd.rs"},
			"PostgreSQL":     {"postgresql.org"},
			"RabbitMQ":       {"rabbitmq.com"},
			"Redis":          {"redis.io"},
			"Docker Compose": {"docs.docker.com"},
			"Nginx":          {"nginx.org"},
			"TypeScript":     {"typescriptlang.org"},
		},
		URLs: map[string]string{
			"Go":             "https://go.dev/doc/",
			"Gin":            "https://gin-gonic.com/docs/",
			"React":          "https://react.dev/learn",
			"Zustand":        "https://zustand.docs.pmnd.rs/",
			"PostgreSQL":     "https://www.postgresql.org/docs/",
			"RabbitMQ":       "https://www.rabbitmq.com/docs",
			"Redis":          "https://redis.io/docs/latest/",
			"Docker Compose": "https://docs.docker.com/compose/",
			"Nginx":          "https://nginx.org/en/docs/",
			"TypeScript":     "https://www.typescriptlang.org/docs/",
		},
	}
}

func (r OfficialRegistry) Resolve(technology, versionConstraint string) (OfficialSource, error) {
	technology = strings.TrimSpace(technology)
	rawURL, ok := r.URLs[technology]
	if !ok {
		return OfficialSource{}, fmt.Errorf("no official source registered for %q", technology)
	}
	if err := ValidateOfficialURL(rawURL, r.Domains[technology]); err != nil {
		return OfficialSource{}, err
	}
	return OfficialSource{Technology: technology, VersionConstraint: strings.TrimSpace(versionConstraint), URL: rawURL}, nil
}

func ValidateOfficialURL(rawURL string, allowedDomains []string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil {
		return fmt.Errorf("official URL must be HTTPS without credentials")
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host == "" || isPrivateHost(host) {
		return fmt.Errorf("official URL points to a private or invalid host")
	}
	if port := parsed.Port(); port != "" && port != "443" {
		return fmt.Errorf("official URL must use the default HTTPS port")
	}
	for _, allowed := range allowedDomains {
		allowed = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(allowed), "."))
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return nil
		}
	}
	return fmt.Errorf("official URL domain %q is not allowlisted", host)
}

func FinalizeOfficialSource(source OfficialSource, content string, retrievedAt time.Time) (OfficialSource, error) {
	if err := ValidateOfficialURL(source.URL, []string{mustHostname(source.URL)}); err != nil {
		return OfficialSource{}, err
	}
	if strings.TrimSpace(content) == "" {
		return OfficialSource{}, fmt.Errorf("official source content is empty")
	}
	sum := sha256.Sum256([]byte(content))
	source.Content = content
	source.ContentHash = "sha256:" + hex.EncodeToString(sum[:])
	source.RetrievedAt = retrievedAt.UTC()
	return source, nil
}

func mustHostname(rawURL string) string { parsed, _ := url.Parse(rawURL); return parsed.Hostname() }

func isPrivateHost(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return host == "localhost" || strings.HasSuffix(host, ".local")
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}
