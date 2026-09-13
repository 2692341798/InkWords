package textbook

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type receiptLoader []byte

func (r receiptLoader) Load(string) ([]byte, error) { return r, nil }

type forbiddenCorrectionGenerator struct{ calls int }

func (g *forbiddenCorrectionGenerator) Generate(context.Context, SampleGenerationRequest) (SampleGeneration, error) {
	g.calls++
	return SampleGeneration{}, nil
}

func TestCorrectionWorkerUsesReceiptInsteadOfProviderAndRejectsWrongFrozenRequest(t *testing.T) {
	for _, mode := range []string{"current", "retained", "migrated"} {
		t.Run(mode, func(t *testing.T) {
			retained := mode != "current"
			payload := ginSampleTaskPayload(t)
			current := payload
			if retained {
				payload.PromptSchemaVersion = "inkwords.textbook.sample.v15"
				if mode == "migrated" {
					payload.PromptSchemaVersion = "inkwords.textbook.sample.v16"
					payload.QualityContractVersion = "inkwords.sample-quality.v9"
					current.Blueprint.RevisionID = uuid.NewString()
					current.Blueprint.RevisionNumber++
					current.Blueprint.ContentHash = "sha256:" + strings.Repeat("c", 64)
				}
				payload.InputHash = sharedtextbook.SampleGenerationInputHash(payload)
				current.ExpectedChapterVersion++
				current.ParentRevisionID = uuid.NewString()
				current.InputHash = sharedtextbook.SampleGenerationInputHash(current)
			}
			request, _, err := ginFixtureRequest(payload)
			require.NoError(t, err)
			chapter, err := BuildGinRequestLifecycleSample(request.EvidencePack)
			require.NoError(t, err)
			chapter.Markdown = strings.ReplaceAll(chapter.Markdown, "**验证状态：未验证。**", "")
			chapter.Claims[0].ID = request.BlueprintChapter.CriticalClaims[0].ID
			chapter.Claims[0].Critical = true
			chapter.Claims[0].EvidenceIDs = request.BlueprintChapter.CriticalClaims[0].EvidenceIDs
			chapter.ContentHash = digest(chapter.Markdown)
			draft := RejectedDraft{Format: "inkwords.rejected-sample.v1", PromptSchema: payload.PromptSchemaVersion, Request: request, Generation: SampleGeneration{Chapter: chapter, ProviderName: request.GenerationTarget.ProviderName, ModelName: request.GenerationTarget.ModelName, PromptHash: "sha256:original", Quality: RunSampleChapterQualityGates(chapter, request.EvidencePack)}}
			raw, err := json.Marshal(draft)
			require.NoError(t, err)
			edit := SampleCorrectionInput{Format: SampleCorrectionFormat, OriginalReceiptHash: strings.TrimPrefix(digest(string(raw)), "sha256:"), OriginalContentHash: chapter.ContentHash, Reason: "修正未验证标记并保留教学事实。", MarkdownBody: strings.TrimSpace(strings.TrimSuffix(chapter.Markdown, sharedtextbook.RenderPracticeSet(chapter.PracticeSet))) + "\n\n**验证状态：未验证。**", Scenario: chapter.Scenario, Understanding: chapter.Understanding, LearningArc: chapter.LearningArc, PracticeSet: chapter.PracticeSet}
			if retained {
				edit.Baseline = &sharedtextbook.SampleCorrectionBaseline{ExpectedChapterVersion: current.ExpectedChapterVersion, ParentRevisionID: current.ParentRevisionID}
			}
			if mode == "migrated" {
				edit.TargetInputHash = current.InputHash
			}
			payload, err = sharedtextbook.PrepareRetainedSampleCorrection(payload, current, uuid.NewString(), edit)
			require.NoError(t, err)
			generator := &forbiddenCorrectionGenerator{}
			runner := NewSampleTaskRunner(generator).WithCorrectionStore(receiptLoader(raw))
			result, err := runner.Run(t.Context(), payload)
			require.NoError(t, err)
			require.Zero(t, generator.calls)
			require.Equal(t, 2, result.ResultVersion)
			require.NoError(t, result.ValidateAgainst(payload))
			require.NotNil(t, result.Correction)
			require.Equal(t, current.Blueprint.RevisionID, result.BlueprintRevision)
			require.Equal(t, edit.TargetInputHash, result.Correction.TargetInputHash)
			require.NoError(t, sharedtextbook.ValidateCurrentSampleQualityReport(result.QualityReportJSON))
			result.ProviderUsageJSON = []byte(`{"known":true,"provider_call_count":1}`)
			require.Error(t, result.ValidateAgainst(payload))
			_, err = NewSampleTaskRunner(generator).Run(t.Context(), payload)
			require.Error(t, err)
			require.Zero(t, generator.calls)
			draft.Request.ClaimCandidates = nil
			draft.Request.StyleSheet.CodeRules = append(draft.Request.StyleSheet.CodeRules, "another rule")
			raw, err = json.Marshal(draft)
			require.NoError(t, err)
			payload.Correction.Edit.OriginalReceiptHash = strings.TrimPrefix(digest(string(raw)), "sha256:")
			payload.InputHash = sharedtextbook.SampleGenerationInputHash(payload)
			_, err = runner.WithCorrectionStore(receiptLoader(raw)).Run(t.Context(), payload)
			require.Error(t, err)
			require.Zero(t, generator.calls)
		})
	}
}
