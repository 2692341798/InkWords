package parse

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	crawldomain "inkwords-backend/services/parser-service/domain/crawl"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	parserinfra "inkwords-backend/shared/platform/parser"
	sharedmq "inkwords-backend/shared/platform/rabbitmq"
	"inkwords-backend/shared/platform/sourceartifact"
)

func TestTaskConsumer_HandleParseRequested_PersistsParseResult(t *testing.T) {
	tasks := &fakeParseTaskService{}
	parserService := &stubParseTaskService{
		parseFunc: func(src io.Reader, filename string) (ParseResult, error) {
			body, err := io.ReadAll(src)
			require.NoError(t, err)
			require.Equal(t, "courseware.zip", filename)
			require.Equal(t, "zip-bytes", string(body))
			return ParseResult{
				SourceContent: "parsed content",
				ArchiveSummary: &parserinfra.ArchiveSummary{
					TotalFiles: 3,
					KeptFiles:  2,
				},
			}, nil
		},
	}

	consumer := NewTaskConsumer(tasks, parserService)
	err := consumer.HandleParseRequested(context.Background(), sharedmq.ParseRequestedMessage{
		TaskID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Kind:   "parse_archive",
		Payload: json.RawMessage(`{
			"filename":"courseware.zip",
			"content_base64":"emlwLWJ5dGVz"
		}`),
	})

	require.NoError(t, err)
	require.True(t, tasks.markRunningCalled)
	require.Equal(t, "succeeded", tasks.lastStatus)

	var stored map[string]any
	require.NoError(t, json.Unmarshal(tasks.lastResult, &stored))
	require.Equal(t, "parsed content", stored["source_content"])
	summary, ok := stored["archive_summary"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(3), summary["total_files"])
	require.Equal(t, float64(2), summary["kept_files"])
}

func TestTaskConsumer_HandleParseRequested_InvalidPayloadMarksTaskFailed(t *testing.T) {
	tasks := &fakeParseTaskService{}
	consumer := NewTaskConsumer(tasks, &stubParseTaskService{})

	err := consumer.HandleParseRequested(context.Background(), sharedmq.ParseRequestedMessage{
		TaskID:  uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"),
		Kind:    "parse_file",
		Payload: json.RawMessage(`{"filename":"lesson.md","content_base64":"%%%"}`),
	})

	require.NoError(t, err)
	require.Equal(t, "failed", tasks.lastStatus)
	require.Equal(t, "invalid parse payload", tasks.lastErrorMessage)
}

func TestTaskConsumer_TextbookSourceImportReturnsStructuredEvidenceOnly(t *testing.T) {
	content := []byte("# 入门\n\n从这里开始。")
	artifactStore := sourceartifact.NewStore(t.TempDir())
	artifact, err := artifactStore.Stage(context.Background(), bytes.NewReader(content))
	require.NoError(t, err)
	payload := sharedtextbook.SourceImportTaskPayload{
		TaskVersion: 2, TaskSubtype: sharedtextbook.TextbookSourceImportTaskSubtype,
		ProjectID: "project-1", SourceID: "source-1", SnapshotID: "snapshot-1", SourceKind: sharedtextbook.SourceKindMarkdown, SourceRole: sharedtextbook.SourceRolePrimary,
		Locator: "file:///intro.md", Filename: "intro.md", ArtifactToken: artifact.Token, ContentHash: artifact.ContentHash, ByteSize: artifact.ByteSize,
	}
	payload.InputHash = sharedtextbook.SourceImportInputHash(payload.ProjectID, payload.SourceID, payload.SnapshotID, payload.Filename, payload.ContentHash)
	tasks := &fakeParseTaskService{}
	consumer := NewTaskConsumer(tasks, &stubParseTaskService{structuredFunc: func(source io.Reader, request parserinfra.StructuredParseRequest) (parserinfra.StructuredDocument, error) {
		require.True(t, request.LegacyCodeParagraphs)
		require.Equal(t, payload.SourceID, request.SourceID)
		require.Equal(t, payload.SnapshotID, request.SnapshotID)
		actual, readErr := io.ReadAll(source)
		require.NoError(t, readErr)
		require.Equal(t, content, actual)
		return parserinfra.StructuredDocument{SourceID: request.SourceID, Document: sharedtextbook.SourceDocument{ID: "doc-1", SnapshotID: request.SnapshotID, CanonicalLocator: request.CanonicalLocator, Title: "入门", MediaType: "text/markdown", ContentHash: "sha256:doc"}, Chunks: []sharedtextbook.SourceChunk{{ID: "chunk-1", DocumentID: "doc-1", Ordinal: 1, Locator: sharedtextbook.EvidenceLocator{Path: request.Filename, StartLine: 1, EndLine: 1}, TextHash: "sha256:chunk", SearchText: "从这里开始。"}}}, nil
	}}, artifactStore)
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	workspaceID := uuid.New()
	require.NoError(t, consumer.HandleParseRequested(context.Background(), sharedmq.ParseRequestedMessage{TaskID: uuid.New(), Kind: sharedtextbook.TextbookSourceImportTaskSubtype, WorkspaceID: &workspaceID, Payload: raw}))
	require.Equal(t, "succeeded", tasks.lastStatus)
	var stored sharedtextbook.SourceImportTaskResult
	require.NoError(t, json.Unmarshal(tasks.lastResult, &stored))
	require.NoError(t, stored.ValidateAgainst(payload))
}

func TestTaskConsumerPreservesFrozenGoParserVersion(t *testing.T) {
	for _, version := range []int{2, sharedtextbook.SourceImportTaskVersion} {
		content := []byte("package demo\n\nfunc helper() {\n\tx := 1\n\n\t_ = x\n}\n")
		store := sourceartifact.NewStore(t.TempDir())
		artifact, err := store.Stage(context.Background(), bytes.NewReader(content))
		require.NoError(t, err)
		payload := sharedtextbook.SourceImportTaskPayload{TaskVersion: version, TaskSubtype: sharedtextbook.TextbookSourceImportTaskSubtype, ProjectID: "project", SourceID: "source", SnapshotID: "snapshot", SourceKind: sharedtextbook.SourceKindGitRepository, SourceRole: sharedtextbook.SourceRolePrimary, Locator: "https://github.com/gin-gonic/gin", Filename: "file.go", ArtifactToken: artifact.Token, ContentHash: artifact.ContentHash, ByteSize: artifact.ByteSize, ResolvedVersion: "73726dc606796a025971fe451f0aa6f1b9b847f6"}
		payload.InputHash = sharedtextbook.SourceImportInputHash(payload.ProjectID, payload.SourceID, payload.SnapshotID, payload.Filename, payload.ContentHash, payload.ResolvedVersion)
		tasks := &fakeParseTaskService{}
		consumer := NewTaskConsumer(tasks, &stubParseTaskService{structuredFunc: func(source io.Reader, request parserinfra.StructuredParseRequest) (parserinfra.StructuredDocument, error) {
			require.Equal(t, version == 2, request.LegacyCodeParagraphs)
			return parserinfra.NewStructuredParser().Parse(source, request)
		}}, store)
		raw, err := json.Marshal(payload)
		require.NoError(t, err)
		workspaceID := uuid.New()
		require.NoError(t, consumer.HandleParseRequested(context.Background(), sharedmq.ParseRequestedMessage{TaskID: uuid.New(), Kind: payload.TaskSubtype, WorkspaceID: &workspaceID, Payload: raw}))
		require.Equal(t, "succeeded", tasks.lastStatus)
		var result sharedtextbook.SourceImportTaskResult
		require.NoError(t, json.Unmarshal(tasks.lastResult, &result))
		require.NoError(t, result.ValidateAgainst(payload))
		if version == 2 {
			require.Len(t, result.Chunks, 3)
			require.Empty(t, result.Chunks[1].Locator.Symbol)
		} else {
			require.Len(t, result.Chunks, 2)
			require.Equal(t, "helper", result.Chunks[1].Locator.Symbol)
		}
	}
}

func TestTaskConsumer_OfficialWebImportRequiresCompleteBoundedStructuredEvidence(t *testing.T) {
	payload := sharedtextbook.OfficialWebImportTaskPayload{TaskVersion: 1, TaskSubtype: sharedtextbook.TextbookOfficialWebImportTaskSubtype, ProjectID: "project-1", SourceID: "source-1", SnapshotID: "snapshot-1", SourceKind: sharedtextbook.SourceKindOfficialWeb, SourceRole: sharedtextbook.SourceRoleOfficial, EntryURL: "https://gin-gonic.com/en/docs/", AllowedPathPrefixes: []string{"/en/docs"}}
	payload.InputHash = sharedtextbook.OfficialWebImportInputHash(payload.ProjectID, payload.SourceID, payload.SnapshotID, payload.EntryURL, payload.AllowedPathPrefixes)
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	tasks := &fakeParseTaskService{}
	consumer := NewTaskConsumer(tasks, &stubParseTaskService{officialWebFunc: func(source io.Reader, request parserinfra.StructuredParseRequest) (parserinfra.StructuredDocument, error) {
		body, readErr := io.ReadAll(source)
		require.NoError(t, readErr)
		require.Equal(t, []byte("<h1>Gin</h1>"), body)
		require.Equal(t, "https://gin-gonic.com/en/docs/", request.CanonicalLocator)
		require.True(t, request.LegacyOfficialWeb)
		document := sharedtextbook.SourceDocument{ID: "doc-1", SnapshotID: request.SnapshotID, CanonicalLocator: request.CanonicalLocator, Title: "Gin", MediaType: "text/html", ContentHash: "sha256:doc"}
		chunk := sharedtextbook.SourceChunk{ID: "chunk-1", DocumentID: document.ID, Ordinal: 1, Locator: sharedtextbook.EvidenceLocator{URL: request.CanonicalLocator, HeadingPath: []string{"Gin"}}, TextHash: "sha256:chunk", SearchText: "Gin"}
		return parserinfra.StructuredDocument{SourceID: request.SourceID, Document: document, Chunks: []sharedtextbook.SourceChunk{chunk}}, nil
	}}).WithOfficialWebCrawler(&fakeOfficialWebCrawler{manifest: crawldomain.Manifest{Status: "complete", SnapshotInputHash: "sha256:manifest"}, pages: []crawldomain.CapturedPage{{Manifest: crawldomain.PageManifest{CanonicalURL: "https://gin-gonic.com/en/docs/", ContentType: "text/html", FetchedAt: time.Now().UTC()}, Body: []byte("<h1>Gin</h1>")}}})
	workspaceID := uuid.New()
	require.NoError(t, consumer.HandleParseRequested(context.Background(), sharedmq.ParseRequestedMessage{TaskID: uuid.New(), Kind: sharedtextbook.TextbookOfficialWebImportTaskSubtype, WorkspaceID: &workspaceID, Payload: raw}))
	require.Equal(t, "succeeded", tasks.lastStatus)
	var stored sharedtextbook.OfficialWebImportTaskResult
	require.NoError(t, json.Unmarshal(tasks.lastResult, &stored))
	require.NoError(t, stored.ValidateAgainst(payload))
}

type fakeOfficialWebCrawler struct {
	manifest crawldomain.Manifest
	pages    []crawldomain.CapturedPage
	err      error
}

func (crawler *fakeOfficialWebCrawler) CrawlCaptured(_ context.Context, _ crawldomain.Policy) (crawldomain.Manifest, []crawldomain.CapturedPage, error) {
	return crawler.manifest, crawler.pages, crawler.err
}

type fakeParseTaskService struct {
	markRunningCalled bool
	lastStatus        string
	lastResult        []byte
	lastErrorMessage  string
	cancelled         bool
	markRunningErr    error
	isCancelledErr    error
	workspaceMismatch bool
}

func (f *fakeParseTaskService) MarkRunning(_ context.Context, _ uuid.UUID) error {
	f.markRunningCalled = true
	f.lastStatus = "running"
	return f.markRunningErr
}

func (f *fakeParseTaskService) MarkSucceeded(_ context.Context, _ uuid.UUID, result []byte) error {
	f.lastStatus = "succeeded"
	f.lastResult = append([]byte(nil), result...)
	return nil
}

func (f *fakeParseTaskService) MarkFailed(_ context.Context, _ uuid.UUID, message string) error {
	f.lastStatus = "failed"
	f.lastErrorMessage = message
	return nil
}

func (f *fakeParseTaskService) IsCancelled(_ context.Context, _ uuid.UUID) (bool, error) {
	return f.cancelled, f.isCancelledErr
}

func (f *fakeParseTaskService) TextbookWorkspaceMatches(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) (bool, error) {
	return !f.workspaceMismatch, nil
}

func TestTaskConsumerRejectsMissingOrMismatchedTextbookWorkspaceBeforeParsing(t *testing.T) {
	workspaceID := uuid.New()
	for _, test := range []struct {
		name      string
		workspace *uuid.UUID
		mismatch  bool
	}{
		{name: "missing"},
		{name: "mismatched", workspace: &workspaceID, mismatch: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			tasks := &fakeParseTaskService{workspaceMismatch: test.mismatch}
			parserCalled := false
			consumer := NewTaskConsumer(tasks, &stubParseTaskService{structuredFunc: func(io.Reader, parserinfra.StructuredParseRequest) (parserinfra.StructuredDocument, error) {
				parserCalled = true
				return parserinfra.StructuredDocument{}, nil
			}})
			err := consumer.HandleParseRequested(t.Context(), sharedmq.ParseRequestedMessage{TaskID: uuid.New(), Kind: sharedtextbook.TextbookSourceImportTaskSubtype, WorkspaceID: test.workspace, Payload: json.RawMessage(`{}`)})
			require.NoError(t, err)
			require.Equal(t, "failed", tasks.lastStatus)
			require.False(t, tasks.markRunningCalled)
			require.False(t, parserCalled)
		})
	}
}

type stubParseTaskService struct {
	parseFunc       func(src io.Reader, filename string) (ParseResult, error)
	structuredFunc  func(io.Reader, parserinfra.StructuredParseRequest) (parserinfra.StructuredDocument, error)
	archiveFunc     func(io.Reader, parserinfra.StructuredParseRequest) (parserinfra.StructuredArchiveResult, error)
	officialWebFunc func(io.Reader, parserinfra.StructuredParseRequest) (parserinfra.StructuredDocument, error)
}

func (s *stubParseTaskService) Parse(src io.Reader, filename string) (ParseResult, error) {
	if s.parseFunc != nil {
		return s.parseFunc(src, filename)
	}
	return ParseResult{}, errors.New("unexpected parse call")
}

func (s *stubParseTaskService) ParseStructured(source io.Reader, request parserinfra.StructuredParseRequest) (parserinfra.StructuredDocument, error) {
	if s.structuredFunc == nil {
		return parserinfra.StructuredDocument{}, errors.New("unexpected structured parse call")
	}
	return s.structuredFunc(source, request)
}

func (s *stubParseTaskService) ParseStructuredArchive(source io.Reader, request parserinfra.StructuredParseRequest) (parserinfra.StructuredArchiveResult, error) {
	if s.archiveFunc == nil {
		return parserinfra.StructuredArchiveResult{}, errors.New("unexpected structured archive parse call")
	}
	return s.archiveFunc(source, request)
}

func (s *stubParseTaskService) ParseOfficialWebHTML(source io.Reader, request parserinfra.StructuredParseRequest) (parserinfra.StructuredDocument, error) {
	if s.officialWebFunc == nil {
		return parserinfra.StructuredDocument{}, errors.New("unexpected official web parse call")
	}
	return s.officialWebFunc(source, request)
}

func TestDecodeParsePayload_RejectsEmptyFilename(t *testing.T) {
	_, err := decodeParsePayload([]byte(`{"content_base64":"aGVsbG8="}`))
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid parse payload")
}

func TestDecodeParsePayload_DecodesBase64Content(t *testing.T) {
	payload, err := decodeParsePayload([]byte(`{"filename":"lesson.md","content_base64":"aGVsbG8="}`))
	require.NoError(t, err)
	require.Equal(t, "lesson.md", payload.Filename)
	require.Equal(t, []byte("hello"), payload.Content)
	readerBytes, err := io.ReadAll(bytes.NewReader(payload.Content))
	require.NoError(t, err)
	require.Equal(t, "hello", string(readerBytes))
}

func TestConsumeMessage_SuccessButAckFails_ReturnsError(t *testing.T) {
	tasks := &fakeParseTaskService{}
	parserService := &stubParseTaskService{
		parseFunc: func(src io.Reader, filename string) (ParseResult, error) {
			return ParseResult{SourceContent: "ok"}, nil
		},
	}
	consumer := NewTaskConsumer(tasks, parserService)
	ack := &fakeDeliveryAcknowledger{ackErr: errors.New("ack io error")}

	body := []byte(`{"task_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","kind":"parse_file","user_id":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","payload":{"filename":"f.md","content_base64":"aGk="}}`)

	err := consumer.ConsumeMessage(context.Background(), body, ack)
	require.Error(t, err)
	require.ErrorContains(t, err, "ack for parse task")
	require.ErrorContains(t, err, "ack io error")
	require.Equal(t, "succeeded", tasks.lastStatus)
}

func TestConsumeMessage_WorkFailsAndNackFails_RecordsBoth(t *testing.T) {
	tasks := &fakeParseTaskService{markRunningErr: errors.New("db unavailable")}
	parserService := &stubParseTaskService{}
	consumer := NewTaskConsumer(tasks, parserService)
	ack := &fakeDeliveryAcknowledger{nackErr: errors.New("nack io error")}

	body := []byte(`{"task_id":"cccccccc-cccc-cccc-cccc-cccccccccccc","kind":"parse_file","user_id":"dddddddd-dddd-dddd-dddd-dddddddddddd","payload":{"filename":"f.md","content_base64":"aGk="}}`)

	err := consumer.ConsumeMessage(context.Background(), body, ack)
	require.Error(t, err)
	require.ErrorContains(t, err, "nack for parse task")
	require.ErrorContains(t, err, "nack io error")
	require.ErrorContains(t, err, "db unavailable")
	require.True(t, ack.nackCalled)
}

func TestConsumeMessage_MalformedPayload_AcksOnce(t *testing.T) {
	consumer := NewTaskConsumer(&fakeParseTaskService{}, &stubParseTaskService{})
	ack := &fakeDeliveryAcknowledger{}

	err := consumer.ConsumeMessage(context.Background(), []byte(`not json`), ack)
	require.NoError(t, err)
	require.True(t, ack.ackCalled)
	require.False(t, ack.nackCalled)
}

func TestConsumeMessage_TransientWorkError_NacksWithRequeue(t *testing.T) {
	tasks := &fakeParseTaskService{isCancelledErr: errors.New("db timeout")}
	parserService := &stubParseTaskService{}
	consumer := NewTaskConsumer(tasks, parserService)
	ack := &fakeDeliveryAcknowledger{}

	body := []byte(`{"task_id":"eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee","kind":"parse_file","user_id":"ffffffff-ffff-ffff-ffff-ffffffffffff","payload":{"filename":"f.md","content_base64":"aGk="}}`)

	err := consumer.ConsumeMessage(context.Background(), body, ack)
	require.NoError(t, err)
	require.True(t, ack.nackCalled)
	require.True(t, ack.lastNackRequeue)
	require.False(t, ack.ackCalled)
}

type fakeDeliveryAcknowledger struct {
	ackErr    error
	nackErr   error
	ackCalled bool

	nackCalled       bool
	lastNackRequeue  bool
	lastNackMultiple bool
}

func (f *fakeDeliveryAcknowledger) Ack(multiple bool) error {
	f.ackCalled = true
	return f.ackErr
}

func (f *fakeDeliveryAcknowledger) Nack(multiple bool, requeue bool) error {
	f.nackCalled = true
	f.lastNackMultiple = multiple
	f.lastNackRequeue = requeue
	return f.nackErr
}
