package textbookverification

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type snapshotObserver func(string)

func (observe snapshotObserver) Execute(_ context.Context, root, _ string, _ time.Duration, _ []string) (int, string, error) {
	observe(root)
	return 0, "fixture executor; no teaching process launched", nil
}

func TestDependencyProfileRejectsDifferentRunnerBeforeExecution(t *testing.T) {
	root := teachingArtifact(t)
	hash, err := ArtifactTreeHash(root)
	require.NoError(t, err)
	manifest := validManifest(hash)
	manifest.DependencyManifestHash = digest('e')
	executor := &fakeExecutor{}
	report := (Runner{Executor: executor, RunnerImageDigest: digest('b'), ToolchainVersion: "go1.26.8"}).Verify(context.Background(), RunRequest{Manifest: manifest, RootDir: root})
	require.Equal(t, "unverified", string(report.Status))
	require.Contains(t, report.Reason, "exact runner toolchain")
	require.Empty(t, executor.commands)
}

func TestRunnerCommandsUsePrivateSnapshotAndCleanItUp(t *testing.T) {
	root := teachingArtifact(t)
	original, err := os.ReadFile(filepath.Join(root, "main.go"))
	require.NoError(t, err)
	hash, err := ArtifactTreeHash(root)
	require.NoError(t, err)
	manifest := validManifest(hash)
	var seen string
	executor := snapshotObserver(func(snapshot string) {
		require.NotEqual(t, root, snapshot)
		seen = snapshot
		require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte("changed after validation"), 0600))
		current, err := os.ReadFile(filepath.Join(snapshot, "main.go"))
		require.NoError(t, err)
		require.Equal(t, original, current)
	})
	report := (Runner{Executor: executor, RunnerImageDigest: digest('b'), ToolchainVersion: "go1.26.8"}).Verify(context.Background(), RunRequest{Manifest: manifest, RootDir: root})
	require.Equal(t, "verified", string(report.Status), "fixture executor result only")
	require.NotEmpty(t, seen)
	require.NoDirExists(t, seen)
}
