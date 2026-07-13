package projectcourse

import (
	"context"
	"encoding/json"
	"fmt"
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

type fakeOfficialSourceResolver struct{}

func (fakeOfficialSourceResolver) FetchTechnology(technology, versionConstraint string) (OfficialSource, error) {
	content := "fixture official source for " + technology
	return OfficialSource{
		Technology:        technology,
		VersionConstraint: versionConstraint,
		URL:               "https://go.dev/doc/",
		Content:           content,
		ContentHash:       contentHash([]byte(content)),
		RetrievedAt:       time.Unix(1, 0).UTC(),
	}, nil
}

type contractChapterGenerator struct{}

func (contractChapterGenerator) Generate(_ context.Context, chapter sharedkernel.Chapter, pack EvidencePack, _ sharedkernel.AudienceLevel) (ChapterDocument, error) {
	required, _, requiresLab, ok := chapterContractFor(chapter.Type)
	if !ok {
		return ChapterDocument{}, fmt.Errorf("unknown chapter type %q", chapter.Type)
	}
	if requiresLab {
		return ChapterDocument{}, fmt.Errorf("lab verification is deferred for %s", chapter.ID)
	}
	markdown := "# " + chapter.Title + "\n"
	for _, section := range required {
		markdown += "## " + section + "\n本地回归证据。\n"
	}
	evidenceID := pack.SourceEvidence[0].EvidenceID
	claim := sharedkernel.Claim{ClaimID: "claim-" + stableID(chapter.ID), Text: "章节证据存在", ClaimType: "project_fact", Confidence: sharedkernel.ConfidenceObserved, EvidenceIDs: []string{evidenceID}, Status: sharedkernel.ClaimVerified}
	document, _, err := BuildChapterDocument(chapter, pack, ClaimPlan{Claims: []sharedkernel.Claim{claim}, EvidenceIDs: []string{evidenceID}}, markdown, nil, false)
	return document, err
}

type hardFailChapterGenerator struct{}

func (hardFailChapterGenerator) Generate(_ context.Context, chapter sharedkernel.Chapter, pack EvidencePack, _ sharedkernel.AudienceLevel) (ChapterDocument, error) {
	return ChapterDocument{ChapterID: chapter.ID, ChapterType: chapter.Type, Title: chapter.Title, Markdown: "# " + chapter.Title, EvidencePack: pack}, nil
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

func TestCourseTaskRunnerDoesNotCompleteChapterAfterHardQualityFailure(t *testing.T) {
	snapshot := sharedkernel.SourceSnapshot{RepositoryURL: "https://github.com/example/project", RequestedRef: "main", ResolvedCommitSHA: "0123456789abcdef0123456789abcdef01234567", CapturedAt: time.Unix(1, 0).UTC()}
	files := []InventoryEntry{{Path: "main.go", Role: RoleApplication, Disposition: DispositionCovered, ContentHash: "sha256:main", Content: "package main"}}
	blueprint := sharedkernel.Blueprint{CourseID: "course-hard-gate", BlueprintVersion: 1, CommitSHA: snapshot.ResolvedCommitSHA, AudienceLevel: sharedkernel.AudienceProgramming, Volumes: []sharedkernel.Volume{{ID: "volume-1", Title: "硬门禁", Sort: 1, Chapters: []sharedkernel.Chapter{{ID: "map", Title: "项目地图", Sort: 1, Enabled: false, Type: sharedkernel.ChapterProjectMap}, {ID: "flow", Title: "主链路", Sort: 2, Enabled: true, Type: sharedkernel.ChapterMainFlow, EvidenceIDs: evidenceIDsForFiles(files)}}}}}
	payload, err := json.Marshal(generateTaskPayload{CourseID: blueprint.CourseID, RepositoryURL: snapshot.RepositoryURL, ResolvedCommitSHA: snapshot.ResolvedCommitSHA, Blueprint: blueprint})
	require.NoError(t, err)
	runner := NewCourseTaskRunnerWithGenerator(fakeRepositoryAnalyzer{analysis: RepositoryAnalysis{Snapshot: snapshot, Graph: KnowledgeGraph{CommitSHA: snapshot.ResolvedCommitSHA, Files: files}}}, hardFailChapterGenerator{})
	result, err := runner.Run(context.Background(), sharedrabbitmq.GenerationRequestedMessage{Kind: "project_course_generate", Payload: payload})
	require.NoError(t, err)
	var decoded generateTaskResult
	require.NoError(t, json.Unmarshal(result, &decoded))
	require.Equal(t, string(sharedkernel.CourseBlocked), decoded.Status)
	require.Len(t, decoded.Chapters, 1)
	require.Equal(t, "blocked", decoded.Chapters[0].Status)
	require.Contains(t, decoded.Chapters[0].Error, "quality gate")
	var foundHardFailure bool
	for _, gate := range decoded.QualityReport {
		if gate.Name == "chapter_contract" && gate.Result == sharedkernel.GateHardFail {
			foundHardFailure = true
		}
	}
	require.True(t, foundHardFailure)
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

func TestCourseTaskRunnerCoversEveryChapterContractWithDeferredLabs(t *testing.T) {
	snapshot := sharedkernel.SourceSnapshot{RepositoryURL: "https://github.com/example/project", RequestedRef: "main", ResolvedCommitSHA: "0123456789abcdef0123456789abcdef01234567", CapturedAt: time.Unix(1, 0).UTC()}
	files := []InventoryEntry{{Path: "main.go", Role: RoleApplication, Disposition: DispositionCovered, ContentHash: "sha256:main", Content: "package main\nfunc main() {}"}}
	chapterTypes := []sharedkernel.ChapterType{
		sharedkernel.ChapterProjectMap,
		sharedkernel.ChapterTechnicalTheory,
		sharedkernel.ChapterMainFlow,
		sharedkernel.ChapterModuleDeepDive,
		sharedkernel.ChapterDesignTradeoff,
		sharedkernel.ChapterHandsOnLab,
		sharedkernel.ChapterTroubleshooting,
		sharedkernel.ChapterChallenge,
	}
	chapters := make([]sharedkernel.Chapter, 0, len(chapterTypes))
	for index, chapterType := range chapterTypes {
		title := string(chapterType)
		if chapterType == sharedkernel.ChapterTechnicalTheory {
			title = "PostgreSQL 原理"
		}
		chapters = append(chapters, sharedkernel.Chapter{ID: fmt.Sprintf("chapter-%d", index), Title: title, Sort: index + 1, Enabled: true, Type: chapterType, EvidenceIDs: evidenceIDsForFiles(files)})
	}
	blueprint := sharedkernel.Blueprint{CourseID: "course-contracts", BlueprintVersion: 1, CommitSHA: snapshot.ResolvedCommitSHA, AudienceLevel: sharedkernel.AudienceProgramming, Volumes: []sharedkernel.Volume{{ID: "volume-1", Title: "合同回归", Sort: 1, Chapters: chapters}}}
	payload, err := json.Marshal(generateTaskPayload{CourseID: blueprint.CourseID, RepositoryURL: snapshot.RepositoryURL, ResolvedCommitSHA: snapshot.ResolvedCommitSHA, Blueprint: blueprint})
	require.NoError(t, err)
	runner := NewCourseTaskRunnerWithGenerator(
		fakeRepositoryAnalyzer{analysis: RepositoryAnalysis{Snapshot: snapshot, Graph: KnowledgeGraph{CommitSHA: snapshot.ResolvedCommitSHA, Files: files}}},
		contractChapterGenerator{},
		fakeOfficialSourceResolver{},
	)
	result, err := runner.Run(context.Background(), sharedrabbitmq.GenerationRequestedMessage{Kind: "project_course_generate", Payload: payload})
	require.NoError(t, err)
	var decoded generateTaskResult
	require.NoError(t, json.Unmarshal(result, &decoded))
	require.Equal(t, string(sharedkernel.CoursePartiallyBlocked), decoded.Status)
	require.Len(t, decoded.Chapters, len(chapterTypes))
	succeeded, blocked := 0, 0
	for _, chapter := range decoded.Chapters {
		switch chapter.Status {
		case "succeeded":
			succeeded++
		case "blocked":
			blocked++
		default:
			t.Fatalf("unexpected chapter status %q", chapter.Status)
		}
	}
	require.Equal(t, 6, succeeded)
	require.Equal(t, 2, blocked)
	require.NotEmpty(t, decoded.QualityReport)
	for _, gate := range decoded.QualityReport {
		require.NotEqual(t, sharedkernel.GateHardFail, gate.Result, gate.Name)
	}
	for _, checkpoint := range decoded.Checkpoints {
		require.NoError(t, checkpoint.Validate())
	}
}
