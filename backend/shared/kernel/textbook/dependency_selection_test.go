package textbook

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTeachingDependencySelectionRequiresExplicitProjectSourceAndPin(t *testing.T) {
	s := TeachingDependencySelection{WorkspaceID: uuid.NewString(), ProjectID: uuid.NewString(), ChapterID: uuid.NewString(), Snapshot: SourceSnapshot{ID: uuid.NewString(), SourceID: uuid.NewString(), Kind: SourceKindGitRepository, Role: SourceRolePrimary, Locator: "https://github.com/gin-gonic/gin", ResolvedVersion: strings.Repeat("a", 40), ContentHash: "sha256:" + strings.Repeat("b", 64), CapturedAt: time.Now().UTC()}, Module: "github.com/gin-gonic/gin", Version: "v1.12.0", Toolchain: "go1.26.8", DependencyManifestHash: "sha256:" + strings.Repeat("c", 64)}
	require.NoError(t, s.Validate())
	s.BrowserObservation = &VerificationCommand{Kind: "browser_page", BrowserPath: "/orders", ExpectedText: "orders-list"}
	require.NoError(t, s.Validate())
	detached := s.Clone()
	detached.BrowserObservation.BrowserPath = "https://example.com"
	require.Error(t, detached.Validate())
	require.Equal(t, "/orders", s.BrowserObservation.BrowserPath)
	s.BrowserObservation = nil
	manifest := TeachingArtifactManifest{Format: TeachingArtifactManifestFormat, ArtifactID: uuid.NewString(), RevisionID: uuid.NewString(), ArtifactHash: "sha256:" + strings.Repeat("1", 64), BookContractHash: "sha256:" + strings.Repeat("2", 64), StyleSheetHash: "sha256:" + strings.Repeat("3", 64), Language: "go", ToolchainVersion: s.Toolchain, DependencyManifestHash: s.DependencyManifestHash, DependencySelection: &s, Commands: []VerificationCommand{{Kind: "go_test"}}}
	before, err := TeachingArtifactVerificationInputHash(manifest, "sha256:"+strings.Repeat("4", 64))
	require.NoError(t, err)
	s.Snapshot.ContentHash = "sha256:" + strings.Repeat("5", 64)
	after, err := TeachingArtifactVerificationInputHash(manifest, "sha256:"+strings.Repeat("4", 64))
	require.NoError(t, err)
	require.NotEqual(t, before, after, "source selection changes invalidate previous receipts")
	manifest.DependencyManifestHash = "sha256:" + strings.Repeat("6", 64)
	require.Error(t, manifest.Validate(), "selection cannot grant a different dependency inventory")
	for name, change := range map[string]func(*TeachingDependencySelection){
		"missing chapter":  func(s *TeachingDependencySelection) { s.ChapterID = "" },
		"other repository": func(s *TeachingDependencySelection) { s.Module = "github.com/other/gin" },
		"credentials": func(s *TeachingDependencySelection) {
			s.Snapshot.Locator = "https://user:pass@github.com/gin-gonic/gin"
		},
		"query":                      func(s *TeachingDependencySelection) { s.Snapshot.Locator += "?token=value" },
		"floating version":           func(s *TeachingDependencySelection) { s.Version = "latest" },
		"short hash":                 func(s *TeachingDependencySelection) { s.DependencyManifestHash = "sha256:a" },
		"official supporting source": func(s *TeachingDependencySelection) { s.Snapshot.Role = SourceRoleOfficial },
		"non exact toolchain":        func(s *TeachingDependencySelection) { s.Toolchain = "go1.26" },
	} {
		t.Run(name, func(t *testing.T) { bad := s; change(&bad); require.Error(t, bad.Validate()) })
	}
}
