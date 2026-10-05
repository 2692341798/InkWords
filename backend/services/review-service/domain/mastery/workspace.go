package mastery

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ObjectiveWorkspace exposes the frozen learning target and recent append-only
// answers. Outcomes remain learner self-assessments, not model grading.
type ObjectiveWorkspace struct {
	Objective Objective       `json:"objective"`
	Attempts  []AttemptRecord `json:"attempts"`
}

// Workspace checks ownership before reading any learner answers. Only the
// latest twenty attempts are returned; older evidence stays in the store.
func (service *Service) Workspace(ctx context.Context, workspaceID, objectiveID uuid.UUID) (ObjectiveWorkspace, error) {
	if service == nil || service.store == nil || workspaceID == uuid.Nil || objectiveID == uuid.Nil {
		return ObjectiveWorkspace{}, fmt.Errorf("invalid mastery workspace request")
	}
	objective, err := service.store.GetObjective(ctx, workspaceID, objectiveID)
	if err != nil {
		return ObjectiveWorkspace{}, err
	}
	attempts, err := service.store.ListAttempts(ctx, objectiveID)
	if err != nil {
		return ObjectiveWorkspace{}, err
	}
	if len(attempts) > 20 {
		attempts = attempts[len(attempts)-20:]
	}
	if attempts == nil {
		attempts = []AttemptRecord{}
	}
	return ObjectiveWorkspace{Objective: objective, Attempts: attempts}, nil
}
