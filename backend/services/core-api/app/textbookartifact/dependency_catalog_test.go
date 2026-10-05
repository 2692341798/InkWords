package textbookartifact

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func TestLocalDependencyCatalogScopesOptionsAndRevalidatesFiles(t *testing.T) {
	bundle := projectionDependencyBundle(t, "catalog")
	dir, root := t.TempDir(), t.TempDir()
	for _, file := range bundle.Files() {
		p := filepath.Join(root, file.Path)
		if file.Path == "inkwords-dependencies.json" {
			p = filepath.Join(dir, "manifest.json")
		}
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0700))
		require.NoError(t, os.WriteFile(p, file.Content, 0600))
	}
	w, ch := uuid.New(), uuid.New()
	selection := sharedtextbook.TeachingDependencySelection{WorkspaceID: w.String(), ProjectID: uuid.NewString(), ChapterID: ch.String(), Snapshot: sharedtextbook.SourceSnapshot{ID: uuid.NewString(), SourceID: uuid.NewString(), Kind: sharedtextbook.SourceKindGitRepository, Role: sharedtextbook.SourceRolePrimary, Locator: "https://example.com/library", ResolvedVersion: bundle.Origin().Commit, ContentHash: "sha256:" + strings.Repeat("a", 64), CapturedAt: time.Now().UTC()}, Module: bundle.Origin().Module, Version: bundle.Origin().Version, Toolchain: bundle.Toolchain(), DependencyManifestHash: bundle.Hash()}
	data, err := json.Marshal(selection)
	require.NoError(t, err)
	selectionPath := filepath.Join(dir, "selection.json")
	require.NoError(t, os.WriteFile(selectionPath, data, 0600))
	entry := DependencyCatalogEntry{ID: "prepared", Title: "已准备的固定依赖", SelectionPath: selectionPath, SelectionHash: fmt.Sprintf("sha256:%x", sha256.Sum256(data)), Root: root, ManifestPath: filepath.Join(dir, "manifest.json")}
	catalogPath := filepath.Join(dir, "catalog.json")
	writeCatalog := func(entries []DependencyCatalogEntry) {
		data, err := json.Marshal(map[string]any{"contract": "inkwords.dependency-catalog.v1", "entries": entries})
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(catalogPath, data, 0600))
	}
	writeCatalog([]DependencyCatalogEntry{entry})
	resolver := &selectionResolver{snapshot: selection.Snapshot}
	catalog := NewLocalDependencyCatalog(catalogPath, resolver)
	options, err := catalog.Options(t.Context(), w, ch)
	require.NoError(t, err)
	require.Len(t, options, 1)
	data, err = json.Marshal(options)
	require.NoError(t, err)
	require.NotContains(t, string(data), dir)
	require.NotContains(t, string(data), root)
	options, err = catalog.Options(t.Context(), uuid.New(), ch)
	require.NoError(t, err)
	require.Empty(t, options)
	_, _, err = catalog.Resolve(t.Context(), w, uuid.New(), entry.ID)
	require.Error(t, err)
	_, _, err = catalog.Resolve(t.Context(), w, ch, root)
	require.Error(t, err)
	writeCatalog([]DependencyCatalogEntry{entry, entry})
	_, err = catalog.Options(t.Context(), w, ch)
	require.Error(t, err)
	writeCatalog([]DependencyCatalogEntry{entry})
	require.NoError(t, os.WriteFile(filepath.Join(root, "vendor/example.com/library/library.go"), []byte("changed"), 0600))
	_, _, err = catalog.Resolve(t.Context(), w, ch, entry.ID)
	require.Error(t, err)
	require.NoError(t, os.WriteFile(selectionPath, []byte(`{}`), 0600))
	_, err = catalog.Options(t.Context(), w, ch)
	require.ErrorContains(t, err, "hash mismatch")
	require.NoError(t, os.WriteFile(catalogPath, []byte(`{"contract":"inkwords.dependency-catalog.v1","entries":[],"command":"sh"}`), 0600))
	_, err = catalog.Options(t.Context(), w, ch)
	require.Error(t, err)
	options, err = NewLocalDependencyCatalog("", resolver).Options(t.Context(), w, ch)
	require.NoError(t, err)
	require.Empty(t, options)
}
