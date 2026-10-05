package textbook

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type sampleCorrectionCreator interface {
	CreateSampleCorrectionTask(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, sharedtextbook.SampleCorrectionInput) (SampleGenerationTask, error)
}

// CorrectSample submits an explicit structured edit; it cannot supply a quality
// report or provider result and never directly appends a chapter revision.
func (h *Handler) CorrectSample(c *gin.Context) {
	workspaceID, chapterID, ok := textbookChapterIDs(c)
	if !ok {
		return
	}
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	creator, ok := h.taskCreator.(sampleCorrectionCreator)
	if !ok {
		textbookError(c, http.StatusServiceUnavailable, "TEXTBOOK_GENERATION_UNAVAILABLE")
		return
	}
	var request struct {
		OriginalTaskID uuid.UUID                            `json:"original_task_id"`
		Edit           sharedtextbook.SampleCorrectionInput `json:"edit"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, (1<<20)+1024)
	if err := decodeStrictJSON(c, &request); err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	task, err := creator.CreateSampleCorrectionTask(c.Request.Context(), workspaceID, projectID, chapterID, request.OriginalTaskID, request.Edit)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"code": 0, "data": gin.H{"task_id": task.ID, "status": task.Status}})
}
