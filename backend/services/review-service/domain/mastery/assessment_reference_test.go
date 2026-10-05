package mastery

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAssessmentReferenceBindsIdentityWithoutBecomingAnswerOrRuntimeEvidence(t *testing.T) {
	input, feedback := assessmentFixture(t, Reproduce)
	legacyHash := AssessmentInputHash(input)
	input.TaskReference = &AssessmentTaskReference{TaskID: "reproduce", PracticeContentHash: assessmentDigest("practice"), ExpectedAnswer: "运行已经通过，这是参考答案独有的断言。"}
	require.NoError(t, input.Validate())
	require.Equal(t, AssessmentReferenceContractVersion, input.ContractVersion())
	require.NotEqual(t, legacyHash, AssessmentInputHash(input))
	referenceHash := AssessmentInputHash(input)
	input.TaskReference.ExpectedAnswer += "修订"
	require.NotEqual(t, referenceHash, AssessmentInputHash(input))
	feedback.Criteria[0].AnswerQuote = input.TaskReference.ExpectedAnswer
	require.ErrorContains(t, feedback.Validate(input), "quote")
	feedback.Criteria[0].AnswerQuote = input.Answer
	for index, criterion := range input.Rubric {
		if criterion.RequiresRuntime {
			score := 4
			feedback.Criteria[index].Score = &score
		}
	}
	require.ErrorContains(t, feedback.Validate(input), "runtime")
}

func TestAssessmentReferenceRejectsMalformedAndCrossTaskBindings(t *testing.T) {
	for name, mutate := range map[string]func(*AssessmentTaskReference){
		"missing task":   func(ref *AssessmentTaskReference) { ref.TaskID = "" },
		"invalid hash":   func(ref *AssessmentTaskReference) { ref.PracticeContentHash = "wrong" },
		"empty key":      func(ref *AssessmentTaskReference) { ref.ExpectedAnswer = " " },
		"oversized key":  func(ref *AssessmentTaskReference) { ref.ExpectedAnswer = strings.Repeat("中", 4001) },
		"other task":     func(ref *AssessmentTaskReference) { ref.TaskID = "other" },
		"other snapshot": func(ref *AssessmentTaskReference) { ref.PracticeContentHash = assessmentDigest("other") },
	} {
		t.Run(name, func(t *testing.T) {
			input, _ := assessmentFixture(t, Reproduce)
			code := assessmentCodeArtifact(t)
			input.AttemptID, input.ObjectiveID, input.RevisionID, input.LearnerArtifact = code.AttemptID, code.ObjectiveID, code.RevisionID, code
			input.TaskReference = &AssessmentTaskReference{TaskID: code.TaskID, PracticeContentHash: code.PracticeContentHash, ExpectedAnswer: "满足任务约束的实现。"}
			require.NoError(t, input.Validate())
			mutate(input.TaskReference)
			require.Error(t, input.Validate())
		})
	}
}

func TestAssessmentReferenceMatchesSavedAttemptTaskAndContent(t *testing.T) {
	input, _ := assessmentFixture(t, Explain)
	input.TaskReference = &AssessmentTaskReference{TaskID: "explain", PracticeContentHash: assessmentDigest("practice"), ExpectedAnswer: "冻结答案。"}
	record := AttemptRecord{PracticeTaskID: "explain", PracticeContentHash: input.TaskReference.PracticeContentHash}
	require.True(t, assessmentLearnerMatchesRecord(input, record, Objective{}))
	record.PracticeTaskID = "another-task"
	require.False(t, assessmentLearnerMatchesRecord(input, record, Objective{}))
	record.PracticeTaskID, record.PracticeContentHash = "explain", assessmentDigest("another-revision")
	require.False(t, assessmentLearnerMatchesRecord(input, record, Objective{}))
}
