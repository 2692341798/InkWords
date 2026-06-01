package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSeriesChapterUnderstanding_RejectsMissingMechanismAndExamples(t *testing.T) {
	understanding := SeriesChapterUnderstanding{
		ChapterGoal:         "解释 Gin 路由链路",
		ReaderQuestions:     []string{"请求如何进入 handler"},
		MustExplain:         nil,
		MustIncludeExamples: nil,
	}

	err := validateSeriesChapterUnderstanding(understanding)

	require.ErrorContains(t, err, "must_explain")
}

func TestValidateSeriesChapterDraft_RequiresExampleAndRepro(t *testing.T) {
	draft := SeriesChapterDraft{
		DraftMarkdown: "## Gin 路由\n\n这里只讲概念，没有命令。",
		CoverageCheck: SeriesChapterCoverageCheck{
			GoalCovered:        true,
			MechanismExplained: true,
			ExamplesPresent:    false,
			ReproPresent:       false,
			EdgeCasesPresent:   true,
		},
	}

	err := validateSeriesChapterDraft(draft)

	require.ErrorContains(t, err, "examples_present")
}

func TestValidateSeriesChapterReview_RequiresRevisionActions(t *testing.T) {
	review := SeriesChapterReview{
		DepthIssues:     []string{"没有解释中间件链如何短路"},
		ExampleIssues:   []string{"没有 curl 示例"},
		RevisionActions: nil,
	}

	err := validateSeriesChapterReview(review)

	require.ErrorContains(t, err, "revision_actions")
}
