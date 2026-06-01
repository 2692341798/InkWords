# Review Targeted Questioning Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把知识漫游复习升级为基于文章内容的结构化追问，并向前端返回清晰的命中点、遗漏点与下一步补充建议。

**Architecture:** 后端在创建 session 时把原始笔记提炼为结构化 `SessionOutline`，后续提问、提示与反馈全部围绕这份快照驱动。前端不再只展示一段笼统鼓励文案，而是消费结构化反馈并渲染“当前轮目标 / 你答到的点 / 还漏掉的点 / 下一步建议”。

**Tech Stack:** Go 1.21 + Gin + GORM、React 18 + TypeScript + Zustand + Vitest

---

### Task 1: 扩展后端会话快照与返回类型

**Files:**
- Modify: `backend/internal/domain/review/dto.go`
- Modify: `backend/internal/domain/review/session_builder.go`
- Test: `backend/internal/domain/review/session_service_test.go`

- [ ] **Step 1: 先写失败测试，锁定结构化快照输出**

```go
func TestService_CreateSession_BuildsStructuredSnapshot(t *testing.T) {
	t.Parallel()

	svc := newTestReviewServiceWithNotes([]ReviewNote{{
		NotePath:      "wiki/concepts/并发控制与速率限制.md",
		Title:         "并发控制与速率限制",
		SourceTitle:   "InkWords 内容生成平台架构解析系列",
		Body:          "并发控制用来限制同时执行的任务数量，避免资源打满。信号量负责发放并回收执行许可。速率限制负责控制单位时间内的请求频率，避免 API 被突发流量冲垮。实际落地时，通常会把并发上限、队列等待和重试退避组合起来。",
		PreferredMode: model.ReviewModeDetailedQA,
	}})

	resp, err := svc.CreateSession(context.Background(), uuid.New(), CreateSessionRequest{
		NotePath:  "wiki/concepts/并发控制与速率限制.md",
		Mode:      model.ReviewModeDetailedQA,
		EntryType: model.ReviewEntryTypeManualSelect,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.SessionOutline.MainQuestion)
	require.NotEmpty(t, resp.SessionOutline.Checkpoints)
	require.NotEmpty(t, resp.CurrentRoundGoal)
	require.Contains(t, resp.NextQuestion, "并发控制")
}
```

- [ ] **Step 2: 运行单测，确认当前实现缺字段而失败**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/review -run 'TestService_CreateSession_BuildsStructuredSnapshot' -v
```

Expected: FAIL，提示 `SessionOutline` 或 `CurrentRoundGoal` 缺失。

- [ ] **Step 3: 最小实现结构化快照类型与提炼逻辑**

```go
type SessionOutline struct {
	Summary          string   `json:"summary"`
	MainQuestion     string   `json:"main_question"`
	CoreConcepts     []string `json:"core_concepts"`
	ProcessSteps     []string `json:"process_steps"`
	ApplicationCases []string `json:"application_cases"`
	Checkpoints      []string `json:"checkpoints"`
}

func buildSessionOutline(note ReviewNote) SessionOutline {
	segments := splitSentenceFragments(note.Body)
	return SessionOutline{
		Summary:          truncateRunes(strings.TrimSpace(note.Body), 180),
		MainQuestion:     buildMainQuestion(note.Title, segments),
		CoreConcepts:     pickDistinctFragments(segments, 2),
		ProcessSteps:     pickStepFragments(segments, 2),
		ApplicationCases: pickApplicationFragments(segments, 1),
		Checkpoints:      buildCheckpoints(note.Title, segments),
	}
}
```

- [ ] **Step 4: 把 `CreateSession` / `GetSession` 响应接上新字段**

```go
outline := buildSessionOutline(note)

return ReviewSessionResponse{
	SessionID:        session.ID,
	Status:           session.Status,
	Mode:             session.Mode,
	Title:            session.NoteTitle,
	OpeningPrompt:    openingPrompt(req.Mode, outline),
	InitialHints:     initialHints(req.Mode, outline),
	SessionOutline:   outline,
	CurrentRoundGoal: currentRoundGoal(req.Mode, 0, outline),
	NextQuestion:     nextQuestionForSession(session, []model.ReviewTurn{openingTurn}, outline),
	TurnIndex:        openingTurn.TurnIndex,
	Turns:            []ReviewTurnResponse{toTurnResponse(openingTurn)},
}
```

- [ ] **Step 5: 运行测试，确认快照结构落地**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/review -run 'TestService_CreateSession_(CapturesSnapshot|BuildsStructuredSnapshot)' -v
```

Expected: PASS

### Task 2: 扩展后端结构化反馈与针对性追问

**Files:**
- Modify: `backend/internal/domain/review/dto.go`
- Modify: `backend/internal/domain/review/feedback_builder.go`
- Modify: `backend/internal/domain/review/session_builder.go`
- Modify: `backend/internal/domain/review/session_service.go`
- Test: `backend/internal/domain/review/feedback_builder_test.go`
- Test: `backend/internal/domain/review/session_service_test.go`

- [ ] **Step 1: 先写失败测试，锁定命中/遗漏反馈**

```go
func TestService_Respond_ReturnsStructuredStageFeedback(t *testing.T) {
	t.Parallel()

	svc := newTestReviewServiceWithNotes([]ReviewNote{{
		NotePath:      "wiki/concepts/并发控制与速率限制.md",
		Title:         "并发控制与速率限制",
		SourceTitle:   "InkWords 内容生成平台架构解析系列",
		Body:          "并发控制用来限制同时执行的任务数量，避免资源打满。信号量负责发放并回收执行许可。速率限制负责控制单位时间内的请求频率，避免 API 被突发流量冲垮。",
		PreferredMode: model.ReviewModeDetailedQA,
	}})
	userID := uuid.New()

	created, err := svc.CreateSession(context.Background(), userID, CreateSessionRequest{
		NotePath:  "wiki/concepts/并发控制与速率限制.md",
		Mode:      model.ReviewModeDetailedQA,
		EntryType: model.ReviewEntryTypeToday,
	})
	require.NoError(t, err)

	resp, err := svc.Respond(context.Background(), userID, created.SessionID, RespondRequest{
		Answer: "这篇文章主要讲如何通过并发控制和信号量保护资源。",
	})
	require.NoError(t, err)
	require.Equal(t, "答对较多", resp.ReviewFeedback.Judgement)
	require.NotEmpty(t, resp.ReviewFeedback.HitPoints)
	require.NotEmpty(t, resp.ReviewFeedback.MissedPoints)
	require.NotEmpty(t, resp.CurrentRoundGoal)
}
```

- [ ] **Step 2: 运行相关测试，确认旧响应不满足断言**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/review -run 'Test(BuildReviewFeedback_ReportsHitsAndMisses|Service_Respond_ReturnsStructuredStageFeedback)' -v
```

Expected: FAIL，提示 `ReviewFeedback`、`CurrentRoundGoal` 或判断逻辑未实现。

- [ ] **Step 3: 实现结构化反馈构建器**

```go
type ReviewFeedback struct {
	Judgement   string   `json:"judgement"`
	HitPoints   []string `json:"hit_points"`
	MissedPoints []string `json:"missed_points"`
	Suggestion  string   `json:"suggestion"`
}

func buildReviewFeedback(outline SessionOutline, answer string) ReviewFeedback {
	hits, misses := matchCheckpoints(outline.Checkpoints, answer)
	return ReviewFeedback{
		Judgement:    classifyReviewAnswer(len(hits), len(outline.Checkpoints)),
		HitPoints:    defaultHitPoints(hits),
		MissedPoints: defaultMissedPoints(misses),
		Suggestion:   buildSuggestion(misses),
	}
}
```

- [ ] **Step 4: 用文章快照驱动 opening / hint / detailed question**

```go
func nextDetailedQuestion(answerCount int, outline SessionOutline) string {
	switch answerCount {
	case 0:
		return firstNonEmpty(outline.MainQuestion, "这篇文章最核心在讲什么？")
	case 1:
		return buildConceptQuestion(outline)
	default:
		return buildApplicationQuestion(outline)
	}
}
```

- [ ] **Step 5: 在 `Respond` / `Finish` 返回结构化反馈**

```go
outline := decodeSessionOutline(session.MetadataSnapshot)
reviewFeedback := buildReviewFeedback(outline, answer)
stageFeedback := buildStageFeedback(session.Mode, reviewFeedback)

return RespondResponse{
	SessionID:        session.ID,
	SessionStatus:    session.Status,
	TurnIndex:        session.TurnCount,
	StageFeedback:    stageFeedback,
	ReviewFeedback:   reviewFeedback,
	CurrentRoundGoal: currentRoundGoal(session.Mode, answerCount, outline),
	NextQuestion:     nextQuestion,
	Completed:        false,
}
```

- [ ] **Step 6: 运行后端 review 域测试**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/review/... -v
```

Expected: PASS

### Task 3: 扩展前端类型与会话状态消费

**Files:**
- Modify: `frontend/src/services/review.ts`
- Modify: `frontend/src/hooks/useKnowledgeReview.ts`
- Test: `frontend/src/services/review.test.ts`
- Test: `frontend/src/hooks/useKnowledgeReview.test.tsx`

- [ ] **Step 1: 先写失败测试，锁定前端对新字段的消费**

```ts
it('preserves review feedback fields returned by respond', async () => {
  mockFetch.mockResolvedValue({
    ok: true,
    status: 200,
    json: async () => ({
      code: 200,
      data: {
        session_id: 'session-1',
        session_status: 'in_progress',
        turn_index: 3,
        stage_feedback: '你已经答到主线。',
        current_round_goal: '补充关键概念',
        review_feedback: {
          judgement: '部分答对',
          hit_points: ['答到主线'],
          missed_points: ['没提速率限制'],
          suggestion: '补充速率限制和适用场景',
        },
        completed: false,
        final_feedback: { summary: '', strengths: [], gaps: [], next_focus: [] },
      },
    }),
  } as Response)

  const resp = await reviewService.respond('session-1', { answer: '并发控制是控制任务数' })
  expect(resp.review_feedback?.judgement).toBe('部分答对')
  expect(resp.current_round_goal).toBe('补充关键概念')
})
```

- [ ] **Step 2: 扩展 TS 类型定义**

```ts
export interface SessionOutline {
  summary: string
  main_question: string
  core_concepts: string[]
  process_steps: string[]
  application_cases: string[]
  checkpoints: string[]
}

export interface ReviewFeedback {
  judgement: string
  hit_points: string[]
  missed_points: string[]
  suggestion: string
}
```

- [ ] **Step 3: 更新 Hook，把结构化反馈写入当前 session**

```ts
setCurrentSession({
  ...currentSession,
  status: response.session_status,
  next_question: response.next_question,
  current_round_goal: response.current_round_goal,
  latest_review_feedback: response.review_feedback ?? null,
  turn_index: response.turn_index,
  turns: nextTurns,
})
```

- [ ] **Step 4: 运行前端服务与 Hook 测试**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend
npm run test -- review.test.ts useKnowledgeReview.test.tsx
```

Expected: PASS

### Task 4: 升级复习会话卡片展示

**Files:**
- Modify: `frontend/src/components/review/ReviewSessionCard.tsx`
- Test: `frontend/src/components/review/ReviewSessionCard.test.tsx`

- [ ] **Step 1: 先写失败测试，锁定 UI 要显示的新信息**

```tsx
it('renders round goal and structured review feedback', () => {
  const html = renderToStaticMarkup(
    <ReviewSessionCard
      session={{
        session_id: 'session-1',
        status: 'in_progress',
        mode: 'detailed_qa',
        title: '并发控制与速率限制',
        opening_prompt: '先说主线',
        initial_hints: [],
        current_round_goal: '先讲清楚这篇文章的主线问题',
        latest_review_feedback: {
          judgement: '部分答对',
          hit_points: ['答到了并发控制限制任务数量'],
          missed_points: ['没有提到速率限制控制请求频率'],
          suggestion: '下一轮补上速率限制和为什么需要它',
        },
        turn_index: 2,
        turns: [],
      }}
      selectedMode="detailed_qa"
      latestStageFeedback="你已经抓到主线。"
      latestHint={null}
      finalFeedback={null}
      onModeChange={() => {}}
      onRespond={async () => {}}
      onRequestHint={async () => {}}
      onFinish={async () => {}}
    />,
  )

  expect(html).toContain('本轮目标')
  expect(html).toContain('部分答对')
  expect(html).toContain('你答到的点')
  expect(html).toContain('你还漏掉的点')
})
```

- [ ] **Step 2: 实现新的反馈区域**

```tsx
{session.current_round_goal ? (
  <div className="mt-4 rounded-2xl border border-sky-200 bg-sky-50 px-4 py-3">
    <p className="text-sm font-medium text-sky-900">本轮目标</p>
    <p className="mt-1 text-sm leading-6 text-sky-900">{session.current_round_goal}</p>
  </div>
) : null}

{session.latest_review_feedback ? (
  <div className="mt-4 rounded-2xl border border-emerald-200 bg-emerald-50 p-4">
    <p className="text-sm font-medium text-emerald-900">{session.latest_review_feedback.judgement}</p>
    <p className="mt-3 text-xs font-medium uppercase tracking-wide text-emerald-700">你答到的点</p>
    <p className="mt-1 text-sm text-emerald-900">{session.latest_review_feedback.hit_points.join('；')}</p>
    <p className="mt-3 text-xs font-medium uppercase tracking-wide text-emerald-700">你还漏掉的点</p>
    <p className="mt-1 text-sm text-emerald-900">{session.latest_review_feedback.missed_points.join('；')}</p>
  </div>
) : null}
```

- [ ] **Step 3: 运行组件测试**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend
npm run test -- ReviewSessionCard.test.tsx
```

Expected: PASS

### Task 5: 全量验证与文档同步

**Files:**
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_PRD.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`

- [ ] **Step 1: 更新 API / PRD / 架构文档中的复习接口与交互描述**

```md
- `ReviewSessionResponse` 新增 `session_outline`、`current_round_goal`
- `RespondResponse` 新增 `review_feedback`
- 复习阶段反馈升级为“命中点 / 遗漏点 / 下一步建议”
```

- [ ] **Step 2: 运行后端、前端测试与诊断**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/domain/review/... -v

cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend
npm run test -- review.test.ts useKnowledgeReview.test.tsx ReviewSessionCard.test.tsx
```

Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add backend/internal/domain/review frontend/src/components/review frontend/src/hooks/useKnowledgeReview.ts frontend/src/services/review.ts .trae/documents docs/superpowers/plans/2026-06-01-review-targeted-questioning.md
git commit -m "feat(review): add article-driven questioning feedback"
```
