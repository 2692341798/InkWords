package textbook

import (
	"context"
	"fmt"
	"strings"
	"time"

	sharedgeneration "inkwords-backend/shared/kernel/generation"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// SampleGenerator is the provider-independent boundary used by the task layer.
type SampleGenerator interface {
	Generate(context.Context, SampleGenerationRequest) (SampleGeneration, error)
}

// SampleGeneration is always a candidate and carries its deterministic gate result.
type SampleGeneration struct {
	Chapter              SampleChapter                  `json:"chapter"`
	Candidate            sharedtextbook.ChapterRevision `json:"candidate"`
	Quality              QualityReport                  `json:"quality"`
	ProviderName         string                         `json:"provider_name"`
	ModelName            string                         `json:"model_name"`
	ProviderUsage        sharedgeneration.Usage         `json:"provider_usage"`
	ProviderCalls        int                            `json:"provider_calls"`
	ProviderLatency      time.Duration                  `json:"provider_latency"`
	PromptHash           string                         `json:"prompt_hash"`
	CacheHit             bool                           `json:"cache_hit"`
	RejectedDraftHash    string                         `json:"rejected_draft_hash,omitempty"`
	RejectedDraftStorage string                         `json:"rejected_draft_storage,omitempty"`
}

// GenerationUsageRecord is the persisted, provider-neutral telemetry shape.
// Cost stays unknown until a reviewed pricing policy can calculate it from the
// provider/model/region effective at the time of generation.
type GenerationUsageRecord struct {
	sharedgeneration.Usage
	LocalCacheHit      bool  `json:"local_cache_hit"`
	ProviderCallCount  int   `json:"provider_call_count"`
	ProviderLatencyMS  int64 `json:"provider_latency_ms"`
	EstimatedCostKnown bool  `json:"estimated_cost_known"`
}

func (generation SampleGeneration) UsageRecord() GenerationUsageRecord {
	latencyMS := generation.ProviderLatency.Milliseconds()
	if latencyMS < 0 {
		latencyMS = 0
	}
	return GenerationUsageRecord{Usage: generation.ProviderUsage, LocalCacheHit: generation.CacheHit, ProviderCallCount: generation.ProviderCalls, ProviderLatencyMS: latencyMS, EstimatedCostKnown: false}
}

// FakeSampleGenerator provides the CI-safe baseline and must stay behaviorally compatible with real adapters.
type FakeSampleGenerator struct{}

// Generate constructs the fixed Gin sample without contacting a model or executing target code.
func (FakeSampleGenerator) Generate(ctx context.Context, request SampleGenerationRequest) (SampleGeneration, error) {
	if err := ctx.Err(); err != nil {
		return SampleGeneration{}, err
	}
	if err := request.Validate(); err != nil {
		return SampleGeneration{}, err
	}
	if request.GenerationTarget.ProviderName != sharedtextbook.SampleFixtureProviderName || request.GenerationTarget.ModelName != sharedtextbook.SampleFixtureModelName {
		return SampleGeneration{}, fmt.Errorf("sample generation target does not match fixture provider")
	}
	if request.BlueprintChapter.Profile != sharedtextbook.ChapterProfileConcept {
		return SampleGeneration{}, fmt.Errorf("fixed fixture supports only the concept chapter profile")
	}
	chapter, err := BuildGinRequestLifecycleSampleForAudience(request.EvidencePack, request.Audience)
	if err != nil {
		return SampleGeneration{}, err
	}
	quality := RunSampleChapterQualityGates(chapter, request.EvidencePack)
	if !quality.Passed {
		return SampleGeneration{}, fmt.Errorf("fixed Gin sample failed quality gates: %s", strings.Join(quality.Failures, "; "))
	}
	candidate, err := CandidateRevision(chapter, request.EvidencePack, request.BookContract.RevisionID, request.StyleSheet.RevisionID, sharedtextbook.GenerationEvidencePackHash(request.EvidencePack))
	if err != nil {
		return SampleGeneration{}, err
	}
	return SampleGeneration{
		Chapter:       chapter,
		Candidate:     candidate,
		Quality:       quality,
		ProviderName:  sharedtextbook.SampleFixtureProviderName,
		ModelName:     sharedtextbook.SampleFixtureModelName,
		ProviderUsage: sharedgeneration.Usage{Known: false},
		PromptHash:    digest("inkwords.textbook.sample.fixture.v2\n" + request.ProjectID + "\n" + string(request.Audience) + "\n" + request.BookContract.RevisionID + "\n" + request.StyleSheet.RevisionID + "\n" + sharedtextbook.GenerationEvidencePackHash(request.EvidencePack)),
	}, nil
}

var _ SampleGenerator = FakeSampleGenerator{}
