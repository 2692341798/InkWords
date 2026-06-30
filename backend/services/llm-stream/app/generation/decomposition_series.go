package generation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/semaphore"

	llm "inkwords-backend/shared/platform/llm"
	"inkwords-backend/shared/platform/parser"
	sharedblog "inkwords-backend/shared/kernel/blog"
	"inkwords-backend/shared/kernel/prompt"
)

const (
	seriesChapterSourceRuneLimit = 1000000
	seriesOldContentRuneLimit    = 500000
	seriesContentTruncatedSuffix = "\n\n... [Content Truncated due to length limits] ..."
)

// GenerateSeriesWithProfile 基于大纲和源内容流式生成系列博客。
//
//nolint:gocyclo
func (s *DecompositionService) GenerateSeriesWithProfile(
	ctx context.Context,
	userID uuid.UUID,
	parentID uuid.UUID,
	seriesTitle string,
	outline []sharedblog.Chapter,
	sourceContent string,
	sourceType string,
	gitURL string,
	scenarioMode prompt.ScenarioMode,
	style string,
	profile prompt.PromptProfile,
	progressChan chan<- string,
	errChan chan<- error,
) {
	defer close(progressChan)
	defer close(errChan)

	if !scenarioMode.IsValid() {
		scenarioMode = prompt.DefaultScenarioModeForSource(sourceType)
	}
	profile = normalizePromptProfile(profile, scenarioMode)

	sendSeriesSystemProgress(progressChan, "正在准备环境...")

	var cachePath string
	if sourceType == "git" && gitURL != "" {
		sendSeriesSystemProgress(progressChan, "正在准备环境与代码...")
		dir, err := s.gitFetcher.GetCachedRepoPath(gitURL, func(msg string) {
			sendSeriesSystemProgress(progressChan, msg)
		})
		if err == nil {
			cachePath = dir
		}
	}

	sendSeriesSystemProgress(progressChan, "正在初始化数据库与生成队列...")

	parentTitle := "Git 源码解析系列"
	if sourceType == "file" {
		parentTitle = "文件解析系列"
	}
	if seriesTitle != "" {
		parentTitle = seriesTitle
	}

	updatedOutline, err := s.ensureSeriesParentAndDrafts(ctx, userID, parentID, parentTitle, sourceType, gitURL, outline)
	if err != nil {
		errChan <- fmt.Errorf("prepare series persistence: %w", err)
		return
	}
	outline = updatedOutline

	var resultCollector *seriesTaskResultCollector
	if taskOnlyPersistenceMode() {
		resultCollector = newSeriesTaskResultCollector(parentID.String(), parentTitle)
	}

	maxWorkers := maxWorkersForModel("deepseek-v4-pro", len(outline))
	sem := semaphore.NewWeighted(int64(maxWorkers))
	var wg sync.WaitGroup

	sendSeriesSystemProgress(progressChan, "开始生成系列博客内容...")
	for i, chapter := range outline {
		if ctx.Err() != nil {
			errChan <- ctx.Err()
			break
		}

		if chapter.Action == "skip" && chapter.ID != "" {
			_ = s.handleSkippedSeriesChapter(ctx, userID, chapter)
			endMsg := map[string]interface{}{
				"status":       "completed",
				"chapter_sort": chapter.Sort,
				"title":        chapter.Title,
			}
			endBytes, _ := json.Marshal(endMsg)
			progressChan <- string(endBytes)
			continue
		}

		if err := sem.Acquire(ctx, 1); err != nil {
			errChan <- err
			break
		}

		wg.Add(1)
		go func(i int, chapter sharedblog.Chapter) {
			defer sem.Release(1)
			defer wg.Done()

			chapterSourceContent := resolveSeriesChapterSourceContent(sourceType, cachePath, sourceContent, chapter)
			oldContent := s.resolveSeriesOldContent(ctx, userID, chapter)
			qualityResult, streamErr := s.runSeriesChapterQualityPipeline(ctx, seriesQualityPipelineInput{
				SeriesTitle:          parentTitle,
				ReaderProfile:        buildSeriesReaderProfile(scenarioMode),
				Outline:              outline,
				ChapterIndex:         i,
				Chapter:              chapter,
				ChapterSourceContent: chapterSourceContent,
				GitURL:               gitURL,
				OldContent:           oldContent,
				UserID:               fmt.Sprintf("series-%s", parentID.String()),
				ProgressChan:         progressChan,
			})
			if streamErr != nil {
				s.handleSeriesChapterFailure(ctx, userID, chapter, streamErr, resultCollector)

				errMsg := map[string]interface{}{
					"status":       "error",
					"chapter_sort": chapter.Sort,
					"message":      fmt.Sprintf("chapter %d generation failed: %v", chapter.Sort, streamErr),
				}
				errBytes, _ := json.Marshal(errMsg)
				progressChan <- string(errBytes)
				return
			}

			content := qualityResult.FinalMarkdown
			wordCount := len([]rune(content))

			llmModel := "deepseek-v4-flash"
			if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
				llmModel = envModel
			}
			techStacks := decodeTechStacksJSON(s.extractSeriesChapterTechStacks(ctx, llmModel, content))

			if err := s.handleSeriesChapterCompletion(ctx, userID, parentID, sourceType, chapter, content, wordCount, techStacks, qualityResult, resultCollector); err != nil {
				errMsg := map[string]interface{}{
					"status":       "error",
					"chapter_sort": chapter.Sort,
					"message":      fmt.Sprintf("failed to persist chapter %d: %v", chapter.Sort, err),
				}
				errBytes, _ := json.Marshal(errMsg)
				progressChan <- string(errBytes)
				return
			}

			endMsg := map[string]interface{}{
				"status":       "completed",
				"chapter_sort": chapter.Sort,
				"title":        chapter.Title,
			}
			endBytes, _ := json.Marshal(endMsg)
			progressChan <- string(endBytes)
		}(i, chapter)
	}

	wg.Wait()

	if ctx.Err() == nil {
		s.generateSeriesIntro(ctx, userID, parentID, seriesTitle, outline, scenarioMode, prompt.ArticleStyle(style), profile, resultCollector, progressChan, errChan)
		if taskOnlyPersistenceMode() && resultCollector != nil {
			resultJSON, err := resultCollector.BuildTaskResult()
			if err != nil {
				errChan <- fmt.Errorf("build generate_series task result: %w", err)
				return
			}
			s.StoreGenerateSeriesTaskResult(parentID, resultJSON)
		}
	}
}

func (c *seriesTaskResultCollector) BuildTaskResult() ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("series task result collector is required")
	}

	c.mu.Lock()
	chapters := append([]seriesChapterTaskResult(nil), c.Chapters...)
	parentBlogID := c.ParentBlogID
	parentTitle := c.ParentTitle
	parentContent := c.ParentContent
	estimatedTokens := c.EstimatedTokens
	usage := c.Usage
	c.mu.Unlock()

	if usage.EstimatedTokens == 0 {
		usage.EstimatedTokens = estimatedTokens
	}

	return json.Marshal(map[string]any{
		"result_version":   1,
		"task_type":        "generation",
		"task_subtype":     "generate_series",
		"persistence_mode": "task_only",
		"final_status":     "succeeded",
		"usage": map[string]any{
			"estimated_tokens":         usage.EstimatedTokens,
			"prompt_tokens":            usage.PromptTokens,
			"completion_tokens":        usage.CompletionTokens,
			"prompt_cache_hit_tokens":  usage.PromptCacheHitTokens,
			"prompt_cache_miss_tokens": usage.PromptCacheMissTokens,
		},
		"payload": map[string]any{
			"parent_blog": map[string]any{
				"blog_id": parentBlogID,
				"title":   parentTitle,
				"content": parentContent,
			},
			"chapters": chapters,
		},
	})
}

func sendSeriesSystemProgress(progressChan chan<- string, message string) {
	bytes, _ := json.Marshal(map[string]interface{}{
		"status":  "progress",
		"message": message,
	})
	progressChan <- string(bytes)
}

// --- Persistence helpers ---

func (s *DecompositionService) ensureSeriesParentAndDrafts(
	ctx context.Context,
	userID uuid.UUID,
	parentID uuid.UUID,
	parentTitle string,
	sourceType string,
	gitURL string,
	outline []sharedblog.Chapter,
) ([]sharedblog.Chapter, error) {
	return s.seriesPersistence.EnsureSeriesParentAndDrafts(ctx, sharedblog.SeriesDraftPreflightInput{
		UserID:      userID,
		ParentID:    parentID,
		ParentTitle: parentTitle,
		SourceType:  sourceType,
		GitURL:      gitURL,
		Outline:     outline,
	})
}

func (s *DecompositionService) handleSeriesChapterCompletion(
	ctx context.Context,
	userID uuid.UUID,
	parentID uuid.UUID,
	sourceType string,
	chapter sharedblog.Chapter,
	content string,
	wordCount int,
	techStacks []string,
	qualityResult seriesChapterFinal,
	collector *seriesTaskResultCollector,
) error {
	if taskOnlyPersistenceMode() {
		collector.AddChapterSuccessWithQuality(chapter, content, wordCount, techStacks, qualityResult)
		return nil
	}

	techStacksJSON, err := json.Marshal(techStacks)
	if err != nil {
		return fmt.Errorf("marshal series chapter tech stacks: %w", err)
	}

	var blogID uuid.UUID
	if chapter.ID != "" {
		parsedID, err := uuid.Parse(chapter.ID)
		if err != nil {
			return fmt.Errorf("parse chapter blog id: %w", err)
		}
		blogID = parsedID
	}

	return s.seriesPersistence.SaveSeriesChapter(ctx, sharedblog.SeriesChapterPersistenceInput{
		UserID:     userID,
		ParentID:   parentID,
		BlogID:     blogID,
		Chapter:    chapter,
		SourceType: sourceType,
		Content:    content,
		WordCount:  wordCount,
		TechStacks: techStacksJSON,
	})
}

func (s *DecompositionService) handleSeriesChapterFailure(
	ctx context.Context,
	userID uuid.UUID,
	chapter sharedblog.Chapter,
	streamErr error,
	collector *seriesTaskResultCollector,
) {
	if taskOnlyPersistenceMode() {
		collector.AddChapterFailure(chapter, streamErr.Error())
		return
	}
	if chapter.ID == "" {
		return
	}
	blogID, err := uuid.Parse(chapter.ID)
	if err != nil {
		return
	}
	_ = s.seriesPersistence.MarkSeriesChapterFailed(ctx, userID, blogID)
}

func (s *DecompositionService) handleSkippedSeriesChapter(ctx context.Context, userID uuid.UUID, chapter sharedblog.Chapter) error {
	if strings.TrimSpace(chapter.ID) == "" {
		return nil
	}
	blogID, err := uuid.Parse(chapter.ID)
	if err != nil {
		return err
	}
	return s.seriesPersistence.UpdateSkippedSeriesChapterMeta(ctx, userID, blogID, chapter)
}

func decodeTechStacksJSON(raw json.RawMessage) []string {
	var techStacks []string
	if len(raw) == 0 {
		return []string{}
	}
	if err := json.Unmarshal(raw, &techStacks); err != nil {
		return []string{}
	}
	return techStacks
}

func (s *DecompositionService) resolveSeriesOldContent(ctx context.Context, userID uuid.UUID, chapter sharedblog.Chapter) string {
	if chapter.Action != "regenerate" || strings.TrimSpace(chapter.ID) == "" {
		return ""
	}
	blogID, err := uuid.Parse(chapter.ID)
	if err != nil {
		return ""
	}
	oldContent, err := s.seriesPersistence.LoadSeriesOldContent(ctx, userID, blogID)
	if err != nil {
		return ""
	}
	return truncateSeriesContent(oldContent, seriesOldContentRuneLimit)
}

// --- Source helpers ---

//nolint:gosec,noctx,staticcheck
func resolveSeriesChapterSourceContent(sourceType, cachePath, fallbackSourceContent string, chapter sharedblog.Chapter) string {
	if sourceType != "git" || cachePath == "" || len(chapter.Files) == 0 {
		return truncateSeriesContent(fallbackSourceContent, seriesChapterSourceRuneLimit)
	}

	var builder strings.Builder
	for _, filePath := range chapter.Files {
		filePath = strings.TrimSpace(filePath)
		if filePath == "" {
			continue
		}

		cmdCheck := exec.Command("git", "cat-file", "-t", "HEAD:"+filePath)
		cmdCheck.Dir = cachePath
		objectTypeBytes, err := cmdCheck.Output()
		if err != nil {
			continue
		}

		if strings.TrimSpace(string(objectTypeBytes)) == "tree" {
			appendSeriesDirectorySource(&builder, cachePath, filePath)
			continue
		}

		appendSeriesFileSource(&builder, cachePath, filePath)
	}

	if builder.Len() == 0 {
		return truncateSeriesContent(fallbackSourceContent, seriesChapterSourceRuneLimit)
	}

	return truncateSeriesContent(builder.String(), seriesChapterSourceRuneLimit)
}

//nolint:gosec,noctx
func appendSeriesDirectorySource(builder *strings.Builder, cachePath, dirPath string) {
	cmdList := exec.Command("git", "ls-tree", "-r", "--name-only", "HEAD", dirPath)
	cmdList.Dir = cachePath
	output, err := cmdList.Output()
	if err != nil {
		return
	}

	files := strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n")
	for _, filePath := range files {
		filePath = strings.TrimSpace(filePath)
		if filePath == "" {
			continue
		}
		appendSeriesFileSource(builder, cachePath, filePath)
	}
}

//nolint:all
func appendSeriesFileSource(builder *strings.Builder, cachePath, filePath string) {
	if parser.IsBinaryExt(strings.ToLower(filepath.Ext(filePath))) {
		return
	}

	cmdShow := exec.Command("git", "show", "HEAD:"+filePath)
	cmdShow.Dir = cachePath
	data, err := cmdShow.Output()
	if err != nil {
		return
	}

	builder.WriteString(fmt.Sprintf("--- File: %s ---\n%s\n\n", filePath, string(data)))
}

func truncateSeriesContent(content string, runeLimit int) string {
	runes := []rune(content)
	if len(runes) <= runeLimit {
		return content
	}
	return string(runes[:runeLimit]) + seriesContentTruncatedSuffix
}

// --- Intro generation ---

//nolint:gocyclo,staticcheck
func (s *DecompositionService) generateSeriesIntro(
	ctx context.Context,
	userID uuid.UUID,
	parentID uuid.UUID,
	seriesTitle string,
	outline []sharedblog.Chapter,
	scenarioMode prompt.ScenarioMode,
	style prompt.ArticleStyle,
	profile prompt.PromptProfile,
	collector *seriesTaskResultCollector,
	progressChan chan<- string,
	errChan chan<- error,
) {
	sendIntroProgress := func(status string, content string, message string) {
		msg := map[string]interface{}{
			"status":       status,
			"chapter_sort": 0,
			"content":      content,
			"message":      message,
			"title":        "系列导读",
		}
		bytes, _ := json.Marshal(msg)
		progressChan <- string(bytes)
	}

	sendIntroProgress("generating", "", "")

	var outlineStrBuilder strings.Builder
	for _, ch := range outline {
		outlineStrBuilder.WriteString(fmt.Sprintf("- %s: %s\n", ch.Title, ch.Summary))
	}

	if !scenarioMode.IsValid() {
		scenarioMode = prompt.ScenarioModeEbookInterpretation
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
		if resolved, err := s.promptReq.ResolveWithProfile(ctx, userID, scenarioMode, style, profile); err == nil && resolved != "" {
			requirements = resolved
		}
	}

	promptText := fmt.Sprintf(`请根据以下系列文章的大纲，编写一篇高质量的"系列导读"或"总结"文章（约500-800字）。
这篇文章将作为整个系列的入口，吸引读者阅读。
系列标题：%s
各章节大纲：
%s

写作要求：
%s

额外要求：
1. 简明扼要地介绍这个系列将要解决的问题和核心价值。
2. 简述各个章节的精彩看点，引导读者循序渐进地阅读。
3. 结尾给出学习建议或寄语。
`, seriesTitle, outlineStrBuilder.String(), requirements)

	messages := []llm.Message{
		{Role: "system", Content: profile.SystemRole + "\n\n你擅长编写引人入胜的系列导读。"},
		{Role: "user", Content: promptText},
	}

	llmModel := "deepseek-v4-flash"
	if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
		llmModel = envModel
	}

	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()

	chunkChan := make(chan string, 100)
	internalErrChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(internalErrChan)

		tempChunkChan := make(chan string)
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			for chunk := range tempChunkChan {
				chunkChan <- chunk
			}
		}()

		_, err := s.llmClient.GenerateStream(streamCtx, llmModel, messages, tempChunkChan)
		wg.Wait()
		if err != nil {
			internalErrChan <- err
		}
	}()

	var contentBuilder strings.Builder
	idleTimeout := 60 * time.Second
	timer := time.NewTimer(idleTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		case <-timer.C:
			streamCancel()
			errChan <- fmt.Errorf("intro generation idle timeout")
			return
		case err, ok := <-internalErrChan:
			if ok && err != nil {
				sendIntroProgress("error", "", err.Error())
				if !taskOnlyPersistenceMode() {
					_ = s.seriesPersistence.MarkSeriesIntroFailed(ctx, userID, parentID)
				}
				return
			}
		case chunk, ok := <-chunkChan:
			if !ok {
				finalContent := contentBuilder.String()
				if taskOnlyPersistenceMode() {
					collector.SetParentContent(finalContent)
					sendIntroProgress("completed", "", "")
					return
				}
				if err := s.seriesPersistence.SaveSeriesIntro(ctx, userID, parentID, finalContent); err != nil {
					errChan <- fmt.Errorf("persist series intro: %w", err)
					return
				}
				sendIntroProgress("completed", "", "")
				return
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(idleTimeout)
			contentBuilder.WriteString(chunk)
			sendIntroProgress("streaming", chunk, "")
		}
	}
}

// --- Tech stacks extraction ---

func (s *DecompositionService) extractSeriesChapterTechStacks(ctx context.Context, llmModel, content string) json.RawMessage {
	var techStacks json.RawMessage
	extractPrompt := "请从以下文章内容中提取出涉及的核心技术栈名称（如 React, Go, Docker 等），以 JSON 数组格式返回，不要有任何其他多余字符。\n\n例如：[\"React\", \"Go\"]\n\n文章内容：\n\n" + content
	extractMessages := []llm.Message{{Role: "user", Content: extractPrompt}}
	extractedJSON, _, err := s.llmClient.GenerateJSONWithOptions(ctx, llmModel, extractMessages, llm.LightweightChatOptions("", 512))
	if err != nil || len(extractedJSON) == 0 {
		return techStacks
	}

	var parsed []string
	if json.Unmarshal([]byte(extractedJSON), &parsed) != nil {
		return techStacks
	}

	return json.RawMessage(extractedJSON)
}
