package textbook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// SampleCorrectionFormat identifies an offline edit against an immutable receipt.
const SampleCorrectionFormat = sharedtextbook.SampleCorrectionFormat

// SampleCorrectionInput excludes source, chapter and provider identities. The
// practice section is rendered from PracticeSet instead of edited twice.
type SampleCorrectionInput = sharedtextbook.SampleCorrectionInput

// SampleCorrectionProposal is reviewable local output, never an accepted task
// result. Original usage belongs to the source generation, not this correction.
type SampleCorrectionProposal struct {
	Format              string                `json:"format"`
	Origin              string                `json:"origin"`
	OriginalReceiptHash string                `json:"original_receipt_hash"`
	OriginalContentHash string                `json:"original_content_hash"`
	FrozenRequestHash   string                `json:"frozen_request_hash"`
	CorrectionInputHash string                `json:"correction_input_hash"`
	OriginalPromptHash  string                `json:"original_prompt_hash"`
	OriginalProvider    string                `json:"original_provider"`
	OriginalModel       string                `json:"original_model"`
	OriginalUsage       GenerationUsageRecord `json:"original_usage"`
	Reason              string                `json:"reason"`
	Chapter             SampleChapter         `json:"chapter"`
	Quality             QualityReport         `json:"quality"`
	ProviderCalls       int                   `json:"provider_calls"`
	CandidatePersisted  bool                  `json:"candidate_persisted"`
}

// PrepareSampleCorrection checks identity before editing and reruns every
// current quality gate. It has no provider, persistence or execution port.
func PrepareSampleCorrection(receipt []byte, input SampleCorrectionInput) (SampleCorrectionProposal, error) {
	if len(receipt) == 0 || len(receipt) > 2<<20 || input.Format != SampleCorrectionFormat || strings.TrimPrefix(digest(string(receipt)), "sha256:") != input.OriginalReceiptHash {
		return SampleCorrectionProposal{}, fmt.Errorf("invalid or changed correction receipt")
	}
	var draft RejectedDraft
	if err := decodeCorrectionJSON(receipt, &draft); err != nil {
		return SampleCorrectionProposal{}, err
	}
	if err := draft.Validate(); err != nil {
		return SampleCorrectionProposal{}, err
	}
	if input.OriginalContentHash != draft.Generation.Chapter.ContentHash || utf8.RuneCountInString(strings.TrimSpace(input.Reason)) < 8 || utf8.RuneCountInString(input.Reason) > 2000 || strings.TrimSpace(input.MarkdownBody) == "" {
		return SampleCorrectionProposal{}, fmt.Errorf("invalid correction identity, reason or body")
	}
	if strings.Contains(input.MarkdownBody, "inkwords.practice-set") || strings.Contains(input.MarkdownBody, "## 六维练习与答案") {
		return SampleCorrectionProposal{}, fmt.Errorf("correction body must omit the generated practice section")
	}
	encoded, err := json.Marshal(input)
	if err != nil || len(encoded) > 1<<20 {
		return SampleCorrectionProposal{}, fmt.Errorf("correction input exceeds limit")
	}
	// Detach editable slices so a caller cannot mutate the checked proposal.
	var detached SampleCorrectionInput
	if err := json.Unmarshal(encoded, &detached); err != nil {
		return SampleCorrectionProposal{}, fmt.Errorf("invalid correction document")
	}
	identities := map[string]sharedtextbook.LearningTaskMode{}
	for _, task := range draft.Generation.Chapter.PracticeSet.Tasks {
		identities[task.ID] = task.Mode
	}
	if len(detached.PracticeSet.Tasks) != len(identities) {
		return SampleCorrectionProposal{}, fmt.Errorf("correction must retain practice identities")
	}
	for _, task := range detached.PracticeSet.Tasks {
		if mode, ok := identities[task.ID]; !ok || mode != task.Mode {
			return SampleCorrectionProposal{}, fmt.Errorf("correction changed practice identity")
		}
	}
	chapter := draft.Generation.Chapter
	chapter.Profile = draft.Request.BlueprintChapter.Profile
	chapter.Scenario, chapter.Understanding = detached.Scenario, detached.Understanding
	chapter.LearningArc, chapter.PracticeSet = detached.LearningArc, detached.PracticeSet
	chapter.Markdown = strings.TrimSpace(detached.MarkdownBody) + "\n\n" + sharedtextbook.RenderPracticeSet(chapter.PracticeSet)
	chapter.ContentHash = digest(chapter.Markdown)
	chapter.GenerationMode, chapter.RuntimeVerification = "automated_local_correction", "unverified"
	quality := RunSampleChapterQualityGates(chapter, draft.Request.EvidencePack)
	if failures := requiredBlueprintClaimFailures(chapter.Claims, draft.Request.BlueprintChapter.CriticalClaims); len(failures) > 0 {
		quality.Passed = false
		quality.Failures = append(quality.Failures, failures...)
	}
	frozenRequest, err := json.Marshal(draft.Request)
	if err != nil {
		return SampleCorrectionProposal{}, fmt.Errorf("invalid frozen correction request")
	}
	return SampleCorrectionProposal{
		Format: SampleCorrectionFormat, Origin: chapter.GenerationMode, OriginalReceiptHash: input.OriginalReceiptHash,
		OriginalContentHash: input.OriginalContentHash, FrozenRequestHash: digest(string(frozenRequest)), CorrectionInputHash: digest(string(encoded)),
		OriginalPromptHash: draft.Generation.PromptHash, OriginalProvider: draft.Generation.ProviderName, OriginalModel: draft.Generation.ModelName,
		OriginalUsage: draft.Generation.UsageRecord(), Reason: detached.Reason, Chapter: chapter, Quality: quality,
	}, nil
}

func decodeCorrectionJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid correction JSON")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return fmt.Errorf("correction JSON must contain exactly one document")
	}
	return nil
}

// DecodeSampleCorrection rejects unknown identity/provenance fields and trailing
// documents before a CLI or application can pass an edit into the checker.
func DecodeSampleCorrection(data []byte) (SampleCorrectionInput, error) {
	var input SampleCorrectionInput
	if len(data) == 0 || len(data) > 1<<20 {
		return input, fmt.Errorf("correction input exceeds limit")
	}
	err := decodeCorrectionJSON(data, &input)
	return input, err
}
