package mastery

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func assessmentFixture(t *testing.T, skill Skill) (AssessmentInput, AssessmentFeedback) {
	t.Helper()
	input := AssessmentInput{AttemptID: "attempt-1", ObjectiveID: "objective-1", RevisionID: "approved-1", Skill: skill, Answer: "先按方法选树，再按路径查找处理函数。", Prompt: "说明路由怎样匹配请求。", Evidence: []AssessmentEvidence{{ID: "source-1", Kind: "source", Excerpt: "请求先按方法选择树，再按路径查找处理函数。"}}}
	input.Evidence[0].ContentHash = assessmentDigest(input.Evidence[0].Excerpt)
	for _, dimension := range assessmentDimensions(skill) {
		input.Rubric = append(input.Rubric, AssessmentCriterion{ID: dimension.ID, Description: "结合路由匹配解释" + dimension.ID, RequiresRuntime: dimension.RequiresRuntime})
	}
	feedback := AssessmentFeedback{CorrectPoints: []AssessmentFinding{}, MissingPoints: []AssessmentFinding{}, Misconceptions: []AssessmentFinding{}, NextHint: AssessmentFinding{Text: "方法相同但路径不同会发生什么？", EvidenceIDs: []string{"source-1"}}, Remediation: []AssessmentFinding{{Text: "回看按方法选择树的源码说明。", EvidenceIDs: []string{"source-1"}}}}
	for _, criterion := range input.Rubric {
		score := 3
		item := CriterionAssessment{ID: criterion.ID, Score: &score, Reason: "回答说明了主要步骤。", AnswerQuote: input.Answer, EvidenceIDs: []string{"source-1"}}
		if criterion.RequiresRuntime {
			item.Score = nil
			item.Reason = "缺少本次工件的验证运行记录。"
		}
		feedback.Criteria = append(feedback.Criteria, item)
	}
	return input, feedback
}

func TestAssessmentBindsEveryCriterionToExistingEvidenceAndAnswer(t *testing.T) {
	input, feedback := assessmentFixture(t, Explain)
	require.NoError(t, input.Validate())
	require.NoError(t, feedback.Validate(input))
	feedback.Criteria[0].AnswerQuote = "模型编造的作答片段"
	require.ErrorContains(t, feedback.Validate(input), "answer quote")
	feedback.Criteria[0].AnswerQuote = input.Answer
	feedback.Criteria[0].EvidenceIDs = []string{"invented-source"}
	require.ErrorContains(t, feedback.Validate(input), "evidence")
}

func TestAssessmentCannotAwardRuntimeScoresFromProseOrAnotherArtifact(t *testing.T) {
	input, feedback := assessmentFixture(t, Reproduce)
	require.NoError(t, feedback.Validate(input))
	score := 4
	for index := range feedback.Criteria {
		if feedback.Criteria[index].ID == "runtime" {
			feedback.Criteria[index].Score = &score
		}
	}
	require.ErrorContains(t, feedback.Validate(input), "runtime")
	input.ArtifactHash = assessmentDigest("current artifact")
	input.Evidence = append(input.Evidence, AssessmentEvidence{ID: "run-1", Kind: "runtime", ContentHash: assessmentDigest("ok"), Excerpt: "ok", VerificationRunID: "verified-run-1", ArtifactHash: assessmentDigest("other artifact")})
	require.ErrorContains(t, input.Validate(), "artifact")
}

func TestAssessmentRejectsMissingDimensionsAndChangedEvidenceBytes(t *testing.T) {
	input, feedback := assessmentFixture(t, Explain)
	input.Evidence[0].Excerpt += "changed"
	require.ErrorContains(t, input.Validate(), "hash")
	input, feedback = assessmentFixture(t, Explain)
	feedback.Criteria = feedback.Criteria[1:]
	require.ErrorContains(t, feedback.Validate(input), "criteria")
	input.Rubric = input.Rubric[1:]
	require.ErrorContains(t, input.Validate(), "dimension")
}

func TestAssessmentCorrectionsAreReplayableAndDoNotOverwriteOriginal(t *testing.T) {
	input, feedback := assessmentFixture(t, Explain)
	before, err := json.Marshal(feedback)
	require.NoError(t, err)
	score := 2
	change := feedback.Criteria[0]
	change.Score = &score
	change.Reason = "人工发现没有说明方法不匹配的边界。"
	correction := AssessmentCorrection{ID: "correction-1", ReviewerID: "workspace-1", CorrectedAt: time.Unix(1, 0), Reason: "按冻结评分依据修正", InputHash: AssessmentInputHash(input), PreviousHash: AssessmentFeedbackHash(feedback), Changes: []CriterionAssessment{change}}
	corrected, err := ReplayAssessmentCorrections(input, feedback, []AssessmentCorrection{correction})
	require.NoError(t, err)
	require.Equal(t, 2, *corrected.Criteria[0].Score)
	after, err := json.Marshal(feedback)
	require.NoError(t, err)
	require.Equal(t, string(before), string(after))
	correction.PreviousHash = assessmentDigest("stale")
	_, err = ReplayAssessmentCorrections(input, feedback, []AssessmentCorrection{correction})
	require.ErrorContains(t, err, "stale")
}

func TestAssessmentCorrectionsCannotCrossAttemptsOrInventRuntimeVerification(t *testing.T) {
	input, feedback := assessmentFixture(t, Reproduce)
	score := 4
	change := feedback.Criteria[2]
	change.Score = &score
	correction := AssessmentCorrection{ID: "correction-1", ReviewerID: "workspace-1", CorrectedAt: time.Unix(1, 0), Reason: "人工填写修正理由", InputHash: AssessmentInputHash(input), PreviousHash: AssessmentFeedbackHash(feedback), Changes: []CriterionAssessment{change}}
	_, err := ReplayAssessmentCorrections(input, feedback, []AssessmentCorrection{correction})
	require.ErrorContains(t, err, "runtime")
	input.AttemptID = "other-attempt"
	_, err = ReplayAssessmentCorrections(input, feedback, []AssessmentCorrection{correction})
	require.ErrorContains(t, err, "another input")
}

func TestAssessmentAcceptsKnownRuntimeScoresOnlyForMatchingArtifactEvidence(t *testing.T) {
	input, feedback := assessmentFixture(t, Reproduce)
	input.ArtifactHash = assessmentDigest("the submitted teaching artifact")
	input.Evidence = append(input.Evidence, AssessmentEvidence{ID: "runtime-1", Kind: "runtime", ContentHash: assessmentDigest("test failed"), Excerpt: "test failed", VerificationRunID: "run-1", ArtifactHash: input.ArtifactHash})
	zero := 0
	for index := range feedback.Criteria {
		if input.Rubric[index].RequiresRuntime {
			feedback.Criteria[index].Score = &zero
			feedback.Criteria[index].EvidenceIDs = []string{"runtime-1"}
		}
	}
	require.NoError(t, feedback.Validate(input), "a verified failure can justify zero, rather than unknown")
}

func TestAssessmentRequiresCompleteRubricsForAllSixSkills(t *testing.T) {
	for _, skill := range Skills {
		t.Run(string(skill), func(t *testing.T) {
			input, feedback := assessmentFixture(t, skill)
			require.NoError(t, input.Validate())
			require.NoError(t, feedback.Validate(input))
			originalHash := AssessmentInputHash(input)
			input.Answer += "这是新的作答。"
			require.NotEqual(t, originalHash, AssessmentInputHash(input))
		})
	}
}
