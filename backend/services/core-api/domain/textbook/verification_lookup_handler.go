package textbook

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
)

type verificationTaskReader interface {
	GetTextbookVerificationTask(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*TextbookVerificationTask, error)
}

// GetArtifactVerification reads the existing task; it never admits execution.
func (h *Handler) GetArtifactVerification(c *gin.Context) {
	workspaceID, chapterID, ok := textbookChapterIDs(c)
	if !ok {
		return
	}
	artifactID, err := uuid.Parse(c.Param("artifactID"))
	if err != nil || artifactID == uuid.Nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	reader, ok := h.verificationCreator.(verificationTaskReader)
	if !ok {
		textbookError(c, http.StatusServiceUnavailable, "TEXTBOOK_GENERATION_UNAVAILABLE")
		return
	}
	task, err := reader.GetTextbookVerificationTask(c.Request.Context(), workspaceID, chapterID, artifactID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": task})
}
