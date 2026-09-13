package textbook

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

const ginFixtureCommit = "73726dc606796a025971fe451f0aa6f1b9b847f6"

// SampleTaskRunner turns one frozen textbook task into an unapproved candidate result.
// Source evidence remains scoped to the fixed Gin snapshot for all supported audiences.
type SampleTaskRunner struct {
	generator       SampleGenerator
	correctionStore RejectedDraftLoader
}

// SampleGenerationRejectedError carries only safe, structured failure
// telemetry across the in-process worker boundary. The rejected manuscript is
// intentionally omitted so a hard-gate failure cannot become an implicit
// candidate. The quality report may retain bounded, non-executable excerpts
// locating specific failures, never the whole rejected manuscript.
type SampleGenerationRejectedError struct {
	cause   error
	failure sharedtextbook.SampleGenerationTaskFailureResult
}

func (err *SampleGenerationRejectedError) Error() string {
	if err == nil || err.cause == nil {
		return "generated textbook candidate was rejected"
	}
	return err.cause.Error()
}

func (err *SampleGenerationRejectedError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.cause
}

// FailureResult returns a copy of the safe failure evidence for task storage.
func (err *SampleGenerationRejectedError) FailureResult() sharedtextbook.SampleGenerationTaskFailureResult {
	if err == nil {
		return sharedtextbook.SampleGenerationTaskFailureResult{}
	}
	return err.failure
}

// NewSampleTaskRunner constructs the worker boundary with a deterministic generator by default.
func NewSampleTaskRunner(generators ...SampleGenerator) *SampleTaskRunner {
	generator := SampleGenerator(FakeSampleGenerator{})
	if len(generators) > 0 && generators[0] != nil {
		generator = generators[0]
	}
	return &SampleTaskRunner{generator: generator}
}

// Run validates the frozen input before and after generation. It never writes textbook state.
func (r *SampleTaskRunner) Run(ctx context.Context, payload sharedtextbook.SampleGenerationTaskPayload) (sharedtextbook.SampleGenerationTaskResult, error) {
	if r == nil || r.generator == nil {
		return sharedtextbook.SampleGenerationTaskResult{}, fmt.Errorf("sample task runner is not configured")
	}
	if err := payload.Validate(); err != nil {
		return sharedtextbook.SampleGenerationTaskResult{}, err
	}
	request, aliases, err := ginFixtureRequest(payload)
	if err != nil {
		return sharedtextbook.SampleGenerationTaskResult{}, err
	}
	if payload.Correction != nil {
		return r.runCorrection(ctx, payload, request, aliases)
	}
	generation, err := r.generator.Generate(ctx, request)
	if err != nil {
		if generation.ProviderName != "" && len(generation.Quality.Failures) > 0 {
			usageJSON, qualityJSON, marshalErr := marshalSampleGenerationEvidence(generation)
			if marshalErr != nil {
				return sharedtextbook.SampleGenerationTaskResult{}, marshalErr
			}
			failure := sharedtextbook.SampleGenerationTaskFailureResult{
				ResultVersion: 1, TaskSubtype: sharedtextbook.TextbookSampleGenerationTaskSubtype, FinalStatus: "failed",
				ProjectID: payload.ProjectID, ChapterID: payload.ChapterID, InputHash: payload.InputHash,
				ProviderName: generation.ProviderName, ModelName: generation.ModelName,
				ProviderUsageJSON: usageJSON, QualityReportJSON: qualityJSON, PromptHash: generation.PromptHash,
				CandidatePersisted: false,
				RejectedDraftHash:  generation.RejectedDraftHash, RejectedDraftStorage: generation.RejectedDraftStorage,
			}
			if validateErr := failure.ValidateAgainst(payload); validateErr != nil {
				return sharedtextbook.SampleGenerationTaskResult{}, fmt.Errorf("validate rejected sample telemetry: %w", validateErr)
			}
			return sharedtextbook.SampleGenerationTaskResult{}, &SampleGenerationRejectedError{cause: err, failure: failure}
		}
		return sharedtextbook.SampleGenerationTaskResult{}, err
	}
	return buildSampleTaskResult(payload, generation, aliases, nil)
}

func buildSampleTaskResult(payload sharedtextbook.SampleGenerationTaskPayload, generation SampleGeneration, aliases map[string]string, correction *sharedtextbook.SampleCorrectionProvenance) (sharedtextbook.SampleGenerationTaskResult, error) {
	runbook, err := BuildGinSampleVideoRunbookProjection(generation.Chapter.Markdown, generation.Chapter.ContentHash)
	if err != nil {
		return sharedtextbook.SampleGenerationTaskResult{}, fmt.Errorf("build Gin video runbook: %w", err)
	}
	documentJSON, err := json.Marshal(struct {
		Format          string                                     `json:"format"`
		Sample          SampleChapter                              `json:"sample"`
		EvidenceAliases map[string]string                          `json:"evidence_aliases"`
		VideoRunbook    sharedtextbook.VideoRunbookProjection      `json:"video_runbook"`
		Correction      *sharedtextbook.SampleCorrectionProvenance `json:"correction,omitempty"`
	}{Format: "inkwords.textbook.sample.v1", Sample: generation.Chapter, EvidenceAliases: aliases, VideoRunbook: runbook, Correction: correction})
	if err != nil {
		return sharedtextbook.SampleGenerationTaskResult{}, fmt.Errorf("marshal sample chapter: %w", err)
	}
	usageJSON, qualityJSON, err := marshalSampleGenerationEvidence(generation)
	if err != nil {
		return sharedtextbook.SampleGenerationTaskResult{}, err
	}
	result := sharedtextbook.SampleGenerationTaskResult{
		ResultVersion:          1,
		TaskSubtype:            sharedtextbook.TextbookSampleGenerationTaskSubtype,
		ProjectID:              payload.ProjectID,
		ChapterID:              payload.ChapterID,
		ExpectedChapterVersion: payload.ExpectedChapterVersion,
		ParentRevisionID:       payload.ParentRevisionID,
		InputHash:              payload.InputHash,
		Markdown:               generation.Chapter.Markdown,
		DocumentJSON:           documentJSON,
		ContentHash:            generation.Chapter.ContentHash,
		BookContractRevision:   payload.BookContract.RevisionID,
		StyleSheetRevision:     payload.StyleSheet.RevisionID,
		BlueprintRevision:      payload.Blueprint.RevisionID,
		EvidencePackHash:       sharedtextbook.GenerationEvidencePackHash(payload.EvidencePack),
		PromptHash:             generation.PromptHash,
		ProviderName:           generation.ProviderName,
		ModelName:              generation.ModelName,
		ProviderUsageJSON:      usageJSON,
		QualityReportJSON:      qualityJSON,
		Correction:             correction,
	}
	if correction != nil {
		result.ResultVersion = 2
	}
	if err := result.ValidateAgainst(payload); err != nil {
		return sharedtextbook.SampleGenerationTaskResult{}, fmt.Errorf("validate sample result: %w", err)
	}
	return result, nil
}

func marshalSampleGenerationEvidence(generation SampleGeneration) (json.RawMessage, json.RawMessage, error) {
	qualityJSON, err := json.Marshal(generation.Quality)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal quality report: %w", err)
	}
	usageJSON, err := json.Marshal(generation.UsageRecord())
	if err != nil {
		return nil, nil, fmt.Errorf("marshal provider usage: %w", err)
	}
	return usageJSON, qualityJSON, nil
}

func ginFixtureRequest(payload sharedtextbook.SampleGenerationTaskPayload) (SampleGenerationRequest, map[string]string, error) {
	for _, snapshot := range append([]sharedtextbook.SourceSnapshot{payload.EvidencePack.PrimarySnapshot}, payload.EvidencePack.PrimarySnapshots...) {
		if snapshot.Kind != sharedtextbook.SourceKindGitRepository || strings.TrimSuffix(snapshot.Locator, "/") != "https://github.com/gin-gonic/gin" || snapshot.ResolvedVersion != ginFixtureCommit {
			return SampleGenerationRequest{}, nil, fmt.Errorf("deterministic sample supports only the fixed Gin v1.12.0 primary snapshot")
		}
	}
	aliasesBySymbol := map[string]string{
		"(*RouterGroup).GET":          "gin-routergroup-get",
		"(*RouterGroup).handle":       "gin-routergroup-handle",
		"(*Engine).addRoute":          "gin-engine-add-route",
		"(*Engine).ServeHTTP":         "gin-engine-serve-http",
		"(*Engine).handleHTTPRequest": "gin-engine-handle-http-request",
		"(*node).getValue":            "gin-node-get-value",
	}
	aliases := make(map[string]string, len(aliasesBySymbol))
	aliasByChunkID := make(map[string]string, len(aliasesBySymbol))
	requiredSymbols := 0
	pack := EvidencePack{PrimarySnapshot: payload.EvidencePack.PrimarySnapshot, PrimarySnapshots: append([]sharedtextbook.SourceSnapshot(nil), payload.EvidencePack.PrimarySnapshots...), OfficialSources: append([]sharedtextbook.SourceSnapshot(nil), payload.EvidencePack.OfficialSources...), Excerpts: make(map[string]string, len(payload.EvidencePack.Evidence))}
	for _, evidence := range payload.EvidencePack.Evidence {
		alias, known := aliasesBySymbol[evidence.Locator.Symbol]
		known = known && evidence.SourceRole == sharedtextbook.SourceRolePrimary
		if !known {
			// Keep every explicitly selected supporting excerpt. Stable aliases
			// for the six lifecycle anchors must not discard deeper mechanisms.
			alias = "source-" + evidence.ChunkID
		} else {
			requiredSymbols++
		}
		if _, duplicate := aliases[alias]; duplicate {
			return SampleGenerationRequest{}, nil, fmt.Errorf("Gin fixture has duplicate evidence for %s", evidence.Locator.Symbol)
		}
		evidence.ID = alias
		pack.Evidence = append(pack.Evidence, evidence)
		pack.Excerpts[alias] = payload.EvidencePack.Excerpts["evidence-"+evidence.ChunkID]
		aliases[alias] = "evidence-" + evidence.ChunkID
		aliasByChunkID[evidence.ChunkID] = alias
	}
	if requiredSymbols != len(aliasesBySymbol) {
		return SampleGenerationRequest{}, nil, fmt.Errorf("Gin fixture requires registration and request lookup source symbols")
	}
	blueprintChapter, ok := blueprintChapterByID(payload.Blueprint, payload.ChapterID)
	if !ok {
		return SampleGenerationRequest{}, nil, fmt.Errorf("Gin fixture chapter is missing from the approved blueprint")
	}
	aliasedChapter, err := aliasBlueprintChapterEvidence(blueprintChapter, aliasByChunkID)
	if err != nil {
		return SampleGenerationRequest{}, nil, err
	}
	return SampleGenerationRequest{ProjectID: payload.ProjectID, GenerationTarget: payload.GenerationTarget, Audience: payload.Audience, BookContract: payload.BookContract, StyleSheet: payload.StyleSheet, BlueprintChapter: aliasedChapter, EvidencePack: pack}, aliases, nil
}

func blueprintChapterByID(blueprint sharedtextbook.Blueprint, chapterID string) (sharedtextbook.BlueprintChapter, bool) {
	for _, volume := range blueprint.Volumes {
		for _, chapter := range volume.Chapters {
			if chapter.ID == chapterID {
				return chapter, true
			}
		}
	}
	return sharedtextbook.BlueprintChapter{}, false
}

func aliasBlueprintChapterEvidence(chapter sharedtextbook.BlueprintChapter, aliases map[string]string) (sharedtextbook.BlueprintChapter, error) {
	aliased := chapter
	aliased.EvidenceIDs = make([]string, 0, len(chapter.EvidenceIDs))
	for _, evidenceID := range chapter.EvidenceIDs {
		alias, ok := aliases[evidenceID]
		if !ok {
			return sharedtextbook.BlueprintChapter{}, fmt.Errorf("Gin fixture blueprint evidence %q has no source alias", evidenceID)
		}
		aliased.EvidenceIDs = append(aliased.EvidenceIDs, alias)
	}
	aliased.CriticalClaims = make([]sharedtextbook.ClaimRequirement, 0, len(chapter.CriticalClaims))
	for _, requirement := range chapter.CriticalClaims {
		mapped := requirement
		mapped.EvidenceIDs = make([]string, 0, len(requirement.EvidenceIDs))
		for _, evidenceID := range requirement.EvidenceIDs {
			alias, ok := aliases[evidenceID]
			if !ok {
				return sharedtextbook.BlueprintChapter{}, fmt.Errorf("Gin fixture critical claim evidence %q has no source alias", evidenceID)
			}
			mapped.EvidenceIDs = append(mapped.EvidenceIDs, alias)
		}
		aliased.CriticalClaims = append(aliased.CriticalClaims, mapped)
	}
	return aliased, nil
}
