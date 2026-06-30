package obsidian

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

type HTTPError struct {
	StatusCode int
	Body       []byte
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("obsidian http error: status=%d", e.StatusCode)
}

type RestAPIStore struct {
	baseURL *url.URL
	apiKey  string
	client  *http.Client
}

func NewStoreFromEnv() (Store, error) {
	return NewRestAPIStoreFromEnv()
}

func NewRestAPIStoreFromEnv() (*RestAPIStore, error) {
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
		certPEM, err := os.ReadFile(certPath) //nolint:gosec
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
			InsecureSkipVerify: insecureSkipVerify, //nolint:gosec
		},
	}

	return NewRestAPIStore(u, apiKey, &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}), nil
}

func NewRestAPIStore(baseURL *url.URL, apiKey string, client *http.Client) *RestAPIStore {
	return &RestAPIStore{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  client,
	}
}

func (s *RestAPIStore) vaultURL(p string) string {
	u := *s.baseURL
	trailingSlash := strings.HasSuffix(p, "/")
	p = strings.TrimSuffix(p, "/")
	u.Path = path.Join(s.baseURL.Path, "vault", p)
	if trailingSlash {
		u.Path += "/"
	}
	return u.String()
}

//nolint:unparam
func (s *RestAPIStore) do(ctx context.Context, method string, p string, headers map[string]string, contentType string, body []byte) ([]byte, int, error) {
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
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return respBody, resp.StatusCode, HTTPError{StatusCode: resp.StatusCode, Body: respBody}
	}

	return respBody, resp.StatusCode, nil
}

func (s *RestAPIStore) Read(ctx context.Context, p string) ([]byte, error) {
	resp, _, err := s.do(ctx, http.MethodGet, p, nil, "", nil)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *RestAPIStore) Put(ctx context.Context, p string, contentType string, body []byte) error {
	_, _, err := s.do(ctx, http.MethodPut, p, nil, contentType, body)
	return err
}

func (s *RestAPIStore) Post(ctx context.Context, p string, contentType string, body []byte) error {
	_, _, err := s.do(ctx, http.MethodPost, p, nil, contentType, body)
	return err
}

func (s *RestAPIStore) Patch(ctx context.Context, p string, headers map[string]string, contentType string, body []byte) error {
	_, _, err := s.do(ctx, http.MethodPatch, p, headers, contentType, body)
	return err
}

func (s *RestAPIStore) List(ctx context.Context, dirPath string) ([]string, error) {
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
		out := make([]string, 0, len(objectList))
		for _, item := range objectList {
			if value, ok := item["path"].(string); ok {
				out = append(out, value)
				continue
			}
			if value, ok := item["name"].(string); ok {
				out = append(out, value)
			}
		}
		return out, nil
	}

	var wrapped map[string]json.RawMessage
	if json.Unmarshal(resp, &wrapped) == nil {
		if raw, ok := wrapped["files"]; ok {
			var files []string
			if json.Unmarshal(raw, &files) == nil {
				return files, nil
			}
			var anyFiles []any
			if json.Unmarshal(raw, &anyFiles) == nil {
				out := make([]string, 0, len(anyFiles))
				for _, value := range anyFiles {
					if file, ok := value.(string); ok {
						out = append(out, file)
					}
				}
				return out, nil
			}
		}
	}

	return nil, errors.New("无法解析 Obsidian 目录列表响应")
}

func IsNotFound(err error) bool {
	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == http.StatusNotFound
	}
	return false
}
