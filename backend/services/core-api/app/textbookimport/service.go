// Package textbookimport freezes local file imports before parser-service runs.
package textbookimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	coretask "inkwords-backend/services/core-api/domain/task"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/sourceartifact"
)

type sourceImportPreparer interface {
	PrepareSourceImport(context.Context, uuid.UUID, textbookdomain.PrepareSourceImportInput) (sharedtextbook.SourceImportTaskPayload, error)
	PrepareOfficialWebImport(context.Context, uuid.UUID, textbookdomain.PrepareOfficialWebImportInput) (sharedtextbook.OfficialWebImportTaskPayload, error)
}

// CreateOfficialWebImportTask freezes a confirmed official-document boundary
// and queues the remote crawl. It does not fetch from core-api or persist a
// source snapshot before parser-service returns a complete result.
func (service *Service) CreateOfficialWebImportTask(ctx context.Context, workspaceID, projectID, sourceID uuid.UUID, allowedPathPrefixes []string) (textbookdomain.SourceImportTask, error) {
	if service == nil || service.preparer == nil || service.taskCreator == nil || workspaceID == uuid.Nil {
		return textbookdomain.SourceImportTask{}, errors.New("official web import is not configured")
	}
	payload, err := service.preparer.PrepareOfficialWebImport(ctx, workspaceID, textbookdomain.PrepareOfficialWebImportInput{ProjectID: projectID, SourceID: sourceID, SnapshotID: uuid.New(), AllowedPathPrefixes: append([]string(nil), allowedPathPrefixes...)})
	if err != nil {
		return textbookdomain.SourceImportTask{}, err
	}
	if err := payload.Validate(); err != nil {
		return textbookdomain.SourceImportTask{}, fmt.Errorf("validate official web import task: %w", err)
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return textbookdomain.SourceImportTask{}, fmt.Errorf("marshal official web import task: %w", err)
	}
	task, err := service.taskCreator.CreateParseTask(ctx, coretask.CreateParseTaskInput{WorkspaceID: workspaceID, TaskSubtype: sharedtextbook.TextbookOfficialWebImportTaskSubtype, IdempotencyKey: "textbook-official-web-import:" + payload.InputHash, Payload: rawPayload})
	if err != nil {
		return textbookdomain.SourceImportTask{}, err
	}
	return textbookdomain.SourceImportTask{ID: task.ID, Status: string(task.Status)}, nil
}

type parseTaskCreator interface {
	CreateParseTask(context.Context, coretask.CreateParseTaskInput) (coretask.JobTask, error)
}

type sourceArtifactStager interface {
	Stage(context.Context, io.Reader) (sourceartifact.Artifact, error)
}

// Service keeps raw file transport separate from source facts. Parser-service
// receives a frozen payload and core-api owns all subsequent persistence.
type Service struct {
	preparer    sourceImportPreparer
	taskCreator parseTaskCreator
	artifacts   sourceArtifactStager
}

func NewService(preparer sourceImportPreparer, taskCreator parseTaskCreator, artifacts sourceArtifactStager) *Service {
	return &Service{preparer: preparer, taskCreator: taskCreator, artifacts: artifacts}
}

// CreateSourceImportTask creates an idempotent structured-parse task for one registered source.
func (service *Service) CreateSourceImportTask(ctx context.Context, workspaceID, projectID, sourceID uuid.UUID, filename string, content io.Reader, resolvedVersion string) (textbookdomain.SourceImportTask, error) {
	if service == nil || service.preparer == nil || service.taskCreator == nil || service.artifacts == nil || workspaceID == uuid.Nil {
		return textbookdomain.SourceImportTask{}, errors.New("textbook source import is not configured")
	}
	filename = strings.TrimSpace(filename)
	if filename == "" || content == nil {
		return textbookdomain.SourceImportTask{}, errors.New("source import file is required")
	}
	artifact, err := service.artifacts.Stage(ctx, content)
	if err != nil {
		return textbookdomain.SourceImportTask{}, fmt.Errorf("stage local source: %w", err)
	}
	snapshotID := uuid.New()
	payload, err := service.preparer.PrepareSourceImport(ctx, workspaceID, textbookdomain.PrepareSourceImportInput{
		ProjectID: projectID, SourceID: sourceID, SnapshotID: snapshotID, Filename: filename, ContentHash: artifact.ContentHash, ResolvedVersion: resolvedVersion, ByteSize: artifact.ByteSize,
	})
	if err != nil {
		return textbookdomain.SourceImportTask{}, err
	}
	payload.ArtifactToken = artifact.Token
	payload.InputHash = sharedtextbook.SourceImportInputHash(payload.ProjectID, payload.SourceID, payload.SnapshotID, payload.Filename, payload.ContentHash, payload.ResolvedVersion)
	if err := payload.Validate(); err != nil {
		return textbookdomain.SourceImportTask{}, fmt.Errorf("validate source import task: %w", err)
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return textbookdomain.SourceImportTask{}, fmt.Errorf("marshal source import task: %w", err)
	}
	task, err := service.taskCreator.CreateParseTask(ctx, coretask.CreateParseTaskInput{
		WorkspaceID: workspaceID, TaskSubtype: sharedtextbook.TextbookSourceImportTaskSubtype,
		IdempotencyKey: "textbook-source-import:" + payload.InputHash, Payload: rawPayload,
	})
	if err != nil {
		return textbookdomain.SourceImportTask{}, err
	}
	return textbookdomain.SourceImportTask{ID: task.ID, Status: string(task.Status)}, nil
}
