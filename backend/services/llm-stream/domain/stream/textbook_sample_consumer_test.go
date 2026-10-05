package stream

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

type fakeTextbookSampleRunner struct {
	result sharedtextbook.SampleGenerationTaskResult
	err    error
}

func (r fakeTextbookSampleRunner) Run(context.Context, sharedtextbook.SampleGenerationTaskPayload) (sharedtextbook.SampleGenerationTaskResult, error) {
	return r.result, r.err
}

type sampleFailureEvidenceError struct {
	failure sharedtextbook.SampleGenerationTaskFailureResult
}

func (err sampleFailureEvidenceError) Error() string {
	return "generated textbook candidate failed quality gates: undefined_repeated_acronym: POST"
}

func (err sampleFailureEvidenceError) FailureResult() sharedtextbook.SampleGenerationTaskFailureResult {
	return err.failure
}

type reusableTextbookTaskService struct {
	fakeTaskService
	result []byte
	key    string
}

func (service *reusableTextbookTaskService) FindCompletedTextbookStageResult(_ context.Context, _ uuid.UUID, stageExecutionKey string) ([]byte, bool, error) {
	service.key = stageExecutionKey
	return append([]byte(nil), service.result...), true, nil
}

type countingTextbookSampleRunner struct {
	calls int
}

func (runner *countingTextbookSampleRunner) Run(context.Context, sharedtextbook.SampleGenerationTaskPayload) (sharedtextbook.SampleGenerationTaskResult, error) {
	runner.calls++
	return sharedtextbook.SampleGenerationTaskResult{}, nil
}

func TestTaskConsumerRunsTextbookSampleAsTaskOnlyResult(t *testing.T) {
	payload := streamSamplePayload()
	result := sharedtextbook.SampleGenerationTaskResult{ResultVersion: 1, TaskSubtype: sharedtextbook.TextbookSampleGenerationTaskSubtype, ProjectID: payload.ProjectID, ChapterID: payload.ChapterID, ExpectedChapterVersion: payload.ExpectedChapterVersion, InputHash: payload.InputHash, Markdown: "# 样章", DocumentJSON: []byte(`{"format":"inkwords.chapter.v1"}`), ContentHash: taskTestDigest("# 样章"), BookContractRevision: payload.BookContract.RevisionID, StyleSheetRevision: payload.StyleSheet.RevisionID, BlueprintRevision: payload.Blueprint.RevisionID, EvidencePackHash: sharedtextbook.GenerationEvidencePackHash(payload.EvidencePack), PromptHash: "sha256:prompt", ProviderName: sharedtextbook.SampleFixtureProviderName, ModelName: sharedtextbook.SampleFixtureModelName, ProviderUsageJSON: []byte(`{}`), QualityReportJSON: []byte(`{"contract_version":"` + sharedtextbook.SampleQualityContractVersion + `","passed":true}`)}
	tasks := &fakeTaskService{}
	consumer := NewTaskConsumer(tasks, &fakeStreamService{}).WithTextbookSampleRunner(fakeTextbookSampleRunner{result: result})
	taskID, workspaceID := uuid.New(), uuid.New()
	err := consumer.HandleGenerationRequested(context.Background(), sharedrabbitmq.GenerationRequestedMessage{TaskID: taskID, Kind: sharedtextbook.TextbookSampleGenerationTaskSubtype, WorkspaceID: &workspaceID, Payload: mustMarshalSamplePayload(t, payload)})
	require.NoError(t, err)
	require.Equal(t, TaskStatusSucceeded, tasks.lastStatus)
	require.Contains(t, string(tasks.lastResult), `"task_subtype":"textbook_sample_generate"`)
	require.Len(t, tasks.appendEvents, 2)
	require.Contains(t, string(tasks.appendEvents[0].Payload), `"checkpoint":"started"`)
	require.Contains(t, string(tasks.appendEvents[0].Payload), `"stage_execution_key":"`+sharedtextbook.TaskStageInputHash(taskID.String(), sharedtextbook.SampleGenerationStage, payload.InputHash)+`"`)
	require.Contains(t, string(tasks.appendEvents[1].Payload), `"checkpoint":"result_ready"`)
	require.Contains(t, string(tasks.appendEvents[1].Payload), `"result":{"result_version":1`)
}

func TestTaskConsumerRestoresCompletedTextbookStageOnRetryWithoutRunningModel(t *testing.T) {
	payload := streamSamplePayload()
	taskID, workspaceID := uuid.New(), uuid.New()
	cachedResult := []byte(`{"task_subtype":"textbook_sample_generate"}`)
	tasks := &reusableTextbookTaskService{result: cachedResult}
	runner := &countingTextbookSampleRunner{}
	consumer := NewTaskConsumer(tasks, &fakeStreamService{}).WithTextbookSampleRunner(runner)

	err := consumer.HandleGenerationRequested(context.Background(), sharedrabbitmq.GenerationRequestedMessage{TaskID: taskID, Kind: sharedtextbook.TextbookSampleGenerationTaskSubtype, WorkspaceID: &workspaceID, Payload: mustMarshalSamplePayload(t, payload)})
	require.NoError(t, err)
	require.Equal(t, sharedtextbook.TaskStageInputHash(taskID.String(), sharedtextbook.SampleGenerationStage, payload.InputHash), tasks.key)
	require.Zero(t, runner.calls)
	require.False(t, tasks.markRunningCalled)
	require.Empty(t, tasks.appendEvents)
	require.Equal(t, TaskStatusSucceeded, tasks.lastStatus)
	require.JSONEq(t, string(cachedResult), string(tasks.lastResult))
}

func TestTaskConsumerRetainsSafeEvidenceWhenSampleQualityRejectsOutput(t *testing.T) {
	payload := streamSamplePayload()
	failure := sharedtextbook.SampleGenerationTaskFailureResult{
		ResultVersion: 1, TaskSubtype: sharedtextbook.TextbookSampleGenerationTaskSubtype, FinalStatus: "failed",
		ProjectID: payload.ProjectID, ChapterID: payload.ChapterID, InputHash: payload.InputHash,
		ProviderName: payload.GenerationTarget.ProviderName, ModelName: payload.GenerationTarget.ModelName,
		ProviderUsageJSON: []byte(`{"known":true,"input_tokens":101,"output_tokens":202,"provider_call_count":1}`),
		QualityReportJSON: []byte(`{"contract_version":"` + sharedtextbook.SampleQualityContractVersion + `","passed":false,"failures":["undefined_repeated_acronym: POST"]}`),
		PromptHash:        "sha256:prompt", CandidatePersisted: false,
	}
	tasks := &fakeTaskService{}
	runner := fakeTextbookSampleRunner{err: sampleFailureEvidenceError{failure: failure}}
	consumer := NewTaskConsumer(tasks, &fakeStreamService{}).WithTextbookSampleRunner(runner)
	taskID, workspaceID := uuid.New(), uuid.New()

	err := consumer.HandleGenerationRequested(context.Background(), sharedrabbitmq.GenerationRequestedMessage{TaskID: taskID, Kind: sharedtextbook.TextbookSampleGenerationTaskSubtype, WorkspaceID: &workspaceID, Payload: mustMarshalSamplePayload(t, payload)})
	require.NoError(t, err)
	require.Equal(t, TaskStatusFailed, tasks.lastStatus)
	require.Len(t, tasks.appendEvents, 2)
	require.Contains(t, string(tasks.appendEvents[1].Payload), `"checkpoint":"quality_rejected"`)
	require.Contains(t, string(tasks.appendEvents[1].Payload), `"input_tokens":101`)
	require.NotContains(t, string(tasks.appendEvents[1].Payload), "markdown")
}

func TestTaskConsumerRejectsMissingOrMismatchedTextbookWorkspaceBeforeModel(t *testing.T) {
	payload := streamSamplePayload()
	workspaceID := uuid.New()
	for _, test := range []struct {
		name      string
		workspace *uuid.UUID
		mismatch  bool
	}{
		{name: "missing"},
		{name: "mismatched", workspace: &workspaceID, mismatch: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			tasks := &fakeTaskService{workspaceMismatch: test.mismatch}
			runner := &countingTextbookSampleRunner{}
			consumer := NewTaskConsumer(tasks, &fakeStreamService{}).WithTextbookSampleRunner(runner)
			err := consumer.HandleGenerationRequested(t.Context(), sharedrabbitmq.GenerationRequestedMessage{TaskID: uuid.New(), Kind: sharedtextbook.TextbookSampleGenerationTaskSubtype, WorkspaceID: test.workspace, Payload: mustMarshalSamplePayload(t, payload)})
			require.NoError(t, err)
			require.Equal(t, TaskStatusFailed, tasks.lastStatus)
			require.False(t, tasks.markRunningCalled)
			require.Zero(t, runner.calls)
		})
	}
}

func streamSamplePayload() sharedtextbook.SampleGenerationTaskPayload {
	primary := sharedtextbook.SourceSnapshot{ID: "snapshot-primary", SourceID: "source-primary", Kind: sharedtextbook.SourceKindGitRepository, Role: sharedtextbook.SourceRolePrimary, Locator: "https://github.com/example/project", ResolvedVersion: "0123456789abcdef0123456789abcdef01234567", ContentHash: "sha256:primary", CapturedAt: time.Unix(1, 0).UTC()}
	book := sharedtextbook.BookContract{RevisionID: "book-r1", ProjectID: "project-1", RevisionNumber: 1, ContentHash: "sha256:book", Promise: "讲清楚路由。", Reader: sharedtextbook.ReaderModel{Audience: sharedtextbook.AudienceFoundation, LearningOutcomes: []string{"解释路由"}}, ChapterProfiles: []sharedtextbook.ChapterProfile{sharedtextbook.ChapterProfileConcept}, TerminologyVersion: "v1", PublicationProfile: "personal_learning"}
	style := sharedtextbook.StyleSheet{RevisionID: "style-r1", ProjectID: "project-1", RevisionNumber: 1, ContentHash: "sha256:style", Language: "zh-CN", TerminologyRules: []string{"先白话"}, CodeRules: []string{"标记验证"}, CitationRules: []string{"引用证据"}}
	blueprint := sharedtextbook.Blueprint{RevisionID: "blueprint-r1", ProjectID: "project-1", RevisionNumber: 1, ContentHash: "sha256:blueprint", BookContractRevision: book.RevisionID, StyleSheetRevision: style.RevisionID, Volumes: []sharedtextbook.BlueprintVolume{{ID: "volume-1", Title: "开始", Sort: 1, Chapters: []sharedtextbook.BlueprintChapter{{ID: "chapter-1", Title: "路由", Sort: 1, Profile: sharedtextbook.ChapterProfileConcept, EvidenceIDs: []string{"chunk-1"}}}}}}
	evidence := sharedtextbook.EvidenceRef{ID: "evidence-chunk-1", SnapshotID: primary.ID, DocumentID: "document-1", ChunkID: "chunk-1", Locator: sharedtextbook.EvidenceLocator{Path: "router.go", StartLine: 1, EndLine: 3}, ContentHash: "sha256:chunk", Confidence: sharedtextbook.EvidenceConfidenceDocumented, SourceRole: sharedtextbook.SourceRolePrimary}
	payload := sharedtextbook.SampleGenerationTaskPayload{TaskVersion: sharedtextbook.SampleGenerationTaskVersion, TaskSubtype: sharedtextbook.TextbookSampleGenerationTaskSubtype, PromptSchemaVersion: sharedtextbook.SamplePromptSchemaVersion, QualityContractVersion: sharedtextbook.SampleQualityContractVersion, GenerationTarget: sharedtextbook.SampleGenerationTarget{ProviderName: sharedtextbook.SampleFixtureProviderName, ModelName: sharedtextbook.SampleFixtureModelName}, ProjectID: "project-1", ChapterID: "chapter-1", Audience: sharedtextbook.AudienceFoundation, BookContract: book, StyleSheet: style, Blueprint: blueprint, EvidencePack: sharedtextbook.GenerationEvidencePack{PrimarySnapshot: primary, Evidence: []sharedtextbook.EvidenceRef{evidence}, Excerpts: map[string]string{evidence.ID: "路由登记源码"}}}
	payload.InputHash = sharedtextbook.SampleGenerationInputHash(payload)
	return payload
}

func mustMarshalSamplePayload(t *testing.T, payload sharedtextbook.SampleGenerationTaskPayload) []byte {
	t.Helper()
	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	return encoded
}

func taskTestDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}
