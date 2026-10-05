package textbook

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sharedgeneration "inkwords-backend/shared/kernel/generation"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	platformllm "inkwords-backend/shared/platform/llm"
)

const realTextbookGenerationOptIn = "approved"

// TestRealTextbookGenerationProducesSelectedAudienceCandidates is never enabled
// by a normal test run. A protected real-acceptance dispatch must opt in to the
// selected bounded calls and supply an artifact directory before any call starts.
func TestRealTextbookGenerationProducesSelectedAudienceCandidates(t *testing.T) {
	if strings.TrimSpace(os.Getenv("INKWORDS_REAL_TEXTBOOK_GENERATION_ACCEPTANCE")) != realTextbookGenerationOptIn {
		t.Skip("real textbook generation acceptance requires protected-environment opt-in")
	}
	apiKey := strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))
	model := realGenerationSetting(os.Getenv("TEXTBOOK_STANDARD_MODEL"), os.Getenv("DEEPSEEK_MODEL"))
	outputDir := strings.TrimSpace(os.Getenv("INKWORDS_REAL_TEXTBOOK_OUTPUT_DIR"))
	require.NotEmpty(t, apiKey, "real textbook acceptance requires a DeepSeek API key")
	require.NotEmpty(t, model, "real textbook acceptance requires an explicit model")
	require.NotEmpty(t, outputDir, "real textbook acceptance requires an explicit artifact directory")
	require.NoError(t, os.MkdirAll(outputDir, 0o700))
	require.NoError(t, os.Chmod(outputDir, 0o700))
	existing, err := os.ReadDir(outputDir)
	require.NoError(t, err)
	require.Empty(t, existing, "使用新的空目录，禁止覆盖既有真实验收结果")

	adapter := platformllm.NewDeepSeekGenerationAdapterWithTimeout(
		platformllm.NewDeepSeekClient(apiKey),
		15*time.Minute,
	)
	generator := NewCachedPortSampleGenerator(
		adapter,
		"deepseek",
		model,
		DefaultSampleGenerationBudget(),
		NewMemoryGenerationResultCache(),
	)
	audiences, err := realGenerationAudiences(os.Getenv("INKWORDS_REAL_TEXTBOOK_AUDIENCES"))
	require.NoError(t, err)
	requests := prepareRealGenerationRequests(t, os.Getenv("INKWORDS_REAL_TEXTBOOK_REQUEST_DIR"), model, audiences)
	contentHashes := map[string]bool{}
	promptHashes := map[string]bool{}
	for _, request := range requests {
		audience := request.Audience

		callCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		generation, err := generator.Generate(callCtx, request)
		cancel()
		require.NoError(t, writeRealGenerationAttempt(outputDir, request, generation, err != nil))
		require.NoError(t, writeRealGenerationArtifacts(outputDir, audience, generation))
		require.NoError(t, err, "audience %s failed; later audience calls were not started", audience)
		require.True(t, generation.Quality.Passed)
		require.True(t, generation.Quality.ManualReviewRequired)
		require.Equal(t, audience, generation.Chapter.Audience)
		require.Equal(t, sharedtextbook.RevisionKindCandidate, generation.Candidate.Kind)
		require.Equal(t, "unverified", generation.Chapter.RuntimeVerification)
		require.Equal(t, 1, generation.ProviderCalls)
		require.True(t, generation.ProviderUsage.Known)
		require.Positive(t, generation.ProviderUsage.InputTokens)
		require.Positive(t, generation.ProviderUsage.OutputTokens)
		require.False(t, contentHashes[generation.Chapter.ContentHash], "audience manuscripts must differ")
		require.False(t, promptHashes[generation.PromptHash], "audience Provider requests must differ")
		contentHashes[generation.Chapter.ContentHash] = true
		promptHashes[generation.PromptHash] = true

		t.Logf(
			"audience=%s provider=%s model=%s content_hash=%s prompt_hash=%s input_tokens=%d output_tokens=%d cached_tokens=%d latency_ms=%d",
			audience,
			generation.ProviderName,
			generation.ModelName,
			generation.Chapter.ContentHash,
			generation.PromptHash,
			generation.ProviderUsage.InputTokens,
			generation.ProviderUsage.OutputTokens,
			generation.ProviderUsage.CachedTokens,
			generation.ProviderLatency.Milliseconds(),
		)
	}
}

func TestRealTextbookGenerationAcceptanceInputsAreDistinctWithoutCallingProvider(t *testing.T) {
	audiences := []sharedtextbook.AudienceLevel{
		sharedtextbook.AudienceFoundation,
		sharedtextbook.AudienceProgramming,
		sharedtextbook.AudienceStackFamiliar,
	}
	identities := map[string]bool{}
	promptHashes := map[string]bool{}
	for _, audience := range audiences {
		request := sampleGenerationRequest(t)
		request.GenerationTarget = sharedtextbook.SampleGenerationTarget{ProviderName: "deepseek", ModelName: "deepseek-v4-flash"}
		request.Audience = audience
		request.BookContract.Reader.Audience = audience
		require.NoError(t, request.Validate())
		encoded, err := json.Marshal(request)
		require.NoError(t, err)
		identity := digest(string(encoded))
		require.False(t, identities[identity], "each audience must freeze a distinct request")
		identities[identity] = true
		providerRequest, err := providerRequestForSample(request.GenerationTarget.ModelName, request)
		require.NoError(t, err)
		promptHash := digest(providerRequestHashInput(providerRequest))
		require.False(t, promptHashes[promptHash], "each audience must produce a distinct Provider request")
		promptHashes[promptHash] = true
		budget, err := sharedgeneration.CheckRequestBudget(DefaultSampleGenerationBudget(), providerRequest)
		require.NoError(t, err)
		require.False(t, budget.RequiresCompression)
		t.Logf("audience=%s prompt_hash=%s estimated_input_tokens=%d allowed_input_tokens=%d reserved_output_tokens=%d", audience, promptHash, budget.EstimatedInput, budget.AllowedInput, sharedtextbook.SampleReservedOutputTokens)
	}
}

func TestRealGenerationAudiencesRejectsUnknownOrDuplicateSelection(t *testing.T) {
	selected, err := realGenerationAudiences("programming,stack_familiar")
	require.NoError(t, err)
	require.Equal(t, []sharedtextbook.AudienceLevel{sharedtextbook.AudienceProgramming, sharedtextbook.AudienceStackFamiliar}, selected)
	_, err = realGenerationAudiences("foundation,foundation")
	require.Error(t, err)
	_, err = realGenerationAudiences("unknown")
	require.Error(t, err)
}

func TestWriteRealGenerationArtifactsCreatesPrivateReviewableFiles(t *testing.T) {
	generation := SampleGeneration{
		Chapter: SampleChapter{
			Audience: sharedtextbook.AudienceFoundation,
			Markdown: "# Gin 请求路由\n\n候选样章正文。\n",
		},
		ProviderName: "deepseek",
		ModelName:    "deepseek-v4-flash",
		PromptHash:   "sha256:test-prompt",
	}

	outputDir := t.TempDir()
	require.NoError(t, writeRealGenerationArtifacts(outputDir, sharedtextbook.AudienceFoundation, generation))

	jsonPath := filepath.Join(outputDir, "foundation.json")
	markdownPath := filepath.Join(outputDir, "foundation.md")
	encoded, err := os.ReadFile(jsonPath)
	require.NoError(t, err)
	var decoded SampleGeneration
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	require.Equal(t, generation.Chapter.Audience, decoded.Chapter.Audience)
	require.Equal(t, generation.ProviderName, decoded.ProviderName)
	require.Equal(t, generation.PromptHash, decoded.PromptHash)

	markdown, err := os.ReadFile(markdownPath)
	require.NoError(t, err)
	require.Equal(t, generation.Chapter.Markdown, string(markdown))
	for _, path := range []string{jsonPath, markdownPath} {
		info, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func writeRealGenerationArtifacts(outputDir string, audience sharedtextbook.AudienceLevel, generation SampleGeneration) error {
	encoded, err := json.MarshalIndent(generation, "", "  ")
	if err != nil {
		return err
	}
	base := filepath.Join(outputDir, string(audience))
	if err := os.WriteFile(base+".json", encoded, 0o600); err != nil {
		return err
	}
	return os.WriteFile(base+".md", []byte(generation.Chapter.Markdown), 0o600)
}

func realGenerationSetting(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func realGenerationAudiences(raw string) ([]sharedtextbook.AudienceLevel, error) {
	if strings.TrimSpace(raw) == "" {
		return []sharedtextbook.AudienceLevel{
			sharedtextbook.AudienceFoundation,
			sharedtextbook.AudienceProgramming,
			sharedtextbook.AudienceStackFamiliar,
		}, nil
	}
	allowed := map[sharedtextbook.AudienceLevel]bool{
		sharedtextbook.AudienceFoundation:    true,
		sharedtextbook.AudienceProgramming:   true,
		sharedtextbook.AudienceStackFamiliar: true,
	}
	seen := map[sharedtextbook.AudienceLevel]bool{}
	selected := make([]sharedtextbook.AudienceLevel, 0, 3)
	for _, value := range strings.Split(raw, ",") {
		audience := sharedtextbook.AudienceLevel(strings.TrimSpace(value))
		if !allowed[audience] || seen[audience] {
			return nil, fmt.Errorf("invalid or duplicate real generation audience: %q", audience)
		}
		seen[audience] = true
		selected = append(selected, audience)
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("real generation audience selection is empty")
	}
	return selected, nil
}
