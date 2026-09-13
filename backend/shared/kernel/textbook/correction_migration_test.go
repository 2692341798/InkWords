package textbook

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func migrationInputs(t *testing.T) (SampleGenerationTaskPayload, SampleGenerationTaskPayload, SampleCorrectionInput) {
	t.Helper()
	source := validSampleGenerationTaskPayload()
	source.PromptSchemaVersion = "inkwords.textbook.sample.v16"
	source.QualityContractVersion = "inkwords.sample-quality.v9"
	source.InputHash = SampleGenerationInputHash(source)
	var current SampleGenerationTaskPayload
	raw, err := json.Marshal(source)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &current))
	current.PromptSchemaVersion = SamplePromptSchemaVersion
	current.QualityContractVersion = SampleQualityContractVersion
	current.ExpectedChapterVersion++
	current.ParentRevisionID = uuid.NewString()
	current.Blueprint.RevisionID = uuid.NewString()
	current.Blueprint.RevisionNumber++
	current.Blueprint.ContentHash = "sha256:" + strings.Repeat("c", 64)
	chapter := current.Blueprint.Volumes[0].Chapters[0]
	chapter.ID = uuid.NewString()
	chapter.Title = "新增实操章"
	chapter.Sort++
	current.Blueprint.Volumes[0].Chapters = append(current.Blueprint.Volumes[0].Chapters, chapter)
	current.InputHash = SampleGenerationInputHash(current)
	edit := SampleCorrectionInput{Format: SampleCorrectionFormat, OriginalReceiptHash: strings.Repeat("a", 64), OriginalContentHash: "sha256:" + strings.Repeat("b", 64), MarkdownBody: "corrected", Reason: "显式迁移旧回执，保留原任务合同并按当前门禁检查。", Baseline: &SampleCorrectionBaseline{ExpectedChapterVersion: current.ExpectedChapterVersion, ParentRevisionID: current.ParentRevisionID}}
	return source, current, edit
}

func TestCorrectionMigrationRequiresExplicitTargetAndPreservesBothIdentities(t *testing.T) {
	source, current, edit := migrationInputs(t)
	require.NoError(t, source.ValidateRetainedCorrectionSource())
	require.Error(t, source.Validate(), "old source cannot be executed")
	_, err := PrepareRetainedSampleCorrection(source, current, uuid.NewString(), edit)
	require.Error(t, err, "no implicit contract migration")
	edit.TargetInputHash = current.InputHash
	payload, err := PrepareRetainedSampleCorrection(source, current, uuid.NewString(), edit)
	require.NoError(t, err)
	require.Equal(t, SampleMigratedCorrectionTaskVersion, payload.TaskVersion)
	require.NoError(t, payload.Validate())
	require.Equal(t, source, payload.WithoutCorrection())
	require.Equal(t, current.Blueprint, payload.Blueprint)
	require.Equal(t, SampleQualityContractVersion, payload.QualityContractVersion)
	require.True(t, payload.MatchesCorrectionBaseline(current))
	current.Blueprint.Volumes[0].Chapters[1].Title = "changed after enqueue"
	current.InputHash = SampleGenerationInputHash(current)
	require.False(t, payload.MatchesCorrectionBaseline(current))
	require.NoError(t, payload.Validate(), "queued snapshots must not alias caller slices")
	source.Blueprint.Volumes[0].Chapters[0].Title = "changed source in memory"
	require.NoError(t, payload.Validate())
}

func TestCorrectionMigrationRejectsChangedScopeAndStaleHashes(t *testing.T) {
	for name, mutate := range map[string]func(*SampleGenerationTaskPayload, *SampleCorrectionInput){
		"missing baseline": func(_ *SampleGenerationTaskPayload, e *SampleCorrectionInput) { e.Baseline = nil },
		"stale target": func(_ *SampleGenerationTaskPayload, e *SampleCorrectionInput) {
			e.TargetInputHash = "sha256:" + strings.Repeat("f", 64)
		},
		"target chapter": func(p *SampleGenerationTaskPayload, _ *SampleCorrectionInput) {
			p.Blueprint.Volumes[0].Chapters[0].Title += " changed"
		},
		"volume title": func(p *SampleGenerationTaskPayload, _ *SampleCorrectionInput) {
			p.Blueprint.Volumes[0].Title += " changed"
		},
		"model": func(p *SampleGenerationTaskPayload, _ *SampleCorrectionInput) {
			p.GenerationTarget.ModelName = "another-model"
		},
		"style": func(p *SampleGenerationTaskPayload, _ *SampleCorrectionInput) {
			p.StyleSheet.CodeRules = append(p.StyleSheet.CodeRules, "new rule")
		},
	} {
		t.Run(name, func(t *testing.T) {
			source, current, edit := migrationInputs(t)
			edit.TargetInputHash = current.InputHash
			mutate(&current, &edit)
			current.InputHash = SampleGenerationInputHash(current)
			if name != "stale target" {
				edit.TargetInputHash = current.InputHash
			}
			_, err := PrepareRetainedSampleCorrection(source, current, uuid.NewString(), edit)
			require.Error(t, err)
		})
	}
}

func TestRetainedFailureKeepsItsOwnQualityContract(t *testing.T) {
	source, _, _ := migrationInputs(t)
	failure := SampleGenerationTaskFailureResult{ResultVersion: 1, TaskSubtype: source.TaskSubtype, FinalStatus: "failed", ProjectID: source.ProjectID, ChapterID: source.ChapterID, InputHash: source.InputHash, ProviderName: source.GenerationTarget.ProviderName, ModelName: source.GenerationTarget.ModelName, ProviderUsageJSON: []byte(`{}`), PromptHash: "sha256:original", QualityReportJSON: []byte(`{"contract_version":"inkwords.sample-quality.v9","passed":false,"failures":["unverified"]}`)}
	require.NoError(t, failure.ValidateRetainedCorrectionSource(source))
	require.Error(t, failure.ValidateAgainst(source))
	failure.QualityReportJSON = []byte(`{"contract_version":"inkwords.sample-quality.v10","passed":false,"failures":["unverified"]}`)
	require.Error(t, failure.ValidateRetainedCorrectionSource(source), "old report cannot be relabelled")
	failure.QualityReportJSON = []byte(`{"contract_version":"inkwords.sample-quality.v9","passed":true,"failures":["unverified"]}`)
	require.Error(t, failure.ValidateRetainedCorrectionSource(source), "a passed source cannot supply a rejected receipt")
	for _, pair := range [][2]string{{"inkwords.textbook.sample.v16", SampleQualityContractVersion}, {SamplePromptSchemaVersion, "inkwords.sample-quality.v9"}, {"inkwords.textbook.sample.v14", "inkwords.sample-quality.v9"}} {
		source.PromptSchemaVersion, source.QualityContractVersion = pair[0], pair[1]
		source.InputHash = SampleGenerationInputHash(source)
		require.Error(t, source.ValidateRetainedCorrectionSource())
	}
}

func TestMigratedCorrectionRejectsRehashedIdentityTampering(t *testing.T) {
	for _, name := range []string{"original hash", "current hash", "source contract", "current contract", "missing source baseline", "task version", "existing chapter", "moved chapter", "other existing chapter"} {
		t.Run(name, func(t *testing.T) {
			source, current, edit := migrationInputs(t)
			edit.TargetInputHash = current.InputHash
			payload, err := PrepareRetainedSampleCorrection(source, current, uuid.NewString(), edit)
			require.NoError(t, err)
			switch name {
			case "original hash":
				payload.Correction.OriginalInputHash = "sha256:" + strings.Repeat("e", 64)
			case "current hash":
				payload.Correction.Edit.TargetInputHash = "sha256:" + strings.Repeat("e", 64)
			case "source contract":
				payload.Correction.Migration.SourcePromptSchemaVersion = "inkwords.textbook.sample.v99"
			case "current contract":
				payload.PromptSchemaVersion = "inkwords.textbook.sample.v15"
			case "missing source baseline":
				payload.Correction.SourceBaseline = nil
			case "task version":
				payload.TaskVersion = SampleGenerationTaskVersion
			case "existing chapter":
				payload.Blueprint.Volumes[0].Chapters[0].Title += " changed"
			case "moved chapter":
				volume := payload.Blueprint.Volumes[0]
				volume.ID = "moved"
				volume.Chapters = volume.Chapters[:1]
				payload.Blueprint.Volumes[0].Chapters = payload.Blueprint.Volumes[0].Chapters[1:]
				payload.Blueprint.Volumes = append(payload.Blueprint.Volumes, volume)
			case "other existing chapter":
				other := payload.Blueprint.Volumes[0].Chapters[1]
				payload.Correction.Migration.SourceBlueprint.Volumes[0].Chapters = append(payload.Correction.Migration.SourceBlueprint.Volumes[0].Chapters, other)
				payload.Blueprint.Volumes[0].Chapters[1].Title += " changed"
			}
			// Recompute both identities for scope changes: topology checks must
			// still reject moved/edited old chapters, independent of checksums.
			if name == "existing chapter" || name == "moved chapter" || name == "other existing chapter" {
				payload.Correction.Edit.TargetInputHash = payload.currentCorrectionInputHash()
				payload.Correction.OriginalInputHash = payload.WithoutCorrection().InputHash
			}
			payload.InputHash = SampleGenerationInputHash(payload)
			require.Error(t, payload.Validate())
		})
	}
}
