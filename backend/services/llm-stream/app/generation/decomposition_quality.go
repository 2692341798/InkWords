package generation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	llm "inkwords-backend/shared/platform/llm"
	sharedblog "inkwords-backend/shared/kernel/blog"
)
//nolint:all
func buildSeriesSharedPromptPrefix(seriesTitle string, readerProfile string, outline []sharedblog.Chapter) string {
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

func (s *DecompositionService) generateSeriesChapterUnderstanding(
	ctx context.Context,
	llmModel string,
	seriesPrefix string,
	chapter sharedblog.Chapter,
	chapterSourceContent string,
	userID string,
) (seriesChapterUnderstanding, seriesChapterUsage, error) {
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
		return seriesChapterUnderstanding{}, seriesChapterUsage{}, err
	}

	result, parseErr := parseSeriesUnderstanding(raw)
	if parseErr == nil {
		return result, usageFromCompletionUsage(usage), nil
	}

	repairedRaw, repairUsage, err := s.repairSeriesJSONOutput(ctx, llmModel, userID, seriesPrefix, "章节理解", raw, parseErr)
	totalUsage := usageFromCompletionUsage(usage).add(usageFromCompletionUsage(repairUsage))
	if err != nil {
		return seriesChapterUnderstanding{}, totalUsage, parseErr
	}
	result, err = parseSeriesUnderstanding(repairedRaw)
	if err != nil {
		return seriesChapterUnderstanding{}, totalUsage, err
	}
	return result, totalUsage, nil
}

func parseSeriesUnderstanding(raw string) (seriesChapterUnderstanding, error) {
	var result seriesChapterUnderstanding
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return seriesChapterUnderstanding{}, fmt.Errorf("unmarshal chapter understanding: %w", err)
	}
	if err := validateSeriesChapterUnderstanding(result); err != nil {
		return seriesChapterUnderstanding{}, err
	}
	return result, nil
}

func (s *DecompositionService) generateSeriesChapterDraft(
	ctx context.Context,
	llmModel string,
	seriesPrefix string,
	input seriesQualityPipelineInput,
	understanding seriesChapterUnderstanding,
	userID string,
) (seriesChapterDraft, seriesChapterUsage, error) {
	messages := []llm.Message{
		{Role: "system", Content: seriesPrefix + "\n当前阶段：章节写作"},
		{Role: "user", Content: buildSeriesDraftPrompt(input, understanding)},
	}

	options := llm.DefaultChatOptions()
	options.UserID = userID
	options.MaxTokens = 5000
	raw, usage, err := s.llmClient.GenerateJSONWithOptions(ctx, llmModel, messages, options)
	if err != nil {
		return seriesChapterDraft{}, seriesChapterUsage{}, err
	}

	result, parseErr := parseSeriesDraft(raw)
	if parseErr == nil {
		return result, usageFromCompletionUsage(usage), nil
	}

	repairedRaw, repairUsage, err := s.repairSeriesJSONOutput(ctx, llmModel, userID, seriesPrefix, "章节草稿", raw, parseErr)
	totalUsage := usageFromCompletionUsage(usage).add(usageFromCompletionUsage(repairUsage))
	if err != nil {
		return seriesChapterDraft{}, totalUsage, parseErr
	}
	result, err = parseSeriesDraft(repairedRaw)
	if err != nil {
		return seriesChapterDraft{}, totalUsage, err
	}
	return result, totalUsage, nil
}

func parseSeriesDraft(raw string) (seriesChapterDraft, error) {
	var result seriesChapterDraft
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return seriesChapterDraft{}, fmt.Errorf("unmarshal chapter draft: %w", err)
	}
	if err := validateSeriesChapterDraft(result); err != nil {
		return seriesChapterDraft{}, err
	}
	return result, nil
}

//nolint:all
func buildSeriesDraftPrompt(
	input seriesQualityPipelineInput,
	understanding seriesChapterUnderstanding,
) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("当前章节标题：%s\n", input.Chapter.Title))
	builder.WriteString(fmt.Sprintf("章节摘要：%s\n", input.Chapter.Summary))
	builder.WriteString("请基于以下章节理解结果，先产出「结构化草稿 JSON」，字段必须包含 draft_markdown、coverage_check、example_inventory。\n")
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

func (s *DecompositionService) reviewSeriesChapterDraft(
	ctx context.Context,
	llmModel string,
	seriesPrefix string,
	chapter sharedblog.Chapter,
	understanding seriesChapterUnderstanding,
	draft seriesChapterDraft,
	userID string,
) (seriesChapterReview, seriesChapterUsage, error) {
	messages := []llm.Message{
		{Role: "system", Content: seriesPrefix + "\n当前阶段：章节审稿"},
		{Role: "user", Content: buildSeriesReviewPrompt(chapter, understanding, draft)},
	}
	options := llm.DefaultChatOptions()
	options.UserID = userID
	options.MaxTokens = 1800
	raw, usage, err := s.llmClient.GenerateJSONWithOptions(ctx, llmModel, messages, options)
	if err != nil {
		return seriesChapterReview{}, seriesChapterUsage{}, err
	}

	result, parseErr := parseSeriesReview(raw)
	if parseErr == nil {
		return result, usageFromCompletionUsage(usage), nil
	}

	repairedRaw, repairUsage, err := s.repairSeriesJSONOutput(ctx, llmModel, userID, seriesPrefix, "章节审稿", raw, parseErr)
	totalUsage := usageFromCompletionUsage(usage).add(usageFromCompletionUsage(repairUsage))
	if err != nil {
		return seriesChapterReview{}, totalUsage, parseErr
	}
	result, err = parseSeriesReview(repairedRaw)
	if err != nil {
		return seriesChapterReview{}, totalUsage, err
	}
	return result, totalUsage, nil
}

func parseSeriesReview(raw string) (seriesChapterReview, error) {
	var result seriesChapterReview
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return seriesChapterReview{}, fmt.Errorf("unmarshal chapter review: %w", err)
	}
	if err := validateSeriesChapterReview(result); err != nil {
		return seriesChapterReview{}, err
	}
	return result, nil
}

//nolint:all
func buildSeriesReviewPrompt(
	chapter sharedblog.Chapter,
	understanding seriesChapterUnderstanding,
	draft seriesChapterDraft,
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

func (s *DecompositionService) repairSeriesChapterDraftForReview(
	ctx context.Context,
	llmModel string,
	seriesPrefix string,
	userID string,
	input seriesQualityPipelineInput,
	understanding seriesChapterUnderstanding,
	draft seriesChapterDraft,
	review seriesChapterReview,
) (seriesChapterDraft, seriesChapterUsage, error) {
	options := llm.DefaultChatOptions()
	options.UserID = userID
	options.MaxTokens = 5000
	raw, usage, err := s.llmClient.GenerateJSONWithOptions(ctx, llmModel, []llm.Message{
		{Role: "system", Content: seriesPrefix + "\n当前阶段：章节草稿定向修复"},
		{Role: "user", Content: buildSeriesDraftRepairPrompt(input, understanding, draft, review)},
	}, options)
	if err != nil {
		return seriesChapterDraft{}, seriesChapterUsage{}, err
	}

	repaired, parseErr := parseSeriesDraft(raw)
	if parseErr == nil {
		return repaired, usageFromCompletionUsage(usage), nil
	}

	repairedRaw, repairUsage, err := s.repairSeriesJSONOutput(ctx, llmModel, userID, seriesPrefix, "章节草稿修复", raw, parseErr)
	totalUsage := usageFromCompletionUsage(usage).add(usageFromCompletionUsage(repairUsage))
	if err != nil {
		return seriesChapterDraft{}, totalUsage, parseErr
	}
	repaired, err = parseSeriesDraft(repairedRaw)
	if err != nil {
		return seriesChapterDraft{}, totalUsage, err
	}
	return repaired, totalUsage, nil
}

//nolint:all
func buildSeriesDraftRepairPrompt(
	input seriesQualityPipelineInput,
	understanding seriesChapterUnderstanding,
	draft seriesChapterDraft,
	review seriesChapterReview,
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

//nolint:staticcheck
func buildSeriesFinalizePrompt(
	input seriesQualityPipelineInput,
	understanding seriesChapterUnderstanding,
	draft seriesChapterDraft,
	review seriesChapterReview,
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

func (s *DecompositionService) finalizeSeriesChapterDraft(
	ctx context.Context,
	llmModel string,
	input seriesQualityPipelineInput,
	seriesPrefix string,
	understanding seriesChapterUnderstanding,
	draft seriesChapterDraft,
	review seriesChapterReview,
) (seriesChapterFinal, error) {
	chunkChan := make(chan string, 100)
	errChan := make(chan error, 1)
	usageChan := make(chan llm.CompletionUsage, 1)
	var finalBuilder strings.Builder

	go func() {
		options := llm.DefaultChatOptions()
		options.UserID = input.UserID
		_, usage, err := s.llmClient.GenerateStreamWithOptions(ctx, llmModel, []llm.Message{
			{Role: "system", Content: seriesPrefix + "\n当前阶段：定向补强与轻统稿"},
			{Role: "user", Content: buildSeriesFinalizePrompt(input, understanding, draft, review)},
		}, chunkChan, options)
		usageChan <- usage
		errChan <- err
	}()

	for chunk := range chunkChan {
		finalBuilder.WriteString(chunk)
		bytes, _ := json.Marshal(map[string]interface{}{
			"status":       "streaming",
			"chapter_sort": input.Chapter.Sort,
			"title":        input.Chapter.Title,
			"content":      chunk,
		})
		input.ProgressChan <- string(bytes)
	}

	if err := <-errChan; err != nil {
		return seriesChapterFinal{}, err
	}
	usage := <-usageChan
	bytes, _ := json.Marshal(map[string]interface{}{
		"status":                   "usage",
		"chapter_sort":             input.Chapter.Sort,
		"title":                    input.Chapter.Title,
		"prompt_tokens":            usage.PromptTokens,
		"completion_tokens":        usage.CompletionTokens,
		"prompt_cache_hit_tokens":  usage.PromptCacheHitTokens,
		"prompt_cache_miss_tokens": usage.PromptCacheMissTokens,
	})
	input.ProgressChan <- string(bytes)

	return seriesChapterFinal{
		FinalMarkdown:    finalBuilder.String(),
		ResolvedIssues:   append([]string(nil), review.RevisionActions...),
		ResidualRisks:    nil,
		Usage:            usageFromCompletionUsage(usage),
		QualityScorecard: review.Scorecard,
		RevisionActions:  append([]string(nil), review.RevisionActions...),
	}, nil
}

func (s *DecompositionService) runSeriesChapterQualityPipeline(
	ctx context.Context,
	input seriesQualityPipelineInput,
) (seriesChapterFinal, error) {
	seriesPrefix := buildSeriesSharedPromptPrefix(input.SeriesTitle, input.ReaderProfile, input.Outline)
	understandingModel := "deepseek-v4-flash"
	draftModel := "deepseek-v4-flash"
	reviewModel := "deepseek-v4-pro"
	finalModel := "deepseek-v4-pro"
	var totalUsage seriesChapterUsage

	sendQualityProgress(input.ProgressChan, input.Chapter.Sort, input.Chapter.Title, "understanding")
	understanding, understandingUsage, err := s.generateSeriesChapterUnderstanding(ctx, understandingModel, seriesPrefix, input.Chapter, input.ChapterSourceContent, input.UserID)
	if err != nil {
		return seriesChapterFinal{}, err
	}
	totalUsage = totalUsage.add(understandingUsage)

	sendQualityProgress(input.ProgressChan, input.Chapter.Sort, input.Chapter.Title, "drafting")
	draft, draftUsage, err := s.generateSeriesChapterDraft(ctx, draftModel, seriesPrefix, input, understanding, input.UserID)
	if err != nil {
		return seriesChapterFinal{}, err
	}
	totalUsage = totalUsage.add(draftUsage)

	sendQualityProgress(input.ProgressChan, input.Chapter.Sort, input.Chapter.Title, "reviewing")
	review, reviewUsage, err := s.reviewSeriesChapterDraft(ctx, reviewModel, seriesPrefix, input.Chapter, understanding, draft, input.UserID)
	if err != nil {
		return seriesChapterFinal{}, err
	}
	totalUsage = totalUsage.add(reviewUsage)

	if scorecardBelowThreshold(review.Scorecard, 4) {
		sendQualityProgress(input.ProgressChan, input.Chapter.Sort, input.Chapter.Title, "repairing")
		repairedDraft, repairUsage, err := s.repairSeriesChapterDraftForReview(ctx, draftModel, seriesPrefix, input.UserID, input, understanding, draft, review)
		if err != nil {
			return seriesChapterFinal{}, err
		}
		draft = repairedDraft
		totalUsage = totalUsage.add(repairUsage)
	}

	sendQualityProgress(input.ProgressChan, input.Chapter.Sort, input.Chapter.Title, "revising")
	final, err := s.finalizeSeriesChapterDraft(ctx, finalModel, input, seriesPrefix, understanding, draft, review)
	if err != nil {
		return seriesChapterFinal{}, err
	}
	totalUsage = totalUsage.add(final.Usage)
	final.Usage = totalUsage
	final.QualityScorecard = review.Scorecard
	final.RevisionActions = append([]string(nil), review.RevisionActions...)
	return final, nil
}

func sendQualityProgress(progressChan chan<- string, chapterSort int, title string, status string) {
	bytes, _ := json.Marshal(map[string]interface{}{
		"status":       status,
		"chapter_sort": chapterSort,
		"title":        title,
	})
	progressChan <- string(bytes)
}
