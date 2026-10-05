package textbook

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRecommendToolAndBuildRunbookAreExplicitAboutManualCapture(t *testing.T) {
	recommendation, err := RecommendTool("go", "调用链")
	require.NoError(t, err)
	require.Equal(t, "GoLand", recommendation.Primary)
	require.True(t, recommendation.ManualCaptureRequired)
	steps, err := BuildRunbook("csharp", "内存")
	require.NoError(t, err)
	require.Len(t, steps, 3)
	require.Contains(t, steps[1].Action, "断点")
	require.NotEmpty(t, steps[1].Recovery)
	require.NotEmpty(t, steps[1].ToolVersion)
	require.NotEmpty(t, steps[1].ShortcutOrMenu)
	require.NotEmpty(t, steps[1].Input)
	_, err = RecommendTool("rust", "debug")
	require.Error(t, err)
}

func TestRunbooksAdaptObservationStepsToToolAndGoal(t *testing.T) {
	visualStudio, err := BuildRunbook("cpp", "内存")
	require.NoError(t, err)
	require.Len(t, visualStudio, 3)
	require.Equal(t, "Visual Studio 2022", visualStudio[2].Tool)
	require.Contains(t, visualStudio[2].ShortcutOrMenu, "F11")
	require.Contains(t, visualStudio[2].Input, "采样条件")

	browser, err := BuildRunbook("web", "性能")
	require.NoError(t, err)
	require.Len(t, browser, 3)
	require.Equal(t, "浏览器开发者工具", browser[0].Tool)
	require.Contains(t, browser[0].Input, "Performance")
	require.Contains(t, browser[1].Action, "清空旧记录")
}

func TestVideoRunbookProjectionIsRevisionSafeUntilHumanCapture(t *testing.T) {
	projection, err := BuildVideoRunbookProjection("go", "调用链")
	require.NoError(t, err)
	require.NoError(t, projection.Validate())
	require.Equal(t, "inkwords.video-runbook.v1", projection.Format)
	require.Equal(t, "GoLand", projection.Recommendation.Primary)
	require.True(t, projection.ManualCapturePending)
	require.Equal(t, "unverified", string(projection.VerificationStatus))
	require.NotEmpty(t, projection.CaptureChecklist)
}
