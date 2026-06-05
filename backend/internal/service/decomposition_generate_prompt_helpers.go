package service

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"

	"inkwords-backend/internal/infra/llm"
	"inkwords-backend/internal/prompt"
)

func buildSeriesChapterExtraRequirements(gitURL string, outline []Chapter, chapterIndex int) string {
	extraRequirements := ""
	reqIndex := 7
	if gitURL != "" {
		extraRequirements += fmt.Sprintf("%d. **源码仓库引用**：请在文章开头或合适的位置，引用本项目的 Git 仓库地址：%s\n", reqIndex, gitURL)
		reqIndex++
	}
	if chapterIndex+1 < len(outline) {
		extraRequirements += fmt.Sprintf("%d. **下期预告**：请在文章结尾处，明确预告下一篇文章的内容：\"下期预告：%s\"\n", reqIndex, outline[chapterIndex+1].Title)
	}
	return extraRequirements
}

func buildSeriesReaderProfile(scenarioMode prompt.ScenarioMode) string {
	switch scenarioMode {
	case prompt.ScenarioModeBeginnerWalkthrough:
		return "零基础读者，需要白话解释、步骤拆解和常见坑提醒"
	case prompt.ScenarioModeOpenBookExamReview:
		return "需要快速定位知识点、例题和结论的复习型读者"
	default:
		return "希望理解原理、案例和实现细节的技术博客读者"
	}
}

func (s *DecompositionService) resolveSeriesOldContent(ctx context.Context, chapter Chapter) string {
	if chapter.Action != "regenerate" || strings.TrimSpace(chapter.ID) == "" {
		return ""
	}

	blogID, err := uuid.Parse(chapter.ID)
	if err != nil {
		return ""
	}

	oldContent, err := s.seriesPersistence.LoadSeriesOldContent(ctx, blogID)
	if err != nil {
		return ""
	}

	return truncateSeriesContent(oldContent, seriesOldContentRuneLimit)
}

func normalizePromptProfile(profile prompt.PromptProfile, scenarioMode prompt.ScenarioMode) prompt.PromptProfile {
	if !scenarioMode.IsValid() {
		scenarioMode = prompt.ScenarioModeEbookInterpretation
	}
	if profile.Key == "" {
		return prompt.FallbackPromptProfileForScenario(scenarioMode)
	}

	return prompt.ResolvePromptProfileKey(string(profile.Key), scenarioMode)
}

func normalizeResolvedPromptProfile(
	profile prompt.PromptProfile,
	resolved prompt.ResolvedPromptProfile,
	defaultReason string,
) prompt.ResolvedPromptProfile {
	if strings.TrimSpace(defaultReason) == "" {
		defaultReason = "已按场景默认提示词生成。"
	}
	if resolved.Key == "" {
		return newResolvedPromptProfile(profile, profile.DocumentKind, defaultReason)
	}
	if resolved.DisplayName == "" {
		resolved.DisplayName = profile.DisplayName
	}
	if resolved.DocumentKind == "" {
		resolved.DocumentKind = profile.DocumentKind
	}
	if resolved.Reason == "" {
		resolved.Reason = defaultReason
	}
	return resolved
}

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
	if !scenarioMode.IsValid() {
		scenarioMode = prompt.DefaultScenarioModeForSource(sourceType)
	}
	profile = normalizePromptProfile(profile, scenarioMode)

	requirements := strings.TrimSpace(strings.Join([]string{
		prompt.DefaultScenarioRequirements(scenarioMode),
		prompt.DefaultStyleRequirements(scenarioMode, prompt.ArticleStyleGeneral),
	}, "\n\n"))
	requirements = strings.TrimSpace(strings.Join([]string{
		profile.GenerateRequirements,
		requirements,
	}, "\n\n"))
	if s.promptReq != nil {
		if resolved, err := s.promptReq.ResolveWithProfile(ctx, userID, scenarioMode, prompt.ArticleStyle(style), profile); err == nil && resolved != "" {
			requirements = resolved
		}
	}

	promptText := fmt.Sprintf(`请根据上述源内容，以及本章节的大纲，生成一篇高质量技术博客章节。

写作要求：
%s

硬性约束：
1. 所有生成的 Mermaid 图表代码块绝对禁止包含自定义样式关键字（如 style, classDef, linkStyle 等），必须使用基础语法。在 Mermaid 图表中，如果节点文本包含特殊字符（如括号、幂符号等，例如 O(1), O(n^2)），必须使用双引号将节点文本包裹起来，例如 A["O(1)"] 而不是 A[O(1)]。
2. 请务必完整输出，不要遗漏关键知识点。如果内容较长，请合理分配篇幅，确保文章结构完整，包含结尾总结。
%s

本章节大纲：
- 标题: %s
- 摘要: %s
- 排序: %d
`, requirements, buildSeriesChapterExtraRequirements(gitURL, outline, chapterIndex), chapter.Title, chapter.Summary, chapter.Sort)

	if oldContent != "" {
		promptText += fmt.Sprintf("\n【注意：本章节为旧版博客的更新重写】\n以下是该章节在旧版本项目中的博客内容，供你作为松散参考。\n你可以参考旧内容中解释抽象概念的比喻、业务知识点或行文风格，但必须以本次提供的最新源码为准进行重写或调整，如果最新代码逻辑发生了改变，请以最新代码为准。\n旧版本内容：\n---\n%s\n---\n", oldContent)
	}

	messages := []llm.Message{
		{Role: "system", Content: profile.SystemRole + "\n\n项目源内容如下：\n" + chapterSourceContent},
		{Role: "user", Content: promptText},
	}

	llmModel := "deepseek-v4-flash"
	if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
		llmModel = envModel
	}

	return messages, llmModel, nil
}
