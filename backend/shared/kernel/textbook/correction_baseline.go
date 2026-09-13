package textbook

import (
	"errors"
	"fmt"
)

// ErrCorrectionBaselineChanged distinguishes stale caller state from invalid edits.
var ErrCorrectionBaselineChanged = errors.New("correction baseline changed")

// ValidateRetainedCorrectionSource accepts only the supported immutable source
// contracts. Ordinary generation continues to require the current prompt.
func (payload SampleGenerationTaskPayload) ValidateRetainedCorrectionSource() error {
	if payload.Correction != nil {
		return fmt.Errorf("correction source must be an original generation")
	}
	return payload.validate(true)
}

// PrepareRetainedSampleCorrection preserves the original receipt identity and
// freezes a separate CAS target only when the caller explicitly supplies it.
func PrepareRetainedSampleCorrection(source, current SampleGenerationTaskPayload, taskID string, edit SampleCorrectionInput) (SampleGenerationTaskPayload, error) {
	if err := source.ValidateRetainedCorrectionSource(); err != nil {
		return SampleGenerationTaskPayload{}, err
	}
	if err := current.Validate(); err != nil {
		return SampleGenerationTaskPayload{}, err
	}
	if edit.TargetInputHash != "" {
		return prepareMigratedSampleCorrection(source, current, taskID, edit)
	}
	payload := source
	payload.Correction = &SampleCorrectionTask{OriginalTaskID: taskID, OriginalInputHash: source.InputHash, Edit: edit}
	payload.TaskVersion = SampleCorrectionTaskVersion
	if edit.Baseline != nil {
		if edit.Baseline.ExpectedChapterVersion != current.ExpectedChapterVersion || edit.Baseline.ParentRevisionID != current.ParentRevisionID {
			return SampleGenerationTaskPayload{}, ErrCorrectionBaselineChanged
		}
		payload.Correction.SourceBaseline = &SampleCorrectionBaseline{ExpectedChapterVersion: source.ExpectedChapterVersion, ParentRevisionID: source.ParentRevisionID}
		payload.ExpectedChapterVersion = current.ExpectedChapterVersion
		payload.ParentRevisionID = current.ParentRevisionID
		payload.TaskVersion = SampleRebasedCorrectionTaskVersion
	}
	if !payload.MatchesCorrectionBaseline(current) {
		return SampleGenerationTaskPayload{}, ErrCorrectionBaselineChanged
	}
	payload.InputHash = SampleGenerationInputHash(payload)
	return payload, payload.Validate()
}

// MatchesCorrectionBaseline compares approved inputs and the frozen CAS target.
// A migrated correction pins the complete current input, including its blueprint;
// legacy correction versions retain their original prompt-only compatibility.
func (payload SampleGenerationTaskPayload) MatchesCorrectionBaseline(current SampleGenerationTaskPayload) bool {
	if payload.Correction == nil || current.Correction != nil || current.Validate() != nil {
		return false
	}
	if payload.Correction.Migration != nil {
		return payload.Validate() == nil && payload.currentCorrectionInputHash() == current.InputHash
	}
	baseline := payload.WithoutCorrection()
	baseline.ExpectedChapterVersion = payload.ExpectedChapterVersion
	baseline.ParentRevisionID = payload.ParentRevisionID
	current.PromptSchemaVersion = baseline.PromptSchemaVersion
	return SampleGenerationInputHash(current) == SampleGenerationInputHash(baseline)
}
