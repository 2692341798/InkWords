package masteryassessment

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"inkwords-backend/services/review-service/domain/mastery"
	"inkwords-backend/shared/kernel/generation"
)

func decisionOutput(t *testing.T, input mastery.AssessmentInput, output generation.Result) generation.Result {
	t.Helper()
	var response map[string]any
	require.NoError(t, json.Unmarshal([]byte(output.Output), &response))
	decisions := make([]mastery.AssessmentDecision, len(input.Rubric))
	for index, criterion := range input.Rubric {
		decisions[index] = mastery.AssessmentDecision{CriterionID: criterion.ID, Requirement: criterion.Description, Status: "satisfied", Gaps: []mastery.AssessmentDecisionGap{}}
		for _, raw := range response["criteria"].([]any) {
			item := raw.(map[string]any)
			if item["id"] != criterion.ID {
				continue
			}
			if item["score"] == nil {
				decisions[index].Status = "unknown"
			} else if item["score"].(float64) < 3 {
				decisions[index].Status = "unsatisfied"
				decisions[index].Gaps = []mastery.AssessmentDecisionGap{{RequirementQuote: criterion.Description, Reason: "受控评测夹具没有满足本项"}}
			}
		}
	}
	response["decisions"] = decisions
	data, err := json.Marshal(response)
	require.NoError(t, err)
	output.Output = string(data)
	return output
}

func TestRejectedAcceptanceArtifactDoesNotInventZeroScores(t *testing.T) {
	outputDir := t.TempDir()
	result := Result{ProviderCalls: 1, Usage: generation.Usage{Known: true, InputTokens: 120, OutputTokens: 80}, Rejection: &FeedbackRejection{Code: "decision_score", CriterionID: "causality"}}
	require.NoError(t, writeRealAssessmentArtifact(outputDir, "rejected", acceptanceScore{}, result))
	encoded, err := os.ReadFile(filepath.Join(outputDir, "rejected.json"))
	require.NoError(t, err)
	var artifact map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(encoded, &artifact))
	require.NotContains(t, artifact, "acceptance")
	require.Contains(t, string(encoded), "decision_score")
}

func TestDecisionRequestRequiresExactCoverageAndPreservesLegacySchema(t *testing.T) {
	input, output := fixture(t)
	input.TaskReference = &mastery.AssessmentTaskReference{TaskID: "explain", PracticeContentHash: digest("practice"), ExpectedAnswer: "按方法和路径查找。"}
	generator := NewGenerator(&capturedPort{}, "test-provider", "test-model")
	legacy, err := generator.Preview(input)
	require.NoError(t, err)
	input.DecisionPolicy = mastery.AssessmentDecisionPolicy
	output = decisionOutput(t, input, output)
	port := &capturedPort{result: output}
	result, err := NewGenerator(port, "test-provider", "test-model").Assess(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, mastery.AssessmentDecisionContractVersion, result.Contract)
	require.Len(t, result.Decisions, len(input.Rubric))
	require.NotEqual(t, legacy.RequestHash, result.RequestHash)
	require.Contains(t, string(port.request.ResponseSchema), "decisions")
	require.NotContains(t, string(feedbackSchema), "decisions", "saved legacy request schema is unchanged")
	require.Contains(t, port.request.SystemInstruction, "requirement_quote")
	input.DecisionPolicy = ""
	_, err = NewGenerator(port, "test-provider", "test-model").Assess(context.Background(), input)
	require.ErrorIs(t, err, ErrInvalidFeedback, "unsolicited decisions must not change a legacy result")
}

func TestDecisionGeneratorRejectsContradictionsAndKeepsUsageWithoutRetry(t *testing.T) {
	for _, name := range []string{"missing", "null", "unmet passing", "unbound gap", "unknown property"} {
		t.Run(name, func(t *testing.T) {
			input, output := fixture(t)
			input.TaskReference = &mastery.AssessmentTaskReference{TaskID: "explain", PracticeContentHash: digest("practice"), ExpectedAnswer: "按方法和路径查找。"}
			input.DecisionPolicy = mastery.AssessmentDecisionPolicy
			output = decisionOutput(t, input, output)
			var response map[string]any
			require.NoError(t, json.Unmarshal([]byte(output.Output), &response))
			first := response["decisions"].([]any)[0].(map[string]any)
			switch name {
			case "missing":
				delete(response, "decisions")
			case "null":
				response["decisions"] = nil
			case "unmet passing":
				first["status"] = "unsatisfied"
				first["gaps"] = []mastery.AssessmentDecisionGap{{RequirementQuote: input.Rubric[0].Description, Reason: "没有体现要求"}}
			case "unbound gap":
				first["status"] = "unsatisfied"
				response["criteria"].([]any)[0].(map[string]any)["score"] = 2
				first["gaps"] = []mastery.AssessmentDecisionGap{{RequirementQuote: "题外要求", Reason: "没有体现要求"}}
			case "unknown property":
				first["approved"] = true
			}
			data, err := json.Marshal(response)
			require.NoError(t, err)
			output.Output = string(data)
			port := &capturedPort{result: output}
			result, err := NewGenerator(port, "test-provider", "test-model").Assess(context.Background(), input)
			require.ErrorIs(t, err, ErrInvalidFeedback)
			require.Nil(t, result.Feedback)
			require.Nil(t, result.Decisions)
			require.Equal(t, 120, result.Usage.InputTokens)
			require.Equal(t, 1, port.calls)
			require.NotNil(t, result.Rejection)
			if name == "unmet passing" {
				require.Equal(t, "decision_score", result.Rejection.Code)
				require.Equal(t, "accuracy", result.Rejection.CriterionID)
			}
			if name == "unbound gap" {
				require.Equal(t, "decision_gap_requirement", result.Rejection.Code)
			}
		})
	}
}
