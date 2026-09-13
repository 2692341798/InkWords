package wiki

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildNoteSourceUsesMountedLocalVault(t *testing.T) {
	vaultRoot := t.TempDir()
	conceptsDir := filepath.Join(vaultRoot, "wiki", "concepts")
	require.NoError(t, os.MkdirAll(conceptsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(conceptsDir, "本地知识卡.md"), []byte(`---
type: concept
title: "本地知识卡"
review:
  enabled: true
---

`+strings.Repeat("这是一段可以独立复习并用于验证本地知识库读取的正文。", 12)), 0o600))

	t.Setenv(localVaultRootEnv, vaultRoot)
	t.Setenv("OBSIDIAN_REST_API_BASE_URL", "")
	t.Setenv("OBSIDIAN_REST_API_KEY", "")

	notes, err := BuildNoteSource("wiki").ListEligibleNotes(context.Background())
	require.NoError(t, err)
	require.Len(t, notes, 1)
	require.Equal(t, "wiki/concepts/本地知识卡.md", notes[0].NotePath)
}

func TestLocalNoteStoreRejectsPathsOutsideVault(t *testing.T) {
	vaultRoot := t.TempDir()
	externalRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(externalRoot, "outside.md"), []byte("outside"), 0o600))
	require.NoError(t, os.Symlink(filepath.Join(externalRoot, "outside.md"), filepath.Join(vaultRoot, "escaped.md")))
	store, err := newLocalNoteStore(vaultRoot)
	require.NoError(t, err)

	_, err = store.Read(context.Background(), "../outside.md")
	require.Error(t, err)

	_, err = store.Read(context.Background(), "escaped.md")
	require.Error(t, err)
}
