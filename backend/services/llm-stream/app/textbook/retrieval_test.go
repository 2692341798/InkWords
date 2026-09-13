package textbook

import (
	"context"
	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"testing"
	"time"
)

func TestLexicalRetrieverImplementsFutureRetrieverPort(t *testing.T) {
	var retriever Retriever = LexicalRetriever{}
	_, err := retriever.Retrieve(context.Background(), "路由 handle", []RetrievalCandidate{retrievalCandidate("primary", sharedtextbook.SourceRolePrimary, "路由 handle", []string{"路由"}), retrievalCandidate("official", sharedtextbook.SourceRoleOfficial, "路由 handle", []string{"路由"})}, 2)
	require.NoError(t, err)
}

func TestRetrieveExplainsPrimaryPriorityAndIsDeterministic(t *testing.T) {
	primary := retrievalCandidate("primary", sharedtextbook.SourceRolePrimary, "路由 handle 会合并 handlers", []string{"路由"})
	official := retrievalCandidate("official", sharedtextbook.SourceRoleOfficial, "路由 handle 会合并 handlers", []string{"路由"})
	plan, err := Retrieve("路由 handle", []RetrievalCandidate{official, primary}, 2)
	require.NoError(t, err)
	require.Equal(t, "primary-chunk", plan.Selected[0].Chunk.ID)
	require.Contains(t, plan.Selected[0].Reasons, "主资料优先")
	require.NotEmpty(t, plan.InputHash)
	pack, err := BuildEvidencePack(plan)
	require.NoError(t, err)
	require.NoError(t, pack.Validate())
	require.Len(t, pack.Evidence, 2)
}

func retrievalCandidate(prefix string, role sharedtextbook.SourceRole, text string, headings []string) RetrievalCandidate {
	snapshot := sharedtextbook.SourceSnapshot{ID: prefix + "-snapshot", SourceID: prefix + "-source", Kind: sharedtextbook.SourceKindMarkdown, Role: role, Locator: prefix + ".md", ResolvedVersion: "v1", ContentHash: "sha256:" + prefix, CapturedAt: time.Now()}
	document := sharedtextbook.SourceDocument{ID: prefix + "-document", SnapshotID: snapshot.ID, CanonicalLocator: prefix + ".md", Title: prefix, MediaType: "text/markdown", ContentHash: "sha256:" + prefix}
	chunk := sharedtextbook.SourceChunk{ID: prefix + "-chunk", DocumentID: document.ID, Ordinal: 1, HeadingPath: headings, Locator: sharedtextbook.EvidenceLocator{Path: prefix + ".md", StartLine: 1, EndLine: 1}, TextHash: "sha256:" + prefix, SearchText: text}
	return RetrievalCandidate{Snapshot: snapshot, Document: document, Chunk: chunk}
}
