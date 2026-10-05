package textbook

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	coretask "inkwords-backend/services/core-api/domain/task"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func assertCorrectionPersistence(t *testing.T, database *gorm.DB, workspaceID, projectID, chapterID uuid.UUID, target sharedtextbook.SampleGenerationTarget) {
	t.Helper()
	for _, mode := range []string{"current", "retained", "migrated"} {
		t.Run("correction-"+mode, func(t *testing.T) {
			assertCorrectionPersistenceBaseline(t, database, workspaceID, projectID, chapterID, target, mode)
		})
	}
}

func assertCorrectionPersistenceBaseline(t *testing.T, database *gorm.DB, workspaceID, projectID, chapterID uuid.UUID, target sharedtextbook.SampleGenerationTarget, mode string) {
	t.Helper()
	retained := mode != "current"
	tx := database.Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	repo := NewGormRepository(tx, target)
	if retained {
		_, err := repo.AppendRevision(t.Context(), workspaceID, AppendRevisionInput{ChapterID: chapterID, ExpectedVersion: 0, Kind: sharedtextbook.RevisionKindDraft, Markdown: "existing author draft", DocumentJSON: []byte(`{}`), ContentHash: strings.Repeat("d", 64), CreatedBy: RevisionCreatorManual})
		require.NoError(t, err)
	}
	payload, err := repo.PrepareSampleGeneration(t.Context(), workspaceID, projectID, chapterID, target)
	require.NoError(t, err)
	current := payload
	if retained {
		payload.PromptSchemaVersion = "inkwords.textbook.sample.v15"
		if mode == "migrated" {
			payload.PromptSchemaVersion = "inkwords.textbook.sample.v16"
			payload.QualityContractVersion = "inkwords.sample-quality.v9"
		}
		payload.ExpectedChapterVersion--
		payload.ParentRevisionID = ""
		payload.InputHash = sharedtextbook.SampleGenerationInputHash(payload)
	}
	marshal := func(v any) []byte { b, e := json.Marshal(v); require.NoError(t, e); return b }
	if mode == "migrated" {
		service := NewService(repo)
		chapter, err := service.CreateChapter(t.Context(), workspaceID, CreateChapterInput{ProjectID: projectID, SortOrder: 2, Title: "新增迁移章", ChapterProfile: sharedtextbook.ChapterProfileConcept})
		require.NoError(t, err)
		var volumes []sharedtextbook.BlueprintVolume
		require.NoError(t, json.Unmarshal(marshal(payload.Blueprint.Volumes), &volumes))
		added := volumes[0].Chapters[0]
		added.ID, added.Title, added.Sort = chapter.ID.String(), chapter.Title, 2
		volumes[0].Chapters = append(volumes[0].Chapters, added)
		blueprint, err := service.CreateBlueprint(t.Context(), workspaceID, CreateBlueprintInput{ProjectID: projectID, Volumes: volumes})
		require.NoError(t, err)
		_, err = service.ApproveBlueprint(t.Context(), workspaceID, projectID, blueprint.ID)
		require.NoError(t, err)
		current, err = repo.PrepareSampleGeneration(t.Context(), workspaceID, projectID, chapterID, target)
		require.NoError(t, err)
	}
	usage := []byte(`{"known":true,"input_tokens":123,"provider_call_count":1}`)
	failure := sharedtextbook.SampleGenerationTaskFailureResult{ResultVersion: 1, TaskSubtype: sharedtextbook.TextbookSampleGenerationTaskSubtype, FinalStatus: "failed", ProjectID: payload.ProjectID, ChapterID: payload.ChapterID, InputHash: payload.InputHash, ProviderName: target.ProviderName, ModelName: target.ModelName, ProviderUsageJSON: usage, QualityReportJSON: []byte(`{"contract_version":"` + payload.QualityContractVersion + `","passed":false,"failures":["unverified_runtime_evidence"]}`), PromptHash: "sha256:original", RejectedDraftHash: strings.Repeat("a", 64), RejectedDraftStorage: "saved"}
	original := coretask.JobTask{TaskType: "generation", TaskSubtype: sharedtextbook.TextbookSampleGenerationTaskSubtype, WorkspaceID: &workspaceID, TextbookChapterID: &chapterID, Status: coretask.JobTaskStatusFailed, PayloadJSON: marshal(payload), ResultJSON: marshal(failure)}
	require.NoError(t, tx.Create(&original).Error)
	edit := sharedtextbook.SampleCorrectionInput{Format: sharedtextbook.SampleCorrectionFormat, OriginalReceiptHash: failure.RejectedDraftHash, OriginalContentHash: "sha256:" + strings.Repeat("b", 64), Reason: "修正教学机制并保留原始来源和用量。", MarkdownBody: "corrected"}
	if retained {
		edit.Baseline = &sharedtextbook.SampleCorrectionBaseline{ExpectedChapterVersion: current.ExpectedChapterVersion, ParentRevisionID: current.ParentRevisionID}
	}
	if mode == "migrated" {
		edit.TargetInputHash = current.InputHash
	}
	payload, err = sharedtextbook.PrepareRetainedSampleCorrection(payload, current, original.ID.String(), edit)
	require.NoError(t, err)
	require.NoError(t, payload.Validate())
	task := coretask.JobTask{TaskType: "generation", TaskSubtype: sharedtextbook.TextbookSampleGenerationTaskSubtype, WorkspaceID: &workspaceID, TextbookChapterID: &chapterID, Status: coretask.JobTaskStatusRunning, PayloadJSON: marshal(payload), ResultJSON: []byte(`{}`)}
	require.NoError(t, tx.Create(&task).Error)
	provenance := &sharedtextbook.SampleCorrectionProvenance{Origin: "automated_local_correction", OriginalTaskID: original.ID.String(), OriginalInputHash: payload.Correction.OriginalInputHash, OriginalReceiptHash: failure.RejectedDraftHash, OriginalContentHash: payload.Correction.Edit.OriginalContentHash, CorrectionInputHash: sharedtextbook.SampleCorrectionInputHash(payload.Correction.Edit), FrozenRequestHash: "sha256:" + strings.Repeat("c", 64), OriginalPromptHash: failure.PromptHash, OriginalUsageJSON: usage, Reason: payload.Correction.Edit.Reason}
	provenance.TargetInputHash = edit.TargetInputHash
	markdown := "# corrected"
	hash := sourceImportTestDigest([]byte(markdown))
	result := sharedtextbook.SampleGenerationTaskResult{ResultVersion: 2, TaskSubtype: payload.TaskSubtype, ProjectID: payload.ProjectID, ChapterID: payload.ChapterID, ExpectedChapterVersion: payload.ExpectedChapterVersion, ParentRevisionID: payload.ParentRevisionID, InputHash: payload.InputHash, Markdown: markdown, ContentHash: hash, BookContractRevision: payload.BookContract.RevisionID, StyleSheetRevision: payload.StyleSheet.RevisionID, BlueprintRevision: payload.Blueprint.RevisionID, EvidencePackHash: sharedtextbook.GenerationEvidencePackHash(payload.EvidencePack), PromptHash: provenance.CorrectionInputHash, ProviderName: target.ProviderName, ModelName: target.ModelName, ProviderUsageJSON: []byte(`{"known":false,"provider_call_count":0}`), QualityReportJSON: []byte(`{"contract_version":"` + sharedtextbook.SampleQualityContractVersion + `","passed":true}`), Correction: provenance}
	result.DocumentJSON = marshal(map[string]any{"correction": provenance, "sample": map[string]any{"markdown": markdown, "content_hash": hash, "generation_mode": "automated_local_correction", "runtime_verification": "unverified"}})
	resultMap := func() map[string]any {
		var v map[string]any
		require.NoError(t, json.Unmarshal(marshal(result), &v))
		return v
	}
	require.NoError(t, result.ValidateAgainst(payload))
	// A retried source task is no longer the retained failed baseline.
	require.NoError(t, tx.Model(&coretask.JobTask{}).Where("id = ?", original.ID).Update("status", "running").Error)
	require.Error(t, repo.PersistTextbookSampleResult(t.Context(), task.ID, resultMap()))
	require.NoError(t, tx.Model(&coretask.JobTask{}).Where("id = ?", original.ID).Update("status", "failed").Error)
	// Approved input can change while a worker is checking a retained receipt.
	require.NoError(t, tx.Model(&Project{}).Where("id = ?", projectID).Update("approved_blueprint_revision_id", nil).Error)
	require.Error(t, repo.PersistTextbookSampleResult(t.Context(), task.ID, resultMap()))
	require.NoError(t, tx.Model(&Project{}).Where("id = ?", projectID).Update("approved_blueprint_revision_id", payload.Blueprint.RevisionID).Error)
	require.NoError(t, repo.PersistTextbookSampleResult(t.Context(), task.ID, resultMap()))
	require.NoError(t, repo.PersistTextbookSampleResult(t.Context(), task.ID, resultMap()))
	var count int64
	require.NoError(t, tx.Model(&ChapterRevision{}).Where("generation_task_id = ?", task.ID).Count(&count).Error)
	require.EqualValues(t, 1, count)
	context, err := repo.GetGeneratedRevisionContext(t.Context(), task.ID)
	require.NoError(t, err)
	require.Equal(t, payload.ExpectedChapterVersion+1, context.Revision.RevisionNumber)
	require.Equal(t, RevisionCreatorGeneration, context.Revision.CreatedBy)
	var retainedSource coretask.JobTask
	require.NoError(t, tx.First(&retainedSource, "id = ?", original.ID).Error)
	require.JSONEq(t, string(original.PayloadJSON), string(retainedSource.PayloadJSON))
	require.JSONEq(t, string(original.ResultJSON), string(retainedSource.ResultJSON))
	// A distinct delivery based on the stale chapter cannot overwrite r1.
	task.ID = uuid.New()
	require.NoError(t, tx.Create(&task).Error)
	require.ErrorIs(t, repo.PersistTextbookSampleResult(t.Context(), task.ID, resultMap()), ErrVersionConflict)
	var plan []string
	require.NoError(t, tx.Raw("EXPLAIN (FORMAT TEXT) SELECT id,status FROM job_tasks WHERE id = ? AND workspace_id = ? FOR UPDATE", task.ID, workspaceID).Scan(&plan).Error)
	t.Logf("correction serialization query plan: %v", plan)
}
