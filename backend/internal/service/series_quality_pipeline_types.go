package service

import (
	"fmt"
	"strings"
)

// SeriesChapterUnderstanding 表示章节理解阶段产出的结构化分析结果。
type SeriesChapterUnderstanding struct {
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

// SeriesChapterCoverageCheck 表示章节草稿是否满足最基础的覆盖率门禁。
type SeriesChapterCoverageCheck struct {
	GoalCovered        bool `json:"goal_covered"`
	MechanismExplained bool `json:"mechanism_explained"`
	ExamplesPresent    bool `json:"examples_present"`
	ReproPresent       bool `json:"repro_present"`
	EdgeCasesPresent   bool `json:"edge_cases_present"`
}

// SeriesChapterExample 表示章节草稿中承载关键论点的示例清单。
type SeriesChapterExample struct {
	ExampleType   string `json:"example_type"`
	SupportsClaim string `json:"supports_claim"`
}

// SeriesChapterDraft 表示章节写作阶段产出的结构化草稿。
type SeriesChapterDraft struct {
	DraftMarkdown    string                     `json:"draft_markdown"`
	CoverageCheck    SeriesChapterCoverageCheck `json:"coverage_check"`
	ExampleInventory []SeriesChapterExample     `json:"example_inventory"`
}

// SeriesChapterReview 表示章节审稿阶段给出的结构化问题与修订动作。
type SeriesChapterReview struct {
	DepthIssues     []string               `json:"depth_issues"`
	ExampleIssues   []string               `json:"example_issues"`
	StructureIssues []string               `json:"structure_issues"`
	RevisionActions []string               `json:"revision_actions"`
	Scorecard       SeriesChapterScorecard `json:"scorecard"`
}

// SeriesChapterScorecard 表示章节质量审稿的四维评分。
type SeriesChapterScorecard struct {
	Depth           int `json:"depth"`
	Examples        int `json:"examples"`
	Reproducibility int `json:"reproducibility"`
	Clarity         int `json:"clarity"`
}

// SeriesChapterFinal 表示终稿补强阶段回收的最终章节结果。
type SeriesChapterFinal struct {
	FinalMarkdown    string                 `json:"final_markdown"`
	ResolvedIssues   []string               `json:"resolved_issues"`
	ResidualRisks    []string               `json:"residual_risks"`
	Usage            SeriesChapterUsage     `json:"usage"`
	QualityScorecard SeriesChapterScorecard `json:"quality_scorecard"`
	RevisionActions  []string               `json:"revision_actions"`
}

// SeriesChapterUsage 表示单章节流水线阶段的模型用量摘要。
type SeriesChapterUsage struct {
	EstimatedTokens       int `json:"estimated_tokens,omitempty"`
	PromptTokens          int `json:"prompt_tokens"`
	CompletionTokens      int `json:"completion_tokens"`
	PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens"`
	PromptCacheMissTokens int `json:"prompt_cache_miss_tokens"`
}

func (u SeriesChapterUsage) add(other SeriesChapterUsage) SeriesChapterUsage {
	return SeriesChapterUsage{
		EstimatedTokens:       u.EstimatedTokens + other.EstimatedTokens,
		PromptTokens:          u.PromptTokens + other.PromptTokens,
		CompletionTokens:      u.CompletionTokens + other.CompletionTokens,
		PromptCacheHitTokens:  u.PromptCacheHitTokens + other.PromptCacheHitTokens,
		PromptCacheMissTokens: u.PromptCacheMissTokens + other.PromptCacheMissTokens,
	}
}

// Why: 这些校验函数是后续质量流水线的硬门禁，先在结构体边界拦截缺失字段，
// 能避免不完整的 LLM 结构化输出继续流入后续阶段，降低“看似成功但内容空心”的风险。
func validateSeriesChapterUnderstanding(item SeriesChapterUnderstanding) error {
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

func validateSeriesChapterDraft(item SeriesChapterDraft) error {
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

func validateSeriesChapterReview(item SeriesChapterReview) error {
	if len(item.RevisionActions) == 0 {
		return fmt.Errorf("revision_actions is required")
	}
	return nil
}

func scorecardBelowThreshold(scorecard SeriesChapterScorecard, threshold int) bool {
	return scorecard.Depth > 0 && scorecard.Depth < threshold ||
		scorecard.Examples > 0 && scorecard.Examples < threshold ||
		scorecard.Reproducibility > 0 && scorecard.Reproducibility < threshold ||
		scorecard.Clarity > 0 && scorecard.Clarity < threshold
}
