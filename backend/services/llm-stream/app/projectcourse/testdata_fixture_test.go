package projectcourse

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInkWordsFixtureManifestIsMinimalAndNonExecutable(t *testing.T) {
	path := filepath.Join("testdata", "inkwords-fixture", "manifest.json")
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	var manifest struct {
		Entries []struct {
			Path string `json:"path"`
		} `json:"entries"`
		Constraints struct {
			SourceContentIncluded    bool `json:"source_content_included"`
			ExecutesTargetRepository bool `json:"executes_target_repository"`
			MaxEntryCount            int  `json:"max_entry_count"`
		} `json:"constraints"`
	}
	require.NoError(t, json.Unmarshal(content, &manifest))
	require.NotEmpty(t, manifest.Entries)
	require.LessOrEqual(t, len(manifest.Entries), manifest.Constraints.MaxEntryCount)
	require.False(t, manifest.Constraints.SourceContentIncluded)
	require.False(t, manifest.Constraints.ExecutesTargetRepository)
	for _, entry := range manifest.Entries {
		require.NotContains(t, entry.Path, "..")
		require.NotContains(t, entry.Path, "\\")
	}
}
