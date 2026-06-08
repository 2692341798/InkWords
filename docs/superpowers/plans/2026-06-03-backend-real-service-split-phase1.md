# Backend Real Service Split Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `review-service`、`parser-service`、`export-service` 从共享 `backend/internal/*` 结构迁移到 `backend/services/*` 真实服务目录，并保持现有 API、Compose 启动方式和运行行为不变。

**Architecture:** 先建立 `backend/services/<service>/cmd|app|domain|infra|transport` 骨架，再把三个边界最清晰的服务逐个迁移到新目录。迁移期间允许旧目录与新目录短暂并存，但任何新增业务代码必须优先落在新服务目录，`shared/` 仅承载稳定基础能力。

**Tech Stack:** Go 1.25+, Gin, GORM, RabbitMQ, Docker Compose, PostgreSQL, Obsidian Local REST API

---

## 文件结构锁定

### 本次新增目录

- `backend/services/review-service/cmd/`
- `backend/services/review-service/app/bootstrap/`
- `backend/services/review-service/domain/review/`
- `backend/services/review-service/infra/db/`
- `backend/services/review-service/infra/wiki/`
- `backend/services/review-service/transport/http/middleware/`
- `backend/services/review-service/transport/http/v1/`
- `backend/services/parser-service/cmd/`
- `backend/services/parser-service/app/bootstrap/`
- `backend/services/parser-service/domain/parse/`
- `backend/services/parser-service/infra/db/`
- `backend/services/parser-service/infra/mq/`
- `backend/services/parser-service/infra/parser/`
- `backend/services/parser-service/transport/http/middleware/`
- `backend/services/parser-service/transport/http/v1/`
- `backend/services/export-service/cmd/`
- `backend/services/export-service/app/bootstrap/`
- `backend/services/export-service/domain/export/`
- `backend/services/export-service/infra/db/`
- `backend/services/export-service/infra/mq/`
- `backend/services/export-service/infra/obsidian/`
- `backend/services/export-service/infra/artifact/`
- `backend/services/export-service/transport/http/middleware/`
- `backend/services/export-service/transport/http/v1/`
- `backend/shared/kernel/auth/`
- `backend/shared/kernel/httpx/`
- `backend/shared/kernel/response/`
- `backend/shared/platform/postgres/`
- `backend/shared/platform/rabbitmq/`

### 本次重点修改文件

- `backend/go.mod`
- `backend/Dockerfile`
- `docker-compose.yml`
- `backend/cmd/review-service/main.go`
- `backend/cmd/parser-service/main.go`
- `backend/cmd/export-service/main.go`
- `.trae/documents/InkWords_Architecture.md`
- `.trae/documents/InkWords_Development_Plan_and_Log.md`
- `.trae/documents/InkWords_Conversation_Log.md`
- `README.md`

### 迁移原则

- 保持 `command: ["./review-service"]`、`["./parser-service"]`、`["./export-service"]` 不变，只调整二进制来源。
- 保持 `/api/v1/review/*`、`/api/v1/project/parse`、`/api/v1/blogs/:id/export*` 路由不变。
- 先复制稳定基础件到 `shared/`，再让三个新服务引用 `shared/`，不要直接让三个新服务继续耦合旧 `internal/*`。

### Task 1: 建立真实服务骨架与共享底座

**Files:**
- Create: `backend/services/review-service/cmd/main.go`
- Create: `backend/services/review-service/app/bootstrap/bootstrap.go`
- Create: `backend/services/parser-service/cmd/main.go`
- Create: `backend/services/parser-service/app/bootstrap/bootstrap.go`
- Create: `backend/services/export-service/cmd/main.go`
- Create: `backend/services/export-service/app/bootstrap/bootstrap.go`
- Create: `backend/shared/kernel/httpx/server.go`
- Create: `backend/shared/platform/postgres/core.go`
- Create: `backend/shared/platform/rabbitmq/connection.go`
- Modify: `backend/Dockerfile`

- [ ] **Step 1: 先为 Phase 1 建骨架目录**

Run:

```bash
mkdir -p \
  backend/services/review-service/{cmd,app/bootstrap,domain/review,infra/db,infra/wiki,transport/http/middleware,transport/http/v1} \
  backend/services/parser-service/{cmd,app/bootstrap,domain/parse,infra/db,infra/mq,infra/parser,transport/http/middleware,transport/http/v1} \
  backend/services/export-service/{cmd,app/bootstrap,domain/export,infra/db,infra/mq,infra/obsidian,infra/artifact,transport/http/middleware,transport/http/v1} \
  backend/shared/kernel/{auth,httpx,response} \
  backend/shared/platform/{postgres,rabbitmq}
```

Expected: 所有目录创建成功，无报错。

- [ ] **Step 2: 写共享 HTTP Server 底座**

Create `backend/shared/kernel/httpx/server.go`:

```go
package httpx

import (
	"context"
	"net/http"
	"time"
)

type ShutdownableServer interface {
	Shutdown(context.Context) error
}

func NewServer(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       60 * time.Second,
	}
}

func ShutdownOnContextDone(signalContext context.Context, server ShutdownableServer, timeout time.Duration) error {
	<-signalContext.Done()

	shutdownContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return server.Shutdown(shutdownContext)
}
```

- [ ] **Step 3: 写共享 PostgreSQL 与 RabbitMQ 连接工厂**

Create `backend/shared/platform/postgres/core.go`:

```go
package postgres

import (
	"fmt"

	"gorm.io/gorm"

	"inkwords-backend/internal/infra/db"
)

func InitCore(dsn string) (*gorm.DB, error) {
	if err := db.InitCoreDB(dsn); err != nil {
		return nil, fmt.Errorf("init core db: %w", err)
	}
	return db.DB, nil
}

func InitReview(dsn string) (*gorm.DB, error) {
	if err := db.InitReviewDB(dsn); err != nil {
		return nil, fmt.Errorf("init review db: %w", err)
	}
	return db.DB, nil
}
```

Create `backend/shared/platform/rabbitmq/connection.go`:

```go
package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func Dial(url string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}

	return conn, channel, nil
}
```

- [ ] **Step 4: 让 Dockerfile 开始编译新目录入口**

Modify `backend/Dockerfile` build stage:

```dockerfile
RUN go build -o core-api ./cmd/core-api && \
    go build -o llm-stream ./cmd/llm-stream && \
    go build -o parser-service ./services/parser-service/cmd && \
    go build -o export-service ./services/export-service/cmd && \
    go build -o review-service ./services/review-service/cmd
```

Expected: 新入口二进制开始由 `backend/services/*/cmd` 产出。

- [ ] **Step 5: 先跑一次最小编译检查**

Run:

```bash
cd backend && go test ./... >/tmp/backend-phase1-task1.log && tail -n 20 /tmp/backend-phase1-task1.log
```

Expected: 如果失败，失败点应只来自“新建骨架未接线完成”，不允许出现 import 循环。

- [ ] **Step 6: 提交骨架**

```bash
git add backend/services backend/shared backend/Dockerfile
git commit -m "refactor(backend): scaffold phase1 service-owned directories"
```

### Task 2: 迁移 `review-service` 到新目录

**Files:**
- Create: `backend/services/review-service/domain/review/service.go`
- Create: `backend/services/review-service/domain/review/handler.go`
- Create: `backend/services/review-service/infra/wiki/note_source.go`
- Create: `backend/services/review-service/transport/http/v1/routes.go`
- Create: `backend/services/review-service/app/bootstrap/bootstrap.go`
- Create: `backend/services/review-service/cmd/main.go`
- Test: `backend/internal/domain/review/*.go`
- Modify: `docker-compose.yml`

- [ ] **Step 1: 先复制 review 领域代码到服务目录**

Run:

```bash
cp backend/internal/domain/review/*.go backend/services/review-service/domain/review/
```

Expected: `review` 领域文件全部出现在新目录。

- [ ] **Step 2: 抽出 review 的 Wiki 依赖到服务私有 infra**

Create `backend/services/review-service/infra/wiki/note_source.go`:

```go
package wiki

import (
	"context"
	"log"
	"strings"

	reviewdomain "inkwords-backend/services/review-service/domain/review"
	"inkwords-backend/internal/service"
)

type unavailableReviewNoteSource struct {
	err error
}

func (s unavailableReviewNoteSource) ListEligibleNotes(context.Context) ([]reviewdomain.ReviewNote, error) {
	return nil, s.err
}

func BuildNoteSource(rootDir string) reviewdomain.NoteSource {
	store, err := service.NewObsidianStoreFromEnv()
	if err != nil {
		log.Printf("Review note source initialization failed: %v", err)
		return unavailableReviewNoteSource{err: err}
	}

	if strings.TrimSpace(rootDir) == "" {
		rootDir = "wiki"
	}

	return reviewdomain.NewReviewNoteSource(store, rootDir)
}
```

- [ ] **Step 3: 写 review 服务私有路由注册**

Create `backend/services/review-service/transport/http/v1/routes.go`:

```go
package v1

import (
	"github.com/gin-gonic/gin"
	reviewdomain "inkwords-backend/services/review-service/domain/review"
)

func RegisterReviewRoutes(r *gin.Engine, authMiddleware gin.HandlerFunc, handler *reviewdomain.Handler) {
	v1 := r.Group("/api/v1")
	reviewGroup := v1.Group("/review")
	reviewGroup.Use(authMiddleware)
	reviewGroup.GET("/today", handler.GetTodayCard)
	reviewGroup.GET("/history", handler.GetHistory)
	reviewGroup.POST("/pick", handler.PickRandom)
	reviewGroup.GET("/notes", handler.ListNotes)
	reviewGroup.POST("/sessions", handler.CreateSession)
	reviewGroup.GET("/sessions/:id", handler.GetSession)
	reviewGroup.POST("/sessions/:id/respond", handler.Respond)
	reviewGroup.POST("/sessions/:id/hint", handler.RequestHint)
	reviewGroup.POST("/sessions/:id/finish", handler.Finish)
}
```

- [ ] **Step 4: 用 bootstrap 收口 review-service 装配**

Create `backend/services/review-service/app/bootstrap/bootstrap.go`:

```go
package bootstrap

import (
	"os"

	"github.com/gin-gonic/gin"

	reviewdomain "inkwords-backend/services/review-service/domain/review"
	"inkwords-backend/services/review-service/infra/wiki"
	reviewroutes "inkwords-backend/services/review-service/transport/http/v1"
	"inkwords-backend/internal/transport/http/middleware"
	transportv1api "inkwords-backend/internal/transport/http/v1/api"
	"inkwords-backend/shared/platform/postgres"
)

func BuildRouter() (*gin.Engine, error) {
	dbConn, err := postgres.InitReview(os.Getenv("REVIEW_DATABASE_URL"))
	if err != nil {
		return nil, err
	}

	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestID(), middleware.RequestLogger("review-service"))
	transportv1api.RegisterHealthRoutes(r, transportv1api.NewHealthAPI("review-service", map[string]transportv1api.ReadinessCheck{
		"db": transportv1api.NewGormReadinessCheck(dbConn),
	}))

	reviewRepo := reviewdomain.NewGormRepository(dbConn)
	reviewService := reviewdomain.NewService(reviewRepo, wiki.BuildNoteSource(os.Getenv("OBSIDIAN_WIKI_DIR")))
	reviewHandler := reviewdomain.NewHandler(reviewService)
	reviewroutes.RegisterReviewRoutes(r, middleware.AuthMiddleware(), reviewHandler)

	return r, nil
}
```

- [ ] **Step 5: 改用新的 `cmd/main.go`**

Create `backend/services/review-service/cmd/main.go`:

```go
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"inkwords-backend/services/review-service/app/bootstrap"
	"inkwords-backend/shared/kernel/httpx"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default environment variables")
	}
}

func main() {
	r, err := bootstrap.BuildRouter()
	if err != nil {
		log.Fatalf("bootstrap review-service failed: %v", err)
	}

	server := httpx.NewServer(r)
	signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		if err := httpx.ShutdownOnContextDone(signalContext, server, 15*time.Second); err != nil {
			log.Printf("Server shutdown failed: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server startup failed: %v", err)
	}
}
```

- [ ] **Step 6: 跑 review 相关测试**

Run:

```bash
cd backend && go test ./internal/domain/review/... -v
```

Expected: review 既有单测继续通过。

- [ ] **Step 7: 用 Compose 只验证 review-service**

Run:

```bash
docker compose --env-file backend/.env up -d --build review-service
docker compose ps review-service
```

Expected: `review-service` 为 `Up` 或 `healthy`。

- [ ] **Step 8: 提交 review-service 迁移**

```bash
git add backend/services/review-service backend/Dockerfile
git commit -m "refactor(review): move review-service into service-owned structure"
```

### Task 3: 迁移 `parser-service` 到新目录

**Files:**
- Create: `backend/services/parser-service/domain/parse/service.go`
- Create: `backend/services/parser-service/domain/parse/handler.go`
- Create: `backend/services/parser-service/domain/parse/task_consumer.go`
- Create: `backend/services/parser-service/infra/parser/doc_parser.go`
- Create: `backend/services/parser-service/transport/http/v1/routes.go`
- Create: `backend/services/parser-service/app/bootstrap/bootstrap.go`
- Create: `backend/services/parser-service/cmd/main.go`
- Test: `backend/internal/domain/fileparse/*.go`

- [ ] **Step 1: 复制 parse 领域代码**

Run:

```bash
cp backend/internal/domain/fileparse/*.go backend/services/parser-service/domain/parse/
```

- [ ] **Step 2: 复制 parser infra，并让服务目录拥有解析实现**

Run:

```bash
cp backend/internal/infra/parser/*.go backend/services/parser-service/infra/parser/
```

Expected: `DocParser`、`ArchiveParser`、Git 相关解析能力都已落在新服务目录。

- [ ] **Step 3: 写 parser-service 私有路由**

Create `backend/services/parser-service/transport/http/v1/routes.go`:

```go
package v1

import (
	"github.com/gin-gonic/gin"
	parsedomain "inkwords-backend/services/parser-service/domain/parse"
)

func RegisterParserRoutes(r *gin.Engine, authMiddleware gin.HandlerFunc, handler *parsedomain.Handler) {
	v1 := r.Group("/api/v1")
	projectGroup := v1.Group("/project")
	projectGroup.Use(authMiddleware)
	projectGroup.POST("/parse", handler.Parse)
}
```

- [ ] **Step 4: 用 bootstrap 收口 parser-service 装配与消费端**

Create `backend/services/parser-service/app/bootstrap/bootstrap.go`:

```go
package bootstrap

import (
	"context"
	"os"

	"github.com/gin-gonic/gin"

	parsedomain "inkwords-backend/services/parser-service/domain/parse"
	parserinfra "inkwords-backend/services/parser-service/infra/parser"
	parserroutes "inkwords-backend/services/parser-service/transport/http/v1"
	taskdomain "inkwords-backend/internal/domain/task"
	"inkwords-backend/internal/service"
	"inkwords-backend/internal/transport/http/middleware"
	transportv1api "inkwords-backend/internal/transport/http/v1/api"
	"inkwords-backend/shared/platform/postgres"
)

func BuildRouter() (*gin.Engine, *parsedomain.Service, *taskdomain.Service, error) {
	dbConn, err := postgres.InitCore(os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, nil, nil, err
	}

	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestID(), middleware.RequestLogger("parser-service"))
	r.MaxMultipartMemory = 888 << 20
	transportv1api.RegisterHealthRoutes(r, transportv1api.NewHealthAPI("parser-service", map[string]transportv1api.ReadinessCheck{
		"db": transportv1api.NewGormReadinessCheck(dbConn),
	}))

	userService := service.NewUserService(dbConn)
	docParser := parserinfra.NewDocParser()
	archiveParser := parserinfra.NewArchiveParser(docParser)
	parseService := parsedomain.NewService(docParser, archiveParser)
	taskService := taskdomain.NewService(taskdomain.NewGormRepository(dbConn), nil)
	parseHandler := parsedomain.NewHandler(parseService, userService)
	parserroutes.RegisterParserRoutes(r, middleware.AuthMiddleware(), parseHandler)

	return r, parseService, taskService, nil
}
```

- [ ] **Step 5: 为 parser-service 补服务私有 consumer 文件**

Create `backend/services/parser-service/domain/parse/task_consumer.go`:

```go
package parse

import (
	"context"
	"encoding/json"
	"log"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"

	taskdomain "inkwords-backend/internal/domain/task"
	"inkwords-backend/internal/infra/mq"
	sharedmq "inkwords-backend/shared/platform/rabbitmq"
)

func StartParseConsumer(ctx context.Context, taskService *taskdomain.Service, parseService *Service) (func(), error) {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Println("RabbitMQ is not configured, parse consumer disabled")
		return func() {}, nil
	}

	conn, channel, err := sharedmq.Dial(rabbitURL)
	if err != nil {
		return func() {}, err
	}

	exchangeName := envOrDefault("RABBITMQ_EXCHANGE", "inkwords.events")
	queueName := envOrDefault("RABBITMQ_PARSE_QUEUE", "inkwords.parse")
	routingKey := mq.ParseRequestedMessage{}.RoutingKey()

	if err := channel.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	queue, err := channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	if err := channel.QueueBind(queue.Name, routingKey, exchangeName, false, nil); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	deliveries, err := channel.Consume(queue.Name, "parser-service-parse-worker", false, false, false, false, nil)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	consumer := NewTaskConsumer(taskService, parseService)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case delivery, ok := <-deliveries:
				if !ok {
					return
				}

				var message mq.ParseRequestedMessage
				if err := json.Unmarshal(delivery.Body, &message); err != nil {
					log.Printf("invalid parse message payload: %v", err)
					_ = delivery.Ack(false)
					continue
				}

				if err := consumer.HandleParseRequested(ctx, message); err != nil {
					log.Printf("parse task handling failed for %s: %v", message.TaskID, err)
					_ = delivery.Nack(false, true)
					continue
				}

				_ = delivery.Ack(false)
			}
		}
	}()

	return func() {
		_ = channel.Close()
		_ = conn.Close()
	}, nil
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
```

- [ ] **Step 6: 为 parser-service 写新入口**

Create `backend/services/parser-service/cmd/main.go`:

```go
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"inkwords-backend/services/parser-service/app/bootstrap"
	parsedomain "inkwords-backend/services/parser-service/domain/parse"
	"inkwords-backend/shared/kernel/httpx"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default environment variables")
	}
}

func main() {
	r, parseService, taskService, err := bootstrap.BuildRouter()
	if err != nil {
		log.Fatalf("bootstrap parser-service failed: %v", err)
	}

	server := httpx.NewServer(r)
	signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	stopConsumer, err := parsedomain.StartParseConsumer(signalContext, taskService, parseService)
	if err != nil {
		log.Printf("RabbitMQ parse consumer initialization skipped: %v", err)
	}
	defer stopConsumer()

	go func() {
		if err := httpx.ShutdownOnContextDone(signalContext, server, 15*time.Second); err != nil {
			log.Printf("Server shutdown failed: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server startup failed: %v", err)
	}
}
```

- [ ] **Step 7: 跑 parse 相关测试**

Run:

```bash
cd backend && go test ./internal/domain/fileparse/... ./internal/infra/parser/... -v
```

Expected: parse 领域和 parser infra 现有测试继续通过。

- [ ] **Step 8: 验证 parser-service 容器**

Run:

```bash
docker compose --env-file backend/.env up -d --build parser-service
docker compose ps parser-service
```

Expected: `parser-service` 为 `Up` 或 `healthy`。

- [ ] **Step 9: 提交 parser-service 迁移**

```bash
git add backend/services/parser-service backend/Dockerfile
git commit -m "refactor(parser): move parser-service into service-owned structure"
```

### Task 4: 迁移 `export-service` 到新目录

**Files:**
- Create: `backend/services/export-service/domain/export/service.go`
- Create: `backend/services/export-service/domain/export/consumer.go`
- Create: `backend/services/export-service/infra/artifact/store.go`
- Create: `backend/services/export-service/transport/http/v1/routes.go`
- Create: `backend/services/export-service/app/bootstrap/bootstrap.go`
- Create: `backend/services/export-service/cmd/main.go`
- Test: `backend/internal/domain/task/*.go`
- Test: `backend/internal/service/pdf_export_test.go`

- [ ] **Step 1: 复制 export 相关业务实现**

Run:

```bash
cp backend/internal/service/pdf_export*.go backend/services/export-service/domain/export/ || true
cp backend/internal/service/obsidian*.go backend/services/export-service/domain/export/ || true
cp backend/internal/domain/task/export_*.go backend/services/export-service/domain/export/ || true
```

Expected: PDF、Obsidian、导出产物相关实现进入服务私有目录。

- [ ] **Step 2: 把导出产物存储下沉到服务私有 infra**

Create `backend/services/export-service/infra/artifact/store.go`:

```go
package artifact

import (
	"time"

	taskdomain "inkwords-backend/internal/domain/task"
)

func NewStore(dir string) *taskdomain.ExportArtifactStore {
	return taskdomain.NewExportArtifactStore(dir, 15*time.Minute, time.Now)
}
```

- [ ] **Step 3: 写 export-service 私有路由**

Create `backend/services/export-service/transport/http/v1/routes.go`:

```go
package v1

import (
	"github.com/gin-gonic/gin"
	"inkwords-backend/internal/transport/http/v1/api"
)

func RegisterExportRoutes(r *gin.Engine, authMiddleware gin.HandlerFunc, blogAPI *api.BlogAPI) {
	v1 := r.Group("/api/v1")
	blogGroup := v1.Group("/blogs")
	blogGroup.Use(authMiddleware)
	blogGroup.GET("/:id/export", blogAPI.ExportSeries)
	blogGroup.GET("/:id/export/pdf", blogAPI.ExportSeriesPDF)
	blogGroup.POST("/:id/export/obsidian", blogAPI.ExportToObsidian)
	blogGroup.POST("/:id/export/obsidian/series", blogAPI.ExportSeriesToObsidian)
}
```

- [ ] **Step 4: 用 bootstrap 收口 export-service 装配**

Create `backend/services/export-service/app/bootstrap/bootstrap.go`:

```go
package bootstrap

import (
	"os"

	"github.com/gin-gonic/gin"

	blogdomain "inkwords-backend/internal/domain/blog"
	taskdomain "inkwords-backend/internal/domain/task"
	"inkwords-backend/internal/service"
	"inkwords-backend/internal/transport/http/middleware"
	"inkwords-backend/internal/transport/http/v1/api"
	artifact "inkwords-backend/services/export-service/infra/artifact"
	exportroutes "inkwords-backend/services/export-service/transport/http/v1"
	"inkwords-backend/shared/platform/postgres"
)

func BuildRouter() (*gin.Engine, *taskdomain.ExportConsumer, error) {
	dbConn, err := postgres.InitCore(os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, nil, err
	}

	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestID(), middleware.RequestLogger("export-service"))
	api.RegisterHealthRoutes(r, api.NewHealthAPI("export-service", map[string]api.ReadinessCheck{
		"db": api.NewGormReadinessCheck(dbConn),
	}))

	blogService := service.NewBlogService()
	blogRepo := blogdomain.NewGormRepository(dbConn)
	blogDomainService := blogdomain.NewService(blogRepo)
	blogDomainHandler := blogdomain.NewHandlerWithLegacy(blogDomainService, blogService)
	taskService := taskdomain.NewService(taskdomain.NewGormRepository(dbConn), nil)
	blogAPI := api.NewBlogAPIWithDeps(blogService, blogDomainHandler)
	exportroutes.RegisterExportRoutes(r, middleware.AuthMiddleware(), blogAPI)

	artifactStore := artifact.NewStore(envOrDefault("EXPORT_ARTIFACTS_DIR", "/app/export-artifacts"))
	consumer := taskdomain.NewExportConsumer(taskService, blogService, artifactStore)

	return r, consumer, nil
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
```

- [ ] **Step 5: 为 export-service 补服务私有 consumer 文件**

Create `backend/services/export-service/domain/export/consumer.go`:

```go
package export

import (
	"context"
	"encoding/json"
	"log"
	"os"

	taskdomain "inkwords-backend/internal/domain/task"
	"inkwords-backend/internal/infra/mq"
	sharedmq "inkwords-backend/shared/platform/rabbitmq"
)

func StartExportConsumer(ctx context.Context, consumer *taskdomain.ExportConsumer, queueName string) (func(), error) {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Println("RabbitMQ is not configured, export consumer disabled")
		return func() {}, nil
	}

	conn, channel, err := sharedmq.Dial(rabbitURL)
	if err != nil {
		return func() {}, err
	}

	exchangeName := envOrDefault("RABBITMQ_EXCHANGE", "inkwords.events")
	routingKey := mq.ExportRequestedMessage{}.RoutingKey()

	if err := channel.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	queue, err := channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	if err := channel.QueueBind(queue.Name, routingKey, exchangeName, false, nil); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	deliveries, err := channel.Consume(queue.Name, "export-service-pdf-worker", false, false, false, false, nil)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case delivery, ok := <-deliveries:
				if !ok {
					return
				}

				var message taskdomain.ExportRequestedMessage
				if err := json.Unmarshal(delivery.Body, &message); err != nil {
					log.Printf("invalid export message payload: %v", err)
					_ = delivery.Ack(false)
					continue
				}

				if err := consumer.HandleExportRequested(ctx, message); err != nil {
					log.Printf("export task handling failed for %s: %v", message.TaskID, err)
					_ = delivery.Nack(false, true)
					continue
				}

				_ = delivery.Ack(false)
			}
		}
	}()

	return func() {
		_ = channel.Close()
		_ = conn.Close()
	}, nil
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
```

- [ ] **Step 6: 写 export-service 新入口**

Create `backend/services/export-service/cmd/main.go`:

```go
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"inkwords-backend/services/export-service/app/bootstrap"
	"inkwords-backend/shared/kernel/httpx"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default environment variables")
	}
}

func main() {
	r, consumer, err := bootstrap.BuildRouter()
	if err != nil {
		log.Fatalf("bootstrap export-service failed: %v", err)
	}

	server := httpx.NewServer(r)
	signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	stopConsumer, err := export.StartExportConsumer(signalContext, consumer, "inkwords.export")
	if err != nil {
		log.Printf("RabbitMQ export consumer initialization skipped: %v", err)
	}
	defer stopConsumer()

	go func() {
		if err := httpx.ShutdownOnContextDone(signalContext, server, 15*time.Second); err != nil {
			log.Printf("Server shutdown failed: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server startup failed: %v", err)
	}
}
```

- [ ] **Step 7: 补充入口 import 并跑导出链路相关测试**

Update `backend/services/export-service/cmd/main.go` imports:

```go
import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"inkwords-backend/services/export-service/app/bootstrap"
	export "inkwords-backend/services/export-service/domain/export"
	"inkwords-backend/shared/kernel/httpx"
)
```

Run:

```bash
cd backend && go test ./internal/domain/task/... ./internal/service/... -run 'Export|PDF|Obsidian' -v
```

Expected: 导出相关测试继续通过。

- [ ] **Step 8: 验证 export-service 容器**

Run:

```bash
docker compose --env-file backend/.env up -d --build export-service
docker compose ps export-service
```

Expected: `export-service` 为 `Up` 或 `healthy`。

- [ ] **Step 9: 提交 export-service 迁移**

```bash
git add backend/services/export-service backend/Dockerfile
git commit -m "refactor(export): move export-service into service-owned structure"
```

### Task 5: 切换 Compose 与构建入口到新服务目录

**Files:**
- Modify: `backend/Dockerfile`
- Modify: `docker-compose.yml`
- Test: `docs/runbooks/microservices-smoke-check.md`

- [ ] **Step 1: 保持 Compose 服务名不变，只切换构建入口**

确认 `docker-compose.yml` 继续保留：

```yaml
parser-service:
  command: ["./parser-service"]

export-service:
  command: ["./export-service"]

review-service:
  command: ["./review-service"]
```

Expected: Compose 不改服务名、不改对外网络拓扑。

- [ ] **Step 2: 重建三个服务并检查日志**

Run:

```bash
docker compose --env-file backend/.env down
docker compose --env-file backend/.env up -d --build review-service parser-service export-service
docker compose logs --no-color review-service parser-service export-service | tail -n 200
```

Expected: 新日志中无 `package not found`、`missing symbol`、`panic: missing handler`。

- [ ] **Step 3: 跑完整后端测试**

Run:

```bash
cd backend && go test ./...
```

Expected: 全量通过。

- [ ] **Step 4: 提交 Compose 接线**

```bash
git add backend/Dockerfile docker-compose.yml
git commit -m "chore(compose): wire phase1 services to new build entrypoints"
```

### Task 6: 文档同步与收尾验证

**Files:**
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `README.md`
- Modify: `docs/runbooks/microservices-smoke-check.md`

- [ ] **Step 1: 更新架构文档**

在 `.trae/documents/InkWords_Architecture.md` 追加：

```md
- 2026-06-03：后端真实服务拆分 Phase 1 落地。`review-service`、`parser-service`、`export-service` 已迁入 `backend/services/<service>/` 服务自有目录；`shared/` 开始承接极薄基础层，旧 `internal/*` 进入过渡并存状态。
```

- [ ] **Step 2: 更新开发日志与对话日志**

在 `.trae/documents/InkWords_Development_Plan_and_Log.md` 追加：

```md
## 2026-06-03 后端真实服务拆分 Phase 1
- 完成 `review-service / parser-service / export-service` 目录归属迁移
- 保持 Compose 服务名、HTTP 路由和对外入口不变
- 为下一批 `core-api / llm-stream` 深拆分建立 `services/* + shared/*` 骨架
```

在 `.trae/documents/InkWords_Conversation_Log.md` 追加：

```md
- 用户要求：把项目目录从“看起来不像微服务”调整为真实服务拆分。
- 本轮结论：先迁 `review-service / parser-service / export-service`，后续再拆 `core-api / llm-stream`。
```

- [ ] **Step 3: 更新 README 与 runbook**

在 `README.md` 添加说明：

```md
## Backend 服务目录

后端正在从共享 `internal/*` 结构迁移到 `backend/services/<service>/` 真实服务目录。
Phase 1 已迁移：`review-service`、`parser-service`、`export-service`。
```

在 `docs/runbooks/microservices-smoke-check.md` 添加检查点：

```md
- 确认 `review-service`、`parser-service`、`export-service` 的二进制来自 `backend/services/<service>/cmd`
- 若构建失败，优先检查 `backend/Dockerfile` 中对应的 `go build` 入口
```

- [ ] **Step 4: 执行最终冒烟**

Run:

```bash
docker compose --env-file backend/.env down
docker compose --env-file backend/.env up -d --build
docker compose ps
curl -I http://localhost
curl http://localhost/api/v1/ping
```

Expected:

```text
所有容器 Up/healthy
HTTP/1.1 200 OK
{"code":200,"message":"pong","data":null}
```

- [ ] **Step 5: 提交文档同步**

```bash
git add .trae/documents/InkWords_Architecture.md \
        .trae/documents/InkWords_Conversation_Log.md \
        .trae/documents/InkWords_Development_Plan_and_Log.md \
        README.md \
        docs/runbooks/microservices-smoke-check.md
git commit -m "docs(backend): document phase1 real service split"
```
