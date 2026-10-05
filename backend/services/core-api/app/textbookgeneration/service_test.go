package textbookgeneration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	coretask "inkwords-backend/services/core-api/domain/task"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type fixedPayloadPreparer struct {
	payload sharedtextbook.SampleGenerationTaskPayload
	target  sharedtextbook.SampleGenerationTarget
}

func (preparer *fixedPayloadPreparer) PrepareSampleGeneration(_ context.Context, _, _, _ uuid.UUID, target sharedtextbook.SampleGenerationTarget) (sharedtextbook.SampleGenerationTaskPayload, error) {
	preparer.target = target
	return preparer.payload, nil
}

type capturedGenerationTaskCreator struct {
	input coretask.CreateGenerationTaskInput
	calls int
}

func (creator *capturedGenerationTaskCreator) CreateGenerationTask(_ context.Context, input coretask.CreateGenerationTaskInput) (coretask.JobTask, error) {
	creator.input = input
	creator.calls++
	return coretask.JobTask{ID: uuid.New(), Status: coretask.JobTaskStatusQueued}, nil
}

func TestCreateSampleGenerationTaskUsesFrozenInputAsIdempotencyKey(t *testing.T) {
	workspaceID := uuid.New()
	payload := textbookGenerationPayload()
	preparer := &fixedPayloadPreparer{payload: payload}
	creator := &capturedGenerationTaskCreator{}
	service := NewService(preparer, creator)

	preflight, err := service.PrepareSampleGeneration(context.Background(), workspaceID, uuid.New(), uuid.New())
	require.NoError(t, err)
	require.Equal(t, payload.InputHash, preflight.InputHash)
	require.Equal(t, 1, preflight.EvidenceReferenceCount)
	require.True(t, preflight.WithinBudget)
	require.True(t, preflight.RequiresConfirmation)
	require.Equal(t, "unknown_before_worker", preflight.CacheStatus)
	require.False(t, preflight.EstimatedCostKnown)
	require.Zero(t, creator.calls, "read-only preflight must not create a task")

	chapterID := uuid.New()
	_, err = service.CreateSampleGenerationTask(context.Background(), workspaceID, uuid.New(), chapterID, preflight.InputHash)
	require.NoError(t, err)
	require.Equal(t, 1, creator.calls)
	require.Equal(t, workspaceID, creator.input.WorkspaceID)
	require.NotNil(t, creator.input.TextbookChapterID)
	require.Equal(t, chapterID, *creator.input.TextbookChapterID)
	require.Equal(t, sharedtextbook.TextbookSampleGenerationTaskSubtype, creator.input.TaskSubtype)
	require.Equal(t, payload.GenerationTarget, preparer.target)
	require.Equal(t, "textbook-sample:"+payload.InputHash, creator.input.IdempotencyKey)
	require.JSONEq(t, string(mustMarshalPayload(t, payload)), string(creator.input.Payload))
}

func TestCreateSampleGenerationTaskRejectsMissingOrStaleConfirmationBeforeQueueing(t *testing.T) {
	payload := textbookGenerationPayload()
	creator := &capturedGenerationTaskCreator{}
	service := NewService(&fixedPayloadPreparer{payload: payload}, creator)

	for _, confirmation := range []string{"", "sha256:" + strings.Repeat("b", 64)} {
		_, err := service.CreateSampleGenerationTask(context.Background(), uuid.New(), uuid.New(), uuid.New(), confirmation)
		require.ErrorIs(t, err, textbookdomain.ErrInvalidState)
	}
	require.Zero(t, creator.calls)
}

func TestSampleGenerationTargetFromConfigRequiresAnExplicitExternalModel(t *testing.T) {
	fixture, err := SampleGenerationTargetFromConfig("fake", "")
	require.NoError(t, err)
	require.Equal(t, sharedtextbook.SampleFixtureProviderName, fixture.ProviderName)
	require.Equal(t, sharedtextbook.SampleFixtureModelName, fixture.ModelName)

	deepseek, err := SampleGenerationTargetFromConfig("DeepSeek", "deepseek-v4-flash")
	require.NoError(t, err)
	require.Equal(t, sharedtextbook.SampleGenerationTarget{ProviderName: "deepseek", ModelName: "deepseek-v4-flash"}, deepseek)

	_, err = SampleGenerationTargetFromConfig("deepseek", "")
	require.ErrorContains(t, err, "target is incomplete")
	_, err = SampleGenerationTargetFromConfig("unknown", "model")
	require.ErrorContains(t, err, "unsupported")
}

func textbookGenerationPayload() sharedtextbook.SampleGenerationTaskPayload {
	digest := "sha256:" + strings.Repeat("a", 64)
	primary := sharedtextbook.SourceSnapshot{ID: "snapshot-1", SourceID: "source-1", Kind: sharedtextbook.SourceKindGitRepository, Role: sharedtextbook.SourceRolePrimary, Locator: "https://github.com/gin-gonic/gin", ResolvedVersion: strings.Repeat("b", 40), ContentHash: digest, CapturedAt: time.Unix(1, 0).UTC()}
	evidence := sharedtextbook.EvidenceRef{ID: "evidence-1", SnapshotID: primary.ID, DocumentID: "document-1", ChunkID: "chunk-1", Locator: sharedtextbook.EvidenceLocator{Path: "router.go", StartLine: 1, EndLine: 2}, ContentHash: digest, Confidence: sharedtextbook.EvidenceConfidenceObserved, SourceRole: sharedtextbook.SourceRolePrimary}
	book := sharedtextbook.BookContract{RevisionID: "book-1", ProjectID: "project-1", RevisionNumber: 1, ContentHash: digest, Promise: "从零讲清楚路由。", Reader: sharedtextbook.ReaderModel{Audience: sharedtextbook.AudienceFoundation, LearningOutcomes: []string{"解释路由"}}, ChapterProfiles: []sharedtextbook.ChapterProfile{sharedtextbook.ChapterProfileConcept}, TerminologyVersion: "v1", PublicationProfile: "personal_learning"}
	style := sharedtextbook.StyleSheet{RevisionID: "style-1", ProjectID: "project-1", RevisionNumber: 1, ContentHash: digest, Language: "zh-CN", TerminologyRules: []string{"先白话"}, CodeRules: []string{"标记验证"}, CitationRules: []string{"引用证据"}}
	blueprint := sharedtextbook.Blueprint{RevisionID: "blueprint-1", ProjectID: "project-1", RevisionNumber: 1, ContentHash: digest, BookContractRevision: book.RevisionID, StyleSheetRevision: style.RevisionID, Volumes: []sharedtextbook.BlueprintVolume{{ID: "volume-1", Title: "入门", Sort: 1, Chapters: []sharedtextbook.BlueprintChapter{{ID: "chapter-1", Title: "路由", Sort: 1, Profile: sharedtextbook.ChapterProfileConcept, EvidenceIDs: []string{evidence.ChunkID}}}}}}
	payload := sharedtextbook.SampleGenerationTaskPayload{TaskVersion: sharedtextbook.SampleGenerationTaskVersion, TaskSubtype: sharedtextbook.TextbookSampleGenerationTaskSubtype, PromptSchemaVersion: sharedtextbook.SamplePromptSchemaVersion, QualityContractVersion: sharedtextbook.SampleQualityContractVersion, GenerationTarget: sharedtextbook.SampleGenerationTarget{ProviderName: sharedtextbook.SampleFixtureProviderName, ModelName: sharedtextbook.SampleFixtureModelName}, ProjectID: "project-1", ChapterID: "chapter-1", Audience: sharedtextbook.AudienceFoundation, BookContract: book, StyleSheet: style, Blueprint: blueprint, EvidencePack: sharedtextbook.GenerationEvidencePack{PrimarySnapshot: primary, Evidence: []sharedtextbook.EvidenceRef{evidence}, Excerpts: map[string]string{evidence.ID: "router source"}}}
	payload.InputHash = sharedtextbook.SampleGenerationInputHash(payload)
	return payload
}

func mustMarshalPayload(t *testing.T, payload sharedtextbook.SampleGenerationTaskPayload) []byte {
	t.Helper()
	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	return encoded
}
