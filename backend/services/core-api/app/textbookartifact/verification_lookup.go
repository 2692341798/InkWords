package textbookartifact

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	coretask "inkwords-backend/services/core-api/domain/task"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type verificationTaskReader interface {
	FindTextbookVerificationTask(context.Context, uuid.UUID, string) (*coretask.JobTask, error)
}

// GetTextbookVerificationTask restores only the task for this chapter's exact
// frozen manifest. Reading history remains available if execution is disabled.
func (service *VerificationTaskService) GetTextbookVerificationTask(ctx context.Context, workspaceID, chapterID, artifactID uuid.UUID) (*textbookdomain.TextbookVerificationTask, error) {
	if service == nil || service.chapters == nil || workspaceID == uuid.Nil || chapterID == uuid.Nil || artifactID == uuid.Nil {
		return nil, textbookdomain.ErrNotFound
	}
	if attempts, ok := service.tasks.(verificationAttempts); ok {
		return service.listVerificationAttemptProjection(ctx, workspaceID, chapterID, artifactID, attempts)
	}
	reader, ok := service.tasks.(verificationTaskReader)
	if !ok {
		return nil, errors.New("verification task lookup unavailable")
	}
	workspace, err := service.chapters.GetChapterWorkspace(ctx, workspaceID, chapterID)
	if err != nil {
		return nil, err
	}
	for _, artifact := range workspace.CodeArtifacts {
		if artifact.ID != artifactID || artifact.Kind != sharedtextbook.CodeArtifactTeachingImplementation {
			continue
		}
		item, err := reader.FindTextbookVerificationTask(ctx, workspaceID, "textbook-verify:"+artifact.ID.String()+":"+artifact.ManifestHash)
		if err != nil || item == nil {
			return nil, err
		}
		var frozen sharedtextbook.ArtifactVerificationRequest
		if json.Unmarshal(item.PayloadJSON, &frozen) != nil || frozen.Validate() != nil || frozen.ArtifactID != artifact.ID.String() || frozen.RevisionID != artifact.RevisionID.String() || frozen.ArtifactHash != artifact.ArtifactHash || frozen.ManifestHash != artifact.ManifestHash {
			return nil, textbookdomain.ErrInvalidState
		}
		return &textbookdomain.TextbookVerificationTask{ID: item.ID, Status: string(item.Status)}, nil
	}
	return nil, textbookdomain.ErrNotFound
}
