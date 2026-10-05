package textbookartifact

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/teachingartifact"
)

type dependencySourceResolver interface {
	GetDependencySourceSnapshot(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) (sharedtextbook.SourceSnapshot, error)
}

type dependencyCandidateResolver interface {
	dependencySourceResolver
	ValidateDependencyCandidateSource(context.Context, textbookdomain.ChapterRevision, sharedtextbook.TeachingDependencySelection) error
}

// SelectedOfflineGoBundle is a checked operator selection. Its private state
// prevents callers from changing the chapter or source after validation.
type SelectedOfflineGoBundle struct {
	selection sharedtextbook.TeachingDependencySelection
	bundle    *teachingartifact.OfflineGoBundle
}

// Selection returns detached provenance for preview and audit output.
func (s *SelectedOfflineGoBundle) Selection() sharedtextbook.TeachingDependencySelection {
	return s.selection.Clone()
}

// ReadSelectedOfflineGoBundle reads bounded local operator files and verifies
// the selected source against the database. It never stages or runs anything.
func ReadSelectedOfflineGoBundle(ctx context.Context, resolver dependencySourceResolver, selectionPath, selectionHash, root, manifestPath string) (*SelectedOfflineGoBundle, error) {
	selection, err := readDependencySelection(selectionPath, selectionHash)
	if err != nil {
		return nil, err
	}
	if err := validateSelectionSource(ctx, resolver, selection); err != nil {
		return nil, err
	}
	manifest, err := readSelectionFile(manifestPath, 2<<20)
	if err != nil {
		return nil, fmt.Errorf("read dependency manifest: %w", err)
	}
	bundle, err := teachingartifact.LoadOfflineGoBundle(ctx, root, manifest, selection.DependencyManifestHash)
	if err != nil {
		return nil, err
	}
	if bundle.Toolchain() != selection.Toolchain || bundle.Origin() != (teachingartifact.GoDependencyOrigin{Module: selection.Module, Version: selection.Version, Commit: selection.Snapshot.ResolvedVersion}) {
		return nil, fmt.Errorf("dependency bundle differs from the selected source or toolchain")
	}
	return &SelectedOfflineGoBundle{selection: selection, bundle: bundle}, nil
}

func readDependencySelection(selectionPath, selectionHash string) (sharedtextbook.TeachingDependencySelection, error) {
	var selection sharedtextbook.TeachingDependencySelection
	data, err := readSelectionFile(selectionPath, 16<<10)
	if err != nil {
		return selection, fmt.Errorf("read dependency selection: %w", err)
	}
	if fmt.Sprintf("sha256:%x", sha256.Sum256(data)) != selectionHash {
		return selection, fmt.Errorf("dependency selection hash mismatch")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&selection); err != nil {
		return selection, fmt.Errorf("invalid dependency selection JSON")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return selection, fmt.Errorf("trailing dependency selection data")
	}
	if err := selection.Validate(); err != nil {
		return selection, err
	}
	return selection, nil
}

func validateSelectionSource(ctx context.Context, resolver dependencySourceResolver, selection sharedtextbook.TeachingDependencySelection) error {
	if resolver == nil {
		return fmt.Errorf("dependency source resolver unavailable")
	}
	actual, err := resolver.GetDependencySourceSnapshot(ctx, uuid.MustParse(selection.WorkspaceID), uuid.MustParse(selection.ProjectID), uuid.MustParse(selection.ChapterID), uuid.MustParse(selection.Snapshot.ID))
	if err != nil {
		return fmt.Errorf("selected dependency source unavailable: %w", err)
	}
	if !selection.MatchesSnapshot(actual) {
		return fmt.Errorf("selected dependency source changed")
	}
	return nil
}

// ManuscriptInput rechecks live source membership and candidate identity before
// producing the exact bytes to register. No fallback strips failed dependencies.
func (s *SelectedOfflineGoBundle) ManuscriptInput(ctx context.Context, resolver dependencyCandidateResolver, generated *textbookdomain.GeneratedRevisionContext) (Input, error) {
	if s == nil || generated == nil || generated.WorkspaceID.String() != s.selection.WorkspaceID || generated.Revision.ChapterID.String() != s.selection.ChapterID {
		return Input{}, fmt.Errorf("dependency selection does not match the candidate chapter")
	}
	if err := validateSelectionSource(ctx, resolver, s.selection); err != nil {
		return Input{}, err
	}
	if err := resolver.ValidateDependencyCandidateSource(ctx, generated.Revision, s.selection); err != nil {
		return Input{}, err
	}
	input, err := ManuscriptGoInputForOfflineBundle(generated.Revision, s.selection.Toolchain, s.bundle.Origin(), s.bundle)
	if err != nil {
		return Input{}, err
	}
	selection := s.selection.Clone()
	if selection.BrowserObservation != nil {
		input.Commands = append(input.Commands, *selection.BrowserObservation)
	}
	data, err := json.Marshal(selection)
	if err != nil {
		return Input{}, err
	}
	input.ArtifactID = uuid.NewSHA1(uuid.NameSpaceURL, []byte(fmt.Sprintf("inkwords:manuscript-go:v4:%s:%x", input.ArtifactID, sha256.Sum256(data))))
	input.dependencySelection = &selection
	input.BookContractHash, input.StyleSheetHash = generated.BookContractHash, generated.StyleSheetHash
	return input, nil
}

func readSelectionFile(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fmt.Errorf("expected a bounded regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, fmt.Errorf("operator file changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, fmt.Errorf("operator file exceeds read budget")
	}
	return data, nil
}
