package masteryassessment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"inkwords-backend/services/review-service/domain/mastery"
)

// This offline review creates proposals for the fixed operator-authored
// evaluation answers. It neither records a human decision nor updates learning.
func TestFrozenRealFeedbackCorrectionProposals(t *testing.T) {
	source := os.Getenv("INKWORDS_ASSESSMENT_CORRECTION_REPLAY_SOURCE")
	if source == "" {
		t.Skip("requires explicit frozen evaluation receipts")
	}
	destination := os.Getenv("INKWORDS_ASSESSMENT_CORRECTION_REPLAY_OUTPUT")
	require.NotEmpty(t, destination)
	require.NoError(t, os.MkdirAll(destination, 0o700))
	require.NoError(t, os.Chmod(destination, 0o700))
	cases := []struct{ name, answer string }{
		{"complete-causal-answer", causalAcceptanceAnswer},
		{"incomplete-answer", incompleteAcceptanceAnswer},
		{"misconception-answer", misconceptionAcceptanceAnswer},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			path := filepath.Join(source, item.name+".json")
			original, err := os.ReadFile(path)
			require.NoError(t, err)
			var receipt realAssessmentArtifact
			require.NoError(t, json.Unmarshal(original, &receipt))
			require.Equal(t, item.name, receipt.Case)
			require.NotNil(t, receipt.Input)
			require.Equal(t, item.answer, receipt.Input.Answer)
			require.Equal(t, mastery.AssessmentInputHash(*receipt.Input), receipt.Result.InputHash)
			require.NoError(t, receipt.Result.ValidateFeedback(*receipt.Input))
			var draft mastery.AssessmentFeedback
			data, err := json.Marshal(receipt.Result.Feedback)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(data, &draft))
			proposal := CorrectionInput{ID: uuid.New(), PreviousHash: mastery.AssessmentFeedbackHash(draft), Changes: []mastery.CriterionAssessment{}}
			if item.name == "complete-causal-answer" {
				require.Len(t, draft.Remediation, 1)
				require.Contains(t, draft.Remediation[0].Text, "EnableHandleMethodNotAllowed")
				proposal.Reason = "自动复核提案：来源使用 HandleMethodNotAllowed，不能发明 EnableHandleMethodNotAllowed，也不能把方法树辅助函数写成未经来源证明的请求调用点。"
				draft.NextHint.EvidenceIDs = []string{"gin-request-dispatch", "gin-method-tree-identity"}
				draft.Remediation[0] = mastery.AssessmentFinding{Text: "分别追踪登记时按方法保存处理函数，以及请求分派时按方法选择路由树的过程。将 HandleMethodNotAllowed 分别设为 false 和 true，用只登记 GET /ping 的最小示例发送 POST /ping，对照来源解释 404/405；实际结果须运行后记录。", EvidenceIDs: []string{"gin-route-registration", "gin-request-dispatch", "gin-method-tree-identity"}}
			} else {
				for index, criterion := range draft.Criteria {
					if criterion.ID != "completeness" {
						continue
					}
					oldReason := strings.TrimPrefix(criterion.Reason, "本项尚未满足：")
					reason := "没有给出正确的注册与请求分派过程；关于未登记路径的错误结论另见边界项。还需补充登记保存处理函数、查询命中后执行等流程。"
					proposal.Reason = "自动复核提案：作答已谈到未登记路径，但结论错误；不应将说错的内容描述成完全没有提及。保留原分数，纠正理由和遗漏表述。"
					if item.name == "incomplete-answer" {
						require.NotNil(t, criterion.Score)
						require.Equal(t, 0, *criterion.Score)
						score := 1
						criterion.Score = &score
						reason = "作答已表达收到请求后查找路由、执行处理函数的粗略顺序，应保留这部分得分；仍缺注册阶段、按方法和路径分派，以及命中和未命中的具体结果。"
						proposal.Reason = "自动复核提案：作答已体现查找后执行的粗略顺序，完整性应按严重缺失而非完全未体现评 1 分；仍保留所有未覆盖要求。"
					}
					criterion.Reason = "本项尚未满足：" + reason
					draft.Criteria[index] = criterion
					proposal.Changes = append(proposal.Changes, criterion)
					found := false
					for j := range draft.MissingPoints {
						if draft.MissingPoints[j].Text == oldReason {
							draft.MissingPoints[j].Text = reason
							found = true
						}
					}
					require.True(t, found)
				}
				require.Len(t, proposal.Changes, 1)
			}
			proposal.Findings = &mastery.AssessmentFindingCorrection{CorrectPoints: draft.CorrectPoints, MissingPoints: draft.MissingPoints, Misconceptions: draft.Misconceptions, NextHint: draft.NextHint, Remediation: draft.Remediation}
			// The synthetic actor/time exist only inside this validation call.
			// They are never exported as a human-review or application record.
			replay := mastery.AssessmentCorrection{ID: proposal.ID.String(), ReviewerID: "offline-test-not-human", CorrectedAt: time.Unix(1, 0).UTC(), Reason: proposal.Reason, InputHash: receipt.Result.InputHash, PreviousHash: proposal.PreviousHash, Changes: proposal.Changes, Findings: proposal.Findings}
			effective, err := mastery.ReplayAssessmentCorrections(*receipt.Input, *receipt.Result.Feedback, []mastery.AssessmentCorrection{replay})
			require.NoError(t, err)
			require.Equal(t, draft, effective)
			require.NotEqual(t, proposal.PreviousHash, mastery.AssessmentFeedbackHash(effective))
			replay.PreviousHash = "sha256:stale"
			_, err = mastery.ReplayAssessmentCorrections(*receipt.Input, *receipt.Result.Feedback, []mastery.AssessmentCorrection{replay})
			require.Error(t, err)
			after, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, original, after)
			require.NoError(t, receipt.Result.ValidateFeedback(*receipt.Input))
			artifact := map[string]any{"origin": "automated_correction_proposal", "status": "pending_user_review", "case": item.name, "original_file": path, "original_sha256": digest(string(original)), "input_hash": receipt.Result.InputHash, "proposal": proposal, "proposed_effective_feedback": effective, "proposed_effective_hash": mastery.AssessmentFeedbackHash(effective), "domain_replay_verified": true, "provider_calls": 0, "human_reviews": 0, "learning_record_writes": 0}
			encoded, err := json.MarshalIndent(artifact, "", "  ")
			require.NoError(t, err)
			file, err := os.OpenFile(filepath.Join(destination, item.name+".json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
			require.NoError(t, err)
			_, err = file.Write(encoded)
			require.NoError(t, err)
			require.NoError(t, file.Close())
		})
	}
}
