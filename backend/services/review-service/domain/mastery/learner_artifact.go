package mastery

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type learnerArtifactStore interface {
	GetLearnerArtifact(context.Context, uuid.UUID) (*sharedtextbook.LearnerArtifact, error)
}

// GetLearnerArtifact reads submitted source only after checking both objective
// ownership and the original answer identity. It never starts verification.
func (service *Service) GetLearnerArtifact(ctx context.Context, owner, objectiveID, attemptID uuid.UUID) (*sharedtextbook.LearnerArtifact, error) {
	if service == nil || service.store == nil || owner == uuid.Nil || objectiveID == uuid.Nil || attemptID == uuid.Nil {
		return nil, ErrPracticeMismatch
	}
	objective, err := service.store.GetObjective(ctx, owner, objectiveID)
	if err != nil {
		return nil, err
	}
	attempt, err := service.store.GetAttempt(ctx, objectiveID, attemptID)
	if err != nil {
		return nil, err
	}
	if attempt.LearnerArtifactHash == "" {
		return nil, nil
	}
	store, ok := service.store.(learnerArtifactStore)
	if !ok {
		return nil, fmt.Errorf("learner code storage unavailable")
	}
	artifact, err := store.GetLearnerArtifact(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	if artifact == nil || artifact.Validate() != nil || artifact.WorkspaceID != owner.String() || artifact.ObjectiveID != objectiveID.String() || artifact.AttemptID != attemptID.String() || artifact.SnapshotHash != attempt.LearnerArtifactHash || objective.PracticeRevisionID == nil || artifact.RevisionID != objective.PracticeRevisionID.String() || artifact.PracticeContentHash != objective.PracticeContentHash || attempt.PracticeSessionID == nil || artifact.SessionID != attempt.PracticeSessionID.String() || artifact.TaskID != attempt.PracticeTaskID || string(artifact.Skill) != attempt.Skill || !artifact.SubmittedAt.Equal(attempt.AttemptedAt.UTC().Truncate(time.Microsecond)) {
		return nil, ErrPracticeMismatch
	}
	return artifact, nil
}
