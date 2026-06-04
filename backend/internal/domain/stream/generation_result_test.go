package stream

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildGenerateSingleTaskResult_ProducesTaskOnlyContract(t *testing.T) {
	result, err := BuildGenerateSingleTaskResult(GenerateSingleTaskResultInput{
		BlogID:          "11111111-1111-1111-1111-111111111111",
		Title:           "文件解析生成的博客",
		Content:         "# 标题\n\n正文",
		SourceType:      "file",
		WordCount:       7,
		TechStacks:      []string{"Go", "Docker"},
		EstimatedTokens: 14,
	})
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(result, &decoded))
	require.Equal(t, float64(1), decoded["result_version"])
	require.Equal(t, "generation", decoded["task_type"])
	require.Equal(t, "generate_single", decoded["task_subtype"])
	require.Equal(t, "task_only", decoded["persistence_mode"])
	require.Equal(t, "succeeded", decoded["final_status"])
}

func TestBuildContinueTaskResult_ProducesFinalContent(t *testing.T) {
	result, err := BuildContinueTaskResult(ContinueTaskResultInput{
		BlogID:          "11111111-1111-1111-1111-111111111111",
		AppendedContent: "追加内容",
		FinalContent:    "旧内容追加内容",
		EstimatedTokens: 8,
	})
	require.NoError(t, err)
	require.Contains(t, string(result), `"task_subtype":"continue"`)
	require.Contains(t, string(result), `"final_content":"旧内容追加内容"`)
}
