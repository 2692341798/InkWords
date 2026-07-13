package projectcourse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

func testSnapshot() sharedkernel.SourceSnapshot {
	return sharedkernel.SourceSnapshot{RepositoryURL: "https://github.com/example/repo", RequestedRef: "main", ResolvedCommitSHA: "0123456789abcdef0123456789abcdef01234567", CapturedAt: time.Unix(1, 0).UTC()}
}

func TestBuildEvidencePackUsesStableSnapshotAndRanges(t *testing.T) {
	snapshot := testSnapshot()
	files, err := BuildInventory([]InventoryInput{{Path: "main.go", Content: []byte("package main\n\nfunc main() {}\n")}}, InventoryOptions{MaxContentBytes: 1000})
	require.NoError(t, err)
	graph := KnowledgeGraph{CommitSHA: snapshot.ResolvedCommitSHA, Files: files, Symbols: []SymbolRecord{{Path: "main.go", Name: "main", Kind: SymbolFunction, StartLine: 3, EndLine: 3}}}
	id := evidenceIDsForFiles(files)[0]
	pack, err := BuildEvidencePack(snapshot, "chapter-1", []string{id}, graph, nil)
	require.NoError(t, err)
	require.Equal(t, snapshot.ResolvedCommitSHA, pack.SourceEvidence[0].CommitSHA)
	require.Equal(t, 3, pack.SourceEvidence[0].StartLine)
	require.Equal(t, 3, pack.SourceEvidence[0].EndLine)
}

func TestBuildEvidencePackRejectsMissingOrExcludedEvidence(t *testing.T) {
	snapshot := testSnapshot()
	graph := KnowledgeGraph{CommitSHA: snapshot.ResolvedCommitSHA, Files: []InventoryEntry{{Path: "secret.bin", Role: RoleBinary, Disposition: DispositionExcluded, ContentHash: "sha256:x"}}}
	_, err := BuildEvidencePack(snapshot, "chapter-1", []string{"evidence-missing"}, graph, nil)
	require.ErrorContains(t, err, "not present")
	_, err = BuildEvidencePack(snapshot, "chapter-1", []string{"evidence-" + stableID("secret.bin\x00sha256:x")}, graph, nil)
	require.ErrorContains(t, err, "excluded")
}

func TestBuildChapterDocumentRejectsUnverifiedClaims(t *testing.T) {
	snapshot := testSnapshot()
	files, err := BuildInventory([]InventoryInput{{Path: "main.go", Content: []byte("package main\n")}}, InventoryOptions{MaxContentBytes: 1000})
	require.NoError(t, err)
	graph := KnowledgeGraph{CommitSHA: snapshot.ResolvedCommitSHA, Files: files}
	id := evidenceIDsForFiles(files)[0]
	pack, err := BuildEvidencePack(snapshot, "chapter-1", []string{id}, graph, nil)
	require.NoError(t, err)
	_, _, err = BuildChapterDocument(sharedkernel.Chapter{ID: "chapter-1", Type: sharedkernel.ChapterMainFlow, Title: "主链路"}, pack, ClaimPlan{Claims: []sharedkernel.Claim{{ClaimID: "claim-1", Text: "无证据事实", ClaimType: "project_fact", Confidence: sharedkernel.ConfidenceObserved, EvidenceIDs: []string{id}, Status: sharedkernel.ClaimUnsupported}}}, "# 主链路", nil, false)
	require.ErrorContains(t, err, "validate claim plan")
}
