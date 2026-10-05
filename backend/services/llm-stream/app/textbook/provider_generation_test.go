package textbook

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sharedgeneration "inkwords-backend/shared/kernel/generation"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type capturedGenerationPort struct {
	request sharedgeneration.Request
	result  sharedgeneration.Result
	calls   int
}

func (port *capturedGenerationPort) Generate(_ context.Context, request sharedgeneration.Request) (sharedgeneration.Result, error) {
	port.request = request
	port.calls++
	return port.result, nil
}

func TestPortSampleGeneratorKeepsEvidenceAsDataAndReturnsOnlyCandidate(t *testing.T) {
	request := sampleGenerationRequest(t)
	chapter, err := BuildGinRequestLifecycleSample(request.EvidencePack)
	require.NoError(t, err)
	encoded, err := json.Marshal(chapter)
	require.NoError(t, err)
	port := &capturedGenerationPort{result: sharedgeneration.Result{Provider: "fake-provider", Model: "configured-model", Output: string(encoded), Usage: sharedgeneration.Usage{Known: true, InputTokens: 21, OutputTokens: 34}}}
	generator := NewPortSampleGenerator(port, "configured-model")

	generation, err := generator.Generate(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, "configured-model", port.request.Model)
	require.Contains(t, string(port.request.ResponseSchema), `"practice_set"`)
	require.Equal(t, sharedtextbook.SampleReservedOutputTokens, port.request.MaxOutputTokens)
	require.Len(t, generation.Chapter.PracticeSet.Tasks, 6)
	require.Contains(t, generation.Chapter.Markdown, sharedtextbook.RenderPracticeSet(generation.Chapter.PracticeSet))
	require.Equal(t, sampleChapterSystemInstruction, port.request.SystemInstruction)
	require.NotContains(t, port.request.SystemInstruction, request.EvidencePack.Excerpts["gin-routergroup-get"])
	require.NotContains(t, port.request.TaskInstruction, request.EvidencePack.Excerpts["gin-routergroup-get"])
	require.Contains(t, port.request.TaskInstruction, `"blueprint_chapter"`)
	require.Contains(t, port.request.TaskInstruction, "RouterGroup.GET 将 GET 方法交给 handle")
	require.Contains(t, port.request.TaskInstruction, `"[evidence:gin-routergroup-get]"`)
	require.Contains(t, port.request.TaskInstruction, `"[evidence:gin-engine-handle-http-request]"`)
	require.Contains(t, port.request.TaskInstruction, "type routeNode struct")
	require.Contains(t, port.request.TaskInstruction, "addRoute 和 findRoute 的参数都必须显式包含 method 与 path")
	require.Contains(t, port.request.TaskInstruction, "禁止将 method 与完整 path 拼接为单个键")
	require.Contains(t, port.request.TaskInstruction, "先按 method 选择根节点，再按路径分段逐层访问 children")
	require.Contains(t, port.request.TaskInstruction, "共享 api 节点")
	require.Contains(t, port.request.TaskInstruction, "同一路径在两个方法下分别登记")
	require.Contains(t, port.request.TaskInstruction, "不等同于 Gin 生产实现的压缩路径树")
	require.Contains(t, port.request.TaskInstruction, "必须先执行 go mod init，再执行 go test ./...")
	require.Contains(t, port.request.TaskInstruction, "每个 import 都必须在对应文件中实际使用")
	require.Contains(t, port.request.TaskInstruction, `"json_output_shape_example"`)
	require.Contains(t, port.request.TaskInstruction, `"final_self_check"`)
	require.Contains(t, port.request.TaskInstruction, "POST（提交方法）")
	require.Contains(t, port.request.TaskInstruction, "main.go 没有调用 strings.* 时绝对不得导入 strings")
	require.Contains(t, port.request.TaskInstruction, "不得 import github.com/gin-gonic/gin")
	require.Contains(t, string(port.request.ResponseSchema), `"learning_arc"`)
	require.Contains(t, string(port.request.ResponseSchema), `"additionalProperties":false`)
	require.Len(t, port.request.Evidence, 6)
	require.Equal(t, "gin-routergroup-get", port.request.Evidence[0].ID)
	require.Contains(t, port.request.Evidence[0].Locator, "(*RouterGroup).GET")
	require.True(t, generation.Quality.Passed, generation.Quality.Failures)
	require.Equal(t, sharedtextbook.RevisionKindCandidate, generation.Candidate.Kind)
	require.Equal(t, "model_candidate", generation.Chapter.GenerationMode)
	require.Equal(t, "unverified", generation.Chapter.RuntimeVerification)
	require.Equal(t, "fake-provider", generation.ProviderName)
	require.Equal(t, "configured-model", generation.ModelName)
	require.True(t, generation.ProviderUsage.Known)
	require.Equal(t, 1, generation.ProviderCalls)
	require.GreaterOrEqual(t, generation.ProviderLatency, time.Duration(0))
	require.Contains(t, port.request.TaskInstruction, "新建”或“打开")
	require.Contains(t, port.request.TaskInstruction, "逐字出现“执行”“预期”“若”")
	require.Equal(t, sharedtextbook.SamplePromptSchemaVersion, generator.cacheKey(request, port.request).PromptSchema)
}

func TestPortSampleGeneratorReturnsUsageWhenHardQualityGateRejectsOutput(t *testing.T) {
	request := sampleGenerationRequest(t)
	chapter, err := BuildGinRequestLifecycleSample(request.EvidencePack)
	require.NoError(t, err)
	chapter.Markdown += "\n显然，这是一个故意不合格的候选稿。"
	encoded, err := json.Marshal(chapter)
	require.NoError(t, err)
	port := &capturedGenerationPort{result: sharedgeneration.Result{Provider: "fake-provider", Model: "configured-model", Output: string(encoded), Usage: sharedgeneration.Usage{Known: true, InputTokens: 101, OutputTokens: 202}}}
	generator := NewPortSampleGenerator(port, "configured-model")

	generation, err := generator.Generate(context.Background(), request)
	require.ErrorContains(t, err, "failed quality gates")
	require.False(t, generation.Quality.Passed)
	require.Contains(t, generation.Quality.Failures, "anti_self_study_phrase: 显然")
	require.Equal(t, "fake-provider", generation.ProviderName)
	require.Equal(t, "configured-model", generation.ModelName)
	require.Equal(t, 101, generation.ProviderUsage.InputTokens)
	require.Equal(t, 202, generation.ProviderUsage.OutputTokens)
	require.Equal(t, 1, generation.ProviderCalls)
	require.NotEmpty(t, generation.PromptHash)
	require.Empty(t, generation.Candidate.ID)
}

func TestProviderSampleTaskInstructionMatchesAudienceLevel(t *testing.T) {
	tests := []struct {
		name      string
		audience  sharedtextbook.AudienceLevel
		expected  string
		forbidden string
	}{
		{name: "foundation", audience: sharedtextbook.AudienceFoundation, expected: "面向零基础读者"},
		{name: "programming", audience: sharedtextbook.AudienceProgramming, expected: "面向已有编程基础但不熟悉当前技术栈的读者", forbidden: "面向零基础读者"},
		{name: "stack familiar", audience: sharedtextbook.AudienceStackFamiliar, expected: "面向熟悉当前技术栈的读者", forbidden: "面向零基础读者"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := sampleGenerationRequest(t)
			request.Audience = test.audience
			request.BookContract.Reader.Audience = test.audience
			chapter, err := BuildGinRequestLifecycleSample(request.EvidencePack)
			require.NoError(t, err)
			encoded, err := json.Marshal(chapter)
			require.NoError(t, err)
			port := &capturedGenerationPort{result: sharedgeneration.Result{Provider: "fake-provider", Model: "configured-model", Output: string(encoded)}}
			generation, err := NewPortSampleGenerator(port, "configured-model").Generate(context.Background(), request)
			require.NoError(t, err)
			require.Equal(t, 1, port.calls)
			require.Equal(t, test.audience, generation.Chapter.Audience)
			require.Equal(t, sharedtextbook.RevisionKindCandidate, generation.Candidate.Kind)
			providerRequest := port.request
			var schema struct {
				Properties struct {
					Audience struct {
						Enum []sharedtextbook.AudienceLevel `json:"enum"`
					} `json:"audience"`
				} `json:"properties"`
			}
			require.NoError(t, json.Unmarshal(providerRequest.ResponseSchema, &schema))
			require.Contains(t, schema.Properties.Audience.Enum, test.audience)
			parts := strings.SplitN(providerRequest.TaskInstruction, "trusted_author_contract:\n", 2)
			require.Len(t, parts, 2)
			var contract struct {
				Audience sharedtextbook.AudienceLevel `json:"audience"`
				Example  struct {
					Audience sharedtextbook.AudienceLevel `json:"audience"`
				} `json:"json_output_shape_example"`
			}
			require.NoError(t, json.Unmarshal([]byte(parts[1]), &contract))
			require.Equal(t, test.audience, contract.Audience)
			require.Equal(t, test.audience, contract.Example.Audience)
			require.Contains(t, providerRequest.TaskInstruction, test.expected)
			if test.forbidden != "" {
				require.NotContains(t, providerRequest.TaskInstruction, test.forbidden)
			}
		})
	}
}

func TestPortSampleGeneratorRejectsOverBudgetEvidenceBeforeCallingProvider(t *testing.T) {
	request := sampleGenerationRequest(t)
	port := &capturedGenerationPort{}
	generator := NewBudgetedPortSampleGenerator(port, "configured-model", TokenBudget{MaxInput: 10, ReservedOutput: 2})

	_, err := generator.Generate(context.Background(), request)
	require.ErrorContains(t, err, "不会静默截断")
	require.Zero(t, port.calls)
}

func TestPortSampleGeneratorRejectsAWorkerTargetMismatchBeforeProviderCall(t *testing.T) {
	request := sampleGenerationRequest(t)
	request.GenerationTarget = sharedtextbook.SampleGenerationTarget{ProviderName: "deepseek", ModelName: "deepseek-v4-flash"}
	port := &capturedGenerationPort{}
	generator := NewCachedPortSampleGenerator(port, "openai", "gpt-test", defaultSampleGenerationBudget, NewMemoryGenerationResultCache())

	_, err := generator.Generate(context.Background(), request)
	require.ErrorContains(t, err, "target does not match configured provider")
	require.Zero(t, port.calls)
}

func TestCachedPortSampleGeneratorReusesOnlyIdenticalEvidenceContract(t *testing.T) {
	request := sampleGenerationRequest(t)
	chapter, err := BuildGinRequestLifecycleSample(request.EvidencePack)
	require.NoError(t, err)
	encoded, err := json.Marshal(chapter)
	require.NoError(t, err)
	request.GenerationTarget.ProviderName = "fixture-provider"
	port := &capturedGenerationPort{result: sharedgeneration.Result{Provider: "fixture-provider", Model: "configured-model", Output: string(encoded)}}
	generator := NewCachedPortSampleGenerator(port, "fixture-provider", "configured-model", defaultSampleGenerationBudget, NewMemoryGenerationResultCache())

	first, err := generator.Generate(context.Background(), request)
	require.NoError(t, err)
	second, err := generator.Generate(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, 1, port.calls)
	require.False(t, first.CacheHit)
	require.True(t, second.CacheHit)
	require.Equal(t, 1, first.ProviderCalls)
	require.Zero(t, second.ProviderCalls)

	changed := request
	changed.EvidencePack.Excerpts = make(map[string]string, len(request.EvidencePack.Excerpts))
	for key, value := range request.EvidencePack.Excerpts {
		changed.EvidencePack.Excerpts[key] = value
	}
	changed.EvidencePack.Excerpts["gin-routergroup-get"] += " // changed"
	chapter, err = BuildGinRequestLifecycleSample(changed.EvidencePack)
	require.NoError(t, err)
	encoded, err = json.Marshal(chapter)
	require.NoError(t, err)
	port.result.Output = string(encoded)
	_, err = generator.Generate(context.Background(), changed)
	require.NoError(t, err)
	require.Equal(t, 2, port.calls, "changed evidence must never reuse cached generated prose")
}

func sampleGenerationRequest(t *testing.T) SampleGenerationRequest {
	t.Helper()
	return SampleGenerationRequest{
		ProjectID:        "project-gin",
		GenerationTarget: sharedtextbook.SampleGenerationTarget{ProviderName: "fake-provider", ModelName: "configured-model"},
		Audience:         sharedtextbook.AudienceFoundation,
		BookContract: sharedtextbook.BookContract{
			RevisionID: "book-contract-1", ProjectID: "project-gin", RevisionNumber: 1, ContentHash: "sha256:contract", Promise: "从真实问题讲透 Gin。",
			Reader:          sharedtextbook.ReaderModel{Audience: sharedtextbook.AudienceFoundation, LearningOutcomes: []string{"解释路由登记"}},
			ChapterProfiles: []sharedtextbook.ChapterProfile{sharedtextbook.ChapterProfileConcept}, TerminologyVersion: "terms-1", PublicationProfile: "personal_learning",
		},
		StyleSheet: sharedtextbook.StyleSheet{
			RevisionID: "style-sheet-1", ProjectID: "project-gin", RevisionNumber: 1, ContentHash: "sha256:style", Language: "zh-CN",
			TerminologyRules: []string{"先白话后术语"}, CodeRules: []string{"标记验证状态"}, CitationRules: []string{"关键事实引用证据"},
		},
		BlueprintChapter: sharedtextbook.BlueprintChapter{ID: ginRequestLifecycleChapterID, Title: "路由登记与请求匹配", Sort: 1, Profile: sharedtextbook.ChapterProfileConcept, EvidenceIDs: []string{"gin-routergroup-get", "gin-routergroup-handle", "gin-engine-add-route", "gin-engine-serve-http", "gin-engine-handle-http-request", "gin-node-get-value"}, CriticalClaims: []sharedtextbook.ClaimRequirement{
			{ID: "gin-get-delegates-to-handle", Label: "RouterGroup.GET 将 GET 方法交给 handle", EvidenceIDs: []string{"gin-routergroup-get"}},
			{ID: "gin-handle-combines-chain", Label: "RouterGroup.handle 会合并处理函数链", EvidenceIDs: []string{"gin-routergroup-handle"}},
			{ID: "gin-engine-adds-route", Label: "Engine.addRoute 将路径加入方法路由树", EvidenceIDs: []string{"gin-engine-add-route"}},
			{ID: "gin-serve-http-dispatches", Label: "Engine.ServeHTTP 将请求交给 handleHTTPRequest", EvidenceIDs: []string{"gin-engine-serve-http"}},
			{ID: "gin-request-selects-route", Label: "handleHTTPRequest 按方法选择树并按路径查找处理函数", EvidenceIDs: []string{"gin-engine-handle-http-request", "gin-node-get-value"}},
		}},
		EvidencePack: ginEvidencePack(t),
	}
}
