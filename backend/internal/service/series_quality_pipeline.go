package service

import (
	"context"
	"encoding/json"
	"fmt"
	blogcontracts "inkwords-backend/internal/domain/blog/contracts"
	"strings"

	llm "inkwords-backend/shared/platform/llm"
)

type seriesQualityPipelineInput struct {
	SeriesTitle          string
	ReaderProfile        string
	Outline              []blogcontracts.Chapter
	ChapterIndex         int
	Chapter              blogcontracts.Chapter
	ChapterSourceContent string
	GitURL               string
	OldContent           string
	UserID               string
	ProgressChan         chan<- string
}

// buildSeriesSharedPromptPrefix 固定系列级规则前缀，尽量让同一系列不同阶段复用相同提示词前缀。
func buildSeriesSharedPromptPrefix(seriesTitle string, readerProfile string, outline []blogcontracts.Chapter) string {
	var builder strings.Builder
	builder.WriteString("你正在为一个系列博客生成其中一篇高质量章节。\n")
	builder.WriteString(fmt.Sprintf("系列标题：%s\n", strings.TrimSpace(seriesTitle)))
	builder.WriteString(fmt.Sprintf("目标读者：%s\n", strings.TrimSpace(readerProfile)))
	builder.WriteString("系列总大纲：\n")
	for _, chapter := range outline {
		builder.WriteString(fmt.Sprintf("- %d. %s\n", chapter.Sort, chapter.Title))
	}
	builder.WriteString("统一术语：同一概念在全系列中保持同名；章节标题、读者画像、总大纲和本门禁在每个章节请求中保持字面一致。\n")
	builder.WriteString("统一质量门禁：必须解释机制、提供案例、给出复现方式、指出边界情况。\n")

	return builder.String()
}

func usageFromCompletionUsage(usage llm.CompletionUsage) SeriesChapterUsage {
	return SeriesChapterUsage{
		PromptTokens:          usage.PromptTokens,
		CompletionTokens:      usage.CompletionTokens,
		PromptCacheHitTokens:  usage.PromptCacheHitTokens,
		PromptCacheMissTokens: usage.PromptCacheMissTokens,
	}
}

func estimateSeriesUsageFromText(text string) SeriesChapterUsage {
	return SeriesChapterUsage{EstimatedTokens: len([]rune(text)) * 2}
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

func (s *DecompositionService) repairSeriesJSONOutput(
	ctx context.Context,
	llmModel string,
	userID string,
	seriesPrefix string,
	stageName string,
	raw string,
	validationErr error,
) (string, llm.CompletionUsage, error) {
	messages := []llm.Message{
		{Role: "system", Content: seriesPrefix + "\n当前阶段：" + stageName + " JSON 修复"},
		{
			Role: "user",
			Content: fmt.Sprintf(
				"下面是上一轮输出的 JSON 或近似 JSON，但它未通过结构化校验。\n校验错误：%v\n\n原始输出：\n%s\n\n请只修复缺失字段、布尔门禁或 JSON 格式，保持已有有效内容，不要扩写成新文章，返回严格 JSON。",
				validationErr,
				raw,
			),
		},
	}

	return s.llmClient.GenerateJSONWithOptions(ctx, llmModel, messages, llm.LightweightChatOptions(userID, 1800))
}

// generateSeriesChapterUnderstanding 调用模型先产出章节理解结果，避免后续草稿阶段直接凭空展开。
func (s *DecompositionService) generateSeriesChapterUnderstanding(
	ctx context.Context,
	llmModel string,
	seriesPrefix string,
	chapter blogcontracts.Chapter,
	chapterSourceContent string,
	userID string,
) (SeriesChapterUnderstanding, SeriesChapterUsage, error) {
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

	raw, usage, err := s.llmClient.GenerateJSONWithOptions(ctx, llmModel, messages, llm.LightweightChatOptions(userID, 1200))
	if err != nil {
		return SeriesChapterUnderstanding{}, SeriesChapterUsage{}, err
	}

	result, parseErr := parseSeriesChapterUnderstanding(raw)
	if parseErr == nil {
		return result, usageFromCompletionUsage(usage), nil
	}

	repairedRaw, repairUsage, err := s.repairSeriesJSONOutput(ctx, llmModel, userID, seriesPrefix, "章节理解", raw, parseErr)
	totalUsage := usageFromCompletionUsage(usage).add(usageFromCompletionUsage(repairUsage))
	if err != nil {
		return SeriesChapterUnderstanding{}, totalUsage, parseErr
	}
	result, err = parseSeriesChapterUnderstanding(repairedRaw)
	if err != nil {
		return SeriesChapterUnderstanding{}, totalUsage, err
	}
	return result, totalUsage, nil
}

// parseSeriesChapterDraft 负责把草稿阶段的 JSON 输出解析为受门禁保护的结构化结果。
func parseSeriesChapterDraft(raw string) (SeriesChapterDraft, error) {
	var result SeriesChapterDraft
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return SeriesChapterDraft{}, fmt.Errorf("unmarshal chapter draft: %w", err)
	}
	if err := validateSeriesChapterDraft(result); err != nil {
		return SeriesChapterDraft{}, err
	}

	return result, nil
}

// parseSeriesChapterReview 负责把审稿阶段的 JSON 输出解析为受门禁保护的结构化结果。
func parseSeriesChapterReview(raw string) (SeriesChapterReview, error) {
	var result SeriesChapterReview
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return SeriesChapterReview{}, fmt.Errorf("unmarshal chapter review: %w", err)
	}
	if err := validateSeriesChapterReview(result); err != nil {
		return SeriesChapterReview{}, err
	}

	return result, nil
}

func buildSeriesDraftPrompt(
	input seriesQualityPipelineInput,
	understanding SeriesChapterUnderstanding,
) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("当前章节标题：%s\n", input.Chapter.Title))
	builder.WriteString(fmt.Sprintf("章节摘要：%s\n", input.Chapter.Summary))
	builder.WriteString("请基于以下章节理解结果，先产出“结构化草稿 JSON”，字段必须包含 draft_markdown、coverage_check、example_inventory。\n")
	builder.WriteString("要求：必须解释机制、给出至少一个可复现案例、补充边界情况，用中文写作，Markdown 要适合直接作为技术博客章节。\n")
	builder.WriteString(fmt.Sprintf("chapter_goal：%s\n", understanding.ChapterGoal))
	builder.WriteString(fmt.Sprintf("must_explain：%s\n", strings.Join(understanding.MustExplain, "；")))
	builder.WriteString(fmt.Sprintf("must_include_examples：%s\n", strings.Join(understanding.MustIncludeExamples, "；")))
	if len(understanding.ReaderQuestions) > 0 {
		builder.WriteString(fmt.Sprintf("reader_questions：%s\n", strings.Join(understanding.ReaderQuestions, "；")))
	}
	if strings.TrimSpace(input.OldContent) != "" {
		builder.WriteString("\n旧版本内容（仅作松散参考，最终必须以当前材料为准）：\n")
		builder.WriteString(input.OldContent)
		builder.WriteString("\n")
	}
	builder.WriteString("\n当前章节材料：\n")
	builder.WriteString(input.ChapterSourceContent)
	return builder.String()
}

func buildSeriesReviewPrompt(
	chapter blogcontracts.Chapter,
	understanding SeriesChapterUnderstanding,
	draft SeriesChapterDraft,
) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("请审稿当前章节《%s》，返回严格 JSON，字段必须包含 depth_issues、example_issues、structure_issues、revision_actions、scorecard。\n", chapter.Title))
	builder.WriteString("审稿重点：深度是否足够、案例是否支撑观点、步骤是否可复现、结构是否清晰。\n")
	builder.WriteString(fmt.Sprintf("chapter_goal：%s\n", understanding.ChapterGoal))
	builder.WriteString(fmt.Sprintf("must_explain：%s\n", strings.Join(understanding.MustExplain, "；")))
	builder.WriteString(fmt.Sprintf("must_include_examples：%s\n", strings.Join(understanding.MustIncludeExamples, "；")))
	builder.WriteString("\n当前草稿：\n")
	builder.WriteString(draft.DraftMarkdown)
	return builder.String()
}

func buildSeriesFinalizePrompt(
	input seriesQualityPipelineInput,
	understanding SeriesChapterUnderstanding,
	draft SeriesChapterDraft,
	review SeriesChapterReview,
) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("请将章节《%s》补强为最终可发布版本，直接输出 Markdown，不要输出 JSON。\n", input.Chapter.Title))
	builder.WriteString("目标：只根据审稿意见做定向补强和轻统稿，保留草稿中有效内容，避免重写成另一篇文章。\n")
	builder.WriteString(fmt.Sprintf("chapter_goal：%s\n", understanding.ChapterGoal))
	if len(review.RevisionActions) > 0 {
		builder.WriteString(fmt.Sprintf("revision_actions：%s\n", strings.Join(review.RevisionActions, "；")))
	}
	extraRequirements := buildSeriesChapterExtraRequirements(input.GitURL, input.Outline, input.ChapterIndex)
	if extraRequirements != "" {
		builder.WriteString("附加要求：\n")
		builder.WriteString(extraRequirements)
	}
	builder.WriteString("\n当前草稿：\n")
	builder.WriteString(draft.DraftMarkdown)
	return builder.String()
}

func buildSeriesDraftRepairPrompt(
	input seriesQualityPipelineInput,
	understanding SeriesChapterUnderstanding,
	draft SeriesChapterDraft,
	review SeriesChapterReview,
) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("请只修补章节《%s》的草稿缺口，返回严格 JSON，字段仍为 draft_markdown、coverage_check、example_inventory。\n", input.Chapter.Title))
	builder.WriteString("要求：保留草稿已有有效段落，不要重写整篇，只补齐审稿指出的缺口和低分维度。\n")
	builder.WriteString(fmt.Sprintf("chapter_goal：%s\n", understanding.ChapterGoal))
	builder.WriteString(fmt.Sprintf("revision_actions：%s\n", strings.Join(review.RevisionActions, "；")))
	builder.WriteString(fmt.Sprintf("scorecard：depth=%d examples=%d reproducibility=%d clarity=%d\n", review.Scorecard.Depth, review.Scorecard.Examples, review.Scorecard.Reproducibility, review.Scorecard.Clarity))
	builder.WriteString("\n当前草稿：\n")
	builder.WriteString(draft.DraftMarkdown)
	return builder.String()
}

func (s *DecompositionService) repairSeriesChapterDraftForReview(
	ctx context.Context,
	llmModel string,
	seriesPrefix string,
	userID string,
	input seriesQualityPipelineInput,
	understanding SeriesChapterUnderstanding,
	draft SeriesChapterDraft,
	review SeriesChapterReview,
) (SeriesChapterDraft, SeriesChapterUsage, error) {
	options := llm.DefaultChatOptions()
	options.UserID = userID
	options.MaxTokens = 5000
	raw, usage, err := s.llmClient.GenerateJSONWithOptions(ctx, llmModel, []llm.Message{
		{Role: "system", Content: seriesPrefix + "\n当前阶段：章节草稿定向修复"},
		{Role: "user", Content: buildSeriesDraftRepairPrompt(input, understanding, draft, review)},
	}, options)
	if err != nil {
		return SeriesChapterDraft{}, SeriesChapterUsage{}, err
	}

	repaired, parseErr := parseSeriesChapterDraft(raw)
	if parseErr == nil {
		return repaired, usageFromCompletionUsage(usage), nil
	}

	repairedRaw, repairUsage, err := s.repairSeriesJSONOutput(ctx, llmModel, userID, seriesPrefix, "章节草稿修复", raw, parseErr)
	totalUsage := usageFromCompletionUsage(usage).add(usageFromCompletionUsage(repairUsage))
	if err != nil {
		return SeriesChapterDraft{}, totalUsage, parseErr
	}
	repaired, err = parseSeriesChapterDraft(repairedRaw)
	if err != nil {
		return SeriesChapterDraft{}, totalUsage, err
	}
	return repaired, totalUsage, nil
}

// generateSeriesChapterDraft 调用模型产出带门禁信息的章节草稿。
func (s *DecompositionService) generateSeriesChapterDraft(
	ctx context.Context,
	llmModel string,
	seriesPrefix string,
	input seriesQualityPipelineInput,
	understanding SeriesChapterUnderstanding,
	userID string,
) (SeriesChapterDraft, SeriesChapterUsage, error) {
	messages := []llm.Message{
		{Role: "system", Content: seriesPrefix + "\n当前阶段：章节写作"},
		{Role: "user", Content: buildSeriesDraftPrompt(input, understanding)},
	}

	options := llm.DefaultChatOptions()
	options.UserID = userID
	options.MaxTokens = 5000
	raw, usage, err := s.llmClient.GenerateJSONWithOptions(ctx, llmModel, messages, options)
	if err != nil {
		return SeriesChapterDraft{}, SeriesChapterUsage{}, err
	}

	result, parseErr := parseSeriesChapterDraft(raw)
	if parseErr == nil {
		return result, usageFromCompletionUsage(usage), nil
	}

	repairedRaw, repairUsage, err := s.repairSeriesJSONOutput(ctx, llmModel, userID, seriesPrefix, "章节草稿", raw, parseErr)
	totalUsage := usageFromCompletionUsage(usage).add(usageFromCompletionUsage(repairUsage))
	if err != nil {
		return SeriesChapterDraft{}, totalUsage, parseErr
	}
	result, err = parseSeriesChapterDraft(repairedRaw)
	if err != nil {
		return SeriesChapterDraft{}, totalUsage, err
	}
	return result, totalUsage, nil
}

// reviewSeriesChapterDraft 调用模型对草稿执行结构化技术审稿。
func (s *DecompositionService) reviewSeriesChapterDraft(
	ctx context.Context,
	llmModel string,
	seriesPrefix string,
	chapter blogcontracts.Chapter,
	understanding SeriesChapterUnderstanding,
	draft SeriesChapterDraft,
	userID string,
) (SeriesChapterReview, SeriesChapterUsage, error) {
	messages := []llm.Message{
		{Role: "system", Content: seriesPrefix + "\n当前阶段：章节审稿"},
		{Role: "user", Content: buildSeriesReviewPrompt(chapter, understanding, draft)},
	}
	options := llm.DefaultChatOptions()
	options.UserID = userID
	options.MaxTokens = 1800
	raw, usage, err := s.llmClient.GenerateJSONWithOptions(ctx, llmModel, messages, options)
	if err != nil {
		return SeriesChapterReview{}, SeriesChapterUsage{}, err
	}

	result, parseErr := parseSeriesChapterReview(raw)
	if parseErr == nil {
		return result, usageFromCompletionUsage(usage), nil
	}

	repairedRaw, repairUsage, err := s.repairSeriesJSONOutput(ctx, llmModel, userID, seriesPrefix, "章节审稿", raw, parseErr)
	totalUsage := usageFromCompletionUsage(usage).add(usageFromCompletionUsage(repairUsage))
	if err != nil {
		return SeriesChapterReview{}, totalUsage, parseErr
	}
	result, err = parseSeriesChapterReview(repairedRaw)
	if err != nil {
		return SeriesChapterReview{}, totalUsage, err
	}
	return result, totalUsage, nil
}

// runSeriesChapterQualityPipeline 负责在真正向前端流式输出前，先走完理解、草稿和审稿三段质量门禁。
func (s *DecompositionService) runSeriesChapterQualityPipeline(
	ctx context.Context,
	input seriesQualityPipelineInput,
) (SeriesChapterFinal, error) {
	seriesPrefix := buildSeriesSharedPromptPrefix(input.SeriesTitle, input.ReaderProfile, input.Outline)
	understandingModel := "deepseek-v4-flash"
	draftModel := "deepseek-v4-flash"
	reviewModel := "deepseek-v4-pro"
	finalModel := "deepseek-v4-pro"
	var totalUsage SeriesChapterUsage

	sendSeriesProgressPayload(input.ProgressChan, map[string]interface{}{
		"status":       "understanding",
		"chapter_sort": input.Chapter.Sort,
		"title":        input.Chapter.Title,
	})
	understanding, understandingUsage, err := s.generateSeriesChapterUnderstanding(
		ctx,
		understandingModel,
		seriesPrefix,
		input.Chapter,
		input.ChapterSourceContent,
		input.UserID,
	)
	if err != nil {
		return SeriesChapterFinal{}, err
	}
	totalUsage = totalUsage.add(understandingUsage)

	sendSeriesProgressPayload(input.ProgressChan, map[string]interface{}{
		"status":       "drafting",
		"chapter_sort": input.Chapter.Sort,
		"title":        input.Chapter.Title,
	})
	draft, draftUsage, err := s.generateSeriesChapterDraft(ctx, draftModel, seriesPrefix, input, understanding, input.UserID)
	if err != nil {
		return SeriesChapterFinal{}, err
	}
	totalUsage = totalUsage.add(draftUsage)

	sendSeriesProgressPayload(input.ProgressChan, map[string]interface{}{
		"status":       "reviewing",
		"chapter_sort": input.Chapter.Sort,
		"title":        input.Chapter.Title,
	})
	review, reviewUsage, err := s.reviewSeriesChapterDraft(ctx, reviewModel, seriesPrefix, input.Chapter, understanding, draft, input.UserID)
	if err != nil {
		return SeriesChapterFinal{}, err
	}
	totalUsage = totalUsage.add(reviewUsage)

	if scorecardBelowThreshold(review.Scorecard, 4) {
		sendSeriesProgressPayload(input.ProgressChan, map[string]interface{}{
			"status":       "repairing",
			"chapter_sort": input.Chapter.Sort,
			"title":        input.Chapter.Title,
		})
		repairedDraft, repairUsage, err := s.repairSeriesChapterDraftForReview(ctx, draftModel, seriesPrefix, input.UserID, input, understanding, draft, review)
		if err != nil {
			return SeriesChapterFinal{}, err
		}
		draft = repairedDraft
		totalUsage = totalUsage.add(repairUsage)
	}

	sendSeriesProgressPayload(input.ProgressChan, map[string]interface{}{
		"status":       "revising",
		"chapter_sort": input.Chapter.Sort,
		"title":        input.Chapter.Title,
	})
	final, err := s.finalizeSeriesChapterDraft(ctx, finalModel, input, seriesPrefix, understanding, draft, review)
	if err != nil {
		return SeriesChapterFinal{}, err
	}
	totalUsage = totalUsage.add(final.Usage)
	final.Usage = totalUsage
	final.QualityScorecard = review.Scorecard
	final.RevisionActions = append([]string(nil), review.RevisionActions...)
	return final, nil
}
