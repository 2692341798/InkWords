package masteryassessment

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"inkwords-backend/services/review-service/domain/mastery"
)

func TestJudgmentSchemaPinsTaskAndSourceIdentitiesWithoutPromotingAnswer(t *testing.T) {
	input, _ := fixture(t)
	input.TaskReference = &mastery.AssessmentTaskReference{TaskID: "explain", PracticeContentHash: digest("practice"), ExpectedAnswer: "按方法和路径查找"}
	input.DecisionPolicy = mastery.AssessmentJudgmentPolicy
	input.Evidence[0].ID = "gin-engine-add-route"
	generator := NewGeneratorWithOptions(nil, "test-provider", "test-model", Options{GroundedIdentities: true})
	request, err := generator.request(input)
	require.NoError(t, err)
	var schema map[string]any
	require.NoError(t, json.Unmarshal(request.ResponseSchema, &schema))
	judgments := schema["properties"].(map[string]any)["judgments"].(map[string]any)
	properties := judgments["items"].(map[string]any)["properties"].(map[string]any)
	require.Equal(t, float64(len(input.Rubric)), judgments["maxItems"])
	ids := properties["criterion_id"].(map[string]any)["enum"]
	want := []any{}
	for _, criterion := range input.Rubric {
		want = append(want, criterion.ID)
	}
	require.Equal(t, want, ids)
	for _, field := range []map[string]any{properties["evidence_ids"].(map[string]any), schema["$defs"].(map[string]any)["finding"].(map[string]any)["properties"].(map[string]any)["evidence_ids"].(map[string]any)} {
		require.Equal(t, []any{"gin-engine-add-route"}, field["items"].(map[string]any)["enum"])
	}
	require.Contains(t, request.SystemInstruction, "连续原文")
	input.Evidence[0].ID = "another-approved-source"
	changed, err := generator.request(input)
	require.NoError(t, err)
	require.NotEqual(t, string(request.ResponseSchema), string(changed.ResponseSchema))
	// The improvement is limited to canonical judgments; old request contracts
	// keep their historical schema and decoding behavior.
	input.DecisionPolicy = ""
	legacy, err := generator.request(input)
	require.NoError(t, err)
	require.NotContains(t, string(legacy.ResponseSchema), "another-approved-source")
}

func TestGroundedCodeVocabularyIncludesOnlyActualRuntimeAndPreservesUnknownRule(t *testing.T) {
	input := codeAcceptanceInput(t, "correct-method-selection", correctMethodSelection)
	generator := NewGeneratorWithOptions(nil, "test-provider", "test-model", Options{GroundedIdentities: true})
	request, err := generator.request(input)
	require.NoError(t, err)
	require.NotContains(t, string(request.ResponseSchema), "learner-runtime:")
	input.ArtifactHash = input.LearnerArtifact.SnapshotHash
	input.Evidence = append(input.Evidence, mastery.AssessmentEvidence{ID: "learner-runtime:fixture", Kind: "runtime", Excerpt: "passed", ContentHash: digest("passed"), ArtifactHash: input.ArtifactHash, VerificationRunID: "fixture"})
	request, err = generator.request(input)
	require.NoError(t, err)
	var schema map[string]any
	require.NoError(t, json.Unmarshal(request.ResponseSchema, &schema))
	properties := schema["properties"].(map[string]any)["judgments"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)
	refs := properties["evidence_ids"].(map[string]any)
	ids := refs["items"].(map[string]any)["enum"].([]any)
	require.Contains(t, ids, "learner-runtime:fixture")
	require.NotContains(t, ids, "assessment-answer")
	require.NotContains(t, ids, "assessment-learner-code")
	require.Equal(t, float64(0), refs["minItems"], "domain still restricts uncited unknowns to missing runtime")
	require.Contains(t, properties, "answer_span")
}
