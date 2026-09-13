package masteryassessment

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"inkwords-backend/services/review-service/domain/mastery"
	"inkwords-backend/shared/kernel/generation"
)

func judgmentOutput(t *testing.T, input mastery.AssessmentInput, output generation.Result) generation.Result {
	t.Helper()
	var old struct{ mastery.AssessmentFeedback }
	require.NoError(t, json.Unmarshal([]byte(output.Output), &old))
	items := make([]mastery.AssessmentJudgment, len(old.Criteria))
	for index, item := range old.Criteria {
		items[index] = mastery.AssessmentJudgment{CriterionID: item.ID, Score: item.Score, Status: "satisfied", AnswerQuote: item.AnswerQuote, AnswerPath: item.AnswerPath, EvidenceIDs: item.EvidenceIDs, Gaps: []mastery.AssessmentJudgmentGap{}}
		if item.Score == nil {
			items[index].Status = "unknown"
			items[index].UnknownReason = "没有同一作答的运行证据"
		} else if *item.Score < 3 {
			items[index].Status = "unsatisfied"
			for _, criterion := range input.Rubric {
				if criterion.ID == item.ID {
					items[index].Gaps = []mastery.AssessmentJudgmentGap{{Kind: "misconception", RequirementQuote: criterion.Description, Reason: "本次测试没有满足此要求"}}
				}
			}
		}
	}
	response := map[string]any{"judgments": items, "next_hint": old.NextHint, "remediation": old.Remediation}
	data, err := json.Marshal(response)
	require.NoError(t, err)
	if input.LearnerArtifact != nil {
		var raw map[string]any
		require.NoError(t, json.Unmarshal(data, &raw))
		for _, item := range raw["judgments"].([]any) {
			if _, found := item.(map[string]any)["answer_path"]; !found {
				item.(map[string]any)["answer_path"] = ""
			}
		}
		data, err = json.Marshal(raw)
		require.NoError(t, err)
	}
	output.Output = string(data)
	return output
}

func TestCanonicalProviderProducesOnlyOneGradingLedgerAndRejectsProjectionTampering(t *testing.T) {
	input, output := fixture(t)
	input.TaskReference = &mastery.AssessmentTaskReference{TaskID: "explain", PracticeContentHash: digest("practice"), ExpectedAnswer: "按方法和路径查找"}
	input.DecisionPolicy = mastery.AssessmentJudgmentPolicy
	output = judgmentOutput(t, input, output)
	port := &capturedPort{result: output}
	result, err := NewGenerator(port, "test-provider", "test-model").Assess(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, mastery.AssessmentJudgmentContractVersion, result.Contract)
	require.NoError(t, result.ValidateFeedback(input))
	require.Len(t, result.Judgments, 5)
	require.Equal(t, "所引作答满足本项要求。", result.Feedback.Criteria[0].Reason)
	var schema map[string]any
	require.NoError(t, json.Unmarshal(port.request.ResponseSchema, &schema))
	properties := schema["properties"].(map[string]any)
	require.Contains(t, properties, "judgments")
	for _, name := range []string{"criteria", "decisions", "missing_points", "misconceptions", "correct_points"} {
		require.NotContains(t, properties, name)
	}
	result.Feedback.Criteria[0].Reason = "额外生成的独立理由"
	require.Error(t, result.ValidateFeedback(input))
	input.DecisionPolicy = mastery.AssessmentDecisionPolicy
	request, err := requestForAssessment("test-model", input)
	require.NoError(t, err)
	require.Contains(t, string(request.ResponseSchema), "decisions")
	require.NotContains(t, string(request.ResponseSchema), "judgments")
}

func TestCanonicalProviderRejectsLegacyDuplicatedFeedbackAndOmittedFields(t *testing.T) {
	for _, name := range []string{"criteria", "reason", "score", "answer_quote", "unknown_reason", "answer_path"} {
		t.Run(name, func(t *testing.T) {
			input, output := fixture(t)
			input.TaskReference = &mastery.AssessmentTaskReference{TaskID: "explain", PracticeContentHash: digest("practice"), ExpectedAnswer: "按方法和路径查找"}
			input.DecisionPolicy = mastery.AssessmentJudgmentPolicy
			output = judgmentOutput(t, input, output)
			var response map[string]any
			require.NoError(t, json.Unmarshal([]byte(output.Output), &response))
			item := response["judgments"].([]any)[0].(map[string]any)
			switch name {
			case "criteria":
				response["criteria"] = []any{}
			case "reason":
				item["reason"] = "再次生成的扣分说明"
			case "answer_path":
				item["answer_path"] = ""
			default:
				delete(item, name)
			}
			data, err := json.Marshal(response)
			require.NoError(t, err)
			output.Output = string(data)
			port := &capturedPort{result: output}
			result, err := NewGenerator(port, "test-provider", "test-model").Assess(context.Background(), input)
			require.ErrorIs(t, err, ErrInvalidFeedback)
			require.Nil(t, result.Feedback)
			require.Equal(t, 120, result.Usage.InputTokens)
			require.Equal(t, 1, port.calls)
		})
	}
}
