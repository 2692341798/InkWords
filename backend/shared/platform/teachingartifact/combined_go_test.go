package teachingartifact

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func combinedFixture(t *testing.T) ([]File, string) {
	t.Helper()
	root, data, _ := offlineFixture(t)
	var manifest GoDependencyManifest
	require.NoError(t, json.Unmarshal(data, &manifest))
	// A dependency larger than the legacy teaching budget must not consume or
	// enlarge the independently enforced budget for manuscript bytes.
	large := []byte("package library\n//" + strings.Repeat("x", int(MaxArtifactBytes)))
	require.NoError(t, os.WriteFile(filepath.Join(root, manifest.Files[2].Path), large, 0o600))
	manifest.Files[2].Bytes = int64(len(large))
	manifest.Files[2].SHA256 = fmt.Sprintf("sha256:%x", sha256.Sum256(large))
	data, err := json.Marshal(manifest)
	require.NoError(t, err)
	hash := dependencyDigest(data)
	bundle, err := LoadOfflineGoBundle(context.Background(), root, data, hash)
	require.NoError(t, err)
	files := append(bundle.Files(), File{Path: "main.go", Content: []byte("package main\nfunc main(){}\n")}, File{Path: "main_test.go", Content: []byte("package main\n")})
	return files, hash
}

func TestCombinedGoTreeUsesSeparateBudgetsAndPinnedSnapshots(t *testing.T) {
	files, dependencyHash := combinedFixture(t)
	store := NewStore(t.TempDir())
	_, err := store.Stage(context.Background(), files)
	require.Error(t, err, "ordinary staging cannot silently accept a dependency profile")
	staged, err := store.StageWithGoDependencies(context.Background(), files, dependencyHash)
	require.NoError(t, err)
	_, err = store.Resolve(staged.Token)
	require.Error(t, err, "legacy reads must not authorize a larger dependency tree")
	root, err := store.ResolveWithGoDependencies(staged.Token, dependencyHash)
	require.NoError(t, err)
	manifest := sharedtextbook.TeachingArtifactManifest{Format: sharedtextbook.TeachingArtifactManifestFormat, ArtifactID: "11111111-1111-1111-1111-111111111111", RevisionID: "22222222-2222-2222-2222-222222222222", ArtifactHash: staged.ArtifactHash, BookContractHash: "sha256:" + strings.Repeat("b", 64), StyleSheetHash: "sha256:" + strings.Repeat("c", 64), Language: "go", ToolchainVersion: "go1.26.9", DependencyManifestHash: dependencyHash, Commands: []sharedtextbook.VerificationCommand{{Kind: "go_test"}}}
	_, _, err = VerifiedManifestSnapshot(context.Background(), root, manifest)
	require.ErrorContains(t, err, "toolchain")
	manifest.ToolchainVersion = "go1.26.8"
	validated, release, err := VerifiedManifestSnapshot(context.Background(), root, manifest)
	require.NoError(t, err)
	require.DirExists(t, validated)
	release()
	snapshot, cleanup, err := VerifiedSnapshot(context.Background(), root, staged.ArtifactHash, dependencyHash)
	require.NoError(t, err)
	defer cleanup()
	require.Equal(t, staged.ArtifactHash, mustTreeHash(t, snapshot))
	require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte("mutated"), 0o600))
	require.Equal(t, staged.ArtifactHash, mustTreeHash(t, snapshot), "snapshot is detached from later source edits")
	_, _, err = VerifiedSnapshot(context.Background(), root, staged.ArtifactHash, dependencyHash)
	require.Error(t, err)
}

func TestCopyBoundedTreeDoesNotOverwriteExistingDestination(t *testing.T) {
	source, target := t.TempDir(), t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(source, "main.go"), []byte("package main"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(target, "main.go"), []byte("user content"), 0600))
	require.Error(t, CopyBoundedTree(context.Background(), source, target))
	data, err := os.ReadFile(filepath.Join(target, "main.go"))
	require.NoError(t, err)
	require.Equal(t, "user content", string(data))
}

func TestCombinedGoTreeRejectsWrongPinsExtraFilesAndTeachingOverflow(t *testing.T) {
	files, dependencyHash := combinedFixture(t)
	for _, mode := range []string{"wrong pin", "extra teaching", "oversized teaching", "changed dependency", "missing lock"} {
		t.Run(mode, func(t *testing.T) {
			input := append([]File(nil), files...)
			hash := dependencyHash
			switch mode {
			case "wrong pin":
				hash = "sha256:" + strings.Repeat("0", 64)
			case "extra teaching":
				input = append(input, File{Path: "other.go", Content: []byte("package main")})
			case "oversized teaching":
				input[len(input)-1].Content = []byte(strings.Repeat("x", int(MaxArtifactBytes)+1))
			case "changed dependency":
				input[0].Content = []byte("module changed")
			case "missing lock":
				for i, f := range input {
					if f.Path == "inkwords-dependencies.json" {
						input = append(input[:i], input[i+1:]...)
						break
					}
				}
			}
			store := NewStore(t.TempDir())
			_, err := store.StageWithGoDependencies(context.Background(), input, hash)
			require.Error(t, err)
		})
	}
}
