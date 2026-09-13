package task

import (
	"context"
	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// FindTextbookVerificationTask restores an existing execution without creating
// work. The caller resolves the immutable artifact before choosing its key.
func (s *Service) FindTextbookVerificationTask(ctx context.Context, workspaceID uuid.UUID, key string) (*JobTask, error) {
	if workspaceID == uuid.Nil {
		return nil, ErrTaskAccessDenied
	}
	item, err := s.repo.FindByWorkspaceIdempotencyKey(ctx, workspaceID, taskTypeVerification, key)
	if err != nil || item == nil {
		return item, err
	}
	if item.WorkspaceID == nil || *item.WorkspaceID != workspaceID || item.TaskSubtype != sharedtextbook.TextbookTeachingArtifactVerifyTaskSubtype {
		return nil, ErrTaskAccessDenied
	}
	return item, nil
}
