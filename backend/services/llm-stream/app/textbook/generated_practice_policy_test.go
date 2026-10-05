package textbook

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	sharedgeneration "inkwords-backend/shared/kernel/generation"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func TestGeneratedPracticeUsesSystemRuntimePolicy(t *testing.T) {
	for _, omitFlags := range []bool{true, false} {
		t.Run(map[bool]string{true: "omitted", false: "contradictory"}[omitFlags], func(t *testing.T) {
			request := sampleGenerationRequest(t)
			chapter, err := BuildGinRequestLifecycleSample(request.EvidencePack)
			require.NoError(t, err)
			expected := chapter.PracticeSet
			chapter.Markdown = strings.TrimSpace(strings.TrimSuffix(chapter.Markdown, sharedtextbook.RenderPracticeSet(expected)))
			encoded, err := json.Marshal(chapter)
			require.NoError(t, err)
			var output map[string]any
			require.NoError(t, json.Unmarshal(encoded, &output))
			for _, task := range output["practice_set"].(map[string]any)["tasks"].([]any) {
				for _, raw := range task.(map[string]any)["rubric"].([]any) {
					criterion := raw.(map[string]any)
					if omitFlags {
						delete(criterion, "requires_runtime")
					} else {
						criterion["requires_runtime"] = !criterion["requires_runtime"].(bool)
					}
				}
			}
			encoded, err = json.Marshal(output)
			require.NoError(t, err)
			port := &capturedGenerationPort{result: sharedgeneration.Result{Provider: "fake-provider", Model: "configured-model", Output: string(encoded)}}
			result, err := NewPortSampleGenerator(port, "configured-model").Generate(context.Background(), request)
			require.NoError(t, err)
			require.Equal(t, expected, result.Chapter.PracticeSet)
			require.Contains(t, result.Chapter.Markdown, sharedtextbook.RenderPracticeSet(expected))
			require.Equal(t, "unverified", result.Chapter.RuntimeVerification)
			require.NotContains(t, string(port.request.ResponseSchema), `"requires_runtime"`)
			require.Contains(t, port.request.TaskInstruction, "运行证据标记由系统")
		})
	}
}

// TestFrozenRejectedDraftRuntimePolicyComparison is an explicit offline replay,
// separate from live generation and from immutable-receipt rechecking.
func TestFrozenRejectedDraftRuntimePolicyComparison(t *testing.T) {
	path := os.Getenv("INKWORDS_PRACTICE_POLICY_RECEIPT")
	if path == "" {
		t.Skip("set INKWORDS_PRACTICE_POLICY_RECEIPT for an offline comparison")
	}
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var draft RejectedDraft
	require.NoError(t, json.Unmarshal(data, &draft))
	require.NoError(t, draft.Validate())
	before, err := RecheckRejectedDraft(draft, draft.Generation.Chapter.Markdown)
	require.NoError(t, err)
	require.False(t, before.Quality.Passed)
	// Detach every authored field before applying the new generation policy.
	encoded, err := json.Marshal(draft.Generation.Chapter)
	require.NoError(t, err)
	var chapter SampleChapter
	require.NoError(t, json.Unmarshal(encoded, &chapter))
	applyGeneratedPracticePolicy(&chapter.PracticeSet)
	originalAfter, err := json.Marshal(draft.Generation.Chapter)
	require.NoError(t, err)
	require.Equal(t, encoded, originalAfter)
	require.NoError(t, chapter.PracticeSet.Validate(chapter.EvidenceIDs))
	if !strings.Contains(chapter.Markdown, "<!-- "+sharedtextbook.PracticeSetVersion+":") {
		chapter.Markdown = strings.TrimSpace(chapter.Markdown) + "\n\n" + sharedtextbook.RenderPracticeSet(chapter.PracticeSet)
	}
	chapter.ContentHash = digest(chapter.Markdown)
	after := RunSampleChapterQualityGates(chapter, draft.Request.EvidencePack)
	after.Failures = append(after.Failures, requiredBlueprintClaimFailures(chapter.Claims, draft.Request.BlueprintChapter.CriticalClaims)...)
	after.Passed = len(after.Failures) == 0
	require.False(t, after.Passed, "the known rejected draft must still fail content gates")
	require.NotContains(t, strings.Join(after.Failures, ";"), "runtime boundary")
	unchanged, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, data, unchanged)
	if dir := os.Getenv("INKWORDS_PRACTICE_POLICY_COMPARISON_DIR"); dir != "" {
		report := map[string]any{
			"origin": "automated_offline_policy_comparison", "provider_calls": 0, "candidate_persisted": false,
			"receipt_hash": digest(string(data)), "original_prompt_schema": draft.PromptSchema,
			"evaluated_prompt_schema": sharedtextbook.SamplePromptSchemaVersion,
			"original_content_hash":   before.OriginalContentHash, "evaluated_content_hash": chapter.ContentHash,
			"before": before.Quality, "after": after,
		}
		encoded, err := json.MarshalIndent(report, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.MkdirAll(dir, 0o700))
		file, err := os.OpenFile(filepath.Join(dir, "runtime-policy-comparison.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		require.NoError(t, err)
		_, err = file.Write(encoded)
		require.NoError(t, err)
		require.NoError(t, file.Close())
	}
}

func TestGeneratedPracticePolicyDoesNotRepairAuthoredFailures(t *testing.T) {
	cases := []struct {
		name string
		edit func(*SampleChapter)
	}{
		{"missing criterion", func(c *SampleChapter) { c.PracticeSet.Tasks[0].Rubric = c.PracticeSet.Tasks[0].Rubric[:4] }},
		{"unknown criterion", func(c *SampleChapter) { c.PracticeSet.Tasks[0].Rubric[0].ID = "unknown" }},
		{"duplicate criterion", func(c *SampleChapter) { c.PracticeSet.Tasks[0].Rubric[0].ID = c.PracticeSet.Tasks[0].Rubric[1].ID }},
		{"empty description", func(c *SampleChapter) { c.PracticeSet.Tasks[0].Rubric[0].Description = "" }},
		{"unknown mode", func(c *SampleChapter) { c.PracticeSet.Tasks[0].Mode = "unknown" }},
		{"missing task", func(c *SampleChapter) { c.PracticeSet.Tasks = c.PracticeSet.Tasks[:5] }},
		{"unknown evidence", func(c *SampleChapter) { c.PracticeSet.Tasks[0].EvidenceIDs = []string{"unknown"} }},
		{"invalid delay", func(c *SampleChapter) { c.PracticeSet.Tasks[5].MinDelayHours = 0 }},
		{"rendered mismatch", func(c *SampleChapter) {
			c.PracticeSet.Tasks[2].Rubric[2].RequiresRuntime = false
			c.Markdown += "\n\n" + sharedtextbook.RenderPracticeSet(c.PracticeSet)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := sampleGenerationRequest(t)
			chapter, err := BuildGinRequestLifecycleSample(request.EvidencePack)
			require.NoError(t, err)
			chapter.Markdown = strings.TrimSpace(strings.TrimSuffix(chapter.Markdown, sharedtextbook.RenderPracticeSet(chapter.PracticeSet)))
			tc.edit(&chapter)
			encoded, err := json.Marshal(chapter)
			require.NoError(t, err)
			port := &capturedGenerationPort{result: sharedgeneration.Result{Provider: "fake-provider", Model: "configured-model", Output: string(encoded)}}
			result, err := NewPortSampleGenerator(port, "configured-model").Generate(context.Background(), request)
			require.ErrorContains(t, err, "failed quality gates")
			require.False(t, result.Quality.Passed)
			require.Empty(t, result.Candidate.ID)
		})
	}
}
