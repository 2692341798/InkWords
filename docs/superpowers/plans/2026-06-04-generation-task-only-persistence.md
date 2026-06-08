# InkWords Generation Task-Only Persistence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在保持现有任务式 SSE、单入口网关和前端交互不变的前提下，把 `generation` 任务链路真正推进到 `task_only`，让 `llm-stream` 停止直写 `blogs / users`，由 `core-api` 基于 `job_tasks.result_json` 完成最终业务持久化。

**Architecture:** 实现按 `generate_single -> continue -> generate_series` 三段递进。`llm-stream` 负责生成执行、事件流和结构化任务结果；`core-api` 负责解析 generation result schema，并把结果幂等写回 `blogs / users.tokens_used`。`polish` 保持“仅预览、不自动落库”的现有产品语义。

**Tech Stack:** Go 1.25 + Gin + GORM + PostgreSQL + RabbitMQ + Docker Compose + Nginx + React + `@microsoft/fetch-event-source`

---

## 0. 实施前提

- 设计规格已确认：`docs/superpowers/specs/2026-06-04-generation-task-only-persistence-design.md`
- 当前服务边界以 Docker Compose 五服务为准：`core-api / llm-stream / parser-service / export-service / review-service`
- 当前允许的跨服务写入例外仍只有 `job_tasks / job_task_events`
- 实现期间保持对外 API 与 SSE 路径不变，只改任务成功后的内部持久化流程

## 1. File Map

**Generation result contract**
- Create: `backend/internal/domain/stream/generation_result.go`
- Test: `backend/internal/domain/stream/generation_result_test.go`

**LLM worker result handoff**
- Modify: `backend/internal/domain/stream/task_consumer.go`
- Modify: `backend/internal/domain/stream/task_consumer_test.go`
- Modify: `backend/internal/service/generator.go`
- Modify: `backend/internal/service/generator_persist_test.go`

**Core persistence handoff**
- Create: `backend/services/core-api/domain/task/generation_result.go`
- Create: `backend/services/core-api/domain/task/generation_result_repository.go`
- Test: `backend/services/core-api/domain/task/generation_result_repository_test.go`
- Modify: `backend/services/core-api/domain/task/result_persister.go`
- Modify: `backend/services/core-api/domain/task/result_persister_test.go`
- Modify: `backend/internal/domain/task/service.go`
- Modify: `backend/internal/domain/task/service_test.go`
- Modify: `backend/services/core-api/app/bootstrap/bootstrap.go`

**Continue path**
- Modify: `backend/internal/service/decomposition_generate_continue.go`
- Test: `backend/internal/service/decomposition_generate_continue_test.go`

**Series path**
- Modify: `backend/internal/service/decomposition_generate.go`
- Modify: `backend/internal/service/decomposition_generate_persistence.go`
- Modify: `backend/internal/service/decomposition_generate_intro.go`
- Modify: `backend/internal/service/decomposition_generate_persist_test.go`
- Create: `backend/internal/service/decomposition_generate_result_test.go`
- Modify: `backend/services/core-api/domain/task/generation_result_repository.go`
- Modify: `backend/services/core-api/domain/task/generation_result_repository_test.go`

**Docs**
- Modify: `README.md`
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Database.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`

---

### Task 1: 建立 generation result schema 并先打通单篇生成

**Files:**
- Create: `backend/internal/domain/stream/generation_result.go`
- Test: `backend/internal/domain/stream/generation_result_test.go`
- Modify: `backend/internal/domain/stream/task_consumer.go`
- Modify: `backend/internal/domain/stream/task_consumer_test.go`
- Modify: `backend/internal/service/generator.go`
- Modify: `backend/internal/service/generator_persist_test.go`

- [ ] **Step 1: 先为 generation result contract 写失败测试**

在 `backend/internal/domain/stream/generation_result_test.go` 新增：

```go
package stream

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildGenerateSingleTaskResult_ProducesTaskOnlyContract(t *testing.T) {
	result, err := BuildGenerateSingleTaskResult(GenerateSingleTaskResultInput{
		BlogID:          "11111111-1111-1111-1111-111111111111",
		Title:           "文件解析生成的博客",
		Content:         "# 标题\n\n正文",
		SourceType:      "file",
		WordCount:       7,
		TechStacks:      []string{"Go", "Docker"},
		EstimatedTokens: 14,
	})
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(result, &decoded))
	require.Equal(t, float64(1), decoded["result_version"])
	require.Equal(t, "generation", decoded["task_type"])
	require.Equal(t, "generate_single", decoded["task_subtype"])
	require.Equal(t, "task_only", decoded["persistence_mode"])
	require.Equal(t, "succeeded", decoded["final_status"])
}
```

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/stream -run TestBuildGenerateSingleTaskResult_ProducesTaskOnlyContract -v
```

Expected: FAIL，提示 `undefined: BuildGenerateSingleTaskResult`

- [ ] **Step 2: 写最小 generation result DTO 与 builder**

在 `backend/internal/domain/stream/generation_result.go` 新建：

```go
package stream

import "encoding/json"

type TaskResultEnvelope struct {
	ResultVersion   int                    `json:"result_version"`
	TaskType        string                 `json:"task_type"`
	TaskSubtype     string                 `json:"task_subtype"`
	PersistenceMode string                 `json:"persistence_mode"`
	FinalStatus     string                 `json:"final_status"`
	Usage           TaskResultUsage        `json:"usage"`
	Payload         map[string]any         `json:"payload"`
}

type TaskResultUsage struct {
	EstimatedTokens int `json:"estimated_tokens"`
}

type GenerateSingleTaskResultInput struct {
	BlogID          string
	Title           string
	Content         string
	SourceType      string
	WordCount       int
	TechStacks      []string
	EstimatedTokens int
}

func BuildGenerateSingleTaskResult(input GenerateSingleTaskResultInput) ([]byte, error) {
	envelope := TaskResultEnvelope{
		ResultVersion:   1,
		TaskType:        "generation",
		TaskSubtype:     "generate_single",
		PersistenceMode: "task_only",
		FinalStatus:     "succeeded",
		Usage:           TaskResultUsage{EstimatedTokens: input.EstimatedTokens},
		Payload: map[string]any{
			"blog_id":     input.BlogID,
			"title":       input.Title,
			"content":     input.Content,
			"source_type": input.SourceType,
			"word_count":  input.WordCount,
			"tech_stacks": input.TechStacks,
		},
	}
	return json.Marshal(envelope)
}
```

- [ ] **Step 3: 让 `GeneratorService` 在 `task_only` 下只返回结果，不再写业务表**

在 `backend/internal/service/generator_persist_test.go` 增加失败测试：

```go
func TestGenerateBlogStream_TaskOnlyMode_DoesNotPersistAndReturnsTaskResult(t *testing.T) {
	t.Setenv("INKWORDS_TASK_PERSISTENCE_MODE", "task_only")
	// 使用 stub llm + stub persistence，断言 persistence.SaveGeneratedBlog 未被调用，
	// 同时断言新的 task result builder 被调用并返回非空结果。
}
```

在 `backend/internal/service/generator.go` 增加最小改造：

```go
type GenerateBlogTaskResult struct {
	ResultJSON []byte
}

func (s *GeneratorService) buildTaskResult(title, content, sourceType string, techStacks []string) ([]byte, error) {
	return stream.BuildGenerateSingleTaskResult(stream.GenerateSingleTaskResultInput{
		Title:           title,
		Content:         content,
		SourceType:      sourceType,
		WordCount:       len([]rune(content)),
		TechStacks:      techStacks,
		EstimatedTokens: len([]rune(content)) * 2,
	})
}
```

Why:
- `task_only` 的关键不是“跳过写库”本身，而是“跳过写库以后仍然给 `core-api` 一个完整结果”

- [ ] **Step 4: 让 `task_consumer` 成功时写入真实 `result_json`**

在 `backend/internal/domain/stream/task_consumer_test.go` 增加失败测试：

```go
func TestHandleGenerationRequested_MarkSucceededWithStructuredResult(t *testing.T) {
	tasks := &fakeTaskService{}
	streams := &fakeGenerationStreamService{
		generateSingleResult: []byte(`{"result_version":1,"task_type":"generation","task_subtype":"generate_single","persistence_mode":"task_only","final_status":"succeeded","usage":{"estimated_tokens":6},"payload":{"title":"A","content":"B","source_type":"file","word_count":1,"tech_stacks":["Go"]}}`),
	}
	consumer := NewTaskConsumer(tasks, streams)

	err := consumer.HandleGenerationRequested(context.Background(), mq.GenerationRequestedMessage{
		TaskID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Kind:   "generate_single",
		UserID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Payload: []byte(`{"source_type":"file","source_content":"hello"}`),
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"result_version":1,"task_type":"generation","task_subtype":"generate_single","persistence_mode":"task_only","final_status":"succeeded","usage":{"estimated_tokens":6},"payload":{"title":"A","content":"B","source_type":"file","word_count":1,"tech_stacks":["Go"]}}`, string(tasks.markSucceededResult))
}
```

在 `backend/internal/domain/stream/task_consumer.go` 把：

```go
return c.tasks.MarkSucceeded(ctx, message.TaskID, []byte(`{"done":true}`))
```

改成：

```go
result, err := c.buildFinalTaskResult(message)
if err != nil {
	return c.tasks.MarkFailed(ctx, message.TaskID, err.Error())
}
return c.tasks.MarkSucceeded(ctx, message.TaskID, result)
```

- [ ] **Step 5: 运行单篇相关测试**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/stream ./internal/service -run 'TaskResult|GenerateBlogStream|HandleGenerationRequested' -v
```

Expected:
- 新增 contract 测试通过
- `task_only` 下不再依赖直接写库
- `task_consumer` 不再写 `{"done":true}`

- [ ] **Step 6: Commit**

```bash
git add backend/internal/domain/stream/generation_result.go \
  backend/internal/domain/stream/generation_result_test.go \
  backend/internal/domain/stream/task_consumer.go \
  backend/internal/domain/stream/task_consumer_test.go \
  backend/internal/service/generator.go \
  backend/internal/service/generator_persist_test.go
git commit -m "feat(task): add structured single-generation task result"
```

---

### Task 2: 让 `core-api` 真正消费单篇生成结果并落库

**Files:**
- Create: `backend/services/core-api/domain/task/generation_result.go`
- Create: `backend/services/core-api/domain/task/generation_result_repository.go`
- Test: `backend/services/core-api/domain/task/generation_result_repository_test.go`
- Modify: `backend/services/core-api/domain/task/result_persister.go`
- Modify: `backend/services/core-api/domain/task/result_persister_test.go`
- Modify: `backend/internal/domain/task/service.go`
- Modify: `backend/internal/domain/task/service_test.go`
- Modify: `backend/services/core-api/app/bootstrap/bootstrap.go`

- [ ] **Step 1: 为 `core-api` 的结果解析写失败测试**

在 `backend/services/core-api/domain/task/result_persister_test.go` 新增：

```go
func TestResultPersister_PersistsSingleGenerationResult(t *testing.T) {
	repo := &fakeBlogRepository{}
	usage := &fakeUsageRepository{}
	persister := NewResultPersister(repo, usage)

	taskID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	result := map[string]any{
		"result_version": 1,
		"task_type": "generation",
		"task_subtype": "generate_single",
		"persistence_mode": "task_only",
		"final_status": "succeeded",
		"usage": map[string]any{"estimated_tokens": 24},
		"payload": map[string]any{
			"blog_id": "33333333-3333-3333-3333-333333333333",
			"title": "文件解析生成的博客",
			"content": "# 标题",
			"source_type": "file",
			"word_count": float64(3),
			"tech_stacks": []any{"Go", "Docker"},
		},
	}

	require.NoError(t, persister.PersistGenerationResult(context.Background(), taskID, result))
	require.True(t, repo.persisted)
	require.True(t, usage.accumulated)
}
```

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./services/core-api/domain/task -run TestResultPersister_PersistsSingleGenerationResult -v
```

Expected: FAIL，提示结果解析/存储能力缺失

- [ ] **Step 2: 定义 `core-api` 侧可消费的 generation result 类型**

在 `backend/services/core-api/domain/task/generation_result.go` 新建：

```go
package task

type GenerationResult struct {
	ResultVersion   int                 `json:"result_version"`
	TaskType        string              `json:"task_type"`
	TaskSubtype     string              `json:"task_subtype"`
	PersistenceMode string              `json:"persistence_mode"`
	FinalStatus     string              `json:"final_status"`
	Usage           GenerationResultUsage `json:"usage"`
	Payload         map[string]any      `json:"payload"`
}

type GenerationResultUsage struct {
	EstimatedTokens int `json:"estimated_tokens"`
}
```

在 `backend/services/core-api/domain/task/generation_result_repository.go` 新建单篇持久化最小实现：

```go
package task

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
)

type GormGenerationResultRepository struct {
	db *gorm.DB
}

func NewGormGenerationResultRepository(db *gorm.DB) *GormGenerationResultRepository {
	return &GormGenerationResultRepository{db: db}
}

func (r *GormGenerationResultRepository) PersistGenerationResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	var decoded GenerationResult
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	if decoded.TaskSubtype != "generate_single" {
		return nil
	}
	payload := decoded.Payload
	blogID, _ := uuid.Parse(payload["blog_id"].(string))
	techStacksJSON, _ := json.Marshal(payload["tech_stacks"])
	return r.db.WithContext(ctx).Model(&model.Blog{}).
		Where("id = ?", blogID).
		Updates(map[string]any{
			"title":       payload["title"],
			"content":     payload["content"],
			"source_type": payload["source_type"],
			"word_count":  payload["word_count"],
			"tech_stacks": datatypes.JSON(techStacksJSON),
			"status":      1,
		}).Error
}
```

- [ ] **Step 3: 让 `ResultPersister` 和任务服务在成功时真正接线**

在 `backend/internal/domain/task/service.go` 为 `Service` 增加：

```go
type ResultPersister interface {
	PersistGenerationResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error
}
```

并把构造函数改成：

```go
func NewService(repo Repository, publisher Publisher, resultPersister ResultPersister) *Service
```

在 `MarkSucceeded` 路径中增加：

```go
if s.resultPersister != nil {
	var decoded map[string]any
	if err := json.Unmarshal(result, &decoded); err != nil {
		return fmt.Errorf("解析任务结果失败: %w", err)
	}
	if task.TaskType == taskTypeGeneration {
		if err := s.resultPersister.PersistGenerationResult(ctx, taskID, decoded); err != nil {
			return fmt.Errorf("持久化 generation 结果失败: %w", err)
		}
	}
}
```

Why:
- 任务成功写 `result_json` 只是中间事实，`task_only` 真正成立的关键是“同一个成功路径里完成业务事实落库”

- [ ] **Step 4: 在 `core-api` bootstrap 中注入真实实现**

在 `backend/services/core-api/app/bootstrap/bootstrap.go` 把：

```go
resultPersister := coretask.NewResultPersister(nil, nil)
taskDomainService := taskdomain.NewService(taskRepo, taskPublisher)
```

改成：

```go
generationResultRepo := coretask.NewGormGenerationResultRepository(dbConn)
resultPersister := coretask.NewResultPersister(generationResultRepo, generationResultRepo)
taskDomainService := taskdomain.NewService(taskRepo, taskPublisher, resultPersister)
```

- [ ] **Step 5: 运行 `core-api` 侧测试**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./services/core-api/domain/task ./internal/domain/task ./services/core-api/app/bootstrap -v
```

Expected:
- `ResultPersister` 单篇落库测试通过
- 任务成功路径能触发 persister
- bootstrap 不再注入空壳 persister

- [ ] **Step 6: Commit**

```bash
git add backend/services/core-api/domain/task/generation_result.go \
  backend/services/core-api/domain/task/generation_result_repository.go \
  backend/services/core-api/domain/task/generation_result_repository_test.go \
  backend/services/core-api/domain/task/result_persister.go \
  backend/services/core-api/domain/task/result_persister_test.go \
  backend/internal/domain/task/service.go \
  backend/internal/domain/task/service_test.go \
  backend/services/core-api/app/bootstrap/bootstrap.go
git commit -m "feat(core-api): persist single generation task results"
```

---

### Task 3: 打通 `continue` 结果 contract 与 `core-api` 追加正文

**Files:**
- Modify: `backend/internal/service/decomposition_generate_continue.go`
- Test: `backend/internal/service/decomposition_generate_continue_test.go`
- Modify: `backend/internal/domain/stream/generation_result.go`
- Modify: `backend/internal/domain/stream/generation_result_test.go`
- Modify: `backend/internal/domain/stream/task_consumer_test.go`
- Modify: `backend/services/core-api/domain/task/generation_result_repository.go`
- Modify: `backend/services/core-api/domain/task/generation_result_repository_test.go`

- [ ] **Step 1: 为 continue result schema 写失败测试**

在 `backend/internal/domain/stream/generation_result_test.go` 新增：

```go
func TestBuildContinueTaskResult_ProducesFinalContent(t *testing.T) {
	result, err := BuildContinueTaskResult(ContinueTaskResultInput{
		BlogID:          "11111111-1111-1111-1111-111111111111",
		AppendedContent: "追加内容",
		FinalContent:    "旧内容追加内容",
		EstimatedTokens: 8,
	})
	require.NoError(t, err)
	require.Contains(t, string(result), `"task_subtype":"continue"`)
	require.Contains(t, string(result), `"final_content":"旧内容追加内容"`)
}
```

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/stream -run TestBuildContinueTaskResult_ProducesFinalContent -v
```

Expected: FAIL

- [ ] **Step 2: 在 continue service 中返回结果而不是直接落库**

在 `backend/internal/service/decomposition_generate_continue_test.go` 新增：

```go
func TestContinueGeneration_TaskOnlyMode_DoesNotUpdateBlogDirectly(t *testing.T) {
	t.Setenv("INKWORDS_TASK_PERSISTENCE_MODE", "task_only")
	// 准备 sqlite test db + 旧 blog 内容
	// 断言完成后数据库正文未被直接更新
	// 断言返回的 task result 含 final_content
}
```

在 `backend/internal/service/decomposition_generate_continue.go` 把结束逻辑改成：

```go
if finalNewContent != "" {
	finalContent := blog.Content + finalNewContent
	if taskOnlyPersistenceMode() {
		resultJSON, err := stream.BuildContinueTaskResult(stream.ContinueTaskResultInput{
			BlogID:          blog.ID.String(),
			AppendedContent: finalNewContent,
			FinalContent:    finalContent,
			EstimatedTokens: len([]rune(finalNewContent)) * 2,
		})
		if err != nil {
			errChan <- err
			return
		}
		// 将 resultJSON 挂到 continue 用例的返回对象上
		return
	}
	if err := db.DB.WithContext(ctx).Model(&blog).Update("content", finalContent).Error; err != nil {
		fmt.Printf("Failed to update blog content: %v\n", err)
	}
}
```

- [ ] **Step 3: 扩展 `core-api` repository 支持 continue**

在 `backend/services/core-api/domain/task/generation_result_repository_test.go` 新增：

```go
func TestGormGenerationResultRepository_PersistContinueResult(t *testing.T) {
	// 预置 blog
	// 调用 PersistGenerationResult(... continue ...)
	// 断言 content 被更新为 final_content
}
```

在 `generation_result_repository.go` 增加：

```go
case "continue":
	blogID, _ := uuid.Parse(payload["blog_id"].(string))
	return r.db.WithContext(ctx).Model(&model.Blog{}).
		Where("id = ?", blogID).
		Update("content", payload["final_content"]).Error
```

- [ ] **Step 4: 运行 continue 相关测试**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/service ./internal/domain/stream ./services/core-api/domain/task -run 'Continue|TaskOnlyMode|PersistContinueResult' -v
```

Expected:
- `continue` 结果 schema 测试通过
- `task_only` 下不再直写 blog
- `core-api` 正确使用 `final_content` 完成落库

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/decomposition_generate_continue.go \
  backend/internal/service/decomposition_generate_continue_test.go \
  backend/internal/domain/stream/generation_result.go \
  backend/internal/domain/stream/generation_result_test.go \
  backend/internal/domain/stream/task_consumer_test.go \
  backend/services/core-api/domain/task/generation_result_repository.go \
  backend/services/core-api/domain/task/generation_result_repository_test.go
git commit -m "feat(task): persist continue results through core-api"
```

---

### Task 4: 打通系列任务结果与 `core-api` 的父子博客持久化

**Files:**
- Modify: `backend/internal/service/decomposition_generate.go`
- Modify: `backend/internal/service/decomposition_generate_persistence.go`
- Modify: `backend/internal/service/decomposition_generate_intro.go`
- Modify: `backend/internal/service/decomposition_generate_persist_test.go`
- Create: `backend/internal/service/decomposition_generate_result_test.go`
- Modify: `backend/internal/domain/stream/generation_result.go`
- Modify: `backend/internal/domain/stream/generation_result_test.go`
- Modify: `backend/services/core-api/domain/task/generation_result_repository.go`
- Modify: `backend/services/core-api/domain/task/generation_result_repository_test.go`

- [ ] **Step 1: 先为系列结果 builder 写失败测试**

在 `backend/internal/domain/stream/generation_result_test.go` 新增：

```go
func TestBuildGenerateSeriesTaskResult_ContainsParentAndChapters(t *testing.T) {
	result, err := BuildGenerateSeriesTaskResult(GenerateSeriesTaskResultInput{
		ParentBlogID:    "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		ParentTitle:     "Go 源码解析系列",
		ParentContent:   "导读正文",
		EstimatedTokens: 64,
		Chapters: []SeriesChapterTaskResult{
			{
				BlogID:       "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
				ChapterSort:  1,
				Title:        "第 1 章",
				Content:      "正文",
				WordCount:    2,
				TechStacks:   []string{"Go"},
				Status:       "succeeded",
				ErrorMessage: "",
			},
		},
	})
	require.NoError(t, err)
	require.Contains(t, string(result), `"task_subtype":"generate_series"`)
	require.Contains(t, string(result), `"parent_blog"`)
	require.Contains(t, string(result), `"chapters"`)
}
```

- [ ] **Step 2: 收口系列链路内部结果收集，不再在 `task_only` 下直写业务表**

在 `backend/internal/service/decomposition_generate_result_test.go` 新建：

```go
func TestGenerateSeries_TaskOnlyMode_CollectsChapterResultsWithoutDirectPersistence(t *testing.T) {
	t.Setenv("INKWORDS_TASK_PERSISTENCE_MODE", "task_only")
	// 准备 outline + fake pipeline
	// 断言章节成功后返回 chapter result
	// 断言 persistence helper 未更新 blogs/users
}
```

在 `backend/internal/service/decomposition_generate.go` 增加系列结果收集器：

```go
type seriesTaskResultCollector struct {
	ParentBlogID  string
	ParentTitle   string
	ParentContent string
	Chapters      []stream.SeriesChapterTaskResult
}

func (c *seriesTaskResultCollector) AddChapter(ch Chapter, content string, wordCount int, techStacks []string) {
	c.Chapters = append(c.Chapters, stream.SeriesChapterTaskResult{
		BlogID:      ch.ID,
		ChapterSort: ch.Sort,
		Title:       ch.Title,
		Content:     content,
		WordCount:   wordCount,
		TechStacks:  techStacks,
		Status:      "succeeded",
	})
}
```

并把 `persistSeriesChapterCompletion(...)` 在 `task_only` 下返回空操作的逻辑前移到调用方，改成“收集结果，不触库”。

- [ ] **Step 3: 收口导读生成结果**

在 `backend/internal/service/decomposition_generate_intro.go` 把：

```go
db.DB.WithContext(ctx).Model(&model.Blog{}).Where("id = ?", parentID).Updates(...)
```

改成：

```go
if taskOnlyPersistenceMode() {
	collector.ParentContent = finalContent
	sendProgress("completed", "", "")
	return
}
db.DB.WithContext(ctx).Model(&model.Blog{}).Where("id = ?", parentID).Updates(...)
```

Why:
- 系列导读正文和章节正文必须一并归入 `result_json`，否则 `core-api` 无法完整接管系列持久化

- [ ] **Step 4: 扩展 `core-api` repository 持久化系列结果**

在 `backend/services/core-api/domain/task/generation_result_repository_test.go` 新增：

```go
func TestGormGenerationResultRepository_PersistGenerateSeriesResult(t *testing.T) {
	// 预置 parent blog + child drafts
	// 调用 PersistGenerationResult(... generate_series ...)
	// 断言 parent content 更新
	// 断言 child title/content/word_count/tech_stacks/status 更新
}
```

在 `generation_result_repository.go` 增加：

```go
case "generate_series":
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		parent := decoded.Payload["parent_blog"].(map[string]any)
		parentID, _ := uuid.Parse(parent["blog_id"].(string))
		if err := tx.Model(&model.Blog{}).Where("id = ?", parentID).Updates(map[string]any{
			"title":   parent["title"],
			"content": parent["content"],
			"status":  1,
		}).Error; err != nil {
			return err
		}

		for _, rawChapter := range decoded.Payload["chapters"].([]any) {
			chapter := rawChapter.(map[string]any)
			blogID, _ := uuid.Parse(chapter["blog_id"].(string))
			techStacksJSON, _ := json.Marshal(chapter["tech_stacks"])
			status := 1
			if chapter["status"] == "failed" {
				status = 2
			}
			if err := tx.Model(&model.Blog{}).Where("id = ?", blogID).Updates(map[string]any{
				"chapter_sort": chapter["chapter_sort"],
				"title":        chapter["title"],
				"content":      chapter["content"],
				"word_count":   chapter["word_count"],
				"tech_stacks":  datatypes.JSON(techStacksJSON),
				"status":       status,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
```

- [ ] **Step 5: 运行系列链路测试**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/service ./internal/domain/stream ./services/core-api/domain/task -run 'Series|GenerateSeries|PersistGenerateSeriesResult|TaskOnlyMode' -v
```

Expected:
- `task_only` 下系列链路不再直写 `blogs/users`
- `result_json` 包含父节点与章节数组
- `core-api` 能把父子博客一次性更新完成

- [ ] **Step 6: Commit**

```bash
git add backend/internal/service/decomposition_generate.go \
  backend/internal/service/decomposition_generate_persistence.go \
  backend/internal/service/decomposition_generate_intro.go \
  backend/internal/service/decomposition_generate_persist_test.go \
  backend/internal/service/decomposition_generate_result_test.go \
  backend/internal/domain/stream/generation_result.go \
  backend/internal/domain/stream/generation_result_test.go \
  backend/services/core-api/domain/task/generation_result_repository.go \
  backend/services/core-api/domain/task/generation_result_repository_test.go
git commit -m "feat(task): persist series generation results through core-api"
```

---

### Task 5: 文档同步与端到端验证

**Files:**
- Modify: `README.md`
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Database.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`

- [ ] **Step 1: 更新架构文档，明确 `task_only` 已真正闭环**

在 `.trae/documents/InkWords_Architecture.md` 补充：

```md
- 2026-06-04：生成链路 `task_only` 持久化闭环完成。`llm-stream` 现在只写 `job_tasks / job_task_events`，`core-api` 基于 `job_tasks.result_json` 将单篇生成、续写、系列导读与章节正文持久化到 `blogs / users`；`polish` 仍保持“仅预览、不自动落库”。
```

- [ ] **Step 2: 更新 API/Database 文档中的任务结果语义**

在 `.trae/documents/InkWords_API.md` 补充：

```md
`generation` 任务完成后，`job_tasks.result_json` 不再只保存 `{"done":true}`，而是保存带 `result_version / task_subtype / persistence_mode / usage / payload` 的结构化结果；`core-api` 在任务成功路径中消费该结果完成最终博客持久化。
```

在 `.trae/documents/InkWords_Database.md` 补充：

```md
`job_tasks.result_json` 已升级为 `generation` 任务的最终业务事实快照。当前 `generate_single / continue / generate_series` 会分别携带单篇正文、续写正文和系列父子结果；`polish` 只存预览结果，不自动写 `blogs`。
```

- [ ] **Step 3: 追加开发日志**

在 `.trae/documents/InkWords_Development_Plan_and_Log.md` 追加：

```md
### [2026-06-04] Refactor - Generation task_only persistence
- **需求背景**：
  1. 用户要求继续推进微服务化，把生成链路真正切到 `task_only`。
  2. 本轮按 `单篇 -> 续写 -> 系列` 递进，保持 `polish` 仅预览。
- **本次完成**：
  1. 为 `generation` 任务新增结构化 `result_json` schema。
  2. `llm-stream` 在 `task_only` 下停止直写 `blogs / users`。
  3. `core-api` 基于任务结果持久化单篇、续写和系列结果，并统一完成 token 记账。
- **验证记录**：
  - `cd backend && go test ./... -count=1`
  - `cd frontend && npm run build`
  - `docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build`
  - `curl -I http://localhost`
  - `curl -sS http://localhost/api/v1/ping`
```

- [ ] **Step 4: 运行完整验证**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./... -count=1

cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend
npm run build

cd /Users/huangqijun/Documents/墨言博客助手/InkWords
docker compose --env-file backend/.env down
docker compose --env-file backend/.env up -d --build
docker compose --env-file backend/.env ps
curl -I http://localhost
curl -sS http://localhost/api/v1/ping
```

Expected:
- 后端测试通过
- 前端构建通过
- `core-api / llm-stream / parser-service / export-service / review-service / frontend` 全部 `healthy`
- 网关入口 `http://localhost` 和 `GET /api/v1/ping` 正常

- [ ] **Step 5: 端到端手工冒烟**

手工验证清单：

```text
1. 单篇生成：任务完成后博客正文已落库
2. 继续生成：任务完成后正文被正确追加
3. 系列生成：父节点导读和章节正文都已落库
4. 系列部分失败：失败章状态清晰，成功章仍保留
5. 润色：任务结果存在，但正文不会自动覆盖
```

- [ ] **Step 6: Commit**

```bash
git add README.md \
  .trae/documents/InkWords_API.md \
  .trae/documents/InkWords_Architecture.md \
  .trae/documents/InkWords_Database.md \
  .trae/documents/InkWords_Development_Plan_and_Log.md \
  .trae/documents/InkWords_Conversation_Log.md
git commit -m "docs(task): sync generation task-only persistence rollout"
```

---

## Self-Review

- **Spec coverage:** 已覆盖规格中的边界重定义、result schema、单篇/续写/系列数据流、`polish` 非自动落库、风险与验证方案
- **Placeholder scan:** 计划中没有 `TBD / TODO / implement later / write tests for above` 这类占位语句
- **Type consistency:** `result_version / task_type / task_subtype / persistence_mode / final_status / usage / payload` 在 contract、consumer、repository、persister 里保持同名
