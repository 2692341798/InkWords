package textbook

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	sharedgeneration "inkwords-backend/shared/kernel/generation"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type rejectedSampleGenerator struct {
	generation SampleGeneration
}

func (generator rejectedSampleGenerator) Generate(context.Context, SampleGenerationRequest) (SampleGeneration, error) {
	return generator.generation, errors.New("generated textbook candidate failed quality gates: undefined_repeated_acronym: POST")
}

func TestSampleTaskRunnerProducesOnlyEvidenceBoundGinCandidate(t *testing.T) {
	payload := ginSampleTaskPayload(t)
	runner := NewSampleTaskRunner()
	result, err := runner.Run(context.Background(), payload)
	require.NoError(t, err)
	require.NoError(t, result.ValidateAgainst(payload))
	require.Contains(t, string(result.DocumentJSON), "evidence_aliases")
	var document struct {
		VideoRunbook sharedtextbook.VideoRunbookProjection `json:"video_runbook"`
	}
	require.NoError(t, json.Unmarshal(result.DocumentJSON, &document))
	require.NoError(t, document.VideoRunbook.Validate())
	var usage GenerationUsageRecord
	require.NoError(t, json.Unmarshal(result.ProviderUsageJSON, &usage))
	require.False(t, usage.Known)
	require.False(t, usage.LocalCacheHit)
	require.Zero(t, usage.ProviderCallCount)
	require.False(t, usage.EstimatedCostKnown)
	require.Equal(t, "Visual Studio Code", document.VideoRunbook.Recommendation.Primary)
	require.Contains(t, document.VideoRunbook.Steps[0].Input, result.ContentHash)
	require.Contains(t, document.VideoRunbook.Steps[2].Action, "隔离")
	require.True(t, document.VideoRunbook.ManualCapturePending)
	require.Equal(t, sharedtextbook.ArtifactStatusUnverified, document.VideoRunbook.VerificationStatus)
	for _, step := range document.VideoRunbook.Steps {
		require.NotContains(t, step.ShortcutOrMenu, "Shift+F9")
		require.NotContains(t, step.Recovery, "右键 main.go 选择 Run")
	}
	var quality QualityReport
	require.NoError(t, json.Unmarshal(result.QualityReportJSON, &quality))
	require.True(t, quality.Passed)
	require.True(t, quality.ManualReviewRequired)
	require.Equal(t, sampleManualReviewDimensions, quality.ManualReviewDimensions)
}

func TestSampleTaskRunnerRejectsUnknownSnapshotBeforeGenerating(t *testing.T) {
	payload := ginSampleTaskPayload(t)
	payload.EvidencePack.PrimarySnapshot.ResolvedVersion = "0123456789abcdef0123456789abcdef01234567"
	for index := range payload.EvidencePack.PrimarySnapshots {
		payload.EvidencePack.PrimarySnapshots[index].ResolvedVersion = "0123456789abcdef0123456789abcdef01234567"
	}
	payload.InputHash = sharedtextbook.SampleGenerationInputHash(payload)
	_, err := NewSampleTaskRunner().Run(context.Background(), payload)
	require.ErrorContains(t, err, "fixed Gin v1.12.0")
}

func TestGinEvidenceAdapterPreservesEachApprovedAudience(t *testing.T) {
	for _, audience := range []sharedtextbook.AudienceLevel{sharedtextbook.AudienceFoundation, sharedtextbook.AudienceProgramming, sharedtextbook.AudienceStackFamiliar} {
		t.Run(string(audience), func(t *testing.T) {
			payload := ginSampleTaskPayload(t)
			payload.Audience = audience
			payload.BookContract.Reader.Audience = audience
			payload.InputHash = sharedtextbook.SampleGenerationInputHash(payload)
			require.NoError(t, payload.Validate())
			request, aliases, err := ginFixtureRequest(payload)
			require.NoError(t, err)
			require.NoError(t, request.Validate())
			require.Equal(t, audience, request.Audience)
			require.Equal(t, audience, request.BookContract.Reader.Audience)
			require.Len(t, aliases, 6)
		})
	}
}

func TestGinEvidenceAdapterKeepsAdditionalApprovedSourceMechanisms(t *testing.T) {
	payload := ginSampleTaskPayload(t)
	extra := payload.EvidencePack.Evidence[5]
	extra.ID, extra.ChunkID, extra.Locator.Symbol = "evidence-chunk-prefix", "chunk-prefix", "longestCommonPrefix"
	payload.EvidencePack.Evidence = append(payload.EvidencePack.Evidence, extra)
	payload.EvidencePack.Excerpts[extra.ID] = "func longestCommonPrefix(a, b string) int { return 0 }"
	chapter := &payload.Blueprint.Volumes[0].Chapters[0]
	chapter.EvidenceIDs = append(chapter.EvidenceIDs, extra.ChunkID)
	chapter.CriticalClaims[0].EvidenceIDs = append(chapter.CriticalClaims[0].EvidenceIDs, extra.ChunkID)
	payload.InputHash = sharedtextbook.SampleGenerationInputHash(payload)
	require.NoError(t, payload.Validate())
	request, aliases, err := ginFixtureRequest(payload)
	require.NoError(t, err)
	require.NoError(t, request.Validate())
	require.Len(t, request.EvidencePack.Evidence, 7)
	alias := request.BlueprintChapter.EvidenceIDs[6]
	require.Equal(t, extra.ID, aliases[alias])
	require.Equal(t, payload.EvidencePack.Excerpts[extra.ID], request.EvidencePack.Excerpts[alias])
	require.Contains(t, request.BlueprintChapter.CriticalClaims[0].EvidenceIDs, alias)
	require.Equal(t, "longestCommonPrefix", request.EvidencePack.Evidence[6].Locator.Symbol)
}

func TestGinEvidenceAdapterDoesNotReplacePrimaryAnchorsWithSupportingSymbols(t *testing.T) {
	payload := ginSampleTaskPayload(t)
	extra := payload.EvidencePack.Evidence[0]
	extra.ID, extra.ChunkID = "evidence-supporting-get", "supporting-get"
	extra.SnapshotID = payload.EvidencePack.OfficialSources[0].ID
	extra.SourceRole = sharedtextbook.SourceRoleOfficial
	payload.EvidencePack.Evidence = append(payload.EvidencePack.Evidence, extra)
	payload.EvidencePack.Excerpts[extra.ID] = "An official explanation of GET."
	payload.Blueprint.Volumes[0].Chapters[0].EvidenceIDs = append(payload.Blueprint.Volumes[0].Chapters[0].EvidenceIDs, extra.ChunkID)
	payload.InputHash = sharedtextbook.SampleGenerationInputHash(payload)
	request, aliases, err := ginFixtureRequest(payload)
	require.NoError(t, err)
	require.NoError(t, request.Validate())
	require.Equal(t, payload.EvidencePack.Evidence[0].ID, aliases["gin-routergroup-get"])
	require.Equal(t, extra.ID, aliases["source-supporting-get"])
	// A second primary declaration with the same anchor is ambiguous; choosing
	// whichever snapshot happens to occur first would hide a source conflict.
	payload.EvidencePack.Evidence[6].SnapshotID = payload.EvidencePack.PrimarySnapshot.ID
	payload.EvidencePack.Evidence[6].SourceRole = sharedtextbook.SourceRolePrimary
	_, _, err = ginFixtureRequest(payload)
	require.ErrorContains(t, err, "duplicate evidence")
}

func TestSampleTaskRunnerBuildsDistinctOfflineCandidateForEachAudience(t *testing.T) {
	markers := map[sharedtextbook.AudienceLevel]string{
		sharedtextbook.AudienceFoundation:    "不要求你先会 Go 或 Gin",
		sharedtextbook.AudienceProgramming:   "假设你已经会读 Go 函数、map 和单元测试",
		sharedtextbook.AudienceStackFamiliar: "假设你已经能创建 Gin 路由并使用调试器",
	}
	hashes := map[string]bool{}
	for audience, marker := range markers {
		t.Run(string(audience), func(t *testing.T) {
			payload := ginSampleTaskPayload(t)
			payload.Audience = audience
			payload.BookContract.Reader.Audience = audience
			payload.InputHash = sharedtextbook.SampleGenerationInputHash(payload)

			result, err := NewSampleTaskRunner().Run(context.Background(), payload)
			require.NoError(t, err)
			require.Contains(t, result.Markdown, marker)
			require.False(t, hashes[result.ContentHash], "audience-specific task output must not reuse another manuscript")
			hashes[result.ContentHash] = true

			var document struct {
				Sample SampleChapter `json:"sample"`
			}
			require.NoError(t, json.Unmarshal(result.DocumentJSON, &document))
			require.Equal(t, audience, document.Sample.Audience)
		})
	}
}

func TestSampleTaskRunnerReturnsSafeFailureEvidenceWithoutRejectedProse(t *testing.T) {
	payload := ginSampleTaskPayload(t)
	generation := SampleGeneration{
		Chapter:      SampleChapter{Markdown: "rejected manuscript must not be persisted"},
		Quality:      QualityReport{ContractVersion: sharedtextbook.SampleQualityContractVersion, Passed: false, Failures: []string{"undefined_repeated_acronym: POST"}, FailureDiagnostics: DiagnoseUndefinedAcronyms("POST 首次出现。POST 再出现。")},
		ProviderName: payload.GenerationTarget.ProviderName, ModelName: payload.GenerationTarget.ModelName,
		ProviderUsage: sharedgeneration.Usage{Known: true, InputTokens: 21, OutputTokens: 34},
		ProviderCalls: 1, ProviderLatency: 25, PromptHash: "sha256:prompt",
	}
	runner := NewSampleTaskRunner(rejectedSampleGenerator{generation: generation})

	_, err := runner.Run(context.Background(), payload)
	var rejected *SampleGenerationRejectedError
	require.ErrorAs(t, err, &rejected)
	failure := rejected.FailureResult()
	require.NoError(t, failure.ValidateAgainst(payload))
	encoded, marshalErr := json.Marshal(failure)
	require.NoError(t, marshalErr)
	require.NotContains(t, string(encoded), "rejected manuscript")
	require.Contains(t, string(encoded), `"input_tokens":21`)
	require.Contains(t, string(encoded), `"candidate_persisted":false`)
	require.Contains(t, string(encoded), `"failure_diagnostics"`)
	require.Contains(t, string(encoded), "POST 首次出现")
}

func ginSampleTaskPayload(t *testing.T) sharedtextbook.SampleGenerationTaskPayload {
	t.Helper()
	pack := ginEvidencePack(t)
	excerpts := make(map[string]string, len(pack.Evidence))
	for index := range pack.Evidence {
		originalID := pack.Evidence[index].ID
		pack.Evidence[index].ChunkID = "chunk-" + originalID
		pack.Evidence[index].ID = "evidence-" + pack.Evidence[index].ChunkID
		excerpts[pack.Evidence[index].ID] = pack.Excerpts[originalID]
	}
	pack.Excerpts = excerpts
	book := sharedtextbook.BookContract{RevisionID: "book-contract-1", ProjectID: "project-gin", RevisionNumber: 1, ContentHash: "sha256:contract", Promise: "从真实问题讲透 Gin。", Reader: sharedtextbook.ReaderModel{Audience: sharedtextbook.AudienceFoundation, LearningOutcomes: []string{"解释路由登记"}}, ChapterProfiles: []sharedtextbook.ChapterProfile{sharedtextbook.ChapterProfileConcept}, TerminologyVersion: "terms-1", PublicationProfile: "personal_learning"}
	style := sharedtextbook.StyleSheet{RevisionID: "style-sheet-1", ProjectID: "project-gin", RevisionNumber: 1, ContentHash: "sha256:style", Language: "zh-CN", TerminologyRules: []string{"先白话后术语"}, CodeRules: []string{"标记验证状态"}, CitationRules: []string{"关键事实引用证据"}}
	evidenceIDs := make([]string, 0, len(pack.Evidence))
	for _, evidence := range pack.Evidence {
		evidenceIDs = append(evidenceIDs, evidence.ChunkID)
	}
	payload := sharedtextbook.SampleGenerationTaskPayload{TaskVersion: sharedtextbook.SampleGenerationTaskVersion, TaskSubtype: sharedtextbook.TextbookSampleGenerationTaskSubtype, PromptSchemaVersion: sharedtextbook.SamplePromptSchemaVersion, QualityContractVersion: sharedtextbook.SampleQualityContractVersion, GenerationTarget: sharedtextbook.SampleGenerationTarget{ProviderName: sharedtextbook.SampleFixtureProviderName, ModelName: sharedtextbook.SampleFixtureModelName}, ProjectID: "project-gin", ChapterID: ginRequestLifecycleChapterID, Audience: sharedtextbook.AudienceFoundation, BookContract: book, StyleSheet: style, Blueprint: sharedtextbook.Blueprint{RevisionID: "blueprint-1", ProjectID: "project-gin", RevisionNumber: 1, ContentHash: "sha256:blueprint", BookContractRevision: book.RevisionID, StyleSheetRevision: style.RevisionID, Volumes: []sharedtextbook.BlueprintVolume{{ID: "volume-1", Title: "Gin", Sort: 1, Chapters: []sharedtextbook.BlueprintChapter{{ID: ginRequestLifecycleChapterID, Title: "路由登记", Sort: 1, Profile: sharedtextbook.ChapterProfileConcept, EvidenceIDs: evidenceIDs, CriticalClaims: []sharedtextbook.ClaimRequirement{{ID: "route-registration", Label: "解释 GET 如何进入路由登记", EvidenceIDs: evidenceIDs}}}}}}}, EvidencePack: pack}
	payload.InputHash = sharedtextbook.SampleGenerationInputHash(payload)
	return payload
}
