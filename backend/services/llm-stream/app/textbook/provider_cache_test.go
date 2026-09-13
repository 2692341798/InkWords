package textbook

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	sharedgeneration "inkwords-backend/shared/kernel/generation"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func TestCachedSampleGeneratorDoesNotRetainRejectedResponses(t *testing.T) {
	for _, failure := range []string{"quality", "invalid JSON", "wrong model"} {
		t.Run(failure, func(t *testing.T) {
			request := sampleGenerationRequest(t)
			chapter, err := BuildGinRequestLifecycleSample(request.EvidencePack)
			require.NoError(t, err)
			valid, err := json.Marshal(chapter)
			require.NoError(t, err)
			result := sharedgeneration.Result{Provider: "fake-provider", Model: "configured-model", Output: string(valid)}
			switch failure {
			case "quality":
				chapter.Markdown += "\n显然，这段文字不能进入候选稿。"
				invalid, err := json.Marshal(chapter)
				require.NoError(t, err)
				result.Output = string(invalid)
			case "invalid JSON":
				result.Output = "{"
			case "wrong model":
				result.Model = "unexpected-model"
			}
			port := &capturedGenerationPort{result: result}
			generator := NewCachedPortSampleGenerator(port, "fake-provider", "configured-model", defaultSampleGenerationBudget, NewMemoryGenerationResultCache())

			_, err = generator.Generate(context.Background(), request)
			require.Error(t, err)
			require.Equal(t, 1, port.calls, "a rejected response must not trigger an automatic repair call")

			// A separately requested attempt must reach the provider, not replay
			// the response that already failed validation.
			port.result = sharedgeneration.Result{Provider: "fake-provider", Model: "configured-model", Output: string(valid)}
			generation, err := generator.Generate(context.Background(), request)
			require.NoError(t, err)
			require.False(t, generation.CacheHit)
			require.Equal(t, 2, port.calls)
			require.True(t, generation.Quality.Passed)

			cached, err := generator.Generate(context.Background(), request)
			require.NoError(t, err)
			require.True(t, cached.CacheHit)
			require.Zero(t, cached.ProviderCalls)
			require.Equal(t, 2, port.calls)
		})
	}
}

func TestSampleCacheIdentityIncludesGenerationOptionsAndQualityVersion(t *testing.T) {
	request := sampleGenerationRequest(t)
	generator := NewCachedPortSampleGenerator(&capturedGenerationPort{}, "fake-provider", "configured-model", defaultSampleGenerationBudget, NewMemoryGenerationResultCache())
	providerRequest, err := providerRequestForSample("configured-model", request)
	require.NoError(t, err)
	key := generator.cacheKey(request, providerRequest)
	require.Equal(t, sharedtextbook.SampleQualityContractVersion, key.QualityContract)
	original := key.Digest()
	key.QualityContract += ".changed"
	require.NotEqual(t, original, key.Digest())

	for name, change := range map[string]func(*sharedgeneration.Request){
		"output budget":      func(request *sharedgeneration.Request) { request.MaxOutputTokens-- },
		"response schema":    func(request *sharedgeneration.Request) { request.ResponseSchema = json.RawMessage(`{"type":"object"}`) },
		"system instruction": func(request *sharedgeneration.Request) { request.SystemInstruction += "\n新的写作约束。" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := providerRequest
			change(&changed)
			require.NotEqual(t, original, generator.cacheKey(request, changed).Digest())
		})
	}
}

func TestCachedSampleGeneratorBindsChapterBlueprintAndContractRevision(t *testing.T) {
	changes := map[string]func(*SampleGenerationRequest){
		"chapter identity": func(request *SampleGenerationRequest) { request.BlueprintChapter.ID = "another-chapter" },
		"chapter title":    func(request *SampleGenerationRequest) { request.BlueprintChapter.Title = "新的章节目标" },
		"book revision": func(request *SampleGenerationRequest) {
			request.BookContract.RevisionID = "book-contract-2"
			request.BookContract.RevisionNumber++
		},
		"style revision": func(request *SampleGenerationRequest) {
			request.StyleSheet.RevisionID = "style-sheet-2"
			request.StyleSheet.RevisionNumber++
		},
		"blueprint claim": func(request *SampleGenerationRequest) {
			request.BlueprintChapter.CriticalClaims[0].Label = "从请求登记步骤解释 GET 的转发"
		},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			request := sampleGenerationRequest(t)
			chapter, err := BuildGinRequestLifecycleSample(request.EvidencePack)
			require.NoError(t, err)
			original, err := json.Marshal(chapter)
			require.NoError(t, err)
			port := &capturedGenerationPort{result: sharedgeneration.Result{Provider: "fake-provider", Model: "configured-model", Output: string(original)}}
			generator := NewCachedPortSampleGenerator(port, "fake-provider", "configured-model", defaultSampleGenerationBudget, NewMemoryGenerationResultCache())
			_, err = generator.Generate(context.Background(), request)
			require.NoError(t, err)

			change(&request)
			chapter.Markdown += "\n\n本节新增一个独立思考角度：沿登记路径解释处理函数如何保存。"
			updated, err := json.Marshal(chapter)
			require.NoError(t, err)
			port.result.Output = string(updated)
			generation, err := generator.Generate(context.Background(), request)
			require.NoError(t, err)
			require.Equal(t, 2, port.calls, "changed chapter or contract identity must not reuse previous prose")
			require.False(t, generation.CacheHit)
			require.Equal(t, chapter.Markdown, generation.Chapter.Markdown)
		})
	}
}
