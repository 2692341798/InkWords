package generation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	sharedblog "inkwords-backend/shared/kernel/blog"
)

func TestQualityValidatorsRejectMissingRequiredDetails(t *testing.T) {
	require.ErrorContains(t, validateSeriesChapterUnderstanding(seriesChapterUnderstanding{ChapterGoal: "目标"}), "must_explain")
	require.ErrorContains(t, validateSeriesChapterDraft(seriesChapterDraft{
		DraftMarkdown: "正文", CoverageCheck: seriesChapterCoverageCheck{MechanismExplained: true},
	}), "examples_present")
	require.ErrorContains(t, validateSeriesChapterReview(seriesChapterReview{}), "revision_actions")
}

func TestBuildSeriesSharedPromptPrefixStableAcrossStages(t *testing.T) {
	outline := []sharedblog.Chapter{{Sort: 1, Title: "入口"}, {Sort: 2, Title: "调度"}}
	a := buildSeriesSharedPromptPrefix("Go 源码", "面向小白", outline)
	b := buildSeriesSharedPromptPrefix("Go 源码", "面向小白", outline)
	require.Equal(t, a, b)
	require.Contains(t, a, "统一质量门禁")
}

func TestParseSeriesUnderstandingRejectsInvalidJSON(t *testing.T) {
	_, err := parseSeriesUnderstanding(`{"chapter_goal":"解释调度器","must_include_examples":["示例"]}`)
	require.ErrorContains(t, err, "must_explain")
}

type qualityPipelineHarness struct {
	service         *DecompositionService
	server          *httptest.Server
	mu              sync.Mutex
	jsonResponses   []string
	streamResponses []string
}

func newQualityPipelineHarness(t *testing.T, jsonResponses, streamResponses []string) *qualityPipelineHarness {
	t.Helper()
	h := &qualityPipelineHarness{
		service: NewDecompositionService(nil, nil, nil), jsonResponses: append([]string(nil), jsonResponses...),
		streamResponses: append([]string(nil), streamResponses...),
	}
	h.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()
		var request struct {
			Stream         bool              `json:"stream"`
			ResponseFormat map[string]string `json:"response_format"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		h.mu.Lock()
		defer h.mu.Unlock()
		if request.Stream {
			require.NotEmpty(t, h.streamResponses)
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q},\"finish_reason\":null}]}\n\n", h.streamResponses[0])
			_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
			_, _ = fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":1200,\"completion_tokens\":500,\"prompt_cache_hit_tokens\":900,\"prompt_cache_miss_tokens\":300}}\n\n")
			h.streamResponses = h.streamResponses[1:]
			return
		}
		require.Equal(t, "json_object", request.ResponseFormat["type"])
		require.NotEmpty(t, h.jsonResponses)
		_, _ = fmt.Fprintf(w, `{"choices":[{"message":{"content":%q}}]}`, h.jsonResponses[0])
		h.jsonResponses = h.jsonResponses[1:]
	}))
	h.service.llmClient.APIURL = h.server.URL
	h.service.llmClient.Client = h.server.Client()
	t.Cleanup(h.server.Close)
	return h
}

func qualityInput(progress chan<- string) seriesQualityPipelineInput {
	chapter := sharedblog.Chapter{Sort: 1, Title: "Gin 路由", Summary: "请求如何到达 handler"}
	return seriesQualityPipelineInput{
		SeriesTitle: "Gin 原理系列", ReaderProfile: "零基础读者", Outline: []sharedblog.Chapter{chapter},
		Chapter: chapter, ChapterSourceContent: `router.GET("/ping", handler)`, ProgressChan: progress,
	}
}

func TestQualityPipelineStreamsOnlyFinalStageInOrder(t *testing.T) {
	h := newQualityPipelineHarness(t, []string{
		`{"chapter_goal":"解释请求流转","reader_questions":["如何进入 handler"],"must_explain":["路由树匹配"],"must_include_examples":["curl"],"avoid_overlap":[],"bridge_context":{}}`,
		`{"draft_markdown":"## Gin 路由","coverage_check":{"goal_covered":true,"mechanism_explained":true,"examples_present":true,"repro_present":true,"edge_cases_present":true},"example_inventory":[{"example_type":"code","supports_claim":"路由注册"}]}`,
		`{"depth_issues":[],"example_issues":[],"structure_issues":[],"revision_actions":["补充 curl"],"scorecard":{"depth":4,"examples":4,"reproducibility":4,"clarity":4}}`,
	}, []string{"最终正文"})
	progress := make(chan string, 16)
	result, err := h.service.runSeriesChapterQualityPipeline(context.Background(), qualityInput(progress))
	require.NoError(t, err)
	require.Equal(t, "最终正文", result.FinalMarkdown)
	require.Equal(t, 1200, result.Usage.PromptTokens)
	close(progress)
	var statuses []string
	for raw := range progress {
		var payload map[string]any
		require.NoError(t, json.Unmarshal([]byte(raw), &payload))
		statuses = append(statuses, payload["status"].(string))
	}
	require.True(t, slices.Equal([]string{"understanding", "drafting", "reviewing", "revising", "streaming", "usage"}, statuses))
}

func TestQualityPipelineRepairsLowScorecardDraft(t *testing.T) {
	h := newQualityPipelineHarness(t, []string{
		`{"chapter_goal":"解释请求流转","must_explain":["路由树"],"must_include_examples":["curl"],"bridge_context":{}}`,
		`{"draft_markdown":"草稿","coverage_check":{"mechanism_explained":true,"examples_present":true,"repro_present":true},"example_inventory":[{"example_type":"code","supports_claim":"路由"}]}`,
		`{"revision_actions":["补齐复现"],"scorecard":{"depth":3,"examples":3,"reproducibility":3,"clarity":4}}`,
		`{"draft_markdown":"修复草稿","coverage_check":{"mechanism_explained":true,"examples_present":true,"repro_present":true},"example_inventory":[{"example_type":"command","supports_claim":"复现"}]}`,
	}, []string{"终稿"})
	progress := make(chan string, 16)
	result, err := h.service.runSeriesChapterQualityPipeline(context.Background(), qualityInput(progress))
	require.NoError(t, err)
	require.Equal(t, []string{"补齐复现"}, result.RevisionActions)
	close(progress)
	var statuses []string
	for raw := range progress {
		var payload map[string]any
		require.NoError(t, json.Unmarshal([]byte(raw), &payload))
		statuses = append(statuses, payload["status"].(string))
	}
	require.Contains(t, statuses, "repairing")
}

func TestQualityPipelineReturnsErrorWhenReviewRepairStillHasNoActions(t *testing.T) {
	h := newQualityPipelineHarness(t, []string{
		`{"chapter_goal":"解释请求流转","must_explain":["路由树"],"must_include_examples":["curl"],"bridge_context":{}}`,
		`{"draft_markdown":"草稿","coverage_check":{"mechanism_explained":true,"examples_present":true,"repro_present":true},"example_inventory":[{"example_type":"code","supports_claim":"路由"}]}`,
		`{"revision_actions":[],"scorecard":{"depth":3,"examples":3,"reproducibility":3,"clarity":4}}`,
		`{"revision_actions":[],"scorecard":{"depth":3,"examples":3,"reproducibility":3,"clarity":4}}`,
	}, nil)
	progress := make(chan string, 16)
	_, err := h.service.runSeriesChapterQualityPipeline(context.Background(), qualityInput(progress))
	require.ErrorContains(t, err, "revision_actions")
}
