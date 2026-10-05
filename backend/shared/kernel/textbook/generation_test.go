package textbook

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSampleGenerationTaskPayloadRejectsUnfrozenOrMismatchedEvidence(t *testing.T) {
	payload := validSampleGenerationTaskPayload()
	payload.InputHash = SampleGenerationInputHash(payload)
	require.NoError(t, payload.Validate())

	payload.EvidencePack.Evidence = nil
	payload.InputHash = SampleGenerationInputHash(payload)
	require.ErrorContains(t, payload.Validate(), "generation needs source evidence")

	payload = validSampleGenerationTaskPayload()
	payload.EvidencePack.Evidence[0].ChunkID = "different-chunk"
	payload.InputHash = SampleGenerationInputHash(payload)
	require.ErrorContains(t, payload.Validate(), "omits blueprint evidence")
}

func TestSampleGenerationTaskPayloadDetectsPayloadTampering(t *testing.T) {
	payload := validSampleGenerationTaskPayload()
	payload.InputHash = SampleGenerationInputHash(payload)
	payload.EvidencePack.Excerpts["evidence-chunk-1"] = "changed after task creation"
	require.ErrorContains(t, payload.Validate(), "input hash does not match")
}

func TestSampleGenerationInputHashChangesWithGenerationContracts(t *testing.T) {
	payload := validSampleGenerationTaskPayload()
	baseline := SampleGenerationInputHash(payload)

	payload.PromptSchemaVersion = SamplePromptSchemaVersion + ".changed"
	require.NotEqual(t, baseline, SampleGenerationInputHash(payload))

	payload = validSampleGenerationTaskPayload()
	payload.QualityContractVersion = SampleQualityContractVersion + ".changed"
	require.NotEqual(t, baseline, SampleGenerationInputHash(payload))

	payload = validSampleGenerationTaskPayload()
	payload.GenerationTarget.ModelName = "another-model"
	require.NotEqual(t, baseline, SampleGenerationInputHash(payload))
}

func TestSampleGenerationFailureResultKeepsRejectedTelemetryBoundToFrozenInput(t *testing.T) {
	payload := validSampleGenerationTaskPayload()
	payload.InputHash = SampleGenerationInputHash(payload)
	result := SampleGenerationTaskFailureResult{
		ResultVersion: 1, TaskSubtype: TextbookSampleGenerationTaskSubtype, FinalStatus: "failed",
		ProjectID: payload.ProjectID, ChapterID: payload.ChapterID, InputHash: payload.InputHash,
		ProviderName: payload.GenerationTarget.ProviderName, ModelName: payload.GenerationTarget.ModelName,
		ProviderUsageJSON: json.RawMessage(`{"known":true,"input_tokens":21,"output_tokens":34}`),
		QualityReportJSON: json.RawMessage(`{"contract_version":"` + SampleQualityContractVersion + `","passed":false,"failures":["undefined_repeated_acronym: POST"]}`),
		PromptHash:        "sha256:prompt", CandidatePersisted: false,
	}
	require.NoError(t, result.ValidateAgainst(payload))

	result.CandidatePersisted = true
	result.RejectedDraftStorage = "saved"
	result.RejectedDraftHash = "not-a-hash"
	require.ErrorContains(t, result.ValidateAgainst(payload), "draft hash invalid")
	result.RejectedDraftHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	require.ErrorContains(t, result.ValidateAgainst(payload), "provenance is invalid")
	result.CandidatePersisted = false
	require.NoError(t, result.ValidateAgainst(payload))
	result.RejectedDraftStorage = "unavailable"
	require.ErrorContains(t, result.ValidateAgainst(payload), "cannot have a hash")
	result.RejectedDraftHash = ""
	require.NoError(t, result.ValidateAgainst(payload))
	result.QualityReportJSON = json.RawMessage(`{"contract_version":"` + SampleQualityContractVersion + `","passed":true}`)
	require.ErrorContains(t, result.ValidateAgainst(payload), "quality report is invalid")
}

func TestSampleGenerationTaskResultMustMatchFrozenInput(t *testing.T) {
	payload := validSampleGenerationTaskPayload()
	payload.InputHash = SampleGenerationInputHash(payload)
	markdown := "# 候选稿"
	result := SampleGenerationTaskResult{ResultVersion: 1, TaskSubtype: TextbookSampleGenerationTaskSubtype, ProjectID: payload.ProjectID, ChapterID: payload.ChapterID, ExpectedChapterVersion: payload.ExpectedChapterVersion, InputHash: payload.InputHash, Markdown: markdown, DocumentJSON: json.RawMessage(`{"format":"inkwords.chapter.v1"}`), ContentHash: contentDigest(markdown), BookContractRevision: payload.BookContract.RevisionID, StyleSheetRevision: payload.StyleSheet.RevisionID, BlueprintRevision: payload.Blueprint.RevisionID, EvidencePackHash: GenerationEvidencePackHash(payload.EvidencePack), PromptHash: "sha256:prompt", ProviderName: SampleFixtureProviderName, ModelName: SampleFixtureModelName, ProviderUsageJSON: json.RawMessage(`{"input_tokens":"unknown"}`), QualityReportJSON: json.RawMessage(`{"contract_version":"` + SampleQualityContractVersion + `","passed":true}`)}
	require.NoError(t, result.ValidateAgainst(payload))

	result.ProviderName = "deepseek"
	require.ErrorContains(t, result.ValidateAgainst(payload), "provenance does not match")
	result.ProviderName = SampleFixtureProviderName
	result.InputHash = "sha256:other"
	require.ErrorContains(t, result.ValidateAgainst(payload), "identity does not match")
}

func TestTaskStageInputHashSeparatesTaskAndStageWithoutChangingFrozenInput(t *testing.T) {
	inputHash := "sha256:input"
	first := TaskStageInputHash("task-1", SampleGenerationStage, inputHash)
	require.NotEmpty(t, first)
	require.Equal(t, first, TaskStageInputHash("task-1", SampleGenerationStage, inputHash))
	require.NotEqual(t, first, TaskStageInputHash("task-2", SampleGenerationStage, inputHash))
	require.NotEqual(t, first, TaskStageInputHash("task-1", "other_stage", inputHash))
	require.Empty(t, TaskStageInputHash("task-1", SampleGenerationStage, "not-a-hash"))
}

func validSampleGenerationTaskPayload() SampleGenerationTaskPayload {
	primary := SourceSnapshot{ID: "snapshot-primary", SourceID: "source-primary", Kind: SourceKindGitRepository, Role: SourceRolePrimary, Locator: "https://github.com/example/project", ResolvedVersion: "0123456789abcdef0123456789abcdef01234567", ContentHash: "sha256:primary", CapturedAt: time.Unix(1, 0).UTC()}
	official := SourceSnapshot{ID: "snapshot-official", SourceID: "source-official", Kind: SourceKindOfficialWeb, Role: SourceRoleOfficial, Locator: "https://example.com/docs", ResolvedVersion: "2026-09-03", ContentHash: "sha256:official", CapturedAt: time.Unix(1, 0).UTC()}
	book := BookContract{RevisionID: "book-r1", ProjectID: "project-1", RevisionNumber: 1, ContentHash: "sha256:book", Promise: "讲清楚路由。", Reader: ReaderModel{Audience: AudienceFoundation, LearningOutcomes: []string{"解释路由"}}, ChapterProfiles: []ChapterProfile{ChapterProfileConcept}, TerminologyVersion: "v1", PublicationProfile: "personal_learning"}
	style := StyleSheet{RevisionID: "style-r1", ProjectID: "project-1", RevisionNumber: 1, ContentHash: "sha256:style", Language: "zh-CN", TerminologyRules: []string{"先白话"}, CodeRules: []string{"标记验证"}, CitationRules: []string{"引用证据"}}
	blueprint := Blueprint{RevisionID: "blueprint-r1", ProjectID: "project-1", RevisionNumber: 1, ContentHash: "sha256:blueprint", BookContractRevision: book.RevisionID, StyleSheetRevision: style.RevisionID, Volumes: []BlueprintVolume{{ID: "volume-1", Title: "开始", Sort: 1, Chapters: []BlueprintChapter{{ID: "chapter-1", Title: "路由", Sort: 1, Profile: ChapterProfileConcept, EvidenceIDs: []string{"chunk-1"}}}}}}
	evidence := EvidenceRef{ID: "evidence-chunk-1", SnapshotID: primary.ID, DocumentID: "document-1", ChunkID: "chunk-1", Locator: EvidenceLocator{Path: "router.go", StartLine: 1, EndLine: 3}, ContentHash: "sha256:chunk", Confidence: EvidenceConfidenceDocumented, SourceRole: SourceRolePrimary}
	return SampleGenerationTaskPayload{TaskVersion: SampleGenerationTaskVersion, TaskSubtype: TextbookSampleGenerationTaskSubtype, PromptSchemaVersion: SamplePromptSchemaVersion, QualityContractVersion: SampleQualityContractVersion, GenerationTarget: SampleGenerationTarget{ProviderName: SampleFixtureProviderName, ModelName: SampleFixtureModelName}, ProjectID: "project-1", ChapterID: "chapter-1", Audience: AudienceFoundation, BookContract: book, StyleSheet: style, Blueprint: blueprint, EvidencePack: GenerationEvidencePack{PrimarySnapshot: primary, OfficialSources: []SourceSnapshot{official}, Evidence: []EvidenceRef{evidence}, Excerpts: map[string]string{evidence.ID: "路由登记源码"}}}
}
