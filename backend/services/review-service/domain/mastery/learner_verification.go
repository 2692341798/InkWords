package mastery

import (
	"context"

	"github.com/google/uuid"
	textbook "inkwords-backend/shared/kernel/textbook"
)

// PrepareLearnerVerificationPlan resolves the original code and approved task
// under workspace ownership. This read-only assembly never starts execution.
func (service *Service) PrepareLearnerVerificationPlan(ctx context.Context, owner, objectiveID, attemptID uuid.UUID, runner textbook.LearnerRunnerIdentity) (textbook.LearnerVerificationPlan, error) {
	input, err := service.PrepareLearnerVerificationInput(ctx, owner, objectiveID, attemptID, runner)
	if err != nil {
		return textbook.LearnerVerificationPlan{}, err
	}
	return input.Plan, nil
}

// PrepareLearnerVerificationInput returns only owner-resolved immutable data.
// The transport body is never allowed to supply these files or the task.
func (service *Service) PrepareLearnerVerificationInput(ctx context.Context, owner, objectiveID, attemptID uuid.UUID, runner textbook.LearnerRunnerIdentity) (textbook.LearnerVerificationInput, error) {
	artifact, err := service.GetLearnerArtifact(ctx, owner, objectiveID, attemptID)
	if err != nil {
		return textbook.LearnerVerificationInput{}, err
	}
	if artifact == nil {
		return textbook.LearnerVerificationInput{}, ErrPracticeMismatch
	}
	objective, err := service.store.GetObjective(ctx, owner, objectiveID)
	if err != nil {
		return textbook.LearnerVerificationInput{}, err
	}
	projection, err := objective.practiceProjection()
	if err != nil || projection == nil {
		return textbook.LearnerVerificationInput{}, ErrPracticeMismatch
	}
	plan, err := textbook.NewLearnerVerificationPlan(*artifact, *projection, runner)
	if err != nil {
		return textbook.LearnerVerificationInput{}, err
	}
	for _, task := range projection.PracticeSet.Tasks {
		if task.ID == artifact.TaskID && task.Mode == artifact.Skill {
			return textbook.LearnerVerificationInput{Format: textbook.LearnerVerificationInputFormat, Plan: plan, Artifact: *artifact, Task: task}, nil
		}
	}
	return textbook.LearnerVerificationInput{}, ErrPracticeMismatch
}
