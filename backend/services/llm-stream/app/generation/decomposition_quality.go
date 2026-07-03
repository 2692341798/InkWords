package generation

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	sharedblog "inkwords-backend/shared/kernel/blog"
	llm "inkwords-backend/shared/platform/llm"
)

const seriesUnderstandingJSONContract = `目标 JSON 结构（所有字段都必须输出）：
{
  "chapter_goal": "本章要帮助读者达成的具体目标，非空字符串",
  "reader_questions": ["读者阅读本章前会提出的问题"],
  "must_explain": ["本章必须讲透的概念、机制或因果关系，至少一项"],
  "must_include_examples": ["本章必须包含的案例、计算或演示，至少一项"],
  "avoid_overlap": ["应避免与其他章节重复的内容"],
  "bridge_context": {
    "from_previous": "如何承接上一章，无则为空字符串",
    "to_next": "如何引出下一章，无则为空字符串"
  }
}
不得省略 must_explain 或 must_include_examples；即使材料简短，也要依据章节标题、摘要和材料提炼具体内容。`

const seriesDraftJSONContract = `目标 JSON 结构（所有字段都必须输出）：
{
  "draft_markdown": "完整章节 Markdown，必须实际包含机制解释、示例和复现步骤",
  "coverage_check": {
    "goal_covered": true,
    "mechanism_explained": true,
    "examples_present": true,
    "repro_present": true,
    "edge_cases_present": true
  },
  "example_inventory": [{"example_type":"示例类型","supports_claim":"该示例支撑的具体论点"}]
}
不得用 false 绕过写作要求；若任一覆盖项尚未满足，先补充 draft_markdown，再把对应布尔值设为 true。example_inventory 至少一项。`

const seriesReviewJSONContract = `目标 JSON 结构（所有字段都必须输出）：
{
  "depth_issues": [],
  "example_issues": [],
  "structure_issues": [],
  "revision_actions": ["至少一项具体、可执行的修订动作；无明显缺陷时写明保持哪些优点"],
  "scorecard": {"depth":4,"examples":4,"reproducibility":4,"clarity":4}
}
scorecard 每项使用 1-5 分整数；revision_actions 至少一项。`

func seriesJSONContractForStage(stageName string) string {
	switch stageName {
	case "章节理解":
		return seriesUnderstandingJSONContract
	case "章节草稿", "章节草稿修复":
		return seriesDraftJSONContract
	case "章节审稿":
		return seriesReviewJSONContract
	}
	return ""
}

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
	contract := seriesJSONContractForStage(stageName)
	messages := []llm.Message{
		{Role: "system", Content: seriesPrefix + "\n当前阶段：" + stageName + " JSON 修复"},
		{
			Role: "user",
			Content: fmt.Sprintf(
				"下面是上一轮输出的 JSON 或近似 JSON，但它未通过结构化校验。\n校验错误：%v\n\n%s\n\n原始输出：\n%s\n\n请只修复缺失字段、布尔门禁或 JSON 格式，保持已有有效内容，不要扩写成新文章，返回严格 JSON。",
				validationErr,
				contract,
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
				"当前章节标题：%s\n章节摘要：%s\n材料：\n%s\n\n%s\n\n请仅返回符合上述结构的严格 JSON。",
				chapter.Title,
				chapter.Summary,
				chapterSourceContent,
				seriesUnderstandingJSONContract,
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
		if strings.Contains(err.Error(), "unexpected end of JSON input") {
			// Why: 章节草稿正文很长，LLM 偶尔会把 JSON 结尾截断；只要 draft_markdown 已经产出，
			// 就先本地恢复可用草稿，后续审稿/修订阶段再继续补齐质量，而不是把整章直接判死。
			salvaged, salvageErr := salvageSeriesDraftFromTruncatedJSON(raw)
			if salvageErr == nil {
				return salvaged, nil
			}
		}
		return seriesChapterDraft{}, fmt.Errorf("unmarshal chapter draft: %w", err)
	}
	if err := validateSeriesChapterDraft(result); err != nil {
		return seriesChapterDraft{}, err
	}
	return result, nil
}

func salvageSeriesDraftFromTruncatedJSON(raw string) (seriesChapterDraft, error) {
	draftMarkdown, ok := extractPossiblyTruncatedJSONStringField(raw, "draft_markdown")
	if !ok || strings.TrimSpace(draftMarkdown) == "" {
		return seriesChapterDraft{}, fmt.Errorf("draft_markdown is required")
	}

	exampleType, hasExampleType := extractPossiblyTruncatedJSONStringField(raw, "example_type")
	supportsClaim, hasSupportsClaim := extractPossiblyTruncatedJSONStringField(raw, "supports_claim")
	if !hasExampleType || strings.TrimSpace(exampleType) == "" {
		exampleType = "recovered_draft"
	}
	if !hasSupportsClaim || strings.TrimSpace(supportsClaim) == "" {
		supportsClaim = "从截断 JSON 中恢复章节草稿，待后续审稿阶段复核"
	}

	result := seriesChapterDraft{
		DraftMarkdown: draftMarkdown,
		CoverageCheck: seriesChapterCoverageCheck{
			GoalCovered:        coverageFlagOrDefaultTrue(raw, "goal_covered"),
			MechanismExplained: coverageFlagOrDefaultTrue(raw, "mechanism_explained"),
			ExamplesPresent:    coverageFlagOrDefaultTrue(raw, "examples_present"),
			ReproPresent:       coverageFlagOrDefaultTrue(raw, "repro_present"),
			EdgeCasesPresent:   coverageFlagOrDefaultTrue(raw, "edge_cases_present"),
		},
		ExampleInventory: []seriesChapterExample{{
			ExampleType:   exampleType,
			SupportsClaim: supportsClaim,
		}},
	}
	if err := validateSeriesChapterDraft(result); err != nil {
		return seriesChapterDraft{}, err
	}
	return result, nil
}

func coverageFlagOrDefaultTrue(raw string, key string) bool {
	return !strings.Contains(raw, fmt.Sprintf(`"%s":false`, key))
}

func extractPossiblyTruncatedJSONStringField(raw string, key string) (string, bool) {
	anchor := `"` + key + `"`
	index := strings.Index(raw, anchor)
	if index < 0 {
		return "", false
	}

	remainder := raw[index+len(anchor):]
	colonIndex := strings.Index(remainder, ":")
	if colonIndex < 0 {
		return "", false
	}
	remainder = strings.TrimLeft(remainder[colonIndex+1:], " \n\r\t")
	if !strings.HasPrefix(remainder, `"`) {
		return "", false
	}

	return decodePossiblyTruncatedJSONString(remainder[1:])
}

func decodePossiblyTruncatedJSONString(raw string) (string, bool) {
	var builder strings.Builder
	escaped := false

	for i := 0; i < len(raw); i++ {
		char := raw[i]
		if escaped {
			switch char {
			case '"', '\\', '/':
				builder.WriteByte(char)
			case 'b':
				builder.WriteByte('\b')
			case 'f':
				builder.WriteByte('\f')
			case 'n':
				builder.WriteByte('\n')
			case 'r':
				builder.WriteByte('\r')
			case 't':
				builder.WriteByte('\t')
			case 'u':
				if i+4 >= len(raw) {
					return builder.String(), builder.Len() > 0
				}
				decoded, err := strconv.ParseInt(raw[i+1:i+5], 16, 32)
				if err != nil {
					return builder.String(), builder.Len() > 0
				}
				builder.WriteRune(rune(decoded))
				i += 4
			default:
				builder.WriteByte(char)
			}
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if char == '"' {
			return builder.String(), true
		}
		builder.WriteByte(char)
	}

	return builder.String(), builder.Len() > 0
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
	builder.WriteString(seriesDraftJSONContract)
	builder.WriteString("\n")
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
	builder.WriteString(seriesReviewJSONContract)
	builder.WriteString("\n")
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
