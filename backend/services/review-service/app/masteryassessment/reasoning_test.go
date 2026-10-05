package masteryassessment

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"inkwords-backend/services/review-service/domain/mastery"
)

func TestReasoningOptionBindsPreviewAndExecutionWithoutChangingLegacy(t *testing.T) {
	input, output := fixture(t)
	port := &capturedPort{result: output}
	legacy := NewGenerator(port, "test-provider", "test-model")
	configured := NewGeneratorWithOptions(port, "test-provider", "test-model", Options{ReasoningEffort: "high"})
	oldPreview, err := legacy.Preview(input)
	require.NoError(t, err)
	unchanged, err := configured.Preview(input)
	require.NoError(t, err)
	require.Equal(t, oldPreview, unchanged)

	input.TaskReference = &mastery.AssessmentTaskReference{TaskID: "explain", PracticeContentHash: digest("practice"), ExpectedAnswer: "按方法和路径查找"}
	input.DecisionPolicy = mastery.AssessmentJudgmentPolicy
	port.result = judgmentOutput(t, input, output)
	plain, err := legacy.Preview(input)
	require.NoError(t, err)
	preview, err := configured.Preview(input)
	require.NoError(t, err)
	require.Equal(t, plain.InputHash, preview.InputHash)
	require.NotEqual(t, plain.RequestHash, preview.RequestHash)
	require.Equal(t, "high", preview.ReasoningEffort)
	require.Equal(t, 3000, preview.MaxOutputTokens)
	result, err := configured.Assess(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, preview.RequestHash, result.RequestHash)
	require.Equal(t, "high", result.ReasoningEffort)
	require.Equal(t, "high", port.request.ReasoningEffort)
	encoded, err := json.Marshal(port.request)
	require.NoError(t, err)
	require.Equal(t, preview.RequestHash, digest("test-provider\n"+string(encoded)))
	require.Equal(t, 1, port.calls)
	invalid := NewGeneratorWithOptions(port, "test-provider", "test-model", Options{ReasoningEffort: "unbounded"})
	_, err = invalid.Assess(context.Background(), input)
	require.Error(t, err)
	require.Equal(t, 1, port.calls)
	extended := NewGeneratorWithOptions(port, "test-provider", "test-model", Options{ReasoningEffort: "high", MaxOutputTokens: 6000})
	larger, err := extended.Preview(input)
	require.NoError(t, err)
	require.Equal(t, 6000, larger.MaxOutputTokens)
	require.NotEqual(t, preview.RequestHash, larger.RequestHash)
	for _, tokens := range []int{-1, 6001} {
		_, err = NewGeneratorWithOptions(port, "test-provider", "test-model", Options{MaxOutputTokens: tokens}).Assess(context.Background(), input)
		require.ErrorIs(t, err, ErrBudgetExceeded)
	}
	require.Equal(t, 1, port.calls)
}

func TestReasoningAcceptanceInputsAreBudgetedWithoutProvider(t *testing.T) {
	for _, effort := range []string{"high", "low"} {
		t.Run(effort, func(t *testing.T) { reasoningAcceptancePreflight(t, effort) })
	}
}

func reasoningAcceptancePreflight(t *testing.T, effort string) {
	t.Helper()
	port := &capturedPort{}
	g := NewGeneratorWithOptions(port, "deepseek", "deepseek-v4-flash", Options{ReasoningEffort: effort})
	for index, answer := range []string{causalAcceptanceAnswer, proceduralAcceptanceAnswer, incompleteAcceptanceAnswer, misconceptionAcceptanceAnswer} {
		input := referenceAcceptanceInput(index, answer)
		input.DecisionPolicy = mastery.AssessmentJudgmentPolicy
		preview, err := g.Preview(input)
		require.NoError(t, err)
		encoded, err := json.Marshal(preview)
		require.NoError(t, err)
		t.Logf("case=%d preview=%s", index+1, encoded)
	}
	require.Zero(t, port.calls)
}
