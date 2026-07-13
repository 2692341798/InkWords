package projectcourse

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

type RepositoryAnalysis struct {
	Snapshot sharedkernel.SourceSnapshot
	Graph    KnowledgeGraph
}

// RepositoryAnalyzer is intentionally injected: implementations may read repository contents,
// but must never build, install, test, or execute scripts from the target repository.
type RepositoryAnalyzer interface {
	Analyze(ctx context.Context, repositoryURL, requestedRef string) (RepositoryAnalysis, error)
}

type CourseTaskRunner struct {
	analyzer        RepositoryAnalyzer
	generator       ChapterGenerator
	officialSources OfficialSourceResolver
}

type OfficialSourceResolver interface {
	FetchTechnology(technology, versionConstraint string) (OfficialSource, error)
}

func NewCourseTaskRunner(analyzer RepositoryAnalyzer) *CourseTaskRunner {
	return &CourseTaskRunner{analyzer: analyzer}
}

func NewCourseTaskRunnerWithGenerator(analyzer RepositoryAnalyzer, generator ChapterGenerator, officialSources ...OfficialSourceResolver) *CourseTaskRunner {
	var resolver OfficialSourceResolver
	if len(officialSources) > 0 {
		resolver = officialSources[0]
	}
	return &CourseTaskRunner{analyzer: analyzer, generator: generator, officialSources: resolver}
}

type analyzeTaskPayload struct {
	CourseID      string `json:"course_id"`
	RepositoryURL string `json:"repository_url"`
	RequestedRef  string `json:"requested_ref"`
	AudienceLevel string `json:"audience_level"`
}

type analyzeTaskResult struct {
	TaskSubtype string                      `json:"task_subtype"`
	CourseID    string                      `json:"course_id"`
	Status      string                      `json:"status"`
	Snapshot    sharedkernel.SourceSnapshot `json:"snapshot"`
	Blueprint   sharedkernel.Blueprint      `json:"blueprint"`
	Coverage    sharedkernel.CoverageMatrix `json:"coverage"`
}

func (r *CourseTaskRunner) Run(ctx context.Context, message sharedrabbitmq.GenerationRequestedMessage) ([]byte, error) {
	if r == nil || r.analyzer == nil {
		return nil, fmt.Errorf("project course repository analyzer is not configured")
	}
	if message.Kind == "project_course_generate" {
		return r.runGenerate(ctx, message)
	}
	if message.Kind != "project_course_analyze" {
		return nil, fmt.Errorf("unsupported project course task kind %q", message.Kind)
	}
	var payload analyzeTaskPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return nil, fmt.Errorf("invalid project course payload")
	}
	if strings.TrimSpace(payload.CourseID) == "" || strings.TrimSpace(payload.RepositoryURL) == "" || strings.TrimSpace(payload.RequestedRef) == "" {
		return nil, fmt.Errorf("project course payload requires course_id, repository_url and requested_ref")
	}
	var audience sharedkernel.AudienceLevel
	if payload.AudienceLevel == "" {
		audience = sharedkernel.AudienceProgramming
	} else {
		audience = sharedkernel.AudienceLevel(payload.AudienceLevel)
	}
	analysis, err := r.analyzer.Analyze(ctx, payload.RepositoryURL, payload.RequestedRef)
	if err != nil {
		return nil, fmt.Errorf("analyze repository: %w", err)
	}
	blueprint, coverage, err := PlanBlueprint(payload.CourseID, analysis.Snapshot, analysis.Graph, audience)
	if err != nil {
		return nil, fmt.Errorf("plan project course blueprint: %w", err)
	}
	return json.Marshal(analyzeTaskResult{TaskSubtype: message.Kind, CourseID: payload.CourseID, Status: string(sharedkernel.CourseAwaitingApproval), Snapshot: analysis.Snapshot, Blueprint: blueprint, Coverage: coverage})
}

type generateTaskPayload struct {
	CourseID          string                 `json:"course_id"`
	RepositoryURL     string                 `json:"repository_url"`
	ResolvedCommitSHA string                 `json:"resolved_commit_sha"`
	Blueprint         sharedkernel.Blueprint `json:"blueprint"`
}

type generatedChapterResult struct {
	ChapterID   string           `json:"chapter_id"`
	VolumeID    string           `json:"volume_id,omitempty"`
	VolumeTitle string           `json:"volume_title,omitempty"`
	Sort        int              `json:"sort"`
	Status      string           `json:"status"`
	Document    *ChapterDocument `json:"document,omitempty"`
	Error       string           `json:"error,omitempty"`
}

type generateTaskResult struct {
	ResultVersion    int                      `json:"result_version"`
	TaskSubtype      string                   `json:"task_subtype"`
	CourseID         string                   `json:"course_id"`
	BlueprintVersion int                      `json:"blueprint_version"`
	CommitSHA        string                   `json:"commit_sha"`
	Status           string                   `json:"status"`
	BlogParentID     string                   `json:"blog_parent_id"`
	Chapters         []generatedChapterResult `json:"chapters"`
}

func (r *CourseTaskRunner) runGenerate(ctx context.Context, message sharedrabbitmq.GenerationRequestedMessage) ([]byte, error) {
	if r.generator == nil {
		return nil, fmt.Errorf("project course chapter generator is not configured")
	}
	var payload generateTaskPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return nil, fmt.Errorf("invalid project course generation payload")
	}
	if strings.TrimSpace(payload.CourseID) == "" || strings.TrimSpace(payload.RepositoryURL) == "" || strings.TrimSpace(payload.ResolvedCommitSHA) == "" {
		return nil, fmt.Errorf("generation payload requires course_id, repository_url and resolved_commit_sha")
	}
	if err := payload.Blueprint.Validate(); err != nil {
		return nil, fmt.Errorf("invalid generation blueprint: %w", err)
	}
	if payload.Blueprint.CourseID != payload.CourseID || payload.Blueprint.CommitSHA != payload.ResolvedCommitSHA {
		return nil, fmt.Errorf("generation payload blueprint does not match course snapshot")
	}
	analysis, err := r.analyzer.Analyze(ctx, payload.RepositoryURL, payload.ResolvedCommitSHA)
	if err != nil {
		return nil, fmt.Errorf("load pinned repository: %w", err)
	}
	if analysis.Snapshot.ResolvedCommitSHA != payload.ResolvedCommitSHA {
		return nil, fmt.Errorf("repository analyzer returned a different commit SHA")
	}
	result := generateTaskResult{ResultVersion: 1, TaskSubtype: message.Kind, CourseID: payload.CourseID, BlueprintVersion: payload.Blueprint.BlueprintVersion, CommitSHA: payload.ResolvedCommitSHA, Status: string(sharedkernel.CourseCompleted), BlogParentID: payload.CourseID}
	for _, volume := range payload.Blueprint.Volumes {
		for _, chapter := range volume.Chapters {
			if !chapter.Enabled {
				continue
			}
			official, officialErr := r.officialSourcesForChapter(chapter)
			if officialErr != nil {
				result.Status = string(sharedkernel.CourseBlocked)
				result.Chapters = append(result.Chapters, generatedChapterResult{ChapterID: chapter.ID, VolumeID: volume.ID, VolumeTitle: volume.Title, Sort: chapter.Sort, Status: "blocked", Error: officialErr.Error()})
				continue
			}
			pack, packErr := BuildEvidencePack(analysis.Snapshot, chapter.ID, chapter.EvidenceIDs, analysis.Graph, official)
			if packErr != nil {
				result.Status = string(sharedkernel.CourseBlocked)
				result.Chapters = append(result.Chapters, generatedChapterResult{ChapterID: chapter.ID, VolumeID: volume.ID, VolumeTitle: volume.Title, Sort: chapter.Sort, Status: "blocked", Error: packErr.Error()})
				continue
			}
			document, generateErr := r.generator.Generate(ctx, chapter, pack, payload.Blueprint.AudienceLevel)
			if generateErr != nil {
				result.Status = string(sharedkernel.CourseBlocked)
				result.Chapters = append(result.Chapters, generatedChapterResult{ChapterID: chapter.ID, VolumeID: volume.ID, VolumeTitle: volume.Title, Sort: chapter.Sort, Status: "blocked", Error: generateErr.Error()})
				continue
			}
			result.Chapters = append(result.Chapters, generatedChapterResult{ChapterID: chapter.ID, VolumeID: volume.ID, VolumeTitle: volume.Title, Sort: chapter.Sort, Status: "succeeded", Document: &document})
		}
	}
	return json.Marshal(result)
}

func (r *CourseTaskRunner) officialSourcesForChapter(chapter sharedkernel.Chapter) ([]OfficialSource, error) {
	_, requiresOfficial, _, ok := chapterContractFor(chapter.Type)
	if !ok || !requiresOfficial {
		return nil, nil
	}
	if r.officialSources == nil {
		return nil, fmt.Errorf("chapter %q requires an official source provider", chapter.ID)
	}
	technology := "Go"
	lowerTitle := strings.ToLower(chapter.Title)
	for candidate, name := range map[string]string{
		"react": "React", "zustand": "Zustand", "postgres": "PostgreSQL", "rabbitmq": "RabbitMQ",
		"redis": "Redis", "compose": "Docker Compose", "docker": "Docker Compose", "nginx": "Nginx", "typescript": "TypeScript", "gin": "Gin",
	} {
		if strings.Contains(lowerTitle, candidate) {
			technology = name
			break
		}
	}
	source, err := r.officialSources.FetchTechnology(technology, "")
	if err != nil {
		return nil, fmt.Errorf("fetch official source for %s: %w", technology, err)
	}
	return []OfficialSource{source}, nil
}
