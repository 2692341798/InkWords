package projectcourse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"inkwords-backend/shared/kernel/projectcourse"
)

func TestPlanBlueprintIsDeterministicAndEvidenceLinked(t *testing.T) {
	snapshot := projectcourse.SourceSnapshot{RepositoryURL: "https://github.com/example/project", RequestedRef: "main", ResolvedCommitSHA: "0123456789abcdef0123456789abcdef01234567", CapturedAt: time.Unix(1, 0)}
	graph := KnowledgeGraph{CommitSHA: snapshot.ResolvedCommitSHA, Files: []InventoryEntry{
		{Path: "README.md", Role: RoleDocumentation, Disposition: DispositionIndexed, ContentHash: "sha256:readme"},
		{Path: "backend/routes.go", Role: RoleTransport, Disposition: DispositionCovered, ContentHash: "sha256:routes"},
		{Path: "assets/logo.png", Role: RoleBinary, Disposition: DispositionExcluded, ContentHash: "sha256:binary"},
	}}
	first, coverage, err := PlanBlueprint("course-1", snapshot, graph, projectcourse.AudienceProgramming)
	require.NoError(t, err)
	second, _, err := PlanBlueprint("course-1", snapshot, graph, projectcourse.AudienceProgramming)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Len(t, first.Volumes[0].Chapters, 3)
	require.NotEmpty(t, first.Volumes[0].Chapters[0].EvidenceIDs)
	require.Equal(t, 2.0/3.0, coverage.CoveredRate(coverage.Files))
	require.Equal(t, projectcourse.ChapterMainFlow, first.Volumes[0].Chapters[2].Type)
}

func TestPlanBlueprintRejectsMismatchedGraphSnapshot(t *testing.T) {
	snapshot := projectcourse.SourceSnapshot{RepositoryURL: "https://github.com/example/project", RequestedRef: "main", ResolvedCommitSHA: "0123456789abcdef0123456789abcdef01234567", CapturedAt: time.Unix(1, 0)}
	_, _, err := PlanBlueprint("course-1", snapshot, KnowledgeGraph{CommitSHA: "fedcba9876543210fedcba9876543210fedcba98"}, projectcourse.AudienceProgramming)
	require.Error(t, err)
}
