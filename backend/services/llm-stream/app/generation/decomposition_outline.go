package generation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	sharedblog "inkwords-backend/shared/kernel/blog"
	"inkwords-backend/shared/kernel/prompt"
	llm "inkwords-backend/shared/platform/llm"
)

const maxOutlineSourceRunes = 15_000_000

type generatedOutline struct {
	SeriesTitle string               `json:"series_title"`
	Chapters    []sharedblog.Chapter `json:"chapters"`
}

func (s *DecompositionService) generateOutline(
	ctx context.Context,
	sourceContent string,
	scenarioMode prompt.ScenarioMode,
	profile prompt.PromptProfile,
) (generatedOutline, error) {
	runes := []rune(sourceContent)
	if len(runes) > maxOutlineSourceRunes {
		sourceContent = string(runes[:maxOutlineSourceRunes]) + "\n\n... [Content Truncated due to length limits] ..."
	}

	profile = normalizePromptProfile(profile, scenarioMode)
	instruction := strings.TrimSpace(strings.Join([]string{
		profile.AnalyzeRequirements,
		outlineBaseInstruction(scenarioMode),
		"场景约束：\n" + outlineScenarioHint(scenarioMode),
		`输出必须是纯 JSON，不包含 Markdown 标记或其他文字，格式如下：
{
  "series_title": "根据内容精准概括的系列标题",
  "chapters": [
    {
      "title": "章节标题",
      "summary": "详细摘要与内容要点",
      "sort": 1,
      "files": [],
      "action": "new"
    }
  ]
}`,
	}, "\n\n"))

	systemLabel := "项目文本内容如下：\n"
	if scenarioMode == prompt.ScenarioModeEbookInterpretation {
		systemLabel = "以下是原文内容：\n"
	}

	messages := []llm.Message{
		{Role: "system", Content: profile.SystemRole + "\n\n" + systemLabel + sourceContent},
		{Role: "user", Content: instruction},
	}

	modelName := "deepseek-v4-flash"
	if configuredModel := strings.TrimSpace(os.Getenv("DEEPSEEK_MODEL")); configuredModel != "" {
		modelName = configuredModel
	}

	options := llm.DefaultChatOptions()
	options.MaxTokens = 6000
	content, _, err := s.llmClient.GenerateJSONWithOptions(ctx, modelName, messages, options)
	if err != nil {
		return generatedOutline{}, fmt.Errorf("llm generation failed: %w", err)
	}

	content = strings.TrimPrefix(strings.TrimSpace(content), "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var outline generatedOutline
	if err := json.Unmarshal([]byte(content), &outline); err != nil {
		return generatedOutline{}, fmt.Errorf("decode llm outline: %w", err)
	}
	if strings.TrimSpace(outline.SeriesTitle) == "" || len(outline.Chapters) == 0 {
		return generatedOutline{}, fmt.Errorf("llm outline is empty")
	}

	return outline, nil
}

func outlineBaseInstruction(mode prompt.ScenarioMode) string {
	if mode == prompt.ScenarioModeEbookInterpretation {
		return "请按原文自然篇章和主题单元生成系列解读大纲，每章聚焦核心思想与原文精义。"
	}
	return "请根据内容复杂度生成结构清晰的系列博客大纲，每一章只聚焦一个明确知识点、模块或步骤。"
}

func outlineScenarioHint(mode prompt.ScenarioMode) string {
	switch mode {
	case prompt.ScenarioModeOpenBookExamReview:
		return "按考点、题型、实验步骤或速查结构拆分，帮助开卷考试快速定位。"
	case prompt.ScenarioModeBeginnerWalkthrough:
		return "按学习路径拆分，覆盖环境准备、关键主链路、实践步骤和常见排错。"
	default:
		return "按原文篇章结构和主题脉络拆分，只做文本解读。"
	}
}
