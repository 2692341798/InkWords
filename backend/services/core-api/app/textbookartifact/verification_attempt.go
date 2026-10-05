package textbookartifact

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	task "inkwords-backend/services/core-api/domain/task"
	textbook "inkwords-backend/services/core-api/domain/textbook"
	shared "inkwords-backend/shared/kernel/textbook"
)

type verificationAttempts interface {
	ListVerificationAttempts(context.Context, uuid.UUID, uuid.UUID) ([]task.JobTask, error)
	CreateVerificationAttempt(context.Context, task.CreateVerificationAttemptInput) (task.JobTask, error)
}

func (service *VerificationTaskService) CreateVerificationAttempt(ctx context.Context, workspaceID, chapterID, artifactID uuid.UUID, input textbook.VerificationAttemptInput) (textbook.TextbookVerificationTask, error) {
	if service == nil || !service.enabled || service.chapters == nil || service.tasks == nil || input.RequestID == uuid.Nil || workspaceID == uuid.Nil || chapterID == uuid.Nil || artifactID == uuid.Nil {
		return textbook.TextbookVerificationTask{}, textbook.ErrInvalidState
	}
	attempts, ok := service.tasks.(verificationAttempts)
	if !ok {
		return textbook.TextbookVerificationTask{}, textbook.ErrInvalidState
	}
	workspace, err := service.chapters.GetChapterWorkspace(ctx, workspaceID, chapterID)
	if err != nil {
		return textbook.TextbookVerificationTask{}, err
	}
	for _, artifact := range workspace.CodeArtifacts {
		if artifact.ID != artifactID || artifact.Kind != shared.CodeArtifactTeachingImplementation {
			continue
		}
		payload := shared.ArtifactVerificationRequest{ArtifactID: artifact.ID.String(), RevisionID: artifact.RevisionID.String(), ArtifactHash: artifact.ArtifactHash, ManifestHash: artifact.ManifestHash}
		row, err := attempts.CreateVerificationAttempt(ctx, task.CreateVerificationAttemptInput{RequestID: input.RequestID, WorkspaceID: workspaceID, ExpectedPreviousTaskID: input.ExpectedPreviousTaskID, Payload: payload})
		if errors.Is(err, task.ErrVerificationAttemptConflict) || errors.Is(err, task.ErrVerificationExecutionActive) {
			return textbook.TextbookVerificationTask{}, textbook.ErrVersionConflict
		}
		if err != nil {
			return textbook.TextbookVerificationTask{}, err
		}
		result := verificationTaskProjection(row)
		// An exact replay can resolve an older task. Only GET of the latest
		// sequence may advertise admission for a subsequent attempt.
		result.CanStartNew = false
		return result, nil
	}
	return textbook.TextbookVerificationTask{}, textbook.ErrNotFound
}

func verificationTaskProjection(row task.JobTask) textbook.TextbookVerificationTask {
	stopped := row.VerificationWorkerToken == nil || row.VerificationWorkerReleasedAt != nil
	terminal := row.Status == task.JobTaskStatusCancelled || row.Status == task.JobTaskStatusFailed || row.Status == task.JobTaskStatusSucceeded
	return textbook.TextbookVerificationTask{ID: row.ID, Status: string(row.Status), Attempt: row.VerificationAttempt, PreviousTaskID: row.PreviousVerificationTaskID, ExecutionStopped: stopped, CanStartNew: terminal && stopped, AttemptContract: "inkwords.verification-attempt.v1"}
}

func (service *VerificationTaskService) listVerificationAttemptProjection(ctx context.Context, workspaceID, chapterID, artifactID uuid.UUID, attempts verificationAttempts) (*textbook.TextbookVerificationTask, error) {
	workspace, err := service.chapters.GetChapterWorkspace(ctx, workspaceID, chapterID)
	if err != nil {
		return nil, err
	}
	for _, artifact := range workspace.CodeArtifacts {
		if artifact.ID != artifactID || artifact.Kind != shared.CodeArtifactTeachingImplementation {
			continue
		}
		rows, err := attempts.ListVerificationAttempts(ctx, workspaceID, artifactID)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return nil, nil
		}
		result := verificationTaskProjection(rows[0])
		result.CanStartNew = result.CanStartNew && service.enabled
		result.History = make([]textbook.TextbookVerificationTask, 0, len(rows))
		for _, row := range rows {
			var payload shared.ArtifactVerificationRequest
			if json.Unmarshal(row.PayloadJSON, &payload) != nil || payload.Validate() != nil || payload.ArtifactID != artifactID.String() || payload.RevisionID != artifact.RevisionID.String() || payload.ArtifactHash != artifact.ArtifactHash || payload.ManifestHash != artifact.ManifestHash {
				return nil, textbook.ErrInvalidState
			}
			prior := verificationTaskProjection(row)
			prior.CanStartNew = false // Only the latest task can be a new attempt's predecessor.
			result.History = append(result.History, prior)
		}
		return &result, nil
	}
	return nil, textbook.ErrNotFound
}
