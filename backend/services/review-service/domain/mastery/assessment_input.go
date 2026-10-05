package mastery

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// AssessmentSource resolves source text from the core API without accepting client excerpts.
type AssessmentSource interface {
	LoadApprovedAssessmentEvidence(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (sharedtextbook.PracticeEvidenceProjection, error)
}

// PrepareAssessmentInput assembles a saved learner answer with its frozen task
// and authoritative source excerpts. It never infers runtime proof from answer text.
func (service *Service) PrepareAssessmentInput(ctx context.Context, workspaceID, objectiveID, attemptID uuid.UUID) (AssessmentInput, error) {
	var input AssessmentInput
	if service == nil || service.store == nil || workspaceID == uuid.Nil || objectiveID == uuid.Nil || attemptID == uuid.Nil {
		return input, fmt.Errorf("invalid assessment identity")
	}
	objective, err := service.store.GetObjective(ctx, workspaceID, objectiveID)
	if err != nil {
		return input, err
	}
	projection, err := objective.practiceProjection()
	if err != nil || projection == nil {
		return input, ErrPracticeMismatch
	}
	source, ok := service.practiceSource.(AssessmentSource)
	if !ok {
		return input, fmt.Errorf("approved assessment source unavailable")
	}
	attempt, err := service.store.GetAttempt(ctx, objectiveID, attemptID)
	if err != nil {
		return input, err
	}
	if attempt.ID != attemptID || attempt.ObjectiveID != objectiveID || attempt.PracticeContentHash != objective.PracticeContentHash || attempt.PracticeSessionID == nil || *attempt.PracticeSessionID == uuid.Nil || !objective.requires(Skill(attempt.Skill)) || strings.TrimSpace(attempt.Answer) == "" {
		return input, ErrPracticeMismatch
	}
	var learnerArtifact *sharedtextbook.LearnerArtifact
	if attempt.LearnerArtifactHash != "" {
		learnerArtifact, err = service.GetLearnerArtifact(ctx, workspaceID, objectiveID, attemptID)
		if err != nil {
			return input, err
		}
		if learnerArtifact == nil {
			return input, ErrPracticeMismatch
		}
	}
	var task *sharedtextbook.PracticeTask
	for index := range projection.PracticeSet.Tasks {
		candidate := &projection.PracticeSet.Tasks[index]
		if candidate.ID == attempt.PracticeTaskID && string(candidate.Mode) == attempt.Skill {
			task = candidate
			break
		}
	}
	if task == nil {
		return input, ErrPracticeMismatch
	}
	chapterID, err := uuid.Parse(projection.ChapterID)
	if err != nil {
		return input, ErrPracticeMismatch
	}
	evidence, err := source.LoadApprovedAssessmentEvidence(ctx, workspaceID, chapterID, *objective.PracticeRevisionID, task.ID)
	if err != nil {
		return input, err
	}
	if evidence.Validate() != nil || evidence.WorkspaceID != workspaceID.String() || evidence.ChapterID != chapterID.String() || evidence.RevisionID != objective.PracticeRevisionID.String() || evidence.ContentHash != objective.PracticeContentHash || evidence.TaskID != task.ID || len(evidence.Sources) != len(task.EvidenceIDs) {
		return input, ErrPracticeMismatch
	}
	byID := map[string]sharedtextbook.PracticeSourceExcerpt{}
	for _, item := range evidence.Sources {
		byID[item.Reference.ID] = item
	}
	input = AssessmentInput{AttemptID: attemptID.String(), ObjectiveID: objectiveID.String(), RevisionID: projection.RevisionID, Skill: Skill(attempt.Skill), Prompt: task.Prompt + "\n变式：" + task.Variation, Answer: attempt.Answer, Rubric: append([]AssessmentCriterion(nil), task.Rubric...)}
	input.LearnerArtifact = learnerArtifact
	input.TaskReference = &AssessmentTaskReference{TaskID: task.ID, PracticeContentHash: projection.ContentHash, ExpectedAnswer: task.ExpectedAnswer}
	input.DecisionPolicy = AssessmentJudgmentPolicy
	for _, id := range task.EvidenceIDs {
		item, exists := byID[id]
		if !exists {
			return AssessmentInput{}, ErrPracticeMismatch
		}
		input.Evidence = append(input.Evidence, AssessmentEvidence{ID: id, Kind: "source", ContentHash: item.ExcerptHash, Excerpt: item.Excerpt})
	}
	if err := input.Validate(); err != nil {
		return AssessmentInput{}, err
	}
	return input, nil
}
