package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"inkwords-backend/internal/infra/llm"
)

// buildSeriesSharedPromptPrefix 固定系列级规则前缀，尽量让同一系列不同阶段复用相同提示词前缀。
func buildSeriesSharedPromptPrefix(seriesTitle string, readerProfile string, outline []Chapter) string {
	var builder strings.Builder
	builder.WriteString("你正在为一个系列博客生成其中一篇高质量章节。\n")
	builder.WriteString(fmt.Sprintf("系列标题：%s\n", strings.TrimSpace(seriesTitle)))
	builder.WriteString(fmt.Sprintf("目标读者：%s\n", strings.TrimSpace(readerProfile)))
	builder.WriteString("系列总大纲：\n")
	for _, chapter := range outline {
		builder.WriteString(fmt.Sprintf("- %d. %s\n", chapter.Sort, chapter.Title))
	}
	builder.WriteString("统一质量门禁：必须解释机制、提供案例、给出复现方式、指出边界情况。\n")

	return builder.String()
}

// parseSeriesChapterUnderstanding 负责把理解阶段的 JSON 输出解析为受门禁保护的结构化结果。
func parseSeriesChapterUnderstanding(raw string) (SeriesChapterUnderstanding, error) {
	var result SeriesChapterUnderstanding
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return SeriesChapterUnderstanding{}, fmt.Errorf("unmarshal chapter understanding: %w", err)
	}
	if err := validateSeriesChapterUnderstanding(result); err != nil {
		return SeriesChapterUnderstanding{}, err
	}

	return result, nil
}

// generateSeriesChapterUnderstanding 调用模型先产出章节理解结果，避免后续草稿阶段直接凭空展开。
func (s *DecompositionService) generateSeriesChapterUnderstanding(
	ctx context.Context,
	llmModel string,
	seriesPrefix string,
	chapter Chapter,
	chapterSourceContent string,
) (SeriesChapterUnderstanding, error) {
	messages := []llm.Message{
		{Role: "system", Content: seriesPrefix + "\n当前阶段：章节理解"},
		{
			Role: "user",
			Content: fmt.Sprintf(
				"当前章节标题：%s\n章节摘要：%s\n材料：\n%s\n\n请返回严格 JSON。",
				chapter.Title,
				chapter.Summary,
				chapterSourceContent,
			),
		},
	}

	raw, err := s.llmClient.GenerateJSON(ctx, llmModel, messages)
	if err != nil {
		return SeriesChapterUnderstanding{}, err
	}

	return parseSeriesChapterUnderstanding(raw)
}
