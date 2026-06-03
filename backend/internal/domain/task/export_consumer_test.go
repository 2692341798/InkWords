package task_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	taskdomain "inkwords-backend/internal/domain/task"
	"inkwords-backend/internal/model"
)

func TestExportConsumer_HandleExportRequested_PersistsDownloadMetadata(t *testing.T) {
	tasks := &fakeExportTaskService{}
	exporter := &stubPDFExporter{
		exportFunc: func(_ context.Context, blogID uuid.UUID, userID uuid.UUID) (string, string, error) {
			require.Equal(t, uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"), blogID)
			require.Equal(t, uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"), userID)

			filePath := filepath.Join(t.TempDir(), "series.pdf")
			require.NoError(t, os.WriteFile(filePath, []byte("pdf"), 0o644))
			return filePath, "Go 源码入门.pdf", nil
		},
	}
	store := taskdomain.NewExportArtifactStore(t.TempDir(), 15*time.Minute, func() time.Time {
		return time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	})
	consumer := taskdomain.NewExportConsumer(tasks, exporter, store)

	err := consumer.HandleExportRequested(context.Background(), taskdomain.ExportRequestedMessage{
		TaskID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Kind:   taskdomain.ExportTaskSubtypePDF,
		UserID: uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		Payload: json.RawMessage(`{
			"blog_id":"cccccccc-cccc-cccc-cccc-cccccccccccc"
		}`),
	})
	require.NoError(t, err)
	require.Equal(t, model.JobTaskStatusSucceeded, tasks.lastStatus)
	require.Contains(t, string(tasks.lastResult), `"content_type":"application/pdf"`)
	require.Contains(t, string(tasks.lastResult), `"filename":"Go 源码入门.pdf"`)
}

func TestExportConsumer_HandleExportRequested_MarksFailedWhenExporterFails(t *testing.T) {
	tasks := &fakeExportTaskService{}
	exporter := &stubPDFExporter{
		exportFunc: func(context.Context, uuid.UUID, uuid.UUID) (string, string, error) {
			return "", "", errors.New("chromium failed")
		},
	}
	consumer := taskdomain.NewExportConsumer(tasks, exporter, taskdomain.NewExportArtifactStore(t.TempDir(), 15*time.Minute, time.Now))

	err := consumer.HandleExportRequested(context.Background(), taskdomain.ExportRequestedMessage{
		TaskID: uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"),
		Kind:   taskdomain.ExportTaskSubtypePDF,
		UserID: uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"),
		Payload: json.RawMessage(`{
			"blog_id":"ffffffff-ffff-ffff-ffff-ffffffffffff"
		}`),
	})
	require.NoError(t, err)
	require.Equal(t, model.JobTaskStatusFailed, tasks.lastStatus)
	require.Equal(t, "chromium failed", tasks.lastErrorMessage)
}

type fakeExportTaskService struct {
	lastStatus       model.JobTaskStatus
	lastResult       []byte
	lastErrorMessage string
	cancelled        bool
}

func (f *fakeExportTaskService) MarkRunning(_ context.Context, _ uuid.UUID) error {
	f.lastStatus = model.JobTaskStatusRunning
	return nil
}

func (f *fakeExportTaskService) MarkSucceeded(_ context.Context, _ uuid.UUID, result []byte) error {
	f.lastStatus = model.JobTaskStatusSucceeded
	f.lastResult = append([]byte(nil), result...)
	return nil
}

func (f *fakeExportTaskService) MarkFailed(_ context.Context, _ uuid.UUID, message string) error {
	f.lastStatus = model.JobTaskStatusFailed
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
