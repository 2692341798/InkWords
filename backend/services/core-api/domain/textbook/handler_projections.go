package textbook

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
)

// GetApprovedChapterProjections reads the current or explicitly selected
// historical approved revision. Drafts and candidates remain ineligible.
func (h *Handler) GetApprovedChapterProjections(c *gin.Context) {
	workspaceID, chapterID, ok := textbookChapterIDs(c)
	if !ok {
		return
	}
	revisionID := uuid.Nil
	if raw, supplied := c.GetQuery("revision_id"); supplied {
		var err error
		revisionID, err = uuid.Parse(raw)
		if err != nil || revisionID == uuid.Nil {
			h.writeError(c, ErrInvalidState)
			return
		}
	}
	projections, err := h.service.GetRevisionProjections(c.Request.Context(), workspaceID, chapterID, revisionID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": projections})
}

// GetPracticeEvidence exposes selected source excerpts for a saved approved task.
func (h *Handler) GetPracticeEvidence(c *gin.Context) {
	workspaceID, chapterID, ok := textbookChapterIDs(c)
	if !ok {
		return
	}
	revisionID, err := uuid.Parse(c.Query("revision_id"))
	if err != nil || revisionID == uuid.Nil {
		h.writeError(c, ErrInvalidState)
		return
	}
	projection, err := h.service.GetPracticeEvidence(c.Request.Context(), workspaceID, chapterID, revisionID, c.Query("task_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": projection})
}
