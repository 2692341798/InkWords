package service

import (
	"encoding/json"
	blogcontracts "inkwords-backend/internal/domain/blog/contracts"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSeriesTaskResultCollector_BuildTaskResultIncludesParentAndChapters(t *testing.T) {
	collector := newSeriesTaskResultCollector(
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"Go 源码解析系列",
	)
	collector.SetParentContent("导读正文")
	collector.AddChapterSuccess(blogcontracts.Chapter{
		ID:    "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		Title: "第 1 章",
		Sort:  1,
	}, "正文", 2, []string{"Go"})
	collector.AddChapterFailure(blogcontracts.Chapter{
		ID:    "cccccccc-cccc-cccc-cccc-cccccccccccc",
		Title: "第 2 章",
		Sort:  2,
	}, "章节生成失败")

	resultJSON, err := collector.BuildTaskResult()
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(resultJSON, &decoded))
	require.Equal(t, "generate_series", decoded["task_subtype"])

	payload, ok := decoded["payload"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, payload, "parent_blog")
	require.Contains(t, payload, "chapters")

	chapters, ok := payload["chapters"].([]any)
	require.True(t, ok)
	require.Len(t, chapters, 2)
}

func TestDecompositionService_TakeGenerateSeriesTaskResult_ReturnsStoredResultOnce(t *testing.T) {
	svc := NewDecompositionService(nil)
	parentID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	expected := []byte(`{"task_subtype":"generate_series"}`)

	svc.StoreGenerateSeriesTaskResult(parentID, expected)

	actual, err := svc.TakeGenerateSeriesTaskResult(parentID)
	require.NoError(t, err)
	require.JSONEq(t, string(expected), string(actual))

	_, err = svc.TakeGenerateSeriesTaskResult(parentID)
	require.Error(t, err)
}
