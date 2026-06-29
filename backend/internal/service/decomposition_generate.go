package service

import (
	"context"
	"encoding/json"
	"fmt"
	blogcontracts "inkwords-backend/internal/domain/blog/contracts"
	"inkwords-backend/internal/prompt"
	"sort"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/sync/semaphore"
)

type seriesChapterTaskResult struct {
	BlogID           string                 `json:"blog_id"`
	ChapterSort      int                    `json:"chapter_sort"`
	Title            string                 `json:"title"`
	Content          string                 `json:"content"`
	WordCount        int                    `json:"word_count"`
	TechStacks       []string               `json:"tech_stacks"`
	Status           string                 `json:"status"`
	ErrorMessage     string                 `json:"error_message"`
	Usage            SeriesChapterUsage     `json:"usage,omitempty"`
	QualityScorecard SeriesChapterScorecard `json:"quality_scorecard,omitempty"`
	RevisionActions  []string               `json:"revision_actions,omitempty"`
	ResolvedIssues   []string               `json:"resolved_issues,omitempty"`
	ResidualRisks    []string               `json:"residual_risks,omitempty"`
}

type seriesTaskResultCollector struct {
	mu              sync.Mutex
	ParentBlogID    string
	ParentTitle     string
	ParentContent   string
	EstimatedTokens int
	Usage           SeriesChapterUsage
	Chapters        []seriesChapterTaskResult
}

func newSeriesTaskResultCollector(parentBlogID string, parentTitle string) *seriesTaskResultCollector {
	return &seriesTaskResultCollector{
		ParentBlogID: parentBlogID,
		ParentTitle:  parentTitle,
		Chapters:     make([]seriesChapterTaskResult, 0),
	}
}

func (c *seriesTaskResultCollector) AddChapterSuccess(chapter blogcontracts.Chapter, content string, wordCount int, techStacks []string) {
	c.AddChapterSuccessWithQuality(chapter, content, wordCount, techStacks, SeriesChapterFinal{})
}

func (c *seriesTaskResultCollector) AddChapterSuccessWithQuality(chapter blogcontracts.Chapter, content string, wordCount int, techStacks []string, qualityResult SeriesChapterFinal) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.Chapters = append(c.Chapters, seriesChapterTaskResult{
		BlogID:           chapter.ID,
		ChapterSort:      chapter.Sort,
		Title:            chapter.Title,
		Content:          content,
		WordCount:        wordCount,
		TechStacks:       append([]string(nil), techStacks...),
		Status:           "succeeded",
		ErrorMessage:     "",
		Usage:            qualityResult.Usage,
		QualityScorecard: qualityResult.QualityScorecard,
		RevisionActions:  append([]string(nil), qualityResult.RevisionActions...),
		ResolvedIssues:   append([]string(nil), qualityResult.ResolvedIssues...),
		ResidualRisks:    append([]string(nil), qualityResult.ResidualRisks...),
	})
	c.EstimatedTokens += wordCount * 2
	qualityUsage := qualityResult.Usage
	if qualityUsage.EstimatedTokens == 0 {
		qualityUsage.EstimatedTokens = wordCount * 2
	}
	c.Usage = c.Usage.add(qualityUsage)
}

func (c *seriesTaskResultCollector) AddChapterFailure(chapter blogcontracts.Chapter, errorMessage string) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.Chapters = append(c.Chapters, seriesChapterTaskResult{
		BlogID:       chapter.ID,
		ChapterSort:  chapter.Sort,
		Title:        chapter.Title,
		Content:      "章节生成失败，请重试。",
		WordCount:    0,
		TechStacks:   []string{},
		Status:       "failed",
		ErrorMessage: errorMessage,
	})
}

func (c *seriesTaskResultCollector) SetParentContent(content string) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.ParentContent = content
	c.EstimatedTokens += len([]rune(content)) * 2
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

	sort.Slice(chapters, func(i, j int) bool {
		return chapters[i].ChapterSort < chapters[j].ChapterSort
	})

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

// GenerateSeries generates blog chapters sequentially based on the outline with streaming.
func (s *DecompositionService) GenerateSeries(
	ctx context.Context,
	userID uuid.UUID,
	parentID uuid.UUID,
	seriesTitle string,
	outline []blogcontracts.Chapter,
	sourceContent string,
	sourceType string,
	gitURL string,
	scenarioMode prompt.ScenarioMode,
	style string,
	progressChan chan<- string,
	errChan chan<- error,
) {
	s.GenerateSeriesWithProfile(
		ctx,
		userID,
		parentID,
		seriesTitle,
		outline,
		sourceContent,
		sourceType,
		gitURL,
		scenarioMode,
		style,
		prompt.PromptProfile{},
		progressChan,
		errChan,
	)
}

func (s *DecompositionService) GenerateSeriesWithProfile(
	ctx context.Context,
	userID uuid.UUID,
	parentID uuid.UUID,
	seriesTitle string,
	outline []blogcontracts.Chapter,
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

	sendSystemProgress := func(msg string) {
		progressMsg := map[string]interface{}{
			"status":  "progress",
			"message": msg,
		}
		bytes, _ := json.Marshal(progressMsg)
		progressChan <- string(bytes)
	}

	sendSystemProgress("正在准备环境...")

	// --- FIX START: Clone repo to precisely feed files ---
	var cachePath string
	if sourceType == "git" && gitURL != "" {
		sendSystemProgress("正在准备环境与代码...")
		dir, err := s.gitFetcher.GetCachedRepoPath(gitURL, sendSystemProgress)
		if err == nil {
			cachePath = dir
		}
	}

	sendSystemProgress("正在初始化数据库与生成队列...")

	parentTitle := "Git 源码解析系列"
	if sourceType == "file" {
		parentTitle = "文件解析系列"
	}
	if seriesTitle != "" {
		parentTitle = seriesTitle
	}

	updatedOutline, err := s.ensureSeriesParentAndDrafts(
		ctx,
		userID,
		parentID,
		parentTitle,
		sourceType,
		gitURL,
		outline,
	)
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

	sendSystemProgress("开始生成系列博客内容...")
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
		go func(i int, chapter blogcontracts.Chapter) {
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
			techStacks := decodeTechStacksJSON(s.extractSeriesChapterTechStacks(ctx, llmModel, content))

			if err := s.handleSeriesChapterCompletion(
				ctx,
				userID,
				parentID,
				sourceType,
				chapter,
				content,
				wordCount,
				techStacks,
				qualityResult,
				resultCollector,
			); err != nil {
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
