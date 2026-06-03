# InkWords RabbitMQ Generation Phase Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在保留现有 `http://localhost` 单入口与 `/api/*` 对外路径不变的前提下，为 InkWords 的生成链路引入 RabbitMQ 任务队列、任务状态存储和 DB 驱动的 SSE 订阅，让 `llm-stream` 从“同步长请求服务”升级为“后台生成 worker + 兼容旧 SSE”。

**Architecture:** 保留现有 `core-api / llm-stream / parser-service / export-service / review-service` 和 Nginx 路由分流；`core-api` 新增任务创建、查询、取消和 SSE 订阅接口，`llm-stream` 新增 RabbitMQ consumer 并复用现有 `stream.Service` 执行生成。任务状态与事件先落 Postgres（`job_tasks`、`job_task_events`），由 `core-api` 通过轮询事件表输出 SSE，避免在第一阶段再引入 Redis pubsub 或跨服务内存总线。

**Tech Stack:** Go 1.25 + Gin + GORM + PostgreSQL + RabbitMQ + Docker Compose + React 18 + Zustand + `@microsoft/fetch-event-source`

---

## Scope

这是一份**第一阶段可执行计划**，只覆盖：

- Phase A：架构收口（正式确认五服务形态，保留 `cmd/server` 仅作本地对照）
- Phase B：生成链路事件化（`generate / continue / polish / analyze / scan`）

明确不包含：

- `parser-service` 的异步化实施
- `export-service` 的异步化实施
- `review-service` 的异步化实施
- 每服务独立数据库

这三条链路在本计划完成后应继续各自独立立项。

## File Map

**Backend model / domain**
- Create: `backend/internal/model/job_task.go`
- Create: `backend/internal/domain/task/dto.go`
- Create: `backend/internal/domain/task/repository.go`
- Create: `backend/internal/domain/task/service.go`
- Create: `backend/internal/domain/task/service_test.go`
- Create: `backend/internal/domain/task/handler.go`
- Create: `backend/internal/domain/task/handler_test.go`

**Backend infra**
- Create: `backend/internal/infra/mq/message.go`
- Create: `backend/internal/infra/mq/rabbitmq.go`
- Create: `backend/internal/infra/mq/rabbitmq_test.go`
- Modify: `backend/internal/infra/db/db.go`

**Backend stream worker**
- Create: `backend/internal/domain/stream/task_consumer.go`
- Create: `backend/internal/domain/stream/task_consumer_test.go`
- Modify: `backend/cmd/llm-stream/main.go`
- Modify: `backend/cmd/core-api/main.go`
- Modify: `backend/internal/transport/http/v1/routes.go`
- Modify: `backend/internal/transport/http/v1/routes_test.go`
- Create: `backend/internal/transport/http/v1/api/task.go`

**Frontend**
- Create: `frontend/src/services/generationTasks.ts`
- Create: `frontend/src/services/generationTasks.test.ts`
- Modify: `frontend/src/hooks/generator/useSeriesGenerator.ts`
- Modify: `frontend/src/store/streamStore.ts`

**Infra / Docs**
- Modify: `docker-compose.yml`
- Modify: `backend/.env.example`
- Modify: `README.md`
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Database.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`

---

### Task 1: 建立任务模型与任务领域服务

**Files:**
- Create: `backend/internal/model/job_task.go`
- Create: `backend/internal/domain/task/dto.go`
- Create: `backend/internal/domain/task/repository.go`
- Create: `backend/internal/domain/task/service.go`
- Test: `backend/internal/domain/task/service_test.go`
- Modify: `backend/internal/infra/db/db.go`

- [ ] **Step 1: 先写任务服务失败测试**

在 `backend/internal/domain/task/service_test.go` 先定义一个内存 fake repository，锁定“创建任务、幂等查重、追加事件、取消任务”四个核心行为：

```go
func TestService_CreateGenerationTask_ReusesIdempotencyKey(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo, nil)

	req := CreateGenerationTaskInput{
		RequestedBy:    uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		TaskSubtype:    "generate_series",
		IdempotencyKey: "series:abc",
		Payload:        []byte(`{"series_title":"Go源码入门"}`),
	}

	first, err := service.CreateGenerationTask(context.Background(), req)
	require.NoError(t, err)

	second, err := service.CreateGenerationTask(context.Background(), req)
	require.NoError(t, err)

	require.Equal(t, first.ID, second.ID)
	require.Equal(t, JobTaskStatusQueued, second.Status)
}

func TestService_AppendEvent_UpdatesStreamingState(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo, nil)

	task := repo.seedTask(JobTask{
		ID:     uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Status: JobTaskStatusRunning,
	})

	err := service.AppendEvent(context.Background(), task.ID, AppendEventInput{
		EventType: "chunk",
		Status:    JobTaskStatusStreaming,
		Payload:   []byte(`{"status":"streaming","chapter_sort":1,"content":"hello"}`),
	})
	require.NoError(t, err)

	stored, err := repo.GetByID(context.Background(), task.ID)
	require.NoError(t, err)
	require.Equal(t, JobTaskStatusStreaming, stored.Status)
}

func TestService_CancelTask_MarksCancelled(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo, nil)

	task := repo.seedTask(JobTask{
		ID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		RequestedBy: uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		Status:      JobTaskStatusQueued,
	})

	err := service.CancelTask(context.Background(), task.ID, task.RequestedBy)
	require.NoError(t, err)

	stored, err := repo.GetByID(context.Background(), task.ID)
	require.NoError(t, err)
	require.Equal(t, JobTaskStatusCancelled, stored.Status)
}
```

- [ ] **Step 2: 运行测试，确认当前失败**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/task -count=1
```

Expected: FAIL，原因是 `task` 领域文件还不存在。

- [ ] **Step 3: 写最小实现（模型 + 服务 + 仓储接口）**

`backend/internal/model/job_task.go` 写入两个 GORM 模型：

```go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type JobTaskStatus string

const (
	JobTaskStatusPending   JobTaskStatus = "pending"
	JobTaskStatusQueued    JobTaskStatus = "queued"
	JobTaskStatusRunning   JobTaskStatus = "running"
	JobTaskStatusStreaming JobTaskStatus = "streaming"
	JobTaskStatusSucceeded JobTaskStatus = "succeeded"
	JobTaskStatusFailed    JobTaskStatus = "failed"
	JobTaskStatusCancelled JobTaskStatus = "cancelled"
)

type JobTask struct {
	ID             uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TaskType       string         `gorm:"size:32;index"`
	TaskSubtype    string         `gorm:"size:64;index"`
	Status         JobTaskStatus  `gorm:"size:16;index"`
	RequestedBy    uuid.UUID      `gorm:"type:uuid;index"`
	IdempotencyKey string         `gorm:"size:255;index"`
	PayloadJSON    datatypes.JSON `gorm:"type:jsonb"`
	ResultJSON     datatypes.JSON `gorm:"type:jsonb"`
	ErrorMessage   string         `gorm:"type:text"`
	RetryCount     int
	StartedAt      *time.Time
	FinishedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type JobTaskEvent struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement"`
	TaskID    uuid.UUID      `gorm:"type:uuid;index"`
	EventType string         `gorm:"size:32;index"`
	Status    JobTaskStatus  `gorm:"size:16;index"`
	Payload   datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt time.Time
}
```

`backend/internal/domain/task/service.go` 先只实现当前计划需要的方法：

```go
type Repository interface {
	FindByIdempotencyKey(ctx context.Context, requestedBy uuid.UUID, taskType, key string) (*model.JobTask, error)
	Create(ctx context.Context, task *model.JobTask) error
	GetByID(ctx context.Context, taskID uuid.UUID) (*model.JobTask, error)
	UpdateStatus(ctx context.Context, taskID uuid.UUID, status model.JobTaskStatus, errorMessage string) error
	AppendEvent(ctx context.Context, event *model.JobTaskEvent) error
	ListEventsAfter(ctx context.Context, taskID uuid.UUID, afterID uint64, limit int) ([]model.JobTaskEvent, error)
}

type Publisher interface {
	PublishGenerationRequested(ctx context.Context, payload GenerationRequestedMessage) error
}
```

`backend/internal/infra/db/db.go` 把两张表加入 core DB 迁移清单：

```go
func autoMigrateCore(database *gorm.DB) error {
	return database.AutoMigrate(
		&model.User{},
		&model.Blog{},
		&model.OAuthToken{},
		&model.UserPromptSettings{},
		&model.JobTask{},
		&model.JobTaskEvent{},
	)
}
```

- [ ] **Step 4: 运行测试，确认任务领域服务通过**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/task -count=1
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add backend/internal/model/job_task.go backend/internal/domain/task backend/internal/infra/db/db.go
git commit -m "feat(task): add generation task model and domain service"
```

---

### Task 2: 引入 RabbitMQ 消息封装与发布器

**Files:**
- Create: `backend/internal/infra/mq/message.go`
- Create: `backend/internal/infra/mq/rabbitmq.go`
- Test: `backend/internal/infra/mq/rabbitmq_test.go`

- [ ] **Step 1: 先写消息层测试**

`backend/internal/infra/mq/rabbitmq_test.go` 先锁定 routing key 和消息序列化：

```go
func TestGenerationRequestedMessage_RoutingKey(t *testing.T) {
	msg := GenerationRequestedMessage{
		TaskID:  uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Kind:    "generate_series",
		UserID:  uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		Payload: json.RawMessage(`{"series_title":"Go 并发"}`),
	}

	require.Equal(t, "generation.requested", msg.RoutingKey())
}

func TestMarshalMessage_ContainsTaskID(t *testing.T) {
	msg := GenerationRequestedMessage{
		TaskID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Kind:   "generate_single",
	}

	body, err := json.Marshal(msg)
	require.NoError(t, err)
	require.Contains(t, string(body), `"task_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"`)
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/infra/mq -count=1
```

Expected: FAIL，原因是 `mq` 包不存在。

- [ ] **Step 3: 写消息结构与 RabbitMQ 发布器**

`backend/internal/infra/mq/message.go`：

```go
package mq

import (
	"encoding/json"

	"github.com/google/uuid"
)

type GenerationRequestedMessage struct {
	TaskID  uuid.UUID       `json:"task_id"`
	Kind    string          `json:"kind"`
	UserID  uuid.UUID       `json:"user_id"`
	Payload json.RawMessage `json:"payload"`
}

func (GenerationRequestedMessage) RoutingKey() string {
	return "generation.requested"
}
```

`backend/internal/infra/mq/rabbitmq.go`：

```go
type Publisher struct {
	channel  *amqp.Channel
	exchange string
}

func NewPublisher(channel *amqp.Channel, exchange string) *Publisher {
	return &Publisher{channel: channel, exchange: exchange}
}

func (p *Publisher) PublishGenerationRequested(ctx context.Context, message GenerationRequestedMessage) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return p.channel.PublishWithContext(
		ctx,
		p.exchange,
		message.RoutingKey(),
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}
```

- [ ] **Step 4: 运行测试，确认通过**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/infra/mq -count=1
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add backend/internal/infra/mq
git commit -m "feat(mq): add rabbitmq generation publisher"
```

---

### Task 3: 在 core-api 中新增任务创建、查询、取消与 SSE 订阅接口

**Files:**
- Create: `backend/internal/domain/task/dto.go`
- Create: `backend/internal/domain/task/handler.go`
- Test: `backend/internal/domain/task/handler_test.go`
- Create: `backend/internal/transport/http/v1/api/task.go`
- Modify: `backend/internal/transport/http/v1/routes.go`
- Test: `backend/internal/transport/http/v1/routes_test.go`
- Modify: `backend/cmd/core-api/main.go`

- [ ] **Step 1: 先写 handler 与路由失败测试**

`backend/internal/domain/task/handler_test.go`：

```go
func TestHandler_CreateGenerationTask_Returns202(t *testing.T) {
	service := newFakeService()
	handler := NewHandler(service)

	router := gin.New()
	router.POST("/api/v1/tasks/generation", func(c *gin.Context) {
		c.Set("user_id", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		handler.CreateGenerationTask(c)
	})

	body := `{"kind":"generate_single","payload":{"source_content":"hello","scenario_mode":"ebook_interpretation"},"idempotency_key":"gen:1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/generation", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusAccepted, resp.Code)
	require.Contains(t, resp.Body.String(), `"task_id"`)
}
```

`backend/internal/transport/http/v1/routes_test.go` 再补一条可达性测试：

```go
func TestRegisterCore_TaskRoutesAreReachable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	authMiddleware := func(c *gin.Context) { c.Next() }
	ok := func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) }

	RegisterCore(r, authMiddleware, CoreHandlers{
		Auth: AuthHandlers{Register: ok, Login: ok, BindGithub: ok, GetCaptcha: ok, OAuthRedirect: ok, OAuthCallback: ok},
		User: UserHandlers{GetProfile: ok, UpdateProfile: ok, UploadAvatar: ok, GetUserStats: ok, GetPromptSettings: ok, UpdatePromptSettings: ok},
		Blog: CoreBlogHandlers{GetUserBlogs: ok, CreateDraftBlog: ok, BatchDeleteBlogs: ok, UpdateBlog: ok},
		Project: CoreProjectHandlers{ScanGithubRepo: ok, Analyze: ok},
		Task: TaskHandlers{CreateGeneration: ok, GetTask: ok, CancelTask: ok, StreamTask: ok},
	})

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/tasks/generation"},
		{http.MethodGet, "/api/v1/tasks/123e4567-e89b-12d3-a456-426614174000"},
		{http.MethodPost, "/api/v1/tasks/123e4567-e89b-12d3-a456-426614174000/cancel"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/task ./internal/transport/http/v1 -count=1
```

Expected: FAIL，因为 `TaskHandlers`、`CreateGenerationTask`、`StreamTask` 还不存在。

- [ ] **Step 3: 写最小实现（task handler + routes + core wiring）**

在 `backend/internal/transport/http/v1/routes.go` 扩展 `CoreHandlers`：

```go
type TaskHandlers struct {
	CreateGeneration gin.HandlerFunc
	GetTask          gin.HandlerFunc
	CancelTask       gin.HandlerFunc
	StreamTask       gin.HandlerFunc
}

type CoreHandlers struct {
	Auth    AuthHandlers
	User    UserHandlers
	Blog    CoreBlogHandlers
	Project CoreProjectHandlers
	Task    TaskHandlers
}
```

并在 `RegisterCore` 中新增：

```go
taskGroup := v1.Group("/tasks")
taskGroup.Use(authMiddleware)
{
	taskGroup.POST("/generation", handlers.Task.CreateGeneration)
	taskGroup.GET("/:id", handlers.Task.GetTask)
	taskGroup.POST("/:id/cancel", handlers.Task.CancelTask)
	taskGroup.GET("/:id/stream", handlers.Task.StreamTask)
}
```

`backend/internal/domain/task/handler.go` 先实现核心接口：

```go
func (h *Handler) CreateGenerationTask(c *gin.Context) {
	var req CreateGenerationTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	task, err := h.service.CreateGenerationTask(c.Request.Context(), CreateGenerationTaskInput{
		RequestedBy:    userID,
		TaskSubtype:    req.Kind,
		IdempotencyKey: req.IdempotencyKey,
		Payload:        req.Payload,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"task_id":    task.ID,
		"status":     task.Status,
		"stream_url": "/api/v1/tasks/" + task.ID.String() + "/stream",
	})
}
```

`StreamTask` 第一阶段用 DB 轮询事件表输出 SSE：

```go
func (h *Handler) StreamTask(c *gin.Context) {
	taskID := uuid.MustParse(c.Param("id"))
	afterID := uint64(0)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			events, done, err := h.service.ListStreamEvents(c.Request.Context(), taskID, afterID)
			if err != nil {
				c.SSEvent("error", "task stream failed")
				return
			}
			for _, event := range events {
				afterID = event.ID
				c.SSEvent(event.EventType, string(event.Payload))
				c.Writer.Flush()
			}
			if done {
				c.SSEvent("done", "[DONE]")
				c.Writer.Flush()
				return
			}
		}
	}
}
```

`backend/cmd/core-api/main.go` 增加 task service 和 task API 装配，并把 handler 注入 `RegisterCore`。

- [ ] **Step 4: 运行测试，确认通过**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/task ./internal/transport/http/v1 -count=1
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add backend/internal/domain/task backend/internal/transport/http/v1 backend/internal/transport/http/v1/api/task.go backend/cmd/core-api/main.go
git commit -m "feat(core-api): add generation task endpoints and sse task stream"
```

---

### Task 4: 在 llm-stream 中新增 RabbitMQ consumer，并复用现有 stream.Service 执行生成

**Files:**
- Create: `backend/internal/domain/stream/task_consumer.go`
- Test: `backend/internal/domain/stream/task_consumer_test.go`
- Modify: `backend/cmd/llm-stream/main.go`
- Modify: `backend/internal/domain/task/service.go`

- [ ] **Step 1: 先写 consumer 失败测试**

`backend/internal/domain/stream/task_consumer_test.go`：

```go
func TestTaskConsumer_RunGenerateSingle_AppendsChunkAndCompletes(t *testing.T) {
	taskService := newFakeTaskService()
	streamService := newFakeStreamService(
		func(ctx context.Context, userID uuid.UUID, req GenerateRequest, chunkChan chan<- string, errChan chan<- error) {
			chunkChan <- "hello"
			close(chunkChan)
			close(errChan)
		},
	)

	consumer := NewTaskConsumer(taskService, streamService)
	message := mq.GenerationRequestedMessage{
		TaskID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Kind:   "generate_single",
		UserID: uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		Payload: json.RawMessage(`{
			"source_type":"file",
			"source_content":"hello world",
			"scenario_mode":"ebook_interpretation"
		}`),
	}

	err := consumer.HandleGenerationRequested(context.Background(), message)
	require.NoError(t, err)
	require.Equal(t, "succeeded", taskService.lastStatus)
	require.Contains(t, string(taskService.appendedPayloads[0]), `"hello"`)
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/stream -run TestTaskConsumer -count=1
```

Expected: FAIL，因为 `NewTaskConsumer` 与 `HandleGenerationRequested` 还不存在。

- [ ] **Step 3: 写最小实现（consumer + 取消轮询）**

`backend/internal/domain/stream/task_consumer.go`：

```go
type taskService interface {
	MarkRunning(ctx context.Context, taskID uuid.UUID) error
	AppendEvent(ctx context.Context, taskID uuid.UUID, input task.AppendEventInput) error
	MarkSucceeded(ctx context.Context, taskID uuid.UUID, result []byte) error
	MarkFailed(ctx context.Context, taskID uuid.UUID, message string) error
	IsCancelled(ctx context.Context, taskID uuid.UUID) (bool, error)
}

type TaskConsumer struct {
	tasks   taskService
	service streamService
}

func (c *TaskConsumer) HandleGenerationRequested(ctx context.Context, message mq.GenerationRequestedMessage) error {
	if err := c.tasks.MarkRunning(ctx, message.TaskID); err != nil {
		return err
	}

	var req GenerateRequest
	if err := json.Unmarshal(message.Payload, &req); err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, "invalid generation payload")
	}

	taskCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-taskCtx.Done():
				return
			case <-ticker.C:
				cancelled, err := c.tasks.IsCancelled(taskCtx, message.TaskID)
				if err == nil && cancelled {
					cancel()
					return
				}
			}
		}
	}()

	chunkChan := make(chan string, 128)
	errChan := make(chan error, 1)
	go c.service.Generate(taskCtx, message.UserID, req, chunkChan, errChan)

	for {
		select {
		case <-taskCtx.Done():
			return c.tasks.MarkFailed(ctx, message.TaskID, "task cancelled")
		case err, ok := <-errChan:
			if ok && err != nil {
				return c.tasks.MarkFailed(ctx, message.TaskID, err.Error())
			}
			errChan = nil
		case chunk, ok := <-chunkChan:
			if !ok {
				return c.tasks.MarkSucceeded(ctx, message.TaskID, []byte(`{"done":true}`))
			}
			if err := c.tasks.AppendEvent(ctx, message.TaskID, task.AppendEventInput{
				EventType: "chunk",
				Status:    model.JobTaskStatusStreaming,
				Payload:   []byte(chunk),
			}); err != nil {
				return err
			}
		}
	}
}
```

`backend/cmd/llm-stream/main.go` 增加 consumer 启动：

```go
conn, ch, err := mq.OpenFromEnv()
if err != nil {
	log.Fatalf("rabbitmq initialization failed: %v", err)
}
defer conn.Close()
defer ch.Close()

publisherlessTaskConsumer := streamdomain.NewTaskConsumer(taskService, streamDomainService)
go func() {
	if err := mq.ConsumeGenerationRequested(signalContext, ch, os.Getenv("RABBITMQ_EXCHANGE"), os.Getenv("RABBITMQ_GENERATION_QUEUE"), publisherlessTaskConsumer.HandleGenerationRequested); err != nil {
		log.Printf("generation consumer stopped: %v", err)
	}
}()
```

- [ ] **Step 4: 运行测试，确认通过**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/stream -run TestTaskConsumer -count=1
go test ./cmd/llm-stream -run '^$'
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add backend/internal/domain/stream/task_consumer.go backend/internal/domain/stream/task_consumer_test.go backend/cmd/llm-stream/main.go
git commit -m "feat(llm-stream): consume generation tasks from rabbitmq"
```

---

### Task 5: 前端生成器切换到“创建任务 + SSE 订阅任务流”

**Files:**
- Create: `frontend/src/services/generationTasks.ts`
- Test: `frontend/src/services/generationTasks.test.ts`
- Modify: `frontend/src/hooks/generator/useSeriesGenerator.ts`
- Modify: `frontend/src/store/streamStore.ts`

- [ ] **Step 1: 先写前端服务失败测试**

`frontend/src/services/generationTasks.test.ts`：

```ts
import { describe, expect, it } from 'vitest'
import { buildGenerationTaskRequest } from './generationTasks'

describe('buildGenerationTaskRequest', () => {
  it('maps single generation payload to task request', () => {
    expect(
      buildGenerationTaskRequest('generate_single', {
        source_type: 'file',
        source_content: 'hello',
        outline: [],
        scenario_mode: 'ebook_interpretation',
      }),
    ).toEqual({
      kind: 'generate_single',
      payload: {
        source_type: 'file',
        source_content: 'hello',
        outline: [],
        scenario_mode: 'ebook_interpretation',
      },
    })
  })
})
```

- [ ] **Step 2: 运行测试，确认失败**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend
npm test -- --run generationTasks.test.ts
```

Expected: FAIL，因为 `generationTasks.ts` 不存在。

- [ ] **Step 3: 写任务服务，并在 useSeriesGenerator 中切换入口**

`frontend/src/services/generationTasks.ts`：

```ts
export interface CreateGenerationTaskResponse {
  task_id: string
  status: string
  stream_url: string
}

export const buildGenerationTaskRequest = (kind: string, payload: Record<string, unknown>) => ({
  kind,
  payload,
})

export async function createGenerationTask(body: ReturnType<typeof buildGenerationTaskRequest>) {
  const response = await fetch('/api/v1/tasks/generation', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...buildAuthHeader(authTokenStore.getSnapshot()),
    },
    body: JSON.stringify(body),
  })
  if (!response.ok) {
    throw new Error('创建生成任务失败')
  }
  return (await response.json()) as CreateGenerationTaskResponse
}
```

在 `frontend/src/hooks/generator/useSeriesGenerator.ts` 中把：

```ts
await fetchEventSourceWithAuth('/api/v1/stream/generate', {
```

替换为两段：

```ts
const task = await createGenerationTask(
  buildGenerationTaskRequest('generate_series', buildSeriesGenerateRequest({...}))
)

await fetchEventSourceWithAuth(task.stream_url, {
  method: 'GET',
  signal: ctrl.signal,
  openWhenHidden: true,
  onmessage(msg) {
    const currentStore = useStreamStore.getState()
    if (msg.event === 'done') {
      currentStore.flushBufferedChapterContents()
      currentStore.flushBufferedContent()
      currentStore.setGenerating(false)
      currentStore.setProgress('生成完成')
      return
    }
    if (msg.event === 'chunk') {
      handleSeriesChunkMessage(currentStore, msg.data)
      return
    }
    if (msg.event === 'error') {
      throw new StopStreamError(msg.data)
    }
  },
})
```

并在 `streamStore.ts` 增加任务态字段：

```ts
currentTaskId: string | null
setCurrentTaskId: (taskId: string | null) => void
```

- [ ] **Step 4: 运行测试并补一次前端类型检查**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend
npm test -- --run generationTasks.test.ts useSeriesGenerator.test.ts
npm run build
```

Expected:
- 测试 PASS
- `vite build` PASS

- [ ] **Step 5: 提交**

```bash
git add frontend/src/services/generationTasks.ts frontend/src/services/generationTasks.test.ts frontend/src/hooks/generator/useSeriesGenerator.ts frontend/src/store/streamStore.ts
git commit -m "feat(frontend): switch generation flow to task-based sse"
```

---

### Task 6: Docker Compose、环境变量与文档同步

**Files:**
- Modify: `docker-compose.yml`
- Modify: `backend/.env.example`
- Modify: `README.md`
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Database.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`

- [ ] **Step 1: 加入 RabbitMQ 服务与环境变量**

在 `docker-compose.yml` 新增：

```yaml
  rabbitmq:
    image: rabbitmq:3-management-alpine
    container_name: inkwords-rabbitmq
    expose:
      - "5672"
      - "15672"
    networks:
      - inkwords-network
    restart: unless-stopped
```

为 `core-api` 与 `llm-stream` 增加：

```yaml
      RABBITMQ_URL: ${RABBITMQ_URL:-amqp://guest:guest@rabbitmq:5672/}
      RABBITMQ_EXCHANGE: ${RABBITMQ_EXCHANGE:-inkwords.events}
      RABBITMQ_GENERATION_QUEUE: ${RABBITMQ_GENERATION_QUEUE:-inkwords.generation}
```

并让这两个服务 `depends_on` 包含 `rabbitmq`。

- [ ] **Step 2: 更新 `.env.example`**

在 `backend/.env.example` 增加：

```env
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
RABBITMQ_EXCHANGE=inkwords.events
RABBITMQ_GENERATION_QUEUE=inkwords.generation
```

- [ ] **Step 3: 做 Compose 与后端编译验证**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
docker compose --env-file backend/.env config
cd backend && go test ./... -run '^$'
cd ../frontend && npm run build
```

Expected:
- `docker compose config` 无错误
- 后端编译通过
- 前端构建通过

- [ ] **Step 4: 做完整冒烟验证**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
docker compose --env-file backend/.env down
docker compose --env-file backend/.env up -d --build
docker compose --env-file backend/.env ps
curl -I http://localhost
```

实际检查：
- 登录成功
- 创建单篇生成任务成功
- 任务 SSE 能持续输出 chunk
- 生成完成后历史列表可见新博客

- [ ] **Step 5: 同步文档**

至少更新以下内容：

- `README.md`
  - 补充 `rabbitmq` 服务说明
  - 补充“生成链路改为任务式 SSE”的说明
- `.trae/documents/InkWords_API.md`
  - 补充 `/api/v1/tasks/generation`
  - 补充 `/api/v1/tasks/:id`
  - 补充 `/api/v1/tasks/:id/stream`
  - 补充 `/api/v1/tasks/:id/cancel`
- `.trae/documents/InkWords_Architecture.md`
  - 追加“RabbitMQ 事件驱动 Phase B”
- `.trae/documents/InkWords_Database.md`
  - 追加 `job_tasks` 与 `job_task_events`
- `.trae/documents/InkWords_Development_Plan_and_Log.md`
  - 记录验证命令与结果

- [ ] **Step 6: 提交**

```bash
git add docker-compose.yml backend/.env.example README.md .trae/documents/InkWords_API.md .trae/documents/InkWords_Architecture.md .trae/documents/InkWords_Database.md .trae/documents/InkWords_Development_Plan_and_Log.md
git commit -m "docs: document rabbitmq-backed generation task architecture"
```

---

## Self-Review

- 本计划只覆盖生成链路事件化，没有把 `parser/export/review` 混入同一实施批次。
- 所有新接口都保持在 `/api/v1` 下，对前端公开入口不变。
- 任务状态、事件表、RabbitMQ 发布器、worker、前端任务订阅都有对应任务。
- 回滚路径保留：旧 `/api/v1/stream/*` 处理链路仍在 `llm-stream`，直到新任务式前端稳定后再决定是否下线旧入口。
- 下一份计划应独立处理：
  - `parser-service` 异步化
  - `export-service` 异步化
  - 可选的 `task cancel` 深化与 Redis/pubsub 优化
