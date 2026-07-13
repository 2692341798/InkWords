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

type fakeChapterGenerator struct{}

func (fakeChapterGenerator) Generate(_ context.Context, chapter sharedkernel.Chapter, pack EvidencePack, _ sharedkernel.AudienceLevel) (ChapterDocument, error) {
	evidenceID := pack.SourceEvidence[0].EvidenceID
	document, _, err := BuildChapterDocument(chapter, pack, ClaimPlan{Claims: []sharedkernel.Claim{{ClaimID: "claim-1", Text: "入口存在", ClaimType: "project_fact", Confidence: sharedkernel.ConfidenceObserved, EvidenceIDs: []string{evidenceID}, Status: sharedkernel.ClaimVerified}}, EvidenceIDs: []string{evidenceID}}, "# "+chapter.Title+"\n## 主链路\n## 数据流\n## 练习\n正文", nil, false)
	return document, err
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

func TestCourseTaskRunnerGeneratesOnlyAgainstPinnedCommit(t *testing.T) {
	snapshot := sharedkernel.SourceSnapshot{RepositoryURL: "https://github.com/example/project", RequestedRef: "0123456789abcdef0123456789abcdef01234567", ResolvedCommitSHA: "0123456789abcdef0123456789abcdef01234567", CapturedAt: time.Unix(1, 0)}
	files := []InventoryEntry{{Path: "main.go", Role: RoleApplication, Disposition: DispositionCovered, ContentHash: "sha256:main", Content: "package main\nfunc main() {}"}}
	chapter := sharedkernel.Chapter{ID: "chapter-1", Title: "入口", Sort: 1, Enabled: true, Type: sharedkernel.ChapterMainFlow, EvidenceIDs: evidenceIDsForFiles(files)}
	mapChapter := sharedkernel.Chapter{ID: "map", Title: "项目地图", Sort: 1, Enabled: false, Type: sharedkernel.ChapterProjectMap, EvidenceIDs: evidenceIDsForFiles(files)}
	chapter.Sort = 2
	blueprint := sharedkernel.Blueprint{CourseID: "course-1", BlueprintVersion: 2, CommitSHA: snapshot.ResolvedCommitSHA, AudienceLevel: sharedkernel.AudienceProgramming, Volumes: []sharedkernel.Volume{{ID: "volume-1", Title: "项目卷一", Sort: 1, Chapters: []sharedkernel.Chapter{mapChapter, chapter}}}}
	payload, err := json.Marshal(generateTaskPayload{CourseID: "course-1", RepositoryURL: snapshot.RepositoryURL, ResolvedCommitSHA: snapshot.ResolvedCommitSHA, Blueprint: blueprint})
	require.NoError(t, err)
	runner := NewCourseTaskRunnerWithGenerator(fakeRepositoryAnalyzer{analysis: RepositoryAnalysis{Snapshot: snapshot, Graph: KnowledgeGraph{CommitSHA: snapshot.ResolvedCommitSHA, Files: files}}}, fakeChapterGenerator{})
	result, err := runner.Run(context.Background(), sharedrabbitmq.GenerationRequestedMessage{Kind: "project_course_generate", Payload: payload})
	require.NoError(t, err)
	var decoded generateTaskResult
	require.NoError(t, json.Unmarshal(result, &decoded))
	require.Equal(t, string(sharedkernel.CourseCompleted), decoded.Status)
	require.Len(t, decoded.Chapters, 1)
	require.Equal(t, "succeeded", decoded.Chapters[0].Status)
	require.NotEmpty(t, decoded.Checkpoints)
	checkpoints := make(map[string]bool, len(decoded.Checkpoints))
	for _, checkpoint := range decoded.Checkpoints {
		require.NoError(t, checkpoint.Validate())
		checkpoints[checkpoint.Checkpoint] = true
	}
	require.True(t, checkpoints["snapshot"])
	require.True(t, checkpoints["inventory"])
	require.True(t, checkpoints["knowledge_graph"])
	require.True(t, checkpoints["blueprint"])
	require.True(t, checkpoints["draft:chapter-1"])
	require.True(t, checkpoints["review:chapter-1"])
	require.True(t, checkpoints["final_gate:chapter-1"])
}

func TestCourseTaskRunnerPreservesSuccessfulChaptersWhenAnotherIsBlocked(t *testing.T) {
	snapshot := sharedkernel.SourceSnapshot{RepositoryURL: "https://github.com/example/project", RequestedRef: "main", ResolvedCommitSHA: "0123456789abcdef0123456789abcdef01234567", CapturedAt: time.Unix(1, 0)}
	files := []InventoryEntry{{Path: "main.go", Role: RoleApplication, Disposition: DispositionCovered, ContentHash: "sha256:main", Content: "package main"}, {Path: "docker-compose.yml", Role: RoleConfiguration, Disposition: DispositionCovered, ContentHash: "sha256:compose", Content: "services:"}}
	blueprint := sharedkernel.Blueprint{CourseID: "course-1", BlueprintVersion: 1, CommitSHA: snapshot.ResolvedCommitSHA, AudienceLevel: sharedkernel.AudienceProgramming, Volumes: []sharedkernel.Volume{{ID: "volume-1", Title: "项目卷一", Sort: 1, Chapters: []sharedkernel.Chapter{
		{ID: "map", Title: "地图", Sort: 1, Enabled: false, Type: sharedkernel.ChapterProjectMap},
		{ID: "flow", Title: "主链路", Sort: 2, Enabled: true, Type: sharedkernel.ChapterMainFlow, EvidenceIDs: evidenceIDsForFiles(files[:1])},
		{ID: "theory", Title: "原理", Sort: 3, Enabled: true, Type: sharedkernel.ChapterTechnicalTheory, EvidenceIDs: evidenceIDsForFiles(files[1:])},
	}}}}
	payload, err := json.Marshal(generateTaskPayload{CourseID: "course-1", RepositoryURL: snapshot.RepositoryURL, ResolvedCommitSHA: snapshot.ResolvedCommitSHA, Blueprint: blueprint})
	require.NoError(t, err)
	runner := NewCourseTaskRunnerWithGenerator(fakeRepositoryAnalyzer{analysis: RepositoryAnalysis{Snapshot: snapshot, Graph: KnowledgeGraph{CommitSHA: snapshot.ResolvedCommitSHA, Files: files}}}, fakeChapterGenerator{})
	result, err := runner.Run(context.Background(), sharedrabbitmq.GenerationRequestedMessage{Kind: "project_course_generate", Payload: payload})
	require.NoError(t, err)
	var decoded generateTaskResult
	require.NoError(t, json.Unmarshal(result, &decoded))
	require.Equal(t, string(sharedkernel.CoursePartiallyBlocked), decoded.Status)
	require.Equal(t, "flow", decoded.Chapters[0].ChapterID)
	require.Equal(t, "succeeded", decoded.Chapters[0].Status)
	require.Equal(t, "theory", decoded.Chapters[1].ChapterID)
	require.Equal(t, "blocked", decoded.Chapters[1].Status)
}
