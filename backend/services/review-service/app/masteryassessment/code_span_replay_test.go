package masteryassessment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"inkwords-backend/services/review-service/domain/mastery"
	"inkwords-backend/shared/kernel/generation"
)

// TestReplayRealCodeResponses preserves original receipts while producing the
// current deterministic wire projection and two reviewable UI correction drafts.
func TestReplayRealCodeResponses(t *testing.T) {
	source := os.Getenv("INKWORDS_CODE_REPLAY_SOURCE")
	if source == "" {
		t.Skip("requires explicit fixed QA responses; no provider calls")
	}
	output := os.Getenv("INKWORDS_CODE_REPLAY_OUTPUT")
	require.NotEmpty(t, output)
	require.NoError(t, os.Mkdir(output, 0700))
	for _, sub := range []string{"receipts", "proposals"} {
		require.NoError(t, os.Mkdir(filepath.Join(output, sub), 0700))
	}
	for _, name := range []string{"correct-method-selection", "wrong-method-selection"} {
		original, err := os.ReadFile(filepath.Join(source, name+".json"))
		require.NoError(t, err)
		var receipt realAssessmentArtifact
		require.NoError(t, json.Unmarshal(original, &receipt))
		diagnostic, err := os.ReadFile(filepath.Join(source, name+"-model-response.json"))
		require.NoError(t, err)
		var raw struct {
			RequestHash string            `json:"request_hash"`
			Response    generation.Result `json:"untrusted_response"`
		}
		require.NoError(t, json.Unmarshal(diagnostic, &raw))
		require.Equal(t, receipt.Result.RequestHash, raw.RequestHash)
		replayed, err := decodeJudgmentResponse(*receipt.Input, raw.Response.Output)
		require.NoError(t, err)
		require.Equal(t, receipt.Result.Judgments, replayed.Judgments)
		require.Equal(t, receipt.Result.Decisions, replayed.Decisions)
		receipt.Result.Feedback = replayed.Feedback
		require.NoError(t, receipt.Result.ValidateFeedback(*receipt.Input))
		preview, err := NewGeneratorWithOptions(&capturedPort{}, receipt.Result.Provider, receipt.Result.Model, Options{MaxOutputTokens: 6000}).Preview(*receipt.Input)
		require.NoError(t, err)
		require.Equal(t, receipt.Result.RequestHash, preview.RequestHash)
		receipt.ReviewStatus = "offline_projection_of_real_response_not_a_new_provider_call"
		data, err := json.MarshalIndent(struct {
			realAssessmentArtifact
			OriginalSHA string `json:"original_receipt_sha256"`
			ResponseSHA string `json:"original_response_sha256"`
		}{receipt, digest(string(original)), digest(string(diagnostic))}, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(output, "receipts", name+".json"), data, 0600))
		proposal := CorrectionInput{ID: uuid.New(), PreviousHash: mastery.AssessmentFeedbackHash(*receipt.Result.Feedback)}
		if name == "correct-method-selection" {
			proposal.Reason = "明确文字自称运行通过不构成证据，保持未知，不添加无关来源。"
			for _, item := range receipt.Result.Feedback.Criteria {
				if item.ID == "runtime" {
					item.Reason = "文字自称运行通过；本次没有同一作答的实际运行记录，不能据此判定运行成功。"
					proposal.Changes = []mastery.CriterionAssessment{item}
				}
			}
		} else {
			proposal.Reason = "原下一步提示直接透露实现步骤，改为先让学习者定位选择错误。"
			f := receipt.Result.Feedback
			proposal.Findings = &mastery.AssessmentFindingCorrection{CorrectPoints: f.CorrectPoints, MissingPoints: f.MissingPoints, Misconceptions: f.Misconceptions, NextHint: mastery.AssessmentFinding{Text: "假设列表里有两种不同方法，而请求的是第二种：目前这段代码会选中哪一项？选中前还应核对什么？", EvidenceIDs: []string{"gin-method-tree-identity"}}, Remediation: f.Remediation}
		}
		correction := mastery.AssessmentCorrection{ID: proposal.ID.String(), ReviewerID: "automated-proposal-not-human", CorrectedAt: time.Now().UTC(), InputHash: receipt.Result.InputHash, PreviousHash: proposal.PreviousHash, Reason: proposal.Reason, Changes: proposal.Changes, Findings: proposal.Findings}
		effective, err := mastery.ReplayAssessmentCorrections(*receipt.Input, *receipt.Result.Feedback, []mastery.AssessmentCorrection{correction})
		require.NoError(t, err)
		draft, err := json.MarshalIndent(map[string]any{"origin": "automated_correction_proposal", "status": "pending_user_review", "original_sha256": digest(string(data)), "proposal": proposal, "proposed_effective_hash": mastery.AssessmentFeedbackHash(effective), "provider_calls": 0, "human_reviews": 0, "learning_record_writes": 0}, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(output, "proposals", name+".json"), draft, 0600))
		t.Logf("case=%s current_projection_valid=true proposal_valid=true provider_calls=0", name)
	}
}
