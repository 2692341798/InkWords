package textbookartifact

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	coretask "inkwords-backend/services/core-api/domain/task"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

const VerificationTaskSubtype = sharedtextbook.TextbookTeachingArtifactVerifyTaskSubtype

// VerificationTaskPayload contains identities only. The runner reloads the
// persisted manifest and content-addressed tree; neither command nor path can
// be supplied by the UI or trusted from this envelope.
type VerificationTaskPayload = sharedtextbook.ArtifactVerificationRequest

type chapterWorkspaceReader interface {
	GetChapterWorkspace(context.Context, uuid.UUID, uuid.UUID) (*textbookdomain.ChapterWorkspace, error)
}

type verificationTaskCreator interface {
	CreateTextbookVerificationTask(context.Context, coretask.CreateTextbookVerificationTaskInput) (coretask.JobTask, error)
}

// VerificationTaskService admits a verification job only after resolving the
// artifact through a workspace-owned chapter. This is deliberately an app
// service because it coordinates textbook and generic task domains.
type VerificationTaskService struct {
	chapters chapterWorkspaceReader
	tasks    verificationTaskCreator
	enabled  bool
}

func NewVerificationTaskService(chapters chapterWorkspaceReader, tasks verificationTaskCreator, enabled bool) *VerificationTaskService {
	return &VerificationTaskService{chapters: chapters, tasks: tasks, enabled: enabled}
}

func (service *VerificationTaskService) CreateTextbookVerificationTask(ctx context.Context, workspaceID, chapterID, artifactID uuid.UUID) (textbookdomain.TextbookVerificationTask, error) {
	if service == nil || !service.enabled || service.chapters == nil || service.tasks == nil || workspaceID == uuid.Nil || chapterID == uuid.Nil || artifactID == uuid.Nil {
		return textbookdomain.TextbookVerificationTask{}, errors.New("textbook artifact verification is not configured")
	}
	if _, ok := service.tasks.(verificationAttempts); ok {
		existing, err := service.GetTextbookVerificationTask(ctx, workspaceID, chapterID, artifactID)
		if err != nil {
			return textbookdomain.TextbookVerificationTask{}, err
		}
		if existing != nil {
			return *existing, nil
		}
		return service.CreateVerificationAttempt(ctx, workspaceID, chapterID, artifactID, textbookdomain.VerificationAttemptInput{RequestID: uuid.New()})
	}
	chapterWorkspace, err := service.chapters.GetChapterWorkspace(ctx, workspaceID, chapterID)
	if err != nil {
		return textbookdomain.TextbookVerificationTask{}, err
	}
	var artifact *textbookdomain.CodeArtifactRow
	for index := range chapterWorkspace.CodeArtifacts {
		if chapterWorkspace.CodeArtifacts[index].ID == artifactID {
			artifact = &chapterWorkspace.CodeArtifacts[index]
			break
		}
	}
	if artifact == nil || artifact.Kind != sharedtextbook.CodeArtifactTeachingImplementation || artifact.ArtifactHash == "" || artifact.ManifestHash == "" {
		return textbookdomain.TextbookVerificationTask{}, textbookdomain.ErrNotFound
	}
	payload, err := json.Marshal(VerificationTaskPayload{ArtifactID: artifact.ID.String(), RevisionID: artifact.RevisionID.String(), ArtifactHash: artifact.ArtifactHash, ManifestHash: artifact.ManifestHash})
	if err != nil {
		return textbookdomain.TextbookVerificationTask{}, fmt.Errorf("marshal textbook verification payload: %w", err)
	}
	task, err := service.tasks.CreateTextbookVerificationTask(ctx, coretask.CreateTextbookVerificationTaskInput{
		WorkspaceID:    workspaceID,
		TaskSubtype:    VerificationTaskSubtype,
		IdempotencyKey: "textbook-verify:" + artifact.ID.String() + ":" + artifact.ManifestHash,
		Payload:        payload,
	})
	if err != nil {
		return textbookdomain.TextbookVerificationTask{}, err
	}
	return textbookdomain.TextbookVerificationTask{ID: task.ID, Status: string(task.Status)}, nil
}
