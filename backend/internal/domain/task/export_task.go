package task

import (
	"time"

	"github.com/google/uuid"
)

const (
	taskTypeExport       = "export"
	ExportTaskSubtypePDF = "export_pdf"
)

// CreateExportTaskInput 描述创建导出任务时服务层需要的输入。
type CreateExportTaskInput struct {
	RequestedBy    uuid.UUID
	TaskSubtype    string
	IdempotencyKey string
	Payload        []byte
}

// ExportPDFPayload 描述 export_pdf worker 需要消费的导出载荷。
type ExportPDFPayload struct {
	BlogID uuid.UUID `json:"blog_id"`
}

// ExportTaskResult 描述导出任务完成后存入 result_json 的受控下载元数据。
type ExportTaskResult struct {
	FileToken   string    `json:"file_token"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	ExpiresAt   time.Time `json:"expires_at"`
}
