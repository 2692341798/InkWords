package textbook

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTeachingArtifactVerificationInputHashInvalidatesForContractChanges(t *testing.T) {
	manifest := TeachingArtifactManifest{
		Format: TeachingArtifactManifestFormat, ArtifactID: uuid.NewString(), RevisionID: uuid.NewString(),
		ArtifactHash: "sha256:" + strings.Repeat("a", 64), BookContractHash: "sha256:" + strings.Repeat("b", 64), StyleSheetHash: "sha256:" + strings.Repeat("c", 64),
		Language: "go", ToolchainVersion: "go1.25.4", Commands: []VerificationCommand{{Kind: "go_test"}},
	}
	first, err := TeachingArtifactVerificationInputHash(manifest, "sha256:"+strings.Repeat("d", 64))
	require.NoError(t, err)

	changedBook := manifest
	changedBook.BookContractHash = "sha256:" + strings.Repeat("e", 64)
	bookHash, err := TeachingArtifactVerificationInputHash(changedBook, "sha256:"+strings.Repeat("d", 64))
	require.NoError(t, err)
	require.NotEqual(t, first, bookHash)

	changedStyle := manifest
	changedStyle.StyleSheetHash = "sha256:" + strings.Repeat("f", 64)
	styleHash, err := TeachingArtifactVerificationInputHash(changedStyle, "sha256:"+strings.Repeat("d", 64))
	require.NoError(t, err)
	require.NotEqual(t, first, styleHash)
	withDependencies := manifest
	withDependencies.DependencyManifestHash = "sha256:" + strings.Repeat("1", 64)
	dependencyHash, err := TeachingArtifactVerificationInputHash(withDependencies, "sha256:"+strings.Repeat("d", 64))
	require.NoError(t, err)
	require.NotEqual(t, first, dependencyHash)
	withDependencies.DependencyManifestHash = "not-a-digest"
	require.Error(t, withDependencies.Validate())
}

func TestLegacyTeachingManifestRetainsSerializedHash(t *testing.T) {
	// A serialized v1 fixture from before dependency support. Adding an empty
	// optional field must not make historical receipts stale.
	raw := []byte(fmt.Sprintf(`{"format":"inkwords.teaching-artifact.v1","artifact_id":"11111111-1111-1111-1111-111111111111","revision_id":"22222222-2222-2222-2222-222222222222","artifact_hash":"sha256:%s","book_contract_hash":"sha256:%s","style_sheet_hash":"sha256:%s","language":"go","toolchain_version":"go1.26.8","commands":[{"kind":"go_test"}]}`, strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64)))
	var manifest TeachingArtifactManifest
	require.NoError(t, json.Unmarshal(raw, &manifest))
	hash, err := TeachingArtifactManifestHash(manifest)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("sha256:%x", sha256.Sum256(raw)), hash)
}

func TestBrowserPageVerificationCommandAllowsOnlyBoundedLocalAssertions(t *testing.T) {
	command := VerificationCommand{Kind: "browser_page", BrowserPath: "/orders", ExpectedText: "订单路由已注册"}
	require.NoError(t, command.Validate())

	command.BrowserPath = "https://example.com/orders"
	require.ErrorContains(t, command.Validate(), "local browser path")
	command.BrowserPath = "/orders?next=http://example.com"
	require.ErrorContains(t, command.Validate(), "local browser path")
	command.BrowserPath = "/orders"
	command.ExpectedText = ""
	require.ErrorContains(t, command.Validate(), "expected text")
}
