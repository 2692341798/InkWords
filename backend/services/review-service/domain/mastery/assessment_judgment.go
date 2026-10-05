package mastery

import (
	"fmt"
	"strings"
)

// AssessmentJudgmentPolicy projects grading prose from a single model judgment
// per frozen criterion instead of asking for independent, conflicting summaries.
const AssessmentJudgmentPolicy = "inkwords.criterion-coverage.v2"

// AssessmentJudgmentContractVersion preserves v1-v5 while changing new provider
// output to a canonical judgment ledger with deterministic feedback projections.
const AssessmentJudgmentContractVersion = "inkwords.mastery-assessment.v6"

// AssessmentJudgment is the provider's original judgment. Requirement text is
// resolved by criterion ID from the frozen input, never supplied by the model.
type AssessmentJudgment struct {
	CriterionID   string                  `json:"criterion_id"`
	Score         *int                    `json:"score"`
	Status        string                  `json:"status"`
	AnswerQuote   string                  `json:"answer_quote"`
	AnswerPath    string                  `json:"answer_path,omitempty"`
	AnswerSpan    *AssessmentCodeSpan     `json:"answer_span,omitempty"`
	EvidenceIDs   []string                `json:"evidence_ids"`
	Gaps          []AssessmentJudgmentGap `json:"gaps"`
	UnknownReason string                  `json:"unknown_reason"`
}

// AssessmentJudgmentGap is the sole source of an omission or misconception in
// derived feedback. Its requirement quote must belong to this exact criterion.
type AssessmentJudgmentGap struct {
	Kind             string `json:"kind"`
	RequirementQuote string `json:"requirement_quote"`
	Reason           string `json:"reason"`
}

// ProjectAssessmentJudgments derives all score reasons and grading finding
// lists from the same judgment ledger. Advice remains source-bound, and user
// corrections remain separate immutable records applied after this projection.
func ProjectAssessmentJudgments(input AssessmentInput, judgments []AssessmentJudgment, hint AssessmentFinding, remediation []AssessmentFinding) (AssessmentFeedback, []AssessmentDecision, error) {
	hint.EvidenceIDs = append([]string(nil), hint.EvidenceIDs...)
	advice := make([]AssessmentFinding, len(remediation))
	for index, finding := range remediation {
		advice[index] = AssessmentFinding{Text: finding.Text, EvidenceIDs: append([]string(nil), finding.EvidenceIDs...)}
	}
	feedback := AssessmentFeedback{Criteria: []CriterionAssessment{}, CorrectPoints: []AssessmentFinding{}, MissingPoints: []AssessmentFinding{}, Misconceptions: []AssessmentFinding{}, NextHint: hint, Remediation: advice}
	if input.DecisionPolicy != AssessmentJudgmentPolicy || input.Validate() != nil {
		return feedback, nil, fmt.Errorf("invalid canonical assessment input")
	}
	if len(judgments) != len(input.Rubric) {
		return feedback, nil, &AssessmentDecisionError{Rule: "decision_coverage"}
	}
	byID := map[string]AssessmentJudgment{}
	for _, judgment := range judgments {
		if _, exists := byID[judgment.CriterionID]; exists {
			return feedback, nil, &AssessmentDecisionError{Rule: "decision_requirement"}
		}
		byID[judgment.CriterionID] = judgment
	}
	decisions := make([]AssessmentDecision, 0, len(input.Rubric))
	for _, criterion := range input.Rubric {
		judgment, found := byID[criterion.ID]
		if !found {
			return feedback, nil, &AssessmentDecisionError{Rule: "decision_requirement"}
		}
		if judgment.Gaps == nil || len(judgment.Gaps) > 5 {
			return feedback, nil, &AssessmentDecisionError{Rule: "decision_gaps", CriterionID: criterion.ID}
		}
		decision := AssessmentDecision{CriterionID: criterion.ID, Requirement: criterion.Description, Status: judgment.Status, Gaps: []AssessmentDecisionGap{}}
		quote, err := input.resolveJudgmentQuote(judgment)
		if err != nil {
			return feedback, nil, err
		}
		item := CriterionAssessment{ID: criterion.ID, Score: judgment.Score, AnswerQuote: quote, AnswerPath: judgment.AnswerPath, EvidenceIDs: append([]string{}, judgment.EvidenceIDs...)}
		if item.Score != nil {
			score := *item.Score
			item.Score = &score
		}
		reasons := []string{}
		for _, gap := range judgment.Gaps {
			if !boundedText(gap.Reason, 300) {
				return feedback, nil, &AssessmentDecisionError{Rule: "decision_gaps", CriterionID: criterion.ID}
			}
			decision.Gaps = append(decision.Gaps, AssessmentDecisionGap{RequirementQuote: gap.RequirementQuote, Reason: gap.Reason})
			reasons = append(reasons, gap.Reason)
			finding := AssessmentFinding{Text: gap.Reason, EvidenceIDs: append([]string(nil), item.EvidenceIDs...)}
			switch gap.Kind {
			case "omission":
				feedback.MissingPoints = append(feedback.MissingPoints, finding)
			case "misconception":
				feedback.Misconceptions = append(feedback.Misconceptions, finding)
			default:
				return feedback, nil, &AssessmentDecisionError{Rule: "decision_gap_kind", CriterionID: criterion.ID}
			}
		}
		if judgment.Status == "unknown" {
			if !boundedText(judgment.UnknownReason, 1000) {
				return feedback, nil, &AssessmentDecisionError{Rule: "decision_unknown_reason", CriterionID: criterion.ID}
			}
			item.Reason = judgment.UnknownReason
		} else {
			if judgment.UnknownReason != "" {
				return feedback, nil, &AssessmentDecisionError{Rule: "decision_unknown_reason", CriterionID: criterion.ID}
			}
			if judgment.Status == "satisfied" {
				item.Reason = "所引作答满足本项要求。"
				if item.Score != nil && *item.Score == 4 {
					item.Reason = "所引作答完整满足本项要求，且边界清楚。"
				}
				feedback.CorrectPoints = append(feedback.CorrectPoints, AssessmentFinding{Text: criterion.Description, EvidenceIDs: append([]string(nil), item.EvidenceIDs...)})
			} else {
				item.Reason = "本项尚未满足：" + strings.Join(reasons, "；")
			}
		}
		feedback.Criteria = append(feedback.Criteria, item)
		decisions = append(decisions, decision)
	}
	if err := ValidateAssessmentDecisions(input, feedback, decisions); err != nil {
		return AssessmentFeedback{}, nil, err
	}
	return feedback, decisions, nil
}
