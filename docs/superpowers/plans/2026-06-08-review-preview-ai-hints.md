# Review Preview AI Hints Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将知识复习流程升级为“先完整预览原文，再进入复述”，并让 review-service 基于原文、会话上下文和用户回答生成针对性反馈与提醒。

**Architecture:** review-service 继续作为复习主边界，复用当前 `ReviewSession` / `ReviewTurn` 持久化，只扩展 DTO 与领域服务，把原文预览、AI 反馈和“遗忘表达”识别封装在 review 域内。前端保持现有 `KnowledgeReview -> useKnowledgeReview -> ReviewSessionCard` 组合，但把会话 UI 改成编辑式双阶段体验：先原文预览，再复述与反馈，且保留 AI 失败时回退到现有规则反馈。

**Tech Stack:** Go 1.21+, Gin, React 18, Zustand, Tailwind CSS, shadcn/ui, Vitest, Go test, DeepSeek client.

---

### Task 1: 后端 DTO 与会话快照扩展

**Files:**
- Modify: `backend/services/review-service/domain/review/dto.go`
- Modify: `backend/services/review-service/domain/review/session_builder.go`
- Modify: `backend/services/review-service/domain/review/session_service.go`
- Test: `backend/services/review-service/domain/review/session_service_test.go`
- Test: `backend/services/review-service/domain/review/handler_test.go`

- [ ] **Step 1: Write the failing Go test for preview fields in create session**

```go
func TestService_CreateSession_ReturnsPreviewContent(t *testing.T) {
	t.Parallel()

	svc := newTestReviewServiceWithNotes([]ReviewNote{{
		NotePath:      "wiki/concepts/review.md",
		Title:         "复习主题",
		SourceTitle:   "知识库",
		Body:          "第一段原文。\n\n第二段原文。",
		PreferredMode: model.ReviewModeLightRecall,
	}})

	resp, err := svc.CreateSession(context.Background(), uuid.New(), CreateSessionRequest{
		NotePath:  "wiki/concepts/review.md",
		Mode:      model.ReviewModeLightRecall,
		EntryType: model.ReviewEntryTypeManualSelect,
	})
	require.NoError(t, err)
	require.Equal(t, "知识库", resp.SourceTitle)
	require.NotEmpty(t, resp.SourcePreview)
	require.Contains(t, resp.SourcePreview, "第一段原文")
	require.False(t, resp.ReadyToAnswer)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./services/review-service/domain/review -run TestService_CreateSession_ReturnsPreviewContent -count=1`
Expected: FAIL with unknown fields like `SourceTitle` / `SourcePreview` / `ReadyToAnswer`

- [ ] **Step 3: Write minimal DTO and session response implementation**

```go
// backend/services/review-service/domain/review/dto.go
type ReviewSessionResponse struct {
	SessionID            uuid.UUID            `json:"session_id"`
	Status               string               `json:"status"`
	Mode                 string               `json:"mode"`
	Title                string               `json:"title"`
	SourceTitle          string               `json:"source_title"`
	SourcePreview        string               `json:"source_preview"`
	ReadyToAnswer        bool                 `json:"ready_to_answer"`
	OpeningPrompt        string               `json:"opening_prompt"`
	InitialHints         []string             `json:"initial_hints"`
	SessionOutline       SessionOutline       `json:"session_outline"`
	CurrentRoundGoal     string               `json:"current_round_goal,omitempty"`
	LatestReviewFeedback *ReviewFeedback      `json:"latest_review_feedback,omitempty"`
	NextQuestion         string               `json:"next_question,omitempty"`
	TurnIndex            int                  `json:"turn_index"`
	Turns                []ReviewTurnResponse `json:"turns,omitempty"`
}

// backend/services/review-service/domain/review/session_builder.go
func buildSessionSnapshot(note ReviewNote) (string, SessionOutline, string) {
	body := strings.TrimSpace(note.Body)
	return truncateRunes(body, 800), buildSessionOutline(note), truncateRunes(body, 2400)
}

// backend/services/review-service/domain/review/session_service.go
summarySnapshot, outline, sourcePreview := buildSessionSnapshot(note)

return ReviewSessionResponse{
	SessionID:        session.ID,
	Status:           session.Status,
	Mode:             session.Mode,
	Title:            session.NoteTitle,
	SourceTitle:      session.SourceTitle,
	SourcePreview:    sourcePreview,
	ReadyToAnswer:    false,
	OpeningPrompt:    opening,
	InitialHints:     hints,
	SessionOutline:   outline,
	CurrentRoundGoal: currentRoundGoal(session.Mode, 0, outline),
	NextQuestion:     nextQuestionForSession(session, []model.ReviewTurn{openingTurn}, outline),
	TurnIndex:        openingTurn.TurnIndex,
	Turns:            []ReviewTurnResponse{toTurnResponse(openingTurn)},
}, nil
```

- [ ] **Step 4: Run focused review tests**

Run: `go test ./services/review-service/domain/review -run 'TestService_CreateSession_(CapturesSnapshot|BuildsStructuredSnapshot|ReturnsPreviewContent)' -count=1`
Expected: PASS

- [ ] **Step 5: Extend handler fixture coverage**

```go
require.Equal(t, "知识库", body.Data.SourceTitle)
require.Contains(t, body.Data.SourcePreview, "第一段原文")
require.False(t, body.Data.ReadyToAnswer)
```

- [ ] **Step 6: Run handler tests**

Run: `go test ./services/review-service/domain/review -run 'TestHandler_(CreateSession|GetTodayCard|GetHistory)' -count=1`
Expected: PASS

### Task 2: 后端 AI 反馈与遗忘表达降级

**Files:**
- Modify: `backend/services/review-service/domain/review/feedback_builder.go`
- Modify: `backend/services/review-service/domain/review/session_service.go`
- Modify: `backend/services/review-service/domain/review/service_test.go`
- Modify: `backend/services/review-service/app/bootstrap/bootstrap.go`
- Possibly create: `backend/services/review-service/domain/review/ai_feedback.go`
- Test: `backend/services/review-service/domain/review/session_service_test.go`

- [ ] **Step 1: Write the failing Go test for “先提示再摘录”**

```go
func TestService_Respond_WhenUserDoesNotRemember_ReturnsHintThenExcerpt(t *testing.T) {
	t.Parallel()

	ai := &stubAIFeedbackGenerator{
		response: AIFeedbackResult{
			Judgement:       "需要提醒",
			HitPoints:       []string{"已经识别出主题"},
			MissedPoints:    []string{"关键定义"},
			Suggestion:      "先回忆它解决的问题。",
			HintText:        "先想想它为什么要限制并发。",
			ExcerptText:     "原文摘录：并发控制用来限制同时执行的任务数量。",
			ShouldShowQuote: true,
		},
	}
	svc := newTestReviewServiceWithAI(ai)

	created, _ := svc.CreateSession(context.Background(), uuid.New(), CreateSessionRequest{...})
	resp, err := svc.Respond(context.Background(), userID, created.SessionID, RespondRequest{
		Answer: "我不记得了",
	})
	require.NoError(t, err)
	require.Equal(t, "先想想它为什么要限制并发。", resp.HintText)
	require.Contains(t, resp.ExcerptText, "原文摘录")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./services/review-service/domain/review -run TestService_Respond_WhenUserDoesNotRemember_ReturnsHintThenExcerpt -count=1`
Expected: FAIL with missing AI generator or missing response fields

- [ ] **Step 3: Add AI feedback contract and minimal fallback behavior**

```go
type AIFeedbackGenerator interface {
	Generate(context.Context, AIFeedbackInput) (AIFeedbackResult, error)
}

type AIFeedbackResult struct {
	Judgement       string
	HitPoints       []string
	MissedPoints    []string
	Suggestion      string
	StageFeedback   string
	NextQuestion    string
	HintText        string
	ExcerptText     string
	ShouldShowQuote bool
}
```

- [ ] **Step 4: Inject generator into Service and implement fallback**

```go
type Service struct {
	repo          reviewRepository
	noteSource    reviewNoteSource
	aiFeedback    AIFeedbackGenerator
	now           func() time.Time
}

if s.aiFeedback != nil {
	result, err := s.aiFeedback.Generate(ctx, buildAIFeedbackInput(session, updatedTurns, outline, answer))
	if err == nil {
		reviewFeedback = ReviewFeedback{
			Judgement:    result.Judgement,
			HitPoints:    ensureFeedbackItems(result.HitPoints, "已经开始围绕主题作答，但还需要更贴近文章主线"),
			MissedPoints: ensureFeedbackItems(result.MissedPoints, "可以再补一个更具体的关键点"),
			Suggestion:   firstNonEmpty(result.Suggestion, buildReviewSuggestion(result.Judgement, result.MissedPoints)),
		}
		stageFeedback = firstNonEmpty(result.StageFeedback, buildStageFeedback(session.Mode, reviewFeedback))
		hintText = strings.TrimSpace(result.HintText)
		excerptText = strings.TrimSpace(result.ExcerptText)
	}
}
```

- [ ] **Step 5: Extend respond DTO for hint/excerpt payload**

```go
type RespondResponse struct {
	SessionID        uuid.UUID      `json:"session_id"`
	SessionStatus    string         `json:"session_status"`
	TurnIndex        int            `json:"turn_index"`
	StageFeedback    string         `json:"stage_feedback,omitempty"`
	CurrentRoundGoal string         `json:"current_round_goal,omitempty"`
	ReviewFeedback   ReviewFeedback `json:"review_feedback"`
	NextQuestion     string         `json:"next_question,omitempty"`
	HintText         string         `json:"hint_text,omitempty"`
	ExcerptText      string         `json:"excerpt_text,omitempty"`
	Completed        bool           `json:"completed"`
	FinalFeedback    FinalFeedback  `json:"final_feedback"`
}
```

- [ ] **Step 6: Wire a concrete DeepSeek-backed generator in bootstrap**

```go
apiKey := os.Getenv("DEEPSEEK_API_KEY")
var aiFeedback reviewdomain.AIFeedbackGenerator
if strings.TrimSpace(apiKey) != "" {
	aiFeedback = reviewdomain.NewDeepSeekAIFeedbackGenerator(
		llm.NewDeepSeekClient(apiKey),
		firstNonEmpty(os.Getenv("DEEPSEEK_REVIEW_MODEL"), "deepseek-chat"),
	)
}
reviewService := reviewdomain.NewService(reviewRepo, wiki.BuildNoteSource(os.Getenv("OBSIDIAN_WIKI_DIR")), aiFeedback)
```

- [ ] **Step 7: Run backend review tests**

Run: `go test ./services/review-service/domain/review/... -count=1`
Expected: PASS

### Task 3: 前端原文预览与会话体验重构

**Files:**
- Modify: `frontend/src/services/review.ts`
- Modify: `frontend/src/hooks/useKnowledgeReview.ts`
- Modify: `frontend/src/components/review/ReviewSessionCard.tsx`
- Modify: `frontend/src/components/review/ReviewSessionCard.test.tsx`
- Modify: `frontend/src/hooks/useKnowledgeReview.test.tsx`
- Modify: `frontend/src/services/review.test.ts`

- [ ] **Step 1: Write the failing Vitest for preview-first session UI**

```tsx
it('renders source preview first and hides answer textarea until review starts', () => {
  const html = renderToStaticMarkup(
    <ReviewSessionCard
      session={{
        session_id: 'session-1',
        status: 'created',
        mode: 'light_recall',
        title: '并发控制与速率限制',
        source_title: '知识库',
        source_preview: '第一段原文。第二段原文。',
        ready_to_answer: false,
        opening_prompt: '先看原文再复述',
        initial_hints: [],
        session_outline: { summary: '摘要', main_question: '主问题', core_concepts: [], process_steps: [], application_cases: [], checkpoints: [] },
        turn_index: 1,
        turns: [],
      }}
      selectedMode="light_recall"
      latestStageFeedback={null}
      latestHint={null}
      finalFeedback={null}
      onModeChange={() => {}}
      onStartAnswering={() => {}}
      onRespond={() => {}}
      onRequestHint={() => {}}
      onFinish={() => {}}
    />,
  )

  expect(html).toContain('原文预览')
  expect(html).toContain('开始复述')
  expect(html).not.toContain('用自己的话讲一遍')
})
```

- [ ] **Step 2: Run UI test to verify it fails**

Run: `npm --prefix frontend test -- ReviewSessionCard`
Expected: FAIL with missing props or missing preview content

- [ ] **Step 3: Extend frontend DTOs and service parsing**

```ts
export interface ReviewSessionResponse {
  session_id: string
  status: string
  mode: ReviewMode
  title: string
  source_title: string
  source_preview: string
  ready_to_answer: boolean
  opening_prompt: string
  initial_hints: string[]
  session_outline: SessionOutline
  current_round_goal?: string
  latest_review_feedback?: ReviewFeedback | null
  next_question?: string
  turn_index: number
  turns?: ReviewTurnResponse[]
}

export interface RespondResponse {
  ...
  hint_text?: string
  excerpt_text?: string
}
```

- [ ] **Step 4: Implement preview-first interaction in the hook**

```ts
const startAnswering = useCallback(() => {
  if (!currentSession || currentSession.ready_to_answer) {
    return
  }

  setCurrentSession({
    ...currentSession,
    ready_to_answer: true,
  })
}, [currentSession, setCurrentSession])
```

- [ ] **Step 5: Implement the redesigned session card**

```tsx
{!session.ready_to_answer ? (
  <div className="rounded-[28px] border border-zinc-200 bg-zinc-50 p-5">
    <p className="text-xs font-medium tracking-[0.24em] text-zinc-500">原文预览</p>
    <h3 className="mt-3 text-lg font-semibold text-zinc-900">{session.source_title || session.title}</h3>
    <div className="mt-4 whitespace-pre-wrap text-sm leading-7 text-zinc-700">
      {session.source_preview}
    </div>
    <Button className="mt-5" onClick={onStartAnswering}>开始复述</Button>
  </div>
) : (
  <div>{/* existing answer, feedback, hint, history sections */}</div>
)}
```

- [ ] **Step 6: Surface backend hint/excerpt in session state**

```ts
if (response.hint_text) {
  setLatestHint(response.hint_text)
}

if (response.excerpt_text) {
  nextTurns = appendTurn(nextTurns, 'system', 'excerpt', response.excerpt_text)
}
```

- [ ] **Step 7: Run frontend tests**

Run: `npm --prefix frontend test -- ReviewSessionCard useKnowledgeReview review`
Expected: PASS

### Task 4: 文档同步与验证

**Files:**
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`
- Modify: `.trae/documents/InkWords_Database.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_PRD.md`
- Modify: `README.md`

- [ ] **Step 1: Update API and PRD language**

```md
- 复习会话创建后先返回原文预览内容，用户确认阅读后再进入复述输入。
- 用户回答时，review-service 会将原文、会话上下文与回答一起发送给 AI 生成针对性提示。
- 当用户自然表达“不记得 / 忘了 / 记不清了”时，系统先给简短提示，再按需返回相关原文摘录。
```

- [ ] **Step 2: Update architecture and database notes**

```md
- review-service 新增 AI feedback generator 作为可选依赖，失败时降级为规则反馈。
- 本次仅复用 `review_sessions` / `review_turns` 快照字段，不新增表结构。
```

- [ ] **Step 3: Run verification commands**

Run: `go test ./services/review-service/domain/review/... -count=1`
Expected: PASS

Run: `npm --prefix frontend test -- ReviewSessionCard useKnowledgeReview review`
Expected: PASS

Run: `git diff -- docs/superpowers/plans/2026-06-08-review-preview-ai-hints.md .trae/documents/InkWords_API.md .trae/documents/InkWords_Architecture.md .trae/documents/InkWords_Conversation_Log.md .trae/documents/InkWords_Database.md .trae/documents/InkWords_Development_Plan_and_Log.md .trae/documents/InkWords_PRD.md README.md`
Expected: review preview / AI 提示相关描述与代码变更一致
