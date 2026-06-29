package stream

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

// fakeTaskService 实现 taskService 接口，用于特征化测试。
type fakeTaskService struct {
	markRunningCalled  bool
	appendEvents       []AppendEventInput
	lastStatus         TaskStatus
	lastResult         []byte
	lastErrorMessage   string
	cancelled          bool
	cancelAfterNCalls  int
	isCancelledCallCnt int
}

func (f *fakeTaskService) MarkRunning(_ context.Context, _ uuid.UUID) error {
	f.markRunningCalled = true
	f.lastStatus = TaskStatusRunning
	return nil
}

func (f *fakeTaskService) AppendEvent(_ context.Context, _ uuid.UUID, input AppendEventInput) error {
	f.appendEvents = append(f.appendEvents, input)
	if input.Status != "" {
		f.lastStatus = input.Status
	}
	return nil
}

func (f *fakeTaskService) MarkSucceeded(_ context.Context, _ uuid.UUID, result []byte) error {
	f.lastStatus = TaskStatusSucceeded
	f.lastResult = append([]byte(nil), result...)
	return nil
}

func (f *fakeTaskService) MarkFailed(_ context.Context, _ uuid.UUID, message string) error {
	f.lastStatus = TaskStatusFailed
	f.lastErrorMessage = message
	return nil
}

func (f *fakeTaskService) IsCancelled(_ context.Context, _ uuid.UUID) (bool, error) {
	f.isCancelledCallCnt++
	if f.cancelAfterNCalls > 0 && f.isCancelledCallCnt >= f.cancelAfterNCalls {
		return true, nil
	}
	return f.cancelled, nil
}

// fakeStreamService 实现 generationStreamService 接口，用于特征化测试。
type fakeStreamService struct {
	generateFunc                      func(ctx context.Context, userID uuid.UUID, req GenerateRequest, chunkChan chan<- string, errChan chan<- error)
	continueFunc                      func(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, chunkChan chan<- string, errChan chan<- error)
	polishFunc                        func(ctx context.Context, req PolishRequest, chunkChan chan<- string, errChan chan<- error)
	buildSingleResultFunc             func(ctx context.Context, req GenerateRequest, content string) ([]byte, error)
	buildSeriesResultFunc             func(ctx context.Context, req GenerateRequest) ([]byte, error)
	buildContinueResultFunc           func(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, appendedContent string) ([]byte, error)
	lastGenerateReq                   GenerateRequest
	lastContinueBlogID                uuid.UUID
	lastPolishReq                     PolishRequest
}

func (f *fakeStreamService) Generate(ctx context.Context, userID uuid.UUID, req GenerateRequest, chunkChan chan<- string, errChan chan<- error) {
	f.lastGenerateReq = req
	if f.generateFunc != nil {
		f.generateFunc(ctx, userID, req, chunkChan, errChan)
		return
	}
	close(chunkChan)
	close(errChan)
}

func (f *fakeStreamService) Continue(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, chunkChan chan<- string, errChan chan<- error) {
	f.lastContinueBlogID = blogID
	if f.continueFunc != nil {
		f.continueFunc(ctx, userID, blogID, chunkChan, errChan)
		return
	}
	close(chunkChan)
	close(errChan)
}

func (f *fakeStreamService) Polish(ctx context.Context, req PolishRequest, chunkChan chan<- string, errChan chan<- error) {
	f.lastPolishReq = req
	if f.polishFunc != nil {
		f.polishFunc(ctx, req, chunkChan, errChan)
		return
	}
	close(chunkChan)
	close(errChan)
}

func (f *fakeStreamService) BuildGenerateSingleTaskResult(ctx context.Context, req GenerateRequest, content string) ([]byte, error) {
	if f.buildSingleResultFunc != nil {
		return f.buildSingleResultFunc(ctx, req, content)
	}
	return nil, nil
}

func (f *fakeStreamService) BuildGenerateSeriesTaskResult(ctx context.Context, req GenerateRequest) ([]byte, error) {
	if f.buildSeriesResultFunc != nil {
		return f.buildSeriesResultFunc(ctx, req)
	}
	return nil, nil
}

func (f *fakeStreamService) BuildContinueTaskResult(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, appendedContent string) ([]byte, error) {
	if f.buildContinueResultFunc != nil {
		return f.buildContinueResultFunc(ctx, userID, blogID, appendedContent)
	}
	return nil, nil
}

func newTestTaskID() uuid.UUID {
	return uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
}

func newTestUserID() uuid.UUID {
	return uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
}

// 场景 1：单篇生成 — 验证 GenerateSingle 的流式输出和错误通道
func TestGenerateSingle_StreamOutputAndErrorChannel(t *testing.T) {
	tasks := &fakeTaskService{}
	streams := &fakeStreamService{
		generateFunc: func(ctx context.Context, userID uuid.UUID, req GenerateRequest, chunkChan chan<- string, errChan chan<- error) {
			require.Equal(t, "hello world", req.SourceContent)
			require.Equal(t, "file", req.SourceType)
			chunkChan <- "段落一"
			chunkChan <- "段落二"
			close(chunkChan)
			close(errChan)
		},
		buildSingleResultFunc: func(ctx context.Context, req GenerateRequest, content string) ([]byte, error) {
			require.Equal(t, "段落一段落二", content)
			require.Equal(t, "file", req.SourceType)
			return json.Marshal(TaskResultEnvelope{
				ResultVersion:   1,
				TaskType:        "generation",
				TaskSubtype:     "generate_single",
				PersistenceMode: "task_only",
				FinalStatus:     "succeeded",
				Usage:           TaskResultUsage{EstimatedTokens: 8, PromptTokens: 100, CompletionTokens: 200},
				Payload:         map[string]any{"content": content},
			})
		},
	}

	consumer := NewTaskConsumer(tasks, streams)
	message := sharedrabbitmq.GenerationRequestedMessage{
		TaskID:  newTestTaskID(),
		Kind:    "generate_single",
		UserID:  newTestUserID(),
		Payload: json.RawMessage(`{"source_type":"file","source_content":"hello world","scenario_mode":"ebook_interpretation"}`),
	}

	err := consumer.HandleGenerationRequested(context.Background(), message)
	require.NoError(t, err)

	assert.True(t, tasks.markRunningCalled, "MarkRunning 应被调用")
	assert.Equal(t, TaskStatusSucceeded, tasks.lastStatus)

	require.Len(t, tasks.appendEvents, 2, "应追加 2 个 chunk 事件")
	assert.Contains(t, string(tasks.appendEvents[0].Payload), "段落一")
	assert.Contains(t, string(tasks.appendEvents[1].Payload), "段落二")

	require.NotNil(t, tasks.lastResult)
	assert.Contains(t, string(tasks.lastResult), `"estimated_tokens":8`)
	assert.Contains(t, string(tasks.lastResult), `"prompt_tokens":100`)
	assert.Contains(t, string(tasks.lastResult), `"completion_tokens":200`)
}

// 场景 2：系列生成 — 验证 GenerateSeries 的章节分配和进度推送
func TestGenerateSeries_ChapterAssignmentAndProgressPush(t *testing.T) {
	tasks := &fakeTaskService{}
	streams := &fakeStreamService{
		generateFunc: func(ctx context.Context, userID uuid.UUID, req GenerateRequest, chunkChan chan<- string, errChan chan<- error) {
			require.NotEmpty(t, req.ParentID, "系列生成必须自动生成 ParentID")
			require.Equal(t, "Go 源码解析系列", req.SeriesTitle)
			require.Len(t, req.Outline, 2)

			chunkChan <- `{"status":"running","chapter_sort":1,"title":"第 1 章：引言"}`
			chunkChan <- `{"status":"completed","chapter_sort":1,"title":"第 1 章：引言"}`
			chunkChan <- `{"status":"running","chapter_sort":2,"title":"第 2 章：核心架构"}`
			chunkChan <- `{"status":"completed","chapter_sort":2,"title":"第 2 章：核心架构"}`
			close(chunkChan)
			close(errChan)
		},
		buildSeriesResultFunc: func(ctx context.Context, req GenerateRequest) ([]byte, error) {
			require.NotEmpty(t, req.ParentID)
			return json.Marshal(TaskResultEnvelope{
				ResultVersion:   1,
				TaskType:        "generation",
				TaskSubtype:     "generate_series",
				PersistenceMode: "task_only",
				FinalStatus:     "succeeded",
				Usage:           TaskResultUsage{EstimatedTokens: 256},
				Payload: map[string]any{
					"parent_blog": SeriesParentTaskResult{
						BlogID:  req.ParentID,
						Title:   req.SeriesTitle,
						Content: "导读正文",
					},
					"chapters": []SeriesChapterTaskResult{
						{BlogID: "ch1", ChapterSort: 1, Title: "第 1 章", Content: "正文1", Status: "succeeded"},
						{BlogID: "ch2", ChapterSort: 2, Title: "第 2 章", Content: "正文2", Status: "succeeded"},
					},
				},
			})
		},
	}

	consumer := NewTaskConsumer(tasks, streams)
	message := sharedrabbitmq.GenerationRequestedMessage{
		TaskID: newTestTaskID(),
		Kind:   "generate_series",
		UserID: newTestUserID(),
		Payload: json.RawMessage(`{
			"source_type":"file",
			"source_content":"Go 源码阅读笔记",
			"series_title":"Go 源码解析系列",
			"outline":[
				{"title":"第 1 章：引言","summary":"简介","sort":1,"files":[]},
				{"title":"第 2 章：核心架构","summary":"架构","sort":2,"files":[]}
			]
		}`),
	}

	err := consumer.HandleGenerationRequested(context.Background(), message)
	require.NoError(t, err)

	assert.True(t, tasks.markRunningCalled)
	assert.Equal(t, TaskStatusSucceeded, tasks.lastStatus)

	require.Len(t, tasks.appendEvents, 4, "应追加 4 个章节进度事件")

	require.NotNil(t, tasks.lastResult)
	var envelope TaskResultEnvelope
	require.NoError(t, json.Unmarshal(tasks.lastResult, &envelope))
	assert.Equal(t, "generate_series", envelope.TaskSubtype)
	assert.Equal(t, 256, envelope.Usage.EstimatedTokens)

	payload := envelope.Payload
	assert.NotNil(t, payload["parent_blog"])
	assert.NotNil(t, payload["chapters"])
}

// 场景 3：Continue — 验证续写请求的正确路由
func TestContinue_RequestRouting(t *testing.T) {
	blogID := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	tasks := &fakeTaskService{}
	streams := &fakeStreamService{
		continueFunc: func(ctx context.Context, userID uuid.UUID, bid uuid.UUID, chunkChan chan<- string, errChan chan<- error) {
			require.Equal(t, blogID, bid, "续写应路由到正确的博客 ID")
			chunkChan <- "续写内容片段"
			close(chunkChan)
			close(errChan)
		},
		buildContinueResultFunc: func(ctx context.Context, userID uuid.UUID, bid uuid.UUID, appendedContent string) ([]byte, error) {
			require.Equal(t, blogID, bid)
			require.Equal(t, "续写内容片段", appendedContent)
			return json.Marshal(TaskResultEnvelope{
				ResultVersion:   1,
				TaskType:        "generation",
				TaskSubtype:     "continue",
				PersistenceMode: "task_only",
				FinalStatus:     "succeeded",
				Usage:           TaskResultUsage{EstimatedTokens: 12, PromptTokens: 50, CompletionTokens: 80},
				Payload: map[string]any{
					"blog_id":          bid.String(),
					"appended_content": appendedContent,
					"final_content":    "旧内容" + appendedContent,
				},
			})
		},
	}

	consumer := NewTaskConsumer(tasks, streams)
	message := sharedrabbitmq.GenerationRequestedMessage{
		TaskID:  newTestTaskID(),
		Kind:    "continue",
		UserID:  newTestUserID(),
		Payload: json.RawMessage(`{"blog_id":"eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"}`),
	}

	err := consumer.HandleGenerationRequested(context.Background(), message)
	require.NoError(t, err)

	assert.True(t, tasks.markRunningCalled)
	assert.Equal(t, TaskStatusSucceeded, tasks.lastStatus)

	require.Len(t, tasks.appendEvents, 1)
	assert.Contains(t, string(tasks.appendEvents[0].Payload), "续写内容片段")

	require.NotNil(t, tasks.lastResult)
	assert.Contains(t, string(tasks.lastResult), `"task_subtype":"continue"`)
	assert.Contains(t, string(tasks.lastResult), `"estimated_tokens":12`)
	assert.Contains(t, string(tasks.lastResult), `"final_content":"旧内容续写内容片段"`)
}

// 场景 4：Polish — 验证润色请求的 payload 构造
func TestPolish_PayloadConstruction(t *testing.T) {
	tasks := &fakeTaskService{}
	streams := &fakeStreamService{
		polishFunc: func(ctx context.Context, req PolishRequest, chunkChan chan<- string, errChan chan<- error) {
			require.Equal(t, "旧标题", req.Title)
			require.Equal(t, "原正文内容需要润色", req.Content)
			chunkChan <- "润色后的段落一"
			chunkChan <- "润色后的段落二"
			close(chunkChan)
			close(errChan)
		},
	}

	consumer := NewTaskConsumer(tasks, streams)
	message := sharedrabbitmq.GenerationRequestedMessage{
		TaskID: newTestTaskID(),
		Kind:   "polish",
		UserID: newTestUserID(),
		Payload: json.RawMessage(`{
			"blog_id":"ffffffff-ffff-ffff-ffff-ffffffffffff",
			"title":"旧标题",
			"content":"原正文内容需要润色"
		}`),
	}

	err := consumer.HandleGenerationRequested(context.Background(), message)
	require.NoError(t, err)

	assert.True(t, tasks.markRunningCalled)
	assert.Equal(t, TaskStatusSucceeded, tasks.lastStatus)

	require.Len(t, tasks.appendEvents, 2)
	assert.Contains(t, string(tasks.appendEvents[0].Payload), "润色后的段落一")
	assert.Contains(t, string(tasks.appendEvents[1].Payload), "润色后的段落二")

	assert.Equal(t, "旧标题", streams.lastPolishReq.Title)
	assert.Equal(t, "原正文内容需要润色", streams.lastPolishReq.Content)
}

// 场景 5：取消 — 验证 context 取消时立即停止生成
func TestCancel_ContextCancellationStopsGeneration(t *testing.T) {
	tasks := &fakeTaskService{
		cancelAfterNCalls: 2, // 第 2 次 IsCancelled 调用返回 true（第 1 次在入口检查，第 2 次在 watcher 轮询时）
	}
	streams := &fakeStreamService{
		generateFunc: func(ctx context.Context, userID uuid.UUID, req GenerateRequest, chunkChan chan<- string, errChan chan<- error) {
			chunkChan <- "初始片段"
			<-ctx.Done()
		},
	}

	consumer := &TaskConsumer{
		tasks:                    tasks,
		streams:                  streams,
		cancellationPollInterval: 10 * time.Millisecond,
	}

	message := sharedrabbitmq.GenerationRequestedMessage{
		TaskID:  newTestTaskID(),
		Kind:    "generate_single",
		UserID:  newTestUserID(),
		Payload: json.RawMessage(`{"source_type":"file","source_content":"content"}`),
	}

	err := consumer.HandleGenerationRequested(context.Background(), message)
	require.NoError(t, err, "取消应返回 nil（成功取消）")

	assert.True(t, tasks.markRunningCalled)

	require.GreaterOrEqual(t, len(tasks.appendEvents), 1)
	assert.Contains(t, string(tasks.appendEvents[0].Payload), "初始片段")

	assert.NotEqual(t, TaskStatusSucceeded, tasks.lastStatus,
		"取消的任务不应标记为 Succeeded")
}

// 场景 5b：取消 — 入口即已取消的任务直接返回
func TestCancel_TaskAlreadyCancelledBeforeRunning(t *testing.T) {
	tasks := &fakeTaskService{
		cancelled: true,
	}
	streams := &fakeStreamService{}

	consumer := NewTaskConsumer(tasks, streams)
	message := sharedrabbitmq.GenerationRequestedMessage{
		TaskID:  newTestTaskID(),
		Kind:    "generate_single",
		UserID:  newTestUserID(),
		Payload: json.RawMessage(`{"source_type":"file","source_content":"content"}`),
	}

	err := consumer.HandleGenerationRequested(context.Background(), message)
	require.NoError(t, err, "已取消的任务应直接返回 nil")

	assert.False(t, tasks.markRunningCalled, "已取消的任务不应调用 MarkRunning")
	assert.Empty(t, tasks.appendEvents, "已取消的任务不应追加事件")
}

// 场景 6：Task result building — 验证 BuildGenerateSingleTaskResult 的 usage 聚合
func TestBuildGenerateSingleTaskResult_UsageAggregation(t *testing.T) {
	t.Run("完整 usage 字段", func(t *testing.T) {
		result, err := BuildGenerateSingleTaskResult(GenerateSingleTaskResultInput{
			BlogID:     "11111111-1111-1111-1111-111111111111",
			Title:      "Go 并发编程实战",
			Content:    "# 标题\n\n这是正文内容",
			SourceType: "file",
			WordCount:  10,
			TechStacks: []string{"Go", "Docker", "Kubernetes"},
			EstimatedTokens: 20,
			Usage: TaskResultUsage{
				EstimatedTokens:       30,
				PromptTokens:          150,
				CompletionTokens:      500,
				PromptCacheHitTokens:  100,
				PromptCacheMissTokens: 50,
			},
		})
		require.NoError(t, err)

		var envelope TaskResultEnvelope
		require.NoError(t, json.Unmarshal(result, &envelope))

		assert.Equal(t, 1, envelope.ResultVersion)
		assert.Equal(t, "generation", envelope.TaskType)
		assert.Equal(t, "generate_single", envelope.TaskSubtype)
		assert.Equal(t, "task_only", envelope.PersistenceMode)
		assert.Equal(t, "succeeded", envelope.FinalStatus)

		assert.Equal(t, 30, envelope.Usage.EstimatedTokens,
			"EstimatedTokens 应使用 Usage 中的显式值")
		assert.Equal(t, 150, envelope.Usage.PromptTokens)
		assert.Equal(t, 500, envelope.Usage.CompletionTokens)
		assert.Equal(t, 100, envelope.Usage.PromptCacheHitTokens)
		assert.Equal(t, 50, envelope.Usage.PromptCacheMissTokens)

		assert.Equal(t, "11111111-1111-1111-1111-111111111111", envelope.Payload["blog_id"])
		assert.Equal(t, "Go 并发编程实战", envelope.Payload["title"])
		assert.Equal(t, "# 标题\n\n这是正文内容", envelope.Payload["content"])
		assert.Equal(t, "file", envelope.Payload["source_type"])
		assert.Equal(t, float64(10), envelope.Payload["word_count"])
	})

	t.Run("Usage.EstimatedTokens 为零时回退到 input.EstimatedTokens", func(t *testing.T) {
		result, err := BuildGenerateSingleTaskResult(GenerateSingleTaskResultInput{
			BlogID:          "22222222-2222-2222-2222-222222222222",
			Title:           "回退测试",
			Content:         "正文",
			SourceType:      "git",
			WordCount:       5,
			TechStacks:      []string{"Python"},
			EstimatedTokens: 40,
			Usage: TaskResultUsage{
				EstimatedTokens: 0, // 零值，应回退
				PromptTokens:    200,
				CompletionTokens: 300,
			},
		})
		require.NoError(t, err)

		var envelope TaskResultEnvelope
		require.NoError(t, json.Unmarshal(result, &envelope))

		assert.Equal(t, 40, envelope.Usage.EstimatedTokens,
			"Usage.EstimatedTokens 为零时应回退到 input.EstimatedTokens")
		assert.Equal(t, 200, envelope.Usage.PromptTokens)
		assert.Equal(t, 300, envelope.Usage.CompletionTokens)
	})

	t.Run("Usage 全为零时 EstimatedTokens 回退", func(t *testing.T) {
		result, err := BuildGenerateSingleTaskResult(GenerateSingleTaskResultInput{
			BlogID:          "33333333-3333-3333-3333-333333333333",
			Title:           "全零测试",
			Content:         "正文",
			SourceType:      "file",
			WordCount:       3,
			TechStacks:      nil,
			EstimatedTokens: 100,
			Usage:           TaskResultUsage{},
		})
		require.NoError(t, err)

		var envelope TaskResultEnvelope
		require.NoError(t, json.Unmarshal(result, &envelope))

		assert.Equal(t, 100, envelope.Usage.EstimatedTokens,
			"Usage 全为零时应回退到 input.EstimatedTokens")
		assert.Equal(t, 0, envelope.Usage.PromptTokens)
		assert.Equal(t, 0, envelope.Usage.CompletionTokens)
	})
}