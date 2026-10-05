package learnerverification

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type fakeResolver struct {
	input sharedtextbook.LearnerVerificationInput
	err   error
}

func (resolver fakeResolver) Resolve(_ context.Context, _ sharedtextbook.LearnerVerificationReference) (sharedtextbook.LearnerVerificationInput, error) {
	return resolver.input, resolver.err
}

type fakeStager struct {
	workspace Workspace
	calls     int
}

func (stager *fakeStager) Stage(_ context.Context, _ sharedtextbook.LearnerArtifact, _ sharedtextbook.LearnerVerificationPlan) (Workspace, error) {
	stager.calls++
	return stager.workspace, nil
}

type fakeExecutor struct {
	exitCode int
	output   string
	err      error
	calls    int
}

func (executor *fakeExecutor) ExecuteLearnerGoTest(_ context.Context, _ string, _ time.Duration) (int, string, error) {
	executor.calls++
	return executor.exitCode, executor.output, executor.err
}

func TestRunnerReReadsFrozenInputAndRunsOnlyTheFixedProfile(t *testing.T) {
	reference, input := learnerRunnerFixture(t)
	cleaned := false
	stager := &fakeStager{workspace: Workspace{RootDir: "/trusted/ephemeral", TreeHash: digest('b'), Cleanup: func() error { cleaned = true; return nil }}}
	executor := &fakeExecutor{output: "ok"}
	now := time.Unix(200, 0).UTC()
	report := (Runner{Resolver: fakeResolver{input: input}, Stager: stager, Executor: executor, Identity: input.Plan.Runner, Now: func() time.Time { return now }}).Verify(context.Background(), reference)

	require.Equal(t, sharedtextbook.LearnerVerificationPassed, report.Status)
	require.Equal(t, 1, stager.calls)
	require.Equal(t, 1, executor.calls)
	require.True(t, cleaned)
	require.NoError(t, report.ValidateFor(reference, input.Plan))
}

func TestRunnerFailsClosedBeforeStagingForUnavailableOrReboundInput(t *testing.T) {
	reference, input := learnerRunnerFixture(t)
	stager := &fakeStager{workspace: Workspace{RootDir: "/trusted", TreeHash: digest('b')}}
	report := (Runner{Resolver: fakeResolver{input: input}, Stager: stager, Identity: input.Plan.Runner}).Verify(context.Background(), reference)
	require.Equal(t, sharedtextbook.LearnerVerificationUnavailable, report.Status)
	require.False(t, report.ExecutionStarted)
	require.Zero(t, stager.calls)

	input.Artifact.Files[0].Content += "// rebound"
	report = (Runner{Resolver: fakeResolver{input: input}, Stager: stager, Executor: &fakeExecutor{}, Identity: input.Plan.Runner}).Verify(context.Background(), reference)
	require.Equal(t, sharedtextbook.LearnerVerificationUnavailable, report.Status)
	require.Zero(t, stager.calls)
}

func TestRunnerSeparatesLearnerFailureTimeoutAndSandboxError(t *testing.T) {
	reference, input := learnerRunnerFixture(t)
	stager := &fakeStager{workspace: Workspace{RootDir: "/trusted", TreeHash: digest('b')}}
	for _, test := range []struct {
		executor *fakeExecutor
		status   sharedtextbook.LearnerVerificationRunStatus
	}{
		{executor: &fakeExecutor{exitCode: 1, output: "FAIL"}, status: sharedtextbook.LearnerVerificationFailed},
		{executor: &fakeExecutor{exitCode: -1, err: context.DeadlineExceeded}, status: sharedtextbook.LearnerVerificationTimedOut},
		{executor: &fakeExecutor{exitCode: -1, err: errors.New("namespace lost")}, status: sharedtextbook.LearnerVerificationErrored},
	} {
		report := (Runner{Resolver: fakeResolver{input: input}, Stager: stager, Executor: test.executor, Identity: input.Plan.Runner}).Verify(context.Background(), reference)
		require.Equal(t, test.status, report.Status)
		require.NoError(t, report.ValidateFor(reference, input.Plan))
	}
}

func TestRunnerRedactsAndBoundsOutput(t *testing.T) {
	reference, input := learnerRunnerFixture(t)
	output := "API_KEY=secret-value\n" + strings.Repeat("界", sharedtextbook.MaxLearnerVerificationOutputBytes)
	report := (Runner{Resolver: fakeResolver{input: input}, Stager: &fakeStager{workspace: Workspace{RootDir: "/trusted", TreeHash: digest('b')}}, Executor: &fakeExecutor{output: output}, Identity: input.Plan.Runner}).Verify(context.Background(), reference)
	require.NotContains(t, report.Output, "secret-value")
	require.LessOrEqual(t, len(report.Output), sharedtextbook.MaxLearnerVerificationOutputBytes)
	require.True(t, report.OutputTruncated)
	require.Greater(t, report.OutputBytes, len(report.Output))
	require.NoError(t, report.ValidateFor(reference, input.Plan))
}

func learnerRunnerFixture(t *testing.T) (sharedtextbook.LearnerVerificationReference, sharedtextbook.LearnerVerificationInput) {
	t.Helper()
	artifact := sharedtextbook.LearnerArtifact{Format: sharedtextbook.LearnerArtifactFormat, WorkspaceID: uuid.NewString(), ObjectiveID: uuid.NewString(), AttemptID: uuid.NewString(), SessionID: uuid.NewString(), RevisionID: uuid.NewString(), TaskID: "task", PracticeContentHash: digest('a'), Skill: sharedtextbook.LearningTaskReproduce, SubmittedAt: time.Unix(100, 0).UTC(), Files: []sharedtextbook.LearnerCodeFile{{Path: "main.go", Content: "package learner\n"}}}
	var err error
	artifact.SnapshotHash, err = sharedtextbook.LearnerArtifactHash(artifact)
	require.NoError(t, err)
	task := sharedtextbook.PracticeTask{ID: "task", Mode: sharedtextbook.LearningTaskReproduce, Prompt: "复现", Variation: "变式", ExpectedAnswer: "答案"}
	runner := sharedtextbook.LearnerRunnerIdentity{ImageDigest: digest('c'), ToolchainVersion: "go1.25.4", SandboxProfileDigest: sharedtextbook.LearnerSandboxProfileDigest}
	plan := planForTask(t, artifact, task, runner)
	reference := sharedtextbook.LearnerVerificationReference{Format: sharedtextbook.LearnerVerificationReferenceFormat, RunID: uuid.NewString(), WorkspaceID: artifact.WorkspaceID, ObjectiveID: artifact.ObjectiveID, AttemptID: artifact.AttemptID, InputHash: plan.InputHash, ClaimToken: strings.Repeat("A", 43)}
	return reference, sharedtextbook.LearnerVerificationInput{Format: sharedtextbook.LearnerVerificationInputFormat, Plan: plan, Artifact: artifact, Task: task}
}

func planForTask(t *testing.T, artifact sharedtextbook.LearnerArtifact, task sharedtextbook.PracticeTask, runner sharedtextbook.LearnerRunnerIdentity) sharedtextbook.LearnerVerificationPlan {
	t.Helper()
	filesHash, err := sharedtextbook.LearnerFilesHash(artifact.Files)
	require.NoError(t, err)
	derived := []sharedtextbook.LearnerCodeFile{{Path: "go.mod", Content: "module inkwords.local/learner\n\ngo 1.25\n"}}
	executionHash, err := sharedtextbook.LearnerFilesHash(append(append([]sharedtextbook.LearnerCodeFile(nil), artifact.Files...), derived...))
	require.NoError(t, err)
	plan := sharedtextbook.LearnerVerificationPlan{Format: sharedtextbook.LearnerVerificationPlanFormat, Profile: sharedtextbook.LearnerGoTestProfile, WorkspaceID: artifact.WorkspaceID, ObjectiveID: artifact.ObjectiveID, AttemptID: artifact.AttemptID, SessionID: artifact.SessionID, ChapterID: uuid.NewString(), RevisionID: artifact.RevisionID, TaskID: artifact.TaskID, Skill: artifact.Skill, PracticeContentHash: artifact.PracticeContentHash, SnapshotHash: artifact.SnapshotHash, FilesHash: filesHash, DerivedFiles: derived, ExecutionFilesHash: executionHash, TaskHash: hashJSON(t, task), Runner: runner, Policy: sharedtextbook.DefaultLearnerGoTestPolicy()}
	plan.InputHash = hashJSON(t, plan)
	return plan
}

func hashJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	return sharedtextbook.PracticeExcerptHash(string(encoded))
}

func digest(character byte) string { return "sha256:" + strings.Repeat(string(character), 64) }
