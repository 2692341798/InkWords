package masteryassessment

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"inkwords-backend/services/review-service/domain/mastery"
)

func TestCanonicalRequestHasOneAnswerAuthorityInsteadOfAnEmptyShadow(t *testing.T) {
	input := referenceAcceptanceInput(0, causalAcceptanceAnswer)
	input.DecisionPolicy = mastery.AssessmentJudgmentPolicy
	request, err := requestForAssessment("fixture", input)
	require.NoError(t, err)
	var metadata map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.SplitN(request.TaskInstruction, "\n", 2)[1]), &metadata))
	require.NotContains(t, metadata, "answer")
	require.NotContains(t, metadata, "evidence")
	require.Equal(t, "assessment-answer", metadata["answer_evidence_id"])
	count := 0
	for _, evidence := range request.Evidence {
		if evidence.ID == metadata["answer_evidence_id"] {
			count++
			require.Equal(t, input.Answer, evidence.Content)
			require.Equal(t, mastery.AssessmentInputHash(input), evidence.Locator)
		}
	}
	require.Equal(t, 1, count)
	input.DecisionPolicy = mastery.AssessmentDecisionPolicy
	legacy, err := requestForAssessment("fixture", input)
	require.NoError(t, err)
	metadata = nil
	require.NoError(t, json.Unmarshal([]byte(strings.SplitN(legacy.TaskInstruction, "\n", 2)[1]), &metadata))
	require.Equal(t, "", metadata["answer"], "historical request identities stay reproducible")
	require.NotContains(t, metadata, "answer_evidence_id")
}

func TestAnswerBindingExtendedBudgetPreflight(t *testing.T) {
	port := &capturedPort{}
	g := NewGeneratorWithOptions(port, "deepseek", "deepseek-v4-flash", Options{ReasoningEffort: "low", MaxOutputTokens: 6000})
	for index, answer := range []string{causalAcceptanceAnswer, proceduralAcceptanceAnswer, incompleteAcceptanceAnswer, misconceptionAcceptanceAnswer} {
		input := referenceAcceptanceInput(index, answer)
		input.DecisionPolicy = mastery.AssessmentJudgmentPolicy
		preview, err := g.Preview(input)
		require.NoError(t, err)
		data, err := json.Marshal(preview)
		require.NoError(t, err)
		t.Logf("case=%d preview=%s", index+1, data)
	}
	require.Zero(t, port.calls)
}
