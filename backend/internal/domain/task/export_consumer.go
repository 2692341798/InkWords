package task

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

type exportPDFService interface {
	ExportSeriesToPDF(ctx context.Context, blogID uuid.UUID, userID uuid.UUID) (string, string, error)
}

type exportTaskService interface {
	MarkRunning(ctx context.Context, taskID uuid.UUID) error
	MarkSucceeded(ctx context.Context, taskID uuid.UUID, result []byte) error
	MarkFailed(ctx context.Context, taskID uuid.UUID, message string) error
	IsCancelled(ctx context.Context, taskID uuid.UUID) (bool, error)
}

// ExportConsumer 把 RabbitMQ 中的 export task 转换成现有 PDF 导出服务调用。
type ExportConsumer struct {
	tasks    exportTaskService
	exporter exportPDFService
	store    *ExportArtifactStore
}

// NewExportConsumer 通过依赖注入组装 export-service 使用的导出 worker。
func NewExportConsumer(tasks exportTaskService, exporter exportPDFService, store *ExportArtifactStore) *ExportConsumer {
	return &ExportConsumer{
		tasks:    tasks,
		exporter: exporter,
		store:    store,
	}
}

// HandleExportRequested 消费一条导出任务消息并把导出结果回写到任务表。
func (c *ExportConsumer) HandleExportRequested(ctx context.Context, message ExportRequestedMessage) error {
	if c == nil || c.tasks == nil || c.exporter == nil || c.store == nil {
		return errors.New("export task consumer dependencies are not configured")
	}
	if message.Kind != ExportTaskSubtypePDF {
		return c.tasks.MarkFailed(ctx, message.TaskID, "unsupported export kind")
	}

	var payload ExportPDFPayload
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
