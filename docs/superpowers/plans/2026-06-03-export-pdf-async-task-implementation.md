# InkWords Export PDF Async Task Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在保持 `http://localhost` 单入口与现有同步 PDF 导出回滚路径不变的前提下，把 `export_pdf` 接入 `job_tasks + RabbitMQ + SSE` 任务中心，并支持任务完成后的受控下载。

**Architecture:** `core-api` 新增 `export` 任务创建与下载入口，负责鉴权、任务生命周期与结果文件分发；`export-service` 新增 `export.requested` consumer，复用现有 PDF 导出逻辑生成文件并写入共享导出目录。前端“导出 PDF”按钮改为“创建任务 -> 订阅 SSE -> 下载文件”，其余导出按钮保持不变。

**Tech Stack:** Go 1.25 + Gin + GORM + PostgreSQL + RabbitMQ + Docker Compose + React 18 + Vitest + `@microsoft/fetch-event-source`

---

## Scope

本计划只覆盖：

- `export_pdf` 任务 subtype
- `POST /api/v1/tasks/export`
- `GET /api/v1/tasks/:id/download`
- `export-service` PDF worker consumer
- 前端 PDF 导出按钮任务化
- 共享导出目录、环境变量和文档同步

明确不包含：

- `export_obsidian_single`
- `export_obsidian_series`
- `export_markdown_zip`
- 对象存储（S3 / MinIO）
- 批量导出 UI 重构

## File Map

**Backend task domain / routing**
- Create: `backend/internal/domain/task/export_task.go`
- Create: `backend/internal/domain/task/download_handler.go`
- Create: `backend/internal/domain/task/download_handler_test.go`
- Modify: `backend/internal/domain/task/dto.go`
- Modify: `backend/internal/domain/task/repository.go`
- Modify: `backend/internal/domain/task/service.go`
- Modify: `backend/internal/domain/task/service_test.go`
- Modify: `backend/internal/domain/task/handler.go`
- Modify: `backend/internal/domain/task/handler_test.go`
- Modify: `backend/internal/transport/http/v1/api/task.go`
- Modify: `backend/internal/transport/http/v1/routes.go`
- Modify: `backend/internal/transport/http/v1/routes_test.go`
- Modify: `backend/cmd/core-api/main.go`
- Modify: `backend/cmd/core-api/main_test.go`

**Backend export worker / infra**
- Create: `backend/internal/domain/task/export_artifact_store.go`
- Create: `backend/internal/domain/task/export_consumer.go`
- Create: `backend/internal/domain/task/export_consumer_test.go`
- Modify: `backend/internal/infra/mq/message.go`
- Modify: `backend/internal/infra/mq/rabbitmq.go`
- Modify: `backend/cmd/export-service/main.go`
- Reuse: `backend/internal/service/pdf_export.go`

**Frontend**
- Create: `frontend/src/services/exportTasks.ts`
- Create: `frontend/src/services/exportTasks.test.ts`
- Modify: `frontend/src/services/sidebarExport.ts`
- Modify: `frontend/src/services/sidebarExport.test.ts`
- Modify: `frontend/src/components/Sidebar.tsx`

**Infra / Docs**
- Modify: `docker-compose.yml`
- Modify: `backend/.env.example`
- Modify: `README.md`
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`

---

### Task 1: 扩展任务领域，支持 `export_pdf` 创建与路由注册

**Files:**
- Create: `backend/internal/domain/task/export_task.go`
- Modify: `backend/internal/domain/task/dto.go`
- Modify: `backend/internal/domain/task/repository.go`
- Modify: `backend/internal/domain/task/service.go`
- Modify: `backend/internal/domain/task/service_test.go`
- Modify: `backend/internal/domain/task/handler.go`
- Modify: `backend/internal/domain/task/handler_test.go`
- Modify: `backend/internal/transport/http/v1/api/task.go`
- Modify: `backend/internal/transport/http/v1/routes.go`
- Modify: `backend/internal/transport/http/v1/routes_test.go`
- Modify: `backend/internal/infra/mq/message.go`
- Modify: `backend/internal/infra/mq/rabbitmq.go`
- Modify: `backend/cmd/core-api/main.go`
- Modify: `backend/cmd/core-api/main_test.go`

- [ ] **Step 1: 先写任务服务与路由失败测试**

在 `backend/internal/domain/task/service_test.go` 追加导出任务创建测试，在 `handler_test.go` 与 `routes_test.go` 锁定新路由：

```go
func TestService_CreateExportTask_PublishesMessage(t *testing.T) {
	repo := newFakeRepository()
	publisher := &fakePublisher{}
	service := NewService(repo, publisher)

	task, err := service.CreateExportTask(context.Background(), CreateExportTaskInput{
		RequestedBy:    uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		TaskSubtype:    ExportTaskSubtypePDF,
		IdempotencyKey: "export-pdf:series-1",
		Payload:        []byte(`{"blog_id":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"}`),
	})
	require.NoError(t, err)
	require.Equal(t, taskTypeExport, task.TaskType)
	require.Len(t, publisher.exportMessages, 1)
	require.Equal(t, ExportTaskSubtypePDF, publisher.exportMessages[0].Kind)
}

func TestHandler_CreateExportTask_ReturnsAccepted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeTaskHandlerService{
		createTaskResult: model.JobTask{
			ID:       uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"),
			TaskType: taskTypeExport,
			Status:   model.JobTaskStatusQueued,
		},
	}
	handler := NewHandler(service, "")

	router := gin.New()
	router.POST("/api/v1/tasks/export", func(c *gin.Context) {
		c.Set("user_id", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		handler.CreateExportTask(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/export", strings.NewReader(`{
		"kind":"export_pdf",
		"payload":{"blog_id":"22222222-2222-2222-2222-222222222222"},
		"idempotency_key":"export-pdf:series-1"
	}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusAccepted, resp.Code)
	require.Contains(t, resp.Body.String(), `"stream_url":"/api/v1/tasks/eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee/stream"`)
}
```

在 `backend/internal/transport/http/v1/routes_test.go` 的 `TaskHandlers` 用例中新增：

```go
Task: TaskHandlers{
	CreateGeneration: ok,
	CreateParse:      ok,
	CreateExport:     ok,
	GetTask:          ok,
	CancelTask:       ok,
	StreamTask:       ok,
	DownloadTask:     ok,
}
```

并增加路径断言：

```go
{method: http.MethodPost, path: "/api/v1/tasks/export"},
{method: http.MethodGet, path: "/api/v1/tasks/123e4567-e89b-12d3-a456-426614174000/download"},
```

- [ ] **Step 2: 运行测试，确认当前失败**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/task ./internal/transport/http/v1 -count=1
```

Expected: FAIL，原因是 `CreateExportTask`、`CreateExportTask` handler、`CreateExport` / `DownloadTask` 路由尚未实现。

- [ ] **Step 3: 写最小实现（DTO / publisher / service / handler / route）**

在 `backend/internal/domain/task/export_task.go` 定义导出任务常量与载荷：

```go
package task

import "github.com/google/uuid"

const (
	taskTypeExport        = "export"
	ExportTaskSubtypePDF  = "export_pdf"
)

type CreateExportTaskInput struct {
	RequestedBy    uuid.UUID
	TaskSubtype    string
	IdempotencyKey string
	Payload        []byte
}
```

在 `backend/internal/domain/task/dto.go` 追加消息结构：

```go
type ExportRequestedMessage struct {
	TaskID  uuid.UUID       `json:"task_id"`
	Kind    string          `json:"kind"`
	UserID  uuid.UUID       `json:"user_id"`
	Payload json.RawMessage `json:"payload"`
}
```

在 `backend/internal/domain/task/repository.go` 扩展 publisher：

```go
type Publisher interface {
	PublishGenerationRequested(ctx context.Context, payload GenerationRequestedMessage) error
	PublishParseRequested(ctx context.Context, payload ParseRequestedMessage) error
	PublishExportRequested(ctx context.Context, payload ExportRequestedMessage) error
}
```

在 `backend/internal/domain/task/service.go` 添加：

```go
func (s *Service) CreateExportTask(ctx context.Context, input CreateExportTaskInput) (model.JobTask, error) {
	return s.createTask(ctx, createTaskParams{
		taskType:       taskTypeExport,
		taskSubtype:    input.TaskSubtype,
		requestedBy:    input.RequestedBy,
		idempotencyKey: input.IdempotencyKey,
		payload:        input.Payload,
		publish: func(task model.JobTask) error {
			if s.publisher == nil {
				return nil
			}
			return s.publisher.PublishExportRequested(ctx, ExportRequestedMessage{
				TaskID:  task.ID,
				Kind:    task.TaskSubtype,
				UserID:  task.RequestedBy,
				Payload: append([]byte(nil), input.Payload...),
			})
		},
	})
}
```

在 `backend/internal/domain/task/handler.go` 追加导出 handler：

```go
func (h *Handler) CreateExportTask(c *gin.Context) {
	h.createTask(c, func(userID uuid.UUID, req CreateGenerationTaskRequest) (model.JobTask, error) {
		return h.service.CreateExportTask(c.Request.Context(), CreateExportTaskInput{
			RequestedBy:    userID,
			TaskSubtype:    req.Kind,
			IdempotencyKey: req.IdempotencyKey,
			Payload:        []byte(req.Payload),
		})
	})
}
```

在 `backend/internal/infra/mq/message.go` 和 `rabbitmq.go` 分别补：

```go
type ExportRequestedMessage struct {
	TaskID  uuid.UUID       `json:"task_id"`
	Kind    string          `json:"kind"`
	UserID  uuid.UUID       `json:"user_id"`
	Payload json.RawMessage `json:"payload"`
}

func (ExportRequestedMessage) RoutingKey() string { return "export.requested" }
```

```go
func (p *Publisher) PublishExportRequested(ctx context.Context, message taskdomain.ExportRequestedMessage) error {
	envelope := ExportRequestedMessage{
		TaskID:  message.TaskID,
		Kind:    message.Kind,
		UserID:  message.UserID,
		Payload: append(json.RawMessage(nil), message.Payload...),
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	return p.channel.PublishWithContext(ctx, p.exchange, envelope.RoutingKey(), false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}
```

最后在路由与入口里补齐：

```go
// routes.go
type TaskHandlers struct {
	CreateGeneration gin.HandlerFunc
	CreateParse      gin.HandlerFunc
	CreateExport     gin.HandlerFunc
	GetTask          gin.HandlerFunc
	CancelTask       gin.HandlerFunc
	StreamTask       gin.HandlerFunc
	DownloadTask     gin.HandlerFunc
}

taskGroup.POST("/export", handlers.Task.CreateExport)
taskGroup.GET("/:id/download", handlers.Task.DownloadTask)
```

```go
// api/task.go
func (a *TaskAPI) CreateExportTask(c *gin.Context) { a.taskDomainHandler.CreateExportTask(c) }
func (a *TaskAPI) DownloadTask(c *gin.Context)     { a.taskDomainHandler.DownloadTask(c) }
```

```go
// cmd/core-api/main.go
Task: transportv1.TaskHandlers{
	CreateGeneration: taskAPI.CreateGenerationTask,
	CreateParse:      taskAPI.CreateParseTask,
	CreateExport:     taskAPI.CreateExportTask,
	GetTask:          taskAPI.GetTask,
	CancelTask:       taskAPI.CancelTask,
	StreamTask:       taskAPI.StreamTask,
	DownloadTask:     taskAPI.DownloadTask,
}
```

- [ ] **Step 4: 运行测试，确认转绿**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/task ./internal/transport/http/v1 -count=1
```

Expected: PASS，`export_pdf` 的任务创建与路由注册测试通过。

- [ ] **Step 5: 提交**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add backend/internal/domain/task/export_task.go \
  backend/internal/domain/task/dto.go \
  backend/internal/domain/task/repository.go \
  backend/internal/domain/task/service.go \
  backend/internal/domain/task/service_test.go \
  backend/internal/domain/task/handler.go \
  backend/internal/domain/task/handler_test.go \
  backend/internal/transport/http/v1/api/task.go \
  backend/internal/transport/http/v1/routes.go \
  backend/internal/transport/http/v1/routes_test.go \
  backend/internal/infra/mq/message.go \
  backend/internal/infra/mq/rabbitmq.go \
  backend/cmd/core-api/main.go \
  backend/cmd/core-api/main_test.go
git commit -m "feat(export): add export task creation endpoints"
```

---

### Task 2: 为 `export-service` 增加 PDF worker 与共享导出目录

**Files:**
- Create: `backend/internal/domain/task/export_artifact_store.go`
- Create: `backend/internal/domain/task/export_consumer.go`
- Create: `backend/internal/domain/task/export_consumer_test.go`
- Modify: `backend/cmd/export-service/main.go`
- Reuse: `backend/internal/service/pdf_export.go`

- [ ] **Step 1: 先写 consumer 失败测试**

在 `backend/internal/domain/task/export_consumer_test.go` 先锁定“生成 PDF 后写结果元数据”和“导出失败时写 failed”：

```go
func TestExportConsumer_HandleExportRequested_PersistsDownloadMetadata(t *testing.T) {
	tasks := &fakeExportTaskService{}
	exporter := &stubPDFExporter{
		exportFunc: func(_ context.Context, blogID uuid.UUID, userID uuid.UUID) (string, string, error) {
			filePath := filepath.Join(t.TempDir(), "series.pdf")
			require.NoError(t, os.WriteFile(filePath, []byte("pdf"), 0o644))
			return filePath, "Go 源码入门.pdf", nil
		},
	}
	store := NewExportArtifactStore(t.TempDir(), 15*time.Minute, func() time.Time {
		return time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	})
	consumer := NewExportConsumer(tasks, exporter, store)

	err := consumer.HandleExportRequested(context.Background(), mq.ExportRequestedMessage{
		TaskID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Kind:   ExportTaskSubtypePDF,
		UserID: uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		Payload: json.RawMessage(`{
			"blog_id":"cccccccc-cccc-cccc-cccc-cccccccccccc"
		}`),
	})
	require.NoError(t, err)
	require.Equal(t, model.JobTaskStatusSucceeded, tasks.lastStatus)
	require.Contains(t, string(tasks.lastResult), `"content_type":"application/pdf"`)
	require.Contains(t, string(tasks.lastResult), `"filename":"Go 源码入门.pdf"`)
}

func TestExportConsumer_HandleExportRequested_MarksFailedWhenExporterFails(t *testing.T) {
	tasks := &fakeExportTaskService{}
	exporter := &stubPDFExporter{
		exportFunc: func(context.Context, uuid.UUID, uuid.UUID) (string, string, error) {
			return "", "", errors.New("chromium failed")
		},
	}
	consumer := NewExportConsumer(tasks, exporter, NewExportArtifactStore(t.TempDir(), 15*time.Minute, time.Now))

	err := consumer.HandleExportRequested(context.Background(), mq.ExportRequestedMessage{
		TaskID: uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"),
		Kind:   ExportTaskSubtypePDF,
		UserID: uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"),
		Payload: json.RawMessage(`{"blog_id":"ffffffff-ffff-ffff-ffff-ffffffffffff"}`),
	})
	require.NoError(t, err)
	require.Equal(t, model.JobTaskStatusFailed, tasks.lastStatus)
	require.Equal(t, "chromium failed", tasks.lastErrorMessage)
}
```

- [ ] **Step 2: 运行测试，确认当前失败**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/task -run 'TestExportConsumer' -count=1
```

Expected: FAIL，原因是 `ExportArtifactStore` 和 `ExportConsumer` 尚未实现。

- [ ] **Step 3: 写最小实现（artifact store + consumer + export-service 启动）**

在 `backend/internal/domain/task/export_artifact_store.go` 创建共享目录文件仓储：

```go
package task

type ExportArtifactStore struct {
	rootDir     string
	ttl         time.Duration
	nowProvider func() time.Time
}

func NewExportArtifactStore(rootDir string, ttl time.Duration, nowProvider func() time.Time) *ExportArtifactStore {
	return &ExportArtifactStore{rootDir: rootDir, ttl: ttl, nowProvider: nowProvider}
}

func (s *ExportArtifactStore) Save(taskID uuid.UUID, sourcePath string, filename string) (ExportTaskResult, error) {
	token := fmt.Sprintf("exp_pdf_%s", taskID.String())
	targetPath := filepath.Join(s.rootDir, token+".pdf")
	if err := os.MkdirAll(s.rootDir, 0o755); err != nil {
		return ExportTaskResult{}, err
	}
	if err := os.Rename(sourcePath, targetPath); err != nil {
		return ExportTaskResult{}, err
	}
	expiresAt := s.nowProvider().Add(s.ttl).UTC()
	return ExportTaskResult{
		FileToken:   token,
		Filename:    filename,
		ContentType: "application/pdf",
		ExpiresAt:   expiresAt,
	}, nil
}

func (s *ExportArtifactStore) PathForToken(token string) string {
	return filepath.Join(s.rootDir, token+".pdf")
}
```

在 `backend/internal/domain/task/export_task.go` 补 payload/result 类型：

```go
type ExportPDFPayload struct {
	BlogID uuid.UUID `json:"blog_id"`
}

type ExportTaskResult struct {
	FileToken   string    `json:"file_token"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	ExpiresAt   time.Time `json:"expires_at"`
}
```

在 `backend/internal/domain/task/export_consumer.go` 编写 worker：

```go
type exportPDFService interface {
	ExportSeriesToPDF(ctx context.Context, blogID uuid.UUID, userID uuid.UUID) (string, string, error)
}

type exportTaskService interface {
	MarkRunning(ctx context.Context, taskID uuid.UUID) error
	MarkSucceeded(ctx context.Context, taskID uuid.UUID, result []byte) error
	MarkFailed(ctx context.Context, taskID uuid.UUID, message string) error
	IsCancelled(ctx context.Context, taskID uuid.UUID) (bool, error)
}

func (c *ExportConsumer) HandleExportRequested(ctx context.Context, message mq.ExportRequestedMessage) error {
	var payload ExportPDFPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, "invalid export payload")
	}
	if cancelled, err := c.tasks.IsCancelled(ctx, message.TaskID); err != nil {
		return err
	} else if cancelled {
		return nil
	}
	if err := c.tasks.MarkRunning(ctx, message.TaskID); err != nil {
		return err
	}
	pdfPath, filename, err := c.exporter.ExportSeriesToPDF(ctx, payload.BlogID, message.UserID)
	if err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, err.Error())
	}
	result, err := c.store.Save(message.TaskID, pdfPath, filename)
	if err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, err.Error())
	}
	body, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return c.tasks.MarkSucceeded(ctx, message.TaskID, body)
}
```

在 `backend/cmd/export-service/main.go` 启动 consumer：

```go
taskRepo := taskdomain.NewGormRepository(db.DB)
taskService := taskdomain.NewService(taskRepo, nil)
artifactStore := taskdomain.NewExportArtifactStore(
	envOrDefault("EXPORT_ARTIFACTS_DIR", "/app/export-artifacts"),
	15*time.Minute,
	time.Now,
)
exportConsumer := taskdomain.NewExportConsumer(taskService, pdfExporter, artifactStore)

stopConsumer, err := startExportTaskConsumer(
	signalContext,
	exportConsumer,
	envOrDefault("RABBITMQ_EXPORT_QUEUE", "inkwords.export"),
)
if err != nil {
	log.Printf("RabbitMQ export consumer initialization skipped: %v", err)
}
defer stopConsumer()
```

消费者启动函数可直接仿照 `parser-service` 的 `startParseTaskConsumer`，只需要把路由键改为：

```go
routingKey := mq.ExportRequestedMessage{}.RoutingKey()
consumerTag := "export-service-pdf-worker"
```

- [ ] **Step 4: 运行测试，确认转绿**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/task ./cmd/export-service -count=1
```

Expected: PASS，consumer 测试与 `export-service` 编译测试通过。

- [ ] **Step 5: 提交**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add backend/internal/domain/task/export_artifact_store.go \
  backend/internal/domain/task/export_consumer.go \
  backend/internal/domain/task/export_consumer_test.go \
  backend/cmd/export-service/main.go
git commit -m "feat(export): add export pdf worker and artifact store"
```

---

### Task 3: 实现 `core-api` 下载接口并通过共享目录提供 PDF

**Files:**
- Create: `backend/internal/domain/task/download_handler.go`
- Create: `backend/internal/domain/task/download_handler_test.go`
- Modify: `backend/internal/domain/task/handler.go`
- Modify: `backend/internal/domain/task/handler_test.go`
- Modify: `backend/internal/transport/http/v1/api/task.go`
- Modify: `backend/internal/transport/http/v1/routes.go`
- Modify: `backend/cmd/core-api/main.go`

- [ ] **Step 1: 先写下载接口失败测试**

在 `backend/internal/domain/task/download_handler_test.go` 先锁定成功下载、任务未完成与文件缺失：

```go
func TestHandler_DownloadTask_ServesPDF(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	taskID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	token := "exp_pdf_" + taskID.String()
	require.NoError(t, os.WriteFile(filepath.Join(dir, token+".pdf"), []byte("pdf"), 0o644))

	service := &fakeTaskHandlerService{
		getTaskResult: model.JobTask{
			ID:       taskID,
			TaskType: taskTypeExport,
			Status:   model.JobTaskStatusSucceeded,
			ResultJSON: datatypes.JSON([]byte(`{
				"file_token":"` + token + `",
				"filename":"系列标题.pdf",
				"content_type":"application/pdf",
				"expires_at":"2026-06-03T13:00:00Z"
			}`)),
		},
	}
	handler := NewHandler(service, dir)

	router := gin.New()
	router.GET("/api/v1/tasks/:id/download", func(c *gin.Context) {
		c.Set("user_id", uuid.MustParse("11111111-1111-1111-1111-111111111111"))
		handler.DownloadTask(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+taskID.String()+"/download", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "application/pdf", resp.Header().Get("Content-Type"))
	require.Contains(t, resp.Header().Get("Content-Disposition"), "系列标题.pdf")
	require.Equal(t, "pdf", resp.Body.String())
}

func TestHandler_DownloadTask_RejectsUnfinishedTask(t *testing.T) {
	service := &fakeTaskHandlerService{
		getTaskResult: model.JobTask{
			ID:       uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
			TaskType: taskTypeExport,
			Status:   model.JobTaskStatusRunning,
		},
	}
	handler := NewHandler(service, t.TempDir())
	// 断言 409
}
```

- [ ] **Step 2: 运行测试，确认当前失败**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/task -run 'TestHandler_DownloadTask' -count=1
```

Expected: FAIL，原因是 `DownloadTask` 尚未实现。

- [ ] **Step 3: 写最小实现（共享目录下载 + 成功后删除）**

将 `backend/internal/domain/task/handler.go` 的构造函数改成带下载目录：

```go
type Handler struct {
	service           taskService
	exportArtifactsDir string
}

func NewHandler(service taskService, exportArtifactsDir string) *Handler {
	return &Handler{
		service:            service,
		exportArtifactsDir: exportArtifactsDir,
	}
}
```

在新文件 `backend/internal/domain/task/download_handler.go` 中实现：

```go
func (h *Handler) DownloadTask(c *gin.Context) {
	taskID, ok := parseTaskID(c)
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	task, err := h.service.GetTask(c.Request.Context(), taskID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	if task.TaskType != taskTypeExport {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task is not downloadable"})
		return
	}
	if task.Status != model.JobTaskStatusSucceeded {
		c.JSON(http.StatusConflict, gin.H{"error": "task is not finished"})
		return
	}

	var result ExportTaskResult
	if err := json.Unmarshal(task.ResultJSON, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid task result"})
		return
	}
	if time.Now().UTC().After(result.ExpiresAt) {
		c.JSON(http.StatusNotFound, gin.H{"error": "download expired"})
		return
	}

	filePath := filepath.Join(h.exportArtifactsDir, result.FileToken+".pdf")
	f, err := os.Open(filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "download artifact missing"})
		return
	}
	defer f.Close()
	defer os.Remove(filePath)

	c.Header("Content-Type", result.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", result.Filename))
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, f)
}
```

在 `backend/cmd/core-api/main.go` 传入共享目录：

```go
taskHandler := taskdomain.NewHandler(
	taskService,
	envOrDefault("EXPORT_ARTIFACTS_DIR", "/app/export-artifacts"),
)
```

- [ ] **Step 4: 运行测试，确认转绿**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/task ./internal/transport/http/v1 ./cmd/core-api -count=1
```

Expected: PASS，下载接口和路由测试通过。

- [ ] **Step 5: 提交**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add backend/internal/domain/task/download_handler.go \
  backend/internal/domain/task/download_handler_test.go \
  backend/internal/domain/task/handler.go \
  backend/internal/domain/task/handler_test.go \
  backend/internal/transport/http/v1/api/task.go \
  backend/internal/transport/http/v1/routes.go \
  backend/cmd/core-api/main.go
git commit -m "feat(export): add task download endpoint"
```

---

### Task 4: 前端把“导出 PDF”按钮切到任务式流程

**Files:**
- Create: `frontend/src/services/exportTasks.ts`
- Create: `frontend/src/services/exportTasks.test.ts`
- Modify: `frontend/src/services/sidebarExport.ts`
- Modify: `frontend/src/services/sidebarExport.test.ts`
- Modify: `frontend/src/components/Sidebar.tsx`

- [ ] **Step 1: 先写前端失败测试**

在 `frontend/src/services/exportTasks.test.ts` 新建任务服务测试：

```ts
it('creates an export_pdf task and returns its stream url', async () => {
  const fetchImpl = vi.fn().mockResolvedValue({
    ok: true,
    status: 202,
    json: async () => ({
      task_id: 'task-export-1',
      status: 'queued',
      stream_url: '/api/v1/tasks/task-export-1/stream',
    }),
  })

  await expect(createExportTask('series-1', { fetchImpl, getToken: () => 'token-123' })).resolves.toEqual({
    task_id: 'task-export-1',
    status: 'queued',
    stream_url: '/api/v1/tasks/task-export-1/stream',
  })
})
```

在 `frontend/src/services/sidebarExport.test.ts` 把 PDF 导出改为任务模型：

```ts
it('creates export tasks and downloads the finished pdf for each selected series', async () => {
  const createExportTask = vi
    .fn()
    .mockResolvedValueOnce({ task_id: 'task-a', stream_url: '/api/v1/tasks/task-a/stream' })
    .mockResolvedValueOnce({ task_id: 'task-b', stream_url: '/api/v1/tasks/task-b/stream' })
  const waitForTaskCompletion = vi
    .fn()
    .mockResolvedValueOnce({ id: 'task-a', status: 'succeeded' })
    .mockResolvedValueOnce({ id: 'task-b', status: 'failed', error_message: '导出失败' })
  const downloadTaskArtifact = vi.fn().mockResolvedValue(undefined)

  const result = await exportSeriesPdfs(
    [createSeries({ id: 'series-a', title: '系列/导读' }), createSeries({ id: 'series-b', title: '系列B' })],
    { createExportTask, waitForTaskCompletion, downloadTaskArtifact },
  )

  expect(downloadTaskArtifact).toHaveBeenCalledWith('task-a', '系列-导读.pdf')
  expect(result).toEqual({
    succeededCount: 1,
    failed: [{ id: 'series-b', title: '系列B', message: '导出失败' }],
  })
})
```

- [ ] **Step 2: 运行测试，确认当前失败**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend
npm test -- --run src/services/exportTasks.test.ts src/services/sidebarExport.test.ts
```

Expected: FAIL，原因是 `exportTasks.ts` 和新的任务式依赖尚未实现。

- [ ] **Step 3: 写最小实现（create task + wait + download）**

在 `frontend/src/services/exportTasks.ts` 写入：

```ts
import { fetchEventSourceWithAuth } from './sse'

type ExportTaskResponse = {
  task_id: string
  status: string
  stream_url: string
}

type ExportTaskSnapshot = {
  id: string
  status: string
  error_message?: string
}

export async function createExportTask(
  blogID: string,
  dependencies: Pick<SidebarExportDependencies, 'fetchImpl' | 'getToken'> = {},
): Promise<ExportTaskResponse> {
  const fetchImpl = dependencies.fetchImpl ?? fetch
  const token = dependencies.getToken?.() ?? localStorage.getItem('token') ?? ''
  const response = await fetchImpl('/api/v1/tasks/export', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      kind: 'export_pdf',
      payload: { blog_id: blogID },
      idempotency_key: `export-pdf:${blogID}`,
    }),
  })
  if (!response.ok) {
    const data = await response.json().catch(() => ({}))
    throw new Error(data.message || data.error || '创建导出任务失败')
  }
  return response.json()
}

export async function waitForTaskCompletion(streamURL: string): Promise<void> {
  await fetchEventSourceWithAuth(streamURL, {
    method: 'GET',
    openWhenHidden: true,
    onmessage(msg) {
      if (msg.event === 'done') return
      if (msg.event === 'error') throw new Error(msg.data || 'PDF 导出失败')
    },
  })
}

export async function downloadTaskArtifact(taskID: string, filename: string): Promise<void> {
  const response = await fetch(`/api/v1/tasks/${taskID}/download`, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${localStorage.getItem('token') ?? ''}`,
    },
  })
  if (!response.ok) {
    const data = await response.json().catch(() => ({}))
    throw new Error(data.message || data.error || 'PDF 下载失败')
  }
  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}
```

在 `frontend/src/services/sidebarExport.ts` 改成任务流：

```ts
import { createExportTask, downloadTaskArtifact, waitForTaskCompletion } from './exportTasks'

export async function exportSeriesPdfs(
  seriesRoots: BlogNode[],
  dependencies: SidebarExportDependencies = {},
) {
  const failed: SidebarExportFailure[] = []
  let succeededCount = 0

  for (const series of seriesRoots) {
    try {
      const task = await (dependencies.createExportTask ?? createExportTask)(series.id, dependencies)
      await (dependencies.waitForTaskCompletion ?? waitForTaskCompletion)(task.stream_url)
      await (dependencies.downloadTaskArtifact ?? downloadTaskArtifact)(
        task.task_id,
        sanitizePdfFilename(series.title),
      )
      succeededCount += 1
    } catch (error) {
      failed.push({
        id: series.id,
        title: series.title,
        message: error instanceof Error ? error.message : '导出失败',
      })
    }
  }

  return { succeededCount, failed }
}
```

在 `frontend/src/components/Sidebar.tsx` 的成功提示保持中文：

```tsx
toast.success(`PDF 已生成，开始下载（成功 ${result.succeededCount} 个）`)
```

- [ ] **Step 4: 运行测试，确认转绿**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend
npm test -- --run src/services/exportTasks.test.ts src/services/sidebarExport.test.ts
npm run build
```

Expected: PASS，服务测试通过，前端构建无类型错误。

- [ ] **Step 5: 提交**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add frontend/src/services/exportTasks.ts \
  frontend/src/services/exportTasks.test.ts \
  frontend/src/services/sidebarExport.ts \
  frontend/src/services/sidebarExport.test.ts \
  frontend/src/components/Sidebar.tsx
git commit -m "feat(export): switch pdf export button to async task flow"
```

---

### Task 5: 配置共享导出目录、文档同步与 Docker 冒烟

**Files:**
- Modify: `docker-compose.yml`
- Modify: `backend/.env.example`
- Modify: `README.md`
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`

- [ ] **Step 1: 先写最小配置与文档检查**

在执行改动前，先确认 Compose 和文档都还没有 `RABBITMQ_EXPORT_QUEUE` 与 `EXPORT_ARTIFACTS_DIR`：

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
grep -n "RABBITMQ_EXPORT_QUEUE\|EXPORT_ARTIFACTS_DIR" docker-compose.yml backend/.env.example README.md .trae/documents/InkWords_API.md .trae/documents/InkWords_Architecture.md
```

Expected: 输出为空或仅命中已有计划文档，说明当前配置与文档尚未同步。

- [ ] **Step 2: 更新 Compose、环境变量和文档**

在 `docker-compose.yml` 增加共享卷和环境变量：

```yaml
services:
  core-api:
    environment:
      EXPORT_ARTIFACTS_DIR: /app/export-artifacts
      RABBITMQ_EXPORT_QUEUE: ${RABBITMQ_EXPORT_QUEUE:-inkwords.export}
    volumes:
      - export-artifacts:/app/export-artifacts:ro

  export-service:
    environment:
      EXPORT_ARTIFACTS_DIR: /app/export-artifacts
      RABBITMQ_EXPORT_QUEUE: ${RABBITMQ_EXPORT_QUEUE:-inkwords.export}
    volumes:
      - export-artifacts:/app/export-artifacts

volumes:
  export-artifacts:
```

在 `backend/.env.example` 追加：

```env
RABBITMQ_EXPORT_QUEUE=inkwords.export
EXPORT_ARTIFACTS_DIR=/app/export-artifacts
```

在 README 与项目文档中同步写入：

- `POST /api/v1/tasks/export`
- `GET /api/v1/tasks/:id/download`
- 当前 `export_pdf` 采用“共享导出目录 + 成功下载即删除 + 15 分钟 TTL”

- [ ] **Step 3: 运行完整验证**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./...

cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend
npm test -- --run src/services/exportTasks.test.ts src/services/sidebarExport.test.ts
npm run build

cd /Users/huangqijun/Documents/墨言博客助手/InkWords
docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build
docker compose --env-file backend/.env ps
curl -I http://localhost
```

Expected:

- Go tests PASS
- Vitest PASS
- frontend build PASS
- `docker compose ps` 显示 `core-api / export-service / frontend` 等服务 `Up`
- `curl -I http://localhost` 返回 `HTTP/1.1 200 OK`

- [ ] **Step 4: 做人工冒烟**

手动执行一次真实导出：

```text
1. 登录系统
2. 在侧边栏选择一个系列根节点
3. 点击“导出 PDF”
4. 观察前端提示“正在创建 PDF 导出任务...” -> “正在生成 PDF，请稍候...”
5. 等待浏览器下载 PDF
6. 再次访问相同任务的 /api/v1/tasks/:id/download，确认返回 404
```

Expected:

- 首次下载成功
- 二次下载因文件已删除返回 404
- 任务状态为 `succeeded`

- [ ] **Step 5: 提交**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add docker-compose.yml \
  backend/.env.example \
  README.md \
  .trae/documents/InkWords_API.md \
  .trae/documents/InkWords_Architecture.md \
  .trae/documents/InkWords_Development_Plan_and_Log.md \
  .trae/documents/InkWords_Conversation_Log.md
git commit -m "docs(export): document async pdf export task flow"
```

---

## Self-Review Checklist

### Spec coverage

- `POST /api/v1/tasks/export`：Task 1
- `export.requested` + worker：Task 2
- 共享导出目录与 TTL：Task 2 / Task 5
- `GET /api/v1/tasks/:id/download`：Task 3
- 前端 PDF 按钮任务化：Task 4
- Docker / docs / 验证：Task 5

### Placeholder scan

- 已检查：无 `TODO`、`TBD`、`implement later`
- 每个任务都包含具体代码片段、命令、预期输出和提交点

### Type consistency

- `task_type=export`
- `task_subtype=export_pdf`
- RabbitMQ routing key：`export.requested`
- 结果结构：`file_token` / `filename` / `content_type` / `expires_at`
- 下载路由：`GET /api/v1/tasks/:id/download`

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-06-03-export-pdf-async-task-implementation.md`.

Two execution options:

1. Subagent-Driven (recommended) - 我按任务拆成独立执行单元，逐步 review 后推进
2. Inline Execution - 我在当前会话里按这个 plan 直接执行

Which approach?
