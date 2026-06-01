# Review Content-Aware Feedback Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade the knowledge-review flow so each session asks article-specific questions and returns explicit hit/miss feedback that tells the user what they answered correctly or missed.

**Architecture:** Keep the existing `review` domain boundaries and replace the generic snapshot/question/feedback helpers with a richer session snapshot that extracts a structured outline from each note. Backend APIs return article-aware review feedback in JSON, and the frontend reuses the current session card while rendering explicit judgement, hit points, missed points, and the current round target.

**Tech Stack:** Go 1.21+, Gin, GORM, React 18, Zustand, Vitest

---

### Task 1: Extend The Review Snapshot Model

**Files:**
- Modify: `backend/internal/domain/review/session_builder.go`
- Modify: `backend/internal/domain/review/session_service.go`
- Modify: `backend/internal/domain/review/dto.go`
- Test: `backend/internal/domain/review/session_service_test.go`

- [ ] **Step 1: Write the failing backend test**

```go
func TestService_CreateSession_BuildsStructuredSnapshot(t *testing.T) {
	t.Parallel()

	svc := newTestReviewServiceWithNotes([]ReviewNote{{
		NotePath:    "wiki/concepts/并发控制与速率限制.md",
		Title:       "并发控制与速率限制",
		SourceTitle: "InkWords 内容生成平台架构解析系列",
		Body: "并发控制用来限制同时执行的任务数量，避免资源打满。信号量负责发放并回收执行许可。速率限制负责控制单位时间内的请求频率，避免 API 被突发流量冲垮。实际落地时，通常会把并发上限、队列等待和重试退避组合起来。",
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

- [ ] **Step 2: Run the single backend test to verify it fails**

Run: `cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend && go test ./internal/domain/review -run TestService_CreateSession_BuildsStructuredSnapshot -v`
Expected: FAIL because `ReviewSessionResponse` does not expose `session_outline` or `current_round_goal`, and the session builder still returns only generic key points.

- [ ] **Step 3: Implement the minimal structured snapshot**

```go
type SessionOutline struct {
	MainQuestion    string   `json:"main_question"`
	CoreConcepts    []string `json:"core_concepts"`
	ProcessSteps    []string `json:"process_steps"`
	ApplicationUses []string `json:"application_uses"`
	Checkpoints     []string `json:"checkpoints"`
}
```

```go
func buildSessionSnapshot(note ReviewNote) (string, SessionOutline) {
	body := strings.TrimSpace(note.Body)
	if len([]rune(body)) > 800 {
		body = string([]rune(body)[:800])
	}

	segments := splitMeaningfulSentences(note.Body, 6)
	outline := SessionOutline{
		MainQuestion:    buildMainQuestion(note.Title),
		CoreConcepts:    collectOutlineItems(segments, 2),
		ProcessSteps:    collectProcessSteps(segments, 2),
		ApplicationUses: collectApplicationUses(segments, 1),
		Checkpoints:     collectCheckpoints(note.Title, segments, 4),
	}

	return body, outline
}
```

```go
summarySnapshot, outline := buildSessionSnapshot(note)
session := model.ReviewSession{
	SummarySnapshot:   summarySnapshot,
	KeyPointsSnapshot: mustMarshalJSON(outline.Checkpoints),
	MetadataSnapshot: mustMarshalJSON(map[string]any{
		"preferred_mode": note.PreferredMode,
		"session_outline": outline,
	}),
}
```

```go
return ReviewSessionResponse{
	SessionID:        session.ID,
	Status:           session.Status,
	Mode:             session.Mode,
	Title:            session.NoteTitle,
	OpeningPrompt:    opening,
	InitialHints:     hints,
	NextQuestion:     nextQuestionForSession(session, []model.ReviewTurn{openingTurn}),
	CurrentRoundGoal: currentRoundGoal(session.Mode, outline, 0),
	SessionOutline:   outline,
	TurnIndex:        openingTurn.TurnIndex,
	Turns:            []ReviewTurnResponse{toTurnResponse(openingTurn)},
}
```

- [ ] **Step 4: Run the single backend test to verify it passes**

Run: `cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend && go test ./internal/domain/review -run TestService_CreateSession_BuildsStructuredSnapshot -v`
Expected: PASS and the session response contains article-aware outline data.

- [ ] **Step 5: Commit the snapshot model checkpoint**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add backend/internal/domain/review/session_builder.go backend/internal/domain/review/session_service.go backend/internal/domain/review/dto.go backend/internal/domain/review/session_service_test.go
git commit -m "feat(review): add structured session outline"
```

### Task 2: Return Explicit Hit/Miss Feedback From Respond And Finish

**Files:**
- Modify: `backend/internal/domain/review/feedback_builder.go`
- Modify: `backend/internal/domain/review/session_service.go`
- Modify: `backend/internal/domain/review/dto.go`
- Test: `backend/internal/domain/review/feedback_builder_test.go`
- Test: `backend/internal/domain/review/session_service_test.go`

- [ ] **Step 1: Write the failing backend tests**

```go
func TestBuildReviewFeedback_ReportsHitsAndMisses(t *testing.T) {
	t.Parallel()

	outline := SessionOutline{
		Checkpoints: []string{
			"并发控制限制同时执行的任务数量",
			"速率限制控制单位时间内的请求频率",
			"信号量负责发放和回收执行许可",
		},
	}

	feedback := buildReviewFeedback(outline, "并发控制是为了限制同时执行的任务数量，通常会配合信号量。")
	require.Equal(t, "答对较多", feedback.Judgement)
	require.Contains(t, feedback.HitPoints, "并发控制限制同时执行的任务数量")
	require.Contains(t, feedback.HitPoints, "信号量负责发放和回收执行许可")
	require.Contains(t, feedback.MissedPoints, "速率限制控制单位时间内的请求频率")
	require.NotEmpty(t, feedback.Suggestion)
}
```

```go
func TestService_Respond_ReturnsStructuredStageFeedback(t *testing.T) {
	t.Parallel()

	session := seedDetailedQASession(t)
	resp, err := session.Service.Respond(context.Background(), session.UserID, session.ID, RespondRequest{
		Answer: "这篇文章主要讲如何通过并发控制和信号量保护资源。",
	})
	require.NoError(t, err)
	require.Equal(t, "答对较多", resp.ReviewFeedback.Judgement)
	require.NotEmpty(t, resp.ReviewFeedback.HitPoints)
	require.NotEmpty(t, resp.ReviewFeedback.MissedPoints)
	require.NotEmpty(t, resp.CurrentRoundGoal)
}
```

- [ ] **Step 2: Run the focused backend tests to verify they fail**

Run: `cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend && go test ./internal/domain/review -run 'TestBuildReviewFeedback_ReportsHitsAndMisses|TestService_Respond_ReturnsStructuredStageFeedback' -v`
Expected: FAIL because no structured review feedback type exists and `RespondResponse` only exposes a plain `stage_feedback` string.

- [ ] **Step 3: Implement the minimal structured feedback**

```go
type ReviewFeedbackDetail struct {
	Judgement    string   `json:"judgement"`
	HitPoints    []string `json:"hit_points"`
	MissedPoints []string `json:"missed_points"`
	Suggestion   string   `json:"suggestion"`
}
```

```go
func buildReviewFeedback(outline SessionOutline, answer string) ReviewFeedbackDetail {
	normalized := normalizeAnswer(answer)
	hitPoints, missedPoints := scoreCheckpoints(outline.Checkpoints, normalized)
	judgement := classifyJudgement(len(hitPoints), len(outline.Checkpoints))

	return ReviewFeedbackDetail{
		Judgement:    judgement,
		HitPoints:    hitPoints,
		MissedPoints: missedPoints,
		Suggestion:   buildSuggestion(missedPoints),
	}
}
```

```go
feedback := buildReviewFeedback(outline, answer)
stageFeedback := buildStageFeedback(session.Mode, outline, feedback, answerCount)

return RespondResponse{
	SessionID:        session.ID,
	SessionStatus:    session.Status,
	TurnIndex:        session.TurnCount,
	StageFeedback:    stageFeedback,
	ReviewFeedback:   feedback,
	CurrentRoundGoal: currentRoundGoal(session.Mode, outline, answerCount),
	NextQuestion:     nextQuestion,
	Completed:        false,
}
```

```go
type FinalFeedback struct {
	Summary        string               `json:"summary"`
	Strengths      []string             `json:"strengths"`
	Gaps           []string             `json:"gaps"`
	NextFocus      []string             `json:"next_focus"`
	ReviewFeedback ReviewFeedbackDetail `json:"review_feedback"`
}
```

- [ ] **Step 4: Run the focused backend tests to verify they pass**

Run: `cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend && go test ./internal/domain/review -run 'TestBuildReviewFeedback_ReportsHitsAndMisses|TestService_Respond_ReturnsStructuredStageFeedback' -v`
Expected: PASS and both staged and final feedback now expose hit/miss details.

- [ ] **Step 5: Commit the feedback checkpoint**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add backend/internal/domain/review/feedback_builder.go backend/internal/domain/review/session_service.go backend/internal/domain/review/dto.go backend/internal/domain/review/feedback_builder_test.go backend/internal/domain/review/session_service_test.go
git commit -m "feat(review): add structured answer feedback"
```

### Task 3: Render Round Targets And Explicit Feedback In The Session Card

**Files:**
- Modify: `frontend/src/services/review.ts`
- Modify: `frontend/src/store/reviewStore.ts`
- Modify: `frontend/src/hooks/useKnowledgeReview.ts`
- Modify: `frontend/src/components/review/ReviewSessionCard.tsx`
- Test: `frontend/src/components/review/ReviewSessionCard.test.tsx`

- [ ] **Step 1: Write the failing frontend test**

```tsx
it('renders the round goal and explicit hit-miss feedback after a response', () => {
  render(
    <ReviewSessionCard
      session={{
        session_id: 'session-1',
        status: 'in_progress',
        mode: 'detailed_qa',
        title: '并发控制与速率限制',
        opening_prompt: '先别看原文，我们从主线开始。',
        initial_hints: [],
        next_question: '并发控制主要解决什么问题？',
        current_round_goal: '这一轮先讲清主旨和问题背景。',
        session_outline: {
          main_question: '并发控制主要解决什么问题？',
          core_concepts: ['并发控制', '信号量'],
          process_steps: [],
          application_uses: ['保护共享资源'],
          checkpoints: ['并发控制限制同时执行的任务数量'],
        },
        turn_index: 2,
        turns: [],
      }}
      selectedMode="detailed_qa"
      latestStageFeedback="你已经抓住了主线。"
      latestReviewFeedback={{
        judgement: '答对较多',
        hit_points: ['并发控制限制同时执行的任务数量'],
        missed_points: ['速率限制控制单位时间内的请求频率'],
        suggestion: '下一轮补上速率限制与并发控制的区别。',
      }}
      latestHint={null}
      finalFeedback={null}
      onModeChange={vi.fn()}
      onRespond={vi.fn()}
      onRequestHint={vi.fn()}
      onFinish={vi.fn()}
    />,
  )

  expect(screen.getByText('本轮目标')).toBeInTheDocument()
  expect(screen.getByText('这一轮先讲清主旨和问题背景。')).toBeInTheDocument()
  expect(screen.getByText('你已经答到')).toBeInTheDocument()
  expect(screen.getByText('这次还没覆盖')).toBeInTheDocument()
  expect(screen.getByText('下一步建议')).toBeInTheDocument()
})
```

- [ ] **Step 2: Run the single frontend test to verify it fails**

Run: `cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- ReviewSessionCard.test.tsx`
Expected: FAIL because the frontend types do not include `current_round_goal`, `session_outline`, or `latestReviewFeedback`, and the card renders only the old text blocks.

- [ ] **Step 3: Implement the minimal frontend rendering**

```ts
export interface ReviewFeedbackDetail {
  judgement: string
  hit_points: string[]
  missed_points: string[]
  suggestion: string
}

export interface SessionOutline {
  main_question: string
  core_concepts: string[]
  process_steps: string[]
  application_uses: string[]
  checkpoints: string[]
}
```

```tsx
{session.current_round_goal ? (
  <div className="mt-3 rounded-2xl border border-blue-200 bg-blue-50 px-4 py-3 text-sm leading-6 text-blue-900">
    <p className="font-medium">本轮目标</p>
    <p className="mt-1">{session.current_round_goal}</p>
  </div>
) : null}
```

```tsx
{latestReviewFeedback ? (
  <div className="mt-4 rounded-2xl border border-amber-200 bg-amber-50 p-4">
    <p className="text-sm font-medium text-amber-900">回答判断：{latestReviewFeedback.judgement}</p>
    <p className="mt-3 text-xs font-medium uppercase tracking-wide text-amber-700">你已经答到</p>
    <p className="mt-1 text-sm text-amber-900">{latestReviewFeedback.hit_points.join('；')}</p>
    <p className="mt-3 text-xs font-medium uppercase tracking-wide text-amber-700">这次还没覆盖</p>
    <p className="mt-1 text-sm text-amber-900">{latestReviewFeedback.missed_points.join('；')}</p>
    <p className="mt-3 text-xs font-medium uppercase tracking-wide text-amber-700">下一步建议</p>
    <p className="mt-1 text-sm text-amber-900">{latestReviewFeedback.suggestion}</p>
  </div>
) : null}
```

- [ ] **Step 4: Run the frontend test to verify it passes**

Run: `cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- ReviewSessionCard.test.tsx`
Expected: PASS and the session card renders the round target plus explicit hit/miss feedback.

- [ ] **Step 5: Commit the frontend rendering checkpoint**

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git add frontend/src/services/review.ts frontend/src/store/reviewStore.ts frontend/src/hooks/useKnowledgeReview.ts frontend/src/components/review/ReviewSessionCard.tsx frontend/src/components/review/ReviewSessionCard.test.tsx
git commit -m "feat(review): show explicit answer feedback"
```

### Task 4: Run Focused Validation And Sync Docs

**Files:**
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_PRD.md`
- Modify: `README.md`
- Check: `backend/internal/domain/review/*.go`
- Check: `frontend/src/components/review/ReviewSessionCard.tsx`

- [ ] **Step 1: Run the backend review test suite**

Run: `cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend && go test ./internal/domain/review/...`
Expected: PASS with the new snapshot and feedback assertions.

- [ ] **Step 2: Run the focused frontend review test suite**

Run: `cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend && npm run test -- ReviewSessionCard.test.tsx ReviewEntryCards.test.tsx useKnowledgeReview.test.tsx reviewStore.test.ts`
Expected: PASS and no regression in the knowledge-review flow.

- [ ] **Step 3: Run diagnostics on edited files**

Run diagnostics for:
- `backend/internal/domain/review/session_builder.go`
- `backend/internal/domain/review/feedback_builder.go`
- `backend/internal/domain/review/session_service.go`
- `frontend/src/services/review.ts`
- `frontend/src/hooks/useKnowledgeReview.ts`
- `frontend/src/components/review/ReviewSessionCard.tsx`

Expected: no new lint or type errors.

- [ ] **Step 4: Update the synced docs**

```md
- `.trae/documents/InkWords_API.md`: record the new review response fields `session_outline`, `current_round_goal`, `review_feedback`.
- `.trae/documents/InkWords_Architecture.md`: describe the article-aware session snapshot and explicit answer feedback flow.
- `.trae/documents/InkWords_Conversation_Log.md`: add this feature decision and scope.
- `.trae/documents/InkWords_Development_Plan_and_Log.md`: add implementation notes and validation commands.
- `.trae/documents/InkWords_PRD.md`: note that review feedback must show hit/miss points against article content.
- `README.md`: briefly mention the upgraded review feedback experience.
```

- [ ] **Step 5: Review the scoped diff**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
git diff -- backend/internal/domain/review frontend/src/services/review.ts frontend/src/store/reviewStore.ts frontend/src/hooks/useKnowledgeReview.ts frontend/src/components/review/ReviewSessionCard.tsx .trae/documents/InkWords_API.md .trae/documents/InkWords_Architecture.md .trae/documents/InkWords_Conversation_Log.md .trae/documents/InkWords_Development_Plan_and_Log.md .trae/documents/InkWords_PRD.md README.md
```

Expected: diff only shows the review-specific backend/frontend upgrades plus the required doc sync.
