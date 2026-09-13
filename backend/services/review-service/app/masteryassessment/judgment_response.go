package masteryassessment

import (
	"encoding/json"
	"io"
	"reflect"
	"strings"

	"inkwords-backend/services/review-service/domain/mastery"
)

func decodeJudgmentResponse(input mastery.AssessmentInput, output string) (Result, error) {
	result := Result{}
	var response struct {
		Judgments   []mastery.AssessmentJudgment `json:"judgments"`
		NextHint    mastery.AssessmentFinding    `json:"next_hint"`
		Remediation []mastery.AssessmentFinding  `json:"remediation"`
	}
	decoder := json.NewDecoder(strings.NewReader(output))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&response) != nil || decoder.Decode(new(any)) != io.EOF {
		return result, ErrInvalidFeedback
	}
	var presence struct {
		Judgments []map[string]json.RawMessage `json:"judgments"`
	}
	if json.Unmarshal([]byte(output), &presence) != nil {
		return result, ErrInvalidFeedback
	}
	for _, item := range presence.Judgments {
		for _, name := range []string{"criterion_id", "score", "status", "answer_quote", "evidence_ids", "gaps", "unknown_reason"} {
			value, found := item[name]
			if !found || name != "score" && string(value) == "null" {
				return result, ErrInvalidFeedback
			}
		}
		path, present := item["answer_path"]
		if input.LearnerArtifact != nil {
			if !present || string(path) == "null" {
				return result, ErrInvalidFeedback
			}
		} else if present {
			return result, ErrInvalidFeedback
		}
	}
	feedback, decisions, err := mastery.ProjectAssessmentJudgments(input, response.Judgments, response.NextHint, response.Remediation)
	if err != nil {
		return result, err
	}
	result.Feedback, result.Decisions, result.Judgments = &feedback, decisions, response.Judgments
	return result, nil
}

// ValidateFeedback replays the original model ledger for new contracts. User
// corrections are validated separately and cannot rewrite the original ledger.
func (result Result) ValidateFeedback(input mastery.AssessmentInput) error {
	if result.Feedback == nil || result.Rejection != nil {
		return ErrInvalidFeedback
	}
	if input.DecisionPolicy != mastery.AssessmentJudgmentPolicy {
		if result.Judgments != nil {
			return ErrInvalidFeedback
		}
		return mastery.ValidateAssessmentDecisions(input, *result.Feedback, result.Decisions)
	}
	projected, decisions, err := mastery.ProjectAssessmentJudgments(input, result.Judgments, result.Feedback.NextHint, result.Feedback.Remediation)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(projected, *result.Feedback) || !reflect.DeepEqual(decisions, result.Decisions) {
		return ErrInvalidFeedback
	}
	return nil
}
