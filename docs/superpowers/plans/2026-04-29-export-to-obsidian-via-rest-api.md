# Export to Obsidian via Local REST API Implementation Plan
  
> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
  
**Goal:** 在不向前端暴露 Obsidian 能力的前提下，InkWords 后端在 Docker 内通过 sidecar 转发同步调用 `obsidian-local-rest-api`，完成 Karpathy LLM Wiki Pattern 的导出写入与索引更新。
  
**Architecture:** 在 `docker-compose.yml` 增加 `obsidian-bridge` sidecar（socat TCP 透传 27125→宿主机 27124）。后端引入 `ObsidianStore` 抽象与 `RestAPIStore` 实现，把现有基于文件系统的 wiki 写入逻辑迁移为调用 REST API 的读写/列目录/追加与精准 PATCH。
  
**Tech Stack:** Go 1.21+, Gin, GORM, Docker Compose, Obsidian Local REST API (HTTPS + API Key), socat
  
---
  
## File Map（将被修改/新增的文件）
  
- Modify: `docker-compose.yml`
- Modify: `backend/.env`（新增配置项，值由用户本地注入，不提交密钥）
- Create: `backend/internal/service/obsidian_store.go`（接口定义）
- Create: `backend/internal/service/obsidian_rest_store.go`（REST API 实现）
- Modify: `backend/internal/service/obsidian_wiki.go`（scaffold 与 index 生成改为使用 ObsidianStore）
- Modify: `backend/internal/service/obsidian_export.go`（导出逻辑改为使用 ObsidianStore）
- Create: `backend/internal/service/obsidian_rest_store_test.go`（单元测试：请求构造/证书加载/错误处理）
- Modify: `backend/internal/service/obsidian_wiki_test.go`（改为使用 fake store 测试 scaffold/index 逻辑）
  
---
  
### Task 1: Docker Compose sidecar（obsidian-bridge）
  
**Files:**
- Modify: `docker-compose.yml`
  
- [ ] **Step 1: 在 compose 中新增 obsidian-bridge 服务（仅内网，不映射宿主机端口）**
  
将以下片段加入 `services:`（与 `backend` 同网络即可）：
  
```yaml
  obsidian-bridge:
    image: alpine/socat:latest
    container_name: inkwords-obsidian-bridge
    command: ["-d", "-d", "TCP-LISTEN:27125,fork,reuseaddr", "TCP:host.docker.internal:27124"]
    extra_hosts:
      - "host.docker.internal:host-gateway"
    restart: unless-stopped
```
  
- [ ] **Step 2: 为 backend 增加对 obsidian-bridge 的依赖（确保启动顺序）**
  
在 `backend.depends_on:` 增加：
  
```yaml
      obsidian-bridge:
        condition: service_started
```
  
- [ ] **Step 3: 写入并约定后端访问的 base URL**
  
在 `backend.environment:`（或 `backend/.env`）加入：
  
```yaml
      - OBSIDIAN_REST_API_BASE_URL=https://obsidian-bridge:27125
```
  
说明：这是内部调用，不对前端暴露。
  
- [ ] **Step 4: 准备证书只读挂载（后端容器）**
  
在 `backend.volumes:` 增加（路径可按你本机实际存放位置调整）：
  
```yaml
      - ${OBSIDIAN_REST_API_CERT_PATH:-/etc/hosts}:/app/obsidian-cert/obsidian-local-rest-api-certificate.crt:ro
```
  
并约定后端 env：
  
```yaml
      - OBSIDIAN_REST_API_CERT_PATH=/app/obsidian-cert/obsidian-local-rest-api-certificate.crt
      - OBSIDIAN_REST_API_INSECURE_SKIP_VERIFY=${OBSIDIAN_REST_API_INSECURE_SKIP_VERIFY:-false}
```
  
- [ ] **Step 5: 一键重启验证容器可用（只验证 bridge 进程能起来）**
  
Run:
  
```bash
docker compose down && docker compose up -d --build
docker ps --format "table {{.Names}}\t{{.Status}}"
```
  
Expected:
- `inkwords-obsidian-bridge` 状态为 Up
  
---
  
### Task 2: 定义 ObsidianStore 接口（最小能力集合）
  
**Files:**
- Create: `backend/internal/service/obsidian_store.go`
  
- [ ] **Step 1: 新增接口文件与类型**
  
```go
package service
  
import "context"
  
type ObsidianStore interface {
	Read(ctx context.Context, path string) ([]byte, error)
	Put(ctx context.Context, path string, contentType string, body []byte) error
	Post(ctx context.Context, path string, contentType string, body []byte) error
	Patch(ctx context.Context, path string, headers map[string]string, contentType string, body []byte) error
	List(ctx context.Context, dirPath string) ([]string, error)
}
```
  
- [ ] **Step 2: 运行后端编译检查**
  
Run:
  
```bash
docker compose exec backend go test ./... -run TestNonExistent 2>/dev/null || true
docker compose exec backend go test ./...
```
  
Expected:
- `go test ./...` 通过（若当前仓库已有测试不稳定，请改为 `go test ./...` 的实际结果为准并记录失败原因）
  
---
  
### Task 3: 实现 RestAPIStore（证书信任 + Bearer Key + 统一错误处理）
  
**Files:**
- Create: `backend/internal/service/obsidian_rest_store.go`
- Create: `backend/internal/service/obsidian_rest_store_test.go`
  
- [ ] **Step 1: 先写一个单元测试：证书加载失败时返回明确错误**
  
```go
package service
  
import (
	"context"
	"testing"
)
  
func TestNewRestAPIStore_CertMissing(t *testing.T) {
	t.Setenv("OBSIDIAN_REST_API_BASE_URL", "https://obsidian-bridge:27125")
	t.Setenv("OBSIDIAN_REST_API_KEY", "dummy")
	t.Setenv("OBSIDIAN_REST_API_CERT_PATH", "/path/not-exist.crt")
  
	_, err := NewRestAPIStoreFromEnv()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	_ = context.Background()
}
```
  
- [ ] **Step 2: 实现 NewRestAPIStoreFromEnv 与 RestAPIStore 骨架（让测试先过）**
  
实现要点（代码需落在 `obsidian_rest_store.go`）：
  
```go
package service
  
import (
	"context"
	"crypto/tls"
	"crypto/x509"
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
  
type RestAPIStore struct {
	baseURL *url.URL
	apiKey  string
	client  *http.Client
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
	if certPath == "" {
		return nil, errors.New("OBSIDIAN_REST_API_CERT_PATH 未配置")
	}
  
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("解析 OBSIDIAN_REST_API_BASE_URL 失败: %w", err)
	}
  
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("读取 Obsidian 证书失败: %w", err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(certPEM) {
		return nil, errors.New("解析 Obsidian 证书失败")
	}
  
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:    caPool,
			ServerName: "127.0.0.1",
			MinVersion: tls.VersionTLS12,
		},
	}
  
	return &RestAPIStore{
		baseURL: u,
		apiKey:  apiKey,
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}, nil
}
  
func (s *RestAPIStore) vaultURL(p string) string {
	joined := path.Join("vault", p)
	u := *s.baseURL
	u.Path = path.Join(s.baseURL.Path, joined)
	return u.String()
}
  
func (s *RestAPIStore) do(ctx context.Context, method string, p string, headers map[string]string, contentType string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, method, s.vaultURL(p), io.NopCloser(strings.NewReader(string(body))))
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
		return respBody, resp.StatusCode, fmt.Errorf("Obsidian REST API 调用失败: status=%d", resp.StatusCode)
	}
	return respBody, resp.StatusCode, nil
}
```
  
注意：实现中不要把 apiKey 打到 error/log。
  
- [ ] **Step 3: 补齐 RestAPIStore 的接口方法（Read/Put/Post/Patch/List）**
  
实现策略：
- `Read`：GET `/vault/{path}`
- `Put`：PUT `/vault/{path}`，Content-Type 传 `text/markdown` 或 `text/plain`
- `Post`：POST `/vault/{path}`（用于 append；若 Obsidian 端要求 target 语义，则用 `Patch`）
- `Patch`：PATCH `/vault/{path}`，headers 透传 `Operation`、`Target-Type`、`Target`
- `List`：GET `/vault/{dirPath}/`（注意结尾 `/`），返回 JSON 列表并解析为文件名数组
  
- [ ] **Step 4: 扩展测试：vaultURL 拼接与鉴权头存在**
  
建议用 `httptest.NewTLSServer` + 自签证书（从 server 拿 CA 加到池中）做白盒验证：
- 请求 path 以 `/vault/...` 开头
- Header 有 `Authorization: Bearer dummy`
  
- [ ] **Step 5: 运行单测**
  
Run:
  
```bash
docker compose exec backend go test ./internal/service -run TestNewRestAPIStore_CertMissing -v
docker compose exec backend go test ./internal/service -v
```
  
Expected:
- 测试通过
  
---
  
### Task 4: 改造 wiki scaffold 与 folder index 生成：从文件系统迁移到 Store
  
**Files:**
- Modify: `backend/internal/service/obsidian_wiki.go`
- Modify: `backend/internal/service/obsidian_wiki_test.go`
  
- [ ] **Step 1: 写一个 fake store（内存 map）用于测试**
  
在 `obsidian_wiki_test.go` 内新增：
  
```go
type fakeStore struct {
	files map[string][]byte
	dirs  map[string][]string
}
```
  
并实现最小方法集合，用于验证 scaffold 写入了预期的 `_index.md` 内容。
  
- [ ] **Step 2: 将 ensureWikiScaffold / writeFolderIndex / writeDomainsIndex 改为依赖 ObsidianStore**
  
改造原则：
- 不再 `os.MkdirAll`、`os.ReadDir`、`os.WriteFile`
- 目录存在性由 REST API 的语义负责（PUT 写文件时路径应存在；若 REST API 不自动创建目录，则在实现层补齐“创建目录”策略：优先通过写入占位文件或使用 API 的 list/put 行为验证）
  
最小落地：先把 `writeFolderIndex` 做成“基于 `store.List` 获取文件名列表”。
  
- [ ] **Step 3: 用 fake store 跑单测确保 `_index.md` 生成内容正确**
  
断言点：
- `_index.md` 包含 `---` frontmatter
- 列表行形如 `- [[concepts/Name|Name]]`
  
- [ ] **Step 4: 运行 service 测试**
  
Run:
  
```bash
docker compose exec backend go test ./internal/service -run Test -v
```
  
Expected:
- 通过
  
---
  
### Task 5: 改造导出逻辑：从写文件迁移到 RestAPIStore
  
**Files:**
- Modify: `backend/internal/service/obsidian_export.go`
  
- [ ] **Step 1: 为 BlogService 增加一个可注入的 store（DI）**
  
方案：
- 在 `BlogService` struct 中增加 `obsidianStore ObsidianStore`
- 在现有构造处注入（从 env 初始化 `RestAPIStore`；如果初始化失败，导出接口返回明确中文错误）
  
- [ ] **Step 2: 将 ExportToObsidian 改为使用 store.Put 写入**
  
关键替换：
- 原 `os.WriteFile(filePath, ...)` → `store.Put(ctx, "wiki/concepts/<title>.md", "text/markdown", contentBytes)`
  
注意：本项目现有挂载路径是 `/app/obsidian`，而 REST API 的 vault 路径要以你实际 vault 根为准。这里建议在 REST 层统一约定 “写入 vault 的 `wiki/` 目录”，即所有 path 都以 `wiki/...` 开头。
  
- [ ] **Step 3: 将 ExportSeriesToObsidian 改为 store.Put / store.Read / store.Post 或 store.Patch**
  
建议策略（最小改动）：
- `index.md`：先 `Read`，再在内存里按既有规则拼接后 `Put` 覆盖（避免依赖 append 行为差异）
- `log.md`：同上（读-改-写），保证插入位置规则一致
- `hot.md`：直接 `Put` 覆盖
  
- [ ] **Step 4: 运行后端测试**
  
Run:
  
```bash
docker compose exec backend go test ./... -v
```
  
Expected:
- 通过
  
---
  
### Task 6: E2E 手工验证（真实 Obsidian）
  
**Files:**
- No code files
  
- [ ] **Step 1: 确认 Obsidian 插件在宿主机运行**
  
宿主机执行：
  
```bash
curl -k https://127.0.0.1:27124/ | head -n 5
```
  
Expected:
- 返回 JSON（不要求有鉴权）
  
- [ ] **Step 2: 通过后端接口触发导出（前端按钮或 curl）**
  
如果使用前端按钮：登录后点击「导出到 Obsidian」。
  
如果用 curl（示例，按你实际鉴权方式填 token）：
  
```bash
curl -sS -X POST \
  -H "Authorization: Bearer <INKWORDS_JWT>" \
  http://localhost/api/v1/blogs/<BLOG_ID>/export/obsidian
```
  
Expected:
- 返回成功 JSON
  
- [ ] **Step 3: 在 Obsidian 中检查写入结果**
  
检查：
- `wiki/concepts/` 新增或更新对应 md
- `wiki/index.md`、`wiki/hot.md`、`wiki/log.md` 更新
  
---
  
## Plan Self-Review（已执行）
  
- Spec coverage：覆盖 sidecar、TLS pin、store 抽象、导出同步、Karpathy wiki 文件更新。
- Placeholder scan：无 TBD/TODO；命令与文件路径明确。
- Consistency：统一约定 REST 写入路径为 `wiki/...`；compose 端口固定 27125。
  
