package mastery

import "fmt"

// AssessmentReferenceContractVersion binds the authored answer key of the
// frozen practice task. Legacy inputs keep their original v1/v2/v3 identities.
const AssessmentReferenceContractVersion = "inkwords.mastery-assessment.v4"

// AssessmentTaskReference is an authored answer key, not a source excerpt,
// learner quote, exclusive correct solution, or proof that code was executed.
type AssessmentTaskReference struct {
	TaskID              string `json:"task_id"`
	PracticeContentHash string `json:"practice_content_hash"`
	ExpectedAnswer      string `json:"expected_answer"`
}

func (input AssessmentInput) validateTaskReference() error {
	ref := input.TaskReference
	if ref == nil {
		return nil
	}
	if !boundedText(ref.TaskID, 200) || !validAssessmentHash(ref.PracticeContentHash) || !boundedText(ref.ExpectedAnswer, 4000) {
		return fmt.Errorf("invalid frozen assessment task reference")
	}
	if input.LearnerArtifact != nil && (ref.TaskID != input.LearnerArtifact.TaskID || ref.PracticeContentHash != input.LearnerArtifact.PracticeContentHash) {
		return fmt.Errorf("assessment task reference does not match learner snapshot")
	}
	return nil
}
