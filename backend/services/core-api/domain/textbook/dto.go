package textbook

import (
	"encoding/json"
	"io"
	"time"

	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// PersistDocumentsInput is a validated batch from a single immutable source snapshot.
type PersistDocumentsInput struct {
	SnapshotID uuid.UUID
	Documents  []sharedtextbook.SourceDocument
	Chunks     []sharedtextbook.SourceChunk
}

// PrepareSourceImportInput contains only source metadata. Raw file bytes stay
// in the parse task payload and never become a textbook-domain method input.
type PrepareSourceImportInput struct {
	ProjectID       uuid.UUID
	SourceID        uuid.UUID
	SnapshotID      uuid.UUID
	Filename        string
	ContentHash     string
	ResolvedVersion string
	ByteSize        int64
}

// PrepareOfficialWebImportInput identifies one already-confirmed official web
// source. The parser worker owns remote access; core-api only freezes the
// source/project boundary and allocates a future snapshot identity.
type PrepareOfficialWebImportInput struct {
	ProjectID           uuid.UUID
	SourceID            uuid.UUID
	SnapshotID          uuid.UUID
	AllowedPathPrefixes []string
}

func marshalStructuredJSON(value any) ([]byte, error) { return json.Marshal(value) }

type CreateProjectInput struct {
	WorkspaceID uuid.UUID
	Title       string
	Audience    sharedtextbook.AudienceLevel
	Primary     CreateSourceInput
}

type CreateSourceInput struct {
	Kind              sharedtextbook.SourceKind
	Role              sharedtextbook.SourceRole
	Locator           string
	OfficialConfirmed bool
	LicenseStatus     string
}

type CreateChapterInput struct {
	ProjectID      uuid.UUID
	SortOrder      int
	Title          string
	ChapterProfile sharedtextbook.ChapterProfile
}

// SampleGenerationPreflight is a read-only estimate for one exact frozen input.
// Provider billing remains unknown until a task actually runs.
type SampleGenerationPreflight struct {
	TaskVersion            int                                   `json:"task_version"`
	PromptSchemaVersion    string                                `json:"prompt_schema_version"`
	QualityContractVersion string                                `json:"quality_contract_version"`
	GenerationTarget       sharedtextbook.SampleGenerationTarget `json:"generation_target"`
	InputHash              string                                `json:"input_hash"`
	EvidenceReferenceCount int                                   `json:"evidence_reference_count"`
	EstimatedInputTokens   int                                   `json:"estimated_input_tokens"`
	AllowedInputTokens     int                                   `json:"allowed_input_tokens"`
	ReservedOutputTokens   int                                   `json:"reserved_output_tokens"`
	WithinBudget           bool                                  `json:"within_budget"`
	EstimateMethod         string                                `json:"estimate_method"`
	CacheStatus            string                                `json:"cache_status"`
	EstimatedCostKnown     bool                                  `json:"estimated_cost_known"`
	RequiresConfirmation   bool                                  `json:"requires_confirmation"`
}

// CreateBookBuildInput deliberately has no status or revision identifiers:
// they are derived from the project's current approved state by the repository.
type CreateBookBuildInput struct {
	ProjectID uuid.UUID
	Notices   []sharedtextbook.PublicationNoticeDraft
}

type AddRightsItemInput struct {
	BuildID           uuid.UUID
	SubjectRef        string
	WorkType          sharedtextbook.RightsWorkType
	RightsBasis       string
	AllowedUse        string
	Attribution       string
	PublicationStatus sharedtextbook.RightsStatus
}

type CompletePublicationReviewInput struct {
	ID               uuid.UUID                             `json:"id"`
	BuildID          uuid.UUID                             `json:"-"`
	ManifestHash     string                                `json:"manifest_hash"`
	ExpectedRevision int                                   `json:"expected_revision"`
	Stage            sharedtextbook.PublicationReviewStage `json:"stage"`
	Reviewer         string                                `json:"reviewer"`
	Verdict          string                                `json:"verdict"`
	Score            *int                                  `json:"score"`
	Scope            string                                `json:"scope"`
	Notes            string                                `json:"notes"`
	EvidenceRefs     []string                              `json:"evidence_refs"`
	HardFailures     []string                              `json:"hard_failures"`
}

// CreateBookContractInput contains the editable teaching promises before the service assigns an immutable revision identity.
type CreateBookContractInput struct {
	ProjectID          uuid.UUID
	Reader             sharedtextbook.ReaderModel
	Promise            string
	ChapterProfiles    []sharedtextbook.ChapterProfile
	TerminologyVersion string
	PublicationProfile string
}

// CreateStyleSheetInput contains the writing rules before the service assigns an immutable revision identity.
type CreateStyleSheetInput struct {
	ProjectID        uuid.UUID
	Language         string
	TerminologyRules []string
	CodeRules        []string
	VisualRules      []string
	CitationRules    []string
	ForbiddenPhrases []string
}

// CreateBlueprintInput contains a reviewable teaching outline. The repository binds it to the
// project's currently approved book contract and style sheet instead of trusting client-supplied IDs.
type CreateBlueprintInput struct {
	ProjectID uuid.UUID
	Volumes   []sharedtextbook.BlueprintVolume
}

type AppendRevisionInput struct {
	ChapterID              uuid.UUID
	ExpectedVersion        int
	Kind                   sharedtextbook.RevisionKind
	Markdown               string
	DocumentJSON           []byte
	ContentHash            string
	CreatedBy              string
	LockOwnerID            uuid.UUID
	LockVersion            int
	ParentRevisionID       *uuid.UUID
	BookContractRevisionID *uuid.UUID
	StyleSheetRevisionID   *uuid.UUID
	BlueprintRevisionID    *uuid.UUID
	EvidencePackHash       string
	PromptHash             string
	ProviderName           string
	ModelName              string
	ProviderUsageJSON      []byte
	QualityReportJSON      []byte
	GenerationTaskID       *uuid.UUID
}

// RegisterGeneratedCodeArtifactInput is internal-only: a generation adapter
// supplies a manifest and generated bytes, never an HTTP filesystem path.
type RegisterGeneratedCodeArtifactInput struct {
	RevisionID   uuid.UUID
	Artifact     sharedtextbook.CodeArtifact
	ManifestJSON []byte
}

// RegisterManuscriptAssetInput contains metadata for one already-staged visual.
// No filesystem path is persisted or accepted at this boundary.
type RegisterManuscriptAssetInput struct {
	ChapterID uuid.UUID
	Asset     sharedtextbook.ManuscriptAsset
}

type VisualAssetUploadInput struct {
	ChapterID, RevisionID, EvidenceID            uuid.UUID
	MediaType, AltText, Source, GenerationMethod string
	VisualPurpose                                sharedtextbook.VisualEvidencePurpose
	RightsStatus                                 sharedtextbook.RightsStatus
	Content                                      io.Reader
}

// GeneratedRevisionContext lets a trusted post-generation projection locate
// its candidate without exposing a task-selected revision to the browser.
type GeneratedRevisionContext struct {
	Revision         ChapterRevision
	WorkspaceID      uuid.UUID
	BookContractHash string
	StyleSheetHash   string
}

type LockInput struct {
	ChapterID       uuid.UUID
	OwnerID         uuid.UUID
	ExpectedVersion int
	LeaseDuration   time.Duration
}

type ApplyCandidateInput struct {
	ReviewerKind        string
	DelegationNote      string
	ChapterID           uuid.UUID
	CandidateRevisionID uuid.UUID
	ExpectedVersion     int
	LockOwnerID         uuid.UUID
	LockVersion         int
	ReviewNote          string
	DimensionScores     []sharedtextbook.DimensionScore
}

type RejectCandidateInput struct {
	ChapterID           uuid.UUID
	CandidateRevisionID uuid.UUID
	ExpectedVersion     int
	LockOwnerID         uuid.UUID
	LockVersion         int
	Reason              string
}

// ProjectWorkspace is the server-authoritative state required to edit one textbook.
// Revisions stay behind chapter-specific endpoints so opening a project remains bounded.
type ProjectWorkspace struct {
	Project         *Project              `json:"project"`
	Sources         []Source              `json:"sources"`
	Chapters        []Chapter             `json:"chapters"`
	BookContract    *BookContractRevision `json:"book_contract,omitempty"`
	StyleSheet      *StyleSheetRevision   `json:"style_sheet,omitempty"`
	Blueprint       *BlueprintRevision    `json:"blueprint,omitempty"`
	LatestBookBuild *BookBuildRow         `json:"latest_book_build,omitempty"`
}

// EditorialWorkspace is the complete, bounded evidence set used by the explicit
// publication-candidate decision for one frozen build.
type EditorialWorkspace struct {
	Build                   *BookBuildRow                               `json:"build"`
	RequiredRightsSubjects  []sharedtextbook.RightsSubject              `json:"required_rights_subjects"`
	RightsItems             []RightsItemRow                             `json:"rights_items"`
	RightsLedger            sharedtextbook.RightsLedger                 `json:"rights_ledger"`
	HumanReviews            []PublicationReviewRow                      `json:"human_reviews"`
	DelegatedReviewContract string                                      `json:"delegated_review_contract"`
	DelegatedReviews        []sharedtextbook.DelegatedPublicationReview `json:"delegated_reviews"`
	AutomatedChecks         []sharedtextbook.AutomatedPublicationCheck  `json:"automated_checks"`
	Preflight               sharedtextbook.PublicationPreflightResult   `json:"preflight"`
}

// ProjectStageState is a backend-derived production-stage status. A stage is never inferred by
// the browser so a stale tab cannot present a local guess as an approved production fact.
type ProjectStageState struct {
	Key    string `json:"key"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// ProjectProgress contains every stage that the textbook workspace currently exposes.
type ProjectProgress struct {
	Stages []ProjectStageState `json:"stages"`
}

// SourceLibraryDocument is a bounded metadata row for the source-library view.
type SourceLibraryDocument struct {
	ID               string    `json:"id"`
	SourceID         uuid.UUID `json:"source_id"`
	SnapshotID       uuid.UUID `json:"snapshot_id"`
	CanonicalLocator string    `json:"canonical_locator"`
	Title            string    `json:"title"`
	MediaType        string    `json:"media_type"`
	ContentHash      string    `json:"content_hash"`
	ArtifactPath     string    `json:"artifact_path,omitempty"`
	ChunkCount       int64     `json:"chunk_count"`
}

// SourceLibraryEvidence is a bounded, non-excerpt identifier that a human can select for a blueprint chapter.
// It deliberately omits source text so opening the workspace never loads the complete document stack.
type SourceLibraryEvidence struct {
	Locator          json.RawMessage `json:"locator"`
	ID               string          `json:"id"`
	DocumentID       string          `json:"document_id"`
	DocumentTitle    string          `json:"document_title"`
	CanonicalLocator string          `json:"canonical_locator"`
	Ordinal          int             `json:"ordinal"`
}

// RetrieveSourceInput is a bounded local lookup request. The returned plan
// contains only provenance and score explanations; excerpts remain server-side.
type RetrieveSourceInput struct {
	ProjectID uuid.UUID
	Query     string
	Limit     int
}

type SourceRetrievalCandidate struct {
	Locator          sharedtextbook.EvidenceLocator `json:"locator"`
	ChunkID          string                         `json:"chunk_id"`
	DocumentID       string                         `json:"document_id"`
	DocumentTitle    string                         `json:"document_title"`
	SnapshotID       string                         `json:"snapshot_id"`
	SourceRole       sharedtextbook.SourceRole      `json:"source_role"`
	CanonicalLocator string                         `json:"canonical_locator"`
	ArtifactPath     string                         `json:"artifact_path,omitempty"`
	Ordinal          int                            `json:"ordinal"`
	Score            int                            `json:"score"`
	Reasons          []string                       `json:"reasons"`
}

type SourceRetrievalPlan struct {
	ID         uuid.UUID                  `json:"id"`
	Query      string                     `json:"query"`
	InputHash  string                     `json:"input_hash"`
	Candidates []SourceRetrievalCandidate `json:"candidates"`
	Selected   []SourceRetrievalCandidate `json:"selected"`
	CreatedAt  time.Time                  `json:"created_at"`
}

// ChapterGenerationTaskRef lets the editor recover the newest server-owned
// generation task without exposing its frozen payload in the chapter response.
type ChapterGenerationTaskRef struct {
	TaskID    uuid.UUID `json:"task_id" gorm:"column:task_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// ChapterWorkspace is the bounded editing state for one workspace-owned chapter.
// It deliberately includes immutable revision history so the client never has to infer it.
type ChapterWorkspace struct {
	Chapter             *Chapter                                 `json:"chapter"`
	GenerationTarget    sharedtextbook.SampleGenerationTarget    `json:"generation_target"`
	HumanReviewContract sharedtextbook.SampleHumanReviewContract `json:"human_review_contract"`
	Revisions           []ChapterRevision                        `json:"revisions"`
	CandidateReviews    []CandidateReview                        `json:"candidate_reviews"`
	CodeArtifacts       []CodeArtifactRow                        `json:"code_artifacts"`
	RuntimeEvidence     []RuntimeEvidenceRow                     `json:"runtime_evidence"`
	Assets              []ManuscriptAssetRow                     `json:"assets"`
	LatestSampleTask    *ChapterGenerationTaskRef                `json:"latest_sample_task,omitempty"`
	Lock                *ChapterLock                             `json:"lock,omitempty"`
}
