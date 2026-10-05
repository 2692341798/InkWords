package textbook

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// TextbookSampleGenerationTaskSubtype is the versioned generation-queue kind for one candidate chapter.
const TextbookSampleGenerationTaskSubtype = "textbook_sample_generate"
const SampleGenerationStage = "sample_generation"
const SampleGenerationTaskVersion = 3
const SamplePromptSchemaVersion = "inkwords.textbook.sample.v17"
const SampleQualityContractVersion = "inkwords.sample-quality.v10"
const SampleFixtureProviderName = "fixture"
const SampleFixtureModelName = "deterministic-gin-v1.12.0"
const SampleMaxInputTokens = 32_000
const SampleReservedOutputTokens = 12_000
const SamplePreflightInstructionReserveTokens = 8_000

// SampleGenerationTarget freezes the selected provider/model into the task
// identity so a worker cannot silently execute a queued task with a different
// runtime configuration.
type SampleGenerationTarget struct {
	ProviderName string `json:"provider_name"`
	ModelName    string `json:"model_name"`
}

func (target SampleGenerationTarget) Validate() error {
	if strings.TrimSpace(target.ProviderName) == "" || strings.TrimSpace(target.ModelName) == "" {
		return fmt.Errorf("sample generation target is incomplete")
	}
	return nil
}

// GenerationEvidencePack is the narrow, immutable source window a generation worker may read.
// It intentionally contains selected excerpts rather than a project-wide document corpus.
type GenerationEvidencePack struct {
	PrimarySnapshot  SourceSnapshot    `json:"primary_snapshot"`
	PrimarySnapshots []SourceSnapshot  `json:"primary_snapshots,omitempty"`
	OfficialSources  []SourceSnapshot  `json:"official_sources,omitempty"`
	Evidence         []EvidenceRef     `json:"evidence"`
	Excerpts         map[string]string `json:"excerpts"`
}

// Validate rejects floating, unlocatable, or uncited source material before prose is built.
func (pack GenerationEvidencePack) Validate() error {
	if err := pack.PrimarySnapshot.Validate(); err != nil {
		return fmt.Errorf("validate primary snapshot: %w", err)
	}
	if pack.PrimarySnapshot.Role != SourceRolePrimary {
		return fmt.Errorf("primary snapshot must have primary role")
	}
	seenPrimary := map[string]bool{pack.PrimarySnapshot.ID: true}
	for _, source := range pack.PrimarySnapshots {
		if err := source.Validate(); err != nil {
			return fmt.Errorf("validate additional primary snapshot: %w", err)
		}
		if source.Role != SourceRolePrimary {
			return fmt.Errorf("additional primary snapshot %q does not have primary role", source.ID)
		}
		if source.SourceID != pack.PrimarySnapshot.SourceID || source.Kind != pack.PrimarySnapshot.Kind || strings.TrimSuffix(source.Locator, "/") != strings.TrimSuffix(pack.PrimarySnapshot.Locator, "/") || source.ResolvedVersion != pack.PrimarySnapshot.ResolvedVersion {
			return fmt.Errorf("additional primary snapshot %q does not belong to the same fixed primary source version", source.ID)
		}
		if seenPrimary[source.ID] {
			return fmt.Errorf("duplicate primary snapshot %q", source.ID)
		}
		seenPrimary[source.ID] = true
	}
	for _, source := range pack.OfficialSources {
		if err := source.Validate(); err != nil {
			return fmt.Errorf("validate official source: %w", err)
		}
		if source.Role != SourceRoleOfficial {
			return fmt.Errorf("supporting snapshot %q is not confirmed official", source.ID)
		}
	}
	if len(pack.Evidence) == 0 {
		return fmt.Errorf("generation needs source evidence")
	}
	knownSnapshots := seenPrimary
	for _, source := range pack.OfficialSources {
		knownSnapshots[source.ID] = true
	}
	seenEvidence := make(map[string]bool, len(pack.Evidence))
	for _, evidence := range pack.Evidence {
		if err := evidence.Validate(); err != nil {
			return fmt.Errorf("validate evidence: %w", err)
		}
		if evidence.SourceRole != SourceRolePrimary && evidence.SourceRole != SourceRoleOfficial {
			return fmt.Errorf("evidence %q has an unapproved source role", evidence.ID)
		}
		if !knownSnapshots[evidence.SnapshotID] {
			return fmt.Errorf("evidence %q references a snapshot outside the evidence pack", evidence.ID)
		}
		if seenEvidence[evidence.ID] {
			return fmt.Errorf("duplicate evidence %q", evidence.ID)
		}
		if strings.TrimSpace(pack.Excerpts[evidence.ID]) == "" {
			return fmt.Errorf("evidence %q has no source excerpt", evidence.ID)
		}
		seenEvidence[evidence.ID] = true
	}
	return nil
}

// GenerationPrimarySnapshotHash binds caches to every selected immutable file
// snapshot from the primary source, not merely whichever file was listed first.
func GenerationPrimarySnapshotHash(pack GenerationEvidencePack) string {
	parts := make([]string, 0, 1+len(pack.PrimarySnapshots))
	for _, source := range append([]SourceSnapshot{pack.PrimarySnapshot}, pack.PrimarySnapshots...) {
		parts = append(parts, source.ID+":"+source.ContentHash)
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// SampleGenerationTaskPayload is the only cross-service input for a textbook candidate.
// The core-api freezes it from approved revisions and selected source chunks before publishing.
type SampleGenerationTaskPayload struct {
	TaskVersion            int                    `json:"task_version"`
	TaskSubtype            string                 `json:"task_subtype"`
	PromptSchemaVersion    string                 `json:"prompt_schema_version"`
	QualityContractVersion string                 `json:"quality_contract_version"`
	GenerationTarget       SampleGenerationTarget `json:"generation_target"`
	ProjectID              string                 `json:"project_id"`
	ChapterID              string                 `json:"chapter_id"`
	ExpectedChapterVersion int                    `json:"expected_chapter_version"`
	ParentRevisionID       string                 `json:"parent_revision_id,omitempty"`
	Audience               AudienceLevel          `json:"audience"`
	BookContract           BookContract           `json:"book_contract"`
	StyleSheet             StyleSheet             `json:"style_sheet"`
	Blueprint              Blueprint              `json:"blueprint"`
	EvidencePack           GenerationEvidencePack `json:"evidence_pack"`
	InputHash              string                 `json:"input_hash"`
	Correction             *SampleCorrectionTask  `json:"correction,omitempty"`
}

// SampleGenerationTaskResult is the worker-to-core result for one unapproved manuscript candidate.
// The core-api validates it again before appending any revision.
type SampleGenerationTaskResult struct {
	ResultVersion          int                         `json:"result_version"`
	TaskSubtype            string                      `json:"task_subtype"`
	ProjectID              string                      `json:"project_id"`
	ChapterID              string                      `json:"chapter_id"`
	ExpectedChapterVersion int                         `json:"expected_chapter_version"`
	ParentRevisionID       string                      `json:"parent_revision_id,omitempty"`
	InputHash              string                      `json:"input_hash"`
	Markdown               string                      `json:"markdown"`
	DocumentJSON           json.RawMessage             `json:"document_json"`
	ContentHash            string                      `json:"content_hash"`
	BookContractRevision   string                      `json:"book_contract_revision"`
	StyleSheetRevision     string                      `json:"style_sheet_revision"`
	BlueprintRevision      string                      `json:"blueprint_revision"`
	EvidencePackHash       string                      `json:"evidence_pack_hash"`
	PromptHash             string                      `json:"prompt_hash"`
	ProviderName           string                      `json:"provider_name"`
	ModelName              string                      `json:"model_name"`
	ProviderUsageJSON      json.RawMessage             `json:"provider_usage_json"`
	QualityReportJSON      json.RawMessage             `json:"quality_report_json"`
	Correction             *SampleCorrectionProvenance `json:"correction,omitempty"`
}

// SampleGenerationTaskFailureResult retains safe provenance, usage, and
// deterministic quality evidence when a provider returned a structurally
// valid chapter that was rejected before it could become a candidate. It must
// never contain the rejected manuscript or raw provider diagnostics.
type SampleGenerationTaskFailureResult struct {
	ResultVersion        int             `json:"result_version"`
	TaskSubtype          string          `json:"task_subtype"`
	FinalStatus          string          `json:"final_status"`
	ProjectID            string          `json:"project_id"`
	ChapterID            string          `json:"chapter_id"`
	InputHash            string          `json:"input_hash"`
	ProviderName         string          `json:"provider_name"`
	ModelName            string          `json:"model_name"`
	ProviderUsageJSON    json.RawMessage `json:"provider_usage_json"`
	QualityReportJSON    json.RawMessage `json:"quality_report_json"`
	PromptHash           string          `json:"prompt_hash"`
	CandidatePersisted   bool            `json:"candidate_persisted"`
	RejectedDraftHash    string          `json:"rejected_draft_hash,omitempty"`
	RejectedDraftStorage string          `json:"rejected_draft_storage,omitempty"`
}

// ValidateAgainst ensures failed-generation telemetry belongs to the exact
// frozen task and describes a rejected current-contract report rather than an
// approvable candidate.
func (result SampleGenerationTaskFailureResult) ValidateAgainst(payload SampleGenerationTaskPayload) error {
	return result.validateAgainst(payload, false)
}

// ValidateRetainedCorrectionSource checks immutable failure provenance without
// authorizing another provider call under a retired prompt contract.
func (result SampleGenerationTaskFailureResult) ValidateRetainedCorrectionSource(payload SampleGenerationTaskPayload) error {
	return result.validateAgainst(payload, true)
}

func (result SampleGenerationTaskFailureResult) validateAgainst(payload SampleGenerationTaskPayload, retained bool) error {
	switch result.RejectedDraftStorage {
	case "":
		if result.RejectedDraftHash != "" {
			return fmt.Errorf("rejected draft storage state missing")
		}
	case "saved":
		if !isFullSHA256Digest("sha256:" + result.RejectedDraftHash) {
			return fmt.Errorf("rejected draft hash invalid")
		}
	case "unavailable":
		if result.RejectedDraftHash != "" {
			return fmt.Errorf("unavailable rejected draft cannot have a hash")
		}
	default:
		return fmt.Errorf("rejected draft storage state invalid")
	}
	validate := payload.Validate
	if retained {
		validate = payload.ValidateRetainedCorrectionSource
	}
	if err := validate(); err != nil {
		return fmt.Errorf("validate task payload: %w", err)
	}
	if result.ResultVersion != 1 || result.TaskSubtype != TextbookSampleGenerationTaskSubtype || result.FinalStatus != "failed" || result.ProjectID != payload.ProjectID || result.ChapterID != payload.ChapterID || result.InputHash != payload.InputHash {
		return fmt.Errorf("sample generation failure identity does not match task payload")
	}
	if result.ProviderName != payload.GenerationTarget.ProviderName || result.ModelName != payload.GenerationTarget.ModelName || !json.Valid(result.ProviderUsageJSON) || !json.Valid(result.QualityReportJSON) || !isSHA256Digest(result.PromptHash) || result.CandidatePersisted {
		return fmt.Errorf("sample generation failure provenance is invalid")
	}
	var report struct {
		ContractVersion string   `json:"contract_version"`
		Passed          bool     `json:"passed"`
		Failures        []string `json:"failures"`
	}
	if err := json.Unmarshal(result.QualityReportJSON, &report); err != nil {
		return fmt.Errorf("decode failed sample quality report: %w", err)
	}
	if report.ContractVersion != payload.QualityContractVersion || report.Passed || len(report.Failures) == 0 {
		return fmt.Errorf("sample generation failure quality report is invalid")
	}
	return nil
}

// ValidateCurrentSampleQualityReport prevents an old or unversioned automated
// report from remaining approvable after hard quality gates change.
func ValidateCurrentSampleQualityReport(raw json.RawMessage) error {
	var report struct {
		ContractVersion string `json:"contract_version"`
		Passed          bool   `json:"passed"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		return fmt.Errorf("decode sample quality report: %w", err)
	}
	if report.ContractVersion != SampleQualityContractVersion {
		return fmt.Errorf("sample quality report uses stale contract %q", report.ContractVersion)
	}
	if !report.Passed {
		return fmt.Errorf("sample quality report did not pass")
	}
	return nil
}

// ValidateAgainst rejects a result that is not a direct product of the frozen task input.
func (result SampleGenerationTaskResult) ValidateAgainst(payload SampleGenerationTaskPayload) error {
	if err := payload.Validate(); err != nil {
		return fmt.Errorf("validate task payload: %w", err)
	}
	expectedVersion := 1
	if payload.Correction != nil {
		expectedVersion = 2
	}
	if result.ResultVersion != expectedVersion || result.TaskSubtype != TextbookSampleGenerationTaskSubtype || result.ProjectID != payload.ProjectID || result.ChapterID != payload.ChapterID || result.ExpectedChapterVersion != payload.ExpectedChapterVersion || result.ParentRevisionID != payload.ParentRevisionID || result.InputHash != payload.InputHash {
		return fmt.Errorf("sample generation result identity does not match task payload")
	}
	if strings.TrimSpace(result.Markdown) == "" || !json.Valid(result.DocumentJSON) || !isSHA256Digest(result.ContentHash) || result.ContentHash != contentDigest(result.Markdown) {
		return fmt.Errorf("sample generation result content is invalid")
	}
	if result.BookContractRevision != payload.BookContract.RevisionID || result.StyleSheetRevision != payload.StyleSheet.RevisionID || result.BlueprintRevision != payload.Blueprint.RevisionID || result.EvidencePackHash != GenerationEvidencePackHash(payload.EvidencePack) || !isSHA256Digest(result.PromptHash) || result.ProviderName != payload.GenerationTarget.ProviderName || result.ModelName != payload.GenerationTarget.ModelName || !json.Valid(result.ProviderUsageJSON) || !json.Valid(result.QualityReportJSON) {
		return fmt.Errorf("sample generation result provenance does not match task payload")
	}
	if err := ValidateCurrentSampleQualityReport(result.QualityReportJSON); err != nil {
		return err
	}
	if payload.Correction == nil && result.Correction != nil {
		return fmt.Errorf("generation result cannot invent correction provenance")
	}
	if payload.Correction != nil {
		if result.Correction == nil {
			return fmt.Errorf("correction result is missing provenance")
		}
		if err := result.Correction.ValidateAgainst(*payload.Correction); err != nil {
			return err
		}
		var usage struct {
			Calls *int `json:"provider_call_count"`
			Known bool `json:"known"`
		}
		if json.Unmarshal(result.ProviderUsageJSON, &usage) != nil || usage.Calls == nil || *usage.Calls != 0 || usage.Known || result.PromptHash != result.Correction.CorrectionInputHash {
			return fmt.Errorf("local correction cannot claim a provider call")
		}
		var document struct {
			Correction *SampleCorrectionProvenance `json:"correction"`
			Sample     struct {
				Markdown            string `json:"markdown"`
				ContentHash         string `json:"content_hash"`
				GenerationMode      string `json:"generation_mode"`
				RuntimeVerification string `json:"runtime_verification"`
			} `json:"sample"`
		}
		if json.Unmarshal(result.DocumentJSON, &document) != nil || document.Correction == nil || document.Sample.Markdown != result.Markdown || document.Sample.ContentHash != result.ContentHash || document.Sample.GenerationMode != "automated_local_correction" || document.Sample.RuntimeVerification != "unverified" {
			return fmt.Errorf("correction document identity mismatch")
		}
		a, _ := json.Marshal(document.Correction)
		b, _ := json.Marshal(result.Correction)
		if string(a) != string(b) {
			return fmt.Errorf("correction document provenance mismatch")
		}
	}
	return nil
}

// Validate checks that the task was frozen from mutually consistent, approved-generation inputs.
func (payload SampleGenerationTaskPayload) Validate() error {
	return payload.validate(false)
}

func (payload SampleGenerationTaskPayload) validate(retained bool) error {
	expectedVersion := SampleGenerationTaskVersion
	if payload.Correction != nil {
		expectedVersion = SampleCorrectionTaskVersion
		if payload.Correction.SourceBaseline != nil {
			expectedVersion = SampleRebasedCorrectionTaskVersion
		}
		if payload.Correction.Migration != nil {
			expectedVersion = SampleMigratedCorrectionTaskVersion
		}
	}
	if payload.TaskVersion != expectedVersion || payload.TaskSubtype != TextbookSampleGenerationTaskSubtype || strings.TrimSpace(payload.ProjectID) == "" || strings.TrimSpace(payload.ChapterID) == "" || payload.ExpectedChapterVersion < 0 {
		return fmt.Errorf("sample generation task identity is incomplete")
	}
	compatibleCorrection := (retained || payload.Correction != nil) && payload.PromptSchemaVersion == "inkwords.textbook.sample.v15"
	retainedV16 := retained && payload.PromptSchemaVersion == "inkwords.textbook.sample.v16" && payload.QualityContractVersion == "inkwords.sample-quality.v9"
	if !retainedV16 && ((payload.PromptSchemaVersion != SamplePromptSchemaVersion && !compatibleCorrection) || payload.QualityContractVersion != SampleQualityContractVersion) {
		return fmt.Errorf("sample generation task uses stale generation contracts")
	}
	if err := payload.GenerationTarget.Validate(); err != nil {
		return err
	}
	if err := payload.Audience.Validate(); err != nil {
		return err
	}
	if err := payload.BookContract.Validate(); err != nil {
		return fmt.Errorf("validate book contract: %w", err)
	}
	if err := payload.StyleSheet.Validate(); err != nil {
		return fmt.Errorf("validate style sheet: %w", err)
	}
	if err := payload.Blueprint.Validate(); err != nil {
		return fmt.Errorf("validate blueprint: %w", err)
	}
	if payload.BookContract.ProjectID != payload.ProjectID || payload.BookContract.Reader.Audience != payload.Audience || payload.StyleSheet.ProjectID != payload.ProjectID || payload.Blueprint.ProjectID != payload.ProjectID || payload.Blueprint.BookContractRevision != payload.BookContract.RevisionID || payload.Blueprint.StyleSheetRevision != payload.StyleSheet.RevisionID {
		return fmt.Errorf("sample generation contracts do not match the task project")
	}
	if err := payload.EvidencePack.Validate(); err != nil {
		return fmt.Errorf("validate evidence pack: %w", err)
	}
	chapter, ok := payload.blueprintChapter()
	if !ok {
		return fmt.Errorf("sample generation chapter is not in the approved blueprint")
	}
	evidenceIDs := make(map[string]bool, len(payload.EvidencePack.Evidence))
	for _, evidence := range payload.EvidencePack.Evidence {
		evidenceIDs[evidence.ChunkID] = true
	}
	for _, evidenceID := range chapter.EvidenceIDs {
		if !evidenceIDs[evidenceID] {
			return fmt.Errorf("sample generation evidence pack omits blueprint evidence %q", evidenceID)
		}
	}
	if !isSHA256Digest(payload.InputHash) || payload.InputHash != SampleGenerationInputHash(payload) {
		return fmt.Errorf("sample generation input hash does not match payload")
	}
	if payload.Correction != nil {
		return payload.Correction.ValidateAgainst(payload)
	}
	return nil
}

func (payload SampleGenerationTaskPayload) blueprintChapter() (BlueprintChapter, bool) {
	for _, volume := range payload.Blueprint.Volumes {
		for _, chapter := range volume.Chapters {
			if chapter.ID == payload.ChapterID {
				return chapter, true
			}
		}
	}
	return BlueprintChapter{}, false
}

// SampleGenerationInputHash is stable across processes and excludes the self-referential hash field.
func SampleGenerationInputHash(payload SampleGenerationTaskPayload) string {
	payload.InputHash = ""
	payload.EvidencePack.Evidence = append([]EvidenceRef(nil), payload.EvidencePack.Evidence...)
	sort.Slice(payload.EvidencePack.Evidence, func(i, j int) bool { return payload.EvidencePack.Evidence[i].ID < payload.EvidencePack.Evidence[j].ID })
	encoded, _ := json.Marshal(payload)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// TaskStageInputHash gives each resumable worker stage a stable execution key.
// It binds a delivery's task identity to exactly one stage and frozen input, so
// repeated deliveries can be recognized without conflating a new task that
// happens to use identical source material.
func TaskStageInputHash(taskID, stage, inputHash string) string {
	if strings.TrimSpace(taskID) == "" || strings.TrimSpace(stage) == "" || !isSHA256Digest(inputHash) {
		return ""
	}
	sum := sha256.Sum256([]byte(taskID + "\n" + stage + "\n" + inputHash))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// GenerationEvidencePackHash identifies the exact evidence and excerpts supplied to a worker.
func GenerationEvidencePackHash(pack GenerationEvidencePack) string {
	encoded, _ := json.Marshal(pack)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func contentDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}
