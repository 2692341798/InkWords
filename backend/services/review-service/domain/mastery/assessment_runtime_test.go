package mastery

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func TestAttachLearnerVerificationBindsExactSnapshotAndProjectsBoundedRuntimeFacts(t *testing.T) {
	basis := practiceBasisFixture()
	var task sharedtextbook.PracticeTask
	for _, candidate := range basis.Learning.PracticeSet.Tasks {
		if candidate.Mode == sharedtextbook.LearningTaskReproduce {
			task = candidate
		}
	}
	objective, attempt := uuid.NewString(), uuid.NewString()
	artifact := sharedtextbook.LearnerArtifact{Format: sharedtextbook.LearnerArtifactFormat, WorkspaceID: basis.WorkspaceID.String(), ObjectiveID: objective, AttemptID: attempt, SessionID: uuid.NewString(), RevisionID: basis.Learning.RevisionID, TaskID: task.ID, PracticeContentHash: basis.Learning.ContentHash, Skill: task.Mode, SubmittedAt: time.Now().UTC(), Files: []sharedtextbook.LearnerCodeFile{{Path: "main.go", Content: "package main\n"}}}
	var err error
	artifact.SnapshotHash, err = sharedtextbook.LearnerArtifactHash(artifact)
	require.NoError(t, err)
	runner := sharedtextbook.LearnerRunnerIdentity{ImageDigest: assessmentDigest("runner"), ToolchainVersion: "go1.25.4", SandboxProfileDigest: sharedtextbook.LearnerSandboxProfileDigest}
	plan, err := sharedtextbook.NewLearnerVerificationPlan(artifact, basis.Learning, runner)
	require.NoError(t, err)
	resolved := sharedtextbook.LearnerVerificationInput{Format: sharedtextbook.LearnerVerificationInputFormat, Plan: plan, Artifact: artifact, Task: task}
	input, _ := assessmentFixture(t, Reproduce)
	input.AttemptID, input.ObjectiveID, input.RevisionID = attempt, objective, basis.Learning.RevisionID
	input.Prompt, input.Rubric, input.LearnerArtifact = task.Prompt+"\n变式："+task.Variation, task.Rubric, &artifact
	now, zero := time.Now().UTC(), 0
	output := strings.Repeat("测", 5000)
	report := sharedtextbook.LearnerVerificationReport{Format: sharedtextbook.LearnerVerificationReportFormat, RunID: uuid.NewString(), InputHash: plan.InputHash, SnapshotHash: plan.SnapshotHash, ExecutionTreeHash: assessmentDigest("tree"), Runner: runner, Profile: plan.Profile, Policy: plan.Policy, Status: sharedtextbook.LearnerVerificationPassed, ExecutionStarted: true, StartedAt: &now, CompletedAt: now, ExitCode: &zero, Output: output, OutputBytes: len(output)}

	attached, err := AttachLearnerVerification(input, resolved, report)
	require.NoError(t, err)
	require.Equal(t, AssessmentCodeRuntimeContractVersion, attached.ContractVersion())
	require.Equal(t, artifact.SnapshotHash, attached.ArtifactHash)
	require.Len(t, attached.Evidence, 2)
	require.Equal(t, "runtime", attached.Evidence[1].Kind)
	require.Equal(t, report.RunID, attached.Evidence[1].VerificationRunID)
	require.Contains(t, attached.Evidence[1].Excerpt, `"assessment_output_truncated":true`)
	require.LessOrEqual(t, len(attached.Evidence[1].Excerpt), 20000)
	input.TaskReference = &AssessmentTaskReference{TaskID: task.ID, PracticeContentHash: artifact.PracticeContentHash, ExpectedAnswer: task.ExpectedAnswer}
	withReference, err := AttachLearnerVerification(input, resolved, report)
	require.NoError(t, err)
	require.Equal(t, AssessmentReferenceContractVersion, withReference.ContractVersion())
	require.Equal(t, attached.Evidence, withReference.Evidence)
	input.TaskReference.ExpectedAnswer += "篡改"
	_, err = AttachLearnerVerification(input, resolved, report)
	require.ErrorContains(t, err, "frozen answer key")
	input.TaskReference.ExpectedAnswer = task.ExpectedAnswer

	changed := resolved
	changed.Artifact.Files = append([]sharedtextbook.LearnerCodeFile(nil), resolved.Artifact.Files...)
	changed.Artifact.Files[0].Content += "// changed\n"
	_, err = AttachLearnerVerification(input, changed, report)
	require.ErrorContains(t, err, "does not match")
	report.Status = sharedtextbook.LearnerVerificationCancelled
	report.ExitCode, report.Reason = nil, "cancelled"
	unchanged, err := AttachLearnerVerification(input, resolved, report)
	require.NoError(t, err)
	require.Empty(t, unchanged.ArtifactHash, "cancelled execution is not grading evidence")
}
