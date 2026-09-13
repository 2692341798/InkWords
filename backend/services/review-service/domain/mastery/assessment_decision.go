package mastery

import (
	"fmt"
	"strings"
)

// AssessmentDecisionPolicy requires explicit coverage of each frozen criterion
// before a model score may become valid automated advice.
const AssessmentDecisionPolicy = "inkwords.criterion-coverage.v1"

// AssessmentDecisionContractVersion adds criterion decisions to new model
// requests while retaining the identities of saved v1-v4 inputs.
const AssessmentDecisionContractVersion = "inkwords.mastery-assessment.v5"

// AssessmentDecision records the model's assessment of one exact requirement.
// It remains automated evidence, not a factual or human-review guarantee.
type AssessmentDecision struct {
	CriterionID string                  `json:"criterion_id"`
	Requirement string                  `json:"requirement"`
	Status      string                  `json:"status"`
	Gaps        []AssessmentDecisionGap `json:"gaps"`
}

// AssessmentDecisionGap anchors a claimed omission to the frozen rubric text.
// Source or reference-answer details cannot introduce a new requirement here.
type AssessmentDecisionGap struct {
	RequirementQuote string `json:"requirement_quote"`
	Reason           string `json:"reason"`
}

// AssessmentDecisionError identifies a bounded validation rule and, only when
// resolved from the frozen rubric, its criterion. It never contains model prose.
type AssessmentDecisionError struct {
	Rule        string
	CriterionID string
}

func (err *AssessmentDecisionError) Error() string { return err.Rule }

// ValidateAssessmentDecisions rejects missing coverage, altered requirements,
// and contradictions between structured judgments and numeric scores. It does
// not infer semantic truth from a valid schema or silently change model scores.
func ValidateAssessmentDecisions(input AssessmentInput, feedback AssessmentFeedback, decisions []AssessmentDecision) error {
	if err := feedback.Validate(input); err != nil {
		return err
	}
	if input.DecisionPolicy == "" {
		if decisions != nil {
			return fmt.Errorf("legacy assessment cannot acquire decision coverage")
		}
		return nil
	}
	if len(decisions) != len(input.Rubric) {
		return &AssessmentDecisionError{Rule: "decision_coverage"}
	}
	rubric := make(map[string]AssessmentCriterion, len(input.Rubric))
	items := make(map[string]CriterionAssessment, len(feedback.Criteria))
	for _, criterion := range input.Rubric {
		rubric[criterion.ID] = criterion
	}
	for _, criterion := range feedback.Criteria {
		items[criterion.ID] = criterion
	}
	seen := map[string]bool{}
	for _, decision := range decisions {
		criterion, found := rubric[decision.CriterionID]
		if !found || seen[decision.CriterionID] || decision.Requirement != criterion.Description {
			return &AssessmentDecisionError{Rule: "decision_requirement"}
		}
		seen[decision.CriterionID] = true
		if decision.Gaps == nil || len(decision.Gaps) > 10 {
			return &AssessmentDecisionError{Rule: "decision_gaps", CriterionID: criterion.ID}
		}
		for _, gap := range decision.Gaps {
			if !boundedText(gap.RequirementQuote, 2000) || !strings.Contains(criterion.Description, gap.RequirementQuote) || !boundedText(gap.Reason, 2000) {
				return &AssessmentDecisionError{Rule: "decision_gap_requirement", CriterionID: criterion.ID}
			}
		}
		score := items[decision.CriterionID].Score
		switch decision.Status {
		case "satisfied":
			if score == nil || *score < 3 || len(decision.Gaps) != 0 {
				return &AssessmentDecisionError{Rule: "decision_score", CriterionID: criterion.ID}
			}
		case "unsatisfied":
			if score == nil || *score > 2 || len(decision.Gaps) == 0 {
				return &AssessmentDecisionError{Rule: "decision_score", CriterionID: criterion.ID}
			}
		case "unknown":
			if score != nil || len(decision.Gaps) != 0 {
				return &AssessmentDecisionError{Rule: "decision_score", CriterionID: criterion.ID}
			}
		default:
			return &AssessmentDecisionError{Rule: "decision_status", CriterionID: criterion.ID}
		}
	}
	return nil
}
