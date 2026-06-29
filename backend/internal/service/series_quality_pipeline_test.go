package service

import (
	"context"
	"encoding/json"
	"fmt"
	blogcontracts "inkwords-backend/internal/domain/blog/contracts"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
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

func TestBuildSeriesSharedPromptPrefix_StableAcrossStages(t *testing.T) {
	prefixA := buildSeriesSharedPromptPrefix(
		"Go 源码解析系列",
		"面向小白",
		[]blogcontracts.Chapter{{Sort: 1, Title: "入口"}, {Sort: 2, Title: "调度"}},
	)
	prefixB := buildSeriesSharedPromptPrefix(
		"Go 源码解析系列",
		"面向小白",
		[]blogcontracts.Chapter{{Sort: 1, Title: "入口"}, {Sort: 2, Title: "调度"}},
	)

	require.Equal(t, prefixA, prefixB)
	require.Contains(t, prefixA, "统一质量门禁")
}

func TestParseSeriesChapterUnderstanding_RejectsInvalidJSON(t *testing.T) {
	_, err := parseSeriesChapterUnderstanding(`{"chapter_goal":"解释调度器","must_include_examples":["示例"]}`)

	require.ErrorContains(t, err, "must_explain")
}

type qualityPipelineTestHarness struct {
	service         *DecompositionService
	server          *httptest.Server
	mu              sync.Mutex
	jsonResponses   []string
	textResponses   []string
	streamResponses []string
}

func newQualityPipelineTestService(t *testing.T, jsonResponses []string, textResponses []string, streamResponses []string) *qualityPipelineTestHarness {
	t.Helper()

	harness := &qualityPipelineTestHarness{
		service:         NewDecompositionService(nil),
		jsonResponses:   append([]string(nil), jsonResponses...),
		textResponses:   append([]string(nil), textResponses...),
		streamResponses: append([]string(nil), streamResponses...),
	}

	harness.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var request struct {
			Stream         bool              `json:"stream"`
			ResponseFormat map[string]string `json:"response_format"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))

		w.Header().Set("Content-Type", "application/json")

		harness.mu.Lock()
		defer harness.mu.Unlock()

		if request.Stream {
			require.NotEmpty(t, harness.streamResponses)
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q},\"finish_reason\":null}]}\n\n", harness.streamResponses[0])
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
			fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":1200,\"completion_tokens\":500,\"prompt_cache_hit_tokens\":900,\"prompt_cache_miss_tokens\":300}}\n\n")
			harness.streamResponses = harness.streamResponses[1:]
			return
		}

		if request.ResponseFormat["type"] == "json_object" {
			require.NotEmpty(t, harness.jsonResponses)
			fmt.Fprintf(w, `{"choices":[{"message":{"content":%q}}]}`, harness.jsonResponses[0])
			harness.jsonResponses = harness.jsonResponses[1:]
			return
		}

		require.NotEmpty(t, harness.textResponses)
		fmt.Fprintf(w, `{"choices":[{"message":{"content":%q}}]}`, harness.textResponses[0])
		harness.textResponses = harness.textResponses[1:]
	}))

	harness.service.llmClient.APIURL = harness.server.URL
	harness.service.llmClient.Client = harness.server.Client()

	t.Cleanup(func() {
		harness.server.Close()
	})

	return harness
}

func TestRunSeriesChapterQualityPipeline_FailsWhenReviewHasNoRevisionActions(t *testing.T) {
	harness := newQualityPipelineTestService(
		t,
		[]string{
			`{"chapter_goal":"解释请求流转","reader_questions":["请求如何进入 handler"],"must_explain":["路由树匹配"],"must_include_examples":["curl 请求"],"avoid_overlap":[],"bridge_context":{"from_previous":"上一章介绍启动","to_next":"下一章介绍中间件"}}`,
			`{"draft_markdown":"## Gin 路由\n\n内容正文","coverage_check":{"goal_covered":true,"mechanism_explained":true,"examples_present":true,"repro_present":true,"edge_cases_present":true},"example_inventory":[{"example_type":"code","supports_claim":"说明路由注册"}]}`,
			`{"depth_issues":["缺少中间件短路"],"example_issues":["没有 curl"],"structure_issues":[],"revision_actions":[],"scorecard":{"depth":3,"examples":2,"reproducibility":3,"clarity":4}}`,
			`{"depth_issues":["缺少中间件短路"],"example_issues":["没有 curl"],"structure_issues":[],"revision_actions":[],"scorecard":{"depth":3,"examples":2,"reproducibility":3,"clarity":4}}`,
		},
		nil,
		nil,
	)
	progressChan := make(chan string, 8)

	_, err := harness.service.runSeriesChapterQualityPipeline(context.Background(), seriesQualityPipelineInput{
		SeriesTitle:          "Gin 原理系列",
		ReaderProfile:        "零基础读者",
		Outline:              []blogcontracts.Chapter{{Sort: 1, Title: "Gin 路由", Summary: "请求如何到达 handler"}},
		Chapter:              blogcontracts.Chapter{Sort: 1, Title: "Gin 路由", Summary: "请求如何到达 handler"},
		ChapterSourceContent: "router.GET(\"/ping\", handler)",
		ProgressChan:         progressChan,
	})

	require.ErrorContains(t, err, "revision_actions")
}

func TestRunSeriesChapterQualityPipeline_StreamsOnlyFinalStageAndPreservesStageOrder(t *testing.T) {
	harness := newQualityPipelineTestService(
		t,
		[]string{
			`{"chapter_goal":"解释请求流转","reader_questions":["请求如何进入 handler"],"must_explain":["路由树匹配"],"must_include_examples":["curl 请求"],"avoid_overlap":[],"bridge_context":{"from_previous":"上一章介绍启动","to_next":"下一章介绍中间件"}}`,
			`{"draft_markdown":"## Gin 路由\n\n这是草稿正文","coverage_check":{"goal_covered":true,"mechanism_explained":true,"examples_present":true,"repro_present":true,"edge_cases_present":true},"example_inventory":[{"example_type":"code","supports_claim":"说明路由注册"}]}`,
			`{"depth_issues":["缺少中间件短路"],"example_issues":["curl 示例需要补足"],"structure_issues":[],"revision_actions":["补充中间件短路说明","补充 curl 复现"],"scorecard":{"depth":4,"examples":4,"reproducibility":4,"clarity":4}}`,
		},
		nil,
		[]string{"终稿第一段终稿第二段"},
	)
	progressChan := make(chan string, 16)

	result, err := harness.service.runSeriesChapterQualityPipeline(context.Background(), seriesQualityPipelineInput{
		SeriesTitle:          "Gin 原理系列",
		ReaderProfile:        "零基础读者",
		Outline:              []blogcontracts.Chapter{{Sort: 1, Title: "Gin 路由", Summary: "请求如何到达 handler"}},
		Chapter:              blogcontracts.Chapter{Sort: 1, Title: "Gin 路由", Summary: "请求如何到达 handler"},
		ChapterSourceContent: "router.GET(\"/ping\", handler)",
		ProgressChan:         progressChan,
	})

	require.NoError(t, err)
	require.Equal(t, "终稿第一段终稿第二段", result.FinalMarkdown)
	require.Equal(t, []string{"补充中间件短路说明", "补充 curl 复现"}, result.ResolvedIssues)

	close(progressChan)

	var statuses []string
	var streamedContent []string
	var usagePayload map[string]any
	for raw := range progressChan {
		var payload map[string]any
		require.NoError(t, json.Unmarshal([]byte(raw), &payload))
		status := payload["status"].(string)
		statuses = append(statuses, status)
		if status == "streaming" {
			streamedContent = append(streamedContent, payload["content"].(string))
		}
		if status == "usage" {
			usagePayload = payload
		}
	}

	require.True(t, slices.Equal([]string{"understanding", "drafting", "reviewing", "revising", "streaming", "usage"}, statuses))
	require.Equal(t, []string{"终稿第一段终稿第二段"}, streamedContent)
	require.NotNil(t, usagePayload)
	require.Equal(t, float64(1200), usagePayload["prompt_tokens"])
	require.Equal(t, float64(500), usagePayload["completion_tokens"])
	require.Equal(t, float64(900), usagePayload["prompt_cache_hit_tokens"])
	require.Equal(t, float64(300), usagePayload["prompt_cache_miss_tokens"])
}

func TestRunSeriesChapterQualityPipeline_RepairsDraftWhenScorecardIsLow(t *testing.T) {
	harness := newQualityPipelineTestService(
		t,
		[]string{
			`{"chapter_goal":"解释请求流转","reader_questions":["请求如何进入 handler"],"must_explain":["路由树匹配"],"must_include_examples":["curl 请求"],"avoid_overlap":[],"bridge_context":{"from_previous":"","to_next":""}}`,
			`{"draft_markdown":"## Gin 路由\n\n草稿缺少复现细节","coverage_check":{"goal_covered":true,"mechanism_explained":true,"examples_present":true,"repro_present":true,"edge_cases_present":true},"example_inventory":[{"example_type":"code","supports_claim":"说明路由注册"}]}`,
			`{"depth_issues":["机制不够细"],"example_issues":["curl 不完整"],"structure_issues":[],"revision_actions":["补齐复现步骤"],"scorecard":{"depth":3,"examples":3,"reproducibility":3,"clarity":4}}`,
			`{"draft_markdown":"## Gin 路由\n\n修复后包含 curl 复现步骤","coverage_check":{"goal_covered":true,"mechanism_explained":true,"examples_present":true,"repro_present":true,"edge_cases_present":true},"example_inventory":[{"example_type":"command","supports_claim":"补齐 curl 复现"}]}`,
		},
		nil,
		[]string{"终稿"},
	)
	progressChan := make(chan string, 16)

	result, err := harness.service.runSeriesChapterQualityPipeline(context.Background(), seriesQualityPipelineInput{
		SeriesTitle:          "Gin 原理系列",
		ReaderProfile:        "零基础读者",
		Outline:              []blogcontracts.Chapter{{Sort: 1, Title: "Gin 路由", Summary: "请求如何到达 handler"}},
		Chapter:              blogcontracts.Chapter{Sort: 1, Title: "Gin 路由", Summary: "请求如何到达 handler"},
		ChapterSourceContent: "router.GET(\"/ping\", handler)",
		ProgressChan:         progressChan,
	})

	require.NoError(t, err)
	require.Equal(t, "终稿", result.FinalMarkdown)
	require.Equal(t, []string{"补齐复现步骤"}, result.RevisionActions)
	require.Equal(t, 3, result.QualityScorecard.Depth)

	close(progressChan)
	var statuses []string
	for raw := range progressChan {
		var payload map[string]any
		require.NoError(t, json.Unmarshal([]byte(raw), &payload))
		statuses = append(statuses, payload["status"].(string))
	}
	require.Contains(t, statuses, "repairing")
}
