package textbook

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func learnerExecutionFixture(t *testing.T) (LearnerVerificationReference, LearnerVerificationInput) {
	t.Helper()
	artifact, projection, runner := learnerVerificationFixture(t)
	plan, err := NewLearnerVerificationPlan(artifact, projection, runner)
	require.NoError(t, err)
	reference := LearnerVerificationReference{Format: LearnerVerificationReferenceFormat, RunID: uuid.NewString(), WorkspaceID: plan.WorkspaceID, ObjectiveID: plan.ObjectiveID, AttemptID: plan.AttemptID, InputHash: plan.InputHash, ClaimToken: strings.Repeat("A", 43)}
	input := LearnerVerificationInput{Format: LearnerVerificationInputFormat, Plan: plan, Artifact: artifact, Task: projection.PracticeSet.Tasks[2]}
	return reference, input
}

func TestLearnerVerificationInputRejectsCommandsPathsAndRebinding(t *testing.T) {
	reference, input := learnerExecutionFixture(t)
	require.NoError(t, input.ValidateFor(reference))

	changedReference := reference
	changedReference.AttemptID = uuid.NewString()
	require.Error(t, input.ValidateFor(changedReference))

	changedInput := input
	changedInput.Artifact.Files[0].Content += "// changed"
	require.Error(t, changedInput.ValidateFor(reference))

	changedInput = input
	changedInput.Task.Rubric[0].Description += " changed"
	require.Error(t, changedInput.ValidateFor(reference))
}

func TestLearnerVerificationCapabilityKeepsScopeSeparateFromRuntimeAvailability(t *testing.T) {
	_, input := learnerExecutionFixture(t)
	available := LearnerVerificationCapability{Format: LearnerVerificationCapabilityFormat, Accepted: true, Available: true, Profile: LearnerGoTestProfile, Runner: &input.Plan.Runner}
	require.NoError(t, available.Validate())
	unavailable := available
	unavailable.Available, unavailable.Reason = false, "namespace preflight failed"
	require.NoError(t, unavailable.Validate())
	unavailable.Accepted = false
	require.Error(t, unavailable.Validate())
}

func TestLearnerVerificationReportSeparatesUnavailableFailureAndSuccess(t *testing.T) {
	reference, input := learnerExecutionFixture(t)
	now := time.Now().UTC()
	unavailable := LearnerVerificationReport{Format: LearnerVerificationReportFormat, RunID: reference.RunID, InputHash: reference.InputHash, SnapshotHash: input.Plan.SnapshotHash, Runner: input.Plan.Runner, Profile: input.Plan.Profile, Policy: input.Plan.Policy, Status: LearnerVerificationUnavailable, CompletedAt: now, Reason: "namespace preflight failed"}
	require.NoError(t, unavailable.ValidateFor(reference, input.Plan))

	started, tree, failedCode := now.Add(-time.Second), "sha256:"+strings.Repeat("b", 64), 1
	failed := LearnerVerificationReport{Format: LearnerVerificationReportFormat, RunID: reference.RunID, InputHash: reference.InputHash, SnapshotHash: input.Plan.SnapshotHash, ExecutionTreeHash: tree, Runner: input.Plan.Runner, Profile: input.Plan.Profile, Policy: input.Plan.Policy, Status: LearnerVerificationFailed, ExecutionStarted: true, StartedAt: &started, CompletedAt: now, ExitCode: &failedCode, Output: "--- FAIL", OutputBytes: len("--- FAIL")}
	require.NoError(t, failed.ValidateFor(reference, input.Plan))

	zero := 0
	failed.Status, failed.ExitCode = LearnerVerificationPassed, &zero
	require.NoError(t, failed.ValidateFor(reference, input.Plan))
	failed.ExitCode = &failedCode
	require.Error(t, failed.ValidateFor(reference, input.Plan))
}

func TestLearnerVerificationReportRejectsPretendExecutionFacts(t *testing.T) {
	reference, input := learnerExecutionFixture(t)
	now := time.Now().UTC()
	code := 0
	report := LearnerVerificationReport{Format: LearnerVerificationReportFormat, RunID: reference.RunID, InputHash: reference.InputHash, SnapshotHash: input.Plan.SnapshotHash, Runner: input.Plan.Runner, Profile: input.Plan.Profile, Policy: input.Plan.Policy, Status: LearnerVerificationUnavailable, CompletedAt: now, Reason: "unavailable", ExitCode: &code}
	require.Error(t, report.ValidateFor(reference, input.Plan), "an unavailable sandbox cannot report an exit code")

	report.ExitCode = nil
	report.Output = strings.Repeat("x", MaxLearnerVerificationOutputBytes)
	report.OutputBytes = len(report.Output) + 1
	report.OutputTruncated = true
	require.NoError(t, report.ValidateFor(reference, input.Plan))
	report.OutputBytes = len(report.Output)
	require.Error(t, report.ValidateFor(reference, input.Plan))
}
