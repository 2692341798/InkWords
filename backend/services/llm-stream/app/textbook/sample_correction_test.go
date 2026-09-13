package textbook

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func rejectedCorrectionFixture(t *testing.T) ([]byte, SampleCorrectionInput) {
	t.Helper()
	request := sampleGenerationRequest(t)
	chapter, err := BuildGinRequestLifecycleSample(request.EvidencePack)
	require.NoError(t, err)
	chapter.Markdown = strings.ReplaceAll(chapter.Markdown, "**验证状态：未验证。**", "")
	chapter.ContentHash = digest(chapter.Markdown)
	draft := RejectedDraft{Format: "inkwords.rejected-sample.v1", PromptSchema: sharedtextbook.SamplePromptSchemaVersion, Request: request, Generation: SampleGeneration{Chapter: chapter, ProviderName: request.GenerationTarget.ProviderName, ModelName: request.GenerationTarget.ModelName, PromptHash: "sha256:original-prompt", ProviderCalls: 1, Quality: RunSampleChapterQualityGates(chapter, request.EvidencePack)}}
	require.NoError(t, draft.Validate())
	raw, err := json.Marshal(draft)
	require.NoError(t, err)
	body := strings.TrimSpace(strings.TrimSuffix(chapter.Markdown, sharedtextbook.RenderPracticeSet(chapter.PracticeSet)))
	input := SampleCorrectionInput{Format: SampleCorrectionFormat, OriginalReceiptHash: strings.TrimPrefix(digest(string(raw)), "sha256:"), OriginalContentHash: chapter.ContentHash, Reason: "同步修正教学正文和结构化练习，保留原始来源。", MarkdownBody: body + "\n\n**验证状态：未验证。**", Scenario: chapter.Scenario, Understanding: chapter.Understanding, LearningArc: chapter.LearningArc, PracticeSet: chapter.PracticeSet}
	return raw, input
}

func TestStructuredCorrectionPreservesReceiptAndRendersUpdatedPractice(t *testing.T) {
	raw, input := rejectedCorrectionFixture(t)
	before := string(raw)
	input.PracticeSet.Tasks[0].ExpectedAnswer += " 本答案经局部修订，仍需用冻结证据核对。"
	proposal, err := PrepareSampleCorrection(raw, input)
	require.NoError(t, err)
	require.True(t, proposal.Quality.Passed, proposal.Quality.Failures)
	require.Equal(t, "automated_local_correction", proposal.Origin)
	require.Equal(t, proposal.Origin, proposal.Chapter.GenerationMode)
	require.Equal(t, "unverified", proposal.Chapter.RuntimeVerification)
	require.Zero(t, proposal.ProviderCalls)
	require.Equal(t, 1, proposal.OriginalUsage.ProviderCallCount)
	require.False(t, proposal.CandidatePersisted)
	require.Contains(t, proposal.Chapter.Markdown, sharedtextbook.RenderPracticeSet(input.PracticeSet))
	require.Equal(t, 1, strings.Count(proposal.Chapter.Markdown, "<!-- inkwords.practice-set.v1:"))
	require.Equal(t, before, string(raw))
	require.NotEqual(t, input.OriginalContentHash, proposal.Chapter.ContentHash)
	input.PracticeSet.Tasks[0].ExpectedAnswer = "later mutation"
	require.NotEqual(t, input.PracticeSet.Tasks[0].ExpectedAnswer, proposal.Chapter.PracticeSet.Tasks[0].ExpectedAnswer)
}

func TestStructuredCorrectionRejectsStaleIdentityAndUnfrozenEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*SampleCorrectionInput)
	}{
		{"receipt", func(i *SampleCorrectionInput) { i.OriginalReceiptHash = strings.Repeat("0", 64) }},
		{"content", func(i *SampleCorrectionInput) { i.OriginalContentHash = "sha256:stale" }},
		{"practice identity", func(i *SampleCorrectionInput) { i.PracticeSet.Tasks[0].ID = "new-task" }},
		{"duplicate answers", func(i *SampleCorrectionInput) { i.MarkdownBody += sharedtextbook.RenderPracticeSet(i.PracticeSet) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, input := rejectedCorrectionFixture(t)
			tc.change(&input)
			_, err := PrepareSampleCorrection(raw, input)
			require.Error(t, err)
		})
	}
	raw, input := rejectedCorrectionFixture(t)
	input.PracticeSet.Tasks[0].EvidenceIDs = []string{"outside-frozen-evidence"}
	proposal, err := PrepareSampleCorrection(raw, input)
	require.NoError(t, err)
	require.False(t, proposal.Quality.Passed)
	require.False(t, proposal.CandidatePersisted)
}

func TestStructuredCorrectionDecoderAndRuntimeGatesFailClosed(t *testing.T) {
	for _, data := range []string{`{"chapter_id":"override"}`, `{} {}`, "null trailing", strings.Repeat("x", (1<<20)+1)} {
		_, err := DecodeSampleCorrection([]byte(data))
		require.Error(t, err)
	}
	raw, input := rejectedCorrectionFixture(t)
	input.MarkdownBody += "\n\n实际输出为成功。"
	proposal, err := PrepareSampleCorrection(raw, input)
	require.NoError(t, err)
	require.Contains(t, proposal.Quality.Failures, "unverified_runtime_evidence")
	input.MarkdownBody = strings.TrimSuffix(input.MarkdownBody, "\n\n实际输出为成功。")
	for i := range input.PracticeSet.Tasks {
		if input.PracticeSet.Tasks[i].Mode == sharedtextbook.LearningTaskRetain {
			input.PracticeSet.Tasks[i].MinDelayHours = 0
		}
	}
	proposal, err = PrepareSampleCorrection(raw, input)
	require.NoError(t, err)
	require.False(t, proposal.Quality.Passed)
}
