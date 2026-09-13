package textbook

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) CreateProject(ctx context.Context, input CreateProjectInput) (*Project, error) {
	if input.WorkspaceID == uuid.Nil || strings.TrimSpace(input.Title) == "" || input.Audience.Validate() != nil || input.Primary.Kind.Validate() != nil || strings.TrimSpace(input.Primary.Locator) == "" {
		return nil, fmt.Errorf("%w: invalid project input", ErrInvalidState)
	}
	input.Primary.Role = sharedtextbook.SourceRolePrimary
	if input.Primary.LicenseStatus == "" {
		input.Primary.LicenseStatus = "pending"
	}
	return s.repository.CreateProject(ctx, input)
}

func (s *Service) GetProject(ctx context.Context, workspaceID, projectID uuid.UUID) (*Project, error) {
	if workspaceID == uuid.Nil || projectID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid project identity", ErrInvalidState)
	}
	return s.repository.GetProject(ctx, workspaceID, projectID)
}

// GetProjectWorkspace loads the bounded editable project overview for its owner workspace.
func (s *Service) GetProjectWorkspace(ctx context.Context, workspaceID, projectID uuid.UUID) (*ProjectWorkspace, error) {
	if workspaceID == uuid.Nil || projectID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid project identity", ErrInvalidState)
	}
	return s.repository.GetProjectWorkspace(ctx, workspaceID, projectID)
}

// GetProjectProgress returns backend-derived production stages for one workspace-owned project.
func (s *Service) GetProjectProgress(ctx context.Context, workspaceID, projectID uuid.UUID) (*ProjectProgress, error) {
	if workspaceID == uuid.Nil || projectID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid project identity", ErrInvalidState)
	}
	return s.repository.GetProjectProgress(ctx, workspaceID, projectID)
}

// ListSourceLibrary reads bounded source metadata owned by the active local workspace.
func (s *Service) ListSourceLibrary(ctx context.Context, workspaceID, projectID uuid.UUID) ([]SourceLibraryDocument, error) {
	if workspaceID == uuid.Nil || projectID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid project identity", ErrInvalidState)
	}
	return s.repository.ListSourceLibrary(ctx, workspaceID, projectID)
}

// ListSourceEvidence returns human-selectable evidence identifiers for a workspace-owned project.
func (s *Service) ListSourceEvidence(ctx context.Context, workspaceID, projectID uuid.UUID, ids ...string) ([]SourceLibraryEvidence, error) {
	if workspaceID == uuid.Nil || projectID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid project identity", ErrInvalidState)
	}
	if err := validateSourceEvidenceIDs(ids); err != nil {
		return nil, err
	}
	return s.repository.ListSourceEvidence(ctx, workspaceID, projectID, ids...)
}

// RetrieveSourceEvidence returns a persisted, deterministic source-selection
// plan for author review. It never sends source excerpts to the browser.
func (s *Service) RetrieveSourceEvidence(ctx context.Context, workspaceID uuid.UUID, input RetrieveSourceInput) (*SourceRetrievalPlan, error) {
	if workspaceID == uuid.Nil || input.ProjectID == uuid.Nil || len(strings.TrimSpace(input.Query)) < 2 || len(input.Query) > 1024 || input.Limit < 1 || input.Limit > 20 {
		return nil, fmt.Errorf("%w: invalid source retrieval request", ErrInvalidState)
	}
	return s.repository.RetrieveSourceEvidence(ctx, workspaceID, input)
}

// GetChapterWorkspace returns revision history only after checking local workspace ownership.
func (s *Service) GetChapterWorkspace(ctx context.Context, workspaceID, chapterID uuid.UUID) (*ChapterWorkspace, error) {
	if workspaceID == uuid.Nil || chapterID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid chapter identity", ErrInvalidState)
	}
	return s.repository.GetChapterWorkspace(ctx, workspaceID, chapterID)
}

// GetApprovedRevisionProjections returns disposable views from the chapter's
// approved revision. The underlying repository state remains unchanged even
// when a malformed document cannot be projected.
func (s *Service) GetApprovedRevisionProjections(ctx context.Context, workspaceID, chapterID uuid.UUID) (ApprovedRevisionProjections, error) {
	return s.GetRevisionProjections(ctx, workspaceID, chapterID, uuid.Nil)
}

// GetRevisionProjections can read a historical approved revision so saved
// learning evidence never changes when a newer manuscript is approved.
func (s *Service) GetRevisionProjections(ctx context.Context, workspaceID, chapterID, revisionID uuid.UUID) (ApprovedRevisionProjections, error) {
	workspace, err := s.GetChapterWorkspace(ctx, workspaceID, chapterID)
	if err != nil {
		return ApprovedRevisionProjections{}, err
	}
	if workspace == nil || workspace.Chapter == nil {
		return ApprovedRevisionProjections{}, ErrInvalidState
	}
	if revisionID == uuid.Nil {
		if workspace.Chapter.ApprovedRevisionID == nil {
			return ApprovedRevisionProjections{}, ErrInvalidState
		}
		revisionID = *workspace.Chapter.ApprovedRevisionID
	}
	for _, revision := range workspace.Revisions {
		if revision.ID == revisionID {
			projection, err := BuildApprovedRevisionProjections(*workspace.Chapter, revision)
			projection.WorkspaceID = workspaceID
			return projection, err
		}
	}
	return ApprovedRevisionProjections{}, ErrInvalidState
}

func (s *Service) AddSource(ctx context.Context, workspaceID, projectID uuid.UUID, input CreateSourceInput) (*Source, error) {
	if workspaceID == uuid.Nil || projectID == uuid.Nil || input.Kind.Validate() != nil || input.Role.Validate() != nil || strings.TrimSpace(input.Locator) == "" || (input.Role == sharedtextbook.SourceRoleOfficial && !input.OfficialConfirmed) {
		return nil, fmt.Errorf("%w: invalid source input", ErrInvalidState)
	}
	if input.LicenseStatus == "" {
		input.LicenseStatus = "pending"
	}
	return s.repository.AddSource(ctx, workspaceID, projectID, input)
}

func (s *Service) CreateChapter(ctx context.Context, workspaceID uuid.UUID, input CreateChapterInput) (*Chapter, error) {
	if workspaceID == uuid.Nil || input.ProjectID == uuid.Nil || input.SortOrder < 1 || strings.TrimSpace(input.Title) == "" || input.ChapterProfile.Validate() != nil {
		return nil, fmt.Errorf("%w: invalid chapter input", ErrInvalidState)
	}
	return s.repository.CreateChapter(ctx, workspaceID, input)
}

// CreateBookBuild captures the approved manuscript set as an immutable review
// artifact. It never accepts a caller-selected revision or publication status.
func (s *Service) CreateBookBuild(ctx context.Context, workspaceID, projectID uuid.UUID, notices ...sharedtextbook.PublicationNoticeDraft) (*BookBuildRow, error) {
	if workspaceID == uuid.Nil || projectID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid book build identity", ErrInvalidState)
	}
	return s.repository.CreateBookBuild(ctx, workspaceID, CreateBookBuildInput{ProjectID: projectID, Notices: notices})
}

// GetEditorialWorkspace returns persisted rights and human-review evidence plus
// fail-closed automated checks for one workspace-owned frozen build.
func (s *Service) GetEditorialWorkspace(ctx context.Context, workspaceID, buildID uuid.UUID) (*EditorialWorkspace, error) {
	if workspaceID == uuid.Nil || buildID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid editorial workspace identity", ErrInvalidState)
	}
	return s.repository.GetEditorialWorkspace(ctx, workspaceID, buildID)
}

// AddRightsItem appends exact, build-specific rights evidence. Existing rows
// cannot be edited after a human review has relied on them.
func (s *Service) AddRightsItem(ctx context.Context, workspaceID uuid.UUID, input AddRightsItemInput) (*RightsItemRow, error) {
	input = trimEditorialInput(input)
	if workspaceID == uuid.Nil || input.BuildID == uuid.Nil || input.WorkType.Validate() != nil || input.PublicationStatus.Validate() != nil || input.SubjectRef == "" || input.RightsBasis == "" || input.AllowedUse == "" || input.Attribution == "" {
		return nil, fmt.Errorf("%w: invalid publication rights item", ErrInvalidState)
	}
	return s.repository.AddRightsItem(ctx, workspaceID, input)
}

// CompletePublicationReview records an explicit human statement and decision.
// The repository assigns its receipt time; identity and stage are never inferred.
func (s *Service) CompletePublicationReview(ctx context.Context, workspaceID uuid.UUID, input CompletePublicationReviewInput) (*PublicationReviewRow, error) {
	input.Reviewer = strings.TrimSpace(input.Reviewer)
	input.Notes = strings.TrimSpace(input.Notes)
	if workspaceID == uuid.Nil || input.validate() != nil {
		return nil, fmt.Errorf("%w: invalid publication review", ErrInvalidState)
	}
	return s.repository.CompletePublicationReview(ctx, workspaceID, input)
}

// PromoteBookBuild is the only transition to publication_candidate. It remains
// an internal preflight state and does not claim publisher or regulator approval.
func (s *Service) PromoteBookBuild(ctx context.Context, workspaceID, buildID uuid.UUID) (*BookBuildRow, error) {
	if workspaceID == uuid.Nil || buildID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid book build promotion", ErrInvalidState)
	}
	return s.repository.PromoteBookBuild(ctx, workspaceID, buildID)
}

// CreateBookContract creates an immutable draft of the project's teaching promise.
func (s *Service) CreateBookContract(ctx context.Context, workspaceID uuid.UUID, input CreateBookContractInput) (*BookContractRevision, error) {
	if workspaceID == uuid.Nil || input.ProjectID == uuid.Nil || input.Reader.Validate() != nil || strings.TrimSpace(input.Promise) == "" || len(input.ChapterProfiles) == 0 || strings.TrimSpace(input.TerminologyVersion) == "" || strings.TrimSpace(input.PublicationProfile) == "" {
		return nil, fmt.Errorf("%w: invalid book contract", ErrInvalidState)
	}
	for _, profile := range input.ChapterProfiles {
		if profile.Validate() != nil {
			return nil, fmt.Errorf("%w: invalid book contract profile", ErrInvalidState)
		}
	}
	return s.repository.CreateBookContract(ctx, workspaceID, input)
}

// CreateStyleSheet creates an immutable draft of the project's writing rules.
func (s *Service) CreateStyleSheet(ctx context.Context, workspaceID uuid.UUID, input CreateStyleSheetInput) (*StyleSheetRevision, error) {
	if workspaceID == uuid.Nil || input.ProjectID == uuid.Nil || strings.TrimSpace(input.Language) == "" || len(input.TerminologyRules) == 0 || len(input.CodeRules) == 0 || len(input.CitationRules) == 0 {
		return nil, fmt.Errorf("%w: invalid style sheet", ErrInvalidState)
	}
	return s.repository.CreateStyleSheet(ctx, workspaceID, input)
}

// CreateBlueprint creates an immutable draft teaching outline for the currently approved contracts.
func (s *Service) CreateBlueprint(ctx context.Context, workspaceID uuid.UUID, input CreateBlueprintInput) (*BlueprintRevision, error) {
	if workspaceID == uuid.Nil || input.ProjectID == uuid.Nil || len(input.Volumes) == 0 {
		return nil, fmt.Errorf("%w: invalid blueprint", ErrInvalidState)
	}
	blueprint := sharedtextbook.Blueprint{RevisionID: "validation", ProjectID: "validation", RevisionNumber: 1, ContentHash: "sha256:validation", BookContractRevision: "validation", StyleSheetRevision: "validation", Volumes: input.Volumes}
	if err := blueprint.Validate(); err != nil {
		return nil, fmt.Errorf("%w: invalid blueprint: %w", ErrInvalidState, err)
	}
	return s.repository.CreateBlueprint(ctx, workspaceID, input)
}

// ApproveBookContract selects one reviewed book contract for future generation.
func (s *Service) ApproveBookContract(ctx context.Context, workspaceID, projectID, revisionID uuid.UUID) (*BookContractRevision, error) {
	if workspaceID == uuid.Nil || projectID == uuid.Nil || revisionID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid book contract approval", ErrInvalidState)
	}
	return s.repository.ApproveBookContract(ctx, workspaceID, projectID, revisionID)
}

// ApproveStyleSheet selects one reviewed style sheet for future generation.
func (s *Service) ApproveStyleSheet(ctx context.Context, workspaceID, projectID, revisionID uuid.UUID) (*StyleSheetRevision, error) {
	if workspaceID == uuid.Nil || projectID == uuid.Nil || revisionID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid style sheet approval", ErrInvalidState)
	}
	return s.repository.ApproveStyleSheet(ctx, workspaceID, projectID, revisionID)
}

// ApproveBlueprint selects one reviewed outline that is bound to the current generation contracts.
func (s *Service) ApproveBlueprint(ctx context.Context, workspaceID, projectID, revisionID uuid.UUID) (*BlueprintRevision, error) {
	if workspaceID == uuid.Nil || projectID == uuid.Nil || revisionID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid blueprint approval", ErrInvalidState)
	}
	return s.repository.ApproveBlueprint(ctx, workspaceID, projectID, revisionID)
}

// PrepareSampleGeneration freezes the approved contracts and explicitly mapped evidence for one chapter.
// Publishing the resulting payload is an application-layer concern, so this method never enqueues work.
func (s *Service) PrepareSampleGeneration(ctx context.Context, workspaceID, projectID, chapterID uuid.UUID, target sharedtextbook.SampleGenerationTarget) (sharedtextbook.SampleGenerationTaskPayload, error) {
	if workspaceID == uuid.Nil || projectID == uuid.Nil || chapterID == uuid.Nil {
		return sharedtextbook.SampleGenerationTaskPayload{}, fmt.Errorf("%w: invalid sample generation request", ErrInvalidState)
	}
	if err := target.Validate(); err != nil {
		return sharedtextbook.SampleGenerationTaskPayload{}, fmt.Errorf("%w: %w", ErrInvalidState, err)
	}
	return s.repository.PrepareSampleGeneration(ctx, workspaceID, projectID, chapterID, target)
}

// LoadGinFixture makes the fixed, offline first-slice evidence available for a Gin project.
func (s *Service) LoadGinFixture(ctx context.Context, workspaceID, projectID uuid.UUID) (*SourceSnapshot, error) {
	if workspaceID == uuid.Nil || projectID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid Gin fixture request", ErrInvalidState)
	}
	return s.repository.LoadGinFixture(ctx, workspaceID, projectID)
}

// PrepareSourceImport freezes a source identity before raw bytes are queued to
// parser-service. It does not persist any documents; only a validated worker
// result may create the immutable source snapshot.
func (s *Service) PrepareSourceImport(ctx context.Context, workspaceID uuid.UUID, input PrepareSourceImportInput) (sharedtextbook.SourceImportTaskPayload, error) {
	if workspaceID == uuid.Nil || input.ProjectID == uuid.Nil || input.SourceID == uuid.Nil || input.SnapshotID == uuid.Nil || strings.TrimSpace(input.Filename) == "" || !isDigest(input.ContentHash) || input.ByteSize < 1 {
		return sharedtextbook.SourceImportTaskPayload{}, fmt.Errorf("%w: invalid source import request", ErrInvalidState)
	}
	return s.repository.PrepareSourceImport(ctx, workspaceID, input)
}

// PrepareOfficialWebImport freezes a confirmed official-document boundary.
// No snapshot is persisted until the parser worker returns complete, validated
// source structures.
func (s *Service) PrepareOfficialWebImport(ctx context.Context, workspaceID uuid.UUID, input PrepareOfficialWebImportInput) (sharedtextbook.OfficialWebImportTaskPayload, error) {
	if workspaceID == uuid.Nil || input.ProjectID == uuid.Nil || input.SourceID == uuid.Nil || input.SnapshotID == uuid.Nil || len(input.AllowedPathPrefixes) == 0 {
		return sharedtextbook.OfficialWebImportTaskPayload{}, fmt.Errorf("%w: invalid official web import request", ErrInvalidState)
	}
	return s.repository.PrepareOfficialWebImport(ctx, workspaceID, input)
}

func (s *Service) AppendRevision(ctx context.Context, workspaceID uuid.UUID, input AppendRevisionInput) (*ChapterRevision, error) {
	if workspaceID == uuid.Nil || input.ChapterID == uuid.Nil || input.ExpectedVersion < 0 || input.Kind.Validate() != nil || len(input.ContentHash) != 64 || input.CreatedBy != RevisionCreatorManual && input.CreatedBy != RevisionCreatorGeneration || len(input.DocumentJSON) == 0 {
		return nil, fmt.Errorf("%w: invalid chapter revision input", ErrInvalidState)
	}
	if input.CreatedBy == RevisionCreatorGeneration {
		if input.Kind != sharedtextbook.RevisionKindCandidate {
			return nil, fmt.Errorf("%w: generated revisions must be candidates", ErrInvalidState)
		}
		if input.BookContractRevisionID == nil || input.StyleSheetRevisionID == nil || input.BlueprintRevisionID == nil || !isDigest(input.EvidencePackHash) || !isDigest(input.PromptHash) || strings.TrimSpace(input.ProviderName) == "" || strings.TrimSpace(input.ModelName) == "" || !json.Valid(input.ProviderUsageJSON) || !json.Valid(input.QualityReportJSON) {
			return nil, fmt.Errorf("%w: generated candidates require complete provenance", ErrInvalidState)
		}
	}
	if input.Kind == sharedtextbook.RevisionKindApproved && (input.LockOwnerID == uuid.Nil || input.LockVersion < 1) {
		return nil, fmt.Errorf("%w: applying an approved revision requires the current chapter lock", ErrRevisionLocked)
	}
	return s.repository.AppendRevision(ctx, workspaceID, input)
}

// RegisterGeneratedCodeArtifact records only a generated candidate's teaching
// implementation. Verification starts from an unverified artifact so callers
// cannot directly register a passing result.
func (s *Service) RegisterGeneratedCodeArtifact(ctx context.Context, workspaceID uuid.UUID, input RegisterGeneratedCodeArtifactInput) (*CodeArtifactRow, error) {
	if workspaceID == uuid.Nil || input.RevisionID == uuid.Nil || !json.Valid(input.ManifestJSON) || input.Artifact.Kind != sharedtextbook.CodeArtifactTeachingImplementation || input.Artifact.Status != sharedtextbook.ArtifactStatusUnverified || input.Artifact.Validate() != nil {
		return nil, fmt.Errorf("%w: invalid generated code artifact", ErrInvalidState)
	}
	if input.Artifact.RevisionID != input.RevisionID.String() {
		return nil, fmt.Errorf("%w: artifact revision mismatch", ErrInvalidState)
	}
	var manifest sharedtextbook.TeachingArtifactManifest
	if err := json.Unmarshal(input.ManifestJSON, &manifest); err != nil || manifest.Validate() != nil || manifest.RevisionID != input.RevisionID.String() || manifest.ArtifactID != input.Artifact.ID || manifest.ArtifactHash != input.Artifact.ArtifactHash {
		return nil, fmt.Errorf("%w: invalid generated artifact manifest", ErrInvalidState)
	}
	manifestHash, err := sharedtextbook.TeachingArtifactManifestHash(manifest)
	if err != nil || manifestHash != input.Artifact.ManifestHash {
		return nil, fmt.Errorf("%w: generated artifact manifest hash mismatch", ErrInvalidState)
	}
	return s.repository.RegisterGeneratedCodeArtifact(ctx, workspaceID, input)
}

func (s *Service) RegisterManuscriptAsset(ctx context.Context, workspaceID uuid.UUID, input RegisterManuscriptAssetInput) (*ManuscriptAssetRow, error) {
	if workspaceID == uuid.Nil || input.ChapterID == uuid.Nil || input.Asset.Validate() != nil {
		return nil, fmt.Errorf("%w: invalid manuscript asset", ErrInvalidState)
	}
	return s.repository.RegisterManuscriptAsset(ctx, workspaceID, input)
}

func isDigest(value string) bool {
	return strings.HasPrefix(value, "sha256:") && len(strings.TrimPrefix(value, "sha256:")) > 0
}

func (s *Service) AcquireLock(ctx context.Context, workspaceID uuid.UUID, input LockInput) (*ChapterLock, error) {
	if workspaceID == uuid.Nil || input.ChapterID == uuid.Nil || input.OwnerID == uuid.Nil || input.ExpectedVersion < 0 || input.LeaseDuration <= 0 || input.LeaseDuration > 15*time.Minute {
		return nil, fmt.Errorf("%w: invalid chapter lock input", ErrInvalidState)
	}
	return s.repository.AcquireLock(ctx, workspaceID, input)
}

func (s *Service) ApplyCandidate(ctx context.Context, workspaceID uuid.UUID, input ApplyCandidateInput) (*ChapterRevision, error) {
	input.ReviewNote = strings.TrimSpace(input.ReviewNote)
	review := sharedtextbook.NewSampleReview(input.DimensionScores, input.ReviewerKind, input.DelegationNote)
	if workspaceID == uuid.Nil || input.ChapterID == uuid.Nil || input.CandidateRevisionID == uuid.Nil || input.ExpectedVersion < 0 || input.LockOwnerID == uuid.Nil || input.LockVersion < 1 || len([]rune(input.ReviewNote)) < sharedtextbook.SampleHumanReviewNoteMinRunes || len([]rune(input.ReviewNote)) > sharedtextbook.SampleHumanReviewNoteMaxRunes || review.Validate() != nil {
		return nil, fmt.Errorf("%w: invalid candidate application", ErrInvalidState)
	}
	return s.repository.ApplyCandidate(ctx, workspaceID, input)
}

// RejectCandidate records a human rationale against the exact immutable
// candidate under the same short edit lease used by apply. It never mutates the
// manuscript head or deletes the generated revision.
func (s *Service) RejectCandidate(ctx context.Context, workspaceID uuid.UUID, input RejectCandidateInput) (*CandidateReview, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	reasonLength := len([]rune(input.Reason))
	if workspaceID == uuid.Nil || input.ChapterID == uuid.Nil || input.CandidateRevisionID == uuid.Nil || input.ExpectedVersion < 0 || input.LockOwnerID == uuid.Nil || input.LockVersion < 1 || reasonLength < 8 || reasonLength > 2000 {
		return nil, fmt.Errorf("%w: invalid candidate rejection", ErrInvalidState)
	}
	return s.repository.RejectCandidate(ctx, workspaceID, input)
}

// PersistDocuments accepts only internally parsed, immutable snapshot material.
func (s *Service) PersistDocuments(ctx context.Context, workspaceID uuid.UUID, input PersistDocumentsInput) error {
	if workspaceID == uuid.Nil || input.SnapshotID == uuid.Nil || len(input.Documents) == 0 || len(input.Chunks) == 0 {
		return fmt.Errorf("%w: invalid structured document batch", ErrInvalidState)
	}
	documents := make(map[string]struct{}, len(input.Documents))
	for _, document := range input.Documents {
		if document.SnapshotID != input.SnapshotID.String() || document.Validate() != nil {
			return fmt.Errorf("%w: invalid source document", ErrInvalidState)
		}
		documents[document.ID] = struct{}{}
	}
	for _, chunk := range input.Chunks {
		if _, exists := documents[chunk.DocumentID]; !exists || chunk.Validate() != nil {
			return fmt.Errorf("%w: invalid source chunk", ErrInvalidState)
		}
	}
	return s.repository.PersistDocuments(ctx, workspaceID, input)
}
