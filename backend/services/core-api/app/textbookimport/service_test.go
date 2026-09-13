package textbookimport

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	coretask "inkwords-backend/services/core-api/domain/task"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/sourceartifact"
)

type fakePreparer struct {
	workspaceID uuid.UUID
	sourceID    uuid.UUID
}

func (preparer *fakePreparer) PrepareSourceImport(_ context.Context, workspaceID uuid.UUID, input textbookdomain.PrepareSourceImportInput) (sharedtextbook.SourceImportTaskPayload, error) {
	preparer.workspaceID, preparer.sourceID = workspaceID, input.SourceID
	return sharedtextbook.SourceImportTaskPayload{TaskVersion: 2, TaskSubtype: sharedtextbook.TextbookSourceImportTaskSubtype, ProjectID: input.ProjectID.String(), SourceID: input.SourceID.String(), SnapshotID: input.SnapshotID.String(), SourceKind: sharedtextbook.SourceKindMarkdown, SourceRole: sharedtextbook.SourceRolePrimary, Locator: "file:///intro.md", Filename: input.Filename, ContentHash: input.ContentHash, ByteSize: input.ByteSize}, nil
}

func (preparer *fakePreparer) PrepareOfficialWebImport(_ context.Context, workspaceID uuid.UUID, input textbookdomain.PrepareOfficialWebImportInput) (sharedtextbook.OfficialWebImportTaskPayload, error) {
	preparer.workspaceID, preparer.sourceID = workspaceID, input.SourceID
	payload := sharedtextbook.OfficialWebImportTaskPayload{TaskVersion: 1, TaskSubtype: sharedtextbook.TextbookOfficialWebImportTaskSubtype, ProjectID: input.ProjectID.String(), SourceID: input.SourceID.String(), SnapshotID: input.SnapshotID.String(), SourceKind: sharedtextbook.SourceKindOfficialWeb, SourceRole: sharedtextbook.SourceRoleOfficial, EntryURL: "https://gin-gonic.com/en/docs/", AllowedPathPrefixes: append([]string(nil), input.AllowedPathPrefixes...)}
	payload.InputHash = sharedtextbook.OfficialWebImportInputHash(payload.ProjectID, payload.SourceID, payload.SnapshotID, payload.EntryURL, payload.AllowedPathPrefixes)
	return payload, nil
}

type fakeParseTasks struct{ input coretask.CreateParseTaskInput }

func (tasks *fakeParseTasks) CreateParseTask(_ context.Context, input coretask.CreateParseTaskInput) (coretask.JobTask, error) {
	tasks.input = input
	return coretask.JobTask{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), Status: coretask.JobTaskStatusQueued}, nil
}

func TestCreateSourceImportTaskFreezesBytesAndUsesStableIdempotency(t *testing.T) {
	workspaceID, projectID, sourceID := uuid.New(), uuid.New(), uuid.New()
	preparer := &fakePreparer{}
	tasks := &fakeParseTasks{}
	service := NewService(preparer, tasks, sourceartifact.NewStore(t.TempDir()))
	content := []byte("# 入门\n\n从这里开始。")
	task, err := service.CreateSourceImportTask(context.Background(), workspaceID, projectID, sourceID, "intro.md", bytes.NewReader(content), "")
	require.NoError(t, err)
	require.Equal(t, "11111111-1111-1111-1111-111111111111", task.ID.String())
	require.Equal(t, sharedtextbook.TextbookSourceImportTaskSubtype, tasks.input.TaskSubtype)
	require.Equal(t, workspaceID, tasks.input.WorkspaceID)
	var payload sharedtextbook.SourceImportTaskPayload
	require.NoError(t, json.Unmarshal(tasks.input.Payload, &payload))
	require.NoError(t, payload.Validate())
	require.Equal(t, "textbook-source-import:"+payload.InputHash, tasks.input.IdempotencyKey)
	require.Equal(t, workspaceID, preparer.workspaceID)
	require.Equal(t, sourceID, preparer.sourceID)
}

func TestCreateSourceImportTaskRejectsMissingFileBeforePreparingSource(t *testing.T) {
	service := NewService(&fakePreparer{}, &fakeParseTasks{}, sourceartifact.NewStore(t.TempDir()))
	_, err := service.CreateSourceImportTask(context.Background(), uuid.New(), uuid.New(), uuid.New(), "intro.md", nil, "")
	require.ErrorContains(t, err, "file is required")
}

func TestCreateOfficialWebImportTaskFreezesConfirmedBoundary(t *testing.T) {
	workspaceID, projectID, sourceID := uuid.New(), uuid.New(), uuid.New()
	preparer := &fakePreparer{}
	tasks := &fakeParseTasks{}
	service := NewService(preparer, tasks, sourceartifact.NewStore(t.TempDir()))
	_, err := service.CreateOfficialWebImportTask(context.Background(), workspaceID, projectID, sourceID, []string{"/en/docs"})
	require.NoError(t, err)
	require.Equal(t, sharedtextbook.TextbookOfficialWebImportTaskSubtype, tasks.input.TaskSubtype)
	require.Equal(t, workspaceID, tasks.input.WorkspaceID)
	var payload sharedtextbook.OfficialWebImportTaskPayload
	require.NoError(t, json.Unmarshal(tasks.input.Payload, &payload))
	require.NoError(t, payload.Validate())
	require.Equal(t, "textbook-official-web-import:"+payload.InputHash, tasks.input.IdempotencyKey)
}
