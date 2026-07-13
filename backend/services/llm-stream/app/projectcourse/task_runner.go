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

type CourseTaskRunner struct{ analyzer RepositoryAnalyzer }

func NewCourseTaskRunner(analyzer RepositoryAnalyzer) *CourseTaskRunner {
	return &CourseTaskRunner{analyzer: analyzer}
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
