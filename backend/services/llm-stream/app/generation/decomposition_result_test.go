package generation

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	sharedblog "inkwords-backend/shared/kernel/blog"
	llm "inkwords-backend/shared/platform/llm"
)

func TestSeriesTaskResultCollectorBuildTaskResultIncludesParentChaptersAndUsage(t *testing.T) {
	collector := newSeriesTaskResultCollector("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Go 源码解析系列")
	collector.SetParentContent("导读正文")
	collector.AddChapterSuccessWithQuality(sharedblog.Chapter{ID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", Title: "第 1 章", Sort: 1}, "正文", 2, []string{"Go"}, seriesChapterFinal{
		Usage: seriesChapterUsage{PromptTokens: 11, CompletionTokens: 7, PromptCacheHitTokens: 5, PromptCacheMissTokens: 6},
	})
	collector.AddChapterFailure(sharedblog.Chapter{ID: "cccccccc-cccc-cccc-cccc-cccccccccccc", Title: "第 2 章", Sort: 2}, "章节生成失败")

	resultJSON, err := collector.BuildTaskResult()
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(resultJSON, &decoded))
	require.Equal(t, "generate_series", decoded["task_subtype"])
	usage := decoded["usage"].(map[string]any)
	require.Equal(t, float64(11), usage["prompt_tokens"])
	require.Equal(t, float64(5), usage["prompt_cache_hit_tokens"])
	payload := decoded["payload"].(map[string]any)
	require.Contains(t, payload, "parent_blog")
	require.Len(t, payload["chapters"], 2)
}

func TestDecompositionServiceTakeGenerateSeriesTaskResultReturnsStoredResultOnce(t *testing.T) {
	svc := NewDecompositionService(nil, nil, nil)
	parentID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	expected := []byte(`{"task_subtype":"generate_series"}`)
	svc.StoreGenerateSeriesTaskResult(parentID, expected)
	actual, err := svc.TakeGenerateSeriesTaskResult(parentID)
	require.NoError(t, err)
	require.JSONEq(t, string(expected), string(actual))
	_, err = svc.TakeGenerateSeriesTaskResult(parentID)
	require.Error(t, err)
}

func TestUsageCollectorsReturnStoredUsageOnce(t *testing.T) {
	svc := NewGeneratorService(nil, nil)
	usage := llm.CompletionUsage{PromptTokens: 3, CompletionTokens: 2, PromptCacheHitTokens: 1, PromptCacheMissTokens: 2}
	svc.storeGeneratedUsage("file", "content", usage)
	require.Equal(t, usage, svc.takeGeneratedUsage("file", "content"))
	require.Zero(t, svc.takeGeneratedUsage("file", "content").PromptTokens)

	decomposition := NewDecompositionService(nil, nil, nil)
	blogID := uuid.New()
	decomposition.storeContinueUsage(blogID, "append", usage)
	require.Equal(t, usage, decomposition.takeContinueUsage(blogID, "append"))
	require.Zero(t, decomposition.takeContinueUsage(blogID, "append").PromptTokens)
}
