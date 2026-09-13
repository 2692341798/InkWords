package masteryassessment

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"inkwords-backend/services/review-service/domain/mastery"
	"inkwords-backend/shared/kernel/generation"
	"testing"
)

func TestRuntimeJudgmentsNeedBothTestObjectAndExecutionEvidence(t *testing.T) {
	input := codeAcceptanceInput(t, "runtime-pair", correctMethodSelection)
	input.ArtifactHash = input.LearnerArtifact.SnapshotHash
	runtimeID := "learner-runtime:00000000-0000-4000-8000-000000000001"
	input.Evidence = append(input.Evidence, mastery.AssessmentEvidence{ID: runtimeID, Kind: "runtime", ContentHash: digest(`{"status":"passed","exit_code":0}`), Excerpt: `{"status":"passed","exit_code":0}`, VerificationRunID: "00000000-0000-4000-8000-000000000001", ArtifactHash: input.ArtifactHash})
	require.NoError(t, input.Validate())
	body := codeSpanResponse(t)
	for _, raw := range body["judgments"].([]any) {
		item := raw.(map[string]any)
		if item["criterion_id"] == "runtime" || item["criterion_id"] == "tests" {
			item["score"], item["status"] = 4, "satisfied"
			item["evidence_ids"] = []string{runtimeID}
			item["unknown_reason"] = ""
		}
	}
	data, err := json.Marshal(body)
	require.NoError(t, err)
	_, err = decodeJudgmentResponse(input, string(data))
	require.Error(t, err, "a receipt is not an answer excerpt")
	for _, raw := range body["judgments"].([]any) {
		item := raw.(map[string]any)
		if item["criterion_id"] == "runtime" || item["criterion_id"] == "tests" {
			item["answer_path"] = "router.go"
			item["answer_span"] = map[string]any{"format": "inkwords.code-line-range.v1", "start_line": 6, "end_line": 11}
		}
	}
	data, err = json.Marshal(body)
	require.NoError(t, err)
	_, err = decodeJudgmentResponse(input, string(data))
	require.NoError(t, err)
	var request generation.Request
	require.NoError(t, addCanonicalCodeRequest(&request, input))
	require.Contains(t, request.SystemInstruction, "不能用运行收据代替作答引文")
	require.Contains(t, request.SystemInstruction, "代码引文只标明受测对象")
	for _, raw := range body["judgments"].([]any) {
		item := raw.(map[string]any)
		if item["criterion_id"] == "runtime" {
			item["evidence_ids"] = []string{"gin-method-tree-identity"}
		}
	}
	data, err = json.Marshal(body)
	require.NoError(t, err)
	_, err = decodeJudgmentResponse(input, string(data))
	require.Error(t, err, "code and source citations cannot replace runtime evidence")
}
