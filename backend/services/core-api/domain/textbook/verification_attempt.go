package textbook

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"net/http"
)

// TextbookVerificationTask exposes observation and admission facts together.
// ExecutionStopped is independent of the user's cancellation status.
type TextbookVerificationTask struct {
	ID               uuid.UUID                  `json:"id"`
	Status           string                     `json:"status"`
	Attempt          int                        `json:"attempt,omitempty"`
	PreviousTaskID   *uuid.UUID                 `json:"previous_task_id,omitempty"`
	ExecutionStopped bool                       `json:"execution_stopped"`
	CanStartNew      bool                       `json:"can_start_new"`
	AttemptContract  string                     `json:"attempt_contract,omitempty"`
	History          []TextbookVerificationTask `json:"history,omitempty"`
}

type TextbookVerificationTaskCreator interface {
	CreateTextbookVerificationTask(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (TextbookVerificationTask, error)
}

// CreateArtifactVerification accepts request identity only; the application
// resolves the chapter-owned frozen manifest before it publishes any work.
func (h *Handler) CreateArtifactVerification(c *gin.Context) {
	workspaceID, chapterID, ok := textbookChapterIDs(c)
	if !ok {
		return
	}
	artifactID, err := uuid.Parse(c.Param("artifactID"))
	if err != nil || artifactID == uuid.Nil || h.verificationCreator == nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	var input VerificationAttemptInput
	if err := decodeStrictJSON(c, &input); err != nil && !errors.Is(err, io.EOF) {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	}
	var task TextbookVerificationTask
	if input.RequestID != uuid.Nil {
		creator, ok := h.verificationCreator.(TextbookVerificationAttemptCreator)
		if !ok {
			textbookError(c, http.StatusBadRequest, "INVALID_STATE")
			return
		}
		task, err = creator.CreateVerificationAttempt(c.Request.Context(), workspaceID, chapterID, artifactID, input)
	} else if input.ExpectedPreviousTaskID != nil {
		textbookError(c, http.StatusBadRequest, "INVALID_STATE")
		return
	} else {
		task, err = h.verificationCreator.CreateTextbookVerificationTask(c.Request.Context(), workspaceID, chapterID, artifactID)
	}
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"code": 0, "data": task})
}

// VerificationAttemptInput binds an explicit run to the attempt the caller saw.
// Paths, commands, environment and manifest content never come from this input.
type VerificationAttemptInput struct {
	RequestID              uuid.UUID  `json:"request_id"`
	ExpectedPreviousTaskID *uuid.UUID `json:"expected_previous_task_id,omitempty"`
}

type TextbookVerificationAttemptCreator interface {
	CreateVerificationAttempt(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, VerificationAttemptInput) (TextbookVerificationTask, error)
}
