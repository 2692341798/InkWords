package masteryassessment

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"inkwords-backend/services/review-service/domain/mastery"
)

// TestReplaySavedProceduralDecisionReceipt is a local-only diagnostic. It
// projects a saved v5 ledger, never claiming that the model produced v6 output.
func TestReplaySavedProceduralDecisionReceipt(t *testing.T) {
	path := os.Getenv("INKWORDS_ASSESSMENT_REPLAY_FILE")
	if path == "" {
		t.Skip("saved local receipt replay requires explicit paths")
	}
	outputPath := os.Getenv("INKWORDS_ASSESSMENT_REPLAY_OUTPUT")
	require.NotEmpty(t, outputPath)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var receipt realAssessmentArtifact
	require.NoError(t, json.Unmarshal(data, &receipt))
	require.Equal(t, "procedural-only-answer", receipt.Case)
	require.NotNil(t, receipt.Input)
	require.Equal(t, mastery.AssessmentDecisionPolicy, receipt.Input.DecisionPolicy)
	require.NoError(t, receipt.Result.ValidateFeedback(*receipt.Input))
	require.Empty(t, receipt.Result.Feedback.Misconceptions)
	judgments := make([]mastery.AssessmentJudgment, 0, len(receipt.Result.Decisions))
	for _, decision := range receipt.Result.Decisions {
		var criterion *mastery.CriterionAssessment
		for index := range receipt.Result.Feedback.Criteria {
			if receipt.Result.Feedback.Criteria[index].ID == decision.CriterionID {
				criterion = &receipt.Result.Feedback.Criteria[index]
			}
		}
		require.NotNil(t, criterion)
		judgment := mastery.AssessmentJudgment{CriterionID: criterion.ID, Score: criterion.Score, Status: decision.Status, AnswerQuote: criterion.AnswerQuote, EvidenceIDs: criterion.EvidenceIDs, Gaps: []mastery.AssessmentJudgmentGap{}}
		for _, gap := range decision.Gaps {
			judgment.Gaps = append(judgment.Gaps, mastery.AssessmentJudgmentGap{Kind: "omission", RequirementQuote: gap.RequirementQuote, Reason: gap.Reason})
		}
		judgments = append(judgments, judgment)
	}
	input := *receipt.Input
	input.DecisionPolicy = mastery.AssessmentJudgmentPolicy
	feedback, decisions, err := mastery.ProjectAssessmentJudgments(input, judgments, receipt.Result.Feedback.NextHint, receipt.Result.Feedback.Remediation)
	require.NoError(t, err)
	require.Len(t, feedback.MissingPoints, 1)
	require.Equal(t, "所引作答满足本项要求。", feedback.Criteria[1].Reason)
	require.Equal(t, "所引作答满足本项要求。", feedback.Criteria[4].Reason)
	projection := struct {
		Origin            string                       `json:"origin"`
		SourceReceiptHash string                       `json:"source_receipt_hash"`
		ProviderCalls     int                          `json:"provider_calls"`
		Input             mastery.AssessmentInput      `json:"input"`
		Feedback          mastery.AssessmentFeedback   `json:"feedback"`
		Decisions         []mastery.AssessmentDecision `json:"decisions"`
		Judgments         []mastery.AssessmentJudgment `json:"judgments"`
	}{"offline_projection_from_v5", digest(string(data)), 0, input, feedback, decisions, judgments}
	encoded, err := json.MarshalIndent(projection, "", "  ")
	require.NoError(t, err)
	f, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	require.NoError(t, err)
	_, err = f.Write(encoded)
	require.NoError(t, f.Close())
	require.NoError(t, err)
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, data, after)
}
