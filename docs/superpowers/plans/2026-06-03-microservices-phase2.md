# Microservices Phase 2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在保持对外入口仍为 `http://localhost` 且对外 API 路由不变的前提下，将后端从当前 `core-api + llm-stream` 继续拆分为 `parser-service / export-service / review-service`，并将 review 数据迁移到独立数据库（线路 A：同一 Postgres 实例、不同 database）。

**Architecture:** 继续以 Nginx 作为单一公开入口，通过路径分流到不同后端服务；服务间不引入消息队列，先以“路由级拆分 + 最小 DI 复制”为主。数据库采用单 Postgres 实例，review 使用独立 database，并提供可回滚的数据迁移步骤。

**Tech Stack:** Go 1.25 + Gin + GORM + PostgreSQL 14（Docker Compose）；Nginx 反向代理；PDF 导出使用 Chromium Headless；Obsidian 导出使用 Obsidian Local REST API。

---

## 0) Baseline（现状确认）

- 当前已完成 Phase 1：`core-api` 与 `llm-stream` 两服务拆分，Nginx 将 `/api/v1/stream/*` 与 `/api/v1/blogs/:id/(continue|polish)` 分流到 `llm-stream`（参考：[nginx.conf](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/frontend/nginx.conf#L18-L51)）。
- 后端 DB 初始化入口当前为单函数 `db.InitDB(dsn)`，并在同一个数据库里 AutoMigrate 全量模型（含 review）：
  - [db.go](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/backend/internal/infra/db/db.go#L15-L43)
- 目标：在不改变前端调用方式的情况下，继续做“后端路由 + 服务职责”的切分，并把 review 数据迁出 core DB。

---

## 1) Target Boundary（Phase 2 服务边界）

### 1.1 服务与路由归属（外部 API 不变）

- `core-api`
  - `/api/v1/auth/*`
  - `/api/v1/user/*`
  - `/api/v1/blogs`（除 export / continue / polish）
  - `/api/v1/project/scan`、`/api/v1/project/analyze`（Legacy，保留）
- `llm-stream`
  - `/api/v1/stream/*`
  - `/api/v1/blogs/:id/(continue|polish)`
- `parser-service`
  - `/api/v1/project/parse`
- `export-service`
  - `/api/v1/blogs/:id/export`
  - `/api/v1/blogs/:id/export/pdf`
  - `/api/v1/blogs/:id/export/obsidian`
  - `/api/v1/blogs/:id/export/obsidian/series`
- `review-service`
  - `/api/v1/review/*`

### 1.2 数据边界（线路 A）

- 单 Postgres 实例 `db`（Compose 服务不变），新增 database：`inkwords_review_db`
- `review-service` 使用 `REVIEW_DATABASE_URL` 连接到 `inkwords_review_db`
- `core-api` 与 `llm-stream` 继续使用 `DATABASE_URL`（原来的 `${POSTGRES_DB}`，默认 `inkwords_db`）
- review 表从 `inkwords_db` 迁移到 `inkwords_review_db`（保留原表作为回滚依据，不在 Phase 2 直接 drop）

---

## Task 1: 设计路由注册的“按服务拆分”能力

**Files:**
- Modify: `backend/internal/transport/http/v1/routes.go`
- Test: `backend/internal/transport/http/v1/routes_test.go`（新增）

- [ ] **Step 1: 新增服务专用 handlers struct**

在 `routes.go` 添加：

```go
type ExportHandlers struct {
	ExportSeries           gin.HandlerFunc
	ExportSeriesPDF        gin.HandlerFunc
	ExportToObsidian       gin.HandlerFunc
	ExportSeriesToObsidian gin.HandlerFunc
}

type ParserHandlers struct {
	Parse gin.HandlerFunc
}

type ReviewOnlyHandlers struct {
	Review ReviewHandlers
}
```

- [ ] **Step 2: 新增 RegisterExport/RegisterParser/RegisterReview**

在 `routes.go` 添加：

```go
func RegisterExport(r *gin.Engine, authMiddleware gin.HandlerFunc, handlers ExportHandlers) {
	if authMiddleware == nil {
		panic("missing middleware: authMiddleware")
	}
	must(handlers.ExportSeries, "Export.ExportSeries")
	must(handlers.ExportSeriesPDF, "Export.ExportSeriesPDF")
	must(handlers.ExportToObsidian, "Export.ExportToObsidian")
	must(handlers.ExportSeriesToObsidian, "Export.ExportSeriesToObsidian")

	v1 := r.Group("/api/v1")
	{
		blogGroup := v1.Group("/blogs")
		blogGroup.Use(authMiddleware)
		{
			blogGroup.GET("/:id/export", handlers.ExportSeries)
			blogGroup.GET("/:id/export/pdf", handlers.ExportSeriesPDF)
			blogGroup.POST("/:id/export/obsidian", handlers.ExportToObsidian)
			blogGroup.POST("/:id/export/obsidian/series", handlers.ExportSeriesToObsidian)
		}
	}
}

func RegisterParser(r *gin.Engine, authMiddleware gin.HandlerFunc, handlers ParserHandlers) {
	if authMiddleware == nil {
		panic("missing middleware: authMiddleware")
	}
	must(handlers.Parse, "Parser.Parse")

	v1 := r.Group("/api/v1")
	{
		projectGroup := v1.Group("/project")
		projectGroup.Use(authMiddleware)
		{
			projectGroup.POST("/parse", handlers.Parse)
		}
	}
}

func RegisterReview(r *gin.Engine, authMiddleware gin.HandlerFunc, handlers ReviewOnlyHandlers) {
	if authMiddleware == nil {
		panic("missing middleware: authMiddleware")
	}
	validateReviewHandlers(handlers.Review)

	v1 := r.Group("/api/v1")
	{
		reviewGroup := v1.Group("/review")
		reviewGroup.Use(authMiddleware)
		{
			reviewGroup.GET("/today", handlers.Review.GetTodayCard)
			reviewGroup.GET("/history", handlers.Review.GetHistory)
			reviewGroup.POST("/pick", handlers.Review.PickRandom)
			reviewGroup.GET("/notes", handlers.Review.ListNotes)
			reviewGroup.POST("/sessions", handlers.Review.CreateSession)
			reviewGroup.GET("/sessions/:id", handlers.Review.GetSession)
			reviewGroup.POST("/sessions/:id/respond", handlers.Review.Respond)
			reviewGroup.POST("/sessions/:id/hint", handlers.Review.RequestHint)
			reviewGroup.POST("/sessions/:id/finish", handlers.Review.Finish)
		}
	}
}
```

并在文件底部新增：

```go
func validateReviewHandlers(h ReviewHandlers) {
	must(h.GetTodayCard, "Review.GetTodayCard")
	must(h.GetHistory, "Review.GetHistory")
	must(h.PickRandom, "Review.PickRandom")
	must(h.ListNotes, "Review.ListNotes")
	must(h.CreateSession, "Review.CreateSession")
	must(h.GetSession, "Review.GetSession")
	must(h.Respond, "Review.Respond")
	must(h.RequestHint, "Review.RequestHint")
	must(h.Finish, "Review.Finish")
}
```

- [ ] **Step 3: 从 RegisterCore 中移除 parse/export/review**

修改 `CoreHandlers`：
- `CoreHandlers.Project` 仅保留 `ScanGithubRepo`、`Analyze`
- `CoreBlogHandlers` 移除 export 四个 handler
- `CoreHandlers` 移除 `Review`

并同步更新 `validateCoreHandlers` 的 must 校验清单。

- [ ] **Step 4: 新增 routes_test，锁定“各 Register 函数只注册自己负责的路由”**

`routes_test.go` 使用 `httptest.NewRecorder` + `gin.New()` 注册路由，分别断言：
- `RegisterParser` 只存在 `/api/v1/project/parse`，不存在 `/api/v1/project/scan`
- `RegisterExport` 存在 `/api/v1/blogs/:id/export`，不存在 `/api/v1/blogs/:id`
- `RegisterReview` 存在 `/api/v1/review/today` 等

建议测试方式：

```go
func TestRegisterParser_OnlyRegistersParseRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterParser(r, func(c *gin.Context) {}, ParserHandlers{
		Parse: func(c *gin.Context) { c.Status(200) },
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/project/parse", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	require.Equal(t, 200, resp.Code)
}
```

- [ ] **Step 5: 运行测试**

Run:

```bash
cd backend && go test ./internal/transport/http/v1 -run 'TestRegister'
```

Expected: PASS

---

## Task 2: 为 review 拆库提供 DB 初始化入口（core vs review）

**Files:**
- Modify: `backend/internal/infra/db/db.go`
- Modify: `backend/cmd/core-api/main.go`
- Modify: `backend/cmd/llm-stream/main.go`
- Create: `backend/cmd/review-service/main.go`

- [ ] **Step 1: 拆分 InitCoreDB / InitReviewDB**

在 `db.go` 中将 `InitDB` 改为明确的 core/review：

```go
func InitCoreDB(dsn string) error {
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		return err
	}

	if err := autoMigrateCore(DB); err != nil {
		log.Printf("Failed to auto migrate database: %v", err)
		return err
	}

	log.Println("Database connection and migration successful")
	return nil
}

func InitReviewDB(dsn string) error {
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		return err
	}

	if err := autoMigrateReview(DB); err != nil {
		log.Printf("Failed to auto migrate database: %v", err)
		return err
	}

	log.Println("Database connection and migration successful")
	return nil
}
```

并将迁移清单拆开（core 不再迁移 review 表）：

```go
func autoMigrateCore(database *gorm.DB) error {
	return database.AutoMigrate(
		&model.User{},
		&model.Blog{},
		&model.OAuthToken{},
		&model.UserPromptSettings{},
	)
}

func autoMigrateReview(database *gorm.DB) error {
	return database.AutoMigrate(
		&model.ReviewSession{},
		&model.ReviewTurn{},
	)
}
```

为兼容现存 `cmd/server` 或其它引用，可保留：

```go
func InitDB(dsn string) error { return InitCoreDB(dsn) }
```

- [ ] **Step 2: core-api 与 llm-stream 改为 InitCoreDB**

在：
- `backend/cmd/core-api/main.go`
- `backend/cmd/llm-stream/main.go`

把 `db.InitDB(dsn)` 改为 `db.InitCoreDB(dsn)`。

- [ ] **Step 3: 新增 review-service main（使用 InitReviewDB）**

新增 `backend/cmd/review-service/main.go`，结构参考 `cmd/core-api/main.go`，但只装配 review 相关依赖：

```go
func main() {
	dsn := os.Getenv("REVIEW_DATABASE_URL")
	if dsn == "" {
		log.Fatal("REVIEW_DATABASE_URL environment variable is not set")
	}
	if err := db.InitReviewDB(dsn); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	r := gin.Default()
	r.GET("/api/v1/ping", func(c *gin.Context) { ... })

	reviewRepo := reviewdomain.NewGormRepository(db.DB)
	reviewNoteSource := buildReviewNoteSource()
	reviewDomainService := reviewdomain.NewService(reviewRepo, reviewNoteSource)
	reviewDomainHandler := reviewdomain.NewHandler(reviewDomainService)

	transportv1.RegisterReview(r, middleware.AuthMiddleware(), transportv1.ReviewOnlyHandlers{
		Review: transportv1.ReviewHandlers{
			GetTodayCard:  reviewDomainHandler.GetTodayCard,
			GetHistory:    reviewDomainHandler.GetHistory,
			PickRandom:    reviewDomainHandler.PickRandom,
			ListNotes:     reviewDomainHandler.ListNotes,
			CreateSession: reviewDomainHandler.CreateSession,
			GetSession:    reviewDomainHandler.GetSession,
			Respond:       reviewDomainHandler.Respond,
			RequestHint:   reviewDomainHandler.RequestHint,
			Finish:        reviewDomainHandler.Finish,
		},
	})

	server := newHTTPServer(r)
	...
}
```

并把 `buildReviewNoteSource` 复制到该 main（保持与 core-api 旧逻辑一致）。

- [ ] **Step 4: core-api 移除 review wiring**

在 `backend/cmd/core-api/main.go`：
- 删除 `reviewdomain` import
- 删除 `reviewRepo/reviewNoteSource/reviewDomainService/reviewDomainHandler` 相关组装
- `transportv1.RegisterCore` 不再传 `Review` 字段（因为 `CoreHandlers` 已移除）

- [ ] **Step 5: 编译验证**

Run:

```bash
cd backend && go test ./... -run '^$'
```

Expected: PASS（只做编译）

---

## Task 3: 拆出 parser-service（/api/v1/project/parse）

**Files:**
- Create: `backend/internal/domain/fileparse/service.go`
- Create: `backend/internal/domain/fileparse/handler.go`
- Create: `backend/cmd/parser-service/main.go`
- Modify: `backend/cmd/core-api/main.go`
- Modify: `backend/internal/transport/http/v1/api/project.go`（移除 Parse 绑定点或保留但 core 不使用）

- [ ] **Step 1: 新建 fileparse domain（仅承接 parse）**

`service.go`（示例结构，使用现有 `infra/parser` 的 `DocParser` 与 `ArchiveParser`）：

```go
type Service struct {
	docParser     *parser.DocParser
	archiveParser *parser.ArchiveParser
	quotaChecker  QuotaChecker
}

type QuotaChecker interface {
	CheckQuota(userID uint) error
}
```

`handler.go`：逻辑以当前 project handler 的 parse 分支为准（参考：[project handler parse](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/backend/internal/domain/project/handler.go#L130-L183)），保持返回结构不变：

```go
func (h *Handler) Parse(c *gin.Context) {
	rawUserID, exists := c.Get("user_id")
	...
	userID := rawUserID.(uint)
	if err := h.quotaChecker.CheckQuota(userID); err != nil { ... }
	file, header, err := c.Request.FormFile("file")
	...
	sourceContent, archiveSummary, err := h.service.Parse(file, header.Filename)
	...
}
```

- [ ] **Step 2: 新增 parser-service main**

`cmd/parser-service/main.go`：
- 使用 `DATABASE_URL` 连接 core DB（用于 quota check）
- 注入 `service.NewUserService(db.DB)` 作为 `QuotaChecker`
- 创建 `DocParser` 与 `ArchiveParser`（复用 `internal/infra/parser`）
- 注册 `transportv1.RegisterParser`

- [ ] **Step 3: core-api 移除 /project/parse 绑定**

在 `cmd/core-api/main.go`：
- 不再把 `Parse` 传入 `RegisterCore`
- 对应 `CoreHandlers.Project` 只保留 `ScanGithubRepo/Analyze`

- [ ] **Step 4: 运行后端单测（包含 parse 相关 package）**

Run:

```bash
cd backend && go test ./internal/domain/fileparse/... ./internal/infra/parser/... ./internal/transport/http/v1/...
```

Expected: PASS

---

## Task 4: 拆出 export-service（导出/渲染）

**Files:**
- Create: `backend/cmd/export-service/main.go`
- Modify: `backend/cmd/core-api/main.go`

- [ ] **Step 1: 新增 export-service main**

`cmd/export-service/main.go`：
- 使用 `DATABASE_URL` 连接 core DB（读取 blogs）
- 装配 blog domain + legacy exporter（复用现存导出实现，不改逻辑）
  - `blogService := service.NewBlogService()`
  - `blogRepo := blogdomain.NewGormRepository(db.DB)`
  - `blogDomainService := blogdomain.NewService(blogRepo)`
  - `blogDomainHandler := blogdomain.NewHandlerWithLegacy(blogDomainService, blogService)`
  - `blogAPI := api.NewBlogAPIWithDeps(blogService, blogDomainHandler)`
- 注册 `transportv1.RegisterExport`，传入 4 个 export handler

- [ ] **Step 2: core-api 移除 export handler 绑定**

在 `cmd/core-api/main.go`：
- 不再将 export 四个 handler 传入 `RegisterCore`
- `CoreBlogHandlers` 只保留 `GetUserBlogs/CreateDraftBlog/BatchDeleteBlogs/UpdateBlog`

- [ ] **Step 3: 编译验证**

Run:

```bash
cd backend && go test ./... -run '^$'
```

Expected: PASS

---

## Task 5: Docker 构建产物扩展（新增 3 个二进制）

**Files:**
- Modify: `backend/Dockerfile`

- [ ] **Step 1: builder stage 编译新增二进制**

在 `backend/Dockerfile` 的 builder stage 追加：

```dockerfile
RUN CGO_ENABLED=0 GOOS=linux go build -o parser-service ./cmd/parser-service
RUN CGO_ENABLED=0 GOOS=linux go build -o export-service ./cmd/export-service
RUN CGO_ENABLED=0 GOOS=linux go build -o review-service ./cmd/review-service
```

- [ ] **Step 2: runtime stage COPY 新增二进制**

追加：

```dockerfile
COPY --from=builder /app/parser-service .
COPY --from=builder /app/export-service .
COPY --from=builder /app/review-service .
```

- [ ] **Step 3: docker build 验证**

Run:

```bash
docker build -t inkwords-backend:phase2 ./backend
```

Expected: build succeeds

---

## Task 6: Compose/Nginx 分流与环境变量调整

**Files:**
- Modify: `docker-compose.yml`
- Modify: `frontend/nginx.conf`
- Modify: `backend/.env.example`（如存在；若不存在则新增）

- [ ] **Step 1: docker-compose.yml 新增 3 个服务**

新增：
- `parser-service`：`command: ["./parser-service"]`，环境变量继承 core-api 的最小子集（至少 `DATABASE_URL/JWT_SECRET/REDIS_URL`）
- `export-service`：`command: ["./export-service"]`，环境变量继承 core-api 的导出相关子集（含 Obsidian 相关 env、DEEPSEEK_API_KEY 用于系列 Ingest）
- `review-service`：`command: ["./review-service"]`，使用 `REVIEW_DATABASE_URL`（指向 `inkwords_review_db`）+ Obsidian 相关 env + `JWT_SECRET`

注意：对外 ports 仍只保留 frontend 的 `80:80`。

- [ ] **Step 2: db 服务新增 init 脚本，用于创建 inkwords_review_db**

新增一个 SQL 文件并挂载到 `db`：

Create: `backend/db/init/00-create-review-db.sql`

```sql
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'inkwords_review_db') THEN
    CREATE DATABASE inkwords_review_db;
  END IF;
END$$;
```

并在 compose 的 `db.volumes` 下增加：

```yaml
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./backend/db/init:/docker-entrypoint-initdb.d:ro
```

- [ ] **Step 3: nginx.conf 增加路由分流**

新增（顺序放在 `/api/` 之前，避免被兜底匹配吞掉）：
- `/api/v1/project/parse` → `parser-service:8080`
- `/api/v1/review/` → `review-service:8080`
- `/api/v1/blogs/.../export...` → `export-service:8080`

示例：

```nginx
location = /api/v1/project/parse {
    proxy_pass http://parser-service:8080/api/v1/project/parse;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}

location ^~ /api/v1/review/ {
    proxy_pass http://review-service:8080/api/v1/review/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}

location ~ ^/api/v1/blogs/[^/]+/export(/pdf)?$ {
    proxy_pass http://export-service:8080$request_uri;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}

location ~ ^/api/v1/blogs/[^/]+/export/obsidian(/series)?$ {
    proxy_pass http://export-service:8080$request_uri;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

- [ ] **Step 4: docker compose config 校验**

Run:

```bash
docker compose --env-file backend/.env config
```

Expected: no errors

---

## Task 7: Review 数据迁移（需要可回滚）

**Files:**
- Create: `docs/runbooks/review-db-migration.md`

- [ ] **Step 1: 写迁移 Runbook（明确回滚）**

`docs/runbooks/review-db-migration.md` 至少包含：
- 前置条件：db 容器可用、已创建 `inkwords_review_db`
- 迁移命令（从 core db 导出两张表并导入 review db）：

```bash
docker exec -t inkwords-db pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t review_sessions -t review_turns > /tmp/review_dump.sql
docker exec -i inkwords-db psql -U "$POSTGRES_USER" -d inkwords_review_db < /tmp/review_dump.sql
```

- 验证：`psql` 查询两边行数一致：

```bash
docker exec -i inkwords-db psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "select count(*) from review_sessions;"
docker exec -i inkwords-db psql -U "$POSTGRES_USER" -d inkwords_review_db -c "select count(*) from review_sessions;"
```

- 回滚：把 `review-service` 的 `REVIEW_DATABASE_URL` 改回指向 core db，并重启 compose。

- [ ] **Step 2: 迁移前后做一次端到端验证**

Run:

```bash
docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build
curl -I http://localhost
```

Expected:
- `HTTP/1.1 200 OK`
- 登录后 review 历史仍可读取（UI 或接口均可）

---

## Task 8: 文档同步（Docs-as-Code）

**Files:**
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Database.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`

- [ ] **Step 1: Architecture 更新 Phase 2 变更记录**

补充变更记录项：
- 新增 `parser-service/export-service/review-service`
- Nginx 分流规则摘要
- 数据库线路 A：新增 `inkwords_review_db` 与 review 拆库

- [ ] **Step 2: API 文档更新“路由仍不变，但后端由 Nginx 分流”**

只更新变更记录（不改接口表格字段），说明：
- `/api/v1/project/parse` → parser-service
- `/api/v1/blogs/:id/export*` → export-service
- `/api/v1/review/*` → review-service

- [ ] **Step 3: Database 文档补充 review 拆库与迁移 Runbook 引用**

- [ ] **Step 4: 开发日志追加一条 Phase 2 记录（含验证命令与结果占位）**

---

## Self-Review Checklist

- [ ] 计划中所有新增服务的路由归属与 Nginx 分流一致
- [ ] review 拆库的回滚路径明确（不 drop 原表）
- [ ] core-api/llm-stream 在迁移后仍能通过 `DATABASE_URL` 正常启动
- [ ] `docker compose --env-file backend/.env config` 可通过
