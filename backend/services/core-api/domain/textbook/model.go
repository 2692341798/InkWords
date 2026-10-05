// Package textbook owns the authoritative persistence model for local textbook projects.
package textbook

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

const (
	StatusDraft              = "draft"
	StatusCandidate          = "candidate"
	StatusApproved           = "approved"
	StatusNeedsEvidence      = "needs_evidence"
	StatusVerificationFailed = "verification_failed"
	StatusArchived           = "archived"

	RevisionCreatorManual     = "manual"
	RevisionCreatorGeneration = "generation"
	CandidateDecisionApproved = "approved"
	CandidateDecisionRejected = "rejected"
)

// Project is a workspace-owned textbook, not a projection into the legacy blog table.
type Project struct {
	ID                             uuid.UUID                    `gorm:"type:uuid;primaryKey" json:"id"`
	WorkspaceID                    uuid.UUID                    `gorm:"type:uuid;not null;index" json:"workspace_id"`
	Title                          string                       `gorm:"type:text;not null" json:"title"`
	AudienceLevel                  sharedtextbook.AudienceLevel `gorm:"type:varchar(32);not null" json:"audience_level"`
	Status                         string                       `gorm:"type:varchar(32);not null" json:"status"`
	PrimarySourceID                *uuid.UUID                   `gorm:"type:uuid" json:"primary_source_id,omitempty"`
	ApprovedBookContractRevisionID *uuid.UUID                   `gorm:"type:uuid" json:"approved_book_contract_revision_id,omitempty"`
	ApprovedStyleSheetRevisionID   *uuid.UUID                   `gorm:"type:uuid" json:"approved_style_sheet_revision_id,omitempty"`
	ApprovedBlueprintRevisionID    *uuid.UUID                   `gorm:"type:uuid" json:"approved_blueprint_revision_id,omitempty"`
	RevisionVersion                int                          `gorm:"type:integer;not null" json:"revision_version"`
	CreatedAt                      time.Time                    `json:"created_at"`
	UpdatedAt                      time.Time                    `json:"updated_at"`
	DeletedAt                      gorm.DeletedAt               `gorm:"index" json:"-"`
}

func (Project) TableName() string { return "textbook_projects" }

func (p *Project) BeforeCreate(*gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// Source records one user-selected primary input or confirmed official supporting input.
type Source struct {
	ID                uuid.UUID                 `gorm:"type:uuid;primaryKey" json:"id"`
	ProjectID         uuid.UUID                 `gorm:"type:uuid;not null;index" json:"project_id"`
	Kind              sharedtextbook.SourceKind `gorm:"type:varchar(32);not null" json:"kind"`
	Role              sharedtextbook.SourceRole `gorm:"type:varchar(32);not null" json:"role"`
	Locator           string                    `gorm:"type:text;not null" json:"locator"`
	OfficialConfirmed bool                      `gorm:"not null" json:"official_confirmed"`
	LicenseStatus     string                    `gorm:"type:varchar(32);not null" json:"license_status"`
	CreatedAt         time.Time                 `json:"created_at"`
	UpdatedAt         time.Time                 `json:"updated_at"`
	DeletedAt         gorm.DeletedAt            `gorm:"index" json:"-"`
}

func (Source) TableName() string { return "textbook_sources" }

func (s *Source) BeforeCreate(*gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// SourceSnapshot pins an imported source version for reproducible evidence.
type SourceSnapshot struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	SourceID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"source_id"`
	ResolvedVersion string         `gorm:"type:text;not null" json:"resolved_version"`
	ContentHash     string         `gorm:"type:char(64);not null" json:"content_hash"`
	CapturedAt      time.Time      `json:"captured_at"`
	Status          string         `gorm:"type:varchar(32);not null" json:"status"`
	LimitsJSON      datatypes.JSON `gorm:"type:jsonb;not null" json:"limits_json"`
	CreatedAt       time.Time      `json:"created_at"`
}

func (SourceSnapshot) TableName() string { return "source_snapshots" }

func (snapshot *SourceSnapshot) BeforeCreate(*gorm.DB) error {
	if snapshot.ID == uuid.Nil {
		snapshot.ID = uuid.New()
	}
	return nil
}

// ParsedDocument is an immutable artifact inside a source snapshot.
type ParsedDocument struct {
	ID               string    `gorm:"type:text;primaryKey" json:"id"`
	SnapshotID       uuid.UUID `gorm:"type:uuid;not null;index" json:"snapshot_id"`
	CanonicalLocator string    `gorm:"type:text;not null" json:"canonical_locator"`
	Title            string    `gorm:"type:text;not null" json:"title"`
	MediaType        string    `gorm:"type:text;not null" json:"media_type"`
	ContentHash      string    `gorm:"type:text;not null" json:"content_hash"`
	ParentID         *string   `gorm:"type:text" json:"parent_id,omitempty"`
	ArtifactPath     string    `gorm:"type:text" json:"artifact_path,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

func (ParsedDocument) TableName() string { return "source_documents" }

// ParsedChunk is the queryable, citeable unit of a ParsedDocument.
type ParsedChunk struct {
	ID           string         `gorm:"type:text;primaryKey" json:"id"`
	DocumentID   string         `gorm:"type:text;not null;index" json:"document_id"`
	Ordinal      int            `gorm:"not null" json:"ordinal"`
	HeadingPath  datatypes.JSON `gorm:"type:jsonb;not null" json:"heading_path"`
	Locator      datatypes.JSON `gorm:"type:jsonb;not null" json:"locator"`
	Paragraph    *int           `json:"paragraph,omitempty"`
	CodeLanguage string         `gorm:"type:text" json:"code_language,omitempty"`
	StartByte    int            `gorm:"not null" json:"start_byte"`
	EndByte      int            `gorm:"not null" json:"end_byte"`
	TextHash     string         `gorm:"type:text;not null" json:"text_hash"`
	SearchText   string         `gorm:"type:text;not null" json:"search_text"`
	CreatedAt    time.Time      `json:"created_at"`
}

func (ParsedChunk) TableName() string { return "source_chunks" }

// RetrievalRun preserves a bounded, explainable selection decision without
// duplicating source excerpts outside their immutable document chunks.
type RetrievalRun struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	ProjectID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"project_id"`
	Query          string         `gorm:"type:text;not null" json:"query"`
	CandidatesJSON datatypes.JSON `gorm:"type:jsonb;not null" json:"candidates_json"`
	SelectedJSON   datatypes.JSON `gorm:"type:jsonb;not null" json:"selected_json"`
	InputHash      string         `gorm:"type:text;not null" json:"input_hash"`
	CreatedAt      time.Time      `json:"created_at"`
}

func (RetrievalRun) TableName() string { return "source_retrieval_runs" }
func (r *RetrievalRun) BeforeCreate(*gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// RevisionDocument is the common persisted shape for immutable book-level revisions.
type RevisionDocument struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	ProjectID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"project_id"`
	RevisionNumber int            `gorm:"type:integer;not null" json:"revision_number"`
	DocumentJSON   datatypes.JSON `gorm:"type:jsonb;not null" json:"document_json"`
	ContentHash    string         `gorm:"type:char(64);not null" json:"content_hash"`
	Status         string         `gorm:"type:varchar(32);not null" json:"status"`
	CreatedAt      time.Time      `json:"created_at"`
}

// BookContractRevision freezes the book-level learning contract at one revision.
type BookContractRevision RevisionDocument

func (BookContractRevision) TableName() string { return "book_contract_revisions" }

// StyleSheetRevision freezes terminology and presentation choices at one revision.
type StyleSheetRevision RevisionDocument

func (StyleSheetRevision) TableName() string { return "style_sheet_revisions" }

// BlueprintRevision records an inspectable plan before chapter generation.
type BlueprintRevision RevisionDocument

func (BlueprintRevision) TableName() string { return "blueprint_revisions" }

// Chapter is a stable position in a textbook. Its content lives in ChapterRevision rows.
type Chapter struct {
	ID                 uuid.UUID                     `gorm:"type:uuid;primaryKey" json:"id"`
	ProjectID          uuid.UUID                     `gorm:"type:uuid;not null;index" json:"project_id"`
	SortOrder          int                           `gorm:"type:integer;not null" json:"sort_order"`
	Title              string                        `gorm:"type:text;not null" json:"title"`
	ChapterProfile     sharedtextbook.ChapterProfile `gorm:"type:varchar(32);not null" json:"chapter_profile"`
	Status             string                        `gorm:"type:varchar(32);not null" json:"status"`
	CurrentRevisionID  *uuid.UUID                    `gorm:"type:uuid" json:"current_revision_id,omitempty"`
	ApprovedRevisionID *uuid.UUID                    `gorm:"type:uuid" json:"approved_revision_id,omitempty"`
	RevisionVersion    int                           `gorm:"type:integer;not null" json:"revision_version"`
	CreatedAt          time.Time                     `json:"created_at"`
	UpdatedAt          time.Time                     `json:"updated_at"`
	DeletedAt          gorm.DeletedAt                `gorm:"index" json:"-"`
}

func (Chapter) TableName() string { return "textbook_chapters" }
func (c *Chapter) BeforeCreate(*gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// ChapterRevision is append-only so approved or manually edited content cannot be overwritten.
type ChapterRevision struct {
	ID                     uuid.UUID                   `gorm:"type:uuid;primaryKey" json:"id"`
	ChapterID              uuid.UUID                   `gorm:"type:uuid;not null;index" json:"chapter_id"`
	RevisionNumber         int                         `gorm:"type:integer;not null" json:"revision_number"`
	Kind                   sharedtextbook.RevisionKind `gorm:"type:varchar(32);not null" json:"kind"`
	Markdown               string                      `gorm:"type:text;not null" json:"markdown"`
	DocumentJSON           datatypes.JSON              `gorm:"type:jsonb;not null" json:"document_json"`
	ContentHash            string                      `gorm:"type:char(64);not null" json:"content_hash"`
	CreatedBy              string                      `gorm:"type:varchar(32);not null" json:"created_by"`
	ParentRevisionID       *uuid.UUID                  `gorm:"type:uuid" json:"parent_revision_id,omitempty"`
	BookContractRevisionID *uuid.UUID                  `gorm:"type:uuid" json:"book_contract_revision_id,omitempty"`
	StyleSheetRevisionID   *uuid.UUID                  `gorm:"type:uuid" json:"style_sheet_revision_id,omitempty"`
	BlueprintRevisionID    *uuid.UUID                  `gorm:"type:uuid" json:"blueprint_revision_id,omitempty"`
	EvidencePackHash       string                      `gorm:"type:text" json:"evidence_pack_hash,omitempty"`
	PromptHash             string                      `gorm:"type:text" json:"prompt_hash,omitempty"`
	ProviderName           string                      `gorm:"type:text" json:"provider_name,omitempty"`
	ModelName              string                      `gorm:"type:text" json:"model_name,omitempty"`
	ProviderUsageJSON      datatypes.JSON              `gorm:"type:jsonb" json:"provider_usage_json,omitempty"`
	QualityReportJSON      datatypes.JSON              `gorm:"type:jsonb" json:"quality_report_json,omitempty"`
	GenerationTaskID       *uuid.UUID                  `gorm:"type:uuid;uniqueIndex:ux_chapter_revisions_generation_task,where:generation_task_id IS NOT NULL" json:"generation_task_id,omitempty"`
	CreatedAt              time.Time                   `json:"created_at"`
}

func (ChapterRevision) TableName() string { return "chapter_revisions" }
func (r *ChapterRevision) BeforeCreate(*gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// CandidateReview is an append-only human decision about one immutable candidate.
// Approval also creates a new approved revision in the same transaction.
type CandidateReview struct {
	ID                     uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	ProjectID              uuid.UUID      `gorm:"type:uuid;not null;index" json:"project_id"`
	ChapterID              uuid.UUID      `gorm:"type:uuid;not null;index" json:"chapter_id"`
	CandidateRevisionID    uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex" json:"candidate_revision_id"`
	ReviewerWorkspaceID    uuid.UUID      `gorm:"type:uuid;not null" json:"reviewer_workspace_id"`
	Decision               string         `gorm:"type:varchar(32);not null" json:"decision"`
	Reason                 string         `gorm:"type:text;not null" json:"reason"`
	CandidateContentHash   string         `gorm:"type:char(64);not null" json:"candidate_content_hash"`
	QualityContractVersion string         `gorm:"type:text;not null" json:"quality_contract_version"`
	HumanReviewJSON        datatypes.JSON `gorm:"type:jsonb;not null" json:"human_review"`
	CreatedAt              time.Time      `json:"created_at"`
}

func (CandidateReview) TableName() string { return "textbook_candidate_reviews" }
func (review *CandidateReview) BeforeCreate(*gorm.DB) error {
	if review.ID == uuid.Nil {
		review.ID = uuid.New()
	}
	return nil
}

// CodeArtifactRow persists only a generated teaching artifact. Source
// walkthroughs remain evidence references and are never executable inputs.
type CodeArtifactRow struct {
	ID              uuid.UUID                       `gorm:"type:uuid;primaryKey" json:"id"`
	RevisionID      uuid.UUID                       `gorm:"type:uuid;not null;index" json:"revision_id"`
	Kind            sharedtextbook.CodeArtifactKind `gorm:"type:varchar(32);not null" json:"kind"`
	Language        string                          `gorm:"type:varchar(64);not null" json:"language"`
	Entrypoint      string                          `gorm:"type:text" json:"entrypoint,omitempty"`
	SourceTreePath  string                          `gorm:"type:text" json:"source_tree_path,omitempty"`
	SourceRef       string                          `gorm:"type:text" json:"source_ref,omitempty"`
	ManifestJSON    datatypes.JSON                  `gorm:"type:jsonb;not null" json:"manifest_json"`
	ManifestHash    string                          `gorm:"type:text;not null" json:"manifest_hash"`
	ArtifactHash    string                          `gorm:"type:text;not null" json:"artifact_hash"`
	LimitationsJSON datatypes.JSON                  `gorm:"type:jsonb;not null" json:"limitations_json"`
	Status          sharedtextbook.ArtifactStatus   `gorm:"type:varchar(32);not null" json:"status"`
	CreatedAt       time.Time                       `json:"created_at"`
}

func (CodeArtifactRow) TableName() string { return "textbook_code_artifacts" }
func (artifact *CodeArtifactRow) BeforeCreate(*gorm.DB) error {
	if artifact.ID == uuid.Nil {
		artifact.ID = uuid.New()
	}
	return nil
}

// RuntimeEvidenceRow is append-only observation provenance. Currentness is
// derived by comparing its input hash with the current artifact and contracts.
type RuntimeEvidenceRow struct {
	ID                     uuid.UUID                          `gorm:"type:uuid;primaryKey" json:"id"`
	RevisionID             uuid.UUID                          `gorm:"type:uuid;not null;index" json:"revision_id"`
	CodeArtifactID         uuid.UUID                          `gorm:"type:uuid;not null;index" json:"code_artifact_id"`
	CodeArtifactHash       string                             `gorm:"type:text;not null" json:"code_artifact_hash"`
	InputHash              string                             `gorm:"type:text;not null" json:"input_hash"`
	Kind                   sharedtextbook.RuntimeEvidenceKind `gorm:"type:varchar(32);not null" json:"kind"`
	Status                 sharedtextbook.ArtifactStatus      `gorm:"type:varchar(32);not null" json:"status"`
	CommandManifestHash    string                             `gorm:"type:text" json:"command_manifest_hash,omitempty"`
	RunnerImageDigest      string                             `gorm:"type:text" json:"runner_image_digest,omitempty"`
	ToolchainVersion       string                             `gorm:"type:text" json:"toolchain_version,omitempty"`
	ToolName               string                             `gorm:"type:text" json:"tool_name,omitempty"`
	ToolVersion            string                             `gorm:"type:text" json:"tool_version,omitempty"`
	SamplingConditionsJSON datatypes.JSON                     `gorm:"type:jsonb;not null" json:"sampling_conditions_json"`
	StructuredOutput       string                             `gorm:"type:text" json:"structured_output,omitempty"`
	RawEvidenceRef         string                             `gorm:"type:text" json:"raw_evidence_ref,omitempty"`
	OutputTruncated        bool                               `gorm:"not null" json:"output_truncated"`
	CapturedAt             *time.Time                         `json:"captured_at,omitempty"`
	ExpiresAt              *time.Time                         `json:"expires_at,omitempty"`
	StaleReason            string                             `gorm:"type:text" json:"stale_reason,omitempty"`
	CreatedAt              time.Time                          `json:"created_at"`
}

func (RuntimeEvidenceRow) TableName() string { return "textbook_runtime_evidence" }
func (evidence *RuntimeEvidenceRow) BeforeCreate(*gorm.DB) error {
	if evidence.ID == uuid.Nil {
		evidence.ID = uuid.New()
	}
	var samplingConditions []string
	if err := json.Unmarshal(evidence.SamplingConditionsJSON, &samplingConditions); err != nil {
		return fmt.Errorf("decode runtime evidence sampling conditions: %w", err)
	}
	return sharedtextbook.RuntimeEvidence{
		ID:                  evidence.ID.String(),
		RevisionID:          evidence.RevisionID.String(),
		CodeArtifactID:      evidence.CodeArtifactID.String(),
		CodeArtifactHash:    evidence.CodeArtifactHash,
		InputHash:           evidence.InputHash,
		Kind:                evidence.Kind,
		Status:              evidence.Status,
		CommandManifestHash: evidence.CommandManifestHash,
		RunnerImageDigest:   evidence.RunnerImageDigest,
		ToolchainVersion:    evidence.ToolchainVersion,
		ToolName:            evidence.ToolName,
		ToolVersion:         evidence.ToolVersion,
		SamplingConditions:  samplingConditions,
		StructuredOutput:    evidence.StructuredOutput,
		RawEvidenceRef:      evidence.RawEvidenceRef,
		OutputTruncated:     evidence.OutputTruncated,
		CapturedAt:          dereferenceTime(evidence.CapturedAt),
		ExpiresAt:           evidence.ExpiresAt,
		StaleReason:         evidence.StaleReason,
	}.Validate()
}

type ManuscriptAssetRow struct {
	ID               uuid.UUID                            `gorm:"type:uuid;primaryKey" json:"id"`
	RevisionID       uuid.UUID                            `gorm:"type:uuid;not null;index" json:"revision_id"`
	EvidenceID       uuid.UUID                            `gorm:"type:uuid;not null;index" json:"evidence_id"`
	StableRef        string                               `gorm:"type:text;not null" json:"stable_ref"`
	Kind             sharedtextbook.AssetKind             `gorm:"type:varchar(32);not null" json:"kind"`
	ContentHash      string                               `gorm:"type:text;not null" json:"content_hash"`
	AltText          string                               `gorm:"type:text;not null" json:"alt_text"`
	Source           string                               `gorm:"type:text;not null" json:"source"`
	GenerationMethod string                               `gorm:"type:text;not null" json:"generation_method"`
	VisualPurpose    sharedtextbook.VisualEvidencePurpose `gorm:"type:varchar(32);not null" json:"visual_purpose"`
	RightsStatus     sharedtextbook.RightsStatus          `gorm:"type:varchar(32);not null" json:"rights_status"`
	Status           sharedtextbook.ArtifactStatus        `gorm:"type:varchar(32);not null" json:"status"`
	CreatedAt        time.Time                            `json:"created_at"`
}

// BookBuildRow is an immutable, project-owned freeze point. The exported
// formats must consume manifest_json rather than look up current chapter state.
type BookBuildRow struct {
	ID                     uuid.UUID                      `gorm:"type:uuid;primaryKey" json:"id"`
	ProjectID              uuid.UUID                      `gorm:"type:uuid;not null;index" json:"project_id"`
	BookContractRevisionID uuid.UUID                      `gorm:"type:uuid;not null" json:"book_contract_revision_id"`
	StyleSheetRevisionID   uuid.UUID                      `gorm:"type:uuid;not null" json:"style_sheet_revision_id"`
	ApprovedRevisionIDs    datatypes.JSON                 `gorm:"type:jsonb;not null" json:"approved_revision_ids"`
	ToolVersionsJSON       datatypes.JSON                 `gorm:"type:jsonb;not null" json:"tool_versions_json"`
	ManifestJSON           datatypes.JSON                 `gorm:"type:jsonb;not null" json:"manifest_json"`
	InputHash              string                         `gorm:"type:text;not null" json:"input_hash"`
	ManifestHash           string                         `gorm:"type:text;not null" json:"manifest_hash"`
	Status                 sharedtextbook.BookBuildStatus `gorm:"type:varchar(32);not null" json:"status"`
	BlockersJSON           datatypes.JSON                 `gorm:"type:jsonb;not null" json:"blockers_json"`
	CreatedAt              time.Time                      `json:"created_at"`
}

func (BookBuildRow) TableName() string { return "textbook_book_builds" }
func (build *BookBuildRow) BeforeCreate(*gorm.DB) error {
	if build.ID == uuid.Nil {
		build.ID = uuid.New()
	}
	return nil
}

// RightsItemRow is immutable publication-rights evidence for one frozen build.
// SubjectRef names the exact prose, code, image, font, trademark, or data item.
type RightsItemRow struct {
	ID                uuid.UUID                     `gorm:"type:uuid;primaryKey" json:"id"`
	BuildID           uuid.UUID                     `gorm:"type:uuid;not null;index" json:"build_id"`
	ProjectID         uuid.UUID                     `gorm:"type:uuid;not null;index" json:"project_id"`
	SubjectRef        string                        `gorm:"type:text;not null" json:"subject_ref"`
	WorkType          sharedtextbook.RightsWorkType `gorm:"type:varchar(32);not null" json:"work_type"`
	RightsBasis       string                        `gorm:"type:text;not null" json:"rights_basis"`
	AllowedUse        string                        `gorm:"type:text;not null" json:"allowed_use"`
	Attribution       string                        `gorm:"type:text;not null" json:"attribution"`
	PublicationStatus sharedtextbook.RightsStatus   `gorm:"type:varchar(32);not null" json:"publication_status"`
	CreatedAt         time.Time                     `json:"created_at"`
}

func (RightsItemRow) TableName() string { return "textbook_rights_items" }
func (item *RightsItemRow) BeforeCreate(*gorm.DB) error {
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	return nil
}

func (item RightsItemRow) ToContract() sharedtextbook.RightsItem {
	return sharedtextbook.RightsItem{
		ID: item.ID.String(), ProjectID: item.ProjectID.String(), BuildID: item.BuildID.String(),
		SubjectRef: item.SubjectRef, WorkType: item.WorkType, RightsBasis: item.RightsBasis,
		AllowedUse: item.AllowedUse, Attribution: item.Attribution, PublicationStatus: item.PublicationStatus,
	}
}

// PublicationReviewRow is a human-authored, build-specific review record.
// Database constraints keep automated checks out of this table.
type PublicationReviewRow struct {
	ContractVersion string                                `gorm:"type:text" json:"contract_version,omitempty"`
	ManifestHash    string                                `gorm:"type:text" json:"manifest_hash,omitempty"`
	Revision        int                                   `gorm:"not null;default:1" json:"revision,omitempty"`
	ReviewerKind    string                                `gorm:"type:text" json:"reviewer_kind,omitempty"`
	Verdict         string                                `gorm:"type:text" json:"verdict,omitempty"`
	Score           int                                   `json:"score"`
	Scope           string                                `gorm:"type:text" json:"scope,omitempty"`
	EvidenceRefs    datatypes.JSON                        `gorm:"type:jsonb" json:"evidence_refs,omitempty"`
	HardFailures    datatypes.JSON                        `gorm:"type:jsonb" json:"hard_failures,omitempty"`
	InputHash       string                                `gorm:"type:text" json:"-"`
	ID              uuid.UUID                             `gorm:"type:uuid;primaryKey" json:"id"`
	BuildID         uuid.UUID                             `gorm:"type:uuid;not null;index" json:"build_id"`
	Stage           sharedtextbook.PublicationReviewStage `gorm:"type:varchar(32);not null" json:"stage"`
	Reviewer        string                                `gorm:"type:text;not null" json:"reviewer"`
	Notes           string                                `gorm:"type:text;not null" json:"notes"`
	Automated       bool                                  `gorm:"not null" json:"automated"`
	CompletedAt     time.Time                             `json:"completed_at"`
	CreatedAt       time.Time                             `json:"created_at"`
}

func (PublicationReviewRow) TableName() string { return "textbook_publication_reviews" }
func (review *PublicationReviewRow) BeforeCreate(*gorm.DB) error {
	if review.ID == uuid.Nil {
		review.ID = uuid.New()
	}
	return nil
}

func (review PublicationReviewRow) ToContract() sharedtextbook.HumanPublicationReview {
	result := sharedtextbook.HumanPublicationReview{
		ID: review.ID.String(), BuildID: review.BuildID.String(), Stage: review.Stage,
		Reviewer: review.Reviewer, Notes: review.Notes, CompletedAt: review.CompletedAt, Automated: review.Automated,
	}
	if review.ContractVersion != "" {
		result.ContractVersion, result.ManifestHash, result.Revision = review.ContractVersion, review.ManifestHash, review.Revision
		result.ReviewerKind, result.Verdict, result.Score, result.Scope = review.ReviewerKind, review.Verdict, review.Score, review.Scope
		_ = json.Unmarshal(review.EvidenceRefs, &result.EvidenceRefs)
		_ = json.Unmarshal(review.HardFailures, &result.HardFailures)
	}
	return result
}

func (ManuscriptAssetRow) TableName() string { return "textbook_manuscript_assets" }
func (asset *ManuscriptAssetRow) BeforeCreate(*gorm.DB) error {
	if asset.ID == uuid.Nil {
		asset.ID = uuid.New()
	}
	return nil
}

// ChapterLock grants one editor a short, versioned lease for a chapter edit.
type ChapterLock struct {
	ChapterID      uuid.UUID `gorm:"type:uuid;primaryKey" json:"chapter_id"`
	OwnerID        uuid.UUID `gorm:"type:uuid;not null" json:"owner_id"`
	Version        int       `gorm:"type:integer;not null" json:"version"`
	LeaseExpiresAt time.Time `json:"lease_expires_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (ChapterLock) TableName() string { return "chapter_locks" }
