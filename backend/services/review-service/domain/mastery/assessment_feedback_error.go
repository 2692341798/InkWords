package mastery

// AssessmentFeedbackError exposes a stable rule and an optional frozen rubric
// identity. It never retains provider prose, invented IDs, answers or evidence.
type AssessmentFeedbackError struct {
	Rule        string
	CriterionID string
}

func (err *AssessmentFeedbackError) Error() string {
	switch err.Rule {
	case "feedback_sections":
		return "assessment criteria or feedback sections are incomplete"
	case "feedback_criteria":
		return "invalid assessment criteria"
	case "feedback_answer_quote":
		return "assessment answer quote is missing or invented"
	case "feedback_runtime_evidence":
		return "runtime score requires matching verification evidence"
	case "feedback_finding_count":
		return "too many assessment findings"
	case "feedback_finding_text":
		return "empty or oversized assessment finding"
	case "feedback_evidence_count":
		return "assessment finding requires evidence"
	case "feedback_evidence_identity":
		return "unknown or duplicate assessment evidence"
	default:
		return "invalid assessment feedback"
	}
}

func feedbackCriterionError(err error, id string) error {
	rule := &AssessmentFeedbackError{}
	if errors.As(err, &rule) {
		return &AssessmentFeedbackError{Rule: rule.Rule, CriterionID: id}
	}
	return err
}
