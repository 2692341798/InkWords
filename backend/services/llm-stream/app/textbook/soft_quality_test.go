package textbook

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAssessSampleChapterSoftQualityReportsReviewRisksWithoutScoring(t *testing.T) {
	longParagraph := strings.Repeat("这个句子同时解释前提机制结果和例外，", 50)
	markdown := `## 为什么要这样做
很短。

## 跟着源码走三步
先进入入口，然后调用登记函数，最终进入查找流程，查找完成后返回处理函数。

## 补充观察

` + longParagraph + `

重复段落用于测试人工审阅提示，它有足够长度并且不会被当成一句无意义的短文本。

重复段落用于测试人工审阅提示，它有足够长度并且不会被当成一句无意义的短文本。

当然，你只需要照做。

### 完整示范
这里只写最终结论。

### 引导练习
提示：照着示例完成。

### 脚手架渐退
去掉提示后完成。

### 独立迁移
提示：继续照着示例完成。`

	advisories := AssessSampleChapterSoftQuality(markdown)
	codes := make([]string, 0, len(advisories))
	for _, item := range advisories {
		codes = append(codes, item.Code)
		require.NotEmpty(t, item.Dimension)
		require.NotEmpty(t, item.Evidence)
		require.NotEmpty(t, item.ReviewQuestion)
	}
	require.ElementsMatch(t, []string{
		"long_paragraph",
		"long_sentence",
		"thin_heading_promise",
		"repeated_paragraph",
		"over_assuring_tone",
		"worked_example_link_weak",
		"section_rhythm_imbalance",
		"flow_visual_opportunity",
		"exercise_scaffolding_not_faded",
	}, codes)
}

func TestSampleQualityReportKeepsSoftReviewSeparateFromHardGate(t *testing.T) {
	pack := ginEvidencePack(t)
	chapter, err := BuildGinRequestLifecycleSample(pack)
	require.NoError(t, err)
	chapter.Markdown += "\n\n" + strings.Repeat("这是一段需要人工考虑是否拆分的补充说明。", 30)
	chapter.ContentHash = digest(chapter.Markdown)

	report := RunSampleChapterQualityGates(chapter, pack)

	require.True(t, report.Passed, report.Failures)
	require.True(t, report.ManualReviewRequired)
	require.Equal(t, sampleManualReviewDimensions, report.ManualReviewDimensions)
	require.NotEmpty(t, report.Advisories)
	require.Contains(t, report.Advisories[0].Code, "long_")
}
