package textbookartifact

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/teachingartifact"
)

type selectionResolver struct {
	snapshot        sharedtextbook.SourceSnapshot
	err             error
	candidateErr    error
	candidateChecks int
}

func (r *selectionResolver) GetDependencySourceSnapshot(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) (sharedtextbook.SourceSnapshot, error) {
	return r.snapshot, r.err
}
func (r *selectionResolver) ValidateDependencyCandidateSource(context.Context, textbookdomain.ChapterRevision, sharedtextbook.TeachingDependencySelection) error {
	r.candidateChecks++
	return r.candidateErr
}

func TestSelectedDependenciesBindChapterSourceAndPersistedManifest(t *testing.T) {
	ctx := context.Background()
	bundle := projectionDependencyBundle(t, "selection test")
	dir, root := t.TempDir(), t.TempDir()
	for _, f := range bundle.Files() {
		p := filepath.Join(root, f.Path)
		if f.Path == "inkwords-dependencies.json" {
			p = filepath.Join(dir, "manifest.json")
		}
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o700))
		require.NoError(t, os.WriteFile(p, f.Content, 0o600))
	}
	s := sharedtextbook.TeachingDependencySelection{WorkspaceID: uuid.NewString(), ProjectID: uuid.NewString(), ChapterID: uuid.NewString(), Snapshot: sharedtextbook.SourceSnapshot{ID: uuid.NewString(), SourceID: uuid.NewString(), Kind: sharedtextbook.SourceKindGitRepository, Role: sharedtextbook.SourceRolePrimary, Locator: "https://example.com/library", ResolvedVersion: bundle.Origin().Commit, ContentHash: "sha256:" + strings.Repeat("d", 64), CapturedAt: time.Now().UTC()}, Module: bundle.Origin().Module, Version: bundle.Origin().Version, Toolchain: bundle.Toolchain(), DependencyManifestHash: bundle.Hash()}
	s.BrowserObservation = &sharedtextbook.VerificationCommand{Kind: "browser_page", BrowserPath: "/orders", ExpectedText: "orders-list"}
	data, err := json.Marshal(s)
	require.NoError(t, err)
	selectionPath := filepath.Join(dir, "selection.json")
	require.NoError(t, os.WriteFile(selectionPath, data, 0o600))
	hash := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
	resolver := &selectionResolver{snapshot: s.Snapshot}
	load := func() (*SelectedOfflineGoBundle, error) {
		return ReadSelectedOfflineGoBundle(ctx, resolver, selectionPath, hash, root, filepath.Join(dir, "manifest.json"))
	}
	selected, err := load()
	require.NoError(t, err)
	detached := selected.Selection()
	detached.BrowserObservation.ExpectedText = "changed after preview"
	require.Equal(t, "orders-list", selected.Selection().BrowserObservation.ExpectedText)
	generated := &textbookdomain.GeneratedRevisionContext{WorkspaceID: uuid.MustParse(s.WorkspaceID), Revision: manuscriptRevision(projectionManuscript("selected-source")), BookContractHash: "sha256:" + strings.Repeat("b", 64), StyleSheetHash: "sha256:" + strings.Repeat("c", 64)}
	generated.Revision.ChapterID = uuid.MustParse(s.ChapterID)
	input, err := selected.ManuscriptInput(ctx, resolver, generated)
	require.NoError(t, err)
	require.Equal(t, 1, resolver.candidateChecks)
	registrar := &recordingRegistrar{}
	store := teachingartifact.NewStore(t.TempDir())
	_, err = NewService(store, registrar).StageAndRegister(ctx, generated.WorkspaceID, input)
	require.NoError(t, err)
	var manifest sharedtextbook.TeachingArtifactManifest
	require.NoError(t, json.Unmarshal(registrar.input.ManifestJSON, &manifest))
	require.Equal(t, s, *manifest.DependencySelection)
	require.NoError(t, manifest.Validate())
	require.Len(t, manifest.Commands, 2)
	require.Equal(t, *s.BrowserObservation, manifest.Commands[1])
	missing := manifest
	missing.Commands = missing.Commands[:1]
	require.ErrorContains(t, missing.Validate(), "missing")
	resolved, err := store.ResolveWithGoDependencies(manifest.ArtifactHash, bundle.Hash())
	require.NoError(t, err)
	_, cleanup, err := teachingartifact.VerifiedManifestSnapshot(ctx, resolved, manifest)
	require.NoError(t, err)
	cleanup()
	manifest.DependencySelection.Version = "v9.0.0"
	_, _, err = teachingartifact.VerifiedManifestSnapshot(ctx, resolved, manifest)
	require.ErrorContains(t, err, "origin")

	generated.WorkspaceID = uuid.New()
	_, err = selected.ManuscriptInput(ctx, resolver, generated)
	require.ErrorContains(t, err, "chapter")
	generated.WorkspaceID = uuid.MustParse(s.WorkspaceID)
	resolver.candidateErr = errors.New("snapshot absent from frozen candidate evidence")
	_, err = selected.ManuscriptInput(ctx, resolver, generated)
	require.ErrorContains(t, err, "frozen candidate")
	resolver.candidateErr = nil
	resolver.snapshot.ContentHash = "sha256:" + strings.Repeat("e", 64)
	_, err = selected.ManuscriptInput(ctx, resolver, generated)
	require.ErrorContains(t, err, "changed")
	_, err = load()
	require.ErrorContains(t, err, "changed")
	resolver.snapshot = s.Snapshot
	resolver.err = errors.New("foreign project")
	_, err = load()
	require.ErrorContains(t, err, "unavailable")
	resolver.err = nil
	require.NoError(t, os.WriteFile(selectionPath, append(data, '\n'), 0o600))
	_, err = load()
	require.ErrorContains(t, err, "hash mismatch")
}
