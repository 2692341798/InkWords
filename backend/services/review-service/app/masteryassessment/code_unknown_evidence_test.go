package masteryassessment

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"inkwords-backend/services/review-service/domain/mastery"
)

func TestCodeUnknownRuntimeDoesNotRequireAnUnrelatedSourceCitation(t *testing.T) {
	input := codeAcceptanceInput(t, "correct-method-selection", correctMethodSelection)
	four := 4
	judgments := []mastery.AssessmentJudgment{}
	for _, criterion := range input.Rubric {
		item := mastery.AssessmentJudgment{CriterionID: criterion.ID, Score: &four, Status: "satisfied", AnswerPath: "router.go", AnswerQuote: "if tree.method == method { return tree.root }", EvidenceIDs: []string{"gin-method-tree-identity"}, Gaps: []mastery.AssessmentJudgmentGap{}}
		if criterion.RequiresRuntime {
			item.Score, item.Status, item.AnswerPath, item.AnswerQuote = nil, "unknown", "", ""
			item.EvidenceIDs = []string{}
			item.UnknownReason = "本次没有提供同一作答的运行记录。"
		}
		judgments = append(judgments, item)
	}
	hint := mastery.AssessmentFinding{Text: "检查没有匹配时的返回值。", EvidenceIDs: []string{"gin-method-tree-identity"}}
	feedback, _, err := mastery.ProjectAssessmentJudgments(input, judgments, hint, []mastery.AssessmentFinding{hint})
	require.NoError(t, err)
	require.Nil(t, feedback.Criteria[2].Score)
	require.Empty(t, feedback.Criteria[2].EvidenceIDs)
	request, err := requestForAssessment("deepseek-v4-flash", input)
	require.NoError(t, err)
	var schema map[string]any
	require.NoError(t, json.Unmarshal(request.ResponseSchema, &schema))
	item := schema["properties"].(map[string]any)["judgments"].(map[string]any)["items"].(map[string]any)
	refs := item["properties"].(map[string]any)["evidence_ids"].(map[string]any)
	require.EqualValues(t, 0, refs["minItems"])

	for _, mutation := range []func(*mastery.AssessmentFeedback){
		func(f *mastery.AssessmentFeedback) { f.Criteria[0].EvidenceIDs = nil },
		func(f *mastery.AssessmentFeedback) { f.Criteria[2].Score = &four },
		func(f *mastery.AssessmentFeedback) { f.Criteria[2].EvidenceIDs = []string{"invented-runtime"} },
		func(f *mastery.AssessmentFeedback) {
			f.Criteria[0].AnswerQuote = "for _, tree := range trees { if tree.method == method { return tree.root } } return nil"
		},
	} {
		copy := feedback
		copy.Criteria = append([]mastery.CriterionAssessment(nil), feedback.Criteria...)
		mutation(&copy)
		require.Error(t, copy.Validate(input))
	}
	legacy := input
	legacy.DecisionPolicy = ""
	require.Error(t, feedback.Validate(legacy), "legacy validation must not change")
	withRuntime := input
	withRuntime.ArtifactHash = input.LearnerArtifact.SnapshotHash
	excerpt := `{"status":"passed"}`
	withRuntime.Evidence = append(append([]mastery.AssessmentEvidence(nil), input.Evidence...), mastery.AssessmentEvidence{ID: "learner-runtime:fixture", Kind: "runtime", Excerpt: excerpt, ContentHash: digest(excerpt), ArtifactHash: withRuntime.ArtifactHash, VerificationRunID: "fixture"})
	require.NoError(t, withRuntime.Validate())
	require.Error(t, feedback.Validate(withRuntime), "an available run cannot be silently treated as absent")
}
