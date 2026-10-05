package textbook

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func equalCorrectionJSON(left, right []byte) bool {
	var a, b any
	return json.Unmarshal(left, &a) == nil && json.Unmarshal(right, &b) == nil && reflect.DeepEqual(a, b)
}

func (r *GormRepository) persistCorrectionResult(ctx context.Context, workspaceID, taskID uuid.UUID, payload sharedtextbook.SampleGenerationTaskPayload, generated sharedtextbook.SampleGenerationTaskResult, input AppendRevisionInput) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var owner struct {
			ID     uuid.UUID
			Status string
		}
		if err := tx.Table("job_tasks").Clauses(clause.Locking{Strength: "UPDATE"}).Select("id,status").Where("id = ? AND workspace_id = ?", taskID, workspaceID).Take(&owner).Error; err != nil {
			return textbookNotFound(err)
		}
		if owner.Status == "cancelled" {
			return ErrInvalidState
		}
		// A replay after successful append returns the same revision even though
		// the chapter version has advanced. Different replay content is refused.
		var existing ChapterRevision
		err := tx.Where("generation_task_id = ?", taskID).First(&existing).Error
		if err == nil {
			if existing.ChapterID != input.ChapterID || existing.ContentHash != input.ContentHash || !equalCorrectionJSON(existing.DocumentJSON, input.DocumentJSON) {
				return ErrInvalidState
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var original struct {
			WorkspaceID uuid.UUID
			TaskSubtype string
			Status      string
			PayloadJSON []byte
			ResultJSON  []byte
		}
		if err := tx.Table("job_tasks").Clauses(clause.Locking{Strength: "SHARE"}).Select("workspace_id, task_subtype, status, payload_json, result_json").Where("id = ?", payload.Correction.OriginalTaskID).Take(&original).Error; err != nil {
			return textbookNotFound(err)
		}
		var source sharedtextbook.SampleGenerationTaskPayload
		var failure sharedtextbook.SampleGenerationTaskFailureResult
		if original.WorkspaceID != workspaceID || original.Status != "failed" || original.TaskSubtype != sharedtextbook.TextbookSampleGenerationTaskSubtype || json.Unmarshal(original.PayloadJSON, &source) != nil || source.Correction != nil || json.Unmarshal(original.ResultJSON, &failure) != nil || failure.ValidateRetainedCorrectionSource(source) != nil || source.InputHash != payload.Correction.OriginalInputHash || failure.RejectedDraftHash != payload.Correction.Edit.OriginalReceiptHash || failure.RejectedDraftStorage != "saved" || generated.Correction == nil || failure.PromptHash != generated.Correction.OriginalPromptHash || !equalCorrectionJSON(failure.ProviderUsageJSON, generated.Correction.OriginalUsageJSON) {
			return ErrInvalidState
		}
		bound := *r
		bound.db = tx
		projectID, err := uuid.Parse(payload.ProjectID)
		if err != nil {
			return ErrInvalidState
		}
		current, err := bound.PrepareSampleGeneration(ctx, workspaceID, projectID, input.ChapterID, payload.GenerationTarget)
		if err != nil {
			return err
		}
		if !payload.MatchesCorrectionBaseline(current) {
			return ErrVersionConflict
		}
		_, err = bound.AppendRevision(ctx, workspaceID, input)
		return err
	})
}
