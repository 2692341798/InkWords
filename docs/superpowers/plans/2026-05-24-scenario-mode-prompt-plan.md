# Scenario Mode Prompt Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 InkWords 落地 `scenario_mode` 场景切换能力，让电子书解读、开卷复习和小白教程三类任务在分析与生成阶段都能使用匹配的 Prompt 约束。

**Architecture:** 保持现有 `project/parse -> stream/analyze -> stream/generate` 主链路不变，在后端新增独立的 `scenario_mode` 枚举与 Prompt 组装层，由 `PromptRequirementsService` 统一合并系统约束、场景约束、风格约束和用户覆盖。前端只做最小必要改动：在生成器页增加场景选择并把字段透传到分析、单篇生成和系列生成请求。

**Tech Stack:** Go 1.25 + Gin + GORM + React 18 + Zustand + Vite + Vitest + Docker Compose

---

## File Structure Map

**Backend prompt model**
- Create: `backend/internal/prompt/scenario_mode.go`
  - 定义 `ScenarioMode` 枚举、校验与默认值逻辑。
- Create: `backend/internal/prompt/default_scenario_requirements.go`
  - 提供三个场景的默认 Prompt 约束。
- Modify: `backend/internal/prompt/default_requirements.go`
  - 保留现有 `article_style` 默认模板，避免回归。

**Backend prompt composition**
- Modify: `backend/internal/service/prompt_requirements.go`
  - 将 `Resolve` 扩展为接收 `scenario_mode + article_style` 并统一组装最终 requirements。
- Modify: `backend/internal/service/prompt_requirements_test.go`
  - 覆盖默认兜底、非法场景兜底、用户覆盖兼容等行为。

**Backend stream contracts and generation**
- Modify: `backend/internal/domain/stream/dto.go`
  - 新增 `ScenarioMode` 请求字段。
- Modify: `backend/internal/transport/http/v1/api/request_models_test.go`
  - 锁定新字段的 JSON tag。
- Modify: `backend/internal/service/generator.go`
  - 单篇生成接入 `scenario_mode`。
- Modify: `backend/internal/service/decomposition_generate.go`
  - 系列章节生成接入 `scenario_mode`。
- Modify: `backend/internal/service/decomposition_generate_intro.go`
  - 系列导读生成接入 `scenario_mode`。
- Modify: `backend/internal/service/decomposition_generate_outline.go`
  - 大纲生成按场景切换拆解策略。

**Backend HTTP flow**
- Modify: `backend/internal/domain/stream/handler.go`
  - Analyze/Generate 从请求体读取 `scenario_mode`，缺省时按来源类型兜底。
- Create: `backend/internal/domain/stream/scenario_mode_test.go`
  - 覆盖请求默认值与兜底映射。

**Frontend state and transport**
- Modify: `frontend/src/store/streamStore.ts`
  - 增加 `scenarioMode` 状态、setter 和 reset 默认值。
- Modify: `frontend/src/hooks/generator/useProjectAnalyzer.ts`
  - Analyze 请求透传 `scenario_mode`。
- Modify: `frontend/src/hooks/generator/useSeriesGenerator.ts`
  - 单篇生成、系列生成都透传 `scenario_mode`。
- Modify: `frontend/src/pages/Generator.tsx`
  - 增加中文场景选择入口并设置默认值。
- Create: `frontend/src/lib/scenarioMode.ts`
  - 收拢前端场景常量、标签和默认映射。
- Create: `frontend/src/lib/scenarioMode.test.ts`
  - 覆盖默认映射逻辑。

**Docs sync**
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_PRD.md`
- Modify: `README.md`
  - 同步接口字段、架构说明、用户入口与场景说明。

### Task 1: 建立后端场景枚举与 Prompt 组装层

**Files:**
- Create: `backend/internal/prompt/scenario_mode.go`
- Create: `backend/internal/prompt/default_scenario_requirements.go`
- Modify: `backend/internal/service/prompt_requirements.go`
- Modify: `backend/internal/service/prompt_requirements_test.go`
- Test: `backend/internal/service/prompt_requirements_test.go`

- [ ] **Step 1: 先写失败测试，锁定场景与风格的合并行为**

```go
func TestPromptRequirementsService_Resolve_UsesScenarioAndStyleDefaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserPromptSettings{}))

	svc := NewPromptRequirementsService(db)
	uid := uuid.New()

	got, err := svc.Resolve(context.Background(), uid, prompt.ScenarioModeBeginnerWalkthrough, prompt.ArticleStyleBeginnerTutorial)
	require.NoError(t, err)
	require.Contains(t, got, "零基础或初学者")
	require.Contains(t, got, "每一步都给出明确操作步骤")
}

func TestPromptRequirementsService_Resolve_FallsBackForInvalidScenario(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserPromptSettings{}))

	svc := NewPromptRequirementsService(db)
	uid := uuid.New()

	got, err := svc.Resolve(context.Background(), uid, prompt.ScenarioMode("bad"), prompt.ArticleStyleGeneral)
	require.NoError(t, err)
	require.Contains(t, got, "高质量技术博客")
	require.NotContains(t, got, "bad")
}

func TestPromptRequirementsService_Resolve_StillHonorsUserOverride(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserPromptSettings{}))

	uid := uuid.New()
	require.NoError(t, db.Create(&model.UserPromptSettings{
		UserID: uid,
		Overrides: datatypes.JSON([]byte(`{"beginner_tutorial":"CUSTOM STYLE"}`)),
	}).Error)

	svc := NewPromptRequirementsService(db)
	got, err := svc.Resolve(context.Background(), uid, prompt.ScenarioModeBeginnerWalkthrough, prompt.ArticleStyleBeginnerTutorial)
	require.NoError(t, err)
	require.Contains(t, got, "零基础或初学者")
	require.Contains(t, got, "CUSTOM STYLE")
}
```

- [ ] **Step 2: 运行测试，确认当前实现失败**

Run: `cd backend && go test ./internal/service -run PromptRequirementsService -v`  
Expected: FAIL，提示 `ScenarioMode` 未定义且 `Resolve` 参数个数不匹配。

- [ ] **Step 3: 增加 `ScenarioMode` 枚举与默认场景 Prompt**

```go
package prompt

type ScenarioMode string

const (
	ScenarioModeEbookInterpretation ScenarioMode = "ebook_interpretation"
	ScenarioModeOpenBookExamReview  ScenarioMode = "open_book_exam_review"
	ScenarioModeBeginnerWalkthrough ScenarioMode = "beginner_walkthrough"
)

func (m ScenarioMode) IsValid() bool {
	switch m {
	case ScenarioModeEbookInterpretation, ScenarioModeOpenBookExamReview, ScenarioModeBeginnerWalkthrough:
		return true
	default:
		return false
	}
}

func DefaultScenarioModeForSource(sourceType string) ScenarioMode {
	switch sourceType {
	case "git":
		return ScenarioModeBeginnerWalkthrough
	default:
		return ScenarioModeEbookInterpretation
	}
}
```

```go
func DefaultScenarioRequirements(mode ScenarioMode) string {
	switch mode {
	case ScenarioModeOpenBookExamReview:
		return `你将面向开卷考试或备考复习场景输出内容。要求：
1. 优先整理考点、步骤、答题抓手、易错点和速查表
2. 少做大段原理推导，重点帮助读者快速翻查和直接作答
3. 对实验或实操内容优先输出步骤模板、命令模板或判断清单`
	case ScenarioModeBeginnerWalkthrough:
		return `你将面向零基础或初学者输出教程。要求：
1. 按准备环境、跑通项目、理解结构、分析主链路的顺序展开
2. 对关键命令、关键文件、关键代码路径给出可执行说明
3. 对常见报错提供定位思路与排查建议`
	default:
		return `你将面向电子书或长文本解读场景输出内容。要求：
1. 先交代原文主题、篇章位置与上下文关系
2. 提炼关键观点并做白话解释
3. 在合适位置加入代表性原文摘录与现实映射`
	}
}
```

- [ ] **Step 4: 改造 `PromptRequirementsService.Resolve`，按层拼装 requirements**

```go
func (s *PromptRequirementsService) Resolve(ctx context.Context, userID uuid.UUID, scenario prompt.ScenarioMode, style prompt.ArticleStyle) (string, error) {
	if !scenario.IsValid() {
		scenario = prompt.ScenarioModeEbookInterpretation
	}
	if !style.IsValid() {
		style = prompt.ArticleStyleGeneral
	}

	styleRequirements := prompt.DefaultRequirements(style)
	userStyleOverride := styleRequirements

	var row model.UserPromptSettings
	if err := s.db.WithContext(ctx).First(&row, "user_id = ?", userID).Error; err == nil && len(row.Overrides) > 0 {
		var overrides map[string]string
		if json.Unmarshal(row.Overrides, &overrides) == nil {
			if v, ok := overrides[string(style)]; ok {
				if v == "" {
					userStyleOverride = styleRequirements
				} else {
					userStyleOverride = v
				}
			}
		}
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return "", err
	}

	return strings.TrimSpace(strings.Join([]string{
		prompt.DefaultScenarioRequirements(scenario),
		userStyleOverride,
	}, "\n\n")), nil
}
```

- [ ] **Step 5: 运行测试，确认 Prompt 组装通过**

Run: `cd backend && go test ./internal/service -run PromptRequirementsService -v`  
Expected: PASS，能覆盖默认兜底、非法值兜底和用户覆盖兼容。

- [ ] **Step 6: 提交后端 Prompt 基础设施**

```bash
git add backend/internal/prompt/scenario_mode.go \
  backend/internal/prompt/default_scenario_requirements.go \
  backend/internal/service/prompt_requirements.go \
  backend/internal/service/prompt_requirements_test.go
git commit -m "feat(prompt): add scenario mode requirements"
```

### Task 2: 让分析与生成链路接入 `scenario_mode`

**Files:**
- Modify: `backend/internal/domain/stream/dto.go`
- Modify: `backend/internal/transport/http/v1/api/request_models_test.go`
- Modify: `backend/internal/domain/stream/handler.go`
- Modify: `backend/internal/service/generator.go`
- Modify: `backend/internal/service/decomposition_generate.go`
- Modify: `backend/internal/service/decomposition_generate_intro.go`
- Modify: `backend/internal/service/decomposition_generate_outline.go`
- Create: `backend/internal/domain/stream/scenario_mode_test.go`
- Test: `backend/internal/transport/http/v1/api/request_models_test.go`
- Test: `backend/internal/domain/stream/scenario_mode_test.go`

- [ ] **Step 1: 先写契约测试和默认值测试**

```go
func TestGenerateRequest_HasScenarioModeField(t *testing.T) {
	rt := reflect.TypeOf(streamdomain.GenerateRequest{})
	field, ok := rt.FieldByName("ScenarioMode")
	require.True(t, ok)
	assert.Equal(t, "scenario_mode", field.Tag.Get("json"))
}
```

```go
func TestNormalizeScenarioMode_DefaultsBySourceType(t *testing.T) {
	assert.Equal(t, prompt.ScenarioModeBeginnerWalkthrough, normalizeScenarioMode("", "git"))
	assert.Equal(t, prompt.ScenarioModeEbookInterpretation, normalizeScenarioMode("", "file"))
	assert.Equal(t, prompt.ScenarioModeOpenBookExamReview, normalizeScenarioMode(string(prompt.ScenarioModeOpenBookExamReview), "file"))
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `cd backend && go test ./internal/transport/http/v1/api ./internal/domain/stream -run ScenarioMode -v`  
Expected: FAIL，提示 `ScenarioMode` 字段和 `normalizeScenarioMode` 尚不存在。

- [ ] **Step 3: 扩展请求 DTO，并在 handler 做统一兜底**

```go
type GenerateRequest struct {
	SourceContent   string    `json:"source_content"`
	SourceType      string    `json:"source_type"`
	Outline         []Chapter `json:"outline"`
	GitURL          string    `json:"git_url"`
	SubDir          string    `json:"sub_dir"`
	SelectedModules []string  `json:"selected_modules"`
	SeriesTitle     string    `json:"series_title"`
	ParentID        string    `json:"parent_id"`
	ArticleStyle    string    `json:"article_style"`
	ScenarioMode    string    `json:"scenario_mode"`
}
```

```go
func normalizeScenarioMode(raw string, sourceType string) prompt.ScenarioMode {
	mode := prompt.ScenarioMode(raw)
	if mode.IsValid() {
		return mode
	}
	return prompt.DefaultScenarioModeForSource(sourceType)
}
```

- [ ] **Step 4: 修改单篇生成和系列生成的 service 签名，向 Prompt 组装透传场景**

```go
func (s *GeneratorService) GenerateBlogStream(
	ctx context.Context,
	userID uuid.UUID,
	sourceContent string,
	sourceType string,
	scenarioMode string,
	style string,
	chunkChan chan<- string,
	errChan chan<- error,
) {
	mode := prompt.ScenarioMode(scenarioMode)
	requirements := prompt.DefaultScenarioRequirements(mode) + "\n\n" + prompt.DefaultRequirements(prompt.ArticleStyleGeneral)
	if s.promptReq != nil {
		if resolved, err := s.promptReq.Resolve(ctx, userID, mode, prompt.ArticleStyle(style)); err == nil && resolved != "" {
			requirements = resolved
		}
	}
	// ... keep existing streaming flow
}
```

```go
func (s *DecompositionService) GenerateSeries(
	ctx context.Context,
	userID uuid.UUID,
	parentID uuid.UUID,
	seriesTitle string,
	outline []Chapter,
	sourceContent string,
	sourceType string,
	gitURL string,
	scenarioMode string,
	style string,
	progressChan chan<- string,
	errChan chan<- error,
) {
	mode := prompt.ScenarioMode(scenarioMode)
	// 在章节 Prompt 和导读 Prompt 中统一使用 mode
}
```

- [ ] **Step 5: 按场景拆分大纲生成 Prompt**

```go
func outlineScenarioHint(mode prompt.ScenarioMode) string {
	switch mode {
	case prompt.ScenarioModeOpenBookExamReview:
		return "请按考点、题型、实验步骤或速查结构拆分章节，优先帮助开卷考试快速定位。"
	case prompt.ScenarioModeBeginnerWalkthrough:
		return "请按学习路径拆分章节，优先覆盖环境准备、目录结构、关键主链路和常见排错。"
	default:
		return "请按篇章、主题脉络或核心观点拆分章节，保证解读性与连贯阅读体验。"
	}
}
```

- [ ] **Step 6: 运行后端聚焦测试**

Run: `cd backend && go test ./internal/transport/http/v1/api ./internal/domain/stream ./internal/service -run 'ScenarioMode|PromptRequirementsService' -v`  
Expected: PASS，接口字段和默认值测试全部通过。

- [ ] **Step 7: 运行生成链路回归测试**

Run: `cd backend && go test ./internal/domain/stream ./internal/service -v`  
Expected: PASS，无现有生成链路回归。

- [ ] **Step 8: 提交分析/生成接线**

```bash
git add backend/internal/domain/stream/dto.go \
  backend/internal/transport/http/v1/api/request_models_test.go \
  backend/internal/domain/stream/handler.go \
  backend/internal/domain/stream/scenario_mode_test.go \
  backend/internal/service/generator.go \
  backend/internal/service/decomposition_generate.go \
  backend/internal/service/decomposition_generate_intro.go \
  backend/internal/service/decomposition_generate_outline.go
git commit -m "feat(stream): wire scenario mode through analyze and generate"
```

### Task 3: 在前端增加场景状态与请求透传

**Files:**
- Create: `frontend/src/lib/scenarioMode.ts`
- Create: `frontend/src/lib/scenarioMode.test.ts`
- Modify: `frontend/src/store/streamStore.ts`
- Modify: `frontend/src/hooks/generator/useProjectAnalyzer.ts`
- Modify: `frontend/src/hooks/generator/useSeriesGenerator.ts`
- Modify: `frontend/src/pages/Generator.tsx`
- Test: `frontend/src/lib/scenarioMode.test.ts`

- [ ] **Step 1: 先写前端默认映射测试**

```ts
import { describe, expect, it } from 'vitest'
import { defaultScenarioModeForSource, scenarioModeLabelMap } from './scenarioMode'

describe('scenarioMode', () => {
  it('maps git sources to beginner walkthrough by default', () => {
    expect(defaultScenarioModeForSource('git')).toBe('beginner_walkthrough')
  })

  it('maps file sources to ebook interpretation by default', () => {
    expect(defaultScenarioModeForSource('file')).toBe('ebook_interpretation')
  })

  it('exposes chinese labels for all supported modes', () => {
    expect(scenarioModeLabelMap.open_book_exam_review).toBe('开卷复习')
  })
})
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `cd frontend && npm test -- src/lib/scenarioMode.test.ts`  
Expected: FAIL，提示 `scenarioMode.ts` 不存在。

- [ ] **Step 3: 新增前端场景常量与 store 状态**

```ts
export type ScenarioMode =
  | 'ebook_interpretation'
  | 'open_book_exam_review'
  | 'beginner_walkthrough'

export const scenarioModeLabelMap: Record<ScenarioMode, string> = {
  ebook_interpretation: '电子书解读',
  open_book_exam_review: '开卷复习',
  beginner_walkthrough: '小白教程',
}

export function defaultScenarioModeForSource(sourceType: 'git' | 'file' | null): ScenarioMode {
  return sourceType === 'git' ? 'beginner_walkthrough' : 'ebook_interpretation'
}
```

```ts
interface StreamState {
  // ...
  scenarioMode: ScenarioMode
  setScenarioMode: (mode: ScenarioMode) => void
}

scenarioMode: 'ebook_interpretation',
setScenarioMode: (mode) => set({ scenarioMode: mode }),
reset: () => set({
  // ...
  scenarioMode: 'ebook_interpretation',
})
```

- [ ] **Step 4: 在 Analyze / Generate 请求中透传 `scenario_mode`**

```ts
body: JSON.stringify({
  git_url: gitUrl,
  selected_modules: selectedModules,
  scenario_mode: store.scenarioMode,
})
```

```ts
body: JSON.stringify({
  source_type: store.sourceType,
  git_url: store.gitUrl,
  source_content: store.sourceContent,
  series_title: store.seriesTitle,
  outline: store.outline,
  parent_id: store.parentBlogId,
  scenario_mode: store.scenarioMode,
})
```

```ts
body: JSON.stringify({
  source_type: 'file',
  source_content: content,
  outline: [],
  scenario_mode: store.scenarioMode,
})
```

- [ ] **Step 5: 在生成器页面加入中文场景选择**

```tsx
<div className="space-y-3">
  <div className="text-sm font-medium text-zinc-700">创作场景</div>
  <div className="grid gap-3 md:grid-cols-3">
    {scenarioModeOptions.map((option) => (
      <button
        key={option.value}
        type="button"
        onClick={() => store.setScenarioMode(option.value)}
        className={cn(
          'rounded-xl border px-4 py-3 text-left transition-colors',
          store.scenarioMode === option.value
            ? 'border-zinc-900 bg-zinc-50'
            : 'border-zinc-200 bg-white hover:border-zinc-400',
        )}
      >
        <div className="text-sm font-semibold text-zinc-900">{option.label}</div>
        <div className="mt-1 text-xs text-zinc-500">{option.description}</div>
      </button>
    ))}
  </div>
</div>
```

- [ ] **Step 6: 运行前端聚焦测试**

Run: `cd frontend && npm test -- src/lib/scenarioMode.test.ts`  
Expected: PASS，默认值与中文标签映射正确。

- [ ] **Step 7: 提交前端透传与交互**

```bash
git add frontend/src/lib/scenarioMode.ts \
  frontend/src/lib/scenarioMode.test.ts \
  frontend/src/store/streamStore.ts \
  frontend/src/hooks/generator/useProjectAnalyzer.ts \
  frontend/src/hooks/generator/useSeriesGenerator.ts \
  frontend/src/pages/Generator.tsx
git commit -m "feat(frontend): add scenario mode selector"
```

### Task 4: 同步文档并做端到端验证

**Files:**
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_PRD.md`
- Modify: `README.md`

- [ ] **Step 1: 更新接口与架构文档**

```md
### `/api/v1/stream/analyze`
- 新增请求字段：`scenario_mode`
- 缺省兜底：
  - `git` -> `beginner_walkthrough`
  - `file` -> `ebook_interpretation`

### `/api/v1/stream/generate`
- 新增请求字段：`scenario_mode`
- 作用范围：
  - 单篇生成
  - 系列章节生成
  - 系列导读生成
```

- [ ] **Step 2: 更新产品与开发日志**

```md
- 2026-05-24：新增 `scenario_mode` 场景切换能力，支持电子书解读、开卷复习、小白教程三类 Prompt 模式，并贯通分析与生成链路。
```

- [ ] **Step 3: 运行聚焦测试套件**

Run: `cd backend && go test ./internal/service ./internal/domain/stream ./internal/transport/http/v1/api -v && cd ../frontend && npm test -- src/lib/scenarioMode.test.ts`  
Expected: PASS，后端与前端新增测试均通过。

- [ ] **Step 4: 重建容器并做手工验证**

Run: `docker compose down && docker compose up -d --build`  
Expected: 容器启动成功，应用可通过 `http://localhost` 访问。

Run through this checklist:

```text
1. 上传《孙子兵法》类 PDF，默认或手动选择“电子书解读”。
2. 确认大纲更偏向篇章/观点/解读，而不是代码教程结构。
3. 上传 ZIP 课件包，切换“开卷复习”。
4. 确认大纲或正文更偏向考点、步骤、易错点、速查形式。
5. 输入 Git 仓库，切换“小白教程”。
6. 确认大纲或正文更偏向环境、目录结构、主链路、排错。
```

- [ ] **Step 5: 对照暂存区并提交文档**

Run: `git diff --staged`  
Expected: 仅包含 `scenario_mode` 相关代码、测试与文档同步。

```bash
git add .trae/documents/InkWords_API.md \
  .trae/documents/InkWords_Architecture.md \
  .trae/documents/InkWords_Conversation_Log.md \
  .trae/documents/InkWords_Development_Plan_and_Log.md \
  .trae/documents/InkWords_PRD.md \
  README.md
git commit -m "docs: document scenario mode generation flow"
```

## Self-Review
- Spec coverage: 已覆盖 `scenario_mode` 枚举、Prompt 分层、分析与生成接线、前端场景选择、默认值兜底、测试与文档同步，没有遗漏规格稿中的首版范围。
- Placeholder scan: 计划中没有 `TODO`、`TBD`、`similar to` 之类占位词，每个任务都给出了具体文件、代码片段、命令和预期结果。
- Type consistency:
  - 后端统一使用 `ScenarioMode`、`ScenarioMode...` 枚举值和 `scenario_mode` JSON 字段。
  - 前端统一使用 `ScenarioMode` 联合类型和同名请求字段。
  - `PromptRequirementsService.Resolve` 在所有调用点都按 `(ctx, userID, scenario, style)` 新签名使用。
