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

func TestSeriesUnderstandingContractMatchesValidationRequirements(t *testing.T) {
	require.Contains(t, seriesUnderstandingJSONContract, `"must_explain"`)
	require.Contains(t, seriesUnderstandingJSONContract, `"must_include_examples"`)
	require.Contains(t, seriesUnderstandingJSONContract, "至少一项")
	require.Equal(t, seriesUnderstandingJSONContract, seriesJSONContractForStage("章节理解"))
	require.Contains(t, seriesJSONContractForStage("章节草稿"), `"mechanism_explained": true`)
	require.Contains(t, seriesJSONContractForStage("章节审稿"), `"revision_actions"`)
	require.Empty(t, seriesJSONContractForStage("终稿"))
}

func TestGenerateSeriesChapterUnderstandingRepairsMissingMustExplain(t *testing.T) {
	h := newQualityPipelineHarness(t, []string{
		`{"chapter_goal":"理解 KNN 分类","reader_questions":["如何预测"],"must_include_examples":["k=3 与 k=5"],"avoid_overlap":[],"bridge_context":{}}`,
		`{"chapter_goal":"理解 KNN 分类","reader_questions":["如何预测"],"must_explain":["多数投票与距离度量"],"must_include_examples":["k=3 与 k=5"],"avoid_overlap":[],"bridge_context":{"from_previous":"","to_next":""}}`,
	}, nil)

	chapter := sharedblog.Chapter{Sort: 2, Title: "KNN 分类基础", Summary: "解释邻居投票"}
	result, _, err := h.service.generateSeriesChapterUnderstanding(
		context.Background(),
		"deepseek-v4-flash",
		buildSeriesSharedPromptPrefix("KNN", "初学者", []sharedblog.Chapter{chapter}),
		chapter,
		"k=3 和 k=5 可能产生不同分类结果",
		"series-test",
	)

	require.NoError(t, err)
	require.Equal(t, []string{"多数投票与距离度量"}, result.MustExplain)
}

func TestGenerateSeriesChapterDraftRepairsFalseCoverageGate(t *testing.T) {
	h := newQualityPipelineHarness(t, []string{
		`{"draft_markdown":"只给结论","coverage_check":{"mechanism_explained":false,"examples_present":true,"repro_present":true},"example_inventory":[{"example_type":"calculation","supports_claim":"k值影响"}]}`,
		`{"draft_markdown":"解释距离度量和多数投票，并给出k=3与k=5的计算步骤。","coverage_check":{"goal_covered":true,"mechanism_explained":true,"examples_present":true,"repro_present":true,"edge_cases_present":true},"example_inventory":[{"example_type":"calculation","supports_claim":"k值影响"}]}`,
	}, nil)
	input := qualityInput(make(chan string, 1))
	understanding := seriesChapterUnderstanding{
		ChapterGoal:         "理解 KNN",
		MustExplain:         []string{"距离度量与多数投票"},
		MustIncludeExamples: []string{"k=3 与 k=5"},
	}

	result, _, err := h.service.generateSeriesChapterDraft(
		context.Background(), "deepseek-v4-flash", "系列契约", input, understanding, "series-test",
	)

	require.NoError(t, err)
	require.True(t, result.CoverageCheck.MechanismExplained)
	require.Contains(t, result.DraftMarkdown, "多数投票")
}

func TestGenerateSeriesChapterDraftSalvagesUnexpectedEOF(t *testing.T) {
	h := newQualityPipelineHarness(t, []string{
		`{"draft_markdown":"## 课程目标与学习成果\n\nKNN 的课程目标包括理解模型原理、掌握 k 值影响，并能独立复现实验步骤。","coverage_check":{"goal_covered":true,"mechanism_explained":true,"examples_present":true,"repro_present":true,"edge_cases_present":true},"example_inventory":[{"example_type":"walkthrough","supports_claim":"课程目标可复现"}`,
		`{"draft_markdown":"## 课程目标与学习成果\n\nKNN 的课程目标包括理解模型原理、掌握 k 值影响，并能独立复现实验步骤。","coverage_check":{"goal_covered":true,"mechanism_explained":true,"examples_present":true,"repro_present":true,"edge_cases_present":true},"example_inventory":[{"example_type":"walkthrough","supports_claim":"课程目标可复现"}`,
	}, nil)
	input := qualityInput(make(chan string, 1))
	understanding := seriesChapterUnderstanding{
		ChapterGoal:         "理解 KNN",
		MustExplain:         []string{"距离度量与多数投票"},
		MustIncludeExamples: []string{"k=3 与 k=5"},
	}

	result, _, err := h.service.generateSeriesChapterDraft(
		context.Background(), "deepseek-v4-flash", "系列契约", input, understanding, "series-test",
	)

	require.NoError(t, err)
	require.Contains(t, result.DraftMarkdown, "课程目标")
	require.True(t, result.CoverageCheck.MechanismExplained)
	require.Len(t, result.ExampleInventory, 1)
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
