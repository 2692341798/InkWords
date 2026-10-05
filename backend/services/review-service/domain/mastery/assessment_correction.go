package mastery

import (
	"encoding/json"
	"fmt"
	"time"
)

// AssessmentCorrection is an append-only human decision. Persistence must
// authenticate ReviewerID and compare PreviousHash in the same transaction.
type AssessmentCorrection struct {
	ID           string                       `json:"id"`
	ReviewerID   string                       `json:"reviewer_id"`
	CorrectedAt  time.Time                    `json:"corrected_at"`
	Reason       string                       `json:"reason"`
	InputHash    string                       `json:"input_hash"`
	PreviousHash string                       `json:"previous_hash"`
	Changes      []CriterionAssessment        `json:"changes"`
	Findings     *AssessmentFindingCorrection `json:"findings,omitempty"`
}

// AssessmentFindingCorrection replaces the five advice sections together. An
// explicit empty list clears a finding; missing sections fail validation.
// Omitting this value preserves old correction bytes and their replay behavior.
type AssessmentFindingCorrection struct {
	CorrectPoints  []AssessmentFinding `json:"correct_points"`
	MissingPoints  []AssessmentFinding `json:"missing_points"`
	Misconceptions []AssessmentFinding `json:"misconceptions"`
	NextHint       AssessmentFinding   `json:"next_hint"`
	Remediation    []AssessmentFinding `json:"remediation"`
}

// AssessmentInputHash binds feedback to the exact saved answer, rubric,
// evidence and task identity under this contract version.
func AssessmentInputHash(input AssessmentInput) string {
	encoded, _ := json.Marshal(input)
	return assessmentDigest(input.ContractVersion() + "\n" + string(encoded))
}

// AssessmentFeedbackHash identifies an exact effective view for a correction CAS.
func AssessmentFeedbackHash(feedback AssessmentFeedback) string {
	encoded, _ := json.Marshal(feedback)
	return assessmentDigest(string(encoded))
}

// ReplayAssessmentCorrections derives a view without mutating automated
// feedback. Human correction cannot manufacture a missing verification run.
func ReplayAssessmentCorrections(input AssessmentInput, original AssessmentFeedback, corrections []AssessmentCorrection) (AssessmentFeedback, error) {
	if err := original.Validate(input); err != nil {
		return AssessmentFeedback{}, err
	}
	encoded, _ := json.Marshal(original)
	var current AssessmentFeedback
	if err := json.Unmarshal(encoded, &current); err != nil {
		return AssessmentFeedback{}, err
	}
	seen := map[string]bool{}
	var previousTime time.Time
	for _, correction := range corrections {
		if !boundedText(correction.ID, 200) || seen[correction.ID] || !boundedText(correction.ReviewerID, 200) || correction.CorrectedAt.IsZero() || correction.CorrectedAt.Before(previousTime) || !boundedText(correction.Reason, 2000) || (len(correction.Changes) == 0 && correction.Findings == nil) {
			return AssessmentFeedback{}, fmt.Errorf("invalid assessment correction")
		}
		seen[correction.ID] = true
		previousTime = correction.CorrectedAt
		if correction.InputHash != AssessmentInputHash(input) {
			return AssessmentFeedback{}, fmt.Errorf("assessment correction belongs to another input")
		}
		if correction.PreviousHash != AssessmentFeedbackHash(current) {
			return AssessmentFeedback{}, fmt.Errorf("stale assessment correction")
		}
		changes := map[string]CriterionAssessment{}
		for _, change := range correction.Changes {
			if _, exists := changes[change.ID]; exists {
				return AssessmentFeedback{}, fmt.Errorf("duplicate corrected criterion")
			}
			changes[change.ID] = change
		}
		for index, criterion := range current.Criteria {
			if change, exists := changes[criterion.ID]; exists {
				// Copy nested values so the caller cannot mutate the derived
				// view later through the submitted correction object.
				data, _ := json.Marshal(change)
				var replacement CriterionAssessment
				if err := json.Unmarshal(data, &replacement); err != nil {
					return AssessmentFeedback{}, err
				}
				// Decode into a fresh value so an omitted empty answer_path
				// clears the old file when a correction cites text instead.
				current.Criteria[index] = replacement
				delete(changes, criterion.ID)
			}
		}
		if len(changes) != 0 {
			return AssessmentFeedback{}, fmt.Errorf("unknown corrected criterion")
		}
		if correction.Findings != nil {
			// Decode independently so caller-owned slices cannot change this view
			// after validation, and omitted fields cannot inherit older advice.
			data, err := json.Marshal(correction.Findings)
			if err != nil {
				return AssessmentFeedback{}, err
			}
			var findings AssessmentFindingCorrection
			if err := json.Unmarshal(data, &findings); err != nil {
				return AssessmentFeedback{}, err
			}
			current.CorrectPoints, current.MissingPoints = findings.CorrectPoints, findings.MissingPoints
			current.Misconceptions, current.NextHint, current.Remediation = findings.Misconceptions, findings.NextHint, findings.Remediation
		}
		if err := current.Validate(input); err != nil {
			return AssessmentFeedback{}, err
		}
	}
	return current, nil
}
