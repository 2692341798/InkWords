package masteryassessment

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"inkwords-backend/services/review-service/domain/mastery"
	shared "inkwords-backend/shared/kernel/textbook"
	llm "inkwords-backend/shared/platform/llm"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRealRuntimeAssessment requires a separately captured sandbox receipt.
// Each invocation is one explicitly selected evaluation, without persistence
// to personal learning state, implicit retries, or synthetic execution output.
func TestRealRuntimeAssessment(t *testing.T) {
	if os.Getenv("INKWORDS_REAL_RUNTIME_ASSESSMENT") != "approved" {
		t.Skip("real provider/runtime evaluation requires explicit opt-in")
	}
	dir := os.Getenv("INKWORDS_RUNTIME_ASSESSMENT_DIR")
	require.NotEmpty(t, dir)
	name := os.Getenv("INKWORDS_RUNTIME_ASSESSMENT_CASE")
	require.Contains(t, []string{"correct-method-selection", "wrong-method-selection"}, name)
	var input mastery.AssessmentInput
	var resolved shared.LearnerVerificationInput
	var reference shared.LearnerVerificationReference
	var report shared.LearnerVerificationReport
	for suffix, value := range map[string]any{"assessment": &input, "input": &resolved, "reference": &reference, "report": &report} {
		data, err := os.ReadFile(filepath.Join(dir, name+"-"+suffix+".json"))
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(data, value))
	}
	require.NoError(t, resolved.ValidateFor(reference))
	require.NoError(t, report.ValidateFor(reference, resolved.Plan))
	require.True(t, report.ExecutionStarted)
	expected := shared.LearnerVerificationPassed
	if name == "wrong-method-selection" {
		expected = shared.LearnerVerificationFailed
	}
	require.Equal(t, expected, report.Status)
	attached, err := mastery.AttachLearnerVerification(input, resolved, report)
	require.NoError(t, err)
	require.Len(t, attached.Evidence, len(input.Evidence)+1)
	key := firstConfigured(os.Getenv("MASTERY_DEEPSEEK_API_KEY"), os.Getenv("DEEPSEEK_API_KEY"))
	require.NotEmpty(t, key)
	model := firstConfigured(os.Getenv("MASTERY_DEEPSEEK_MODEL"), os.Getenv("DEEPSEEK_MODEL"))
	require.Equal(t, "deepseek-v4-flash", model)
	port := &codeDiagnosticPort{Port: llm.NewDeepSeekGenerationAdapterWithTimeout(llm.NewDeepSeekClient(key), 60*time.Second)}
	generator := NewGeneratorWithOptions(port, "deepseek", model, Options{MaxOutputTokens: 6000})
	preview, err := generator.Preview(attached)
	require.NoError(t, err)
	marker, err := os.OpenFile(filepath.Join(dir, name+"-provider-scope.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	require.NoError(t, err)
	require.NoError(t, json.NewEncoder(marker).Encode(map[string]any{"origin": "operator_authored_runtime_grading_evaluation", "max_provider_calls": 1, "retries": 0, "code_execution_by_grader": 0, "human_reviews": 0, "learning_record_writes": 0, "preview": preview, "verification_run_id": report.RunID}))
	require.NoError(t, marker.Close())
	result, err := generator.Assess(context.Background(), attached)
	require.NoError(t, writeRealAssessmentArtifact(dir, name+"-runtime", acceptanceScore{}, result, attached))
	diagnostic, writeErr := os.OpenFile(filepath.Join(dir, name+"-runtime-model-response.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	require.NoError(t, writeErr)
	require.NoError(t, json.NewEncoder(diagnostic).Encode(port.response))
	require.NoError(t, diagnostic.Close())
	require.NoError(t, err, "stop after this one provider call")
	require.Equal(t, 1, result.ProviderCalls)
	criteria := map[string]mastery.CriterionAssessment{}
	for _, c := range result.Feedback.Criteria {
		criteria[c.ID] = c
	}
	for _, id := range []string{"runtime", "tests"} {
		item := criteria[id]
		require.NotNil(t, item.Score, "matching executed receipt must not be treated as absent")
		require.Contains(t, item.EvidenceIDs, "learner-runtime:"+report.RunID)
		t.Logf("case=%s criterion=%s score=%d reason=%s", name, id, *item.Score, item.Reason)
	}
	if name == "correct-method-selection" {
		require.GreaterOrEqual(t, *criteria["tests"].Score, 3)
	} else {
		require.LessOrEqual(t, *criteria["tests"].Score, 1, "failed assertions must not be scored as passed tests")
	}
}
