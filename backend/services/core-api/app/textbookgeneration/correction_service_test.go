package textbookgeneration

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	coretask "inkwords-backend/services/core-api/domain/task"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"strings"
	"testing"
)

type correctionTaskCreator struct {
	capturedGenerationTaskCreator
	source    coretask.JobTask
	workspace uuid.UUID
}

func (c *correctionTaskCreator) GetTextbookTask(_ context.Context, taskID, workspaceID uuid.UUID) (coretask.JobTask, error) {
	if taskID != c.source.ID || workspaceID != c.workspace {
		return coretask.JobTask{}, textbookdomain.ErrNotFound
	}
	return c.source, nil
}

func TestCreateCorrectionBindsRetainedTaskAndCurrentApprovedInput(t *testing.T) {
	for _, mode := range []string{"current", "retained", "migrated"} {
		t.Run(mode, func(t *testing.T) {
			retained := mode != "current"
			payload := textbookGenerationPayload()
			if retained {
				payload.PromptSchemaVersion = "inkwords.textbook.sample.v15"
				if mode == "migrated" {
					payload.PromptSchemaVersion = "inkwords.textbook.sample.v16"
					payload.QualityContractVersion = "inkwords.sample-quality.v9"
				}
			}
			project, chapter, workspace := uuid.New(), uuid.New(), uuid.New()
			payload.ProjectID = project.String()
			payload.BookContract.ProjectID = project.String()
			payload.StyleSheet.ProjectID = project.String()
			payload.Blueprint.ProjectID = project.String()
			payload.ChapterID = chapter.String()
			payload.Blueprint.Volumes[0].Chapters[0].ID = chapter.String()
			payload.InputHash = sharedtextbook.SampleGenerationInputHash(payload)
			failure := sharedtextbook.SampleGenerationTaskFailureResult{ResultVersion: 1, TaskSubtype: payload.TaskSubtype, FinalStatus: "failed", ProjectID: payload.ProjectID, ChapterID: payload.ChapterID, InputHash: payload.InputHash, ProviderName: payload.GenerationTarget.ProviderName, ModelName: payload.GenerationTarget.ModelName, PromptHash: "sha256:original", ProviderUsageJSON: []byte(`{"known":true,"provider_call_count":1}`), QualityReportJSON: []byte(`{"passed":false,"contract_version":"` + payload.QualityContractVersion + `"}`), RejectedDraftHash: strings.Repeat("a", 64), RejectedDraftStorage: "saved"}
			failure.QualityReportJSON = []byte(`{"passed":false,"contract_version":"` + payload.QualityContractVersion + `","failures":["unverified_runtime_evidence"]}`)
			require.NoError(t, failure.ValidateRetainedCorrectionSource(payload))
			rawFailure, err := json.Marshal(failure)
			require.NoError(t, err)
			creator := &correctionTaskCreator{source: coretask.JobTask{ID: uuid.New(), Status: coretask.JobTaskStatusFailed, TaskSubtype: payload.TaskSubtype, PayloadJSON: mustMarshalPayload(t, payload), ResultJSON: rawFailure}, workspace: workspace}
			preparer := &fixedPayloadPreparer{payload: payload}
			if retained {
				preparer.payload.PromptSchemaVersion = sharedtextbook.SamplePromptSchemaVersion
				preparer.payload.QualityContractVersion = sharedtextbook.SampleQualityContractVersion
				if mode == "migrated" {
					preparer.payload.Blueprint.RevisionID = uuid.NewString()
					preparer.payload.Blueprint.RevisionNumber++
					preparer.payload.Blueprint.ContentHash = "sha256:" + strings.Repeat("c", 64)
				}
				preparer.payload.ExpectedChapterVersion++
				preparer.payload.ParentRevisionID = uuid.NewString()
				preparer.payload.InputHash = sharedtextbook.SampleGenerationInputHash(preparer.payload)
			}
			service := NewService(preparer, creator)
			edit := sharedtextbook.SampleCorrectionInput{Format: sharedtextbook.SampleCorrectionFormat, OriginalReceiptHash: failure.RejectedDraftHash, OriginalContentHash: "sha256:" + strings.Repeat("b", 64), Reason: "修订机制解释并保留原模型用量。", MarkdownBody: "corrected body"}
			if retained {
				_, err = service.CreateSampleCorrectionTask(t.Context(), workspace, project, chapter, creator.source.ID, edit)
				require.ErrorIs(t, err, textbookdomain.ErrVersionConflict)
				require.Zero(t, creator.calls)
				edit.Baseline = &sharedtextbook.SampleCorrectionBaseline{ExpectedChapterVersion: preparer.payload.ExpectedChapterVersion, ParentRevisionID: preparer.payload.ParentRevisionID}
			}
			if mode == "migrated" {
				edit.TargetInputHash = preparer.payload.InputHash
			}
			_, err = service.CreateSampleCorrectionTask(t.Context(), workspace, project, chapter, creator.source.ID, edit)
			require.NoError(t, err)
			require.Equal(t, 1, creator.calls)
			var queued sharedtextbook.SampleGenerationTaskPayload
			require.NoError(t, json.Unmarshal(creator.input.Payload, &queued))
			require.NoError(t, queued.Validate())
			if mode == "migrated" {
				require.Equal(t, 6, queued.TaskVersion)
				require.Equal(t, preparer.payload.InputHash, queued.Correction.Edit.TargetInputHash)
			} else if retained {
				require.Equal(t, 5, queued.TaskVersion)
			} else {
				require.Equal(t, 4, queued.TaskVersion)
			}
			require.Equal(t, payload.InputHash, queued.Correction.OriginalInputHash)
			require.Equal(t, "textbook-correction:"+queued.InputHash, creator.input.IdempotencyKey)
			require.Equal(t, "automated_local_correction", generationRetryConfirmation(coretask.JobTask{Status: coretask.JobTaskStatusFailed, TaskSubtype: payload.TaskSubtype, PayloadJSON: creator.input.Payload}).ExecutionMode)
			// Neither stale approved input nor another workspace can queue a correction.
			preparer.payload.ExpectedChapterVersion++
			preparer.payload.InputHash = sharedtextbook.SampleGenerationInputHash(preparer.payload)
			_, err = service.CreateSampleCorrectionTask(t.Context(), workspace, project, chapter, creator.source.ID, edit)
			require.ErrorIs(t, err, textbookdomain.ErrVersionConflict)
			_, err = service.CreateSampleCorrectionTask(t.Context(), uuid.New(), project, chapter, creator.source.ID, edit)
			require.ErrorIs(t, err, textbookdomain.ErrNotFound)
			creator.source.Status = coretask.JobTaskStatusSucceeded
			_, err = service.CreateSampleCorrectionTask(t.Context(), workspace, project, chapter, creator.source.ID, edit)
			require.ErrorIs(t, err, textbookdomain.ErrInvalidState)
			require.Equal(t, 1, creator.calls)
		})
	}
}
