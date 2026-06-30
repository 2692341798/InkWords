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

	parserinfra "inkwords-backend/shared/platform/parser"
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
	markRunningErr    error
	isCancelledErr    error
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
