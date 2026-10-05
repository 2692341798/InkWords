package learnerverification

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

var (
	ErrRunnerNotConfigured = errors.New("learner isolated executor is not configured")
	bearerCredential       = regexp.MustCompile(`(?i)(authorization\s*:\s*bearer\s+)[^\s,;]+`)
	namedCredential        = regexp.MustCompile(`(?i)\b((?:api[_-]?key|access[_-]?token|refresh[_-]?token|password|secret)\s*[:=]\s*)[^\s,;]+`)
)

type InputResolver interface {
	Resolve(ctx context.Context, reference sharedtextbook.LearnerVerificationReference) (sharedtextbook.LearnerVerificationInput, error)
}

type Workspace struct {
	RootDir  string
	TreeHash string
	Cleanup  func() error
}

type WorkspaceStager interface {
	Stage(ctx context.Context, artifact sharedtextbook.LearnerArtifact, plan sharedtextbook.LearnerVerificationPlan) (Workspace, error)
}

// Executor exposes one fixed profile. It cannot receive argv, environment,
// test filters, a submitted path or any network setting.
type Executor interface {
	ExecuteLearnerGoTest(ctx context.Context, rootDir string, timeout time.Duration) (exitCode int, output string, err error)
}

type Runner struct {
	Resolver InputResolver
	Stager   WorkspaceStager
	Executor Executor
	Identity sharedtextbook.LearnerRunnerIdentity
	Now      func() time.Time
}

// Verify resolves the source again from review-service, validates every bound
// identity, stages exact bytes, and invokes only the fixed learner Go profile.
func (runner Runner) Verify(ctx context.Context, reference sharedtextbook.LearnerVerificationReference) sharedtextbook.LearnerVerificationReport {
	now := runner.clock()
	base := sharedtextbook.LearnerVerificationReport{Format: sharedtextbook.LearnerVerificationReportFormat, RunID: reference.RunID, InputHash: reference.InputHash, Runner: runner.Identity, Profile: sharedtextbook.LearnerGoTestProfile, Policy: sharedtextbook.DefaultLearnerGoTestPolicy(), Status: sharedtextbook.LearnerVerificationUnavailable, CompletedAt: now}
	if err := reference.Validate(); err != nil {
		base.Reason = err.Error()
		return base
	}
	if runner.Resolver == nil {
		base.Reason = "learner input resolver is not configured"
		return base
	}
	input, err := runner.Resolver.Resolve(ctx, reference)
	if err != nil {
		base.Reason = "resolve frozen learner input: " + err.Error()
		return base
	}
	base.SnapshotHash = input.Plan.SnapshotHash
	base.Runner = input.Plan.Runner
	base.Profile = input.Plan.Profile
	base.Policy = input.Plan.Policy
	if err := input.ValidateFor(reference); err != nil {
		base.Reason = err.Error()
		return base
	}
	if runner.Identity != input.Plan.Runner {
		base.Reason = "learner runner identity does not match the frozen plan"
		return base
	}
	if runner.Stager == nil || runner.Executor == nil {
		base.Reason = ErrRunnerNotConfigured.Error()
		return base
	}
	workspace, err := runner.Stager.Stage(ctx, input.Artifact, input.Plan)
	if err != nil {
		base.Reason = "stage frozen learner input: " + err.Error()
		return base
	}
	if workspace.Cleanup != nil {
		defer func() { _ = workspace.Cleanup() }()
	}
	if !strings.HasPrefix(workspace.TreeHash, "sha256:") {
		base.Reason = "learner execution tree hash is invalid"
		return base
	}
	timeout := time.Duration(input.Plan.Policy.TimeoutMillis) * time.Millisecond
	started := runner.clock()
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	exitCode, output, executeErr := runner.Executor.ExecuteLearnerGoTest(execCtx, workspace.RootDir, timeout)
	execContextErr := execCtx.Err()
	cancel()
	base.ExecutionStarted = true
	base.StartedAt = &started
	base.ExecutionTreeHash = workspace.TreeHash
	base.CompletedAt = runner.clock()
	base.Output, base.OutputBytes, base.OutputTruncated = boundedRedactedOutput(output)
	if errors.Is(execContextErr, context.DeadlineExceeded) || errors.Is(executeErr, context.DeadlineExceeded) {
		base.Status = sharedtextbook.LearnerVerificationTimedOut
		base.Reason = "learner verification reached its time limit"
		return base
	}
	if errors.Is(execContextErr, context.Canceled) || errors.Is(executeErr, context.Canceled) {
		base.Status = sharedtextbook.LearnerVerificationCancelled
		base.Reason = "learner verification was cancelled"
		return base
	}
	if executeErr != nil || exitCode < 0 {
		base.Status = sharedtextbook.LearnerVerificationErrored
		base.Reason = "learner sandbox failed"
		if executeErr != nil {
			base.Reason += ": " + executeErr.Error()
		}
		return base
	}
	base.ExitCode = &exitCode
	if exitCode == 0 {
		base.Status = sharedtextbook.LearnerVerificationPassed
		return base
	}
	base.Status = sharedtextbook.LearnerVerificationFailed
	base.Reason = "learner checks exited with a non-zero status"
	return base
}

func (runner Runner) clock() time.Time {
	if runner.Now != nil {
		return runner.Now().UTC()
	}
	return time.Now().UTC()
}

func boundedRedactedOutput(output string) (string, int, bool) {
	output = bearerCredential.ReplaceAllString(output, "${1}[REDACTED]")
	output = namedCredential.ReplaceAllString(output, "${1}[REDACTED]")
	total := len(output)
	if total <= sharedtextbook.MaxLearnerVerificationOutputBytes {
		return output, total, false
	}
	output = output[:sharedtextbook.MaxLearnerVerificationOutputBytes]
	for !utf8.ValidString(output) {
		output = output[:len(output)-1]
	}
	return output, total, true
}
