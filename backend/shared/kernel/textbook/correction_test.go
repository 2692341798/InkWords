package textbook

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCorrectionPayloadSeparatesExecutionFromOriginalProviderInput(t *testing.T) {
	base := validSampleGenerationTaskPayload()
	base.InputHash = SampleGenerationInputHash(base)
	before := base.InputHash
	payload := base
	payload.Correction = &SampleCorrectionTask{OriginalTaskID: uuid.NewString(), OriginalInputHash: before, Edit: SampleCorrectionInput{Format: SampleCorrectionFormat, OriginalReceiptHash: strings.Repeat("a", 64), OriginalContentHash: "sha256:" + strings.Repeat("b", 64), MarkdownBody: "corrected", Reason: "修正正文和结构化练习，保留原始来源。"}}
	payload.InputHash = SampleGenerationInputHash(payload)
	require.Error(t, payload.Validate(), "old workers must not treat a correction as a provider task")
	payload.TaskVersion = SampleCorrectionTaskVersion
	payload.InputHash = SampleGenerationInputHash(payload)
	require.NoError(t, payload.Validate())
	require.NotEqual(t, before, payload.InputHash)
	require.Equal(t, before, payload.WithoutCorrection().InputHash)
	payload.Correction.Edit.MarkdownBody += " changed"
	require.Error(t, payload.Validate())
	payload.InputHash = SampleGenerationInputHash(payload)
	require.NoError(t, payload.Validate())
	payload.ExpectedChapterVersion++
	payload.InputHash = SampleGenerationInputHash(payload)
	require.Error(t, payload.Validate(), "a later chapter state requires an explicit new correction baseline")
}

func TestRetainedCorrectionRequiresExplicitCurrentBaseline(t *testing.T) {
	source := validSampleGenerationTaskPayload()
	source.PromptSchemaVersion = "inkwords.textbook.sample.v15"
	source.InputHash = SampleGenerationInputHash(source)
	require.Error(t, source.Validate(), "retained input must never start a provider call")
	require.NoError(t, source.ValidateRetainedCorrectionSource())
	current := source
	current.PromptSchemaVersion = SamplePromptSchemaVersion
	current.ExpectedChapterVersion++
	current.ParentRevisionID = uuid.NewString()
	current.InputHash = SampleGenerationInputHash(current)
	edit := SampleCorrectionInput{Format: SampleCorrectionFormat, OriginalReceiptHash: strings.Repeat("a", 64), OriginalContentHash: "sha256:" + strings.Repeat("b", 64), MarkdownBody: "corrected", Reason: "用户授权修订，绑定当前版本并保留原稿。"}
	_, err := PrepareRetainedSampleCorrection(source, current, uuid.NewString(), edit)
	require.Error(t, err, "no implicit rebase")
	edit.Baseline = &SampleCorrectionBaseline{ExpectedChapterVersion: current.ExpectedChapterVersion, ParentRevisionID: current.ParentRevisionID}
	payload, err := PrepareRetainedSampleCorrection(source, current, uuid.NewString(), edit)
	require.NoError(t, err)
	require.NoError(t, payload.Validate())
	require.Equal(t, SampleRebasedCorrectionTaskVersion, payload.TaskVersion)
	require.Equal(t, source.InputHash, payload.WithoutCorrection().InputHash)
	require.Equal(t, current.ExpectedChapterVersion, payload.ExpectedChapterVersion)
	require.True(t, payload.MatchesCorrectionBaseline(current))
	changed := current
	changed.GenerationTarget.ModelName = "another-model"
	changed.InputHash = SampleGenerationInputHash(changed)
	require.False(t, payload.MatchesCorrectionBaseline(changed), "model changes must fail")
	changed = current
	changed.EvidencePack.Excerpts = map[string]string{"changed": "changed"}
	changed.InputHash = SampleGenerationInputHash(changed)
	require.False(t, payload.MatchesCorrectionBaseline(changed), "source changes must fail")
	current.ExpectedChapterVersion++
	current.InputHash = SampleGenerationInputHash(current)
	require.False(t, payload.MatchesCorrectionBaseline(current))
	_, err = PrepareRetainedSampleCorrection(source, current, uuid.NewString(), edit)
	require.Error(t, err, "stale explicit baseline")
	payload.ExpectedChapterVersion++
	payload.InputHash = SampleGenerationInputHash(payload)
	require.Error(t, payload.Validate(), "tampered CAS must fail even after rehash")
	for _, version := range []string{"inkwords.textbook.sample.v14", "inkwords.textbook.sample.v99"} {
		source.PromptSchemaVersion = version
		source.InputHash = SampleGenerationInputHash(source)
		require.Error(t, source.ValidateRetainedCorrectionSource())
	}
	source.PromptSchemaVersion = "inkwords.textbook.sample.v15"
	source.QualityContractVersion = "old-quality"
	source.InputHash = SampleGenerationInputHash(source)
	require.Error(t, source.ValidateRetainedCorrectionSource())
}
