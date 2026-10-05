package mastery

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AssessmentAlgorithmVersion versions rubric thresholds and their FSRS mapping.
const AssessmentAlgorithmVersion = "fsrs-v4-assessment-v1"

// AssessmentOutcome distinguishes an assessed failure from missing proof.
type AssessmentOutcome string

const (
	AssessmentPassed        AssessmentOutcome = "passed"
	AssessmentNeedsPractice AssessmentOutcome = "needs_practice"
	AssessmentUnverified    AssessmentOutcome = "unverified"
)

// AssessmentApplication is an explicit, immutable decision to use one exact
// feedback snapshot. Corrections after this decision require a new application.
type AssessmentApplication struct {
	ID               uuid.UUID          `json:"id"`
	ObjectiveID      uuid.UUID          `json:"objective_id"`
	AttemptID        uuid.UUID          `json:"attempt_id"`
	JobID            uuid.UUID          `json:"job_id"`
	PreviousID       *uuid.UUID         `json:"previous_id,omitempty"`
	SequenceNo       int                `json:"sequence_no"`
	AppliedBy        uuid.UUID          `json:"applied_by"`
	AppliedAt        time.Time          `json:"applied_at"`
	AlgorithmVersion string             `json:"algorithm_version"`
	InputHash        string             `json:"input_hash"`
	FeedbackHash     string             `json:"feedback_hash"`
	Input            AssessmentInput    `json:"input"`
	Feedback         AssessmentFeedback `json:"feedback"`
}

// ApplyAssessmentOutcome changes only a derived interpretation. It cannot
// invent independence, erase hints, or turn application time into practice time.
func ApplyAssessmentOutcome(attempt Attempt, input AssessmentInput, feedback AssessmentFeedback) (Attempt, error) {
	if err := feedback.Validate(input); err != nil {
		return Attempt{}, err
	}
	if attempt.Skill != input.Skill || attempt.Answer != input.Answer {
		return Attempt{}, fmt.Errorf("assessment does not match saved answer")
	}
	outcome := AssessmentPassed
	for _, criterion := range feedback.Criteria {
		if criterion.Score != nil && *criterion.Score < 3 {
			outcome = AssessmentNeedsPractice
			break
		}
		if criterion.Score == nil {
			outcome = AssessmentUnverified
		}
	}
	attempt.AssessmentOutcome = outcome
	return attempt, nil
}

// ReplayAssessmentApplications overlays explicit decisions at the original
// attempt positions, without appending practice or mutating learner records.
func ReplayAssessmentApplications(objective Objective, records []AttemptRecord, applications []AssessmentApplication) ([]Attempt, error) {
	attempts, err := attemptsFromRecords(records)
	if err != nil {
		return nil, err
	}
	indices := map[uuid.UUID]int{}
	for i, record := range records {
		if record.ObjectiveID != objective.ID {
			return nil, fmt.Errorf("attempt belongs to another objective")
		}
		if _, exists := indices[record.ID]; exists {
			return nil, fmt.Errorf("duplicate saved attempt")
		}
		indices[record.ID] = i
	}
	seen := map[uuid.UUID]bool{}
	latest := map[uuid.UUID]uuid.UUID{}
	var previousTime time.Time
	for i, event := range applications {
		index, exists := indices[event.AttemptID]
		if !exists || event.ID == uuid.Nil || seen[event.ID] || event.JobID == uuid.Nil || event.SequenceNo != i+1 || event.ObjectiveID != objective.ID || event.AppliedBy != objective.WorkspaceID || event.AppliedAt.IsZero() || event.AppliedAt.Before(previousTime) || event.AlgorithmVersion != AssessmentAlgorithmVersion {
			return nil, fmt.Errorf("invalid assessment application identity or sequence")
		}
		record := records[index]
		if !assessmentLearnerMatchesRecord(event.Input, record, objective) {
			return nil, fmt.Errorf("assessment application does not match saved learner files")
		}
		if objective.PracticeRevisionID == nil || record.PracticeSessionID == nil || *record.PracticeSessionID == uuid.Nil || record.PracticeTaskID == "" || record.PracticeContentHash != objective.PracticeContentHash || event.Input.RevisionID != objective.PracticeRevisionID.String() || event.Input.ObjectiveID != objective.ID.String() || event.Input.AttemptID != record.ID.String() || event.AppliedAt.Before(record.AttemptedAt) || event.InputHash != AssessmentInputHash(event.Input) || event.FeedbackHash != AssessmentFeedbackHash(event.Feedback) {
			return nil, fmt.Errorf("assessment application does not match frozen practice evidence")
		}
		previous := latest[event.AttemptID]
		if previous == uuid.Nil && event.PreviousID != nil || previous != uuid.Nil && (event.PreviousID == nil || *event.PreviousID != previous) {
			return nil, fmt.Errorf("broken assessment application chain")
		}
		attempts[index], err = ApplyAssessmentOutcome(attempts[index], event.Input, event.Feedback)
		if err != nil {
			return nil, err
		}
		seen[event.ID], latest[event.AttemptID], previousTime = true, event.ID, event.AppliedAt
	}
	return attempts, nil
}

func effectiveCorrect(attempt Attempt) bool {
	if attempt.AssessmentOutcome != "" {
		return attempt.AssessmentOutcome == AssessmentPassed
	}
	return attempt.Correct
}
