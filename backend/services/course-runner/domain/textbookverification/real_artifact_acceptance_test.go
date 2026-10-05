package textbookverification

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"inkwords-backend/services/course-runner/domain/verification"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// TestRealFrozenTeachingArtifact is an operator-selected isolated acceptance
// run. Its evaluation manifest IDs are not database revisions or human approval.
// No network, persistence, provider, or production task client is constructed.
func TestRealFrozenTeachingArtifact(t *testing.T) {
	if os.Getenv("INKWORDS_REAL_TEACHING_VERIFICATION") != "approved" {
		t.Skip("real frozen teaching verification is opt-in")
	}
	require.Equal(t, "linux", runtime.GOOS)
	dir := os.Getenv("INKWORDS_FROZEN_TEACHING_DIR")
	require.True(t, filepath.IsAbs(dir))
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	require.NoError(t, err)
	require.LessOrEqual(t, len(data), 65536)
	var manifest ArtifactManifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	require.NoError(t, decoder.Decode(&manifest))
	require.Equal(t, io.EOF, decoder.Decode(new(any)))
	require.NoError(t, manifest.Validate())
	manifestHash, err := ManifestHash(manifest)
	require.NoError(t, err)
	require.Equal(t, os.Getenv("INKWORDS_EXPECTED_MANIFEST_HASH"), manifestHash)
	require.Equal(t, []CommandTemplate{{Kind: "go_test"}}, manifest.Commands)
	root := filepath.Join(dir, "artifact")
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	var names []string
	for _, entry := range entries {
		require.True(t, entry.Type().IsRegular())
		names = append(names, entry.Name())
	}
	require.Equal(t, []string{"go.mod", "main.go", "main_test.go"}, names)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	version := exec.CommandContext(ctx, "/usr/local/go/bin/go", "env", "GOVERSION")
	version.Dir, version.Env = "/tmp", []string{"GOTOOLCHAIN=local", "HOME=/tmp", "PATH=/usr/local/go/bin:/usr/bin:/bin"}
	actual, err := version.Output()
	require.NoError(t, err)
	require.Equal(t, manifest.ToolchainVersion, strings.TrimSpace(string(actual)))
	executor := verification.BubblewrapExecutor{Binary: "/usr/bin/bwrap"}
	runner := Runner{Executor: executor, RunnerImageDigest: os.Getenv("INKWORDS_RUNNER_IMAGE_DIGEST"), ToolchainVersion: strings.TrimSpace(string(actual)), Timeout: 30 * time.Second}
	report := Report{Status: sharedtextbook.ArtifactStatusUnverified}
	if err := executor.Preflight(ctx); err != nil {
		report.Reason = err.Error()
	} else {
		report = runner.Verify(ctx, RunRequest{Manifest: manifest, RootDir: root})
	}
	encoded, err := json.Marshal(struct {
		Origin             string `json:"origin"`
		CandidatePersisted bool   `json:"candidate_persisted"`
		ManifestHash       string `json:"manifest_hash"`
		Report             Report `json:"report"`
	}{Origin: "automated_frozen_artifact_acceptance", ManifestHash: manifestHash, Report: report})
	require.NoError(t, err)
	t.Logf("REAL_TEACHING_REPORT=%s", encoded)
	require.Equal(t, sharedtextbook.ArtifactStatusVerified, report.Status, report.Reason)
}
