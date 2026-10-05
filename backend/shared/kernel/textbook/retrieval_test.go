package textbook

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPrimarySourcePriorityCannotCreateAKeywordMatch(t *testing.T) {
	primary := RetrievalCandidate{Snapshot: SourceSnapshot{Role: SourceRolePrimary}, Chunk: SourceChunk{SearchText: "route registration"}}
	score, reasons := scoreRetrievalCandidate([]string{"albums"}, primary)
	require.Zero(t, score)
	require.Empty(t, reasons)
}

func TestRetrievalPrefersTheNamedDeclarationOverACallSite(t *testing.T) {
	declaration := RetrievalCandidate{Snapshot: SourceSnapshot{Role: SourceRolePrimary}, Chunk: SourceChunk{Locator: EvidenceLocator{Symbol: "(*node).getValue"}, SearchText: "func (n *node) getValue() {}"}}
	call := declaration
	call.Chunk.Locator.Symbol = "(*Engine).handleHTTPRequest"
	call.Chunk.SearchText = "value := root.getValue()"
	declScore, reasons := scoreRetrievalCandidate([]string{"getvalue"}, declaration)
	callScore, _ := scoreRetrievalCandidate([]string{"getvalue"}, call)
	require.Greater(t, declScore, callScore)
	require.Contains(t, reasons, "源码符号:getvalue")
}

func TestRetrieveEvidenceKeepsOfficialCPlusPlusSnapshotTraceable(t *testing.T) {
	primarySnapshot := SourceSnapshot{
		ID: "snapshot-project", SourceID: "source-project", Kind: SourceKindGitRepository, Role: SourceRolePrimary,
		Locator: "https://github.com/example/cpp-course", ResolvedVersion: "0123456789abcdef0123456789abcdef01234567", ContentHash: "sha256:project-snapshot", CapturedAt: time.Unix(1, 0).UTC(),
	}
	primaryDocument := SourceDocument{ID: "document-project", SnapshotID: primarySnapshot.ID, CanonicalLocator: "README.md", Title: "C++ 课程项目", MediaType: "text/markdown", ContentHash: "sha256:project-document", ArtifactPath: "README.md"}
	primaryCandidate := RetrievalCandidate{Snapshot: primarySnapshot, Document: primaryDocument, Chunk: SourceChunk{
		ID: "chunk-project-compiler", DocumentID: primaryDocument.ID, Ordinal: 1, Locator: EvidenceLocator{Path: "README.md", StartLine: 1, EndLine: 2}, TextHash: "sha256:project-chunk", SearchText: "This C++ course uses a compiler toolchain.",
	}}
	snapshot := SourceSnapshot{
		ID: "snapshot-cpp", SourceID: "source-cpp", Kind: SourceKindOfficialWeb, Role: SourceRoleOfficial,
		Locator: "https://isocpp.org/get-started", ResolvedVersion: "2026-09-03T00:00:00Z", ContentHash: "sha256:cpp-snapshot", CapturedAt: time.Unix(1, 0).UTC(),
	}
	document := SourceDocument{ID: "document-cpp-tour", SnapshotID: snapshot.ID, CanonicalLocator: "https://isocpp.org/tour", Title: "C++ 导览", MediaType: "text/html", ContentHash: "sha256:cpp-document", ArtifactPath: "tour.html"}
	candidate := RetrievalCandidate{Snapshot: snapshot, Document: document, Chunk: SourceChunk{
		ID: "chunk-cpp-compile", DocumentID: document.ID, Ordinal: 1, HeadingPath: []string{"开始使用 C++", "编译"}, Locator: EvidenceLocator{URL: document.CanonicalLocator, HeadingPath: []string{"开始使用 C++", "编译"}}, TextHash: "sha256:cpp-chunk", SearchText: "C++ compiler toolchain and a minimal program",
	}}

	first, err := RetrieveEvidence("c++ compiler", []RetrievalCandidate{candidate, primaryCandidate}, 2)
	require.NoError(t, err)
	second, err := RetrieveEvidence("c++ compiler", []RetrievalCandidate{candidate, primaryCandidate}, 2)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Len(t, first.Selected, 2)
	require.Contains(t, first.Selected[0].Reasons, "正文关键词:compiler")
	require.Contains(t, first.Selected[1].Reasons, "正文关键词:compiler")

	pack, err := BuildGenerationEvidencePack(first)
	require.NoError(t, err)
	require.Equal(t, snapshot.ID, pack.OfficialSources[0].ID)
	require.Equal(t, primarySnapshot.ID, pack.PrimarySnapshot.ID)
	require.Len(t, pack.Evidence, 2)
	require.Contains(t, pack.Excerpts, "evidence-chunk-cpp-compile")
}

func TestBuildGenerationEvidencePackKeepsMultipleFilesFromOneFixedPrimarySource(t *testing.T) {
	firstSnapshot := SourceSnapshot{ID: "snapshot-routergroup", SourceID: "source-gin", Kind: SourceKindGitRepository, Role: SourceRolePrimary, Locator: "https://github.com/gin-gonic/gin", ResolvedVersion: "73726dc606796a025971fe451f0aa6f1b9b847f6", ContentHash: "sha256:routergroup", CapturedAt: time.Unix(1, 0).UTC()}
	secondSnapshot := firstSnapshot
	secondSnapshot.ID = "snapshot-gin-go"
	secondSnapshot.ContentHash = "sha256:gin-go"
	makeCandidate := func(snapshot SourceSnapshot, documentID, path, chunkID, text string) RetrievalCandidate {
		document := SourceDocument{ID: documentID, SnapshotID: snapshot.ID, CanonicalLocator: path, Title: path, MediaType: "text/x-go", ContentHash: "sha256:" + documentID, ArtifactPath: path}
		return RetrievalCandidate{Snapshot: snapshot, Document: document, Chunk: SourceChunk{ID: chunkID, DocumentID: document.ID, Ordinal: 1, Locator: EvidenceLocator{Path: path, StartLine: 1, EndLine: 3}, TextHash: "sha256:" + chunkID, SearchText: text}}
	}
	plan, err := RetrieveEvidence("路由 请求", []RetrievalCandidate{
		makeCandidate(firstSnapshot, "routergroup-doc", "routergroup.go", "register", "路由 register 请求"),
		makeCandidate(secondSnapshot, "gin-doc", "gin.go", "lookup", "路由 lookup 请求"),
	}, 2)
	require.NoError(t, err)

	pack, err := BuildGenerationEvidencePack(plan)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{firstSnapshot.ID, secondSnapshot.ID}, []string{pack.PrimarySnapshot.ID, pack.PrimarySnapshots[0].ID})
	require.Len(t, pack.Evidence, 2)
	require.NotEqual(t, pack.PrimarySnapshot.ContentHash, GenerationPrimarySnapshotHash(pack))

	changedSource := pack
	changedSource.PrimarySnapshots[0].SourceID = "other-source"
	require.ErrorContains(t, changedSource.Validate(), "same fixed primary source version")
}
