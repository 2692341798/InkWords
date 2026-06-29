package generation

import (
	"fmt"
	"os"
	"strings"
	"sync"

	llm "inkwords-backend/shared/platform/llm"
	sharedblog "inkwords-backend/shared/kernel/blog"
	"inkwords-backend/shared/kernel/prompt"
)

// seriesChapterUsage 表示单章节流水线阶段的模型用量摘要。
type seriesChapterUsage struct {
	EstimatedTokens       int `json:"estimated_tokens,omitempty"`
	PromptTokens          int `json:"prompt_tokens"`
	CompletionTokens      int `json:"completion_tokens"`
	PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens"`
	PromptCacheMissTokens int `json:"prompt_cache_miss_tokens"`
}

func (u seriesChapterUsage) add(other seriesChapterUsage) seriesChapterUsage {
	return seriesChapterUsage{
		EstimatedTokens:       u.EstimatedTokens + other.EstimatedTokens,
		PromptTokens:          u.PromptTokens + other.PromptTokens,
		CompletionTokens:      u.CompletionTokens + other.CompletionTokens,
		PromptCacheHitTokens:  u.PromptCacheHitTokens + other.PromptCacheHitTokens,
		PromptCacheMissTokens: u.PromptCacheMissTokens + other.PromptCacheMissTokens,
	}
}

func usageFromCompletionUsage(usage llm.CompletionUsage) seriesChapterUsage {
	return seriesChapterUsage{
		PromptTokens:          usage.PromptTokens,
		CompletionTokens:      usage.CompletionTokens,
		PromptCacheHitTokens:  usage.PromptCacheHitTokens,
		PromptCacheMissTokens: usage.PromptCacheMissTokens,
	}
}

// seriesChapterUnderstanding 表示章节理解阶段产出的结构化分析结果。
type seriesChapterUnderstanding struct {
	ChapterGoal         string   `json:"chapter_goal"`
	ReaderQuestions     []string `json:"reader_questions"`
	MustExplain         []string `json:"must_explain"`
	MustIncludeExamples []string `json:"must_include_examples"`
	AvoidOverlap        []string `json:"avoid_overlap"`
	BridgeContext       struct {
		FromPrevious string `json:"from_previous"`
		ToNext       string `json:"to_next"`
	} `json:"bridge_context"`
}

// seriesChapterCoverageCheck 表示章节草稿是否满足最基础的覆盖率门禁。
type seriesChapterCoverageCheck struct {
	GoalCovered        bool `json:"goal_covered"`
	MechanismExplained bool `json:"mechanism_explained"`
	ExamplesPresent    bool `json:"examples_present"`
	ReproPresent       bool `json:"repro_present"`
	EdgeCasesPresent   bool `json:"edge_cases_present"`
}

// seriesChapterExample 表示章节草稿中承载关键论点的示例清单。
type seriesChapterExample struct {
	ExampleType   string `json:"example_type"`
	SupportsClaim string `json:"supports_claim"`
}

// seriesChapterDraft 表示章节写作阶段产出的结构化草稿。
type seriesChapterDraft struct {
	DraftMarkdown    string                     `json:"draft_markdown"`
	CoverageCheck    seriesChapterCoverageCheck `json:"coverage_check"`
	ExampleInventory []seriesChapterExample     `json:"example_inventory"`
}

// seriesChapterReview 表示章节审稿阶段给出的结构化问题与修订动作。
type seriesChapterReview struct {
	DepthIssues     []string               `json:"depth_issues"`
	ExampleIssues   []string               `json:"example_issues"`
	StructureIssues []string               `json:"structure_issues"`
	RevisionActions []string               `json:"revision_actions"`
	Scorecard       seriesChapterScorecard `json:"scorecard"`
}

// seriesChapterScorecard 表示章节质量审稿的四维评分。
type seriesChapterScorecard struct {
	Depth           int `json:"depth"`
	Examples        int `json:"examples"`
	Reproducibility int `json:"reproducibility"`
	Clarity         int `json:"clarity"`
}

// seriesChapterFinal 表示终稿补强阶段回收的最终章节结果。
type seriesChapterFinal struct {
	FinalMarkdown    string                 `json:"final_markdown"`
	ResolvedIssues   []string               `json:"resolved_issues"`
	ResidualRisks    []string               `json:"residual_risks"`
	Usage            seriesChapterUsage     `json:"usage"`
	QualityScorecard seriesChapterScorecard `json:"quality_scorecard"`
	RevisionActions  []string               `json:"revision_actions"`
}

func scorecardBelowThreshold(scorecard seriesChapterScorecard, threshold int) bool {
	return scorecard.Depth > 0 && scorecard.Depth < threshold ||
		scorecard.Examples > 0 && scorecard.Examples < threshold ||
		scorecard.Reproducibility > 0 && scorecard.Reproducibility < threshold ||
		scorecard.Clarity > 0 && scorecard.Clarity < threshold
}

func validateSeriesChapterUnderstanding(item seriesChapterUnderstanding) error {
	if strings.TrimSpace(item.ChapterGoal) == "" {
		return fmt.Errorf("chapter_goal is required")
	}
	if len(item.MustExplain) == 0 {
		return fmt.Errorf("must_explain is required")
	}
	if len(item.MustIncludeExamples) == 0 {
		return fmt.Errorf("must_include_examples is required")
	}
	return nil
}

func validateSeriesChapterDraft(item seriesChapterDraft) error {
	if strings.TrimSpace(item.DraftMarkdown) == "" {
		return fmt.Errorf("draft_markdown is required")
	}
	if !item.CoverageCheck.MechanismExplained {
		return fmt.Errorf("mechanism_explained must be true")
	}
	if !item.CoverageCheck.ExamplesPresent {
		return fmt.Errorf("examples_present must be true")
	}
	if !item.CoverageCheck.ReproPresent {
		return fmt.Errorf("repro_present must be true")
	}
	if len(item.ExampleInventory) == 0 {
		return fmt.Errorf("example_inventory is required")
	}
	return nil
}

func validateSeriesChapterReview(item seriesChapterReview) error {
	if len(item.RevisionActions) == 0 {
		return fmt.Errorf("revision_actions is required")
	}
	return nil
}

// seriesChapterTaskResult 表示单章节在任务结果中的快照。
type seriesChapterTaskResult struct {
	BlogID           string                 `json:"blog_id"`
	ChapterSort      int                    `json:"chapter_sort"`
	Title            string                 `json:"title"`
	Content          string                 `json:"content"`
	WordCount        int                    `json:"word_count"`
	TechStacks       []string               `json:"tech_stacks"`
	Status           string                 `json:"status"`
	ErrorMessage     string                 `json:"error_message"`
	Usage            seriesChapterUsage     `json:"usage,omitempty"`
	QualityScorecard seriesChapterScorecard `json:"quality_scorecard,omitempty"`
	RevisionActions  []string               `json:"revision_actions,omitempty"`
	ResolvedIssues   []string               `json:"resolved_issues,omitempty"`
	ResidualRisks    []string               `json:"residual_risks,omitempty"`
}

// seriesTaskResultCollector 聚合系列生成任务的结果。
type seriesTaskResultCollector struct {
	mu              sync.Mutex
	ParentBlogID    string
	ParentTitle     string
	ParentContent   string
	EstimatedTokens int
	Usage           seriesChapterUsage
	Chapters        []seriesChapterTaskResult
}

func newSeriesTaskResultCollector(parentBlogID string, parentTitle string) *seriesTaskResultCollector {
	return &seriesTaskResultCollector{
		ParentBlogID: parentBlogID,
		ParentTitle:  parentTitle,
		Chapters:     make([]seriesChapterTaskResult, 0),
	}
}

func (c *seriesTaskResultCollector) AddChapterSuccess(chapter sharedblog.Chapter, content string, wordCount int, techStacks []string) {
	c.AddChapterSuccessWithQuality(chapter, content, wordCount, techStacks, seriesChapterFinal{})
}

func (c *seriesTaskResultCollector) AddChapterSuccessWithQuality(chapter sharedblog.Chapter, content string, wordCount int, techStacks []string, qualityResult seriesChapterFinal) {
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

func (c *seriesTaskResultCollector) AddChapterFailure(chapter sharedblog.Chapter, errorMessage string) {
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

// seriesQualityPipelineInput 质量流水线输入。
type seriesQualityPipelineInput struct {
	SeriesTitle          string
	ReaderProfile        string
	Outline              []sharedblog.Chapter
	ChapterIndex         int
	Chapter              sharedblog.Chapter
	ChapterSourceContent string
	GitURL               string
	OldContent           string
	UserID               string
	ProgressChan         chan<- string
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

func buildSeriesChapterExtraRequirements(gitURL string, outline []sharedblog.Chapter, chapterIndex int) string {
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

func maxWorkersFromEnv(taskCount int) int {
	return resolveWorkerLimit(taskCount, 3, "MAX_CONCURRENT_WORKERS")
}

func maxWorkersForModel(model string, taskCount int) int {
	normalized := strings.ToLower(strings.TrimSpace(model))
	if strings.Contains(normalized, "pro") {
		return resolveWorkerLimit(taskCount, 2, "LLM_PRO_CONCURRENCY", "MAX_CONCURRENT_WORKERS")
	}
	return resolveWorkerLimit(taskCount, 4, "LLM_FLASH_CONCURRENCY", "MAX_CONCURRENT_WORKERS")
}

func resolveWorkerLimit(taskCount int, defaultLimit int, envKeys ...string) int {
	maxWorkers := defaultLimit
	if maxWorkers <= 0 {
		maxWorkers = 1
	}
	for _, key := range envKeys {
		if v := os.Getenv(key); v != "" {
			if n, err := parseInt(v); err == nil && n > 0 {
				maxWorkers = n
				break
			}
		}
	}
	if maxWorkers > 50 {
		maxWorkers = 50
	}
	if taskCount > 0 && taskCount < maxWorkers {
		maxWorkers = taskCount
	}
	if maxWorkers < 1 {
		return 1
	}
	return maxWorkers
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
