package export

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	shared "inkwords-backend/shared/kernel/textbook"
)

func TestExportDependencyPinRequiresTheFrozenExecutionManifest(t *testing.T) {
	m := shared.TeachingArtifactManifest{Format: shared.TeachingArtifactManifestFormat, ArtifactID: uuid.NewString(), RevisionID: uuid.NewString(), ArtifactHash: "sha256:" + strings.Repeat("a", 64), BookContractHash: "sha256:" + strings.Repeat("b", 64), StyleSheetHash: "sha256:" + strings.Repeat("c", 64), Language: "go", ToolchainVersion: "go1.26.8", DependencyManifestHash: "sha256:" + strings.Repeat("d", 64), Commands: []shared.VerificationCommand{{Kind: "go_test"}}}
	raw, err := json.Marshal(m)
	require.NoError(t, err)
	hash, err := shared.TeachingArtifactManifestHash(m)
	require.NoError(t, err)
	pin, err := artifactDependencyPin(raw, m.ArtifactID, m.ArtifactHash, hash)
	require.NoError(t, err)
	require.Equal(t, m.DependencyManifestHash, pin)
	_, err = artifactDependencyPin(raw, uuid.NewString(), m.ArtifactHash, hash)
	require.Error(t, err)
	_, err = artifactDependencyPin(raw, m.ArtifactID, m.ArtifactHash, "sha256:"+strings.Repeat("e", 64))
	require.Error(t, err)
	pin, err = artifactDependencyPin([]byte(`{}`), m.ArtifactID, m.ArtifactHash, hash)
	require.NoError(t, err)
	require.Empty(t, pin, "legacy records do not grant the dependency budget")
}
