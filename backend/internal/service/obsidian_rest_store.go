package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

type obsidianHTTPError struct {
	StatusCode int
	Body       []byte
}

func (e obsidianHTTPError) Error() string {
	return fmt.Sprintf("obsidian http error: status=%d", e.StatusCode)
}

type restAPIStore struct {
	baseURL *url.URL
	apiKey  string
	client  *http.Client
}

func newRestAPIStoreFromEnv() (*restAPIStore, error) {
	base := strings.TrimSpace(os.Getenv("OBSIDIAN_REST_API_BASE_URL"))
	if base == "" {
		return nil, errors.New("OBSIDIAN_REST_API_BASE_URL 未配置")
	}
	apiKey := strings.TrimSpace(os.Getenv("OBSIDIAN_REST_API_KEY"))
	if apiKey == "" {
		return nil, errors.New("OBSIDIAN_REST_API_KEY 未配置")
	}
	certPath := strings.TrimSpace(os.Getenv("OBSIDIAN_REST_API_CERT_PATH"))
	insecureSkipVerify := strings.EqualFold(strings.TrimSpace(os.Getenv("OBSIDIAN_REST_API_INSECURE_SKIP_VERIFY")), "true")
	if certPath == "" && !insecureSkipVerify {
		return nil, errors.New("OBSIDIAN_REST_API_CERT_PATH 未配置（或设置 OBSIDIAN_REST_API_INSECURE_SKIP_VERIFY=true 仅限本地开发）")
	}

	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("解析 OBSIDIAN_REST_API_BASE_URL 失败: %w", err)
	}

	caPool := x509.NewCertPool()
	if certPath != "" {
		certPEM, err := os.ReadFile(certPath)
		if err != nil {
			return nil, fmt.Errorf("读取 Obsidian 证书失败: %w", err)
		}
		if !caPool.AppendCertsFromPEM(certPEM) {
			return nil, errors.New("解析 Obsidian 证书失败")
		}
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            caPool,
			ServerName:         "127.0.0.1",
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: insecureSkipVerify,
		},
	}

	return newRestAPIStore(u, apiKey, &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}), nil
}

func newRestAPIStore(baseURL *url.URL, apiKey string, client *http.Client) *restAPIStore {
	return &restAPIStore{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  client,
	}
}

func (s *restAPIStore) vaultURL(p string) string {
	u := *s.baseURL
	trailingSlash := strings.HasSuffix(p, "/")
	p = strings.TrimSuffix(p, "/")
	u.Path = path.Join(s.baseURL.Path, "vault", p)
	if trailingSlash {
		u.Path += "/"
	}
	return u.String()
}

func (s *restAPIStore) do(ctx context.Context, method string, p string, headers map[string]string, contentType string, body []byte) ([]byte, int, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.vaultURL(p), reader)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return respBody, resp.StatusCode, obsidianHTTPError{StatusCode: resp.StatusCode, Body: respBody}
	}

	return respBody, resp.StatusCode, nil
}

func (s *restAPIStore) Read(ctx context.Context, p string) ([]byte, error) {
	resp, _, err := s.do(ctx, http.MethodGet, p, nil, "", nil)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *restAPIStore) Put(ctx context.Context, p string, contentType string, body []byte) error {
	_, _, err := s.do(ctx, http.MethodPut, p, nil, contentType, body)
	return err
}

func (s *restAPIStore) Post(ctx context.Context, p string, contentType string, body []byte) error {
	_, _, err := s.do(ctx, http.MethodPost, p, nil, contentType, body)
	return err
}

func (s *restAPIStore) Patch(ctx context.Context, p string, headers map[string]string, contentType string, body []byte) error {
	_, _, err := s.do(ctx, http.MethodPatch, p, headers, contentType, body)
	return err
}

func (s *restAPIStore) List(ctx context.Context, dirPath string) ([]string, error) {
	dir := dirPath
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}

	resp, _, err := s.do(ctx, http.MethodGet, dir, nil, "", nil)
	if err != nil {
		return nil, err
	}

	var stringList []string
	if json.Unmarshal(resp, &stringList) == nil {
		return stringList, nil
	}

	var objectList []map[string]any
	if json.Unmarshal(resp, &objectList) == nil {
		var out []string
		for _, item := range objectList {
			if v, ok := item["path"]; ok {
				if s, ok := v.(string); ok {
					out = append(out, s)
				}
				continue
			}
			if v, ok := item["name"]; ok {
				if s, ok := v.(string); ok {
					out = append(out, s)
				}
			}
		}
		return out, nil
	}

	return nil, errors.New("无法解析 Obsidian 目录列表响应")
}

func isObsidianNotFound(err error) bool {
	var httpErr obsidianHTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == http.StatusNotFound
	}
	return false
}
