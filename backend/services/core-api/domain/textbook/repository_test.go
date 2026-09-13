package textbook

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/datatypes"
	coretask "inkwords-backend/services/core-api/domain/task"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	platformpostgres "inkwords-backend/shared/platform/postgres"
)

func TestTextbookRepositoryPreservesApprovedRevisionsAndWorkspaceBoundaries(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()
	container, err := postgrescontainer.Run(ctx, "postgres:14-alpine", postgrescontainer.WithDatabase("textbook_test"), postgrescontainer.WithUsername("inkwords"), postgrescontainer.WithPassword("inkwords-test-password"), testcontainers.WithAdditionalWaitStrategy(wait.ForListeningPort("5432/tcp").WithStartupTimeout(30*time.Second)))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	database, err := platformpostgres.InitCore(dsn)
	require.NoError(t, err)
	workspace, err := platformpostgres.EnsureLocalWorkspace(ctx, database)
	require.NoError(t, err)

	generationTarget := sharedtextbook.SampleGenerationTarget{ProviderName: "deepseek", ModelName: "deepseek-v4-flash"}
	service := NewService(NewGormRepository(database, generationTarget))
	project, err := service.CreateProject(ctx, CreateProjectInput{WorkspaceID: workspace.ID, Title: "Gin 自学教材", Audience: sharedtextbook.AudienceFoundation, Primary: CreateSourceInput{Kind: sharedtextbook.SourceKindGitRepository, Locator: "https://github.com/gin-gonic/gin"}})
	require.NoError(t, err)
	bookContract, err := service.CreateBookContract(ctx, workspace.ID, CreateBookContractInput{ProjectID: project.ID, Reader: sharedtextbook.ReaderModel{Audience: sharedtextbook.AudienceFoundation, LearningOutcomes: []string{"解释路由登记"}}, Promise: "从真实问题讲透 Gin。", ChapterProfiles: []sharedtextbook.ChapterProfile{sharedtextbook.ChapterProfileConcept}, TerminologyVersion: "terms-1", PublicationProfile: "personal_learning"})
	require.NoError(t, err)
	require.Equal(t, StatusDraft, bookContract.Status)
	var contractDocument sharedtextbook.BookContract
	require.NoError(t, json.Unmarshal(bookContract.DocumentJSON, &contractDocument))
	require.NoError(t, contractDocument.Validate())
	bookContract, err = service.ApproveBookContract(ctx, workspace.ID, project.ID, bookContract.ID)
	require.NoError(t, err)
	require.Equal(t, StatusApproved, bookContract.Status)
	styleSheet, err := service.CreateStyleSheet(ctx, workspace.ID, CreateStyleSheetInput{ProjectID: project.ID, Language: "zh-CN", TerminologyRules: []string{"先白话后术语"}, CodeRules: []string{"标记验证状态"}, CitationRules: []string{"关键事实引用证据"}})
	require.NoError(t, err)
	var styleDocument sharedtextbook.StyleSheet
	require.NoError(t, json.Unmarshal(styleSheet.DocumentJSON, &styleDocument))
	require.NoError(t, styleDocument.Validate())
	styleSheet, err = service.ApproveStyleSheet(ctx, workspace.ID, project.ID, styleSheet.ID)
	require.NoError(t, err)
	require.Equal(t, StatusApproved, styleSheet.Status)
	_, err = service.AddSource(ctx, workspace.ID, project.ID, CreateSourceInput{Kind: sharedtextbook.SourceKindGitRepository, Role: sharedtextbook.SourceRolePrimary, Locator: "https://example.invalid/duplicate"})
	require.ErrorIs(t, err, ErrPrimarySourceExists)
	_, err = service.AddSource(ctx, workspace.ID, project.ID, CreateSourceInput{Kind: sharedtextbook.SourceKindOfficialWeb, Role: sharedtextbook.SourceRoleOfficial, Locator: "https://gin-gonic.com/docs/", OfficialConfirmed: true})
	require.NoError(t, err)
	snapshot := SourceSnapshot{SourceID: *project.PrimarySourceID, ResolvedVersion: strings.Repeat("a", 40), ContentHash: strings.Repeat("b", 64), CapturedAt: time.Now().UTC(), Status: "captured", LimitsJSON: []byte(`{}`)}
	require.NoError(t, database.Create(&snapshot).Error)
	document := sharedtextbook.SourceDocument{ID: "document-gin", SnapshotID: snapshot.ID.String(), CanonicalLocator: "README.md", Title: "Gin", MediaType: "text/markdown", ContentHash: "sha256:gin"}
	chunk := sharedtextbook.SourceChunk{ID: "chunk-gin", DocumentID: document.ID, Ordinal: 1, Locator: sharedtextbook.EvidenceLocator{Path: "README.md", StartLine: 1, EndLine: 1}, TextHash: "sha256:chunk", SearchText: "Gin is a web framework"}
	require.NoError(t, service.PersistDocuments(ctx, workspace.ID, PersistDocumentsInput{SnapshotID: snapshot.ID, Documents: []sharedtextbook.SourceDocument{document}, Chunks: []sharedtextbook.SourceChunk{chunk}}))
	var storedChunk ParsedChunk
	require.NoError(t, database.Where("id = ?", chunk.ID).First(&storedChunk).Error)
	require.Equal(t, chunk.SearchText, storedChunk.SearchText)
	library, err := service.ListSourceLibrary(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Len(t, library, 1)
	require.Equal(t, int64(1), library[0].ChunkCount)
	evidence, err := service.ListSourceEvidence(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Len(t, evidence, 1)
	require.Equal(t, chunk.ID, evidence[0].ID)
	var evidenceLocator sharedtextbook.EvidenceLocator
	require.NoError(t, json.Unmarshal(evidence[0].Locator, &evidenceLocator))
	require.Equal(t, chunk.Locator, evidenceLocator)
	retrieval, err := service.RetrieveSourceEvidence(ctx, workspace.ID, RetrieveSourceInput{ProjectID: project.ID, Query: "Gin framework", Limit: 3})
	require.NoError(t, err)
	require.Len(t, retrieval.Selected, 1)
	require.Equal(t, chunk.ID, retrieval.Selected[0].ChunkID)
	require.Contains(t, retrieval.Selected[0].Reasons, "主资料优先")
	retrievalAgain, err := service.RetrieveSourceEvidence(ctx, workspace.ID, RetrieveSourceInput{ProjectID: project.ID, Query: "Gin framework", Limit: 3})
	require.NoError(t, err)
	require.Equal(t, retrieval.ID, retrievalAgain.ID, "the same source state and query reuse the persisted plan")
	var retrievalRunCount int64
	require.NoError(t, database.Model(&RetrievalRun{}).Count(&retrievalRunCount).Error)
	require.Equal(t, int64(1), retrievalRunCount)
	_, err = service.RetrieveSourceEvidence(ctx, uuid.New(), RetrieveSourceInput{ProjectID: project.ID, Query: "Gin framework", Limit: 3})
	require.ErrorIs(t, err, ErrNotFound)
	require.ErrorIs(t, service.PersistDocuments(ctx, uuid.New(), PersistDocumentsInput{SnapshotID: snapshot.ID, Documents: []sharedtextbook.SourceDocument{document}, Chunks: []sharedtextbook.SourceChunk{chunk}}), ErrNotFound)
	chapter, err := service.CreateChapter(ctx, workspace.ID, CreateChapterInput{ProjectID: project.ID, SortOrder: 1, Title: "请求生命周期", ChapterProfile: sharedtextbook.ChapterProfileConcept})
	require.NoError(t, err)
	workspaceID, chapterID := workspace.ID, chapter.ID
	olderTask := coretask.JobTask{TaskType: "generation", TaskSubtype: sharedtextbook.TextbookSampleGenerationTaskSubtype, Status: coretask.JobTaskStatusFailed, WorkspaceID: &workspaceID, TextbookChapterID: &chapterID, IdempotencyKey: "sample-old", PayloadJSON: datatypes.JSON([]byte(`{"chapter_id":"` + chapter.ID.String() + `"}`)), ResultJSON: datatypes.JSON([]byte(`{}`)), CreatedAt: time.Now().UTC().Add(-time.Minute)}
	newerTask := coretask.JobTask{TaskType: "generation", TaskSubtype: sharedtextbook.TextbookSampleGenerationTaskSubtype, Status: coretask.JobTaskStatusRunning, WorkspaceID: &workspaceID, TextbookChapterID: &chapterID, IdempotencyKey: "sample-new", PayloadJSON: datatypes.JSON([]byte(`{"chapter_id":"` + chapter.ID.String() + `"}`)), ResultJSON: datatypes.JSON([]byte(`{}`)), CreatedAt: time.Now().UTC()}
	require.NoError(t, database.Create(&olderTask).Error)
	require.NoError(t, database.Create(&newerTask).Error)
	recoveryWorkspace, err := service.GetChapterWorkspace(ctx, workspace.ID, chapter.ID)
	require.NoError(t, err)
	require.NotNil(t, recoveryWorkspace.LatestSampleTask)
	require.Equal(t, newerTask.ID, recoveryWorkspace.LatestSampleTask.TaskID)
	require.Equal(t, string(coretask.JobTaskStatusRunning), recoveryWorkspace.LatestSampleTask.Status)
	incompleteBlueprint, err := service.CreateBlueprint(ctx, workspace.ID, CreateBlueprintInput{ProjectID: project.ID, Volumes: []sharedtextbook.BlueprintVolume{{ID: "volume-incomplete", Title: "未完成的事实计划", Sort: 1, Chapters: []sharedtextbook.BlueprintChapter{{ID: chapter.ID.String(), Title: "请求如何走过 Gin", Sort: 1, Profile: sharedtextbook.ChapterProfileConcept, EvidenceIDs: []string{chunk.ID}}}}}})
	require.NoError(t, err)
	_, err = service.ApproveBlueprint(ctx, workspace.ID, project.ID, incompleteBlueprint.ID)
	require.ErrorIs(t, err, ErrClaimCoverageIncomplete)
	blueprint, err := service.CreateBlueprint(ctx, workspace.ID, CreateBlueprintInput{ProjectID: project.ID, Volumes: []sharedtextbook.BlueprintVolume{{ID: "volume-foundation", Title: "先建立直觉", Sort: 1, Chapters: []sharedtextbook.BlueprintChapter{{ID: chapter.ID.String(), Title: "请求如何走过 Gin", Sort: 1, Profile: sharedtextbook.ChapterProfileConcept, OutcomeIDs: []string{"explain-routing"}, EvidenceIDs: []string{chunk.ID}, CriticalClaims: []sharedtextbook.ClaimRequirement{{ID: "route-registration", Label: "解释 GET 如何进入路由登记", EvidenceIDs: []string{chunk.ID}}}}}}}})
	require.NoError(t, err)
	require.Equal(t, StatusDraft, blueprint.Status)
	var blueprintDocument sharedtextbook.Blueprint
	require.NoError(t, json.Unmarshal(blueprint.DocumentJSON, &blueprintDocument))
	require.NoError(t, blueprintDocument.Validate())
	blueprint, err = service.ApproveBlueprint(ctx, workspace.ID, project.ID, blueprint.ID)
	require.NoError(t, err)
	require.Equal(t, StatusApproved, blueprint.Status)
	assertBlueprintChapterMembership(t, database, workspace.ID, project.ID, blueprint)
	assertCorrectionPersistence(t, database, workspace.ID, project.ID, chapter.ID, generationTarget)
	progress, err := service.GetProjectProgress(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Equal(t, []ProjectStageState{
		{Key: "sources", Status: "approved", Reason: "已有 1 个可引用资料片段。"},
		{Key: "blueprint", Status: "approved", Reason: "当前蓝图已人工批准，并绑定当前生成契约。"},
		{Key: "sample", Status: "ready", Reason: "可为蓝图中已映射证据的章节冻结输入并生成候选稿。"},
		{Key: "chapters", Status: "in_progress", Reason: "已有 1 个章节位置，尚无批准母稿。"},
		{Key: "verification", Status: "blocked", Reason: "需要先生成并人工批准候选稿，才能从批准修订创建教学工件。"},
		{Key: "learning", Status: "blocked", Reason: "需要先人工批准至少一章，才能从批准修订创建六维学习目标。"},
		{Key: "publication", Status: "blocked", Reason: "需要先人工批准至少一章，才能冻结待审构建。"},
	}, progress.Stages)

	projectWorkspace, err := service.GetProjectWorkspace(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Equal(t, project.ID, projectWorkspace.Project.ID)
	require.Len(t, projectWorkspace.Sources, 2)
	require.Len(t, projectWorkspace.Chapters, 1)
	require.Equal(t, bookContract.ID, *projectWorkspace.Project.ApprovedBookContractRevisionID)
	require.Equal(t, styleSheet.ID, *projectWorkspace.Project.ApprovedStyleSheetRevisionID)
	require.Equal(t, blueprint.ID, *projectWorkspace.Project.ApprovedBlueprintRevisionID)
	require.Equal(t, bookContract.ID, projectWorkspace.BookContract.ID)
	require.Equal(t, styleSheet.ID, projectWorkspace.StyleSheet.ID)
	require.Equal(t, blueprint.ID, projectWorkspace.Blueprint.ID)
	generationPayload, err := service.PrepareSampleGeneration(ctx, workspace.ID, project.ID, chapter.ID, generationTarget)
	require.NoError(t, err)
	require.NoError(t, generationPayload.Validate())
	require.Equal(t, sharedtextbook.SampleGenerationTaskVersion, generationPayload.TaskVersion)
	require.Equal(t, sharedtextbook.SamplePromptSchemaVersion, generationPayload.PromptSchemaVersion)
	require.Equal(t, sharedtextbook.SampleQualityContractVersion, generationPayload.QualityContractVersion)
	require.Equal(t, generationTarget, generationPayload.GenerationTarget)
	require.Equal(t, project.ID.String(), generationPayload.ProjectID)
	require.Equal(t, chapter.ID.String(), generationPayload.ChapterID)
	require.Equal(t, 0, generationPayload.ExpectedChapterVersion)
	require.Empty(t, generationPayload.ParentRevisionID)
	require.Equal(t, bookContract.ID.String(), generationPayload.BookContract.RevisionID)
	require.Equal(t, styleSheet.ID.String(), generationPayload.StyleSheet.RevisionID)
	require.Equal(t, blueprint.ID.String(), generationPayload.Blueprint.RevisionID)
	require.Len(t, generationPayload.EvidencePack.Evidence, 1)
	require.Equal(t, chunk.ID, generationPayload.EvidencePack.Evidence[0].ChunkID)
	require.NoError(t, database.Model(&ParsedChunk{}).Where("id = ?", chunk.ID).Update("search_text", strings.Repeat("证", maxGenerationEvidenceRunes+1)).Error)
	_, err = service.PrepareSampleGeneration(ctx, workspace.ID, project.ID, chapter.ID, generationTarget)
	require.ErrorIs(t, err, ErrEvidenceBudgetExceeded, "generation must reject an over-budget evidence set instead of silently truncating it")
	require.NoError(t, database.Model(&ParsedChunk{}).Where("id = ?", chunk.ID).Update("search_text", chunk.SearchText).Error)
	initialCandidate, err := service.AppendRevision(ctx, workspace.ID, AppendRevisionInput{ChapterID: chapter.ID, ExpectedVersion: 0, Kind: sharedtextbook.RevisionKindCandidate, Markdown: "first-candidate", DocumentJSON: []byte(`{"blocks":[]}`), ContentHash: strings.Repeat("a", 64), CreatedBy: RevisionCreatorGeneration, BookContractRevisionID: &bookContract.ID, StyleSheetRevisionID: &styleSheet.ID, BlueprintRevisionID: &blueprint.ID, EvidencePackHash: "sha256:evidence-initial", PromptHash: "sha256:prompt-initial", ProviderName: "fake", ModelName: "deterministic-gin-fixture", ProviderUsageJSON: []byte(`{"input_tokens":"unknown"}`), QualityReportJSON: []byte(`{"passed":true}`)})
	require.NoError(t, err)
	require.Nil(t, initialCandidate.ParentRevisionID, "the first generated candidate has no content predecessor")
	projectionTaskID := uuid.New()
	require.NoError(t, database.Model(&ChapterRevision{}).Where("id = ?", initialCandidate.ID).Update("generation_task_id", projectionTaskID).Error)
	projectionContext, err := NewGormRepository(database).GetGeneratedRevisionContext(ctx, projectionTaskID)
	require.NoError(t, err)
	require.Equal(t, "sha256:"+strings.TrimPrefix(bookContract.ContentHash, "sha256:"), projectionContext.BookContractHash)
	require.Equal(t, "sha256:"+strings.TrimPrefix(styleSheet.ContentHash, "sha256:"), projectionContext.StyleSheetHash)
	require.Equal(t, initialCandidate.ID, projectionContext.Revision.ID)
	require.Equal(t, workspace.ID, projectionContext.WorkspaceID)
	progress, err = service.GetProjectProgress(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Equal(t, ProjectStageState{Key: "sample", Status: "ready", Reason: "已有 1 份候选稿，但都不符合当前合同、已批准生成输入或生成目标；请按当前规则重新生成。"}, progress.Stages[2])
	require.Equal(t, ProjectStageState{Key: "verification", Status: "blocked", Reason: "候选稿已生成；需要先人工批准，才能从批准修订创建教学工件。"}, progress.Stages[4])
	manual, err := service.AppendRevision(ctx, workspace.ID, AppendRevisionInput{ChapterID: chapter.ID, ExpectedVersion: initialCandidate.RevisionNumber, Kind: sharedtextbook.RevisionKindDraft, Markdown: "# 请求", DocumentJSON: []byte(`{"blocks":[]}`), ContentHash: strings.Repeat("a", 64), CreatedBy: RevisionCreatorManual})
	require.NoError(t, err)
	_, err = service.AppendRevision(ctx, workspace.ID, AppendRevisionInput{ChapterID: chapter.ID, ExpectedVersion: 0, Kind: sharedtextbook.RevisionKindCandidate, Markdown: "stale", DocumentJSON: []byte(`{"blocks":[]}`), ContentHash: strings.Repeat("b", 64), CreatedBy: RevisionCreatorGeneration, ParentRevisionID: &manual.ID, BookContractRevisionID: &bookContract.ID, StyleSheetRevisionID: &styleSheet.ID, BlueprintRevisionID: &blueprint.ID, EvidencePackHash: "sha256:evidence", PromptHash: "sha256:prompt", ProviderName: "fake", ModelName: "deterministic-gin-fixture", ProviderUsageJSON: []byte(`{"input_tokens":"unknown"}`), QualityReportJSON: []byte(`{"passed":true}`)})
	require.ErrorIs(t, err, ErrVersionConflict)
	_, err = service.AppendRevision(ctx, workspace.ID, AppendRevisionInput{ChapterID: chapter.ID, ExpectedVersion: manual.RevisionNumber, Kind: sharedtextbook.RevisionKindCandidate, Markdown: "missing-blueprint", DocumentJSON: []byte(`{"blocks":[]}`), ContentHash: strings.Repeat("d", 64), CreatedBy: RevisionCreatorGeneration, ParentRevisionID: &manual.ID, BookContractRevisionID: &bookContract.ID, StyleSheetRevisionID: &styleSheet.ID, EvidencePackHash: "sha256:evidence", PromptHash: "sha256:prompt", ProviderName: "fake", ModelName: "deterministic-gin-fixture", ProviderUsageJSON: []byte(`{"input_tokens":"unknown"}`), QualityReportJSON: []byte(`{"passed":true}`)})
	require.ErrorIs(t, err, ErrInvalidState, "generated candidates must name the approved blueprint")
	candidate, err := service.AppendRevision(ctx, workspace.ID, AppendRevisionInput{ChapterID: chapter.ID, ExpectedVersion: manual.RevisionNumber, Kind: sharedtextbook.RevisionKindCandidate, Markdown: "candidate", DocumentJSON: []byte(`{"blocks":[]}`), ContentHash: strings.Repeat("c", 64), CreatedBy: RevisionCreatorGeneration, ParentRevisionID: &manual.ID, BookContractRevisionID: &bookContract.ID, StyleSheetRevisionID: &styleSheet.ID, BlueprintRevisionID: &blueprint.ID, EvidencePackHash: "sha256:evidence", PromptHash: "sha256:prompt", ProviderName: "fake", ModelName: "deterministic-gin-fixture", ProviderUsageJSON: []byte(`{"input_tokens":"unknown"}`), QualityReportJSON: []byte(`{"passed":true}`)})
	require.NoError(t, err)
	require.Equal(t, manual.ID, *candidate.ParentRevisionID)
	require.Equal(t, bookContract.ID, *candidate.BookContractRevisionID)
	require.Equal(t, "sha256:evidence", candidate.EvidencePackHash)
	staleOwner := uuid.New()
	staleLock, err := service.AcquireLock(ctx, workspace.ID, LockInput{ChapterID: chapter.ID, OwnerID: staleOwner, ExpectedVersion: candidate.RevisionNumber, LeaseDuration: time.Minute})
	require.NoError(t, err)
	_, err = service.ApplyCandidate(ctx, workspace.ID, ApplyCandidateInput{ChapterID: chapter.ID, CandidateRevisionID: candidate.ID, ExpectedVersion: candidate.RevisionNumber, LockOwnerID: staleOwner, LockVersion: staleLock.Version, ReviewNote: "已完成人工审阅并确认候选稿。", DimensionScores: passingSampleReviewScores()})
	require.ErrorIs(t, err, ErrInvalidState, "an unversioned quality report must become stale when hard gates change")
	manualAfterCandidate, err := service.AppendRevision(ctx, workspace.ID, AppendRevisionInput{ChapterID: chapter.ID, ExpectedVersion: candidate.RevisionNumber, Kind: sharedtextbook.RevisionKindDraft, Markdown: "# 人工后续修改", DocumentJSON: []byte(`{"blocks":[]}`), ContentHash: strings.Repeat("c", 64), CreatedBy: RevisionCreatorManual})
	require.NoError(t, err)
	owner := staleOwner
	lock, err := service.AcquireLock(ctx, workspace.ID, LockInput{ChapterID: chapter.ID, OwnerID: owner, ExpectedVersion: manualAfterCandidate.RevisionNumber, LeaseDuration: time.Minute})
	require.NoError(t, err)
	require.Equal(t, 2, lock.Version)
	_, err = service.ApplyCandidate(ctx, workspace.ID, ApplyCandidateInput{ChapterID: chapter.ID, CandidateRevisionID: candidate.ID, ExpectedVersion: manualAfterCandidate.RevisionNumber, LockOwnerID: owner, LockVersion: lock.Version, ReviewNote: "已完成人工审阅并确认候选稿。", DimensionScores: passingSampleReviewScores()})
	require.ErrorIs(t, err, ErrVersionConflict, "a candidate whose parent is no longer current must never overwrite manual work")
	candidate, err = service.AppendRevision(ctx, workspace.ID, AppendRevisionInput{ChapterID: chapter.ID, ExpectedVersion: manualAfterCandidate.RevisionNumber, Kind: sharedtextbook.RevisionKindCandidate, Markdown: "wrong-target-candidate", DocumentJSON: []byte(`{"blocks":[]}`), ContentHash: strings.Repeat("c", 64), CreatedBy: RevisionCreatorGeneration, ParentRevisionID: &manualAfterCandidate.ID, BookContractRevisionID: &bookContract.ID, StyleSheetRevisionID: &styleSheet.ID, BlueprintRevisionID: &blueprint.ID, EvidencePackHash: "sha256:evidence", PromptHash: "sha256:prompt", ProviderName: sharedtextbook.SampleFixtureProviderName, ModelName: sharedtextbook.SampleFixtureModelName, ProviderUsageJSON: []byte(`{"input_tokens":"unknown"}`), QualityReportJSON: []byte(`{"contract_version":"` + sharedtextbook.SampleQualityContractVersion + `","passed":true}`)})
	require.NoError(t, err)
	progress, err = service.GetProjectProgress(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Equal(t, ProjectStageState{Key: "sample", Status: "ready", Reason: "已有 1 份候选稿，但都不符合当前合同、已批准生成输入或生成目标；请按当前规则重新生成。"}, progress.Stages[2])
	lock, err = service.AcquireLock(ctx, workspace.ID, LockInput{ChapterID: chapter.ID, OwnerID: owner, ExpectedVersion: candidate.RevisionNumber, LeaseDuration: time.Minute})
	require.NoError(t, err)
	_, err = service.ApplyCandidate(ctx, workspace.ID, ApplyCandidateInput{ChapterID: chapter.ID, CandidateRevisionID: candidate.ID, ExpectedVersion: candidate.RevisionNumber, LockOwnerID: owner, LockVersion: lock.Version, ReviewNote: "已完成人工审阅并确认候选稿。", DimensionScores: passingSampleReviewScores()})
	require.ErrorIs(t, err, ErrInvalidState, "a candidate from a different frozen provider/model target must not be applied")

	wrongTargetCandidate := candidate
	candidate, err = service.AppendRevision(ctx, workspace.ID, AppendRevisionInput{ChapterID: chapter.ID, ExpectedVersion: wrongTargetCandidate.RevisionNumber, Kind: sharedtextbook.RevisionKindCandidate, Markdown: "candidate-after-manual", DocumentJSON: []byte(`{"blocks":[]}`), ContentHash: strings.Repeat("d", 64), CreatedBy: RevisionCreatorGeneration, ParentRevisionID: &wrongTargetCandidate.ID, BookContractRevisionID: &bookContract.ID, StyleSheetRevisionID: &styleSheet.ID, BlueprintRevisionID: &blueprint.ID, EvidencePackHash: "sha256:evidence-current-target", PromptHash: "sha256:prompt-current-target", ProviderName: generationTarget.ProviderName, ModelName: generationTarget.ModelName, ProviderUsageJSON: []byte(`{"input_tokens":"unknown"}`), QualityReportJSON: []byte(`{"contract_version":"` + sharedtextbook.SampleQualityContractVersion + `","passed":true}`)})
	require.NoError(t, err)
	chapterWorkspace, err := service.GetChapterWorkspace(ctx, workspace.ID, chapter.ID)
	require.NoError(t, err)
	require.Equal(t, generationTarget, chapterWorkspace.GenerationTarget)
	progress, err = service.GetProjectProgress(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Equal(t, ProjectStageState{Key: "sample", Status: "in_progress", Reason: "已有 1 份符合当前质量合同与生成目标的候选稿，等待人工审阅和批准。"}, progress.Stages[2])
	require.Equal(t, ProjectStageState{Key: "verification", Status: "blocked", Reason: "候选稿已生成；需要先人工批准，才能从批准修订创建教学工件。"}, progress.Stages[4])
	// The same editor may renew its lock for the new candidate revision.
	lock, err = service.AcquireLock(ctx, workspace.ID, LockInput{ChapterID: chapter.ID, OwnerID: owner, ExpectedVersion: candidate.RevisionNumber, LeaseDuration: time.Minute})
	require.NoError(t, err)
	rejectionInput := RejectCandidateInput{ChapterID: chapter.ID, CandidateRevisionID: candidate.ID, ExpectedVersion: candidate.RevisionNumber, LockOwnerID: owner, LockVersion: lock.Version, Reason: "请求查找证据不足，教学实现仍依赖生产库。"}
	_, err = service.RejectCandidate(ctx, uuid.New(), rejectionInput)
	require.ErrorIs(t, err, ErrNotFound, "another workspace must not review this candidate")
	review, err := service.RejectCandidate(ctx, workspace.ID, rejectionInput)
	require.NoError(t, err)
	require.Equal(t, CandidateDecisionRejected, review.Decision)
	require.Equal(t, candidate.ID, review.CandidateRevisionID)
	require.Equal(t, candidate.ContentHash, review.CandidateContentHash)
	require.Equal(t, sharedtextbook.SampleQualityContractVersion, review.QualityContractVersion)
	require.Equal(t, rejectionInput.Reason, review.Reason)
	reviewAgain, err := service.RejectCandidate(ctx, workspace.ID, rejectionInput)
	require.NoError(t, err)
	require.Equal(t, review.ID, reviewAgain.ID, "same rejection is idempotent")
	rejectionInput.Reason = "这次尝试修改已存在的人工理由，必须被拒绝。"
	_, err = service.RejectCandidate(ctx, workspace.ID, rejectionInput)
	require.ErrorIs(t, err, ErrInvalidState, "an immutable review reason cannot be rewritten")
	_, err = service.ApplyCandidate(ctx, workspace.ID, ApplyCandidateInput{ChapterID: chapter.ID, CandidateRevisionID: candidate.ID, ExpectedVersion: candidate.RevisionNumber, LockOwnerID: owner, LockVersion: lock.Version, ReviewNote: "已完成人工审阅并确认候选稿。", DimensionScores: passingSampleReviewScores()})
	require.ErrorIs(t, err, ErrInvalidState, "a rejected candidate cannot later be applied")

	rejectedCandidate := candidate
	candidate, err = service.AppendRevision(ctx, workspace.ID, AppendRevisionInput{ChapterID: chapter.ID, ExpectedVersion: rejectedCandidate.RevisionNumber, Kind: sharedtextbook.RevisionKindCandidate, Markdown: "candidate-after-rejection", DocumentJSON: []byte(`{"blocks":[]}`), ContentHash: strings.Repeat("d", 64), CreatedBy: RevisionCreatorGeneration, ParentRevisionID: &rejectedCandidate.ID, BookContractRevisionID: &bookContract.ID, StyleSheetRevisionID: &styleSheet.ID, BlueprintRevisionID: &blueprint.ID, EvidencePackHash: "sha256:evidence-v2", PromptHash: "sha256:prompt-v2", ProviderName: generationTarget.ProviderName, ModelName: generationTarget.ModelName, ProviderUsageJSON: []byte(`{"input_tokens":"unknown"}`), QualityReportJSON: []byte(`{"contract_version":"` + sharedtextbook.SampleQualityContractVersion + `","passed":true}`)})
	require.NoError(t, err)
	artifactID := uuid.New()
	artifactHash := "sha256:" + strings.Repeat("9", 64)
	artifactManifest := sharedtextbook.TeachingArtifactManifest{
		Format: sharedtextbook.TeachingArtifactManifestFormat, ArtifactID: artifactID.String(), RevisionID: candidate.ID.String(), ArtifactHash: artifactHash,
		BookContractHash: "sha256:" + strings.TrimPrefix(bookContract.ContentHash, "sha256:"), StyleSheetHash: "sha256:" + strings.TrimPrefix(styleSheet.ContentHash, "sha256:"),
		Language: "go", ToolchainVersion: "go1.25.4", Commands: []sharedtextbook.VerificationCommand{{Kind: "go_test"}},
	}
	artifactManifestHash, err := sharedtextbook.TeachingArtifactManifestHash(artifactManifest)
	require.NoError(t, err)
	artifactManifestJSON, err := json.Marshal(artifactManifest)
	require.NoError(t, err)
	artifact, err := service.RegisterGeneratedCodeArtifact(ctx, workspace.ID, RegisterGeneratedCodeArtifactInput{
		RevisionID: candidate.ID,
		Artifact: sharedtextbook.CodeArtifact{
			ID: artifactID.String(), RevisionID: candidate.ID.String(), Kind: sharedtextbook.CodeArtifactTeachingImplementation, Language: "go", Entrypoint: "main.go",
			ManifestHash: artifactManifestHash, ArtifactHash: artifactHash, Limitations: []string{"只演示最小路由登记。"}, Status: sharedtextbook.ArtifactStatusUnverified,
		},
		ManifestJSON: artifactManifestJSON,
	})
	require.NoError(t, err)
	lock, err = service.AcquireLock(ctx, workspace.ID, LockInput{ChapterID: chapter.ID, OwnerID: owner, ExpectedVersion: candidate.RevisionNumber, LeaseDuration: time.Minute})
	require.NoError(t, err)
	assertDelegatedApproval(t, database, workspace.ID, owner, candidate, lock.Version, generationTarget)
	approved, err := service.ApplyCandidate(ctx, workspace.ID, ApplyCandidateInput{ChapterID: chapter.ID, CandidateRevisionID: candidate.ID, ExpectedVersion: candidate.RevisionNumber, LockOwnerID: owner, LockVersion: lock.Version, ReviewNote: "已完成人工审阅并确认候选稿。", DimensionScores: passingSampleReviewScores()})
	require.NoError(t, err)
	require.Equal(t, sharedtextbook.RevisionKindApproved, approved.Kind)
	require.Equal(t, candidate.Markdown, approved.Markdown)
	require.Equal(t, candidate.ID, *approved.ParentRevisionID)
	require.Equal(t, blueprint.ID, *approved.BlueprintRevisionID)
	require.Equal(t, candidate.PromptHash, approved.PromptHash)
	var approvalReview CandidateReview
	require.NoError(t, database.Where("candidate_revision_id = ?", candidate.ID).First(&approvalReview).Error)
	require.Equal(t, CandidateDecisionApproved, approvalReview.Decision)
	require.Equal(t, sharedtextbook.SampleQualityContractVersion, approvalReview.QualityContractVersion)
	var humanReview sharedtextbook.SampleHumanReview
	require.NoError(t, json.Unmarshal(approvalReview.HumanReviewJSON, &humanReview))
	require.NoError(t, humanReview.Validate())
	progress, err = service.GetProjectProgress(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Equal(t, "approved", progress.Stages[2].Status, "only an approved revision with the current blueprint completes sample approval")
	assertProgressKeepsApprovedManuscriptWithMissingSampleProfile(t, database, workspace.ID, project.ID, bookContract.ID)
	require.Equal(t, ProjectStageState{Key: "verification", Status: "in_progress", Reason: "已有 1 份生成教学工件，但尚无当前有效的隔离运行证据；请勿将预期输出标为已实测。"}, progress.Stages[4])
	require.Equal(t, ProjectStageState{Key: "learning", Status: "ready", Reason: "已有 1 章批准母稿，可从批准修订创建六维学习目标；到期任务仅在打开应用时检查。"}, progress.Stages[5])
	require.Equal(t, ProjectStageState{Key: "publication", Status: "ready", Reason: "已有 1 章批准母稿，可冻结待审构建；它仍需技术、权利、文字、版式和试学审校。"}, progress.Stages[6])
	build, err := service.CreateBookBuild(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Equal(t, sharedtextbook.BookBuildReadyForReview, build.Status)
	require.NotEmpty(t, build.ManifestHash)
	var frozenManifest struct {
		InputHash string `json:"input_hash"`
	}
	require.NoError(t, json.Unmarshal(build.ManifestJSON, &frozenManifest))
	require.True(t, strings.HasPrefix(frozenManifest.InputHash, "sha256:"))
	buildRetry, err := service.CreateBookBuild(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Equal(t, build.ID, buildRetry.ID, "the same frozen inputs must reuse the first build despite built_at")
	require.Equal(t, build.ManifestHash, buildRetry.ManifestHash)
	testBookBuildFreezesQualityReports(t, database, workspace.ID, project.ID, build)
	testBookBuildFreezesVideoRunbooks(t, database, workspace.ID, project.ID, build)
	testFrozenBookCitations(t, database, workspace.ID, project, approved, chunk.ID)
	testFrozenBookNotices(t, database, workspace.ID, project, approved)
	testDelegatedPublicationReview(t, database, workspace.ID, build)
	testHumanPublicationRevisions(t, database, workspace.ID, build)
	testRightsAmendments(t, database, workspace.ID, build)
	testAssetFontAmendmentPersistence(t, database, workspace.ID, build)
	projectWorkspace, err = service.GetProjectWorkspace(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.NotNil(t, projectWorkspace.LatestBookBuild)
	require.Equal(t, build.ID, projectWorkspace.LatestBookBuild.ID, "a refresh must recover the latest frozen build")
	editorial, err := service.GetEditorialWorkspace(ctx, workspace.ID, build.ID)
	require.NoError(t, err)
	require.False(t, editorial.Preflight.Passed)
	require.Empty(t, editorial.RightsItems)
	require.Equal(t, []sharedtextbook.RightsSubject{
		{SubjectRef: "chapter-revision:" + approved.ID.String(), WorkType: sharedtextbook.RightsWorkTypeProse},
		{SubjectRef: "code-artifact:" + artifact.ID.String(), WorkType: sharedtextbook.RightsWorkTypeCode},
	}, editorial.RequiredRightsSubjects)
	require.Len(t, editorial.AutomatedChecks, 5)
	require.Equal(t, sharedtextbook.QualityStatusHardFail, editorial.AutomatedChecks[3].Status, "an unverified frozen artifact must block publication")
	_, err = service.GetEditorialWorkspace(ctx, uuid.New(), build.ID)
	require.ErrorIs(t, err, ErrNotFound, "another local workspace cannot inspect publication evidence")
	_, err = service.PromoteBookBuild(ctx, workspace.ID, build.ID)
	require.ErrorIs(t, err, ErrInvalidState, "promotion must fail closed before human and rights evidence exists")
	rightsInput := AddRightsItemInput{BuildID: build.ID, SubjectRef: "chapter-revision:" + approved.ID.String(), WorkType: sharedtextbook.RightsWorkTypeProse, RightsBasis: "作者原创并已核对引用范围", AllowedUse: "本地教材编辑、导出与送审", Attribution: "InkWords 本地作者", PublicationStatus: sharedtextbook.RightsStatusReady}
	rightsItem, err := service.AddRightsItem(ctx, workspace.ID, rightsInput)
	require.NoError(t, err)
	rightsAgain, err := service.AddRightsItem(ctx, workspace.ID, rightsInput)
	require.NoError(t, err)
	require.Equal(t, rightsItem.ID, rightsAgain.ID, "an exact rights retry is idempotent")
	rightsInput.Attribution = "试图改写不可变权利记录"
	_, err = service.AddRightsItem(ctx, workspace.ID, rightsInput)
	require.ErrorIs(t, err, ErrInvalidState)
	_, err = service.AddRightsItem(ctx, workspace.ID, AddRightsItemInput{BuildID: build.ID, SubjectRef: "code-artifact:" + artifact.ID.String(), WorkType: sharedtextbook.RightsWorkTypeCode, RightsBasis: "作者原创教学实现", AllowedUse: "本地教材编辑、导出与送审", Attribution: "InkWords 本地作者", PublicationStatus: sharedtextbook.RightsStatusReady})
	require.NoError(t, err)
	for _, stage := range sharedtextbook.RequiredPublicationReviewStages() {
		input := humanReviewFixture(build, stage)
		review, err := service.CompletePublicationReview(ctx, workspace.ID, input)
		require.NoError(t, err)
		require.False(t, review.Automated)
		reviewAgain, err := service.CompletePublicationReview(ctx, workspace.ID, input)
		require.NoError(t, err)
		require.Equal(t, review.ID, reviewAgain.ID, "an exact human-review retry is idempotent")
	}
	editorial, err = service.GetEditorialWorkspace(ctx, workspace.ID, build.ID)
	require.NoError(t, err)
	require.False(t, editorial.Preflight.Passed)
	require.Len(t, editorial.HumanReviews, 8)
	_, err = service.PromoteBookBuild(ctx, workspace.ID, build.ID)
	require.ErrorIs(t, err, ErrInvalidState, "human and rights evidence cannot override an unverified frozen artifact")
	require.NoError(t, database.Model(&CodeArtifactRow{}).Where("id = ?", artifact.ID).Update("status", sharedtextbook.ArtifactStatusVerified).Error)
	editorial, err = service.GetEditorialWorkspace(ctx, workspace.ID, build.ID)
	require.NoError(t, err)
	require.False(t, editorial.Preflight.Passed, "a verified status without a current observation is not publication evidence")
	progress, err = service.GetProjectProgress(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Equal(t, "in_progress", progress.Stages[4].Status, "artifact status alone must not make the verification stage approved")
	runnerDigest := "sha256:" + strings.Repeat("8", 64)
	verificationInputHash, err := sharedtextbook.TeachingArtifactVerificationInputHash(artifactManifest, runnerDigest)
	require.NoError(t, err)
	structuredOutput, err := sharedtextbook.MarshalRuntimeObservationOutput(map[string]any{"command": artifactManifest.Commands[0], "status": "verified", "exit_code": 0}, nil)
	require.NoError(t, err)
	capturedAt, expiresAt := time.Now().UTC(), time.Now().UTC().Add(time.Hour)
	runtimeEvidence := RuntimeEvidenceRow{
		RevisionID: candidate.ID, CodeArtifactID: artifact.ID, CodeArtifactHash: artifact.ArtifactHash, InputHash: verificationInputHash,
		Kind: sharedtextbook.RuntimeEvidenceTerminalOutput, Status: sharedtextbook.ArtifactStatusVerified, CommandManifestHash: artifactManifestHash,
		RunnerImageDigest: runnerDigest, ToolchainVersion: artifactManifest.ToolchainVersion, ToolName: "bubblewrap", ToolVersion: "runner-image:" + runnerDigest,
		SamplingConditionsJSON: datatypes.JSON([]byte(`["network disabled"]`)), StructuredOutput: string(structuredOutput), RawEvidenceRef: "database:runtime-evidence", CapturedAt: &capturedAt, ExpiresAt: &expiresAt,
	}
	require.NoError(t, database.Create(&runtimeEvidence).Error)
	build = testBookBuildFreezesRuntimeEvidence(t, database, service, workspace.ID, project.ID, build, runtimeEvidence)
	editorial, err = service.GetEditorialWorkspace(ctx, workspace.ID, build.ID)
	require.NoError(t, err)
	require.True(t, editorial.Preflight.Passed)
	progress, err = service.GetProjectProgress(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Equal(t, "approved", progress.Stages[4].Status)
	build, err = service.PromoteBookBuild(ctx, workspace.ID, build.ID)
	require.NoError(t, err)
	require.Equal(t, sharedtextbook.BookBuildPublicationCandidate, build.Status)
	buildAgain, err := service.PromoteBookBuild(ctx, workspace.ID, build.ID)
	require.NoError(t, err)
	require.Equal(t, build.ID, buildAgain.ID, "promotion is explicit but safely idempotent")
	lockedReview := humanReviewFixture(build, sharedtextbook.PublicationReviewTechnical)
	lockedReview.ExpectedRevision = 1
	_, err = service.CompletePublicationReview(ctx, workspace.ID, lockedReview)
	require.ErrorIs(t, err, ErrInvalidState, "publication-candidate evidence is frozen")
	progress, err = service.GetProjectProgress(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Equal(t, ProjectStageState{Key: "publication", Status: "approved", Reason: "已有 1 个构建通过 InkWords 内部出版预检；这不代表出版社、ISBN 或 CIP 批准。"}, progress.Stages[6])
	newerBookContract, err := service.CreateBookContract(ctx, workspace.ID, CreateBookContractInput{ProjectID: project.ID, Reader: sharedtextbook.ReaderModel{Audience: sharedtextbook.AudienceFoundation, LearningOutcomes: []string{"解释路由登记", "定位路由故障"}}, Promise: "更新后的教材承诺。", ChapterProfiles: []sharedtextbook.ChapterProfile{sharedtextbook.ChapterProfileConcept, sharedtextbook.ChapterProfileTroubleshooting}, TerminologyVersion: "terms-2", PublicationProfile: "personal_learning"})
	require.NoError(t, err)
	_, err = service.ApproveBookContract(ctx, workspace.ID, project.ID, newerBookContract.ID)
	require.NoError(t, err)
	projectWorkspace, err = service.GetProjectWorkspace(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Nil(t, projectWorkspace.Project.ApprovedBlueprintRevisionID, "changing a generation contract invalidates its old blueprint")
	progress, err = service.GetProjectProgress(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Equal(t, "ready", progress.Stages[1].Status, "the stale approved blueprint must never remain eligible")
	_, err = service.ApproveBlueprint(ctx, workspace.ID, project.ID, blueprint.ID)
	require.ErrorIs(t, err, ErrInvalidState, "a blueprint cannot be reapproved after one of its bound contracts changes")
	_, err = service.AppendRevision(ctx, workspace.ID, AppendRevisionInput{ChapterID: chapter.ID, ExpectedVersion: approved.RevisionNumber, Kind: sharedtextbook.RevisionKindCandidate, Markdown: "outdated-contract", DocumentJSON: []byte(`{"blocks":[]}`), ContentHash: strings.Repeat("f", 64), CreatedBy: RevisionCreatorGeneration, ParentRevisionID: &approved.ID, BookContractRevisionID: &bookContract.ID, StyleSheetRevisionID: &styleSheet.ID, BlueprintRevisionID: &blueprint.ID, EvidencePackHash: "sha256:evidence", PromptHash: "sha256:prompt", ProviderName: "fake", ModelName: "deterministic-gin-fixture", ProviderUsageJSON: []byte(`{"input_tokens":"unknown"}`), QualityReportJSON: []byte(`{"passed":true}`)})
	require.ErrorIs(t, err, ErrInvalidState, "a candidate cannot be generated from a superseded approved contract")
	chapterWorkspace, err = service.GetChapterWorkspace(ctx, workspace.ID, chapter.ID)
	require.NoError(t, err)
	require.Equal(t, sharedtextbook.CurrentSampleHumanReviewContract(), chapterWorkspace.HumanReviewContract)
	require.NotNil(t, chapterWorkspace.Chapter.ApprovedRevisionID)
	require.Equal(t, approved.ID, *chapterWorkspace.Chapter.ApprovedRevisionID)
	require.Len(t, chapterWorkspace.Revisions, 8)
	require.Len(t, chapterWorkspace.CandidateReviews, 2)
	decisions := make(map[string]CandidateReview, len(chapterWorkspace.CandidateReviews))
	for _, candidateReview := range chapterWorkspace.CandidateReviews {
		decisions[candidateReview.Decision] = candidateReview
	}
	require.Equal(t, review.ID, decisions[CandidateDecisionRejected].ID)
	require.Equal(t, approved.ID, *chapterWorkspace.Chapter.ApprovedRevisionID)
	require.Equal(t, candidate.ID, decisions[CandidateDecisionApproved].CandidateRevisionID)
	var persistedHumanReview sharedtextbook.SampleHumanReview
	require.NoError(t, json.Unmarshal(decisions[CandidateDecisionApproved].HumanReviewJSON, &persistedHumanReview))
	require.NoError(t, persistedHumanReview.Validate())
	require.Len(t, chapterWorkspace.CodeArtifacts, 1)
	require.Equal(t, artifact.ID, chapterWorkspace.CodeArtifacts[0].ID)
	require.Len(t, chapterWorkspace.RuntimeEvidence, 1)
	require.Equal(t, runtimeEvidence.ID, chapterWorkspace.RuntimeEvidence[0].ID)
	require.Equal(t, approved.ID, chapterWorkspace.Revisions[0].ID)
	require.NotNil(t, chapterWorkspace.Lock)
	_, err = service.AppendRevision(ctx, workspace.ID, AppendRevisionInput{ChapterID: chapter.ID, ExpectedVersion: approved.RevisionNumber, Kind: sharedtextbook.RevisionKindApproved, Markdown: "overwrite", DocumentJSON: []byte(`{"blocks":[]}`), ContentHash: strings.Repeat("e", 64), CreatedBy: RevisionCreatorManual, LockOwnerID: owner, LockVersion: lock.Version})
	require.ErrorIs(t, err, ErrInvalidState)

	_, err = service.AcquireLock(ctx, workspace.ID, LockInput{ChapterID: chapter.ID, OwnerID: uuid.New(), ExpectedVersion: approved.RevisionNumber, LeaseDuration: time.Minute})
	require.ErrorIs(t, err, ErrRevisionLocked)

	fixture, err := service.LoadGinFixture(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Equal(t, ginFixtureCommit, fixture.ResolvedVersion)
	fixtureAgain, err := service.LoadGinFixture(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Equal(t, fixture.ID, fixtureAgain.ID, "the same fixed document bundle must be idempotent")
	library, err = service.ListSourceLibrary(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Len(t, library, 4)
	fixtureChunkCounts := map[string]int64{}
	for _, candidate := range library {
		if candidate.CanonicalLocator == "routergroup.go" || candidate.CanonicalLocator == "gin.go" || candidate.CanonicalLocator == "tree.go" {
			fixtureChunkCounts[candidate.CanonicalLocator] = candidate.ChunkCount
		}
	}
	require.Equal(t, map[string]int64{"routergroup.go": 3, "gin.go": 2, "tree.go": 1}, fixtureChunkCounts)
	fixtureEvidence, err := service.ListSourceEvidence(ctx, workspace.ID, project.ID)
	require.NoError(t, err)
	require.Len(t, fixtureEvidence, 7)
	var fixtureSnapshotCount int64
	require.NoError(t, database.Model(&SourceSnapshot{}).Where("source_id = ? AND resolved_version = ?", *project.PrimarySourceID, ginFixtureCommit).Count(&fixtureSnapshotCount).Error)
	require.Equal(t, int64(3), fixtureSnapshotCount, "a second load must reuse all three immutable file snapshots")

	_, err = service.CreateChapter(ctx, uuid.New(), CreateChapterInput{ProjectID: project.ID, SortOrder: 2, Title: "越权", ChapterProfile: sharedtextbook.ChapterProfileConcept})
	require.ErrorIs(t, err, ErrNotFound)
	projects, err := service.repository.ListProjects(ctx, workspace.ID)
	require.NoError(t, err)
	require.Len(t, projects, 1)
	assertApprovedPracticeEvidence(t, ctx, database, service, workspace.ID, *chapter, chunk)
	require.NoError(t, service.repository.SoftDeleteProject(ctx, workspace.ID, project.ID))
	projects, err = service.repository.ListProjects(ctx, workspace.ID)
	require.NoError(t, err)
	require.Empty(t, projects)
}

func TestMissingChapterProfilesRequiresOneApprovedSamplePerContractProfile(t *testing.T) {
	required := []sharedtextbook.ChapterProfile{sharedtextbook.ChapterProfileConcept, sharedtextbook.ChapterProfileHandsOn, sharedtextbook.ChapterProfileTroubleshooting}
	approved := []sharedtextbook.ChapterProfile{sharedtextbook.ChapterProfileConcept, sharedtextbook.ChapterProfileConcept, sharedtextbook.ChapterProfileTroubleshooting}
	require.Equal(t, []sharedtextbook.ChapterProfile{sharedtextbook.ChapterProfileHandsOn}, missingChapterProfiles(required, approved))
	require.Equal(t, "动手实践", joinChapterProfiles([]sharedtextbook.ChapterProfile{sharedtextbook.ChapterProfileHandsOn}))
}

func passingSampleReviewScores() []sharedtextbook.DimensionScore {
	dimensions := sharedtextbook.SampleManualReviewDimensions()
	scores := make([]sharedtextbook.DimensionScore, 0, len(dimensions))
	for _, dimension := range dimensions {
		scores = append(scores, sharedtextbook.DimensionScore{Dimension: dimension, Score: 3})
	}
	return scores
}
