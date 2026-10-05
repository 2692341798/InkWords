package export

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// Blog is export-service's read model for rows in the shared blogs table.
type Blog struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	WorkspaceID uuid.UUID      `gorm:"type:uuid;index:idx_blogs_workspace_parent_chapter;not null" json:"workspace_id"`
	ParentID    *uuid.UUID     `gorm:"type:uuid;index:idx_blogs_workspace_parent_chapter" json:"parent_id"`
	ChapterSort int            `gorm:"type:integer;index:idx_blogs_workspace_parent_chapter" json:"chapter_sort"`
	Title       string         `gorm:"type:varchar(255);not null" json:"title"`
	Content     string         `gorm:"type:text;not null" json:"content"`
	SourceType  string         `gorm:"type:varchar(50);not null" json:"source_type"`
	SourceURL   string         `gorm:"type:varchar(512)" json:"source_url"`
	IsSeries    bool           `gorm:"type:boolean;default:false" json:"is_series"`
	Status      int16          `gorm:"type:smallint;default:0" json:"status"`
	WordCount   int            `gorm:"type:integer;default:0" json:"word_count"`
	TechStacks  datatypes.JSON `gorm:"type:jsonb" json:"tech_stacks"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Blog) TableName() string {
	return "blogs"
}

// TextbookChapterExport is the export-service's deliberately small, read-only view
// of an approved textbook chapter. It is not a second textbook domain model: the
// authoritative state stays in core-api, and this service follows its approved
// revision pointer before emitting any bytes.
type TextbookChapterExport struct {
	ChapterID              uuid.UUID  `json:"chapter_id"`
	Title                  string     `json:"title"`
	SortOrder              int        `json:"sort_order"`
	RevisionID             uuid.UUID  `json:"revision_id"`
	ProjectionRevisionID   *uuid.UUID `gorm:"column:projection_revision_id" json:"-"`
	RevisionNumber         int        `json:"revision_number"`
	Markdown               string     `json:"markdown"`
	ContentHash            string     `json:"content_hash"`
	BookContractRevisionID *uuid.UUID `json:"book_contract_revision_id,omitempty"`
	StyleSheetRevisionID   *uuid.UUID `json:"style_sheet_revision_id,omitempty"`
	BlueprintRevisionID    *uuid.UUID `json:"blueprint_revision_id,omitempty"`
	EvidencePackHash       string     `json:"evidence_pack_hash,omitempty"`
	PromptHash             string     `json:"prompt_hash,omitempty"`
	ProviderName           string     `json:"provider_name,omitempty"`
	ModelName              string     `json:"model_name,omitempty"`
	// Current contract hashes are read-only context used to derive whether an
	// old observation still supports the approved chapter. They are not part of
	// the portable manuscript projection.
	CurrentBookContractHash string `gorm:"-" json:"-"`
	CurrentStyleSheetHash   string `gorm:"-" json:"-"`
	// These are loaded by explicit read-only queries after the approved revision
	// projection. They are not GORM associations on this DTO.
	CodeArtifacts   []TextbookCodeArtifactExport    `gorm:"-" json:"code_artifacts"`
	RuntimeEvidence []TextbookRuntimeEvidenceExport `gorm:"-" json:"runtime_evidence"`
	Assets          []TextbookManuscriptAssetExport `gorm:"-" json:"assets"`
}

// TextbookBookBuildExport is the read-only, immutable export-service view of
// a BookBuild owned by core-api. The canonical AST comes from manifest_json so
// an export cannot follow a newer approved revision by accident.
type TextbookBookBuildExport struct {
	BuildID                uuid.UUID                                   `json:"build_id"`
	ManifestHash           string                                      `json:"manifest_hash"`
	ManifestJSON           json.RawMessage                             `json:"manifest_json"`
	Book                   sharedtextbook.CanonicalBookAST             `json:"book"`
	RequiredRightsSubjects []sharedtextbook.RightsSubject              `json:"required_rights_subjects"`
	RightsItems            []sharedtextbook.RightsItem                 `json:"rights_items"`
	RightsLedger           *sharedtextbook.RightsLedger                `json:"rights_ledger,omitempty"`
	HumanReviews           []sharedtextbook.HumanPublicationReview     `json:"human_reviews"`
	DelegatedReviews       []sharedtextbook.DelegatedPublicationReview `json:"delegated_reviews"`
	AutomatedChecks        []sharedtextbook.AutomatedPublicationCheck  `json:"automated_checks"`
	Preflight              sharedtextbook.PublicationPreflightResult   `json:"preflight"`
}

// TextbookCodeArtifactExport is the immutable code projection included in a
// chapter provenance package. Its hash, not a database path, identifies the
// tree that a trusted package builder may read.
type TextbookCodeArtifactExport struct {
	ID           uuid.UUID      `json:"id"`
	Kind         string         `json:"kind"`
	Language     string         `json:"language"`
	Entrypoint   string         `json:"entrypoint,omitempty"`
	ManifestJSON datatypes.JSON `json:"manifest"`
	ManifestHash string         `json:"manifest_hash"`
	ArtifactHash string         `json:"artifact_hash"`
	Limitations  datatypes.JSON `json:"limitations"`
	Status       string         `json:"status"`
}

// TextbookRuntimeEvidenceExport intentionally contains only reviewable
// execution metadata and structured output. Raw evidence references remain
// local service internals and are not written into a portable manuscript.
type TextbookRuntimeEvidenceExport struct {
	ID                  uuid.UUID  `json:"id"`
	CodeArtifactID      uuid.UUID  `json:"code_artifact_id"`
	CodeArtifactHash    string     `json:"code_artifact_hash"`
	InputHash           string     `json:"input_hash"`
	Kind                string     `json:"kind"`
	Status              string     `json:"status"`
	CommandManifestHash string     `json:"command_manifest_hash,omitempty"`
	RunnerImageDigest   string     `json:"runner_image_digest,omitempty"`
	ToolchainVersion    string     `json:"toolchain_version,omitempty"`
	ToolName            string     `json:"tool_name,omitempty"`
	ToolVersion         string     `json:"tool_version,omitempty"`
	StructuredOutput    string     `json:"structured_output,omitempty"`
	OutputTruncated     bool       `json:"output_truncated"`
	CapturedAt          *time.Time `json:"captured_at,omitempty"`
	ExpiresAt           *time.Time `json:"expires_at,omitempty"`
	StaleReason         string     `json:"stale_reason,omitempty"`
}

// TextbookManuscriptAssetExport preserves the provenance and rights decision
// beside the binary asset in the package, without exposing a storage path.
type TextbookManuscriptAssetExport struct {
	ID               uuid.UUID `json:"id"`
	EvidenceID       uuid.UUID `json:"evidence_id"`
	StableRef        string    `json:"stable_ref"`
	Kind             string    `json:"kind"`
	ContentHash      string    `json:"content_hash"`
	AltText          string    `json:"alt_text"`
	Source           string    `json:"source"`
	GenerationMethod string    `json:"generation_method"`
	VisualPurpose    string    `json:"visual_purpose"`
	RightsStatus     string    `json:"rights_status"`
	Status           string    `json:"status"`
}
