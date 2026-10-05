package textbook

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOfficialWebImportTaskResultRequiresFrozenBoundaryAndCompleteEvidence(t *testing.T) {
	payload := OfficialWebImportTaskPayload{
		TaskVersion: 1, TaskSubtype: TextbookOfficialWebImportTaskSubtype,
		ProjectID: "project-1", SourceID: "source-1", SnapshotID: "snapshot-1",
		SourceKind: SourceKindOfficialWeb, SourceRole: SourceRoleOfficial,
		EntryURL: "https://gin-gonic.com/en/docs/", AllowedPathPrefixes: []string{"/en/docs"},
	}
	payload.InputHash = OfficialWebImportInputHash(payload.ProjectID, payload.SourceID, payload.SnapshotID, payload.EntryURL, payload.AllowedPathPrefixes)
	require.NoError(t, payload.Validate())

	result := OfficialWebImportTaskResult{
		ResultVersion: 1, TaskSubtype: TextbookOfficialWebImportTaskSubtype,
		ProjectID: payload.ProjectID, SourceID: payload.SourceID, SnapshotID: payload.SnapshotID, InputHash: payload.InputHash,
		ContentHash: "sha256:manifest", ResolvedVersion: "sha256:manifest", CapturedAt: time.Now().UTC(),
		Documents: []SourceDocument{{ID: "doc-1", SnapshotID: payload.SnapshotID, CanonicalLocator: "https://gin-gonic.com/en/docs/", Title: "Gin documentation", MediaType: "text/html", ContentHash: "sha256:document"}},
		Chunks:    []SourceChunk{{ID: "chunk-1", DocumentID: "doc-1", Ordinal: 1, Locator: EvidenceLocator{URL: "https://gin-gonic.com/en/docs/", HeadingPath: []string{"Documentation"}}, TextHash: "sha256:chunk", SearchText: "Start here"}},
	}
	require.NoError(t, result.ValidateAgainst(payload))

	result.ResolvedVersion = "latest"
	require.Error(t, result.ValidateAgainst(payload), "a changing upstream must not masquerade as the captured manifest")
}

func TestOfficialWebImportInputHashSortsPathBoundary(t *testing.T) {
	first := OfficialWebImportInputHash("project", "source", "snapshot", "https://go.dev/", []string{"/doc", "/learn"})
	second := OfficialWebImportInputHash("project", "source", "snapshot", "https://go.dev/", []string{"/learn", "/doc"})
	require.Equal(t, first, second)
}

func TestOfficialWebParserVersionPreservesV2AndSeparatesV3Identity(t *testing.T) {
	payload := OfficialWebImportTaskPayload{TaskVersion: 2, ParserVersion: OfficialWebVisibleContentParserVersion, TaskSubtype: TextbookOfficialWebImportTaskSubtype, ProjectID: "project", SourceID: "source", SnapshotID: "snapshot", SourceKind: SourceKindOfficialWeb, SourceRole: SourceRoleOfficial, EntryURL: "https://go.dev/doc/install", AllowedPathPrefixes: []string{"/doc/install"}}
	hash := func(p OfficialWebImportTaskPayload) string {
		return OfficialWebImportInputHash(p.ProjectID, p.SourceID, p.SnapshotID, p.EntryURL, p.AllowedPathPrefixes, p.ParserVersion)
	}
	payload.InputHash = hash(payload)
	require.NoError(t, payload.Validate())
	oldHash := payload.InputHash
	payload.ParserVersion = OfficialWebParserVersion
	require.Error(t, payload.Validate(), "new parser cannot borrow an old task identity")
	payload.InputHash = hash(payload)
	require.NoError(t, payload.Validate())
	require.NotEqual(t, oldHash, payload.InputHash)
	payload.ParserVersion = "inkwords.official-html.future"
	payload.InputHash = hash(payload)
	require.Error(t, payload.Validate())
}
