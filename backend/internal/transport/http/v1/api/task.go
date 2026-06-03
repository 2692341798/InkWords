package api

import (
	"github.com/gin-gonic/gin"

	taskdomain "inkwords-backend/internal/domain/task"
)

// TaskAPI adapts task-related HTTP routes onto the task domain handler.
type TaskAPI struct {
	taskDomainHandler *taskdomain.Handler
}

// NewTaskAPIWithDeps creates a TaskAPI with explicitly injected dependencies.
func NewTaskAPIWithDeps(taskDomainHandler *taskdomain.Handler) *TaskAPI {
	return &TaskAPI{taskDomainHandler: taskDomainHandler}
}

// CreateGenerationTask proxies the default generation entrypoint onto the task domain handler.
func (a *TaskAPI) CreateGenerationTask(c *gin.Context) {
	a.taskDomainHandler.CreateGenerationTask(c)
}

// CreateParseTask proxies the parse task entrypoint onto the task domain handler.
func (a *TaskAPI) CreateParseTask(c *gin.Context) {
	a.taskDomainHandler.CreateParseTask(c)
}

// CreateExportTask proxies the export task entrypoint onto the task domain handler.
func (a *TaskAPI) CreateExportTask(c *gin.Context) {
	a.taskDomainHandler.CreateExportTask(c)
}

// GetTask proxies task snapshot reads to the task domain handler.
func (a *TaskAPI) GetTask(c *gin.Context) {
	a.taskDomainHandler.GetTask(c)
}

// CancelTask proxies task cancellation requests to the task domain handler.
func (a *TaskAPI) CancelTask(c *gin.Context) {
	a.taskDomainHandler.CancelTask(c)
}

// StreamTask proxies the default task-based SSE subscription onto the task domain handler.
func (a *TaskAPI) StreamTask(c *gin.Context) {
	a.taskDomainHandler.StreamTask(c)
}

// DownloadTask proxies the task artifact download entrypoint onto the task domain handler.
func (a *TaskAPI) DownloadTask(c *gin.Context) {
	a.taskDomainHandler.DownloadTask(c)
}
