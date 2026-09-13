package textbookartifact

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/teachingartifact"
)

func projectionDependencyBundle(t *testing.T, value string) *teachingartifact.OfflineGoBundle {
	t.Helper()
	root := t.TempDir()
	origin := teachingartifact.GoDependencyOrigin{Module: "example.com/library", Version: "v1.0.0", Commit: strings.Repeat("a", 40)}
	sum := "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	manifest := teachingartifact.GoDependencyManifest{Format: teachingartifact.GoDependencyManifestFormat, Origin: origin, Toolchain: "go1.26.8", Modules: []teachingartifact.GoDependencyModule{{Path: origin.Module, Version: origin.Version, Sum: sum, GoModSum: sum}}}
	for _, file := range []teachingartifact.File{
		{Path: "go.mod", Content: []byte("module example.com/inkwords/teaching\n\ngo 1.26.0\n\ntoolchain go1.26.8\n\nrequire example.com/library v1.0.0\n")},
		{Path: "go.sum", Content: []byte("example.com/library v1.0.0 " + sum + "\nexample.com/library v1.0.0/go.mod " + sum + "\n")},
		{Path: "vendor/modules.txt", Content: []byte("# example.com/library v1.0.0\n## explicit; go 1.26.0\nexample.com/library\n")},
		{Path: "vendor/example.com/library/library.go", Content: []byte("package library\n// " + value + "\n")},
	} {
		p := filepath.Join(root, file.Path)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o700))
		require.NoError(t, os.WriteFile(p, file.Content, 0o600))
		manifest.Files = append(manifest.Files, teachingartifact.GoDependencyFile{Path: file.Path, Bytes: int64(len(file.Content)), SHA256: fmt.Sprintf("sha256:%x", sha256.Sum256(file.Content))})
	}
	data, err := json.Marshal(manifest)
	require.NoError(t, err)
	bundle, err := teachingartifact.LoadOfflineGoBundle(context.Background(), root, data, fmt.Sprintf("sha256:%x", sha256.Sum256(data)))
	require.NoError(t, err)
	return bundle
}

func TestOfflineStageAndRegisterPinsTheDependencyManifest(t *testing.T) {
	bundle := projectionDependencyBundle(t, "pinned")
	input, err := ManuscriptGoInputForOfflineBundle(manuscriptRevision(projectionManuscript("same-code")), "go1.26.8", bundle.Origin(), bundle)
	require.NoError(t, err)
	input.BookContractHash = "sha256:" + strings.Repeat("b", 64)
	input.StyleSheetHash = "sha256:" + strings.Repeat("c", 64)
	registrar := &recordingRegistrar{}
	store := teachingartifact.NewStore(t.TempDir())
	_, err = NewService(store, registrar).StageAndRegister(context.Background(), uuid.New(), input)
	require.NoError(t, err)
	var manifest sharedtextbook.TeachingArtifactManifest
	require.NoError(t, json.Unmarshal(registrar.input.ManifestJSON, &manifest))
	require.Equal(t, bundle.Hash(), manifest.DependencyManifestHash)
	_, err = store.ResolveWithGoDependencies(manifest.ArtifactHash, manifest.DependencyManifestHash)
	require.NoError(t, err)
	_, err = store.Resolve(manifest.ArtifactHash)
	require.Error(t, err)
	input.ToolchainVersion = "go1.26.9"
	_, err = NewService(store, registrar).StageAndRegister(context.Background(), uuid.New(), input)
	require.ErrorContains(t, err, "toolchain")
}

func TestOfflineDependencyProjectionPreservesManuscriptAndBindsEveryInput(t *testing.T) {
	revision := manuscriptRevision(projectionManuscript("unchanged"))
	bundle := projectionDependencyBundle(t, "original dependency")
	input, err := ManuscriptGoInputForOfflineBundle(revision, "go1.26.8", bundle.Origin(), bundle)
	require.NoError(t, err)
	legacy, err := ManuscriptGoInputForToolchain(revision, "go1.26.8")
	require.NoError(t, err)
	require.Equal(t, legacy.Files[1:], input.Files[:2])
	require.NotEqual(t, legacy.ArtifactID, input.ArtifactID)
	require.Equal(t, legacy.Commands, input.Commands)
	require.Equal(t, legacy.Limitations, input.Limitations)
	again, err := ManuscriptGoInputForOfflineBundle(revision, "go1.26.8", bundle.Origin(), bundle)
	require.NoError(t, err)
	require.Equal(t, input, again)
	changed := projectionDependencyBundle(t, "new dependency")
	next, err := ManuscriptGoInputForOfflineBundle(revision, "go1.26.8", changed.Origin(), changed)
	require.NoError(t, err)
	require.NotEqual(t, input.ArtifactID, next.ArtifactID)
	store := teachingartifact.NewStore(t.TempDir())
	firstTree, err := store.StageWithGoDependencies(context.Background(), input.Files, bundle.Hash())
	require.NoError(t, err)
	nextTree, err := store.StageWithGoDependencies(context.Background(), next.Files, changed.Hash())
	require.NoError(t, err)
	require.NotEqual(t, firstTree.ArtifactHash, nextTree.ArtifactHash)
	wrong := bundle.Origin()
	wrong.Commit = strings.Repeat("b", 40)
	_, err = ManuscriptGoInputForOfflineBundle(revision, "go1.26.8", wrong, bundle)
	require.Error(t, err)
	_, err = ManuscriptGoInputForOfflineBundle(revision, "go1.26.9", bundle.Origin(), bundle)
	require.Error(t, err)
	_, err = ManuscriptGoInputForOfflineBundle(revision, "go1.26.8", bundle.Origin(), nil)
	require.Error(t, err)
	revision.Markdown += "tampered"
	_, err = ManuscriptGoInputForOfflineBundle(revision, "go1.26.8", bundle.Origin(), bundle)
	require.Error(t, err)
}
