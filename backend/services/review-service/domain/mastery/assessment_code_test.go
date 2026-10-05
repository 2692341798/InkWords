package mastery

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	textbook "inkwords-backend/shared/kernel/textbook"
)

func TestAssessmentLegacyIdentityCheckpoint(t *testing.T) {
	input, feedback := assessmentFixture(t, Explain)
	require.Equal(t, "sha256:d6b4bbb35c2490948a45e812c4dff0294fdf087de6a814f23c52c1f834065052", AssessmentInputHash(input))
	require.Equal(t, "sha256:c36404603e8545ed3b0b866203ad27c6c2a19ca988ce8927a5580bf2b99566c0", AssessmentFeedbackHash(feedback))
}

func TestAssessmentCodeQuotesBindExactFileAndCannotCertifyExecution(t *testing.T) {
	input, feedback := assessmentFixture(t, Reproduce)
	code := assessmentCodeArtifact(t)
	input.AttemptID, input.ObjectiveID, input.RevisionID = code.AttemptID, code.ObjectiveID, code.RevisionID
	input.LearnerArtifact = code
	require.NoError(t, input.Validate())
	require.Equal(t, "inkwords.mastery-assessment.v2", input.ContractVersion())
	feedback.Criteria[0].AnswerPath = "router.go"
	feedback.Criteria[0].AnswerQuote = "func Match() bool { return true }"
	require.NoError(t, feedback.Validate(input))
	feedback.Criteria[0].AnswerPath = "other.go"
	require.ErrorContains(t, feedback.Validate(input), "quote")
	feedback.Criteria[0].AnswerPath = "router.go"
	feedback.Criteria[0].AnswerQuote = "func Match() bool { return false }"
	require.ErrorContains(t, feedback.Validate(input), "quote")
	feedback.Criteria[0].AnswerQuote = "func Match() bool { return true }"
	score := 4
	feedback.Criteria[2].Score = &score
	require.ErrorContains(t, feedback.Validate(input), "runtime")
	input.ArtifactHash = assessmentDigest("textbook reference implementation")
	require.ErrorContains(t, input.Validate(), "learner runtime")
	input.ArtifactHash = ""
	before := AssessmentInputHash(input)
	input.LearnerArtifact.Files[0].Content += "// changed"
	require.Error(t, input.Validate())
	require.NotEqual(t, before, AssessmentInputHash(input))
}

func TestAssessmentCodeCorrectionPreservesPathAndRejectsCrossFileQuotes(t *testing.T) {
	input, feedback := assessmentFixture(t, Reproduce)
	code := assessmentCodeArtifact(t)
	input.AttemptID, input.ObjectiveID, input.RevisionID, input.LearnerArtifact = code.AttemptID, code.ObjectiveID, code.RevisionID, code
	change := feedback.Criteria[0]
	change.AnswerPath, change.AnswerQuote = "router.go", "return true"
	correction := AssessmentCorrection{ID: "correction-1", ReviewerID: code.WorkspaceID, CorrectedAt: time.Now(), Reason: "按原文件核对", InputHash: AssessmentInputHash(input), PreviousHash: AssessmentFeedbackHash(feedback), Changes: []CriterionAssessment{change}}
	effective, err := ReplayAssessmentCorrections(input, feedback, []AssessmentCorrection{correction})
	require.NoError(t, err)
	require.Equal(t, "router.go", effective.Criteria[0].AnswerPath)
	require.Empty(t, feedback.Criteria[0].AnswerPath)
	textChange := effective.Criteria[0]
	textChange.AnswerPath, textChange.AnswerQuote = "", input.Answer
	textCorrection := AssessmentCorrection{ID: "correction-2", ReviewerID: code.WorkspaceID, CorrectedAt: correction.CorrectedAt.Add(time.Second), Reason: "改为引用文字作答", InputHash: AssessmentInputHash(input), PreviousHash: AssessmentFeedbackHash(effective), Changes: []CriterionAssessment{textChange}}
	textFeedback, err := ReplayAssessmentCorrections(input, feedback, []AssessmentCorrection{correction, textCorrection})
	require.NoError(t, err)
	require.Empty(t, textFeedback.Criteria[0].AnswerPath)
	correction.Changes[0].AnswerPath = "missing.go"
	_, err = ReplayAssessmentCorrections(input, feedback, []AssessmentCorrection{correction})
	require.ErrorContains(t, err, "quote")
	input.LearnerArtifact = nil
	feedback.Criteria[0].AnswerPath = "router.go"
	require.ErrorContains(t, feedback.Validate(input), "quote")
}

func assessmentCodeArtifact(t *testing.T) *textbook.LearnerArtifact {
	t.Helper()
	artifact := &textbook.LearnerArtifact{Format: textbook.LearnerArtifactFormat, WorkspaceID: uuid.NewString(), ObjectiveID: uuid.NewString(), AttemptID: uuid.NewString(), SessionID: uuid.NewString(), RevisionID: uuid.NewString(), TaskID: "reproduce", PracticeContentHash: assessmentDigest("practice"), Skill: textbook.LearningTaskMode(Reproduce), SubmittedAt: time.Unix(100, 0).UTC(), Files: []textbook.LearnerCodeFile{{Path: "router.go", Content: "package router\r\nfunc Match() bool { return true }\r\n"}}}
	var err error
	artifact.SnapshotHash, err = textbook.LearnerArtifactHash(*artifact)
	require.NoError(t, err)
	return artifact
}
