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
)

func offlineFixture(t *testing.T) (string, []byte, string) {
	t.Helper()
	root := t.TempDir()
	files := []File{
		{Path: "go.mod", Content: []byte("module example.com/inkwords/teaching\n\ngo 1.26.0\n\ntoolchain go1.26.8\n\nrequire example.com/library v1.0.0\n")},
		{Path: "go.sum", Content: []byte("example.com/library v1.0.0 h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\nexample.com/library v1.0.0/go.mod h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\n")},
		{Path: "vendor/example.com/library/library.go", Content: []byte("package library\n")},
		{Path: "vendor/modules.txt", Content: []byte("# example.com/library v1.0.0\n## explicit; go 1.26.0\nexample.com/library\n")},
	}
	manifest := GoDependencyManifest{Format: GoDependencyManifestFormat, Origin: GoDependencyOrigin{Module: "example.com/library", Version: "v1.0.0", Commit: strings.Repeat("a", 40)}, Toolchain: "go1.26.8", Modules: []GoDependencyModule{{Path: "example.com/library", Version: "v1.0.0", Sum: "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=", GoModSum: "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}}}
	for _, file := range files {
		path := filepath.Join(root, filepath.FromSlash(file.Path))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, file.Content, 0o600))
		manifest.Files = append(manifest.Files, GoDependencyFile{Path: file.Path, Bytes: int64(len(file.Content)), SHA256: fmt.Sprintf("sha256:%x", sha256.Sum256(file.Content))})
	}
	data, err := json.Marshal(manifest)
	require.NoError(t, err)
	return root, data, fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}

func TestOfflineGoBundleVerifiesPinnedBytesAndDoesNotShareMutableFiles(t *testing.T) {
	root, data, hash := offlineFixture(t)
	bundle, err := LoadOfflineGoBundle(context.Background(), root, data, hash)
	require.NoError(t, err)
	require.Equal(t, hash, bundle.Hash())
	require.Equal(t, "go1.26.8", bundle.Toolchain())
	files := bundle.Files()
	require.Len(t, files, 5, "the lock is preserved alongside dependency bytes")
	files[0].Content[0] = '!'
	require.NotEqual(t, files[0].Content, bundle.Files()[0].Content)
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("tampered"), 0o600))
	_, err = LoadOfflineGoBundle(context.Background(), root, data, hash)
	require.Error(t, err)
	require.NotContains(t, string(bundle.Files()[0].Content), "tampered")
}

func TestOfflineGoBundleRejectsUnpinnedUnsafeOrInconsistentInputs(t *testing.T) {
	for _, mode := range []string{"wrong hash", "extra file", "symlink", "missing", "replacement", "vendor version", "wrong sum", "duplicate", "oversized", "unknown field", "trailing JSON", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			root, data, hash := offlineFixture(t)
			var manifest GoDependencyManifest
			require.NoError(t, json.Unmarshal(data, &manifest))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch mode {
			case "wrong hash":
				hash = "sha256:" + strings.Repeat("0", 64)
			case "extra file":
				require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0o600))
			case "symlink":
				require.NoError(t, os.Symlink("go.mod", filepath.Join(root, "link")))
			case "missing":
				require.NoError(t, os.Remove(filepath.Join(root, "vendor", "modules.txt")))
			case "replacement", "vendor version", "wrong sum":
				index, text := 0, "\nreplace example.com/library => /host/private\n"
				if mode == "vendor version" {
					index, text = 3, "# example.com/extra v2.0.0\n"
				}
				if mode == "wrong sum" {
					index, text = 1, "example.com/library v1.0.0 h1:BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBA=\n"
				}
				path := filepath.Join(root, manifest.Files[index].Path)
				original, err := os.ReadFile(path)
				require.NoError(t, err)
				changed := append(original, []byte(text)...)
				require.NoError(t, os.WriteFile(path, changed, 0o600))
				manifest.Files[index].Bytes = int64(len(changed))
				manifest.Files[index].SHA256 = fmt.Sprintf("sha256:%x", sha256.Sum256(changed))
			case "duplicate":
				manifest.Files = append(manifest.Files, manifest.Files[0])
			case "oversized":
				manifest.Files[0].Bytes = MaxGoDependencyBytes + 1
			case "cancelled":
				cancel()
			}
			if mode != "wrong hash" {
				var err error
				data, err = json.Marshal(manifest)
				require.NoError(t, err)
				if mode == "unknown field" {
					data = append(data[:len(data)-1], []byte(`,"shell":"unexpected"}`)...)
				}
				if mode == "trailing JSON" {
					data = append(data, []byte(`{}`)...)
				}
				hash = fmt.Sprintf("sha256:%x", sha256.Sum256(data))
			}
			_, err := LoadOfflineGoBundle(ctx, root, data, hash)
			require.Error(t, err)
		})
	}
}
