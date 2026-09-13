package textbook

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSourceImportParserVersionCreatesANewIdentityWithoutChangingLegacyTasks(t *testing.T) {
	legacyCanonical := "project_id=project\nsource_id=source\nfilename=tree.go\ncontent_hash=sha256:content\nresolved_version=" + strings.Repeat("a", 40)
	legacyHash := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(legacyCanonical)))
	currentHash := SourceImportInputHash("project", "source", "new-snapshot", "tree.go", "sha256:content", strings.Repeat("a", 40))
	require.NotEqual(t, legacyHash, currentHash)
	require.Equal(t, currentHash, SourceImportInputHash("project", "source", "other-snapshot", "tree.go", "sha256:content", strings.Repeat("a", 40)))
	for _, version := range []int{2, SourceImportTaskVersion} {
		payload := SourceImportTaskPayload{TaskVersion: version, TaskSubtype: TextbookSourceImportTaskSubtype, ProjectID: "project", SourceID: "source", SnapshotID: "snapshot", SourceKind: SourceKindGitRepository, SourceRole: SourceRolePrimary, Locator: "https://github.com/gin-gonic/gin", Filename: "tree.go", ContentHash: "sha256:content", ArtifactToken: "sha256:content", ResolvedVersion: strings.Repeat("a", 40), ByteSize: 10, InputHash: legacyHash}
		if version == SourceImportTaskVersion {
			payload.InputHash = currentHash
		}
		require.NoError(t, payload.Validate())
		payload.TaskVersion = 100
		require.Error(t, payload.Validate())
	}
}

func TestSourceImportTaskBindsWorkerOutputToOneFrozenSource(t *testing.T) {
	payload := SourceImportTaskPayload{TaskVersion: 2, TaskSubtype: TextbookSourceImportTaskSubtype, ProjectID: "project-1", SourceID: "source-1", SnapshotID: "snapshot-1", SourceKind: SourceKindMarkdown, SourceRole: SourceRolePrimary, Locator: "file:///intro.md", Filename: "intro.md", ArtifactToken: "sha256:content", ContentHash: "sha256:content", ByteSize: 5}
	payload.InputHash = SourceImportInputHash(payload.ProjectID, payload.SourceID, payload.SnapshotID, payload.Filename, payload.ContentHash)
	document := SourceDocument{ID: "document-1", SnapshotID: payload.SnapshotID, CanonicalLocator: payload.Locator, Title: "入门", MediaType: "text/markdown", ContentHash: "sha256:document"}
	chunk := SourceChunk{ID: "chunk-1", DocumentID: document.ID, Ordinal: 1, Locator: EvidenceLocator{Path: "intro.md", StartLine: 1, EndLine: 1}, TextHash: "sha256:chunk", SearchText: "开始"}
	result := SourceImportTaskResult{ResultVersion: 1, TaskSubtype: TextbookSourceImportTaskSubtype, ProjectID: payload.ProjectID, SourceID: payload.SourceID, SnapshotID: payload.SnapshotID, InputHash: payload.InputHash, Documents: []SourceDocument{document}, Chunks: []SourceChunk{chunk}}
	require.NoError(t, result.ValidateAgainst(payload))
	result.SourceID = "other-source"
	require.Error(t, result.ValidateAgainst(payload))
	require.Equal(t, payload.InputHash, SourceImportInputHash(payload.ProjectID, payload.SourceID, "another-snapshot", payload.Filename, payload.ContentHash), "retries of the same source bytes must reuse one task")
}

func TestGitSourceImportRequiresAnImmutableCommitAndBindsItToTheTaskHash(t *testing.T) {
	commit := strings.Repeat("a", 40)
	payload := SourceImportTaskPayload{TaskVersion: 2, TaskSubtype: TextbookSourceImportTaskSubtype, ProjectID: "project-1", SourceID: "source-1", SnapshotID: "snapshot-1", SourceKind: SourceKindGitRepository, SourceRole: SourceRolePrimary, Locator: "https://github.com/gin-gonic/gin", Filename: "routergroup.go", ArtifactToken: "sha256:content", ContentHash: "sha256:content", ResolvedVersion: commit, ByteSize: 5}
	payload.InputHash = SourceImportInputHash(payload.ProjectID, payload.SourceID, payload.SnapshotID, payload.Filename, payload.ContentHash, payload.ResolvedVersion)
	require.NoError(t, payload.Validate())
	require.NotEqual(t, payload.InputHash, SourceImportInputHash(payload.ProjectID, payload.SourceID, payload.SnapshotID, payload.Filename, payload.ContentHash, strings.Repeat("b", 40)))

	payload.ResolvedVersion = "main"
	require.ErrorContains(t, payload.Validate(), "commit SHA")
}
