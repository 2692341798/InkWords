package mastery

import (
	"fmt"
	"strings"
	"time"
)

// AssessmentCodeContractVersion adds immutable learner files and file-bound quotes.
const AssessmentCodeContractVersion = "inkwords.mastery-assessment.v2"

// AssessmentCodeRuntimeContractVersion adds a matching, completed learner
// VerificationRun projection without changing existing v1/v2 request bytes.
const AssessmentCodeRuntimeContractVersion = "inkwords.mastery-assessment.v3"

// ContractVersion preserves legacy identities while versioning new task references.
func (input AssessmentInput) ContractVersion() string {
	if input.DecisionPolicy == AssessmentJudgmentPolicy {
		return AssessmentJudgmentContractVersion
	}
	if input.DecisionPolicy != "" {
		return AssessmentDecisionContractVersion
	}
	if input.TaskReference != nil {
		return AssessmentReferenceContractVersion
	}
	if input.LearnerArtifact != nil && input.ArtifactHash != "" {
		return AssessmentCodeRuntimeContractVersion
	}
	if input.LearnerArtifact != nil {
		return AssessmentCodeContractVersion
	}
	return AssessmentContractVersion
}

func assessmentLearnerMatchesRecord(input AssessmentInput, record AttemptRecord, objective Objective) bool {
	if ref := input.TaskReference; ref != nil && (ref.TaskID != record.PracticeTaskID || ref.PracticeContentHash != record.PracticeContentHash) {
		return false
	}
	code := input.LearnerArtifact
	if code == nil {
		return record.LearnerArtifactHash == ""
	}
	return code.SnapshotHash == record.LearnerArtifactHash && code.WorkspaceID == objective.WorkspaceID.String() && record.PracticeSessionID != nil && code.SessionID == record.PracticeSessionID.String() && code.TaskID == record.PracticeTaskID && code.PracticeContentHash == record.PracticeContentHash && code.SubmittedAt.Equal(record.AttemptedAt.UTC().Truncate(time.Microsecond))
}

func (input AssessmentInput) validateLearnerArtifact() error {
	code := input.LearnerArtifact
	if code == nil {
		return nil
	}
	if code.Validate() != nil || code.AttemptID != input.AttemptID || code.ObjectiveID != input.ObjectiveID || code.RevisionID != input.RevisionID || string(code.Skill) != string(input.Skill) {
		return fmt.Errorf("invalid assessed learner snapshot")
	}
	// Learner runtime authority is the immutable submitted snapshot identity.
	// The execution-tree hash remains a separate fact inside the run projection.
	if input.ArtifactHash != "" && input.ArtifactHash != code.SnapshotHash {
		return fmt.Errorf("learner runtime evidence belongs to another snapshot")
	}
	return nil
}

func (input AssessmentInput) validAnswerQuote(item CriterionAssessment) bool {
	text := input.Answer
	if item.AnswerPath != "" {
		if input.LearnerArtifact == nil {
			return false
		}
		found := false
		for _, file := range input.LearnerArtifact.Files {
			if file.Path == item.AnswerPath {
				text, found = file.Content, true
				break
			}
		}
		if !found || !boundedText(item.AnswerQuote, 20000) {
			return false
		}
	}
	return len(item.AnswerQuote) <= len(text) && (item.AnswerQuote == "" || strings.Contains(text, item.AnswerQuote)) && (item.Score == nil || *item.Score == 0 || strings.TrimSpace(item.AnswerQuote) != "")
}

// Missing runtime is a fact of the frozen input, not a claim sourced from a
// technical excerpt. Only new code judgments may leave that unknown uncited.
func (input AssessmentInput) unavailableRuntimeWithoutRefs(criterion AssessmentCriterion, item CriterionAssessment) bool {
	if input.DecisionPolicy != AssessmentJudgmentPolicy || input.LearnerArtifact == nil || !criterion.RequiresRuntime || item.Score != nil || item.EvidenceIDs == nil || len(item.EvidenceIDs) != 0 {
		return false
	}
	for _, evidence := range input.Evidence {
		if evidence.Kind == "runtime" {
			return false
		}
	}
	return true
}
