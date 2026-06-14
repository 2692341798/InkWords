package parse

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	parserinfra "inkwords-backend/services/parser-service/infra/parser"
	sharedmq "inkwords-backend/shared/platform/rabbitmq"
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
		UserID: uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
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
		UserID:  uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"),
		Payload: json.RawMessage(`{"filename":"lesson.md","content_base64":"%%%"}`),
	})

	require.NoError(t, err)
	require.Equal(t, "failed", tasks.lastStatus)
	require.Equal(t, "invalid parse payload", tasks.lastErrorMessage)
}

type fakeParseTaskService struct {
	markRunningCalled bool
	lastStatus        string
	lastResult        []byte
	lastErrorMessage  string
	cancelled         bool
}

func (f *fakeParseTaskService) MarkRunning(_ context.Context, _ uuid.UUID) error {
	f.markRunningCalled = true
	f.lastStatus = "running"
	return nil
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
	return f.cancelled, nil
}

type stubParseTaskService struct {
	parseFunc func(src io.Reader, filename string) (ParseResult, error)
}

func (s *stubParseTaskService) Parse(src io.Reader, filename string) (ParseResult, error) {
	if s.parseFunc != nil {
		return s.parseFunc(src, filename)
	}
	return ParseResult{}, errors.New("unexpected parse call")
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
