package textbook

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const SampleCorrectionFormat = "inkwords.sample-correction.v1"
const SampleCorrectionTaskVersion = 4
const SampleRebasedCorrectionTaskVersion = 5
const SampleMigratedCorrectionTaskVersion = 6

// SampleCorrectionBaseline is the caller's explicit compare-and-swap target.
type SampleCorrectionBaseline struct {
	ExpectedChapterVersion int    `json:"expected_chapter_version"`
	ParentRevisionID       string `json:"parent_revision_id"`
}

// SampleCorrectionInput edits teaching content while retaining source identity.
type SampleCorrectionInput struct {
	Baseline            *SampleCorrectionBaseline `json:"baseline,omitempty"`
	TargetInputHash     string                    `json:"target_input_hash,omitempty"`
	Format              string                    `json:"format"`
	OriginalReceiptHash string                    `json:"original_receipt_hash"`
	OriginalContentHash string                    `json:"original_content_hash"`
	Reason              string                    `json:"reason"`
	MarkdownBody        string                    `json:"markdown_body"`
	Scenario            ScenarioFrame             `json:"scenario"`
	Understanding       UnderstandingChain        `json:"understanding"`
	LearningArc         LearningArc               `json:"learning_arc"`
	PracticeSet         PracticeSet               `json:"practice_set"`
}

// SampleCorrectionTask binds an edit to the failed source task selected by core.
type SampleCorrectionTask struct {
	SourceBaseline    *SampleCorrectionBaseline  `json:"source_baseline,omitempty"`
	Migration         *SampleCorrectionMigration `json:"migration,omitempty"`
	OriginalTaskID    string                     `json:"original_task_id"`
	OriginalInputHash string                     `json:"original_input_hash"`
	Edit              SampleCorrectionInput      `json:"edit"`
}

// SampleCorrectionProvenance distinguishes an offline correction from a new
// provider call while preserving the original generation's usage separately.
type SampleCorrectionProvenance struct {
	Origin              string          `json:"origin"`
	TargetInputHash     string          `json:"target_input_hash,omitempty"`
	OriginalTaskID      string          `json:"original_task_id"`
	OriginalInputHash   string          `json:"original_input_hash"`
	OriginalReceiptHash string          `json:"original_receipt_hash"`
	OriginalContentHash string          `json:"original_content_hash"`
	CorrectionInputHash string          `json:"correction_input_hash"`
	FrozenRequestHash   string          `json:"frozen_request_hash"`
	OriginalPromptHash  string          `json:"original_prompt_hash"`
	OriginalUsageJSON   json.RawMessage `json:"original_usage_json"`
	Reason              string          `json:"reason"`
}

func (correction SampleCorrectionTask) ValidateAgainst(payload SampleGenerationTaskPayload) error {
	if _, err := uuid.Parse(correction.OriginalTaskID); err != nil {
		return fmt.Errorf("invalid correction source task")
	}
	if correction.SourceBaseline != nil {
		baseline := correction.Edit.Baseline
		if baseline == nil || baseline.ExpectedChapterVersion != payload.ExpectedChapterVersion || baseline.ParentRevisionID != payload.ParentRevisionID || correction.SourceBaseline.ExpectedChapterVersion < 0 || correction.SourceBaseline.ExpectedChapterVersion > payload.ExpectedChapterVersion {
			return fmt.Errorf("correction baseline does not match explicit edit")
		}
	} else if correction.Edit.Baseline != nil {
		return fmt.Errorf("explicit correction baseline lacks source identity")
	}
	edit, err := json.Marshal(correction.Edit)
	if err != nil || len(edit) > 1<<20 || correction.Edit.Format != SampleCorrectionFormat || !isFullSHA256Digest("sha256:"+correction.Edit.OriginalReceiptHash) || !isFullSHA256Digest(correction.Edit.OriginalContentHash) || strings.TrimSpace(correction.Edit.MarkdownBody) == "" || len([]rune(strings.TrimSpace(correction.Edit.Reason))) < 8 || len([]rune(correction.Edit.Reason)) > 2000 {
		return fmt.Errorf("invalid correction edit")
	}
	base := payload.WithoutCorrection()
	if correction.OriginalInputHash != base.InputHash {
		return fmt.Errorf("correction changed frozen generation input")
	}
	if correction.Migration != nil {
		return payload.validateCorrectionMigration()
	}
	if correction.Edit.TargetInputHash != "" {
		return fmt.Errorf("explicit migration target lacks source contract identity")
	}
	return nil
}

// WithoutCorrection recovers the original generation identity for comparison.
func (payload SampleGenerationTaskPayload) WithoutCorrection() SampleGenerationTaskPayload {
	if payload.Correction != nil && payload.Correction.Migration != nil {
		migration := payload.Correction.Migration
		payload.PromptSchemaVersion = migration.SourcePromptSchemaVersion
		payload.QualityContractVersion = migration.SourceQualityContractVersion
		payload.Blueprint = migration.SourceBlueprint
	}
	if payload.Correction != nil && payload.Correction.SourceBaseline != nil {
		payload.ExpectedChapterVersion = payload.Correction.SourceBaseline.ExpectedChapterVersion
		payload.ParentRevisionID = payload.Correction.SourceBaseline.ParentRevisionID
	}
	payload.Correction = nil
	payload.TaskVersion = SampleGenerationTaskVersion
	payload.InputHash = SampleGenerationInputHash(payload)
	return payload
}

func (provenance SampleCorrectionProvenance) ValidateAgainst(correction SampleCorrectionTask) error {
	if provenance.TargetInputHash != correction.Edit.TargetInputHash {
		return fmt.Errorf("correction provenance changed migration target")
	}
	if provenance.Origin != "automated_local_correction" || provenance.OriginalTaskID != correction.OriginalTaskID || provenance.OriginalInputHash != correction.OriginalInputHash || provenance.OriginalReceiptHash != correction.Edit.OriginalReceiptHash || provenance.OriginalContentHash != correction.Edit.OriginalContentHash || provenance.Reason != correction.Edit.Reason || provenance.CorrectionInputHash != SampleCorrectionInputHash(correction.Edit) || !isFullSHA256Digest(provenance.FrozenRequestHash) || !isSHA256Digest(provenance.OriginalPromptHash) || !json.Valid(provenance.OriginalUsageJSON) {
		return fmt.Errorf("correction provenance does not match frozen edit")
	}
	return nil
}

func SampleCorrectionInputHash(input SampleCorrectionInput) string {
	data, _ := json.Marshal(input)
	return contentDigest(string(data))
}
