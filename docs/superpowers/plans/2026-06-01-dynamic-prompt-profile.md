# Dynamic Prompt Profile Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为文件来源链路引入“先识别内容类型、再锁定动态提示词 profile、再生成大纲与正文”的机制，并彻底移除电子书模式中残留的“高级架构师”硬编码角色。

**Architecture:** 保持现有 `project/parse -> stream/analyze -> stream/generate` 三步主链路不变，只在文件来源的 Analyze 阶段新增一次轻量分类。后端新增 `prompt_profile` 单一来源与 `PromptProfileResolver`，由 Analyze 锁定 `resolved_prompt_profile`，Generate 与系列导读沿用同一个 profile；前端只新增最小必要状态与只读展示，不改变三步 UI 编排。

**Tech Stack:** Go 1.25 + Gin + GORM + DeepSeek SSE + React 18 + Zustand + Vite + Vitest + Docker Compose

---

## File Structure Map

**Backend prompt profile model**
- Create: `backend/internal/prompt/prompt_profile.go`
  - 定义 `PromptProfile`、`ResolvedPromptProfile`、profile 常量、按 `scenario_mode` 的 fallback 逻辑。
- Modify: `backend/internal/prompt/default_requirements.go`
  - 保持 `article_style` 默认模板，同时让 profile 生成要求能与 style 合并。

**Backend classification and prompt assembly**
- Create: `backend/internal/service/prompt_profile_resolver.go`
  - 封装“文件内容 -> `prompt_profile`”分类逻辑，失败时自动降级。
- Create: `backend/internal/service/prompt_profile_resolver_test.go`
  - 覆盖 JSON 解析、非法 key 降级、按场景 fallback。
- Modify: `backend/internal/service/prompt_requirements.go`
  - 将 `PromptProfile` 参与 requirements 组装。
- Modify: `backend/internal/service/prompt_requirements_test.go`
  - 锁定电子书场景下不再回退到“高质量技术博客/可独立复现”。

**Backend analyze/generate flow**
- Modify: `backend/internal/service/decomposition_service.go`
  - 为 `OutlineResult` 增加 `resolved_prompt_profile`。
- Modify: `backend/internal/service/decomposition_generate_outline.go`
  - Analyze 使用锁定后的 profile role 与 profile requirements。
- Modify: `backend/internal/service/generator.go`
  - 单篇生成不再硬编码“高级全栈架构师和技术博主”。
- Modify: `backend/internal/service/decomposition_generate_prompt_helpers.go`
  - 系列章节生成改为使用 profile role。
- Modify: `backend/internal/service/decomposition_generate_intro.go`
  - 系列导读沿用 Analyze 锁定的 profile。
- Modify: `backend/internal/service/decomposition_generate_outline_test.go`
  - 测试大纲 prompt 使用 profile role，而不是旧角色硬编码。
- Modify: `backend/internal/service/decomposition_generate_split_test.go`
  - 测试章节 prompt 使用锁定后的 profile role。

**Backend stream contracts**
- Modify: `backend/internal/domain/stream/dto.go`
  - 在 `GenerateRequest` 中增加 `PromptProfileKey` 和 `DocumentKind`。
- Modify: `backend/internal/domain/stream/service.go`
  - Analyze/Generate 透传新字段。
- Modify: `backend/internal/domain/stream/handler.go`
  - 标准化请求并保持旧客户端兼容。
- Modify: `backend/internal/domain/stream/scenario_mode_test.go`
  - 补充非法/缺省请求下的 fallback 行为。
- Modify: `backend/internal/transport/http/v1/api/request_models_test.go`
  - 锁定 JSON tag。

**Frontend state and transport**
- Modify: `frontend/src/store/streamStore.ts`
  - 增加 `resolvedPromptProfile`、`classificationStatus`、`classificationReason` 及 reset 逻辑。
- Modify: `frontend/src/hooks/generator/fileParserUtils.ts`
  - 定义前端侧 `ResolvedPromptProfile` 类型和 analyze 请求结构。
- Modify: `frontend/src/hooks/generator/useFileParser.ts`
  - Analyze 完成后写入锁定的 profile，并更新中文状态文案。
- Modify: `frontend/src/hooks/generator/useSeriesGenerator.ts`
  - 系列生成/单篇生成请求携带 `prompt_profile_key` 与 `document_kind`。
- Create: `frontend/src/hooks/generator/useSeriesGenerator.test.ts`
  - 锁定 generate request builder 新字段。
- Modify: `frontend/src/pages/generatorViewState.ts`
  - 生成大纲后计算“当前提示词类型”只读标签。
- Modify: `frontend/src/components/generator/GeneratorOutlineStage.tsx`
  - 展示当前锁定的 prompt profile 标签。
- Modify: `frontend/src/hooks/generator/fileAnalyzeRequest.test.ts`
  - 锁定 Analyze 请求与最新场景读取行为保持兼容。

**Docs sync**
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_PRD.md`
- Modify: `README.md`
  - 同步动态提示词 profile 机制、接口字段与交互口径。

### Task 1: 建立 Prompt Profile 模型与分类降级逻辑

**Files:**
- Create: `backend/internal/prompt/prompt_profile.go`
- Create: `backend/internal/service/prompt_profile_resolver.go`
- Create: `backend/internal/service/prompt_profile_resolver_test.go`
- Modify: `backend/internal/service/prompt_requirements.go`
- Modify: `backend/internal/service/prompt_requirements_test.go`
- Test: `backend/internal/service/prompt_profile_resolver_test.go`

- [ ] **Step 1: 先写失败测试，锁定 fallback 与非法 key 降级**

```go
package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"inkwords-backend/internal/prompt"
)

func TestFallbackPromptProfileForScenario_FileEbookUsesClassicInterpretation(t *testing.T) {
	profile := prompt.FallbackPromptProfileForScenario(prompt.ScenarioModeEbookInterpretation)

	require.Equal(t, prompt.PromptProfileClassicTextInterpretation, profile.Key)
	require.Equal(t, "经典文本解读", profile.DisplayName)
	require.Contains(t, profile.SystemRole, "文本解读")
}

func TestResolvePromptProfileKey_FallsBackForUnknownKey(t *testing.T) {
	profile := prompt.ResolvePromptProfileKey("bad_key", prompt.ScenarioModeEbookInterpretation)

	require.Equal(t, prompt.PromptProfileClassicTextInterpretation, profile.Key)
	require.Equal(t, "classic_text_interpretation", string(profile.Key))
}
```

- [ ] **Step 2: 运行测试，确认当前实现失败**

Run: `cd backend && go test ./internal/service -run 'PromptProfile|FallbackPromptProfile' -v`  
Expected: FAIL，提示 `FallbackPromptProfileForScenario` / `ResolvePromptProfileKey` / `PromptProfileClassicTextInterpretation` 未定义。

- [ ] **Step 3: 在 `prompt_profile.go` 中定义 profile 常量、结构体与 fallback**

```go
package prompt

type PromptProfileKey string

const (
	PromptProfileClassicTextInterpretation PromptProfileKey = "classic_text_interpretation"
	PromptProfilePsychologyCommunication   PromptProfileKey = "psychology_communication_book"
	PromptProfileHistoryThought            PromptProfileKey = "history_thought_book"
	PromptProfileLiteratureCommentary      PromptProfileKey = "literature_commentary_book"
	PromptProfileTechnicalManual           PromptProfileKey = "technical_manual_book"
	PromptProfileExamMaterialReview        PromptProfileKey = "exam_material_review"
)

type PromptProfile struct {
	Key                  PromptProfileKey `json:"key"`
	DisplayName          string           `json:"display_name"`
	DocumentKind         string           `json:"document_kind"`
	SystemRole           string           `json:"system_role"`
	AnalyzeRequirements  string           `json:"analyze_requirements"`
	GenerateRequirements string           `json:"generate_requirements"`
}

type ResolvedPromptProfile struct {
	Key          PromptProfileKey `json:"key"`
	DisplayName  string           `json:"display_name"`
	DocumentKind string           `json:"document_kind"`
	Reason       string           `json:"reason"`
}

var promptProfiles = map[PromptProfileKey]PromptProfile{
	PromptProfileClassicTextInterpretation: {
		Key:                  PromptProfileClassicTextInterpretation,
		DisplayName:          "经典文本解读",
		DocumentKind:         "classic_text",
		SystemRole:           "你是一位严谨的中文文本解读专家。",
		AnalyzeRequirements:  "请按原文自身篇章结构与主题脉络拆分章节，不要把内容强行改写成技术教程。",
		GenerateRequirements: "请围绕原文主题、背景、观点与代表性摘录展开白话解读，避免教程式开场白。",
	},
	PromptProfilePsychologyCommunication: {
		Key:                  PromptProfilePsychologyCommunication,
		DisplayName:          "心理学经典解读",
		DocumentKind:         "psychology_communication",
		SystemRole:           "你是一位擅长心理学与沟通主题的中文文本解读作者。",
		AnalyzeRequirements:  "请优先识别沟通冲突、感受、需要、表达方式等主题脉络，并按章节自然拆分。",
		GenerateRequirements: "请重点解释心理机制、沟通案例、概念间关系与现实场景，不要使用工程师身份自述。",
	},
}

func FallbackPromptProfileForScenario(mode ScenarioMode) PromptProfile {
	switch mode {
	case ScenarioModeOpenBookExamReview:
		return promptProfiles[PromptProfileExamMaterialReview]
	default:
		return promptProfiles[PromptProfileClassicTextInterpretation]
	}
}

func ResolvePromptProfileKey(key string, mode ScenarioMode) PromptProfile {
	if profile, ok := promptProfiles[PromptProfileKey(key)]; ok {
		return profile
	}
	return FallbackPromptProfileForScenario(mode)
}
```

- [ ] **Step 4: 新建 resolver，并把分类失败收口到 fallback**

```go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"inkwords-backend/internal/infra/llm"
	"inkwords-backend/internal/prompt"
)

type promptProfileLLM interface {
	GenerateJSON(ctx context.Context, model string, messages []llm.Message) (string, error)
}

type PromptProfileResolver struct {
	llmClient promptProfileLLM
}

func NewPromptProfileResolver(llmClient promptProfileLLM) *PromptProfileResolver {
	return &PromptProfileResolver{llmClient: llmClient}
}

func (r *PromptProfileResolver) ResolveForFile(ctx context.Context, sourceContent string, scenario prompt.ScenarioMode) (prompt.PromptProfile, prompt.ResolvedPromptProfile, error) {
	fallback := prompt.FallbackPromptProfileForScenario(scenario)
	if strings.TrimSpace(sourceContent) == "" || r.llmClient == nil {
		return fallback, prompt.ResolvedPromptProfile{
			Key:          fallback.Key,
			DisplayName:  fallback.DisplayName,
			DocumentKind: fallback.DocumentKind,
			Reason:       "文件内容为空或分类器不可用，已回退到默认提示词。",
		}, nil
	}

	runes := []rune(sourceContent)
	if len(runes) > 16000 {
		sourceContent = string(runes[:16000])
	}

	payload, err := r.llmClient.GenerateJSON(ctx, "deepseek-v4-flash", []llm.Message{
		{Role: "system", Content: "请只做内容类型识别，并返回严格 JSON，不要生成大纲。"},
		{Role: "user", Content: fmt.Sprintf("scenario_mode=%s\n\n以下是文件内容：\n%s", scenario, sourceContent)},
	})
	if err != nil {
		return fallback, prompt.ResolvedPromptProfile{
			Key:          fallback.Key,
			DisplayName:  fallback.DisplayName,
			DocumentKind: fallback.DocumentKind,
			Reason:       "内容分类失败，已回退到默认提示词。",
		}, nil
	}

	var result struct {
		PromptProfileKey string `json:"prompt_profile_key"`
		DocumentKind     string `json:"document_kind"`
		Reason           string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(payload), &result); err != nil {
		return fallback, prompt.ResolvedPromptProfile{
			Key:          fallback.Key,
			DisplayName:  fallback.DisplayName,
			DocumentKind: fallback.DocumentKind,
			Reason:       "分类结果无法解析，已回退到默认提示词。",
		}, nil
	}

	resolved := prompt.ResolvePromptProfileKey(result.PromptProfileKey, scenario)
	return resolved, prompt.ResolvedPromptProfile{
		Key:          resolved.Key,
		DisplayName:  resolved.DisplayName,
		DocumentKind: firstNonEmpty(result.DocumentKind, resolved.DocumentKind),
		Reason:       firstNonEmpty(result.Reason, "已根据文件内容自动匹配提示词。"),
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
```

- [ ] **Step 5: 让 `PromptRequirementsService` 支持 profile 优先组装**

```go
func (s *PromptRequirementsService) ResolveWithProfile(
	ctx context.Context,
	userID uuid.UUID,
	scenario prompt.ScenarioMode,
	style prompt.ArticleStyle,
	profile prompt.PromptProfile,
) (string, error) {
	base, err := s.Resolve(ctx, userID, scenario, style)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(strings.Join([]string{
		profile.GenerateRequirements,
		base,
	}, "\n\n")), nil
}
```

- [ ] **Step 6: 再跑测试，确认 resolver 与 requirements 都通过**

Run: `cd backend && go test ./internal/service -run 'PromptProfile|PromptRequirementsService' -v`  
Expected: PASS，且输出包含 `FallbackPromptProfileForScenario` 与 `ResolveWithProfile` 相关测试通过。

- [ ] **Step 7: 提交**

```bash
git add backend/internal/prompt/prompt_profile.go \
  backend/internal/service/prompt_profile_resolver.go \
  backend/internal/service/prompt_profile_resolver_test.go \
  backend/internal/service/prompt_requirements.go \
  backend/internal/service/prompt_requirements_test.go
git commit -m "feat(prompt): add dynamic prompt profile fallback"
```

### Task 2: 将 Analyze 与 Generate 全链路切到锁定后的 Prompt Profile

**Files:**
- Modify: `backend/internal/service/decomposition_service.go`
- Modify: `backend/internal/service/decomposition_generate_outline.go`
- Modify: `backend/internal/service/decomposition_generate_outline_test.go`
- Modify: `backend/internal/service/generator.go`
- Modify: `backend/internal/service/decomposition_generate_prompt_helpers.go`
- Modify: `backend/internal/service/decomposition_generate_intro.go`
- Modify: `backend/internal/service/decomposition_generate_split_test.go`
- Test: `backend/internal/service/decomposition_generate_outline_test.go`

- [ ] **Step 1: 先写失败测试，锁定不再出现“高级架构师”硬编码**

```go
func TestOutlineBaseInstructionForProfile_UsesProfileRoleForEbook(t *testing.T) {
	profile := prompt.ResolvePromptProfileKey("psychology_communication_book", prompt.ScenarioModeEbookInterpretation)

	systemRole, instruction := outlinePromptForProfile(prompt.ScenarioModeEbookInterpretation, profile)

	require.Contains(t, systemRole, "心理学")
	require.NotContains(t, instruction, "高级架构师")
	require.NotContains(t, systemRole, "高级架构师")
}

func TestBuildSeriesChapterMessages_UsesResolvedPromptProfileRole(t *testing.T) {
	svc := NewDecompositionService(nil)
	profile := prompt.ResolvePromptProfileKey("psychology_communication_book", prompt.ScenarioModeEbookInterpretation)

	messages, _, err := svc.buildSeriesChapterMessages(
		context.Background(),
		uuid.New(),
		Chapter{Title: "第二章", Summary: "逐章解读", Sort: 2},
		[]Chapter{{Title: "第二章", Summary: "逐章解读", Sort: 2}},
		0,
		"原文内容",
		"file",
		"",
		prompt.ScenarioModeEbookInterpretation,
		string(prompt.ArticleStyleGeneral),
		"",
		profile,
	)
	require.NoError(t, err)
	require.Contains(t, messages[0].Content, "心理学")
	require.NotContains(t, messages[0].Content, "高级全栈架构师")
}
```

- [ ] **Step 2: 运行测试，确认当前实现失败**

Run: `cd backend && go test ./internal/service -run 'OutlineBaseInstructionForProfile|BuildSeriesChapterMessages_UsesResolvedPromptProfileRole' -v`  
Expected: FAIL，提示 `outlinePromptForProfile` 未定义，且 `buildSeriesChapterMessages` 参数列表不匹配。

- [ ] **Step 3: 扩展 `OutlineResult`，把锁定后的 profile 带回前端**

```go
type OutlineResult struct {
	SeriesTitle           string                 `json:"series_title"`
	Chapters              []Chapter              `json:"chapters"`
	ParentID              string                 `json:"parent_id,omitempty"`
	ResolvedPromptProfile prompt.ResolvedPromptProfile `json:"resolved_prompt_profile"`
}
```

- [ ] **Step 4: 在 Analyze 中先分类，再用 profile role 构造大纲 prompt**

```go
func outlinePromptForProfile(mode prompt.ScenarioMode, profile prompt.PromptProfile) (string, string) {
	systemRole := profile.SystemRole
	instruction := strings.TrimSpace(strings.Join([]string{
		profile.AnalyzeRequirements,
		outlineBaseInstruction(mode),
		"输出必须是纯 JSON，包含 series_title 和 chapters。",
	}, "\n\n"))
	return systemRole, instruction
}

func (s *DecompositionService) GenerateOutline(
	ctx context.Context,
	sourceContent string,
	scenarioMode prompt.ScenarioMode,
	profile prompt.PromptProfile,
	resolved prompt.ResolvedPromptProfile,
	existingParent *model.Blog,
	existingChildren []model.Blog,
) (*OutlineResult, error) {
	systemRole, instruction := outlinePromptForProfile(scenarioMode, profile)
	messages := []llm.Message{
		{Role: "system", Content: systemRole + "\n\n原文内容如下：\n" + sourceContent},
		{Role: "user", Content: instruction},
	}
	// ...
	return &OutlineResult{
		SeriesTitle:           outline.SeriesTitle,
		Chapters:              outline.Chapters,
		ResolvedPromptProfile: resolved,
	}, nil
}
```

- [ ] **Step 5: 在单篇、章节、导读生成中统一改用锁定后的 profile**

```go
instruction := fmt.Sprintf(`请根据前面提供的源内容输出中文正文。

写作要求：
%s

硬性约束：
1. 禁止输出“好的，收到你的需求”“作为高级架构师”等对话式前言。
2. Mermaid 代码块禁止 style/classDef/linkStyle。
`, requirements)

messages := []llm.Message{
	{Role: "system", Content: profile.SystemRole + "\n\n项目源内容如下：\n" + sourceContent},
	{Role: "user", Content: instruction},
}
```

```go
func (s *DecompositionService) buildSeriesChapterMessages(
	ctx context.Context,
	userID uuid.UUID,
	chapter Chapter,
	outline []Chapter,
	chapterIndex int,
	chapterSourceContent string,
	sourceType string,
	gitURL string,
	scenarioMode prompt.ScenarioMode,
	style string,
	oldContent string,
	profile prompt.PromptProfile,
) ([]llm.Message, string, error) {
	requirements, err := s.promptReq.ResolveWithProfile(ctx, userID, scenarioMode, prompt.ArticleStyle(style), profile)
	if err != nil {
		return nil, "", err
	}
	return []llm.Message{
		{Role: "system", Content: profile.SystemRole + "\n\n项目源内容如下：\n" + chapterSourceContent},
		{Role: "user", Content: requirements},
	}, "deepseek-v4-flash", nil
}
```

- [ ] **Step 6: 保留清洗层，但补一条更广的 meta 段匹配**

```go
hasRoleIntro := strings.Contains(trimmed, "作为高级") ||
	strings.Contains(trimmed, "作为一名") ||
	strings.Contains(trimmed, "文本解读专家") ||
	strings.Contains(trimmed, "我将以") ||
	(strings.Contains(trimmed, "作为") &&
		(strings.Contains(trimmed, "架构师") ||
			strings.Contains(trimmed, "博主") ||
			strings.Contains(trimmed, "助手") ||
			strings.Contains(trimmed, "AI")))
```

- [ ] **Step 7: 运行聚焦测试，确认 Analyze 与 Generate 已使用 profile**

Run: `cd backend && go test ./internal/service -run 'OutlineBaseInstructionForProfile|BuildSeriesChapterMessages_UsesResolvedPromptProfileRole|PromptRequirementsService' -v`  
Expected: PASS，且新增断言均不再包含 `高级架构师` / `高级全栈架构师`。

- [ ] **Step 8: 提交**

```bash
git add backend/internal/service/decomposition_service.go \
  backend/internal/service/decomposition_generate_outline.go \
  backend/internal/service/decomposition_generate_outline_test.go \
  backend/internal/service/generator.go \
  backend/internal/service/decomposition_generate_prompt_helpers.go \
  backend/internal/service/decomposition_generate_intro.go \
  backend/internal/service/decomposition_generate_split_test.go \
  backend/internal/infra/llm/output_sanitize.go
git commit -m "feat(stream): lock analyze and generate to resolved prompt profile"
```

### Task 3: 扩展流式请求契约，并让前端锁定 Analyze 结果

**Files:**
- Modify: `backend/internal/domain/stream/dto.go`
- Modify: `backend/internal/domain/stream/service.go`
- Modify: `backend/internal/domain/stream/handler.go`
- Modify: `backend/internal/domain/stream/scenario_mode_test.go`
- Modify: `backend/internal/transport/http/v1/api/request_models_test.go`
- Modify: `frontend/src/store/streamStore.ts`
- Modify: `frontend/src/hooks/generator/fileParserUtils.ts`
- Modify: `frontend/src/hooks/generator/useFileParser.ts`
- Modify: `frontend/src/hooks/generator/useSeriesGenerator.ts`
- Create: `frontend/src/hooks/generator/useSeriesGenerator.test.ts`
- Modify: `frontend/src/pages/generatorViewState.ts`
- Modify: `frontend/src/components/generator/GeneratorOutlineStage.tsx`
- Test: `frontend/src/hooks/generator/useSeriesGenerator.test.ts`

- [ ] **Step 1: 先写失败测试，锁定请求体和前端锁定标签**

```ts
import { describe, expect, it } from 'vitest'
import { buildSeriesGenerateRequest, buildSingleGenerateRequest } from './useSeriesGenerator'

describe('useSeriesGenerator request builders', () => {
  it('includes locked prompt profile fields in series generation payload', () => {
    expect(
      buildSeriesGenerateRequest({
        sourceType: 'file',
        gitUrl: '',
        sourceContent: 'parsed content',
        seriesTitle: '《非暴力沟通》解读',
        outline: [],
        parentBlogId: 'parent-1',
        scenarioMode: 'ebook_interpretation',
        promptProfileKey: 'psychology_communication_book',
        documentKind: 'psychology_communication',
      }),
    ).toMatchObject({
      prompt_profile_key: 'psychology_communication_book',
      document_kind: 'psychology_communication',
    })
  })

  it('includes locked prompt profile fields in single generation payload', () => {
    expect(
      buildSingleGenerateRequest('parsed content', 'ebook_interpretation', 'classic_text_interpretation', 'classic_text'),
    ).toMatchObject({
      prompt_profile_key: 'classic_text_interpretation',
      document_kind: 'classic_text',
    })
  })
})
```

```ts
import { getGeneratorViewState } from '@/pages/generatorViewState'

it('exposes locked prompt profile label once outline exists', () => {
  const viewState = getGeneratorViewState({
    sourceType: 'file',
    sourceContent: 'parsed',
    modules: null,
    outline: [{ title: '第二章', summary: '逐章解读', sort: 1 }],
    scenarioMode: 'ebook_interpretation',
    resolvedPromptProfile: {
      key: 'psychology_communication_book',
      displayName: '心理学经典解读',
      documentKind: 'psychology_communication',
      reason: '命中沟通与情绪表达主题',
    },
    isScanning: false,
    isAnalyzing: false,
    isGenerating: false,
  })

  expect(viewState.lockedPromptProfileLabel).toBe('心理学经典解读')
})
```

- [ ] **Step 2: 运行前后端测试，确认当前实现失败**

Run: `cd frontend && npm run test -- useSeriesGenerator.test.ts generatorViewState.test.ts`  
Expected: FAIL，提示 `promptProfileKey` / `resolvedPromptProfile` 字段不存在。  

Run: `cd backend && go test ./internal/transport/http/v1/api -run GenerateRequest -v`  
Expected: FAIL，提示 `PromptProfileKey` / `DocumentKind` 字段不存在。

- [ ] **Step 3: 扩展请求 DTO 与前端 store**

```go
type GenerateRequest struct {
	SourceContent    string    `json:"source_content"`
	SourceType       string    `json:"source_type"`
	Outline          []Chapter `json:"outline"`
	GitURL           string    `json:"git_url"`
	SubDir           string    `json:"sub_dir"`
	SelectedModules  []string  `json:"selected_modules"`
	SeriesTitle      string    `json:"series_title"`
	ParentID         string    `json:"parent_id"`
	ArticleStyle     string    `json:"article_style"`
	ScenarioMode     string    `json:"scenario_mode"`
	PromptProfileKey string    `json:"prompt_profile_key"`
	DocumentKind     string    `json:"document_kind"`
}
```

```ts
export interface ResolvedPromptProfile {
  key: string
  displayName: string
  documentKind: string
  reason: string
}

interface StreamState {
  // ...
  resolvedPromptProfile: ResolvedPromptProfile | null
  classificationStatus: 'idle' | 'classifying' | 'resolved' | 'fallback'
  classificationReason: string
  setResolvedPromptProfile: (profile: ResolvedPromptProfile | null, status?: StreamState['classificationStatus']) => void
}
```

- [ ] **Step 4: Analyze 完成后把锁定 profile 写入 store**

```ts
if (data.status === 'analyzing' && data.message.includes('识别')) {
  store.setResolvedPromptProfile(null, 'classifying')
}

if (data.status === 'complete') {
  let outlineResult = data.content
  if (typeof data.content === 'string') {
    outlineResult = JSON.parse(data.content)
  }
  const resolvedProfile = outlineResult.resolved_prompt_profile
  if (resolvedProfile) {
    store.setResolvedPromptProfile({
      key: resolvedProfile.key,
      displayName: resolvedProfile.display_name,
      documentKind: resolvedProfile.document_kind,
      reason: resolvedProfile.reason,
    }, 'resolved')
  }
  store.setSeriesTitle(outlineResult.series_title || '')
  store.setOutline(outlineResult.outline || outlineResult.chapters)
}
```

- [ ] **Step 5: 在生成请求和大纲阶段展示中透传锁定 profile**

```ts
export function buildSeriesGenerateRequest(input: SeriesGenerateRequestInput) {
  return {
    source_type: input.sourceType,
    git_url: input.gitUrl,
    source_content: input.sourceContent,
    series_title: input.seriesTitle,
    outline: input.outline,
    parent_id: input.parentBlogId,
    scenario_mode: input.scenarioMode,
    prompt_profile_key: input.promptProfileKey,
    document_kind: input.documentKind,
  }
}
```

```tsx
{lockedPromptProfileLabel ? (
  <div className="inline-flex items-center rounded-full border border-zinc-200 bg-zinc-100 px-3 py-1 text-xs font-medium text-zinc-600 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300">
    当前提示词类型：{lockedPromptProfileLabel}
  </div>
) : null}
```

- [ ] **Step 6: 跑前端聚焦测试，确认新字段已锁定**

Run: `cd frontend && npm run test -- fileAnalyzeRequest.test.ts useSeriesGenerator.test.ts generatorViewState.test.ts`  
Expected: PASS，且请求 builder 与只读标签测试通过。

- [ ] **Step 7: 提交**

```bash
git add backend/internal/domain/stream/dto.go \
  backend/internal/domain/stream/service.go \
  backend/internal/domain/stream/handler.go \
  backend/internal/domain/stream/scenario_mode_test.go \
  backend/internal/transport/http/v1/api/request_models_test.go \
  frontend/src/store/streamStore.ts \
  frontend/src/hooks/generator/fileParserUtils.ts \
  frontend/src/hooks/generator/useFileParser.ts \
  frontend/src/hooks/generator/useSeriesGenerator.ts \
  frontend/src/hooks/generator/useSeriesGenerator.test.ts \
  frontend/src/pages/generatorViewState.ts \
  frontend/src/components/generator/GeneratorOutlineStage.tsx
git commit -m "feat(frontend): persist resolved prompt profile across analyze and generate"
```

### Task 4: 文档同步、端到端验证与 Docker 回归

**Files:**
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_PRD.md`
- Modify: `README.md`

- [ ] **Step 1: 先补文档差异点清单**

```md
- API：`/api/v1/stream/analyze` 响应新增 `resolved_prompt_profile`
- API：`/api/v1/stream/generate` 请求新增 `prompt_profile_key`、`document_kind`
- 架构：文件来源 Analyze 先做动态分类，再锁定 Prompt Profile
- PRD：配置解析阶段新增“内容类型识别 + 提示词类型展示”说明
- README：说明电子书/文件解读会自动选择匹配提示词
```

- [ ] **Step 2: 运行后端与前端测试**

Run: `cd backend && go test ./internal/service ./internal/domain/stream ./internal/transport/http/v1/api -v`  
Expected: PASS

Run: `cd frontend && npm run test -- fileAnalyzeRequest.test.ts useSeriesGenerator.test.ts generatorViewState.test.ts`  
Expected: PASS

- [ ] **Step 3: 用 Docker 验证文件全链路**

Run: `docker compose down && docker compose --env-file backend/.env up -d --build`  
Expected: 前端通过 `http://localhost` 可访问，文件上传后 Analyze 历史区出现“已识别内容类型/已匹配提示词”，生成正文开头不再出现“高级架构师”角色前言。

- [ ] **Step 4: 用真实案例手验《非暴力沟通》链路**

```md
1. 上传《非暴力沟通》PDF
2. 选择“电子书解读”
3. 点击“生成大纲”
4. 确认页面显示：
   - 当前创作场景：电子书解读
   - 当前提示词类型：心理学经典解读
5. 点击“开始生成”
6. 检查正文首段不含：
   - 好的，收到你的需求
   - 作为高级全栈架构师
   - 作为高级架构师
```

- [ ] **Step 5: 更新文档并提交**

```bash
git add .trae/documents/InkWords_API.md \
  .trae/documents/InkWords_Architecture.md \
  .trae/documents/InkWords_Conversation_Log.md \
  .trae/documents/InkWords_Development_Plan_and_Log.md \
  .trae/documents/InkWords_PRD.md \
  README.md
git commit -m "docs: document dynamic prompt profile workflow"
```

## Self-Review Checklist

- 需求覆盖：
  - 动态提示词机制：Task 1、Task 2
  - 文件来源全链路锁定：Task 2、Task 3
  - 去除“高级架构师”硬编码：Task 2
  - 前端只读展示与请求透传：Task 3
  - 文档同步与 Docker 验证：Task 4
- 占位符扫描：
  - 无 `TODO` / `TBD` / “稍后实现” / “类似前文”。
- 类型一致性：
  - 后端统一使用 `PromptProfileKey`、`ResolvedPromptProfile`
  - 前端统一使用 `resolvedPromptProfile`、`prompt_profile_key`、`document_kind`
