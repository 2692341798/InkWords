package export

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

const ExportTaskSubtypePDF = "export_pdf"

type exportPDFService interface {
	ExportSeriesToPDF(ctx context.Context, blogID uuid.UUID, userID uuid.UUID) (string, string, error)
}

type exportTaskService interface {
	MarkRunning(ctx context.Context, taskID uuid.UUID) error
	MarkSucceeded(ctx context.Context, taskID uuid.UUID, result []byte) error
	MarkFailed(ctx context.Context, taskID uuid.UUID, message string) error
	IsCancelled(ctx context.Context, taskID uuid.UUID) (bool, error)
}

type artifactStore interface {
	Save(taskID uuid.UUID, sourcePath string, filename string) (TaskResult, error)
}

// PDFPayload describes the export_pdf payload consumed by export-service.
type PDFPayload struct {
	BlogID uuid.UUID `json:"blog_id"`
}

// Consumer converts RabbitMQ export tasks into PDF export executions.
type Consumer struct {
	tasks    exportTaskService
	exporter exportPDFService
	store    artifactStore
}

// NewConsumer wires export-service worker dependencies.
func NewConsumer(tasks exportTaskService, exporter exportPDFService, store artifactStore) *Consumer {
	return &Consumer{
		tasks:    tasks,
		exporter: exporter,
		store:    store,
	}
}

// HandleExportRequested consumes one export task and writes task status/result snapshots.
func (c *Consumer) HandleExportRequested(ctx context.Context, message RequestedMessage) error {
	if c == nil || c.tasks == nil || c.exporter == nil || c.store == nil {
		return errors.New("export task consumer dependencies are not configured")
	}
	if message.Kind != ExportTaskSubtypePDF {
		return c.tasks.MarkFailed(ctx, message.TaskID, "unsupported export kind")
	}

	var payload PDFPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil || payload.BlogID == uuid.Nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, "invalid export payload")
	}

	cancelled, err := c.tasks.IsCancelled(ctx, message.TaskID)
	if err != nil {
		return err
	}
	if cancelled {
		return nil
	}

	if err := c.tasks.MarkRunning(ctx, message.TaskID); err != nil {
		return err
	}

	pdfPath, filename, err := c.exporter.ExportSeriesToPDF(ctx, payload.BlogID, message.UserID)
	if err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, err.Error())
	}

	result, err := c.store.Save(message.TaskID, pdfPath, filename)
	if err != nil {
		return c.tasks.MarkFailed(ctx, message.TaskID, err.Error())
	}

	body, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return c.tasks.MarkSucceeded(ctx, message.TaskID, body)
}
