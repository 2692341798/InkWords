package textbookartifact

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExactToolchainCreatesNewManifestIdentityWithoutChangingTeachingFiles(t *testing.T) {
	revision := manuscriptRevision(projectionManuscript("same-code"))
	legacy, err := ManuscriptGoInput(revision)
	require.NoError(t, err)
	current, err := ManuscriptGoInputForToolchain(revision, "go1.26.8")
	require.NoError(t, err)
	require.Equal(t, "go1.26.8", current.ToolchainVersion)
	require.NotEqual(t, legacy.ArtifactID, current.ArtifactID)
	require.Equal(t, legacy.Files[1:], current.Files[1:])
	require.Contains(t, string(current.Files[0].Content), "go 1.26.0")
	require.Contains(t, string(current.Files[0].Content), "toolchain go1.26.8")
	repeat, err := ManuscriptGoInputForToolchain(revision, "go1.26.8")
	require.NoError(t, err)
	require.Equal(t, current.ArtifactID, repeat.ArtifactID)
	newer, err := ManuscriptGoInputForToolchain(revision, "go1.26.9")
	require.NoError(t, err)
	require.NotEqual(t, current.ArtifactID, newer.ArtifactID)
	for _, invalid := range []string{"", "go1.26", "go1.25.8", "go1.26.8; echo unsafe"} {
		_, err := ManuscriptGoInputForToolchain(revision, invalid)
		require.Error(t, err)
	}
}
