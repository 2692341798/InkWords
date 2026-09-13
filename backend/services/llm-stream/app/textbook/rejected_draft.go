package textbook

import (
	"context"
	"encoding/json"
	"fmt"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// RejectedDraft is private diagnostic content, never a candidate or cache entry.
type RejectedDraft struct {
	Format       string                  `json:"format"`
	PromptSchema string                  `json:"prompt_schema"`
	Request      SampleGenerationRequest `json:"request"`
	Generation   SampleGeneration        `json:"generation"`
}

// Validate rejects accepted content and broken source/provenance bindings.
func (draft RejectedDraft) Validate() error {
	if draft.Format != "inkwords.rejected-sample.v1" || draft.PromptSchema == "" || draft.Request.Validate() != nil {
		return fmt.Errorf("invalid rejected draft input")
	}
	g := draft.Generation
	if g.Quality.Passed || len(g.Quality.Failures) == 0 || g.Candidate.ID != "" || g.Chapter.ContentHash != digest(g.Chapter.Markdown) || g.ProviderName != draft.Request.GenerationTarget.ProviderName || g.ModelName != draft.Request.GenerationTarget.ModelName || g.PromptHash == "" {
		return fmt.Errorf("invalid rejected draft provenance")
	}
	if g.Chapter.ID != draft.Request.BlueprintChapter.ID || g.Chapter.Audience != draft.Request.Audience {
		return fmt.Errorf("rejected draft identity mismatch")
	}
	return nil
}

// RejectedDraftStore owns immutable private diagnostic storage.
type RejectedDraftStore interface {
	Save(context.Context, []byte) (string, error)
}

type retainingSampleGenerator struct {
	generator SampleGenerator
	store     RejectedDraftStore
}

// NewRetainingSampleGenerator preserves a rejected result separately from the
// accepted cache. A storage failure does not hide the original quality failure.
func NewRetainingSampleGenerator(generator SampleGenerator, store RejectedDraftStore) SampleGenerator {
	return &retainingSampleGenerator{generator: generator, store: store}
}

func (g *retainingSampleGenerator) Generate(ctx context.Context, request SampleGenerationRequest) (SampleGeneration, error) {
	result, err := g.generator.Generate(ctx, request)
	if err == nil || len(result.Quality.Failures) == 0 || g.store == nil {
		return result, err
	}
	draft := RejectedDraft{Format: "inkwords.rejected-sample.v1", PromptSchema: sharedtextbook.SamplePromptSchemaVersion, Request: request, Generation: result}
	data, saveErr := json.Marshal(draft)
	if validationErr := draft.Validate(); validationErr != nil {
		saveErr = validationErr
	}
	ref := ""
	if saveErr == nil {
		ref, saveErr = g.store.Save(ctx, data)
	}
	if saveErr != nil {
		result.RejectedDraftStorage = "unavailable"
	} else {
		result.RejectedDraftStorage = "saved"
		result.RejectedDraftHash = ref
	}
	return result, err
}

// DraftRecheck reports an offline check without creating an approvable revision.
type DraftRecheck struct {
	Origin              string        `json:"origin"`
	OriginalContentHash string        `json:"original_content_hash"`
	CheckedContentHash  string        `json:"checked_content_hash"`
	ProviderCalls       int           `json:"provider_calls"`
	Quality             QualityReport `json:"quality"`
	CandidatePersisted  bool          `json:"candidate_persisted"`
}

// RecheckRejectedDraft allows only a replacement Markdown body. Evidence,
// practice contracts and identity remain frozen; every quality gate runs again.
func RecheckRejectedDraft(draft RejectedDraft, markdown string) (DraftRecheck, error) {
	if err := draft.Validate(); err != nil {
		return DraftRecheck{}, err
	}
	if len(markdown) == 0 || len(markdown) > 1<<20 {
		return DraftRecheck{}, fmt.Errorf("replacement markdown size invalid")
	}
	// Deep-copy slice-bearing fields before rechecking to preserve the receipt.
	encoded, _ := json.Marshal(draft.Generation.Chapter)
	var chapter SampleChapter
	if err := json.Unmarshal(encoded, &chapter); err != nil {
		return DraftRecheck{}, fmt.Errorf("decode rejected chapter")
	}
	chapter.Markdown, chapter.ContentHash = markdown, digest(markdown)
	chapter.Profile = draft.Request.BlueprintChapter.Profile
	quality := RunSampleChapterQualityGates(chapter, draft.Request.EvidencePack)
	if failures := requiredBlueprintClaimFailures(chapter.Claims, draft.Request.BlueprintChapter.CriticalClaims); len(failures) > 0 {
		quality.Passed = false
		quality.Failures = append(quality.Failures, failures...)
	}
	return DraftRecheck{Origin: "automated_offline_recheck", OriginalContentHash: draft.Generation.Chapter.ContentHash, CheckedContentHash: chapter.ContentHash, Quality: quality}, nil
}
