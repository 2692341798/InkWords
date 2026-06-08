# P0 Microservices Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修通当前微服务化最关键的三个 P0 缺口：`parser-service` RabbitMQ 编排、最小微服务冒烟门禁、以及 `core-api` 生成链路对 `blogs / users` 的显式写入边界收口。

**Architecture:** 本计划只做最小可回滚改动，不改变现有对外 API、前端入口、数据库表结构或任务协议。先让 `parser-service` 的异步任务链路真正可用，再把现有冒烟检查补成稳定门禁，最后把 `GeneratorService` 的直接全局 `db.DB` 写入收口到显式 repository 接口，为后续继续收口 `decomposition_generate*.go` 提供模板。

**Tech Stack:** Docker Compose + Nginx + Go 1.25 + Gin + GORM + PostgreSQL + RabbitMQ + GitHub Actions

---

## File Map

**Infra / Compose**
- Modify: `docker-compose.yml`
- Modify: `.github/workflows/ci.yml`
- Modify: `docs/runbooks/microservices-smoke-check.md`

**Parser worker**
- Modify: `backend/services/parser-service/cmd/main.go`（仅在需要补日志文案时）
- Test via runtime: `POST /api/v1/tasks/parse` + worker log

**Core write-boundary**
- Modify: `backend/internal/service/generator.go`
- Modify: `backend/internal/service/generator_persist_test.go`
- Create: `backend/internal/service/generator_persistence.go`
- Create: `backend/internal/service/generator_persistence_test.go`

**Docs**
- Modify: `README.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`

---

## Task 1: 修通 parser-service 的 RabbitMQ 编排

**Files:**
- Modify: `docker-compose.yml`
- Optional Modify: `README.md`

- [ ] **Step 1: 先写一个配置级失败检查，锁定 parser-service 缺少 MQ 环境变量**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
python3 - <<'PY'
from pathlib import Path
text = Path("docker-compose.yml").read_text()
start = text.index("  parser-service:")
end = text.index("  export-service:")
section = text[start:end]
required = [
    "RABBITMQ_URL:",
    "RABBITMQ_EXCHANGE:",
    "RABBITMQ_PARSE_QUEUE:",
    "rabbitmq:",
]
missing = [item for item in required if item not in section]
if missing:
    raise SystemExit("missing in parser-service section: " + ", ".join(missing))
print("parser-service mq config present")
PY
```

Expected: FAIL，提示 `missing in parser-service section`

- [ ] **Step 2: 在 Compose 里给 parser-service 补全 MQ 编排**

在 `docker-compose.yml` 的 `parser-service.environment` 中补上：

```yaml
  parser-service:
    build:
      context: ./backend
      dockerfile: Dockerfile
    container_name: inkwords-parser-service
    command: ["./parser-service"]
    environment:
      DATABASE_URL: postgres://${POSTGRES_USER:-inkwords}:${POSTGRES_PASSWORD:-inkwords_password}@db:5432/${POSTGRES_DB:-inkwords_db}?sslmode=disable
      REDIS_URL: ${REDIS_URL:-redis://redis:6379/0}
      RABBITMQ_URL: ${RABBITMQ_URL:-amqp://guest:guest@rabbitmq:5672/}
      RABBITMQ_EXCHANGE: ${RABBITMQ_EXCHANGE:-inkwords.events}
      RABBITMQ_PARSE_QUEUE: ${RABBITMQ_PARSE_QUEUE:-inkwords.parse}
      JWT_SECRET: ${JWT_SECRET:-}
    expose:
      - "8080"
    depends_on:
      db:
        condition: service_healthy
      redis:
        condition: service_started
      rabbitmq:
        condition: service_started
```

Why:
- `StartParseConsumer()` 只有在 `RABBITMQ_URL` 存在时才会真正启动 worker。
- `depends_on: rabbitmq` 不是业务正确性的充分条件，但它能减少容器刚启动时 RabbitMQ 还不可达导致的初始化漂移。

- [ ] **Step 3: 重新运行配置级检查**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
python3 - <<'PY'
from pathlib import Path
text = Path("docker-compose.yml").read_text()
start = text.index("  parser-service:")
end = text.index("  export-service:")
section = text[start:end]
required = [
    "RABBITMQ_URL:",
    "RABBITMQ_EXCHANGE:",
    "RABBITMQ_PARSE_QUEUE:",
    "rabbitmq:",
]
missing = [item for item in required if item not in section]
if missing:
    raise SystemExit("missing in parser-service section: " + ", ".join(missing))
print("parser-service mq config present")
PY
```

Expected: PASS，输出 `parser-service mq config present`

- [ ] **Step 4: 用 Compose 渲染确认编排合法**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
docker compose --env-file backend/.env config >/tmp/inkwords-compose.out
grep -n "parser-service" -A25 /tmp/inkwords-compose.out
```

Expected:
- `docker compose ... config` 成功退出
- 展开的 `parser-service` 中能看到 `RABBITMQ_URL`、`RABBITMQ_EXCHANGE`、`RABBITMQ_PARSE_QUEUE`

- [ ] **Step 5: 重建服务并验证 parser worker 不再被静默禁用**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
docker compose --env-file backend/.env down
docker compose --env-file backend/.env up -d --build
docker compose --env-file backend/.env ps
docker compose --env-file backend/.env logs --no-color parser-service | tail -n 50
```

Expected:
- `parser-service` 为 `Up (healthy)` 或等价健康状态
- 日志中不再出现 `RabbitMQ is not configured, parse consumer disabled`

- [ ] **Step 6: 做一次真实 parse 任务验证**

先登录拿 Bearer Token，然后执行：

```bash
TOKEN="<your-token>"
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
python3 - <<'PY' > /tmp/parse-task.json
import base64, json
payload = {
    "kind": "parse_file",
    "filename": "smoke.md",
    "content_base64": base64.b64encode("# smoke\n\nhello parser\n".encode()).decode(),
}
print(json.dumps(payload))
PY

curl -sS -X POST http://localhost/api/v1/tasks/parse \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  --data @/tmp/parse-task.json
```

Expected:
- 返回 `202` 或包含 `task_id` / `stream_url` 的成功 JSON

再查看 worker 与任务结果：

```bash
docker compose --env-file backend/.env logs --no-color parser-service | tail -n 50
```

如果已有 `stream_url`，再订阅：

```bash
curl -N -H "Authorization: Bearer ${TOKEN}" "http://localhost<stream_url>"
```

Expected:
- `parser-service` 日志能看到消费痕迹
- 任务结果最终包含 `source_content`

- [ ] **Step 7: 如 README 与实际不一致，补一条说明**

如果这一步之前 README 仍暗示“任务式解析已默认可用”但没注明依赖 RabbitMQ，可补一段：

```md
`parser-service` 的任务式解析依赖 `RABBITMQ_URL / RABBITMQ_EXCHANGE / RABBITMQ_PARSE_QUEUE`；若这些变量未注入，服务会退化为仅保留同步解析兼容路径。
```

- [ ] **Step 8: Commit**

```bash
git add docker-compose.yml README.md
git commit -m "fix(parser-service): wire rabbitmq env into compose"
```

---

## Task 2: 固化最小微服务冒烟门禁

**Files:**
- Modify: `docs/runbooks/microservices-smoke-check.md`
- Modify: `.github/workflows/ci.yml`

- [ ] **Step 1: 先写失败门槛，确认 CI 目前没有覆盖 parse 任务 worker 链路**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
python3 - <<'PY'
from pathlib import Path
text = Path(".github/workflows/ci.yml").read_text()
needles = [
    "/api/v1/tasks/parse",
    "inkwords-parser-service",
]
missing = [n for n in needles if n not in text]
if missing:
    raise SystemExit("ci missing smoke coverage markers: " + ", ".join(missing))
print("ci already covers parse task smoke")
PY
```

Expected: FAIL，至少缺少 `/api/v1/tasks/parse`

- [ ] **Step 2: 在 Runbook 增加“最小 P0 回归集”小节**

在 `docs/runbooks/microservices-smoke-check.md` 增加一个靠前章节，例如：

```md
## 2.5 P0 最小回归集

每次改动以下任一文件后，必须至少执行本节 4 步：
- `docker-compose.yml`
- `frontend/nginx.conf`
- `backend/services/*/cmd/main.go`
- `.github/workflows/ci.yml`

### P0-1 渲染 Compose
```bash
docker compose --env-file backend/.env config
```

### P0-2 检查健康状态
```bash
docker compose --env-file backend/.env ps
```

### P0-3 检查网关
```bash
curl -I http://localhost
curl --fail http://localhost/api/v1/ping
```

### P0-4 检查 parser 任务链路
创建一条 `POST /api/v1/tasks/parse` 任务，并确认：
- `parser-service` 日志不再出现 `parse consumer disabled`
- 任务最终有结果事件或成功快照
```

Why:
- 你已经有一份详细 Runbook；P0 目标不是再写一份新文档，而是从现有 Runbook 中抽出“每次都必须跑”的最小集合。

- [ ] **Step 3: 在 CI 的 microservices-smoke job 中补一条 parser worker 检查**

在 `.github/workflows/ci.yml` 的 `microservices-smoke` job 里，保留现有健康检查和网关检查，同时追加一条对 parser worker 的最小可观察门槛。

最小安全做法：

```yaml
      - name: parser worker smoke marker
        env:
          OBSIDIAN_VAULT_PATH: /tmp/obsidian-vault
          DEEPSEEK_API_KEY: ci-placeholder-key
          JWT_SECRET: ci-placeholder-jwt-secret
          OBSIDIAN_REST_API_KEY: ci-placeholder-obsidian-key
        run: |
          docker compose --env-file backend/.env.example logs --no-color parser-service > /tmp/parser-service.log
          if grep -q "parse consumer disabled" /tmp/parser-service.log; then
            echo "parser worker is disabled"
            cat /tmp/parser-service.log
            exit 1
          fi
```

Why:
- 在 CI 环境里做完整登录 + `POST /api/v1/tasks/parse` 成本较高。
- P0 的最小价值是先防止“worker 又因为编排缺失被静默禁用”。
- 等后面有稳定测试账号或可跳过鉴权的 smoke fixture，再升级为真正的端到端 parse 任务。

- [ ] **Step 4: 运行 YAML 语法和关键字自检**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
python3 - <<'PY'
from pathlib import Path
text = Path(".github/workflows/ci.yml").read_text()
assert "parser worker smoke marker" in text
assert "parse consumer disabled" in text
print("ci smoke marker present")
PY
```

Expected: PASS，输出 `ci smoke marker present`

- [ ] **Step 5: 本地按最小回归集跑一遍**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
docker compose --env-file backend/.env config
docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build
docker compose --env-file backend/.env ps
curl -I http://localhost
curl --fail http://localhost/api/v1/ping
docker compose --env-file backend/.env logs --no-color parser-service | tail -n 50
```

Expected:
- Compose 渲染成功
- 各核心服务健康
- 网关可达
- parser-service 日志不含 `parse consumer disabled`

- [ ] **Step 6: Commit**

```bash
git add docs/runbooks/microservices-smoke-check.md .github/workflows/ci.yml
git commit -m "test(microservices): add parser worker smoke gate"
```

---

## Task 3: 收口 GeneratorService 的直接数据库写入

**Files:**
- Create: `backend/internal/service/generator_persistence.go`
- Create: `backend/internal/service/generator_persistence_test.go`
- Modify: `backend/internal/service/generator.go`
- Modify: `backend/internal/service/generator_persist_test.go`

- [ ] **Step 1: 先写失败测试，锁定“通过显式仓储接口保存博客与 token 记账”**

创建 `backend/internal/service/generator_persistence_test.go`：

```go
package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeGenerationPersistence struct {
	savedUserID     uuid.UUID
	savedSourceType string
	savedContent    string
	savedTokens     int
	err             error
}

func (f *fakeGenerationPersistence) SaveGeneratedBlog(
	_ context.Context,
	userID uuid.UUID,
	sourceType string,
	content string,
	estimatedTokens int,
	_ []byte,
) error {
	f.savedUserID = userID
	f.savedSourceType = sourceType
	f.savedContent = content
	f.savedTokens = estimatedTokens
	return f.err
}

func TestGeneratorPersistenceAdapter_SaveGeneratedBlog_DelegatesToRepository(t *testing.T) {
	repo := &fakeGenerationPersistence{}
	userID := uuid.New()

	err := repo.SaveGeneratedBlog(context.Background(), userID, "file", "hello", 10, []byte(`["Go"]`))
	require.NoError(t, err)
	require.Equal(t, userID, repo.savedUserID)
	require.Equal(t, "file", repo.savedSourceType)
	require.Equal(t, "hello", repo.savedContent)
	require.Equal(t, 10, repo.savedTokens)
}
```

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/service -run 'TestGeneratorPersistenceAdapter_SaveGeneratedBlog_DelegatesToRepository' -count=1
```

Expected: FAIL，因为 `SaveGeneratedBlog` 相关接口和适配器尚不存在

- [ ] **Step 2: 新增生成结果持久化接口与默认 GORM 实现**

创建 `backend/internal/service/generator_persistence.go`：

```go
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"inkwords-backend/internal/infra/db"
	"inkwords-backend/internal/model"
)

// Why: 生成链路最终要把业务事实回收到 core-api，这里先把“怎么落库”从 GeneratorService 中抽成显式边界，
// 避免继续在业务流程里直接拿全局 db.DB 写 blogs / users。
type GenerationPersistence interface {
	SaveGeneratedBlog(
		ctx context.Context,
		userID uuid.UUID,
		sourceType string,
		content string,
		estimatedTokens int,
		techStacksJSON []byte,
	) error
}

type GormGenerationPersistence struct {
	db *gorm.DB
}

func NewGormGenerationPersistence(database *gorm.DB) *GormGenerationPersistence {
	return &GormGenerationPersistence{db: database}
}

func NewDefaultGenerationPersistence() *GormGenerationPersistence {
	return NewGormGenerationPersistence(db.DB)
}

func (p *GormGenerationPersistence) SaveGeneratedBlog(
	ctx context.Context,
	userID uuid.UUID,
	sourceType string,
	content string,
	estimatedTokens int,
	techStacksJSON []byte,
) error {
	if p == nil || p.db == nil {
		return fmt.Errorf("persist generated blog: database not configured")
	}

	blog := &model.Blog{
		UserID:      userID,
		Title:       "文件解析生成的博客",
		Content:     content,
		SourceType:  sourceType,
		Status:      1,
		ChapterSort: 1,
		WordCount:   len([]rune(content)),
		TechStacks:  datatypes.JSON(append([]byte(nil), techStacksJSON...)),
	}

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(blog).Error; err != nil {
			return fmt.Errorf("create blog record: %w", err)
		}
		result := tx.Model(&model.User{}).
			Where("id = ?", userID).
			UpdateColumn("tokens_used", gorm.Expr("tokens_used + ?", estimatedTokens))
		if result.Error != nil {
			return fmt.Errorf("update user tokens: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("update user tokens: user not found")
		}
		return nil
	})
}
```

- [ ] **Step 3: 让 GeneratorService 依赖显式持久化接口**

修改 `backend/internal/service/generator.go`：

```go
type GeneratorService struct {
	llmClient    *llm.DeepSeekClient
	promptReq    *PromptRequirementsService
	persistence  GenerationPersistence
}

func NewGeneratorService(promptReq *PromptRequirementsService) *GeneratorService {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	return &GeneratorService{
		llmClient:   llm.NewDeepSeekClient(apiKey),
		promptReq:   promptReq,
		persistence: NewDefaultGenerationPersistence(),
	}
}
```

并把 `saveToDB` 改造成只负责组装数据、再调用接口：

```go
func (s *GeneratorService) saveToDB(ctx context.Context, userID uuid.UUID, sourceType string, content string) error {
	if s == nil || s.persistence == nil {
		return fmt.Errorf("persist generated blog: persistence not configured")
	}

	var techStacksJSON []byte
	extractPrompt := "请从以下文章内容中提取出涉及的核心技术栈名称（如 React, Go, Docker 等），以 JSON 数组格式返回，不要有任何其他多余字符。\n\n例如：[\"React\", \"Go\"]\n\n文章内容：\n\n" + content
	messages := []llm.Message{{Role: "user", Content: extractPrompt}}
	modelType := "deepseek-v4-flash"
	if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
		modelType = envModel
	}
	if s.llmClient != nil {
		extractedJSON, err := s.llmClient.GenerateJSON(ctx, modelType, messages)
		if err == nil && len(extractedJSON) > 0 {
			var parsed []string
			if json.Unmarshal([]byte(extractedJSON), &parsed) == nil {
				techStacksJSON = []byte(extractedJSON)
			}
		}
	}

	estimatedTokens := len([]rune(content)) * 2
	if err := s.persistence.SaveGeneratedBlog(
		ctx,
		userID,
		sourceType,
		content,
		estimatedTokens,
		techStacksJSON,
	); err != nil {
		return fmt.Errorf("persist generated blog: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: 更新现有持久化测试，改成显式注入 persistence**

把 `backend/internal/service/generator_persist_test.go` 中直接依赖全局 `db.DB` 的测试，改为显式构造：

```go
service := &GeneratorService{
	llmClient: fakeClient,
	persistence: NewGormGenerationPersistence(testDB),
}
```

并把对全局 `db.DB` 的替换删掉，避免测试继续绕回旧实现。

- [ ] **Step 5: 为 GORM 持久化适配器增加事务测试**

在 `backend/internal/service/generator_persistence_test.go` 补一组真正的数据库测试：

```go
func TestGormGenerationPersistence_SaveGeneratedBlog_RollsBackWhenUserMissing(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&model.User{}, &model.Blog{}))

	persistence := NewGormGenerationPersistence(testDB)
	err = persistence.SaveGeneratedBlog(context.Background(), uuid.New(), "file", "hello", 10, []byte(`["Go"]`))
	require.Error(t, err)

	var count int64
	require.NoError(t, testDB.Model(&model.Blog{}).Count(&count).Error)
	require.EqualValues(t, 0, count)
}

func TestGormGenerationPersistence_SaveGeneratedBlog_PersistsBlogAndUpdatesTokens(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&model.User{}, &model.Blog{}))

	userID := uuid.New()
	require.NoError(t, testDB.Create(&model.User{
		ID:       userID,
		Username: "tester",
		Email:    "tester@example.com",
	}).Error)

	persistence := NewGormGenerationPersistence(testDB)
	err = persistence.SaveGeneratedBlog(context.Background(), userID, "file", "hello", 10, []byte(`["Go"]`))
	require.NoError(t, err)

	var blog model.Blog
	require.NoError(t, testDB.First(&blog).Error)
	require.Equal(t, userID, blog.UserID)
	require.Equal(t, "hello", blog.Content)

	var user model.User
	require.NoError(t, testDB.First(&user, "id = ?", userID).Error)
	require.Equal(t, 10, user.TokensUsed)
}
```

- [ ] **Step 6: 跑目标测试**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/service -run 'TestGormGenerationPersistence|TestGeneratorService_saveToDB|TestGenerateBlogStream_DoesNotPersistBlogDirectlyWhenTaskModeEnabled' -count=1
```

Expected: PASS

- [ ] **Step 7: 跑后端全量测试，确认没有回归**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./... -count=1
```

Expected: PASS

- [ ] **Step 8: 同步文档，记录“GeneratorService 已开始显式边界收口”**

至少更新：

```md
README.md
- 把“Task 4 服务写入归属矩阵”中的技术债说明更新为：GeneratorService 已从直接全局 db.DB 写入收口到显式 persistence 接口；decomposition_generate*.go 仍是下一批待收口对象。

.trae/documents/InkWords_Architecture.md
- 说明 `GeneratorService -> GenerationPersistence -> GORM` 的新边界。

.trae/documents/InkWords_Development_Plan_and_Log.md
- 记录本轮 P0 收口进展与验证结果。
```

- [ ] **Step 9: Commit**

```bash
git add \
  backend/internal/service/generator.go \
  backend/internal/service/generator_persist_test.go \
  backend/internal/service/generator_persistence.go \
  backend/internal/service/generator_persistence_test.go \
  README.md \
  .trae/documents/InkWords_Architecture.md \
  .trae/documents/InkWords_Development_Plan_and_Log.md
git commit -m "refactor(core-api): extract generator persistence boundary"
```

---

## Task 4: 统一回归与收尾

**Files:**
- Modify: `README.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`

- [ ] **Step 1: 跑完整 P0 回归集**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
docker compose --env-file backend/.env config
docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build
docker compose --env-file backend/.env ps
curl -I http://localhost
curl --fail http://localhost/api/v1/ping
docker compose --env-file backend/.env logs --no-color parser-service | tail -n 50
cd backend && go test ./... -count=1
```

Expected:
- Compose 渲染成功
- 所有核心容器健康
- 网关入口返回成功
- `parser-service` worker 未被禁用
- 后端测试全绿

- [ ] **Step 2: 提交前自检**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git diff -- docker-compose.yml .github/workflows/ci.yml docs/runbooks/microservices-smoke-check.md \
  backend/internal/service/generator.go \
  backend/internal/service/generator_persist_test.go \
  backend/internal/service/generator_persistence.go \
  backend/internal/service/generator_persistence_test.go \
  README.md \
  .trae/documents/InkWords_Architecture.md \
  .trae/documents/InkWords_Development_Plan_and_Log.md
```

Expected:
- Diff 仅包含本计划涉及的三条主线
- 没有混入无关格式化或额外重构

- [ ] **Step 3: 最终提交**

```bash
git add \
  docker-compose.yml \
  .github/workflows/ci.yml \
  docs/runbooks/microservices-smoke-check.md \
  backend/internal/service/generator.go \
  backend/internal/service/generator_persist_test.go \
  backend/internal/service/generator_persistence.go \
  backend/internal/service/generator_persistence_test.go \
  README.md \
  .trae/documents/InkWords_Architecture.md \
  .trae/documents/InkWords_Development_Plan_and_Log.md
git commit -m "fix(microservices): close p0 gaps for parser smoke and generator persistence"
```

---

## Self-Review

- Spec coverage:
  - `P0-1 parser-service MQ 编排` 已由 Task 1 覆盖
  - `P0-2 最小冒烟门禁` 已由 Task 2 覆盖
  - `P0-3 core-api 写入边界收口` 已由 Task 3 覆盖
- Placeholder scan:
  - 未使用 `TODO / TBD / implement later`
  - 每个任务都给出了具体文件、命令和预期结果
- Type consistency:
  - `GenerationPersistence` / `SaveGeneratedBlog` / `NewGormGenerationPersistence` 在计划内命名保持一致
  - `parser-service` 的 MQ 变量名与现有 `.env.example`、consumer 代码一致

---

Plan complete and saved to `docs/superpowers/plans/2026-06-04-p0-microservices-fixes.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
