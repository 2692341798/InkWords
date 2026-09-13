package textbook

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// SampleCorrectionMigration preserves the original contracts separately from
// the current candidate inputs. All other source inputs must remain identical.
type SampleCorrectionMigration struct {
	SourcePromptSchemaVersion    string    `json:"source_prompt_schema_version"`
	SourceQualityContractVersion string    `json:"source_quality_contract_version"`
	SourceBlueprint              Blueprint `json:"source_blueprint"`
}

func prepareMigratedSampleCorrection(source, current SampleGenerationTaskPayload, taskID string, edit SampleCorrectionInput) (SampleGenerationTaskPayload, error) {
	if current.Correction != nil || edit.Baseline == nil || edit.TargetInputHash != current.InputHash || edit.Baseline.ExpectedChapterVersion != current.ExpectedChapterVersion || edit.Baseline.ParentRevisionID != current.ParentRevisionID {
		return SampleGenerationTaskPayload{}, ErrCorrectionBaselineChanged
	}
	payload := current
	payload.TaskVersion = SampleMigratedCorrectionTaskVersion
	payload.Correction = &SampleCorrectionTask{
		OriginalTaskID: taskID, OriginalInputHash: source.InputHash, Edit: edit,
		SourceBaseline: &SampleCorrectionBaseline{ExpectedChapterVersion: source.ExpectedChapterVersion, ParentRevisionID: source.ParentRevisionID},
		Migration:      &SampleCorrectionMigration{SourcePromptSchemaVersion: source.PromptSchemaVersion, SourceQualityContractVersion: source.QualityContractVersion, SourceBlueprint: source.Blueprint},
	}
	payload.InputHash = SampleGenerationInputHash(payload)
	if err := payload.Validate(); err != nil {
		return SampleGenerationTaskPayload{}, fmt.Errorf("validate correction migration: %w", err)
	}
	// Queueing must freeze nested source, current blueprint and edit slices too.
	encoded, err := json.Marshal(payload)
	if err != nil {
		return SampleGenerationTaskPayload{}, err
	}
	var frozen SampleGenerationTaskPayload
	err = json.Unmarshal(encoded, &frozen)
	return frozen, err
}

func (payload SampleGenerationTaskPayload) currentCorrectionInputHash() string {
	payload.Correction = nil
	payload.TaskVersion = SampleGenerationTaskVersion
	return SampleGenerationInputHash(payload)
}

func (payload SampleGenerationTaskPayload) validateCorrectionMigration() error {
	correction := payload.Correction
	if correction.SourceBaseline == nil || payload.PromptSchemaVersion != SamplePromptSchemaVersion || payload.QualityContractVersion != SampleQualityContractVersion || !isFullSHA256Digest(correction.Edit.TargetInputHash) || correction.Edit.TargetInputHash != payload.currentCorrectionInputHash() {
		return fmt.Errorf("correction migration requires explicit current contracts and input hash")
	}
	source := payload.WithoutCorrection()
	if source.ExpectedChapterVersion == payload.ExpectedChapterVersion && source.ParentRevisionID != payload.ParentRevisionID {
		return fmt.Errorf("correction migration changed parent without advancing chapter version")
	}
	if err := source.ValidateRetainedCorrectionSource(); err != nil {
		return fmt.Errorf("validate migration source: %w", err)
	}
	if !blueprintOnlyAddsChapters(source.Blueprint, payload.Blueprint) {
		return fmt.Errorf("correction migration changed an existing blueprint volume or chapter")
	}
	return nil
}

// A newer blueprint may add teaching material, but cannot reinterpret the
// retained receipt by editing, moving or removing any pre-existing chapter.
func blueprintOnlyAddsChapters(source, current Blueprint) bool {
	if source.RevisionID == current.RevisionID {
		return reflect.DeepEqual(source, current)
	}
	if current.RevisionNumber <= source.RevisionNumber || source.ProjectID != current.ProjectID || source.BookContractRevision != current.BookContractRevision || source.StyleSheetRevision != current.StyleSheetRevision || source.ContentHash == current.ContentHash {
		return false
	}
	volumes := make(map[string]BlueprintVolume, len(current.Volumes))
	for _, volume := range current.Volumes {
		if _, exists := volumes[volume.ID]; exists {
			return false
		}
		volumes[volume.ID] = volume
	}
	seen := make(map[string]bool, len(source.Volumes))
	for _, old := range source.Volumes {
		next, exists := volumes[old.ID]
		if !exists || seen[old.ID] || old.Title != next.Title || old.Sort != next.Sort {
			return false
		}
		seen[old.ID] = true
		chapters := make(map[string]BlueprintChapter, len(next.Chapters))
		for _, chapter := range next.Chapters {
			chapters[chapter.ID] = chapter
		}
		for _, chapter := range old.Chapters {
			if !reflect.DeepEqual(chapter, chapters[chapter.ID]) {
				return false
			}
		}
	}
	return true
}
