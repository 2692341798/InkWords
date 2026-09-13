package mastery

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

const maxAssessmentRuntimeOutputBytes = 8192

type learnerRuntimeProjection struct {
	Format              string                                      `json:"format"`
	RunID               string                                      `json:"run_id"`
	InputHash           string                                      `json:"input_hash"`
	SnapshotHash        string                                      `json:"snapshot_hash"`
	ExecutionTreeHash   string                                      `json:"execution_tree_hash"`
	Runner              sharedtextbook.LearnerRunnerIdentity        `json:"runner"`
	Profile             string                                      `json:"profile"`
	Policy              sharedtextbook.LearnerExecutionPolicy       `json:"policy"`
	Status              sharedtextbook.LearnerVerificationRunStatus `json:"status"`
	StartedAt           *time.Time                                  `json:"started_at"`
	CompletedAt         time.Time                                   `json:"completed_at"`
	ExitCode            *int                                        `json:"exit_code,omitempty"`
	Output              string                                      `json:"output,omitempty"`
	OutputBytes         int                                         `json:"output_bytes"`
	OutputTruncated     bool                                        `json:"output_truncated"`
	AssessmentTruncated bool                                        `json:"assessment_output_truncated"`
	Reason              string                                      `json:"reason,omitempty"`
}

// AttachLearnerVerification adds only a validated, execution-started report
// for the exact saved learner snapshot and frozen task. It never starts a run.
func AttachLearnerVerification(input AssessmentInput, resolved sharedtextbook.LearnerVerificationInput, report sharedtextbook.LearnerVerificationReport) (AssessmentInput, error) {
	if ref := input.TaskReference; ref != nil && (ref.TaskID != resolved.Task.ID || ref.ExpectedAnswer != resolved.Task.ExpectedAnswer || ref.PracticeContentHash != resolved.Artifact.PracticeContentHash) {
		return AssessmentInput{}, fmt.Errorf("learner verification does not match frozen answer key")
	}
	if input.LearnerArtifact == nil || !reflect.DeepEqual(*input.LearnerArtifact, resolved.Artifact) || resolved.Plan.ObjectiveID != input.ObjectiveID || resolved.Plan.AttemptID != input.AttemptID || resolved.Plan.RevisionID != input.RevisionID || resolved.Plan.SnapshotHash != input.LearnerArtifact.SnapshotHash || resolved.Task.ID != input.LearnerArtifact.TaskID || !reflect.DeepEqual(resolved.Task.Rubric, input.Rubric) {
		return AssessmentInput{}, fmt.Errorf("learner verification does not match assessment input")
	}
	reference := sharedtextbook.LearnerVerificationReference{Format: sharedtextbook.LearnerVerificationReferenceFormat, RunID: report.RunID, WorkspaceID: resolved.Plan.WorkspaceID, ObjectiveID: resolved.Plan.ObjectiveID, AttemptID: resolved.Plan.AttemptID, InputHash: resolved.Plan.InputHash, ClaimToken: strings.Repeat("A", 43)}
	if resolved.ValidateFor(reference) != nil || report.ValidateFor(reference, resolved.Plan) != nil {
		return AssessmentInput{}, fmt.Errorf("learner verification evidence is invalid")
	}
	switch report.Status {
	case sharedtextbook.LearnerVerificationPassed, sharedtextbook.LearnerVerificationFailed, sharedtextbook.LearnerVerificationTimedOut:
	default:
		return input, nil
	}
	output, truncated := truncateRuntimeOutput(report.Output, maxAssessmentRuntimeOutputBytes)
	projection := learnerRuntimeProjection{Format: sharedtextbook.LearnerVerificationReportFormat, RunID: report.RunID, InputHash: report.InputHash, SnapshotHash: report.SnapshotHash, ExecutionTreeHash: report.ExecutionTreeHash, Runner: report.Runner, Profile: report.Profile, Policy: report.Policy, Status: report.Status, StartedAt: report.StartedAt, CompletedAt: report.CompletedAt, ExitCode: report.ExitCode, Output: output, OutputBytes: report.OutputBytes, OutputTruncated: report.OutputTruncated, AssessmentTruncated: truncated, Reason: report.Reason}
	encoded, err := json.Marshal(projection)
	if err != nil {
		return AssessmentInput{}, err
	}
	input.ArtifactHash = report.SnapshotHash
	input.Evidence = append(input.Evidence, AssessmentEvidence{ID: "learner-runtime:" + report.RunID, Kind: "runtime", ContentHash: assessmentDigest(string(encoded)), Excerpt: string(encoded), VerificationRunID: report.RunID, ArtifactHash: report.SnapshotHash})
	if err := input.Validate(); err != nil {
		return AssessmentInput{}, err
	}
	return input, nil
}

func truncateRuntimeOutput(value string, limit int) (string, bool) {
	if len(value) <= limit {
		return value, false
	}
	value = value[:limit]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value, true
}
