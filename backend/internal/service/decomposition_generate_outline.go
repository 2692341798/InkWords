package service

import (
	"context"
	"encoding/json"
	"fmt"
	llm "inkwords-backend/shared/platform/llm"
	"inkwords-backend/internal/model"
	"inkwords-backend/internal/prompt"
	"os"
	"strings"
)

func (s *DecompositionService) GenerateOutline(
	ctx context.Context,
	sourceContent string,
	scenarioMode prompt.ScenarioMode,
	existingParent *model.Blog,
	existingChildren []model.Blog,
) (*OutlineResult, error) {
	return s.GenerateOutlineWithProfile(
		ctx,
		sourceContent,
		scenarioMode,
		prompt.PromptProfile{},
		prompt.ResolvedPromptProfile{},
		existingParent,
		existingChildren,
	)
}

func (s *DecompositionService) GenerateOutlineWithProfile(
	ctx context.Context,
	sourceContent string,
	scenarioMode prompt.ScenarioMode,
	profile prompt.PromptProfile,
	resolved prompt.ResolvedPromptProfile,
	existingParent *model.Blog,
	existingChildren []model.Blog,
) (*OutlineResult, error) {
	runes := []rune(sourceContent)
	if len(runes) > 15000000 {
		sourceContent = string(runes[:15000000]) + "\n\n... [Content Truncated due to length limits] ..."
	}

	if !scenarioMode.IsValid() {
		scenarioMode = prompt.ScenarioModeEbookInterpretation
	}
	profile = normalizePromptProfile(profile, scenarioMode)
	resolved = normalizeResolvedPromptProfile(profile, resolved, "已按当前提示词类型生成大纲。")

	systemRole, instruction := outlinePromptForProfile(scenarioMode, profile)

	if existingParent != nil {
		var existingOutlineBuilder strings.Builder
		existingOutlineBuilder.WriteString(fmt.Sprintf("原系列标题: %s\n原章节列表:\n", existingParent.Title))
		for _, child := range existingChildren {
			existingOutlineBuilder.WriteString(fmt.Sprintf("- 章节ID: %s, 序号: %d, 标题: %s\n", child.ID.String(), child.ChapterSort, child.Title))
		}

		instruction += fmt.Sprintf(`

**注意：用户已经拥有该项目旧版本的博客系列大纲如下**：
%s

由于项目可能发生了更新，你需要对比最新源码与已有博客大纲，生成一个**更新后**的系列博客大纲。
- 对于内容没有发生变化的章节，你可以保留原样，并将 "action" 标记为 "skip"（跳过生成），并**务必填入对应的 "id"**。
- 对于需要根据最新代码更新的旧章节，将 "action" 标记为 "regenerate"（重新生成），并**务必填入对应的 "id"**。
- 对于根据新功能/新模块产生的新章节，将 "action" 标记为 "new"（新增），不要填 "id"。
- 对于已经废弃的功能对应的章节，直接在新的大纲中将其移除即可。
`, existingOutlineBuilder.String())
	}

	instruction += `
输出必须是纯JSON格式，包含 series_title 和 chapters 两个字段，不包含任何Markdown标记或其他文字。
JSON 格式如下：
{
  "series_title": "系列博客的标题（必须根据项目内容精准概括，例如：React 源码解析系列、Vite 配置实战等，绝不要使用通用或宽泛的占位名称）",
  "chapters": [
    {
      "id": "章节的 UUID（如果有）",
      "title": "章节标题",
      "summary": "该章节的详细摘要或内容要点（指导后续生成的具体方向）",
      "sort": 1,
      "files": ["强相关的具体文件路径或目录（必须是相对路径）"],
      "action": "new 或 regenerate 或 skip"
    }
  ]
}`

	// 不同场景使用不同的系统提示标签，避免电子书内容被当作技术项目
	systemLabel := "项目文本内容如下：\n"
	if scenarioMode == prompt.ScenarioModeEbookInterpretation {
		systemLabel = "以下是原文内容：\n"
	}

	messages := []llm.Message{
		{Role: "system", Content: systemRole + "\n\n" + systemLabel + sourceContent},
		{Role: "user", Content: instruction},
	}

	modelName := "deepseek-v4-flash"
	if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
		modelName = envModel
	}

	options := llm.DefaultChatOptions()
	options.MaxTokens = 6000
	content, _, err := s.llmClient.GenerateJSONWithOptions(ctx, modelName, messages, options)
	if err != nil {
		return nil, fmt.Errorf("llm generation failed: %w", err)
	}

	content = strings.TrimPrefix(strings.TrimSpace(content), "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var outline OutlineResult
	if err := json.Unmarshal([]byte(content), &outline); err != nil {
		return nil, fmt.Errorf("failed to unmarshal llm output: %w, output: %s", err, content)
	}
	outline.ResolvedPromptProfile = resolved

	return &outline, nil
}

func outlinePromptForProfile(mode prompt.ScenarioMode, profile prompt.PromptProfile) (string, string) {
	return profile.SystemRole, strings.TrimSpace(strings.Join([]string{
		profile.AnalyzeRequirements,
		outlineBaseInstruction(mode),
		"场景约束：\n" + outlineScenarioHint(mode),
	}, "\n\n"))
}

func outlineScenarioHint(mode prompt.ScenarioMode) string {
	switch mode {
	case prompt.ScenarioModeOpenBookExamReview:
		return "请按考点、题型、实验步骤或速查结构拆分章节，优先帮助开卷考试快速定位。"
	case prompt.ScenarioModeBeginnerWalkthrough:
		return "请按学习路径拆分章节，优先覆盖环境准备、目录结构、关键主链路和常见排错。"
	default:
		return "请按原文自身篇章结构与主题脉络拆分章节，只做文本解读，不要将内容映射到现代商业、技术或管理场景。"
	}
}

// outlineBaseInstruction 返回不同场景下的大纲生成基础指令。
func outlineBaseInstruction(mode prompt.ScenarioMode) string {
	if mode == prompt.ScenarioModeEbookInterpretation {
		return `前面提供的是一本书或长篇文献的内容，请按原文自然篇章结构生成一个系列解读大纲。
我的核心要求是：按原文的章节或主题单元逐章拆分，每章聚焦该篇的核心思想与原文精义，而非现代应用或技术映射。

请根据文本的章节数量充分规划：
- 每一个独立篇章或主题段落都应至少对应一篇解读
- 每篇解读聚焦该篇章的历史背景、核心观点和代表性原文摘录
- 不要将内容强行映射到现代商业、技术或管理场景`
	}

	return `请评估前面提供的项目文本，并生成一个系列博客的大纲。
对于大型项目、源码仓库或复杂内容，必须拆分为系列博客。
我的核心要求是：对于每个核心模块、业务逻辑或重要架构层，**都必须对应至少一篇博客进行详细说明**。不要担心生成的篇数过多！

请根据提供的项目文件数量和内容复杂度，充分且详细地规划章节数量：
- 对于普通项目，不要吝啬章节数，确保每一个独立的功能点、数据流环节、配置模块等都有专属的文章解析。
- 对于特别庞大的框架源码（如 FFmpeg 等），请大胆拆分出数十篇详细章节，做到对核心源码文件的全面覆盖。
- 每一篇博客只聚焦于**一个具体的核心技术点或模块**，并详细说明其原理与实现。`
}
