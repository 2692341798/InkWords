package mastery

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// AssessmentContractVersion binds grading and correction replay to one rubric contract.
const AssessmentContractVersion = "inkwords.mastery-assessment.v1"

// AssessmentCriterion is a frozen, task-specific scoring requirement. Runtime
// criteria require authoritative verification evidence, not a learner's claim.
type AssessmentCriterion = sharedtextbook.PracticeCriterion

// AssessmentEvidence contains selected approved-source excerpts or verified
// runtime observations. The application must resolve these from their owners;
// arbitrary client-provided text must never acquire runtime authority.
type AssessmentEvidence struct {
	ID                string `json:"id"`
	Kind              string `json:"kind"`
	ContentHash       string `json:"content_hash"`
	Excerpt           string `json:"excerpt"`
	VerificationRunID string `json:"verification_run_id,omitempty"`
	ArtifactHash      string `json:"artifact_hash,omitempty"`
}

// AssessmentInput binds a saved answer to a frozen task, rubric and narrow
// evidence window. No API key or mutable model request belongs in this value.
type AssessmentInput struct {
	AttemptID       string                          `json:"attempt_id"`
	ObjectiveID     string                          `json:"objective_id"`
	RevisionID      string                          `json:"revision_id"`
	Skill           Skill                           `json:"skill"`
	Prompt          string                          `json:"prompt"`
	Answer          string                          `json:"answer"`
	ArtifactHash    string                          `json:"artifact_hash,omitempty"`
	Rubric          []AssessmentCriterion           `json:"rubric"`
	Evidence        []AssessmentEvidence            `json:"evidence"`
	LearnerArtifact *sharedtextbook.LearnerArtifact `json:"learner_artifact,omitempty"`
	TaskReference   *AssessmentTaskReference        `json:"task_reference,omitempty"`
	DecisionPolicy  string                          `json:"decision_policy,omitempty"`
}

// Validate fails closed on missing scoring dimensions or unverifiable source identities.
func (input AssessmentInput) Validate() error {
	if !known(input.Skill) || !boundedText(input.AttemptID, 200) || !boundedText(input.ObjectiveID, 200) || !boundedText(input.RevisionID, 200) || !boundedText(input.Prompt, 8000) || !boundedText(input.Answer, 20000) || len(input.Rubric) > 20 || len(input.Evidence) == 0 || len(input.Evidence) > 40 {
		return fmt.Errorf("incomplete assessment input")
	}
	if input.ArtifactHash != "" && !validAssessmentHash(input.ArtifactHash) {
		return fmt.Errorf("invalid assessed artifact hash")
	}
	if err := input.validateLearnerArtifact(); err != nil {
		return err
	}
	if err := input.validateTaskReference(); err != nil {
		return err
	}
	if input.DecisionPolicy != "" && ((input.DecisionPolicy != AssessmentDecisionPolicy && input.DecisionPolicy != AssessmentJudgmentPolicy) || input.TaskReference == nil) {
		return fmt.Errorf("invalid assessment decision policy or missing frozen task")
	}
	criteria := map[string]AssessmentCriterion{}
	for _, criterion := range input.Rubric {
		if !boundedText(criterion.ID, 100) || !boundedText(criterion.Description, 2000) {
			return fmt.Errorf("invalid assessment criterion")
		}
		if _, exists := criteria[criterion.ID]; exists {
			return fmt.Errorf("duplicate assessment criterion")
		}
		criteria[criterion.ID] = criterion
	}
	for _, dimension := range assessmentDimensions(input.Skill) {
		criterion, exists := criteria[dimension.ID]
		if !exists || criterion.RequiresRuntime != dimension.RequiresRuntime {
			return fmt.Errorf("missing or altered assessment dimension: %s", dimension.ID)
		}
	}
	seen := map[string]bool{}
	sources := 0
	for _, evidence := range input.Evidence {
		if seen[evidence.ID] || !boundedText(evidence.ID, 200) || !boundedText(evidence.Excerpt, 20000) || evidence.ContentHash != assessmentDigest(evidence.Excerpt) {
			return fmt.Errorf("invalid assessment evidence identity or hash")
		}
		seen[evidence.ID] = true
		switch evidence.Kind {
		case "source":
			sources++
			if evidence.VerificationRunID != "" || evidence.ArtifactHash != "" {
				return fmt.Errorf("source evidence cannot claim runtime authority")
			}
		case "runtime":
			if !boundedText(evidence.VerificationRunID, 200) || !validAssessmentHash(input.ArtifactHash) || evidence.ArtifactHash != input.ArtifactHash {
				return fmt.Errorf("runtime evidence must match the assessed artifact")
			}
		default:
			return fmt.Errorf("unknown assessment evidence kind")
		}
	}
	if sources == 0 {
		return fmt.Errorf("assessment requires approved source evidence")
	}
	return nil
}

// CriterionAssessment uses null for an unverified score; zero means assessed
// and incorrect. Quotes must be exact substrings of the saved learner answer.
type CriterionAssessment struct {
	ID          string   `json:"id"`
	Score       *int     `json:"score"`
	Reason      string   `json:"reason"`
	AnswerQuote string   `json:"answer_quote"`
	EvidenceIDs []string `json:"evidence_ids"`
	AnswerPath  string   `json:"answer_path,omitempty"`
}

// AssessmentFinding keeps feedback attached to supplied evidence.
type AssessmentFinding struct {
	Text        string   `json:"text"`
	EvidenceIDs []string `json:"evidence_ids"`
}

// AssessmentFeedback is automated advice, never a human review or a mastery
// certificate. Scheduling remains a separate, explicitly applied decision.
type AssessmentFeedback struct {
	Criteria       []CriterionAssessment `json:"criteria"`
	CorrectPoints  []AssessmentFinding   `json:"correct_points"`
	MissingPoints  []AssessmentFinding   `json:"missing_points"`
	Misconceptions []AssessmentFinding   `json:"misconceptions"`
	NextHint       AssessmentFinding     `json:"next_hint"`
	Remediation    []AssessmentFinding   `json:"remediation"`
}

// Validate rejects fabricated quotes, missing criteria, and runtime scores
// supported only by source prose. A null score remains unknown in all modes.
func (feedback AssessmentFeedback) Validate(input AssessmentInput) error {
	if err := input.Validate(); err != nil {
		return err
	}
	if len(feedback.Criteria) != len(input.Rubric) || feedback.CorrectPoints == nil || feedback.MissingPoints == nil || feedback.Misconceptions == nil || len(feedback.Remediation) == 0 {
		return &AssessmentFeedbackError{Rule: "feedback_sections"}
	}
	evidence := map[string]AssessmentEvidence{}
	for _, item := range input.Evidence {
		evidence[item.ID] = item
	}
	criteria := map[string]AssessmentCriterion{}
	for _, item := range input.Rubric {
		criteria[item.ID] = item
	}
	seen := map[string]bool{}
	for _, item := range feedback.Criteria {
		criterion, exists := criteria[item.ID]
		if !exists || seen[item.ID] || !boundedText(item.Reason, 2000) || item.Score != nil && (*item.Score < 0 || *item.Score > 4) {
			return &AssessmentFeedbackError{Rule: "feedback_criteria"}
		}
		seen[item.ID] = true
		if !input.validAnswerQuote(item) {
			return &AssessmentFeedbackError{Rule: "feedback_answer_quote", CriterionID: criterion.ID}
		}
		if !input.unavailableRuntimeWithoutRefs(criterion, item) {
			if err := validateAssessmentRefs(item.EvidenceIDs, evidence); err != nil {
				return feedbackCriterionError(err, criterion.ID)
			}
		}
		if criterion.RequiresRuntime && item.Score != nil {
			hasRuntime := false
			for _, id := range item.EvidenceIDs {
				hasRuntime = hasRuntime || evidence[id].Kind == "runtime"
			}
			if !hasRuntime {
				return &AssessmentFeedbackError{Rule: "feedback_runtime_evidence", CriterionID: criterion.ID}
			}
		}
	}
	groups := [][]AssessmentFinding{feedback.CorrectPoints, feedback.MissingPoints, feedback.Misconceptions, feedback.Remediation, {feedback.NextHint}}
	for _, group := range groups {
		if len(group) > 20 {
			return &AssessmentFeedbackError{Rule: "feedback_finding_count"}
		}
		for _, finding := range group {
			if !boundedText(finding.Text, 2000) {
				return &AssessmentFeedbackError{Rule: "feedback_finding_text"}
			}
			if err := validateAssessmentRefs(finding.EvidenceIDs, evidence); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateAssessmentRefs(ids []string, evidence map[string]AssessmentEvidence) error {
	if len(ids) == 0 || len(ids) > len(evidence) {
		return &AssessmentFeedbackError{Rule: "feedback_evidence_count"}
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if _, exists := evidence[id]; !exists || seen[id] {
			return &AssessmentFeedbackError{Rule: "feedback_evidence_identity"}
		}
		seen[id] = true
	}
	return nil
}

func assessmentDimensions(skill Skill) []AssessmentCriterion {
	return sharedtextbook.PracticeRubricDimensions(sharedtextbook.LearningTaskMode(skill))
}

func boundedText(value string, limit int) bool {
	return strings.TrimSpace(value) != "" && utf8.ValidString(value) && utf8.RuneCountInString(value) <= limit && !strings.ContainsRune(value, 0)
}

func assessmentDigest(value string) string {
	hash := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(hash[:])
}

func validAssessmentHash(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != 71 {
		return false
	}
	_, err := hex.DecodeString(value[7:])
	return err == nil
}
