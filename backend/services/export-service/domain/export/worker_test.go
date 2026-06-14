package export

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestConsumerHandleExportRequestedPersistsDownloadMetadata(t *testing.T) {
	tasks := &fakeExportTaskService{}
	exporter := &stubPDFExporter{
		exportFunc: func(_ context.Context, blogID uuid.UUID, userID uuid.UUID) (string, string, error) {
			require.Equal(t, uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"), blogID)
			require.Equal(t, uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"), userID)
			return "/tmp/series.pdf", "series.pdf", nil
		},
	}
	store := &stubArtifactStore{
		saveFunc: func(taskID uuid.UUID, sourcePath string, filename string) (TaskResult, error) {
			require.Equal(t, uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), taskID)
			require.Equal(t, "/tmp/series.pdf", sourcePath)
			require.Equal(t, "series.pdf", filename)
			return TaskResult{
				FileToken:   "exp_pdf_aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				Filename:    filename,
				ContentType: "application/pdf",
				ExpiresAt:   time.Date(2026, 6, 3, 12, 15, 0, 0, time.UTC),
			}, nil
		},
	}
	consumer := NewConsumer(tasks, exporter, store)

	err := consumer.HandleExportRequested(context.Background(), RequestedMessage{
		TaskID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Kind:   ExportTaskSubtypePDF,
		UserID: uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		Payload: json.RawMessage(`{
			"blog_id":"cccccccc-cccc-cccc-cccc-cccccccccccc"
		}`),
	})
	require.NoError(t, err)
	require.Equal(t, "succeeded", tasks.lastStatus)
	require.Contains(t, string(tasks.lastResult), `"content_type":"application/pdf"`)
	require.Contains(t, string(tasks.lastResult), `"filename":"series.pdf"`)
}

func TestConsumerHandleExportRequestedMarksFailedWhenExporterFails(t *testing.T) {
	tasks := &fakeExportTaskService{}
	exporter := &stubPDFExporter{
		exportFunc: func(context.Context, uuid.UUID, uuid.UUID) (string, string, error) {
			return "", "", errors.New("chromium failed")
		},
	}
	consumer := NewConsumer(tasks, exporter, &stubArtifactStore{})

	err := consumer.HandleExportRequested(context.Background(), RequestedMessage{
		TaskID: uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"),
		Kind:   ExportTaskSubtypePDF,
		UserID: uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"),
		Payload: json.RawMessage(`{
			"blog_id":"ffffffff-ffff-ffff-ffff-ffffffffffff"
		}`),
	})
	require.NoError(t, err)
	require.Equal(t, "failed", tasks.lastStatus)
	require.Equal(t, "chromium failed", tasks.lastErrorMessage)
}

type fakeExportTaskService struct {
	lastStatus       string
	lastResult       []byte
	lastErrorMessage string
	cancelled        bool
}

func (f *fakeExportTaskService) MarkRunning(_ context.Context, _ uuid.UUID) error {
	f.lastStatus = "running"
	return nil
}

func (f *fakeExportTaskService) MarkSucceeded(_ context.Context, _ uuid.UUID, result []byte) error {
	f.lastStatus = "succeeded"
	f.lastResult = append([]byte(nil), result...)
	return nil
}

func (f *fakeExportTaskService) MarkFailed(_ context.Context, _ uuid.UUID, message string) error {
	f.lastStatus = "failed"
	f.lastErrorMessage = message
	return nil
}

func (f *fakeExportTaskService) IsCancelled(_ context.Context, _ uuid.UUID) (bool, error) {
	return f.cancelled, nil
}

type stubPDFExporter struct {
	exportFunc func(ctx context.Context, blogID uuid.UUID, userID uuid.UUID) (string, string, error)
}

func (s *stubPDFExporter) ExportSeriesToPDF(ctx context.Context, blogID uuid.UUID, userID uuid.UUID) (string, string, error) {
	if s.exportFunc == nil {
		return "", "", errors.New("unexpected export call")
	}
	return s.exportFunc(ctx, blogID, userID)
}

type stubArtifactStore struct {
	saveFunc func(taskID uuid.UUID, sourcePath string, filename string) (TaskResult, error)
}

func (s *stubArtifactStore) Save(taskID uuid.UUID, sourcePath string, filename string) (TaskResult, error) {
	if s.saveFunc == nil {
		return TaskResult{}, errors.New("unexpected save call")
	}
	return s.saveFunc(taskID, sourcePath, filename)
}
