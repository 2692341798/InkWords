package textbookgeneration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	coretask "inkwords-backend/services/core-api/domain/task"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type correctionSourceReader interface {
	GetTextbookTask(context.Context, uuid.UUID, uuid.UUID) (coretask.JobTask, error)
}

// CreateSampleCorrectionTask queues a zero-provider edit only against the exact
// failed source task and current approved chapter input. No result is accepted
// from the browser; the private receipt and quality gates remain worker-owned.
func (s *Service) CreateSampleCorrectionTask(ctx context.Context, workspaceID, projectID, chapterID, originalTaskID uuid.UUID, edit sharedtextbook.SampleCorrectionInput) (textbookdomain.SampleGenerationTask, error) {
	reader, ok := s.taskCreator.(correctionSourceReader)
	if !ok || originalTaskID == uuid.Nil {
		return textbookdomain.SampleGenerationTask{}, fmt.Errorf("%w: correction source unavailable", textbookdomain.ErrInvalidState)
	}
	// GetTextbookTask takes task ID first and workspace ID second.
	source, err := reader.GetTextbookTask(ctx, originalTaskID, workspaceID)
	if errors.Is(err, coretask.ErrTaskNotFound) || errors.Is(err, coretask.ErrTaskAccessDenied) {
		return textbookdomain.SampleGenerationTask{}, textbookdomain.ErrNotFound
	}
	if err != nil {
		return textbookdomain.SampleGenerationTask{}, err
	}
	var payload sharedtextbook.SampleGenerationTaskPayload
	var failure sharedtextbook.SampleGenerationTaskFailureResult
	if source.Status != coretask.JobTaskStatusFailed || source.TaskSubtype != sharedtextbook.TextbookSampleGenerationTaskSubtype || json.Unmarshal(source.PayloadJSON, &payload) != nil || payload.Correction != nil || payload.ProjectID != projectID.String() || payload.ChapterID != chapterID.String() || json.Unmarshal(source.ResultJSON, &failure) != nil || failure.ValidateRetainedCorrectionSource(payload) != nil || failure.RejectedDraftStorage != "saved" || failure.RejectedDraftHash != edit.OriginalReceiptHash {
		return textbookdomain.SampleGenerationTask{}, fmt.Errorf("%w: source task is not a retained rejected sample", textbookdomain.ErrInvalidState)
	}
	current, _, err := s.prepare(ctx, workspaceID, projectID, chapterID)
	if err != nil {
		return textbookdomain.SampleGenerationTask{}, err
	}
	payload, err = sharedtextbook.PrepareRetainedSampleCorrection(payload, current, originalTaskID.String(), edit)
	if errors.Is(err, sharedtextbook.ErrCorrectionBaselineChanged) {
		return textbookdomain.SampleGenerationTask{}, fmt.Errorf("%w: correction baseline changed", textbookdomain.ErrVersionConflict)
	}
	if err != nil {
		return textbookdomain.SampleGenerationTask{}, fmt.Errorf("%w: invalid correction input", textbookdomain.ErrInvalidState)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return textbookdomain.SampleGenerationTask{}, err
	}
	created, err := s.taskCreator.CreateGenerationTask(ctx, coretask.CreateGenerationTaskInput{WorkspaceID: workspaceID, TextbookChapterID: &chapterID, TaskSubtype: sharedtextbook.TextbookSampleGenerationTaskSubtype, IdempotencyKey: "textbook-correction:" + payload.InputHash, Payload: raw})
	if err != nil {
		return textbookdomain.SampleGenerationTask{}, err
	}
	return textbookdomain.SampleGenerationTask{ID: created.ID, Status: string(created.Status)}, nil
}
