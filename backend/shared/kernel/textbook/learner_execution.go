package textbook

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var learnerClaimToken = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

const (
	LearnerVerificationCapabilityFormat = "inkwords.learner-verification-capability.v1"
	LearnerVerificationReferenceFormat  = "inkwords.learner-verification-reference.v1"
	LearnerVerificationInputFormat      = "inkwords.learner-verification-input.v1"
	LearnerVerificationReportFormat     = "inkwords.learner-verification-report.v1"
	MaxLearnerVerificationOutputBytes   = 1 << 20
)

// LearnerVerificationCapability is a read-only preflight result. Accepted says
// the product scope was approved; Available says this exact runtime passed its
// current isolation gate. Only Available permits a start request.
type LearnerVerificationCapability struct {
	Format    string                 `json:"format"`
	Accepted  bool                   `json:"accepted"`
	Available bool                   `json:"available"`
	Profile   string                 `json:"profile"`
	Runner    *LearnerRunnerIdentity `json:"runner,omitempty"`
	Reason    string                 `json:"reason,omitempty"`
}

func (capability LearnerVerificationCapability) Validate() error {
	if capability.Format != LearnerVerificationCapabilityFormat || !capability.Accepted || capability.Profile != LearnerGoTestProfile {
		return fmt.Errorf("learner verification capability is invalid")
	}
	if capability.Available {
		if capability.Runner == nil || capability.Runner.Validate() != nil || strings.TrimSpace(capability.Reason) != "" {
			return fmt.Errorf("available learner verification capability lacks a trusted runner")
		}
		return nil
	}
	if strings.TrimSpace(capability.Reason) == "" {
		return fmt.Errorf("unavailable learner verification capability needs a reason")
	}
	if capability.Runner != nil && capability.Runner.Validate() != nil {
		return fmt.Errorf("unavailable learner verification capability has an invalid runner")
	}
	return nil
}

type LearnerVerificationRunStatus string

const (
	LearnerVerificationPassed      LearnerVerificationRunStatus = "passed"
	LearnerVerificationFailed      LearnerVerificationRunStatus = "failed"
	LearnerVerificationTimedOut    LearnerVerificationRunStatus = "timed_out"
	LearnerVerificationCancelled   LearnerVerificationRunStatus = "cancelled"
	LearnerVerificationErrored     LearnerVerificationRunStatus = "runner_error"
	LearnerVerificationUnavailable LearnerVerificationRunStatus = "unavailable"
)

// LearnerVerificationReference is the only message accepted by the runner.
// It grants no path, source bytes, command, environment or execution result;
// the runner must resolve the frozen input again from review-service.
type LearnerVerificationReference struct {
	Format      string `json:"format"`
	RunID       string `json:"run_id"`
	WorkspaceID string `json:"workspace_id"`
	ObjectiveID string `json:"objective_id"`
	AttemptID   string `json:"attempt_id"`
	InputHash   string `json:"input_hash"`
	ClaimToken  string `json:"claim_token"`
}

func (reference LearnerVerificationReference) Validate() error {
	if err := reference.validateIdentity(); err != nil || !learnerClaimToken.MatchString(reference.ClaimToken) {
		return fmt.Errorf("learner verification reference is incomplete")
	}
	return nil
}

func (reference LearnerVerificationReference) validateIdentity() error {
	if reference.Format != LearnerVerificationReferenceFormat || !isFullSHA256Digest(reference.InputHash) {
		return fmt.Errorf("learner verification reference is incomplete")
	}
	for _, id := range []string{reference.RunID, reference.WorkspaceID, reference.ObjectiveID, reference.AttemptID} {
		parsed, err := uuid.Parse(id)
		if err != nil || parsed == uuid.Nil || parsed.String() != id {
			return fmt.Errorf("learner verification reference identity is invalid")
		}
	}
	return nil
}

// LearnerVerificationInput is returned only by the owner service after it has
// authorized the reference. It contains data, never a shell command or path.
type LearnerVerificationInput struct {
	Format   string                  `json:"format"`
	Plan     LearnerVerificationPlan `json:"plan"`
	Artifact LearnerArtifact         `json:"artifact"`
	Task     PracticeTask            `json:"task"`
}

func (input LearnerVerificationInput) ValidateFor(reference LearnerVerificationReference) error {
	if err := reference.validateIdentity(); err != nil {
		return err
	}
	if input.Format != LearnerVerificationInputFormat || input.Plan.WorkspaceID != reference.WorkspaceID || input.Plan.ObjectiveID != reference.ObjectiveID || input.Plan.AttemptID != reference.AttemptID || input.Plan.InputHash != reference.InputHash {
		return fmt.Errorf("learner verification input does not match its reference")
	}
	return input.Plan.ValidateForTask(input.Artifact, input.Plan.ChapterID, input.Task)
}

// LearnerVerificationReport records observable runner facts. A failed learner
// test is a completed execution; unavailable means the sandbox never started.
type LearnerVerificationReport struct {
	Format            string                       `json:"format"`
	RunID             string                       `json:"run_id"`
	InputHash         string                       `json:"input_hash"`
	SnapshotHash      string                       `json:"snapshot_hash"`
	ExecutionTreeHash string                       `json:"execution_tree_hash,omitempty"`
	Runner            LearnerRunnerIdentity        `json:"runner"`
	Profile           string                       `json:"profile"`
	Policy            LearnerExecutionPolicy       `json:"policy"`
	Status            LearnerVerificationRunStatus `json:"status"`
	ExecutionStarted  bool                         `json:"execution_started"`
	StartedAt         *time.Time                   `json:"started_at,omitempty"`
	CompletedAt       time.Time                    `json:"completed_at"`
	ExitCode          *int                         `json:"exit_code,omitempty"`
	Output            string                       `json:"output,omitempty"`
	OutputBytes       int                          `json:"output_bytes"`
	OutputTruncated   bool                         `json:"output_truncated"`
	Reason            string                       `json:"reason,omitempty"`
}

func (report LearnerVerificationReport) ValidateFor(reference LearnerVerificationReference, plan LearnerVerificationPlan) error {
	if err := reference.Validate(); err != nil {
		return err
	}
	if report.Format != LearnerVerificationReportFormat || report.RunID != reference.RunID || report.InputHash != reference.InputHash || report.InputHash != plan.InputHash || report.SnapshotHash != plan.SnapshotHash || report.Runner != plan.Runner || report.Profile != plan.Profile || !reflect.DeepEqual(report.Policy, plan.Policy) || !reflect.DeepEqual(plan.Policy, DefaultLearnerGoTestPolicy()) || report.CompletedAt.IsZero() || len(report.Output) > MaxLearnerVerificationOutputBytes || strings.ContainsRune(report.Output, 0) {
		return fmt.Errorf("learner verification report identity is invalid")
	}
	if report.OutputBytes < len(report.Output) || report.OutputTruncated && report.OutputBytes <= len(report.Output) || !report.OutputTruncated && report.OutputBytes != len(report.Output) {
		return fmt.Errorf("learner verification output accounting is invalid")
	}
	switch report.Status {
	case LearnerVerificationUnavailable:
		if report.ExecutionStarted || report.StartedAt != nil || report.ExitCode != nil || report.ExecutionTreeHash != "" || strings.TrimSpace(report.Reason) == "" {
			return fmt.Errorf("unavailable learner verification must prove no execution started")
		}
	case LearnerVerificationPassed, LearnerVerificationFailed:
		if !report.ExecutionStarted || report.StartedAt == nil || report.StartedAt.IsZero() || report.StartedAt.After(report.CompletedAt) || report.ExitCode == nil || !isFullSHA256Digest(report.ExecutionTreeHash) {
			return fmt.Errorf("completed learner verification requires execution facts")
		}
		if report.Status == LearnerVerificationPassed && *report.ExitCode != 0 || report.Status == LearnerVerificationFailed && *report.ExitCode == 0 {
			return fmt.Errorf("learner verification status contradicts exit code")
		}
	case LearnerVerificationTimedOut, LearnerVerificationCancelled, LearnerVerificationErrored:
		if !report.ExecutionStarted || report.StartedAt == nil || report.StartedAt.IsZero() || report.StartedAt.After(report.CompletedAt) || report.ExitCode != nil || !isFullSHA256Digest(report.ExecutionTreeHash) || strings.TrimSpace(report.Reason) == "" {
			return fmt.Errorf("interrupted learner verification requires bounded execution facts")
		}
	default:
		return fmt.Errorf("unknown learner verification status")
	}
	return nil
}
