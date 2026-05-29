# Knowledge Roaming Review Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 InkWords 落地“知识漫游复习”能力，让用户可以从 Obsidian `wiki/concepts/` 中通过今日推荐、手动随机和手动选择三种入口发起复习，并完成轻提示复述或细致提问训练。

**Architecture:** 保持现有“Obsidian 作为知识正文源、PostgreSQL 作为业务状态源”的边界不变，在后端新增独立 `review` 领域切片，在前端新增独立 `KnowledgeReview` 主视图。首版接口全部采用普通 JSON，不复用现有 `stream` SSE 链路；会话创建时读取 Obsidian 原文并生成训练快照，后续所有追问、提示和反馈都基于同一份快照运行。

**Tech Stack:** Go 1.25 + Gin + GORM + PostgreSQL + React 19 + Zustand + Vite + Vitest + Docker Compose + Obsidian Local REST API

---

## File Structure Map

**Backend model and migration**
- Create: `backend/internal/model/review.go`
  - 定义 `ReviewSession`、`ReviewTurn` 两张表及枚举常量。
- Modify: `backend/internal/infra/db/db.go`
  - 将 `ReviewSession`、`ReviewTurn` 加入 `AutoMigrate`。

**Backend review domain**
- Create: `backend/internal/domain/review/dto.go`
  - 定义 `today/pick/notes/sessions/respond/hint/finish` 的请求响应结构。
- Create: `backend/internal/domain/review/repository.go`
  - 封装 session、turn 和最近复习记录查询。
- Create: `backend/internal/domain/review/service.go`
  - 编排题卡推荐、手动选文列表、session 生命周期和最终反馈。
- Create: `backend/internal/domain/review/handler.go`
  - 暴露 Gin handler，并统一做鉴权用户提取、参数绑定和错误返回。
- Create: `backend/internal/domain/review/note_source.go`
  - 负责从 Obsidian 拉取 `concepts` 笔记、解析 frontmatter、过滤 seed 页与系统页。
- Create: `backend/internal/domain/review/frontmatter.go`
  - 提供最小 frontmatter 解析与 `review` 配置读取。
- Create: `backend/internal/domain/review/picker.go`
  - 负责 `today`、`pick`、`notes` 三类入口的选题规则。
- Create: `backend/internal/domain/review/session_builder.go`
  - 会话创建时生成训练快照、初始提示和第一轮问题。
- Create: `backend/internal/domain/review/feedback_builder.go`
  - 负责阶段反馈和最终总结的结构化输出。

**Backend transport and DI**
- Modify: `backend/internal/transport/http/v1/routes.go`
  - 注册 `/api/v1/review/*` 路由，并将 review handler 纳入 `Handlers`。
- Modify: `backend/internal/transport/http/v1/routes_test.go`
  - 增加 review 路由可达性测试与 handler 缺失校验。
- Modify: `backend/cmd/server/main.go`
  - 完成 `reviewRepo -> reviewService -> reviewHandler` 依赖装配。

**Backend reusable Obsidian integration**
- Modify: `backend/internal/service/obsidian_store.go`
  - 若需要，把 `ObsidianStore` 继续作为 review 领域的只读依赖入口。
- Modify: `backend/internal/service/obsidian_rest_store.go`
  - 若需要，补一个可被 review 复用的构造入口，避免仅在 `obsidian_export.go` 中使用。

**Backend tests**
- Create: `backend/internal/domain/review/frontmatter_test.go`
- Create: `backend/internal/domain/review/note_source_test.go`
- Create: `backend/internal/domain/review/picker_test.go`
- Create: `backend/internal/domain/review/service_test.go`
- Create: `backend/internal/domain/review/handler_test.go`

**Frontend view shell**
- Modify: `frontend/src/store/blogStore.ts`
  - 扩展 `currentView` 为 `generator | dashboard | knowledge-review`。
- Modify: `frontend/src/App.tsx`
  - 接入 `KnowledgeReview` 主视图。
- Modify: `frontend/src/components/Sidebar.tsx`
  - 新增“知识漫游复习”入口。

**Frontend review feature**
- Create: `frontend/src/services/review.ts`
  - 封装所有 review JSON 接口。
- Create: `frontend/src/store/reviewStore.ts`
  - 管理题卡、候选列表、session、turns、当前模式和最近记录。
- Create: `frontend/src/pages/KnowledgeReview.tsx`
  - 负责页面主编排。
- Create: `frontend/src/components/review/ReviewEntryCards.tsx`
  - 展示今日推荐、随机抽题和手动选文入口。
- Create: `frontend/src/components/review/ReviewNotePicker.tsx`
  - 手动选择文章的列表/抽屉。
- Create: `frontend/src/components/review/ReviewSessionCard.tsx`
  - 展示当前题卡、模式选择、回答输入、提示按钮和结束动作。
- Create: `frontend/src/components/review/ReviewHistoryList.tsx`
  - 展示最近复习记录。
- Create: `frontend/src/hooks/useKnowledgeReview.ts`
  - 编排页面加载、创建 session、提交回答、请求提示、结束训练。

**Frontend tests**
- Create: `frontend/src/services/review.test.ts`
- Create: `frontend/src/store/reviewStore.test.ts`
- Create: `frontend/src/pages/knowledgeReviewViewState.test.ts`

**Docs sync**
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`
- Modify: `.trae/documents/InkWords_Database.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_PRD.md`
- Modify: `README.md`

### Task 1: 建立 Review 数据模型与数据库迁移

**Files:**
- Create: `backend/internal/model/review.go`
- Modify: `backend/internal/infra/db/db.go`
- Test: `backend/internal/domain/review/service_test.go`

- [ ] **Step 1: 先写数据库模型的失败测试**

```go
func TestReviewSessionBeforeCreate_AssignsUUID(t *testing.T) {
	session := model.ReviewSession{}
	err := session.BeforeCreate(&gorm.DB{})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, session.ID)
}

func TestInitDB_AutoMigratesReviewTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&model.ReviewSession{}, &model.ReviewTurn{})
	require.NoError(t, err)
	require.True(t, db.Migrator().HasTable(&model.ReviewSession{}))
	require.True(t, db.Migrator().HasTable(&model.ReviewTurn{}))
}
```

- [ ] **Step 2: 运行测试，确认当前实现失败**

Run: `cd backend && go test ./internal/domain/review -run ReviewSession -v`  
Expected: FAIL，提示 `model.ReviewSession`、`model.ReviewTurn` 未定义。

- [ ] **Step 3: 新增 review 模型**

```go
package model

type ReviewSession struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID            uuid.UUID      `gorm:"type:uuid;index:idx_review_sessions_user_note_created;not null" json:"user_id"`
	NotePath          string         `gorm:"type:text;not null;index:idx_review_sessions_user_note_created" json:"note_path"`
	NoteTitle         string         `gorm:"type:varchar(255);not null" json:"note_title"`
	SourceTitle       string         `gorm:"type:varchar(255)" json:"source_title"`
	EntryType         string         `gorm:"type:varchar(32);not null" json:"entry_type"`
	Mode              string         `gorm:"type:varchar(32);not null" json:"mode"`
	Status            string         `gorm:"type:varchar(32);not null;index" json:"status"`
	ReviewReason      string         `gorm:"type:text" json:"review_reason"`
	EstimatedMinutes  int            `gorm:"type:integer;default:0" json:"estimated_minutes"`
	ContentDigest     string         `gorm:"type:text" json:"content_digest"`
	SummarySnapshot   string         `gorm:"type:text" json:"summary_snapshot"`
	KeyPointsSnapshot datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"key_points_snapshot"`
	MetadataSnapshot  datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"metadata_snapshot"`
	HintUsedCount     int            `gorm:"type:integer;default:0" json:"hint_used_count"`
	MaxHintCount      int            `gorm:"type:integer;default:2" json:"max_hint_count"`
	TurnCount         int            `gorm:"type:integer;default:0" json:"turn_count"`
	FinalSummary      string         `gorm:"type:text" json:"final_summary"`
	Strengths         datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"strengths"`
	Gaps              datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"gaps"`
	NextFocus         datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"next_focus"`
	FeedbackTags      datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"feedback_tags"`
	StartedAt         time.Time      `json:"started_at"`
	CompletedAt       *time.Time     `json:"completed_at"`
	AbandonedAt       *time.Time     `json:"abandoned_at"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

type ReviewTurn struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	SessionID      uuid.UUID      `gorm:"type:uuid;not null;index:idx_review_turns_session_turn,unique" json:"session_id"`
	TurnIndex      int            `gorm:"type:integer;not null;index:idx_review_turns_session_turn,unique" json:"turn_index"`
	Role           string         `gorm:"type:varchar(16);not null" json:"role"`
	TurnType       string         `gorm:"type:varchar(32);not null" json:"turn_type"`
	Content        string         `gorm:"type:text;not null" json:"content"`
	EvaluationTags datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"evaluation_tags"`
	ExtraPayload   datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"extra_payload"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}
```

- [ ] **Step 4: 将 review 表加入 AutoMigrate**

```go
err = DB.AutoMigrate(
	&model.User{},
	&model.Blog{},
	&model.OAuthToken{},
	&model.UserPromptSettings{},
	&model.ReviewSession{},
	&model.ReviewTurn{},
)
```

- [ ] **Step 5: 运行数据库相关测试**

Run: `cd backend && go test ./internal/domain/review ./internal/infra/db -v`  
Expected: PASS，review 表能被识别并自动迁移。

- [ ] **Step 6: 提交模型与迁移变更**

```bash
git add backend/internal/model/review.go backend/internal/infra/db/db.go
git commit -m "feat(review): add review session persistence models"
```

### Task 2: 封装 Obsidian 复习笔记读取、frontmatter 解析与过滤器

**Files:**
- Create: `backend/internal/domain/review/frontmatter.go`
- Create: `backend/internal/domain/review/frontmatter_test.go`
- Create: `backend/internal/domain/review/note_source.go`
- Create: `backend/internal/domain/review/note_source_test.go`
- Modify: `backend/internal/service/obsidian_rest_store.go`
- Test: `backend/internal/domain/review/frontmatter_test.go`
- Test: `backend/internal/domain/review/note_source_test.go`

- [ ] **Step 1: 先写失败测试，锁定 seed 页和系统页过滤规则**

```go
func TestParseReviewFrontmatter_ReadsOptionalReviewConfig(t *testing.T) {
	content := `---
type: concept
title: "并发控制与速率限制"
review:
  enabled: true
  preferred_mode: light_recall
  exclude_from_random: false
---

# 并发控制与速率限制

正文内容`

	meta, body := parseFrontmatter(content)
	require.Equal(t, "concept", meta.Type)
	require.Equal(t, "并发控制与速率限制", meta.Title)
	require.True(t, meta.Review.Enabled)
	require.Equal(t, "light_recall", meta.Review.PreferredMode)
	require.Contains(t, body, "正文内容")
}

func TestIsEligibleReviewNote_FiltersSeedAndIndexes(t *testing.T) {
	require.False(t, IsEligibleReviewNote("wiki/index.md", ReviewFrontmatter{Type: "meta"}, "# index"))
	require.False(t, IsEligibleReviewNote("wiki/concepts/种子页.md", ReviewFrontmatter{Type: "concept"}, "Context extracted from [[foo]]"))
	require.True(t, IsEligibleReviewNote("wiki/concepts/并发控制与速率限制.md", ReviewFrontmatter{Type: "concept"}, strings.Repeat("有效正文", 50)))
}
```

- [ ] **Step 2: 运行测试，确认当前实现失败**

Run: `cd backend && go test ./internal/domain/review -run 'ParseReviewFrontmatter|EligibleReviewNote' -v`  
Expected: FAIL，提示 frontmatter 解析函数和过滤器未实现。

- [ ] **Step 3: 实现最小 frontmatter 解析器**

```go
type ReviewFrontmatter struct {
	Type   string
	Title  string
	Review struct {
		Enabled           bool
		PreferredMode     string
		ExcludeFromRandom bool
		MinIntervalDays   int
	}
}

func parseFrontmatter(content string) (ReviewFrontmatter, string) {
	if !strings.HasPrefix(content, "---\n") {
		return ReviewFrontmatter{}, content
	}
	parts := strings.SplitN(content, "\n---\n", 2)
	if len(parts) != 2 {
		return ReviewFrontmatter{}, content
	}
	var meta ReviewFrontmatter
	_ = yaml.Unmarshal([]byte(strings.TrimPrefix(parts[0], "---\n")), &meta)
	return meta, parts[1]
}
```

- [ ] **Step 4: 实现复习笔记源与过滤逻辑**

```go
type ReviewNote struct {
	NotePath      string
	Title         string
	SourceTitle   string
	Body          string
	PreferredMode string
}

func IsEligibleReviewNote(path string, meta ReviewFrontmatter, body string) bool {
	if strings.HasSuffix(path, "_index.md") || path == "wiki/index.md" || path == "wiki/hot.md" || path == "wiki/log.md" {
		return false
	}
	if meta.Type != "concept" {
		return false
	}
	if meta.Review.Enabled == false && strings.Contains(body, "Context extracted from [[") {
		return false
	}
	trimmed := strings.TrimSpace(body)
	if len([]rune(trimmed)) < 120 {
		return false
	}
	if strings.HasPrefix(trimmed, "Context extracted from [[") {
		return false
	}
	return !meta.Review.ExcludeFromRandom
}
```

- [ ] **Step 5: 让 review 领域可以复用 Obsidian Store**

```go
func NewReviewNoteSource(store service.ObsidianStore, rootDir string) *ReviewNoteSource {
	return &ReviewNoteSource{store: store, rootDir: rootDir}
}
```

- [ ] **Step 6: 运行过滤与列表测试**

Run: `cd backend && go test ./internal/domain/review -run 'Frontmatter|NoteSource' -v`  
Expected: PASS，系统页、索引页和 seed 页会被排除，只保留有效 concept 笔记。

- [ ] **Step 7: 提交 Obsidian 复习源能力**

```bash
git add backend/internal/domain/review/frontmatter.go \
  backend/internal/domain/review/frontmatter_test.go \
  backend/internal/domain/review/note_source.go \
  backend/internal/domain/review/note_source_test.go \
  backend/internal/service/obsidian_rest_store.go
git commit -m "feat(review): add obsidian review note source"
```

### Task 3: 实现 today、pick、notes 三类题卡入口

**Files:**
- Create: `backend/internal/domain/review/dto.go`
- Create: `backend/internal/domain/review/repository.go`
- Create: `backend/internal/domain/review/picker.go`
- Create: `backend/internal/domain/review/service.go`
- Create: `backend/internal/domain/review/picker_test.go`
- Create: `backend/internal/domain/review/service_test.go`
- Test: `backend/internal/domain/review/picker_test.go`
- Test: `backend/internal/domain/review/service_test.go`

- [ ] **Step 1: 先写失败测试，锁定 today、pick、notes 的行为**

```go
func TestPicker_PickToday_PrefersUnreviewed(t *testing.T) {
	notes := []ReviewNote{
		{NotePath: "wiki/concepts/a.md", Title: "A"},
		{NotePath: "wiki/concepts/b.md", Title: "B"},
	}
	stats := map[string]ReviewItemState{
		"wiki/concepts/a.md": {CompletedCount: 2},
	}

	got := PickToday(notes, stats, time.Now())
	require.Equal(t, "wiki/concepts/b.md", got.NotePath)
}

func TestService_ListNotes_UsesKeywordFilter(t *testing.T) {
	svc := newTestReviewServiceWithNotes([]ReviewNote{
		{NotePath: "wiki/concepts/并发控制与速率限制.md", Title: "并发控制与速率限制"},
		{NotePath: "wiki/concepts/前端状态管理.md", Title: "前端状态管理"},
	})

	resp, err := svc.ListNotes(context.Background(), uuid.New(), ListNotesQuery{Query: "并发"})
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	require.Equal(t, "并发控制与速率限制", resp.Items[0].Title)
}
```

- [ ] **Step 2: 运行测试，确认当前实现失败**

Run: `cd backend && go test ./internal/domain/review -run 'Picker|ListNotes' -v`  
Expected: FAIL，提示 picker 和 list DTO 未实现。

- [ ] **Step 3: 定义 DTO 与最小 repository 接口**

```go
type ReviewCardResponse struct {
	NotePath         string   `json:"note_path"`
	Title            string   `json:"title"`
	SourceTitle      string   `json:"source_title"`
	ReviewReason     string   `json:"review_reason"`
	EstimatedMinutes int      `json:"estimated_minutes"`
	AvailableModes   []string `json:"available_modes"`
}

type ListNotesQuery struct {
	Query       string
	SeriesTitle string
	Page        int
	PageSize    int
}

type Repository interface {
	GetRecentSessions(ctx context.Context, userID uuid.UUID, limit int) ([]model.ReviewSession, error)
	CreateSession(ctx context.Context, session *model.ReviewSession) error
}
```

- [ ] **Step 4: 实现受控随机 picker**

```go
func PickToday(notes []ReviewNote, stats map[string]ReviewItemState, now time.Time) ReviewNote {
	for _, note := range notes {
		if stats[note.NotePath].CompletedCount == 0 {
			return note
		}
	}
	sort.SliceStable(notes, func(i, j int) bool {
		return stats[notes[i].NotePath].LastReviewedAt.Before(stats[notes[j].NotePath].LastReviewedAt)
	})
	return notes[0]
}

func PickRandom(notes []ReviewNote, recent map[string]bool) ReviewNote {
	for _, note := range notes {
		if !recent[note.NotePath] {
			return note
		}
	}
	return notes[0]
}
```

- [ ] **Step 5: 实现 service 的 `GetTodayCard`、`PickRandomCard`、`ListNotes`**

```go
func (s *Service) GetTodayCard(ctx context.Context, userID uuid.UUID) (ReviewCardResponse, error) {
	notes, err := s.noteSource.ListEligibleNotes(ctx)
	if err != nil {
		return ReviewCardResponse{}, err
	}
	stats, err := s.loadItemState(ctx, userID)
	if err != nil {
		return ReviewCardResponse{}, err
	}
	picked := PickToday(notes, stats, s.now())
	return toReviewCardResponse(picked, "这是你最近导入但还没复习过的一篇内容。"), nil
}
```

- [ ] **Step 6: 运行入口相关测试**

Run: `cd backend && go test ./internal/domain/review -run 'Picker|Today|PickRandom|ListNotes' -v`  
Expected: PASS，三类入口都能按规则返回题卡或候选列表。

- [ ] **Step 7: 提交题卡与手动选文能力**

```bash
git add backend/internal/domain/review/dto.go \
  backend/internal/domain/review/repository.go \
  backend/internal/domain/review/picker.go \
  backend/internal/domain/review/picker_test.go \
  backend/internal/domain/review/service.go \
  backend/internal/domain/review/service_test.go
git commit -m "feat(review): add review card selection flows"
```

### Task 4: 实现 Review Session 生命周期与追问反馈

**Files:**
- Modify: `backend/internal/domain/review/service.go`
- Create: `backend/internal/domain/review/session_builder.go`
- Create: `backend/internal/domain/review/feedback_builder.go`
- Modify: `backend/internal/domain/review/repository.go`
- Create: `backend/internal/domain/review/service_test.go`
- Test: `backend/internal/domain/review/service_test.go`

- [ ] **Step 1: 先写失败测试，锁定 create/get/respond/hint/finish 的状态流转**

```go
func TestService_CreateSession_CapturesSnapshot(t *testing.T) {
	svc := newTestReviewServiceWithNotes([]ReviewNote{{
		NotePath:      "wiki/concepts/并发控制与速率限制.md",
		Title:         "并发控制与速率限制",
		Body:          strings.Repeat("正文内容", 80),
		PreferredMode: "light_recall",
	}})

	resp, err := svc.CreateSession(context.Background(), uuid.New(), CreateSessionRequest{
		NotePath:  "wiki/concepts/并发控制与速率限制.md",
		Mode:      "light_recall",
		EntryType: "manual_select",
	})
	require.NoError(t, err)
	require.Equal(t, "created", resp.Status)
	require.NotEmpty(t, resp.InitialHints)
}

func TestService_RespondDetailedQA_AdvancesThreeRounds(t *testing.T) {
	session := seedDetailedQASession(t)
	resp, err := session.Service.Respond(context.Background(), session.UserID, session.ID, RespondRequest{Answer: "这是主旨"})
	require.NoError(t, err)
	require.Equal(t, "in_progress", resp.SessionStatus)
	require.NotEmpty(t, resp.NextQuestion)
}

func TestService_RequestHint_StopsAtMaxCount(t *testing.T) {
	session := seedLightRecallSession(t)
	_, err := session.Service.RequestHint(context.Background(), session.UserID, session.ID)
	require.NoError(t, err)
	_, err = session.Service.RequestHint(context.Background(), session.UserID, session.ID)
	require.NoError(t, err)
	_, err = session.Service.RequestHint(context.Background(), session.UserID, session.ID)
	require.ErrorContains(t, err, "提示次数已用尽")
}
```

- [ ] **Step 2: 运行测试，确认当前实现失败**

Run: `cd backend && go test ./internal/domain/review -run 'CreateSession|RespondDetailedQA|RequestHint' -v`  
Expected: FAIL，提示 session builder、feedback builder 和 repository 方法未实现。

- [ ] **Step 3: 实现 session 创建与快照构建**

```go
func buildSessionSnapshot(note ReviewNote) (string, []string) {
	body := strings.TrimSpace(note.Body)
	if len([]rune(body)) > 800 {
		body = string([]rune(body)[:800])
	}
	keyPoints := []string{
		"这篇内容主要在解决什么问题",
		"有哪些关键概念或步骤",
		"有没有一个可以迁移到别处的例子",
	}
	return body, keyPoints
}

func openingPrompt(mode string) string {
	if mode == "detailed_qa" {
		return "先别看原文，我们从主线开始，一步一步把它讲清楚。"
	}
	return "先别看原文，试着用自己的话讲讲这篇内容。你不需要一字不差，只要抓住主线。"
}
```

- [ ] **Step 4: 实现 respond/hint/finish 的最小业务规则**

```go
func nextDetailedQuestion(turnCount int) string {
	switch turnCount {
	case 0:
		return "这篇文章最核心在讲什么？"
	case 1:
		return "它的关键概念、步骤或关系是什么？"
	default:
		return "如果让你把它讲给一个新手，你会怎么解释？"
	}
}

func buildFinalFeedback(answer string) FinalFeedback {
	return FinalFeedback{
		Summary:   "这次已经完成一次有效复习。",
		Strengths: []string{"已经尝试主动回忆并输出主线"},
		Gaps:      []string{"还可以补一个更具体的例子或迁移场景"},
		NextFocus: []string{"下次优先讲清楚为什么这样设计"},
	}
}
```

- [ ] **Step 5: 为 session/turn 持久化补齐 repository 方法**

```go
func (r *GormRepository) AppendTurn(ctx context.Context, turn *model.ReviewTurn) error {
	return r.db.WithContext(ctx).Create(turn).Error
}

func (r *GormRepository) UpdateSession(ctx context.Context, session *model.ReviewSession) error {
	return r.db.WithContext(ctx).Save(session).Error
}
```

- [ ] **Step 6: 运行 session 生命周期测试**

Run: `cd backend && go test ./internal/domain/review -run 'CreateSession|Respond|Hint|Finish' -v`  
Expected: PASS，三种模式状态推进、提示上限与最终反馈都符合预期。

- [ ] **Step 7: 提交 session 生命周期能力**

```bash
git add backend/internal/domain/review/service.go \
  backend/internal/domain/review/session_builder.go \
  backend/internal/domain/review/feedback_builder.go \
  backend/internal/domain/review/repository.go \
  backend/internal/domain/review/service_test.go
git commit -m "feat(review): add review session lifecycle"
```

### Task 5: 接入 Gin 路由、Handler 与依赖注入

**Files:**
- Create: `backend/internal/domain/review/handler.go`
- Modify: `backend/internal/transport/http/v1/routes.go`
- Modify: `backend/internal/transport/http/v1/routes_test.go`
- Modify: `backend/cmd/server/main.go`
- Create: `backend/internal/domain/review/handler_test.go`
- Test: `backend/internal/domain/review/handler_test.go`
- Test: `backend/internal/transport/http/v1/routes_test.go`

- [ ] **Step 1: 先写失败测试，锁定 review 路由与 handler 返回契约**

```go
func TestHandler_GetTodayCard_Returns200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		c.Next()
	})

	h := NewHandler(newStubService())
	r.GET("/api/v1/review/today", h.GetTodayCard)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/review/today", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
```

- [ ] **Step 2: 运行测试，确认当前实现失败**

Run: `cd backend && go test ./internal/domain/review ./internal/transport/http/v1 -run 'Review|RoutesAreReachable' -v`  
Expected: FAIL，提示 review handler 未注册且 routes 缺少 review group。

- [ ] **Step 3: 实现 review handler**

```go
func (h *Handler) GetTodayCard(c *gin.Context) {
	uid, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "未授权的访问", "data": nil})
		return
	}
	userID := uid.(uuid.UUID)
	resp, err := h.service.GetTodayCard(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "success", "data": resp})
}
```

- [ ] **Step 4: 注册 review 路由与 DI**

```go
type ReviewHandlers struct {
	GetTodayCard gin.HandlerFunc
	PickRandom   gin.HandlerFunc
	ListNotes    gin.HandlerFunc
	CreateSession gin.HandlerFunc
	GetSession   gin.HandlerFunc
	Respond      gin.HandlerFunc
	RequestHint  gin.HandlerFunc
	Finish       gin.HandlerFunc
}
```

```go
reviewGroup := v1.Group("/review")
reviewGroup.Use(authMiddleware)
{
	reviewGroup.GET("/today", handlers.Review.GetTodayCard)
	reviewGroup.POST("/pick", handlers.Review.PickRandom)
	reviewGroup.GET("/notes", handlers.Review.ListNotes)
	reviewGroup.POST("/sessions", handlers.Review.CreateSession)
	reviewGroup.GET("/sessions/:id", handlers.Review.GetSession)
	reviewGroup.POST("/sessions/:id/respond", handlers.Review.Respond)
	reviewGroup.POST("/sessions/:id/hint", handlers.Review.RequestHint)
	reviewGroup.POST("/sessions/:id/finish", handlers.Review.Finish)
}
```

- [ ] **Step 5: 运行 review handler 与 routes 测试**

Run: `cd backend && go test ./internal/domain/review ./internal/transport/http/v1 -v`  
Expected: PASS，review 路由可达且返回统一 JSON 结构。

- [ ] **Step 6: 提交 transport 与 DI 变更**

```bash
git add backend/internal/domain/review/handler.go \
  backend/internal/domain/review/handler_test.go \
  backend/internal/transport/http/v1/routes.go \
  backend/internal/transport/http/v1/routes_test.go \
  backend/cmd/server/main.go
git commit -m "feat(review): wire review handlers into transport"
```

### Task 6: 新增前端主视图、服务层与状态管理

**Files:**
- Modify: `frontend/src/store/blogStore.ts`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/Sidebar.tsx`
- Create: `frontend/src/services/review.ts`
- Create: `frontend/src/services/review.test.ts`
- Create: `frontend/src/store/reviewStore.ts`
- Create: `frontend/src/store/reviewStore.test.ts`
- Test: `frontend/src/services/review.test.ts`
- Test: `frontend/src/store/reviewStore.test.ts`

- [ ] **Step 1: 先写失败测试，锁定 view 切换与 review service 请求**

```ts
it('stores knowledge-review as a valid current view', () => {
  useBlogStore.getState().setCurrentView('knowledge-review')
  expect(useBlogStore.getState().currentView).toBe('knowledge-review')
  expect(useBlogStore.getState().selectedBlog).toBeNull()
})

it('calls GET /api/v1/review/notes with query params', async () => {
  global.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: async () => ({ code: 200, data: { items: [], total: 0, page: 1, page_size: 20 } }),
  } as Response)

  await reviewService.listNotes({ query: '并发', page: 1, pageSize: 20 })

  expect(fetch).toHaveBeenCalledWith('/api/v1/review/notes?query=%E5%B9%B6%E5%8F%91&page=1&page_size=20', expect.anything())
})
```

- [ ] **Step 2: 运行测试，确认当前实现失败**

Run: `cd frontend && npm test -- src/services/review.test.ts src/store/reviewStore.test.ts`  
Expected: FAIL，提示 `knowledge-review` 不是合法视图且 `reviewService` 未定义。

- [ ] **Step 3: 扩展主视图并新增 review service**

```ts
currentView: 'generator' | 'dashboard' | 'knowledge-review'
setCurrentView: (view: 'generator' | 'dashboard' | 'knowledge-review') => void
```

```ts
export const reviewService = {
  async getToday() {
    return requestJson<ReviewCardResponse>('/api/v1/review/today')
  },
  async pickRandom() {
    return requestJson<ReviewCardResponse>('/api/v1/review/pick', { method: 'POST' })
  },
  async listNotes(params: { query?: string; page?: number; pageSize?: number }) {
    const search = new URLSearchParams()
    if (params.query) search.set('query', params.query)
    search.set('page', String(params.page ?? 1))
    search.set('page_size', String(params.pageSize ?? 20))
    return requestJson<ListNotesResponse>(`/api/v1/review/notes?${search.toString()}`)
  },
}
```

- [ ] **Step 4: 新增 review store**

```ts
interface ReviewState {
  todayCard: ReviewCardResponse | null
  randomCard: ReviewCardResponse | null
  noteOptions: ReviewNoteOption[]
  currentSession: ReviewSessionResponse | null
  selectedMode: 'light_recall' | 'detailed_qa'
  loadToday: () => Promise<void>
  loadRandom: () => Promise<void>
  loadNotes: (query?: string) => Promise<void>
  setSelectedMode: (mode: 'light_recall' | 'detailed_qa') => void
  reset: () => void
}
```

- [ ] **Step 5: 运行前端 service/store 测试**

Run: `cd frontend && npm test -- src/services/review.test.ts src/store/reviewStore.test.ts`  
Expected: PASS，review 接口封装与状态切换行为稳定。

- [ ] **Step 6: 提交前端基础壳与服务层**

```bash
git add frontend/src/store/blogStore.ts \
  frontend/src/App.tsx \
  frontend/src/components/Sidebar.tsx \
  frontend/src/services/review.ts \
  frontend/src/services/review.test.ts \
  frontend/src/store/reviewStore.ts \
  frontend/src/store/reviewStore.test.ts
git commit -m "feat(review): add review view shell and services"
```

### Task 7: 实现知识漫游复习页面、会话交互与最近记录

**Files:**
- Create: `frontend/src/hooks/useKnowledgeReview.ts`
- Create: `frontend/src/pages/KnowledgeReview.tsx`
- Create: `frontend/src/components/review/ReviewEntryCards.tsx`
- Create: `frontend/src/components/review/ReviewNotePicker.tsx`
- Create: `frontend/src/components/review/ReviewSessionCard.tsx`
- Create: `frontend/src/components/review/ReviewHistoryList.tsx`
- Create: `frontend/src/pages/knowledgeReviewViewState.test.ts`
- Test: `frontend/src/pages/knowledgeReviewViewState.test.ts`

- [ ] **Step 1: 先写失败测试，锁定空状态与三种入口文案**

```ts
it('shows three review entry actions', () => {
  render(<KnowledgeReview />)
  expect(screen.getByText('开始今日复习')).toBeInTheDocument()
  expect(screen.getByText('随机抽一篇')).toBeInTheDocument()
  expect(screen.getByText('选择文章复习')).toBeInTheDocument()
})
```

- [ ] **Step 2: 运行测试，确认当前实现失败**

Run: `cd frontend && npm test -- src/pages/knowledgeReviewViewState.test.ts`  
Expected: FAIL，提示 `KnowledgeReview` 页面不存在。

- [ ] **Step 3: 实现页面主编排与三种入口**

```tsx
export function KnowledgeReview() {
  const { todayCard, randomCard, noteOptions, currentSession, loadToday, loadRandom, loadNotes } = useReviewStore()
  const { startSession, respond, requestHint, finish } = useKnowledgeReview()

  useEffect(() => {
    void loadToday()
  }, [loadToday])

  return (
    <div className="flex-1 h-full overflow-y-auto custom-scrollbar">
      <div className="max-w-5xl mx-auto px-4 py-12 space-y-8">
        <ReviewEntryCards onStartToday={loadToday} onPickRandom={loadRandom} onOpenPicker={() => void loadNotes()} />
        <ReviewNotePicker notes={noteOptions} onSelect={(notePath) => startSession(notePath, 'manual_select')} />
        <ReviewSessionCard session={currentSession} onRespond={respond} onRequestHint={requestHint} onFinish={finish} />
        <ReviewHistoryList />
      </div>
    </div>
  )
}
```

- [ ] **Step 4: 实现 hook，把 create/respond/hint/finish 串起来**

```ts
export function useKnowledgeReview() {
  const store = useReviewStore()

  return {
    async startSession(notePath: string, entryType: 'today' | 'manual_random' | 'manual_select') {
      const session = await reviewService.createSession({
        note_path: notePath,
        mode: store.selectedMode,
        entry_type: entryType,
      })
      store.setCurrentSession(session)
    },
    async respond(answer: string) {
      const session = store.currentSession
      if (!session) return
      const next = await reviewService.respond(session.session_id, { answer })
      store.setCurrentSession(next)
    },
  }
}
```

- [ ] **Step 5: 运行页面与交互测试**

Run: `cd frontend && npm test -- src/pages/knowledgeReviewViewState.test.ts src/services/review.test.ts src/store/reviewStore.test.ts`  
Expected: PASS，页面能显示三种入口，并触发对应的 service/store 行为。

- [ ] **Step 6: 提交页面与交互组件**

```bash
git add frontend/src/hooks/useKnowledgeReview.ts \
  frontend/src/pages/KnowledgeReview.tsx \
  frontend/src/components/review/ReviewEntryCards.tsx \
  frontend/src/components/review/ReviewNotePicker.tsx \
  frontend/src/components/review/ReviewSessionCard.tsx \
  frontend/src/components/review/ReviewHistoryList.tsx \
  frontend/src/pages/knowledgeReviewViewState.test.ts
git commit -m "feat(review): add knowledge roaming review page"
```

### Task 8: 文档同步、全量验证与 Docker 联调

**Files:**
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`
- Modify: `.trae/documents/InkWords_Database.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_PRD.md`
- Modify: `README.md`

- [ ] **Step 1: 更新 API 文档，加入 review 路由与请求响应字段**

```md
| `/api/v1/review/today` | GET | 获取今日推荐复习题卡 | JWT Bearer Token |
| `/api/v1/review/pick` | POST | 手动随机抽一篇可复习文章 | `{}` |
| `/api/v1/review/notes` | GET | 获取可手动选择复习的文章列表 | `query`, `page`, `page_size` |
| `/api/v1/review/sessions` | POST | 创建一次复习 session | `{ note_path, mode, entry_type }` |
```

- [ ] **Step 2: 更新架构/数据库/PRD/README 文档**

```md
- 新增 `backend/internal/domain/review` 领域，承接题卡推荐、会话生命周期、提示与反馈。
- PostgreSQL 新增 `review_sessions`、`review_turns` 两张表。
- 前端新增独立主视图 `知识漫游复习`，支持今日推荐、随机抽题、手动选文。
```

- [ ] **Step 3: 运行后端、前端和联调验证**

Run: `cd backend && go test ./...`  
Expected: PASS

Run: `cd frontend && npm test && npm run build`  
Expected: PASS

Run: `docker compose down && docker compose up -d --build`  
Expected: 容器全部启动成功，`http://localhost` 可访问。

- [ ] **Step 4: 做最小人工验收**

```text
1. 登录后在侧边栏能看到“知识漫游复习”入口
2. 进入页面能看到“开始今日复习 / 随机抽一篇 / 选择文章复习”
3. 手动选择文章后能创建 session
4. 回答、请求提示、结束训练都能返回中文反馈
5. 最近复习记录能展示刚完成的 session
```

- [ ] **Step 5: 提交文档与验证结果**

```bash
git add .trae/documents/InkWords_API.md \
  .trae/documents/InkWords_Architecture.md \
  .trae/documents/InkWords_Conversation_Log.md \
  .trae/documents/InkWords_Database.md \
  .trae/documents/InkWords_Development_Plan_and_Log.md \
  .trae/documents/InkWords_PRD.md \
  README.md
git commit -m "docs(review): document knowledge roaming review feature"
```

## Self-Review

### Spec coverage
- `知识漫游复习` 独立入口：由 Task 6、Task 7 覆盖。
- 三种入口（今日推荐、手动随机、手动选文）：由 Task 3、Task 7 覆盖。
- 两种训练模式：由 Task 4、Task 7 覆盖。
- Obsidian `concepts` 筛选、frontmatter 与 seed 页过滤：由 Task 2 覆盖。
- review session / turn 数据落库：由 Task 1、Task 4 覆盖。
- API、页面、数据库、文档同步：由 Task 5、Task 8 覆盖。

### Placeholder scan
- 已避免 `TODO/TBD/implement later` 占位。
- 所有任务均给出实际文件路径、代码骨架与命令。

### Type consistency
- 统一使用：
  - `entry_type`: `today | manual_random | manual_select`
  - `mode`: `light_recall | detailed_qa`
  - `status`: `created | in_progress | completed | abandoned`
- 前后端的 `note_path`、`session_id`、`ReviewCardResponse` 命名保持一致。
