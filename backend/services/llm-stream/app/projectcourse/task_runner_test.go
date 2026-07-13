package projectcourse

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

type fakeRepositoryAnalyzer struct{ analysis RepositoryAnalysis }

func (f fakeRepositoryAnalyzer) Analyze(context.Context, string, string) (RepositoryAnalysis, error) {
	return f.analysis, nil
}

func TestCourseTaskRunnerReturnsAwaitingApprovalBlueprint(t *testing.T) {
	snapshot := sharedkernel.SourceSnapshot{RepositoryURL: "https://github.com/example/project", RequestedRef: "main", ResolvedCommitSHA: "0123456789abcdef0123456789abcdef01234567", CapturedAt: time.Unix(1, 0)}
	runner := NewCourseTaskRunner(fakeRepositoryAnalyzer{analysis: RepositoryAnalysis{Snapshot: snapshot, Graph: KnowledgeGraph{CommitSHA: snapshot.ResolvedCommitSHA, Files: []InventoryEntry{{Path: "main.go", Role: RoleApplication, Disposition: DispositionCovered, ContentHash: "sha256:main"}}}}})
	result, err := runner.Run(context.Background(), sharedrabbitmq.GenerationRequestedMessage{Kind: "project_course_analyze", Payload: []byte(`{"course_id":"course-1","repository_url":"https://github.com/example/project","requested_ref":"main","audience_level":"programming"}`)})
	require.NoError(t, err)
	var decoded analyzeTaskResult
	require.NoError(t, json.Unmarshal(result, &decoded))
	require.Equal(t, string(sharedkernel.CourseAwaitingApproval), decoded.Status)
	require.Equal(t, 1, decoded.Blueprint.BlueprintVersion)
	require.Equal(t, snapshot.ResolvedCommitSHA, decoded.Snapshot.ResolvedCommitSHA)
}

func TestCourseTaskRunnerRejectsMissingAnalyzer(t *testing.T) {
	_, err := NewCourseTaskRunner(nil).Run(context.Background(), sharedrabbitmq.GenerationRequestedMessage{Kind: "project_course_analyze", Payload: []byte(`{}`)})
	require.ErrorContains(t, err, "analyzer is not configured")
}
