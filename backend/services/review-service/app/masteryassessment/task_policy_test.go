package masteryassessment

import (
	"testing"

	"github.com/stretchr/testify/require"
	"inkwords-backend/services/review-service/domain/mastery"
)

func TestTaskPolicyPreservesCalibratedRequestIdentities(t *testing.T) {
	port := &capturedPort{}
	g := NewGeneratorWithTaskPolicy(port, "deepseek", "deepseek-v4-flash", TaskModelPolicy{Text: Options{ReasoningEffort: "low", MaxOutputTokens: 6000}, Code: Options{MaxOutputTokens: 6000}})
	text := referenceAcceptanceInput(0, causalAcceptanceAnswer)
	text.DecisionPolicy = mastery.AssessmentJudgmentPolicy
	preview, err := g.Preview(text)
	require.NoError(t, err)
	require.Equal(t, "sha256:d8089a74caeec70d8ab5c9b16f6227325eb01dca06d67e23bfd31bd72a6dfb90", preview.RequestHash)
	require.Equal(t, "low", preview.ReasoningEffort)
	code := codeAcceptanceInput(t, "correct-method-selection", correctMethodSelection)
	preview, err = g.Preview(code)
	require.NoError(t, err)
	require.Equal(t, "sha256:72863da66c3f8663f38a033226954a4f27553c6ef960928eb376b142edde1780", preview.RequestHash)
	require.Empty(t, preview.ReasoningEffort)
	require.Equal(t, 6000, preview.MaxOutputTokens)
	text.DecisionPolicy = mastery.AssessmentDecisionPolicy
	legacy, err := NewGenerator(port, "deepseek", "deepseek-v4-flash").Preview(text)
	require.NoError(t, err)
	preview, err = g.Preview(text)
	require.NoError(t, err)
	require.Equal(t, legacy, preview)
	require.Zero(t, port.calls)
}
