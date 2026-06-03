# InkWords Docker 微服务化（Phase 1）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将后端拆分为 `core-api` 与 `llm-stream` 两个 Docker 服务，并通过前端 Nginx 按路径分流，实现仅 `llm-stream` 可独立扩容，且前端对外 URL 不变。

**Architecture:** 同一仓库内新增两个 Go 启动入口（两个二进制、两个容器），共享同一 PostgreSQL/Redis 配置；Nginx 将 `/api/v1/stream/*` 与 `/api/v1/blogs/:id/(continue|polish)` 分流到 `llm-stream`，其余 `/api/*` 分流到 `core-api`。

**Tech Stack:** Go 1.25 + Gin + GORM + PostgreSQL + Docker Compose + Nginx + SSE

---

## File Map（将修改/新增的文件）

**Backend**
- Create: `backend/cmd/core-api/main.go`
- Create: `backend/cmd/llm-stream/main.go`
- Modify: `backend/internal/transport/http/v1/routes.go`（拆分注册函数，保留旧 Register 以便回滚）
- Modify: `backend/internal/transport/http/v1/routes_test.go`（新增 RegisterCore/RegisterStream 的测试）
- Modify: `backend/Dockerfile`（构建多个二进制，运行时通过命令选择）

**Infra**
- Modify: `docker-compose.yml`（新增 `core-api`、`llm-stream` 两个服务并支持 `--scale llm-stream=N`）

**Gateway**
- Modify: `frontend/nginx.conf`（按路径分流到 `core-api` / `llm-stream`，并保持 SSE 配置）

**Docs**
- Modify: `.trae/documents/InkWords_API.md`（补充“Docker 微服务分流不改变 API URL，但后端分服务承载”的说明）
- Modify: `README.md`（补充 `--scale llm-stream` 的扩容方式与服务拓扑）

---

### Task 1: 后端路由注册拆分（RegisterCore / RegisterStream）

**Files:**
- Modify: `backend/internal/transport/http/v1/routes.go`
- Test: `backend/internal/transport/http/v1/routes_test.go`

- [ ] **Step 1: 写失败测试（core/stream 分别校验“缺 handler 会 panic”）**

在 `routes_test.go` 增加两个新用例（先红）：

```go
func TestRegisterCore_PanicsWhenHandlerMissing(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    authMiddleware := func(c *gin.Context) { c.Next() }
    ok := func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) }

    defer func() {
        if recover() == nil {
            t.Fatalf("expected panic when handlers are missing")
        }
    }()

    RegisterCore(r, authMiddleware, CoreHandlers{
        Auth: AuthHandlers{
            Register: ok, Login: ok, BindGithub: ok, GetCaptcha: ok, OAuthRedirect: ok, OAuthCallback: ok,
        },
        User: UserHandlers{
            GetProfile: ok, UpdateProfile: ok, UploadAvatar: ok, GetUserStats: ok, GetPromptSettings: ok, UpdatePromptSettings: ok,
        },
        Blog: CoreBlogHandlers{
            GetUserBlogs: ok, CreateDraftBlog: ok, BatchDeleteBlogs: ok, UpdateBlog: ok, ExportSeries: ok, ExportSeriesPDF: ok, ExportToObsidian: ok, ExportSeriesToObsidian: ok,
        },
        Project: ProjectHandlers{ScanGithubRepo: ok, Analyze: ok, Parse: ok},
        Review: ReviewHandlers{},
    })
}

func TestRegisterStream_PanicsWhenHandlerMissing(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    authMiddleware := func(c *gin.Context) { c.Next() }
    ok := func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) }

    defer func() {
        if recover() == nil {
            t.Fatalf("expected panic when handlers are missing")
        }
    }()

    RegisterStream(r, authMiddleware, StreamOnlyHandlers{
        Blog: StreamBlogHandlers{
            ContinueBlog: ok,
            PolishBlog: ok,
        },
        Stream: StreamHandlers{
            ScanStreamHandler: ok, AnalyzeStreamHandler: ok, GenerateStreamHandler: ok,
        },
    })
}
```

- [ ] **Step 2: 跑测试确认失败**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/transport/http/v1 -run 'TestRegister(Core|Stream)_PanicsWhenHandlerMissing' -count=1
```

Expected: FAIL（`RegisterCore` / `RegisterStream` 未定义）

- [ ] **Step 3: 写最小实现（新增 CoreHandlers / StreamOnlyHandlers + RegisterCore / RegisterStream）**

在 `routes.go` 增加类型与函数（保持现有 `Register` 不动，便于回滚）：

```go
type CoreBlogHandlers struct {
    GetUserBlogs           gin.HandlerFunc
    CreateDraftBlog        gin.HandlerFunc
    BatchDeleteBlogs       gin.HandlerFunc
    UpdateBlog             gin.HandlerFunc
    ExportSeries           gin.HandlerFunc
    ExportSeriesPDF        gin.HandlerFunc
    ExportToObsidian       gin.HandlerFunc
    ExportSeriesToObsidian gin.HandlerFunc
}

type CoreHandlers struct {
    Auth    AuthHandlers
    User    UserHandlers
    Blog    CoreBlogHandlers
    Project ProjectHandlers
    Review  ReviewHandlers
}

type StreamBlogHandlers struct {
    ContinueBlog gin.HandlerFunc
    PolishBlog   gin.HandlerFunc
}

type StreamOnlyHandlers struct {
    Blog   StreamBlogHandlers
    Stream StreamHandlers
}

func RegisterCore(r *gin.Engine, authMiddleware gin.HandlerFunc, handlers CoreHandlers) {
    if authMiddleware == nil {
        panic("missing middleware: authMiddleware")
    }
    validateCoreHandlers(handlers)

    v1 := r.Group("/api/v1")
    {
        // /auth
        // /user (authMiddleware)
        // /blogs (authMiddleware) - 不注册 continue/polish
        // /project (authMiddleware)
        // /review (authMiddleware)
    }
}

func RegisterStream(r *gin.Engine, authMiddleware gin.HandlerFunc, handlers StreamOnlyHandlers) {
    if authMiddleware == nil {
        panic("missing middleware: authMiddleware")
    }
    validateStreamOnlyHandlers(handlers)

    v1 := r.Group("/api/v1")
    {
        // /stream (authMiddleware)
        // /blogs/:id/continue & /blogs/:id/polish (authMiddleware)
    }
}

func validateCoreHandlers(h CoreHandlers) { /* 按需 must */ }
func validateStreamOnlyHandlers(h StreamOnlyHandlers) { /* 按需 must */ }
```

- [ ] **Step 4: 增加可达性测试（core/stream 路由可访问）**

在 `routes_test.go` 增加两条用例（覆盖关键路径）：

```go
func TestRegisterCore_RoutesAreReachable(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    authMiddleware := func(c *gin.Context) { c.Next() }
    ok := func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) }

    RegisterCore(r, authMiddleware, CoreHandlers{
        Auth: AuthHandlers{Register: ok, Login: ok, BindGithub: ok, GetCaptcha: ok, OAuthRedirect: ok, OAuthCallback: ok},
        User: UserHandlers{GetProfile: ok, UpdateProfile: ok, UploadAvatar: ok, GetUserStats: ok, GetPromptSettings: ok, UpdatePromptSettings: ok},
        Blog: CoreBlogHandlers{GetUserBlogs: ok, CreateDraftBlog: ok, BatchDeleteBlogs: ok, UpdateBlog: ok, ExportSeries: ok, ExportSeriesPDF: ok, ExportToObsidian: ok, ExportSeriesToObsidian: ok},
        Project: ProjectHandlers{ScanGithubRepo: ok, Analyze: ok, Parse: ok},
        Review: ReviewHandlers{GetTodayCard: ok, GetHistory: ok, PickRandom: ok, ListNotes: ok, CreateSession: ok, GetSession: ok, Respond: ok, RequestHint: ok, Finish: ok},
    })

    for _, tc := range []struct {
        method string
        path string
    }{
        {http.MethodPost, "/api/v1/auth/login"},
        {http.MethodGet, "/api/v1/user/profile"},
        {http.MethodGet, "/api/v1/blogs"},
        {http.MethodPost, "/api/v1/project/parse"},
        {http.MethodGet, "/api/v1/review/today"},
    } {
        req := httptest.NewRequest(tc.method, tc.path, nil)
        w := httptest.NewRecorder()
        r.ServeHTTP(w, req)
        if w.Code != http.StatusOK {
            t.Fatalf("%s %s expected 200, got %d", tc.method, tc.path, w.Code)
        }
    }
}

func TestRegisterStream_RoutesAreReachable(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    authMiddleware := func(c *gin.Context) { c.Next() }
    ok := func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) }

    RegisterStream(r, authMiddleware, StreamOnlyHandlers{
        Blog: StreamBlogHandlers{ContinueBlog: ok, PolishBlog: ok},
        Stream: StreamHandlers{ScanStreamHandler: ok, AnalyzeStreamHandler: ok, GenerateStreamHandler: ok},
    })

    for _, tc := range []struct {
        method string
        path string
    }{
        {http.MethodPost, "/api/v1/stream/generate"},
        {http.MethodPost, "/api/v1/blogs/123/continue"},
        {http.MethodPost, "/api/v1/blogs/123/polish"},
    } {
        req := httptest.NewRequest(tc.method, tc.path, nil)
        w := httptest.NewRecorder()
        r.ServeHTTP(w, req)
        if w.Code != http.StatusOK {
            t.Fatalf("%s %s expected 200, got %d", tc.method, tc.path, w.Code)
        }
    }
}
```

- [ ] **Step 5: 跑测试确认通过**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/transport/http/v1 -count=1
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/transport/http/v1/routes.go backend/internal/transport/http/v1/routes_test.go
git commit -m "refactor(transport): split route registration for core and stream"
```

---

### Task 2: 新增两个后端启动入口（core-api / llm-stream）

**Files:**
- Create: `backend/cmd/core-api/main.go`
- Create: `backend/cmd/llm-stream/main.go`
- Modify: `backend/cmd/server/main.go`（改为复用拆分后的 Register，或保持现状但切到新 Register）
- Test: `backend/cmd/server/main_test.go`（如需要，补充“可启动/可优雅停机”的最小断言）

- [ ] **Step 1: 写失败测试（确保两个入口至少能编译）**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./cmd/... -count=1
```

Expected: FAIL（新入口不存在）

- [ ] **Step 2: 写 `cmd/core-api/main.go`（注册 core 路由）**

核心结构（保持和现有 `cmd/server/main.go` 相同的初始化方式，但只组装 core 依赖）：

```go
package main

import (
    "context"
    "errors"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/joho/godotenv"

    authdomain "inkwords-backend/internal/domain/auth"
    blogdomain "inkwords-backend/internal/domain/blog"
    projectdomain "inkwords-backend/internal/domain/project"
    reviewdomain "inkwords-backend/internal/domain/review"
    userdomain "inkwords-backend/internal/domain/user"
    "inkwords-backend/internal/infra/cache"
    "inkwords-backend/internal/infra/db"
    "inkwords-backend/internal/infra/parser"
    "inkwords-backend/internal/service"
    "inkwords-backend/internal/transport/http/middleware"
    transportv1 "inkwords-backend/internal/transport/http/v1"
    "inkwords-backend/internal/transport/http/v1/api"
)

func init() { _ = godotenv.Load() }

func main() {
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" { log.Fatal("DATABASE_URL environment variable is not set") }
    if err := db.InitDB(dsn); err != nil { log.Fatalf("Database initialization failed: %v", err) }
    if err := cache.InitRedis(); err != nil { log.Printf("Redis initialization failed (cache will be disabled): %v", err) }

    r := gin.Default()
    r.MaxMultipartMemory = 888 << 20
    r.Static("/uploads", "./uploads")
    r.GET("/api/v1/ping", func(c *gin.Context) { c.JSON(200, gin.H{"code": 200, "message": "pong", "data": nil}) })

    userService := service.NewUserService(db.DB)
    blogService := service.NewBlogService()
    promptReqService := service.NewPromptRequirementsService(db.DB)
    decompositionService := service.NewDecompositionService(promptReqService)
    gitFetcher := parser.NewGitFetcher()
    docParser := parser.NewDocParser()

    authRepo := authdomain.NewGormRepository(db.DB)
    authDomainHandler := authdomain.NewHandler(authdomain.NewService(authRepo))

    blogRepo := blogdomain.NewGormRepository(db.DB)
    blogDomainHandler := blogdomain.NewHandlerWithLegacy(blogdomain.NewService(blogRepo), blogService)

    userRepo := userdomain.NewGormRepository(db.DB)
    userDomainHandler := userdomain.NewHandler(userdomain.NewService(userRepo))

    projectDomainHandler := projectdomain.NewHandler(projectdomain.NewService(decompositionService, gitFetcher, docParser, userService))

    reviewRepo := reviewdomain.NewGormRepository(db.DB)
    reviewNoteSource := buildReviewNoteSource()
    reviewDomainHandler := reviewdomain.NewHandler(reviewdomain.NewService(reviewRepo, reviewNoteSource))

    authAPI := api.NewAuthAPIWithDeps(authDomainHandler)
    userAPI := api.NewUserAPIWithDeps(userService, userDomainHandler)
    projectAPI := api.NewProjectAPIWithDeps(decompositionService, gitFetcher, docParser, userService, projectDomainHandler)
    blogAPI := api.NewBlogAPIWithDeps(blogService, blogDomainHandler)

    transportv1.RegisterCore(r, middleware.AuthMiddleware(), transportv1.CoreHandlers{
        Auth: transportv1.AuthHandlers{
            Register: authAPI.Register, Login: authAPI.Login, BindGithub: authAPI.BindGithub, GetCaptcha: authAPI.GetCaptcha, OAuthRedirect: authAPI.OAuthRedirect, OAuthCallback: authAPI.OAuthCallback,
        },
        User: transportv1.UserHandlers{
            GetProfile: userAPI.GetProfile, UpdateProfile: userAPI.UpdateProfile, UploadAvatar: userAPI.UploadAvatar, GetUserStats: userAPI.GetUserStats, GetPromptSettings: userAPI.GetPromptSettings, UpdatePromptSettings: userAPI.UpdatePromptSettings,
        },
        Blog: transportv1.CoreBlogHandlers{
            GetUserBlogs: blogAPI.GetUserBlogs, CreateDraftBlog: blogAPI.CreateDraftBlog, BatchDeleteBlogs: blogAPI.BatchDeleteBlogs, UpdateBlog: blogAPI.UpdateBlog,
            ExportSeries: blogAPI.ExportSeries, ExportSeriesPDF: blogAPI.ExportSeriesPDF, ExportToObsidian: blogAPI.ExportToObsidian, ExportSeriesToObsidian: blogAPI.ExportSeriesToObsidian,
        },
        Project: transportv1.ProjectHandlers{
            ScanGithubRepo: projectAPI.ScanGithubRepo, Analyze: projectAPI.Analyze, Parse: projectAPI.Parse,
        },
        Review: transportv1.ReviewHandlers{
            GetTodayCard: reviewDomainHandler.GetTodayCard, GetHistory: reviewDomainHandler.GetHistory, PickRandom: reviewDomainHandler.PickRandom, ListNotes: reviewDomainHandler.ListNotes,
            CreateSession: reviewDomainHandler.CreateSession, GetSession: reviewDomainHandler.GetSession, Respond: reviewDomainHandler.Respond, RequestHint: reviewDomainHandler.RequestHint, Finish: reviewDomainHandler.Finish,
        },
    })

    server := newHTTPServer(r)
    signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    go shutdownServerOnContextDone(signalContext, server, 15*time.Second)

    log.Printf("Server is running on %s", server.Addr)
    if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
        log.Fatalf("Server startup failed: %v", err)
    }
}
```

- [ ] **Step 3: 写 `cmd/llm-stream/main.go`（只注册 stream/continue/polish）**

核心结构（只组装 stream 相关依赖）：

```go
package main

import (
    "context"
    "errors"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/joho/godotenv"

    streamdomain "inkwords-backend/internal/domain/stream"
    "inkwords-backend/internal/infra/cache"
    "inkwords-backend/internal/infra/db"
    "inkwords-backend/internal/service"
    "inkwords-backend/internal/transport/http/middleware"
    transportv1 "inkwords-backend/internal/transport/http/v1"
    "inkwords-backend/internal/transport/http/v1/api"
)

func init() { _ = godotenv.Load() }

func main() {
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" { log.Fatal("DATABASE_URL environment variable is not set") }
    if err := db.InitDB(dsn); err != nil { log.Fatalf("Database initialization failed: %v", err) }
    if err := cache.InitRedis(); err != nil { log.Printf("Redis initialization failed (cache will be disabled): %v", err) }

    r := gin.Default()
    r.GET("/api/v1/ping", func(c *gin.Context) { c.JSON(200, gin.H{"code": 200, "message": "pong", "data": nil}) })

    userService := service.NewUserService(db.DB)
    promptReqService := service.NewPromptRequirementsService(db.DB)
    generatorService := service.NewGeneratorService(promptReqService)
    decompositionService := service.NewDecompositionService(promptReqService)

    streamDomainService := streamdomain.NewService(generatorService, decompositionService, userService)
    streamDomainHandler := streamdomain.NewHandler(streamDomainService, streamdomain.NewGormBlogReadable())
    streamAPI := api.NewStreamAPIWithDeps(generatorService, decompositionService, userService, streamDomainHandler)

    transportv1.RegisterStream(r, middleware.AuthMiddleware(), transportv1.StreamOnlyHandlers{
        Blog: transportv1.StreamBlogHandlers{
            ContinueBlog: streamAPI.ContinueBlogStreamHandler,
            PolishBlog: streamAPI.PolishBlogStreamHandler,
        },
        Stream: transportv1.StreamHandlers{
            ScanStreamHandler: streamAPI.ScanStreamHandler,
            AnalyzeStreamHandler: streamAPI.AnalyzeStreamHandler,
            GenerateStreamHandler: streamAPI.GenerateBlogStreamHandler,
        },
    })

    server := newHTTPServer(r)
    signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    go shutdownServerOnContextDone(signalContext, server, 15*time.Second)

    log.Printf("Server is running on %s", server.Addr)
    if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
        log.Fatalf("Server startup failed: %v", err)
    }
}
```

- [ ] **Step 4: 调整单体入口 `cmd/server`（保持可用）**

把 `cmd/server/main.go` 的 `transportv1.Register(...)` 改为继续调用旧 `Register` 或改为同时调用 `RegisterCore` + `RegisterStream`（二选一，推荐后者以确保两套注册一致）：

```go
transportv1.RegisterCore(r, middleware.AuthMiddleware(), /* CoreHandlers ... */)
transportv1.RegisterStream(r, middleware.AuthMiddleware(), /* StreamOnlyHandlers ... */)
```

- [ ] **Step 5: 跑编译与单测**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./... -count=1
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/cmd/core-api backend/cmd/llm-stream backend/cmd/server/main.go
git commit -m "feat(backend): add core-api and llm-stream entrypoints"
```

---

### Task 3: Dockerfile 支持多二进制 + Compose 拆服务 + Nginx 分流

**Files:**
- Modify: `backend/Dockerfile`
- Modify: `docker-compose.yml`
- Modify: `frontend/nginx.conf`

- [ ] **Step 1: 后端 Dockerfile 构建三个二进制（server/core-api/llm-stream）**

将 build 阶段从单一 `./cmd/server` 改为多次 `go build`：

```dockerfile
RUN CGO_ENABLED=0 GOOS=linux go build -o core-api ./cmd/core-api
RUN CGO_ENABLED=0 GOOS=linux go build -o llm-stream ./cmd/llm-stream
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server
```

并在 runtime 阶段复制三个二进制：

```dockerfile
COPY --from=builder /app/core-api .
COPY --from=builder /app/llm-stream .
COPY --from=builder /app/server .
```

默认 `CMD` 可保持 `["./server"]`（单体对照），Compose 中分别覆盖 command。

- [ ] **Step 2: Compose 拆分 backend → core-api + llm-stream**

在 `docker-compose.yml`：
- 把原 `backend` 服务改为 `core-api`
- 新增 `llm-stream` 服务（同样 build backend 镜像，但 `command: ["./llm-stream"]`）
- `frontend` 依赖改为 `core-api` 与 `llm-stream`
- `core-api` 保留 `uploads` volume 与 `OBSIDIAN_VAULT_PATH` bind mount；`llm-stream` 不挂载 vault/ uploads

- [ ] **Step 3: 前端 Nginx 按路径分流**

在 `frontend/nginx.conf`：
- 保留静态与 `/uploads/`（指向 `core-api`）
- 新增两段 location：

```nginx
location ^~ /api/v1/stream/ {
    proxy_pass http://llm-stream:8080/api/v1/stream/;
    # SSE 设置同现有
}

location ~ ^/api/v1/blogs/[^/]+/(continue|polish)$ {
    proxy_pass http://llm-stream:8080$request_uri;
    # SSE 设置同现有
}

location /api/ {
    proxy_pass http://core-api:8080/api/;
    # 非 SSE 的通用转发头
}
```

- [ ] **Step 4: 构建并启动（单体回滚可用）**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
docker compose --env-file backend/.env down
docker compose --env-file backend/.env up -d --build
docker compose --env-file backend/.env ps
```

Expected:
- `core-api`、`llm-stream`、`frontend`、`db`、`redis` 状态为 `Up`（db 为 healthy）

- [ ] **Step 5: 冒烟验证（SSE + 非 SSE）**

Run:

```bash
curl -I http://localhost
curl -sS http://localhost/api/v1/ping
```

并用浏览器走完整链路：
- 登录（core-api）
- 生成/分析（llm-stream，SSE）
- 继续生成/润色（llm-stream，SSE）
- 历史列表（core-api）

- [ ] **Step 6: Commit**

```bash
git add backend/Dockerfile docker-compose.yml frontend/nginx.conf
git commit -m "feat(docker): split backend into core-api and llm-stream services"
```

---

### Task 4: 文档同步（Docs-as-Code）

**Files:**
- Modify: `README.md`
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`（如需要）

- [ ] **Step 1: README 增加“独立扩容 llm-stream”的操作说明**

新增示例命令：

```bash
docker compose --env-file backend/.env up -d --build --scale llm-stream=3
```

- [ ] **Step 2: API 文档补充“分流不改变 URL，但由不同服务承载”**

明确：
- `/api/v1/stream/*` 与 `continue/polish` 由 `llm-stream` 承载
- 其余由 `core-api` 承载

- [ ] **Step 3: Commit**

```bash
git add README.md .trae/documents/InkWords_API.md .trae/documents/InkWords_Architecture.md
git commit -m "docs: document docker microservices routing and scaling"
```
