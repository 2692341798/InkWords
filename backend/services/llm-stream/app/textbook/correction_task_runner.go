package textbook

import (
	"context"
	"encoding/json"
	"fmt"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// RejectedDraftLoader reads only content-addressed private generation receipts.
type RejectedDraftLoader interface{ Load(string) ([]byte, error) }

// WithCorrectionStore enables local edits without changing the provider port.
func (r *SampleTaskRunner) WithCorrectionStore(store RejectedDraftLoader) *SampleTaskRunner {
	r.correctionStore = store
	return r
}

func (r *SampleTaskRunner) runCorrection(ctx context.Context, payload sharedtextbook.SampleGenerationTaskPayload, request SampleGenerationRequest, aliases map[string]string) (sharedtextbook.SampleGenerationTaskResult, error) {
	if err := ctx.Err(); err != nil {
		return sharedtextbook.SampleGenerationTaskResult{}, err
	}
	if r.correctionStore == nil {
		return sharedtextbook.SampleGenerationTaskResult{}, fmt.Errorf("private correction receipt store unavailable")
	}
	receipt, err := r.correctionStore.Load(payload.Correction.Edit.OriginalReceiptHash)
	if err != nil {
		return sharedtextbook.SampleGenerationTaskResult{}, fmt.Errorf("private correction receipt unavailable")
	}
	proposal, err := PrepareSampleCorrection(receipt, payload.Correction.Edit)
	if err != nil {
		return sharedtextbook.SampleGenerationTaskResult{}, err
	}
	frozen, err := json.Marshal(request)
	if err != nil || proposal.FrozenRequestHash != digest(string(frozen)) {
		return sharedtextbook.SampleGenerationTaskResult{}, fmt.Errorf("correction receipt does not match frozen task request")
	}
	if !proposal.Quality.Passed {
		return sharedtextbook.SampleGenerationTaskResult{}, fmt.Errorf("corrected sample failed current quality gates")
	}
	originalUsage, err := json.Marshal(proposal.OriginalUsage)
	if err != nil {
		return sharedtextbook.SampleGenerationTaskResult{}, fmt.Errorf("invalid original correction usage")
	}
	provenance := &sharedtextbook.SampleCorrectionProvenance{
		TargetInputHash: payload.Correction.Edit.TargetInputHash,
		Origin:          proposal.Origin, OriginalTaskID: payload.Correction.OriginalTaskID, OriginalInputHash: payload.Correction.OriginalInputHash,
		OriginalReceiptHash: proposal.OriginalReceiptHash, OriginalContentHash: proposal.OriginalContentHash, CorrectionInputHash: proposal.CorrectionInputHash,
		FrozenRequestHash: proposal.FrozenRequestHash, OriginalPromptHash: proposal.OriginalPromptHash, OriginalUsageJSON: originalUsage, Reason: proposal.Reason,
	}
	generation := SampleGeneration{Chapter: proposal.Chapter, Quality: proposal.Quality, ProviderName: proposal.OriginalProvider, ModelName: proposal.OriginalModel, PromptHash: proposal.CorrectionInputHash}
	if err := ctx.Err(); err != nil {
		return sharedtextbook.SampleGenerationTaskResult{}, err
	}
	return buildSampleTaskResult(payload, generation, aliases, provenance)
}
