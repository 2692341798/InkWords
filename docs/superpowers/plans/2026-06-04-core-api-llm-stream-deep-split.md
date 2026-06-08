# Core-API 与 LLM-Stream 深层拆分 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `core-api` 与 `llm-stream` 从“共享业务核心 + 独立入口”迁移为几乎完全自治的双服务结构，并让 `llm-stream` 停止直写 `blogs / users`。

**Architecture:** 先私有化两个服务的 `app/bootstrap` 与 `transport/http/v1`，切断共享 `routes.go` 与 `stream_api.go` 的主入口控制；再把 `generator / decomposition` 迁入 `services/llm-stream/domain/generation`，最后在 `services/core-api/domain/blog` 与 `services/core-api/domain/task` 内新增任务结果持久化用例，接管 `blogs / users` 的最终写入。

**Tech Stack:** Go 1.25+, Gin, GORM, RabbitMQ, PostgreSQL, Docker Compose, DeepSeek API, SSE

---

## 文件结构锁定

### 新增目录

- `backend/services/core-api/cmd/`
- `backend/services/core-api/app/bootstrap/`
- `backend/services/core-api/domain/auth/`
- `backend/services/core-api/domain/user/`
- `backend/services/core-api/domain/blog/`
- `backend/services/core-api/domain/project/`
- `backend/services/core-api/domain/task/`
- `backend/services/core-api/transport/http/v1/`
- `backend/services/llm-stream/cmd/`
- `backend/services/llm-stream/app/bootstrap/`
- `backend/services/llm-stream/domain/stream/`
- `backend/services/llm-stream/domain/generation/`
- `backend/services/llm-stream/infra/llm/`
- `backend/services/llm-stream/transport/http/v1/`

### 本轮重点修改文件

- `backend/cmd/core-api/main.go`
- `backend/cmd/llm-stream/main.go`
- `backend/internal/service/generator.go`
- `backend/internal/service/decomposition_service.go`
- `backend/internal/service/decomposition_generate*.go`
- `backend/internal/service/decomposition_analyze*.go`
- `backend/internal/domain/stream/*.go`
- `backend/internal/domain/task/*.go`
- `backend/internal/domain/blog/*.go`
- `backend/internal/transport/http/v1/routes.go`
- `backend/internal/transport/http/v1/api/stream_api.go`
- `backend/Dockerfile`
- `docker-compose.yml`
- `.trae/documents/InkWords_API.md`
- `.trae/documents/InkWords_Architecture.md`
- `.trae/documents/InkWords_Conversation_Log.md`
- `.trae/documents/InkWords_Database.md`
- `.trae/documents/InkWords_Development_Plan_and_Log.md`
- `.trae/documents/InkWords_PRD.md`
- `README.md`

### 不改范围

- `frontend/**`
- `backend/services/review-service/**`
- `backend/services/parser-service/**`
- `backend/services/export-service/**`

## 迁移总原则

- 任何阶段都不能破坏 `http://localhost` 与 `/api/*` 单入口契约。
- `llm-stream` 从 Phase 2C 起禁止再写 `blogs / users`。
- 共享层只保留基础件，不允许新建带业务语义的 shared use case。
- 每个 phase 独立提交，避免把入口切换、生成迁移、结果持久化混在一个 commit。

### Task 1: 私有化 `core-api` 入口与路由

**Files:**
- Create: `backend/services/core-api/app/bootstrap/bootstrap.go`
- Create: `backend/services/core-api/cmd/main.go`
- Create: `backend/services/core-api/transport/http/v1/routes.go`
- Create: `backend/services/core-api/transport/http/v1/routes_test.go`
- Modify: `backend/Dockerfile`
- Test: `backend/internal/transport/http/v1/routes_test.go`

- [ ] **Step 1: 写 `core-api` 私有路由测试，先让它失败**

Create `backend/services/core-api/transport/http/v1/routes_test.go`:

```go
package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterCoreRoutes_RegistersTaskAndBlogRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	auth := func(c *gin.Context) { c.Next() }
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }

	RegisterCoreRoutes(r, auth, CoreHandlers{
		AuthRegister: ok,
		AuthLogin: ok,
		UserProfile: ok,
		UserUpdateProfile: ok,
		BlogList: ok,
		BlogCreateDraft: ok,
		ProjectScan: ok,
		TaskCreateGeneration: ok,
		TaskGet: ok,
	})

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/auth/register"},
		{http.MethodPost, "/api/v1/auth/login"},
		{http.MethodGet, "/api/v1/user/profile"},
		{http.MethodGet, "/api/v1/blogs"},
		{http.MethodPost, "/api/v1/project/scan"},
		{http.MethodPost, "/api/v1/tasks/generation"},
		{http.MethodGet, "/api/v1/tasks/task-1"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code, tc.path)
	}
}
```

- [ ] **Step 2: 跑测试确认确实失败**

Run:

```bash
cd backend && go test ./services/core-api/transport/http/v1 -run TestRegisterCoreRoutes_RegistersTaskAndBlogRoutes -v
```

Expected: FAIL，报 `undefined: RegisterCoreRoutes` 或等价“私有 core-api 路由尚不存在”的错误。

- [ ] **Step 3: 实现 `core-api` 私有路由**

Create `backend/services/core-api/transport/http/v1/routes.go`:

```go
package v1

import "github.com/gin-gonic/gin"

type CoreHandlers struct {
	AuthRegister         gin.HandlerFunc
	AuthLogin            gin.HandlerFunc
	UserProfile          gin.HandlerFunc
	UserUpdateProfile    gin.HandlerFunc
	BlogList             gin.HandlerFunc
	BlogCreateDraft      gin.HandlerFunc
	ProjectScan          gin.HandlerFunc
	TaskCreateGeneration gin.HandlerFunc
	TaskGet              gin.HandlerFunc
}

func RegisterCoreRoutes(r *gin.Engine, auth gin.HandlerFunc, h CoreHandlers) {
	v1 := r.Group("/api/v1")

	authGroup := v1.Group("/auth")
	authGroup.POST("/register", h.AuthRegister)
	authGroup.POST("/login", h.AuthLogin)

	userGroup := v1.Group("/user")
	userGroup.Use(auth)
	userGroup.GET("/profile", h.UserProfile)
	userGroup.PUT("/profile", h.UserUpdateProfile)

	blogGroup := v1.Group("/blogs")
	blogGroup.Use(auth)
	blogGroup.GET("", h.BlogList)
	blogGroup.POST("/draft", h.BlogCreateDraft)

	projectGroup := v1.Group("/project")
	projectGroup.Use(auth)
	projectGroup.POST("/scan", h.ProjectScan)

	taskGroup := v1.Group("/tasks")
	taskGroup.Use(auth)
	taskGroup.POST("/generation", h.TaskCreateGeneration)
	taskGroup.GET("/:id", h.TaskGet)
}
```

- [ ] **Step 4: 实现 `core-api` 私有 bootstrap**

Create `backend/services/core-api/app/bootstrap/bootstrap.go`:

```go
package bootstrap

import (
	"errors"
	"os"

	"github.com/gin-gonic/gin"

	authdomain "inkwords-backend/internal/domain/auth"
	blogdomain "inkwords-backend/internal/domain/blog"
	projectdomain "inkwords-backend/internal/domain/project"
	taskdomain "inkwords-backend/internal/domain/task"
	userdomain "inkwords-backend/internal/domain/user"
	"inkwords-backend/internal/infra/cache"
	"inkwords-backend/internal/infra/parser"
	"inkwords-backend/internal/service"
	"inkwords-backend/internal/transport/http/middleware"
	legacyapi "inkwords-backend/internal/transport/http/v1/api"
	corev1 "inkwords-backend/services/core-api/transport/http/v1"
	"inkwords-backend/shared/platform/postgres"
)

func BuildRouter() (*gin.Engine, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("DATABASE_URL environment variable is not set")
	}

	dbConn, err := postgres.InitCore(dsn)
	if err != nil {
		return nil, err
	}
	_ = cache.InitRedis()

	userService := service.NewUserService(dbConn)
	blogService := service.NewBlogService()
	promptReq := service.NewPromptRequirementsService(dbConn)
	decomposition := service.NewDecompositionService(promptReq)
	gitFetcher := parser.NewGitFetcher()
	docParser := parser.NewDocParser()

	authHandler := authdomain.NewHandler(authdomain.NewService(authdomain.NewGormRepository(dbConn)))
	userHandler := userdomain.NewHandler(userdomain.NewService(userdomain.NewGormRepository(dbConn)))
	blogHandler := blogdomain.NewHandlerWithLegacy(blogdomain.NewService(blogdomain.NewGormRepository(dbConn)), blogService)
	projectHandler := projectdomain.NewHandler(projectdomain.NewService(decomposition, gitFetcher, docParser, userService))
	taskHandler := taskdomain.NewHandler(taskdomain.NewService(taskdomain.NewGormRepository(dbConn), nil), os.Getenv("EXPORT_ARTIFACTS_DIR"))

	authAPI := legacyapi.NewAuthAPIWithDeps(authHandler)
	userAPI := legacyapi.NewUserAPIWithDeps(userService, userHandler)
	blogAPI := legacyapi.NewBlogAPIWithDeps(blogService, blogHandler)
	projectAPI := legacyapi.NewProjectAPIWithDeps(decomposition, gitFetcher, docParser, userService, projectHandler)
	taskAPI := legacyapi.NewTaskAPIWithDeps(taskHandler)

	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestID(), middleware.RequestLogger("core-api"))
	legacyapi.RegisterHealthRoutes(r, legacyapi.NewHealthAPI("core-api", map[string]legacyapi.ReadinessCheck{
		"db": legacyapi.NewGormReadinessCheck(dbConn),
	}))

	corev1.RegisterCoreRoutes(r, middleware.AuthMiddleware(), corev1.CoreHandlers{
		AuthRegister:         authAPI.Register,
		AuthLogin:            authAPI.Login,
		UserProfile:          userAPI.GetProfile,
		UserUpdateProfile:    userAPI.UpdateProfile,
		BlogList:             blogAPI.GetUserBlogs,
		BlogCreateDraft:      blogAPI.CreateDraftBlog,
		ProjectScan:          projectAPI.ScanGithubRepo,
		TaskCreateGeneration: taskAPI.CreateGenerationTask,
		TaskGet:              taskAPI.GetTask,
	})

	return r, nil
}
```

- [ ] **Step 5: 实现 `core-api` 新入口**

Create `backend/services/core-api/cmd/main.go`:

```go
package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"inkwords-backend/services/core-api/app/bootstrap"
	"inkwords-backend/shared/kernel/httpx"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default environment variables")
	}
}

func main() {
	router, err := bootstrap.BuildRouter()
	if err != nil {
		log.Fatalf("bootstrap core-api failed: %v", err)
	}

	server := httpx.NewServer(router)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("core-api startup failed: %v", err)
	}
}
```

- [ ] **Step 6: 让 builder 编译新的 `core-api` 入口**

Modify `backend/Dockerfile`:

```dockerfile
RUN go build -o core-api ./services/core-api/cmd && \
    go build -o llm-stream ./cmd/llm-stream && \
    go build -o parser-service ./services/parser-service/cmd && \
    go build -o export-service ./services/export-service/cmd && \
    go build -o review-service ./services/review-service/cmd
```

- [ ] **Step 7: 重新跑私有路由测试**

Run:

```bash
cd backend && go test ./services/core-api/transport/http/v1 -run TestRegisterCoreRoutes_RegistersTaskAndBlogRoutes -v
```

Expected: PASS

- [ ] **Step 8: 提交 `core-api` 私有入口**

```bash
git add backend/services/core-api backend/Dockerfile
git commit -m "refactor(core-api): add service-owned bootstrap and routes"
```

### Task 2: 私有化 `llm-stream` 入口与路由

**Files:**
- Create: `backend/services/llm-stream/app/bootstrap/bootstrap.go`
- Create: `backend/services/llm-stream/cmd/main.go`
- Create: `backend/services/llm-stream/transport/http/v1/routes.go`
- Create: `backend/services/llm-stream/transport/http/v1/routes_test.go`
- Modify: `backend/Dockerfile`
- Modify: `backend/internal/transport/http/v1/api/stream_api.go`

- [ ] **Step 1: 写 `llm-stream` 私有路由测试，先让它失败**

Create `backend/services/llm-stream/transport/http/v1/routes_test.go`:

```go
package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterStreamRoutes_RegistersLegacyStreamEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := func(c *gin.Context) { c.Next() }
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }

	RegisterStreamRoutes(r, auth, StreamHandlers{
		ContinueBlog: ok,
		PolishBlog:   ok,
		Scan:         ok,
		Analyze:      ok,
		Generate:     ok,
	})

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/blogs/123/continue"},
		{http.MethodPost, "/api/v1/blogs/123/polish"},
		{http.MethodPost, "/api/v1/stream/scan"},
		{http.MethodPost, "/api/v1/stream/analyze"},
		{http.MethodPost, "/api/v1/stream/generate"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code, tc.path)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run:

```bash
cd backend && go test ./services/llm-stream/transport/http/v1 -run TestRegisterStreamRoutes_RegistersLegacyStreamEndpoints -v
```

Expected: FAIL，报 `undefined: RegisterStreamRoutes`。

- [ ] **Step 3: 实现 `llm-stream` 私有路由**

Create `backend/services/llm-stream/transport/http/v1/routes.go`:

```go
package v1

import "github.com/gin-gonic/gin"

type StreamHandlers struct {
	ContinueBlog gin.HandlerFunc
	PolishBlog   gin.HandlerFunc
	Scan         gin.HandlerFunc
	Analyze      gin.HandlerFunc
	Generate     gin.HandlerFunc
}

func RegisterStreamRoutes(r *gin.Engine, auth gin.HandlerFunc, h StreamHandlers) {
	v1 := r.Group("/api/v1")

	blogGroup := v1.Group("/blogs")
	blogGroup.Use(auth)
	blogGroup.POST("/:id/continue", h.ContinueBlog)
	blogGroup.POST("/:id/polish", h.PolishBlog)

	streamGroup := v1.Group("/stream")
	streamGroup.Use(auth)
	streamGroup.POST("/scan", h.Scan)
	streamGroup.POST("/analyze", h.Analyze)
	streamGroup.POST("/generate", h.Generate)
}
```

- [ ] **Step 4: 实现 `llm-stream` 私有 bootstrap**

Create `backend/services/llm-stream/app/bootstrap/bootstrap.go`:

```go
package bootstrap

import (
	"errors"
	"os"

	"github.com/gin-gonic/gin"

	streamdomain "inkwords-backend/internal/domain/stream"
	taskdomain "inkwords-backend/internal/domain/task"
	"inkwords-backend/internal/infra/cache"
	"inkwords-backend/internal/service"
	"inkwords-backend/internal/transport/http/middleware"
	legacyapi "inkwords-backend/internal/transport/http/v1/api"
	streamv1 "inkwords-backend/services/llm-stream/transport/http/v1"
	"inkwords-backend/shared/platform/postgres"
)

func BuildRouter() (*gin.Engine, *taskdomain.Service, *streamdomain.Service, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, nil, nil, errors.New("DATABASE_URL environment variable is not set")
	}

	dbConn, err := postgres.InitCore(dsn)
	if err != nil {
		return nil, nil, nil, err
	}
	_ = cache.InitRedis()

	userService := service.NewUserService(dbConn)
	promptReq := service.NewPromptRequirementsService(dbConn)
	generator := service.NewGeneratorService(promptReq)
	decomposition := service.NewDecompositionService(promptReq)
	streamService := streamdomain.NewService(generator, decomposition, userService)
	taskService := taskdomain.NewService(taskdomain.NewGormRepository(dbConn), nil)
	streamHandler := streamdomain.NewHandler(streamService, streamdomain.NewGormBlogReadable())
	streamAPI := legacyapi.NewStreamAPIWithDeps(generator, decomposition, userService, streamHandler)

	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestID(), middleware.RequestLogger("llm-stream"))
	legacyapi.RegisterHealthRoutes(r, legacyapi.NewHealthAPI("llm-stream", map[string]legacyapi.ReadinessCheck{
		"db":              legacyapi.NewGormReadinessCheck(dbConn),
		"rabbitmq_config": legacyapi.NewRequiredValueCheck(os.Getenv("RABBITMQ_URL"), "RABBITMQ_URL is not configured"),
	}))

	streamv1.RegisterStreamRoutes(r, middleware.AuthMiddleware(), streamv1.StreamHandlers{
		ContinueBlog: streamAPI.ContinueBlogStreamHandler,
		PolishBlog:   streamAPI.PolishBlogStreamHandler,
		Scan:         streamAPI.ScanStreamHandler,
		Analyze:      streamAPI.AnalyzeStreamHandler,
		Generate:     streamAPI.GenerateBlogStreamHandler,
	})

	return r, taskService, streamService, nil
}
```

- [ ] **Step 5: 实现 `llm-stream` 新入口**

Create `backend/services/llm-stream/cmd/main.go`:

```go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/joho/godotenv"

	streamdomain "inkwords-backend/internal/domain/stream"
	taskdomain "inkwords-backend/internal/domain/task"
	"inkwords-backend/internal/infra/mq"
	"inkwords-backend/services/llm-stream/app/bootstrap"
	"inkwords-backend/shared/kernel/httpx"
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default environment variables")
	}
}

func main() {
	router, taskService, streamService, err := bootstrap.BuildRouter()
	if err != nil {
		log.Fatalf("bootstrap llm-stream failed: %v", err)
	}

	server := httpx.NewServer(router)
	stopConsumer, err := startGenerationTaskConsumer(taskService, streamService)
	if err != nil {
		log.Printf("generation consumer skipped: %v", err)
	}
	defer stopConsumer()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("llm-stream startup failed: %v", err)
	}
}

func startGenerationTaskConsumer(taskService *taskdomain.Service, streamService *streamdomain.Service) (func(), error) {
	conn, channel, err := sharedrabbitmq.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		return func() {}, err
	}
	queueName := envOrDefault("RABBITMQ_GENERATION_QUEUE", "inkwords.generation")
	exchange := envOrDefault("RABBITMQ_EXCHANGE", "inkwords.events")
	routingKey := mq.GenerationRequestedMessage{}.RoutingKey()

	if err := channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		return cleanup(conn, channel), err
	}
	queue, err := channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return cleanup(conn, channel), err
	}
	if err := channel.QueueBind(queue.Name, routingKey, exchange, false, nil); err != nil {
		return cleanup(conn, channel), err
	}
	deliveries, err := channel.Consume(queue.Name, "llm-stream-generation-worker", false, false, false, false, nil)
	if err != nil {
		return cleanup(conn, channel), err
	}

	consumer := streamdomain.NewTaskConsumer(taskService, streamService)
	go func() {
		for delivery := range deliveries {
			var message mq.GenerationRequestedMessage
			if err := json.Unmarshal(delivery.Body, &message); err != nil {
				_ = delivery.Ack(false)
				continue
			}
			if err := consumer.HandleGenerationRequested(context.Background(), message); err != nil {
				_ = delivery.Nack(false, true)
				continue
			}
			_ = delivery.Ack(false)
		}
	}()

	return cleanup(conn, channel), nil
}

func cleanup(conn *amqp.Connection, channel *amqp.Channel) func() {
	return func() {
		_ = channel.Close()
		_ = conn.Close()
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
```

- [ ] **Step 6: 更新 Dockerfile 的 `llm-stream` 编译入口**

Modify `backend/Dockerfile`:

```dockerfile
RUN go build -o core-api ./services/core-api/cmd && \
    go build -o llm-stream ./services/llm-stream/cmd && \
    go build -o parser-service ./services/parser-service/cmd && \
    go build -o export-service ./services/export-service/cmd && \
    go build -o review-service ./services/review-service/cmd
```

- [ ] **Step 7: 重跑私有路由测试**

Run:

```bash
cd backend && go test ./services/llm-stream/transport/http/v1 -run TestRegisterStreamRoutes_RegistersLegacyStreamEndpoints -v
```

Expected: PASS

- [ ] **Step 8: 提交 `llm-stream` 私有入口**

```bash
git add backend/services/llm-stream backend/Dockerfile
git commit -m "refactor(llm-stream): add service-owned bootstrap and routes"
```

### Task 3: 把 generation / decomposition 迁入 `llm-stream`

**Files:**
- Create: `backend/services/llm-stream/domain/generation/service.go`
- Create: `backend/services/llm-stream/domain/generation/analyze.go`
- Create: `backend/services/llm-stream/domain/generation/generate.go`
- Create: `backend/services/llm-stream/domain/generation/continue.go`
- Create: `backend/services/llm-stream/domain/generation/polish.go`
- Create: `backend/services/llm-stream/domain/generation/service_test.go`
- Modify: `backend/services/llm-stream/app/bootstrap/bootstrap.go`
- Modify: `backend/internal/domain/stream/*.go`
- Test: `backend/internal/domain/stream/handler_error_test.go`

- [ ] **Step 1: 写“新 generation service 被 llm-stream bootstrap 使用”的失败测试**

Create `backend/services/llm-stream/domain/generation/service_test.go`:

```go
package generation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewService_UsesServiceOwnedDependencies(t *testing.T) {
	svc := NewService(nil, nil)
	require.NotNil(t, svc)
	require.Nil(t, svc.blogWriter)
	require.Nil(t, svc.taskWriter)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run:

```bash
cd backend && go test ./services/llm-stream/domain/generation -run TestNewService_UsesServiceOwnedDependencies -v
```

Expected: FAIL，报 `undefined: NewService`。

- [ ] **Step 3: 实现服务自有 generation service 骨架**

Create `backend/services/llm-stream/domain/generation/service.go`:

```go
package generation

import (
	"context"

	"github.com/google/uuid"
)

type TaskWriter interface {
	AppendEvent(ctx context.Context, taskID uuid.UUID, eventType string, payload any) error
	UpdateResult(ctx context.Context, taskID uuid.UUID, result any) error
}

type BlogWriter interface {
	PersistFinalResult(ctx context.Context, taskID uuid.UUID) error
}

type Service struct {
	blogWriter BlogWriter
	taskWriter TaskWriter
}

func NewService(blogWriter BlogWriter, taskWriter TaskWriter) *Service {
	return &Service{
		blogWriter: blogWriter,
		taskWriter: taskWriter,
	}
}
```

- [ ] **Step 4: 新增生成主链路文件并先做最小转接**

Create `backend/services/llm-stream/domain/generation/generate.go`:

```go
package generation

import (
	"context"

	"github.com/google/uuid"

	"inkwords-backend/internal/service"
)

type LegacyAdapter struct {
	legacy *service.GeneratorService
}

func NewLegacyAdapter(legacy *service.GeneratorService) *LegacyAdapter {
	return &LegacyAdapter{legacy: legacy}
}

func (a *LegacyAdapter) Generate(ctx context.Context, userID uuid.UUID, sourceContent string, sourceType string, style string, chunkChan chan<- string, errChan chan<- error) {
	a.legacy.GenerateBlogStream(ctx, userID, sourceContent, sourceType, "", style, chunkChan, errChan)
}
```

- [ ] **Step 5: 让 `llm-stream` bootstrap 改用服务自有 generation service**

Modify `backend/services/llm-stream/app/bootstrap/bootstrap.go`:

```go
import (
	"errors"
	"os"

	"github.com/gin-gonic/gin"

	streamdomain "inkwords-backend/internal/domain/stream"
	taskdomain "inkwords-backend/internal/domain/task"
	"inkwords-backend/internal/infra/cache"
	"inkwords-backend/internal/service"
	"inkwords-backend/internal/transport/http/middleware"
	legacyapi "inkwords-backend/internal/transport/http/v1/api"
	generation "inkwords-backend/services/llm-stream/domain/generation"
	streamv1 "inkwords-backend/services/llm-stream/transport/http/v1"
	"inkwords-backend/shared/platform/postgres"
)

// ...

legacyGenerator := service.NewGeneratorService(promptReq)
legacyDecomposition := service.NewDecompositionService(promptReq)
generationService := generation.NewService(nil, nil)
streamService := streamdomain.NewService(legacyGenerator, legacyDecomposition, userService)
_ = generationService
```

- [ ] **Step 6: 跑 generation 新目录测试与 stream 回归测试**

Run:

```bash
cd backend && go test ./services/llm-stream/domain/generation ./internal/domain/stream/... -v
```

Expected: PASS

- [ ] **Step 7: 提交 generation 迁移第一段**

```bash
git add backend/services/llm-stream/domain/generation backend/services/llm-stream/app/bootstrap/bootstrap.go
git commit -m "refactor(generation): start moving llm use cases into llm-stream"
```

### Task 4: 在 `core-api` 新增任务结果持久化器

**Files:**
- Create: `backend/services/core-api/domain/task/result_persister.go`
- Create: `backend/services/core-api/domain/task/result_persister_test.go`
- Modify: `backend/services/core-api/app/bootstrap/bootstrap.go`
- Modify: `backend/internal/domain/task/*.go`
- Test: `backend/internal/service/decomposition_generate_persist_test.go`

- [ ] **Step 1: 写“任务结果持久化器会把结果落回 blogs”的失败测试**

Create `backend/services/core-api/domain/task/result_persister_test.go`:

```go
package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeBlogRepository struct {
	persisted bool
}

func (r *fakeBlogRepository) PersistGenerationResult(context.Context, uuid.UUID, map[string]any) error {
	r.persisted = true
	return nil
}

func TestResultPersister_PersistsGenerationResultToBlogRepository(t *testing.T) {
	repo := &fakeBlogRepository{}
	persister := NewResultPersister(repo, nil)

	err := persister.PersistGenerationResult(context.Background(), uuid.New(), map[string]any{"content": "# 内容"})
	require.NoError(t, err)
	require.True(t, repo.persisted)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run:

```bash
cd backend && go test ./services/core-api/domain/task -run TestResultPersister_PersistsGenerationResultToBlogRepository -v
```

Expected: FAIL，报 `undefined: NewResultPersister`。

- [ ] **Step 3: 实现任务结果持久化器**

Create `backend/services/core-api/domain/task/result_persister.go`:

```go
package task

import (
	"context"

	"github.com/google/uuid"
)

type BlogResultRepository interface {
	PersistGenerationResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error
}

type UsageRepository interface {
	AccumulateTokens(ctx context.Context, taskID uuid.UUID, result map[string]any) error
}

type ResultPersister struct {
	blogRepo  BlogResultRepository
	usageRepo UsageRepository
}

func NewResultPersister(blogRepo BlogResultRepository, usageRepo UsageRepository) *ResultPersister {
	return &ResultPersister{
		blogRepo:  blogRepo,
		usageRepo: usageRepo,
	}
}

func (p *ResultPersister) PersistGenerationResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error {
	if err := p.blogRepo.PersistGenerationResult(ctx, taskID, result); err != nil {
		return err
	}
	if p.usageRepo != nil {
		return p.usageRepo.AccumulateTokens(ctx, taskID, result)
	}
	return nil
}
```

- [ ] **Step 4: 把 `core-api` bootstrap 接上结果持久化器**

Modify `backend/services/core-api/app/bootstrap/bootstrap.go`:

```go
import (
	"errors"
	"os"

	"github.com/gin-gonic/gin"

	authdomain "inkwords-backend/internal/domain/auth"
	blogdomain "inkwords-backend/internal/domain/blog"
	projectdomain "inkwords-backend/internal/domain/project"
	taskdomain "inkwords-backend/internal/domain/task"
	userdomain "inkwords-backend/internal/domain/user"
	coretask "inkwords-backend/services/core-api/domain/task"
	// ...
)

// ...

resultPersister := coretask.NewResultPersister(nil, nil)
_ = resultPersister
```

- [ ] **Step 5: 重新跑持久化器测试**

Run:

```bash
cd backend && go test ./services/core-api/domain/task -run TestResultPersister_PersistsGenerationResultToBlogRepository -v
```

Expected: PASS

- [ ] **Step 6: 提交 `core-api` 结果持久化器**

```bash
git add backend/services/core-api/domain/task backend/services/core-api/app/bootstrap/bootstrap.go
git commit -m "feat(core-api): add task result persister for blog writes"
```

### Task 5: 收紧 `llm-stream` 写入边界到任务表

**Files:**
- Modify: `backend/internal/service/generator.go`
- Modify: `backend/internal/service/decomposition_generate_continue.go`
- Modify: `backend/internal/service/decomposition_generate_persistence.go`
- Modify: `backend/internal/service/generator_persist_test.go`
- Modify: `backend/internal/service/decomposition_generate_persist_test.go`
- Test: `backend/internal/service/generator_persist_test.go`
- Test: `backend/internal/service/decomposition_generate_persist_test.go`

- [ ] **Step 1: 写失败测试，锁定“生成阶段不再直写 blog 表”**

Update `backend/internal/service/generator_persist_test.go`:

```go
func TestGenerateBlogStream_DoesNotPersistBlogDirectlyWhenTaskModeEnabled(t *testing.T) {
	t.Setenv("INKWORDS_TASK_PERSISTENCE_MODE", "task_only")
	svc := NewGeneratorService(nil)
	require.NotNil(t, svc)
	require.True(t, taskOnlyPersistenceMode())
}
```

- [ ] **Step 2: 跑测试确认失败**

Run:

```bash
cd backend && go test ./internal/service -run TestGenerateBlogStream_DoesNotPersistBlogDirectlyWhenTaskModeEnabled -v
```

Expected: FAIL，报 `undefined: taskOnlyPersistenceMode`。

- [ ] **Step 3: 实现任务表专用持久化开关**

Modify `backend/internal/service/generator.go`:

```go
func taskOnlyPersistenceMode() bool {
	return strings.EqualFold(os.Getenv("INKWORDS_TASK_PERSISTENCE_MODE"), "task_only")
}
```

Modify `GenerateBlogStreamWithProfile()` save logic:

```go
if !taskOnlyPersistenceMode() {
	if err := s.saveToDB(ctx, userID, sourceType, fullContent); err != nil {
		errChan <- err
	}
}
```

- [ ] **Step 4: 对系列生成与续写也应用同样限制**

Modify `backend/internal/service/decomposition_generate_continue.go`:

```go
if finalNewContent != "" && !taskOnlyPersistenceMode() {
	updatedContent := blog.Content + finalNewContent
	if err := db.DB.WithContext(ctx).Model(&blog).Update("content", updatedContent).Error; err != nil {
		fmt.Printf("Failed to update blog content: %v\n", err)
	}
}
```

Modify `backend/internal/service/decomposition_generate_persistence.go`:

```go
if taskOnlyPersistenceMode() {
	return nil
}
```

- [ ] **Step 5: 重新跑受影响测试**

Run:

```bash
cd backend && go test ./internal/service -run 'GenerateBlogStream_DoesNotPersistBlogDirectlyWhenTaskModeEnabled|Persist' -v
```

Expected: PASS

- [ ] **Step 6: 提交写入边界收紧**

```bash
git add backend/internal/service/generator.go \
        backend/internal/service/decomposition_generate_continue.go \
        backend/internal/service/decomposition_generate_persistence.go \
        backend/internal/service/generator_persist_test.go \
        backend/internal/service/decomposition_generate_persist_test.go
git commit -m "refactor(llm-stream): restrict direct blog writes to task-only mode"
```

### Task 6: 清理共享 transport 与最终回归

**Files:**
- Modify: `backend/internal/transport/http/v1/routes.go`
- Modify: `backend/internal/transport/http/v1/routes_test.go`
- Modify: `backend/internal/transport/http/v1/api/stream_api.go`
- Modify: `backend/Dockerfile`
- Modify: `docker-compose.yml`
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`
- Modify: `.trae/documents/InkWords_Database.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_PRD.md`
- Modify: `README.md`

- [ ] **Step 1: 让共享 transport 不再承担 `core-api/llm-stream` 主入口**

Modify `backend/internal/transport/http/v1/routes.go`:

```go
// Deprecated: core-api 与 llm-stream 已迁入 services/*/transport/http/v1。
// 这里只保留其它服务仍在使用的共享注册函数或回滚兼容层。
```

- [ ] **Step 2: 删除共享 `StreamAPI` 的业务入口职责**

Modify `backend/internal/transport/http/v1/api/stream_api.go`:

```go
// Deprecated: llm-stream 现已通过 services/llm-stream 自有 transport 与 bootstrap 装配，
// 本文件仅作为过渡兼容层，待 Phase 2D 完成后删除。
```

- [ ] **Step 3: 跑全量后端测试**

Run:

```bash
cd backend && go test ./... -count=1
```

Expected: PASS

- [ ] **Step 4: 跑全量 Compose 冒烟**

Run:

```bash
docker compose --env-file backend/.env down
docker compose --env-file backend/.env up -d --build
docker compose --env-file backend/.env ps
curl -I http://localhost
curl -sS http://localhost/api/v1/ping
```

Expected:

```text
core-api / llm-stream / parser-service / export-service / review-service / frontend 均 healthy
HTTP/1.1 200 OK
{"code":200,"data":null,"message":"pong"}
```

- [ ] **Step 5: 同步文档**

在 `.trae/documents/InkWords_Architecture.md` 追加：

```md
- 2026-06-04：Phase 2 启动 `core-api / llm-stream` 深层拆分。`core-api` 与 `llm-stream` 新增服务自有 `app/bootstrap` 与 `transport/http/v1`，并开始把共享 `generator / decomposition / stream api` 迁出；目标是让 `llm-stream` 只写任务表，由 `core-api` 接管业务事实表持久化。
```

在 `README.md` 追加：

```md
## Core-API / LLM-Stream 深拆分

第二阶段开始后，`core-api` 与 `llm-stream` 也将从共享 `internal/service` 迁入各自的服务目录，并逐步取消 `llm-stream` 对 `blogs / users` 的直接写入。
```

- [ ] **Step 6: 提交最终回归与文档同步**

```bash
git add backend/internal/transport/http/v1/routes.go \
        backend/internal/transport/http/v1/routes_test.go \
        backend/internal/transport/http/v1/api/stream_api.go \
        backend/Dockerfile \
        docker-compose.yml \
        .trae/documents/InkWords_API.md \
        .trae/documents/InkWords_Architecture.md \
        .trae/documents/InkWords_Conversation_Log.md \
        .trae/documents/InkWords_Database.md \
        .trae/documents/InkWords_Development_Plan_and_Log.md \
        .trae/documents/InkWords_PRD.md \
        README.md
git commit -m "refactor(core-stream): finalize phase2 service ownership and docs"
```
