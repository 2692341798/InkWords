package textbookartifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// ErrDependencyConfirmationChanged requires another preview instead of rebasing a confirmation.
var ErrDependencyConfirmationChanged = errors.New("dependency projection confirmation changed")

type dependencyProjectionSource interface {
	generatedRevisionResolver
	dependencyCandidateResolver
}

type selectedDependencyCatalog interface {
	Resolve(context.Context, uuid.UUID, uuid.UUID, string) (*SelectedOfflineGoBundle, string, error)
}

// DependencyProjectionRequest identifies saved content and a configured option; it accepts no files or commands.
type DependencyProjectionRequest struct {
	OptionID    string    `json:"option_id"`
	TaskID      uuid.UUID `json:"task_id"`
	RevisionID  uuid.UUID `json:"revision_id"`
	ContentHash string    `json:"content_hash"`
}

// DependencyProjectionPreview is a bounded summary of the exact unexecuted projection.
type DependencyProjectionPreview struct {
	Contract         string                                     `json:"contract"`
	Request          DependencyProjectionRequest                `json:"request"`
	ArtifactID       uuid.UUID                                  `json:"artifact_id"`
	Selection        sharedtextbook.TeachingDependencySelection `json:"selection"`
	SelectionHash    string                                     `json:"selection_hash"`
	BookContractHash string                                     `json:"book_contract_hash"`
	StyleSheetHash   string                                     `json:"style_sheet_hash"`
	Commands         []sharedtextbook.VerificationCommand       `json:"commands"`
	FileCount        int                                        `json:"file_count"`
	TotalBytes       int64                                      `json:"total_bytes"`
	FilesHash        string                                     `json:"files_hash"`
	Language         string                                     `json:"language"`
	ToolchainVersion string                                     `json:"toolchain_version"`
	Entrypoint       string                                     `json:"entrypoint"`
	Limitations      []string                                   `json:"limitations"`
	ConfirmationHash string                                     `json:"confirmation_hash"`
}

// DependencyProjectionService reuses candidate provenance and immutable staging without invoking generation or execution.
type DependencyProjectionService struct {
	resolver dependencyProjectionSource
	catalog  selectedDependencyCatalog
	stager   artifactProjectionStager
}

// NewDependencyProjectionService wires operator-controlled dependencies to explicit preview/apply actions.
func NewDependencyProjectionService(resolver dependencyProjectionSource, catalog selectedDependencyCatalog, stager artifactProjectionStager) *DependencyProjectionService {
	return &DependencyProjectionService{resolver: resolver, catalog: catalog, stager: stager}
}

// Preview reads and verifies dependencies without staging them or changing authoritative state.
func (s *DependencyProjectionService) Preview(ctx context.Context, workspaceID, chapterID uuid.UUID, request DependencyProjectionRequest) (DependencyProjectionPreview, error) {
	preview, _, err := s.prepare(ctx, workspaceID, chapterID, request)
	return preview, err
}

// Apply recomputes a preview before staging; repeated identical applications use the existing immutable artifact identity.
func (s *DependencyProjectionService) Apply(ctx context.Context, workspaceID, chapterID uuid.UUID, request DependencyProjectionRequest, confirmed string) (*textbookdomain.CodeArtifactRow, error) {
	if confirmed == "" {
		return nil, ErrDependencyConfirmationChanged
	}
	preview, input, err := s.prepare(ctx, workspaceID, chapterID, request)
	if err != nil {
		return nil, err
	}
	if confirmed != preview.ConfirmationHash {
		return nil, ErrDependencyConfirmationChanged
	}
	if s.stager == nil {
		return nil, errors.New("dependency staging unavailable")
	}
	return s.stager.StageAndRegister(ctx, workspaceID, input)
}

func (s *DependencyProjectionService) prepare(ctx context.Context, workspaceID, chapterID uuid.UUID, request DependencyProjectionRequest) (DependencyProjectionPreview, Input, error) {
	fail := func(err error) (DependencyProjectionPreview, Input, error) {
		return DependencyProjectionPreview{}, Input{}, err
	}
	hash, err := hex.DecodeString(request.ContentHash)
	if s == nil || s.resolver == nil || s.catalog == nil || workspaceID == uuid.Nil || chapterID == uuid.Nil || request.TaskID == uuid.Nil || request.RevisionID == uuid.Nil || request.OptionID == "" || len(request.OptionID) > 128 || err != nil || len(hash) != sha256.Size {
		return fail(textbookdomain.ErrInvalidState)
	}
	generated, err := s.resolver.GetGeneratedRevisionContext(ctx, request.TaskID)
	if err != nil {
		return fail(err)
	}
	if generated == nil || generated.WorkspaceID != workspaceID || generated.Revision.ChapterID != chapterID || generated.Revision.ID != request.RevisionID || generated.Revision.ContentHash != request.ContentHash {
		return fail(textbookdomain.ErrInvalidState)
	}
	selected, selectionHash, err := s.catalog.Resolve(ctx, workspaceID, chapterID, request.OptionID)
	if err != nil {
		return fail(err)
	}
	input, err := selected.ManuscriptInput(ctx, s.resolver, generated)
	if err != nil {
		return fail(err)
	}
	// Include every file boundary and byte hash, not just names or a mutable catalog ID.
	type fileIdentity struct {
		Path  string
		Bytes int64
		Hash  string
	}
	files := make([]fileIdentity, 0, len(input.Files))
	var total int64
	for _, file := range input.Files {
		size := int64(len(file.Content))
		total += size
		files = append(files, fileIdentity{file.Path, size, fmt.Sprintf("sha256:%x", sha256.Sum256(file.Content))})
	}
	data, err := json.Marshal(files)
	if err != nil {
		return fail(err)
	}
	preview := DependencyProjectionPreview{Contract: "inkwords.dependency-projection.v1", Request: request, ArtifactID: input.ArtifactID, Selection: selected.Selection(), SelectionHash: selectionHash, BookContractHash: input.BookContractHash, StyleSheetHash: input.StyleSheetHash, Commands: input.Commands, FileCount: len(files), TotalBytes: total, FilesHash: fmt.Sprintf("sha256:%x", sha256.Sum256(data))}
	preview.Language, preview.ToolchainVersion, preview.Entrypoint = input.Language, input.ToolchainVersion, input.Entrypoint
	preview.Limitations = append([]string(nil), input.Limitations...)
	data, err = json.Marshal(preview)
	if err != nil {
		return fail(err)
	}
	preview.ConfirmationHash = fmt.Sprintf("sha256:%x", sha256.Sum256(data))
	return preview, input, nil
}
