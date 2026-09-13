package masteryassessment

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"inkwords-backend/services/review-service/domain/mastery"
)

func TestCanonicalRejectionsIdentifyRulesWithoutRetainingProviderText(t *testing.T) {
	for _, test := range []struct {
		name   string
		code   string
		change func(map[string]any)
	}{
		{"quote", "feedback_answer_quote", func(body map[string]any) {
			body["judgments"].([]any)[0].(map[string]any)["answer_quote"] = "private invented quote"
		}},
		{"evidence", "feedback_evidence_identity", func(body map[string]any) {
			body["judgments"].([]any)[0].(map[string]any)["evidence_ids"] = []string{"private invented evidence"}
		}},
		{"hint", "feedback_finding_text", func(body map[string]any) { body["next_hint"].(map[string]any)["text"] = "" }},
		{"criterion", "decision_requirement", func(body map[string]any) {
			body["judgments"].([]any)[0].(map[string]any)["criterion_id"] = "private invented criterion"
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			input, output := fixture(t)
			input.TaskReference = &mastery.AssessmentTaskReference{TaskID: "explain", PracticeContentHash: digest("practice"), ExpectedAnswer: "按方法和路径查找"}
			input.DecisionPolicy = mastery.AssessmentJudgmentPolicy
			output = judgmentOutput(t, input, output)
			var body map[string]any
			require.NoError(t, json.Unmarshal([]byte(output.Output), &body))
			test.change(body)
			data, err := json.Marshal(body)
			require.NoError(t, err)
			output.Output = string(data)
			port := &capturedPort{result: output}
			result, err := NewGenerator(port, "test-provider", "test-model").Assess(context.Background(), input)
			require.ErrorIs(t, err, ErrInvalidFeedback)
			require.Equal(t, test.code, result.Rejection.Code)
			require.Nil(t, result.Feedback)
			require.Nil(t, result.Judgments)
			require.Equal(t, 1, port.calls)
			if test.name == "quote" || test.name == "evidence" {
				require.Equal(t, input.Rubric[0].ID, result.Rejection.CriterionID)
			} else {
				require.Empty(t, result.Rejection.CriterionID)
			}
			saved, err := json.Marshal(result)
			require.NoError(t, err)
			require.NotContains(t, string(saved), "private invented")
		})
	}
}
