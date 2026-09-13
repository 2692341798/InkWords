package textbook

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// AppendRightsAmendment accepts explicit evidence updates with a predecessor CAS.
func (h *Handler) AppendRightsAmendment(c *gin.Context) {
	workspaceID, buildID, ok := textbookBuildIDs(c)
	if !ok {
		return
	}
	var input AppendRightsAmendmentInput
	if decodeStrictJSON(c, &input) != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	input.BuildID = buildID
	result, err := h.service.AppendRightsAmendment(c.Request.Context(), workspaceID, input)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "data": result})
}
