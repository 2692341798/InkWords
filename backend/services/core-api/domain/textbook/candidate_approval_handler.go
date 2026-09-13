package textbook

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"net/http"
)

func (h *Handler) ApplyCandidate(c *gin.Context) {
	workspaceID, chapterID, ok := textbookChapterIDs(c)
	if !ok {
		return
	}
	candidateID, err := uuid.Parse(c.Param("revisionID"))
	if err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	var request struct {
		ExpectedVersion int                             `json:"expected_version"`
		LockOwnerID     uuid.UUID                       `json:"lock_owner_id"`
		LockVersion     int                             `json:"lock_version"`
		ReviewNote      string                          `json:"review_note"`
		ReviewerKind    string                          `json:"reviewer_kind"`
		DelegationNote  string                          `json:"delegation_note"`
		DimensionScores []sharedtextbook.DimensionScore `json:"dimension_scores"`
	}
	if err := decodeStrictJSON(c, &request); err != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	revision, err := h.service.ApplyCandidate(c.Request.Context(), workspaceID, ApplyCandidateInput{ChapterID: chapterID, CandidateRevisionID: candidateID, ExpectedVersion: request.ExpectedVersion, LockOwnerID: request.LockOwnerID, LockVersion: request.LockVersion, ReviewNote: request.ReviewNote, DimensionScores: request.DimensionScores, ReviewerKind: request.ReviewerKind, DelegationNote: request.DelegationNote})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": revision})
}
