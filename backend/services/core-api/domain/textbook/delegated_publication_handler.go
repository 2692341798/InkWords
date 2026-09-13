package textbook

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// RecordDelegatedPublicationReview accepts an explicit AI decision; server-owned
// actor, build identity, revision and timestamp cannot be spoofed by JSON input.
func (h *Handler) RecordDelegatedPublicationReview(c *gin.Context) {
	workspaceID, buildID, ok := textbookBuildIDs(c)
	if !ok {
		return
	}
	var request RecordDelegatedPublicationReviewInput
	if err := decodeStrictJSON(c, &request); err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	request.BuildID = buildID
	review, err := h.service.RecordDelegatedPublicationReview(c.Request.Context(), workspaceID, request)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "data": review})
}
