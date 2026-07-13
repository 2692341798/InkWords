package task

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// BlogResultRepository defines the core-api owned write boundary for persisting final generation output into blogs.
type BlogResultRepository interface {
	PersistGenerationResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error
}

// UsageRepository defines the optional token accounting side effect that still belongs to core-api.
type UsageRepository interface {
	AccumulateTokens(ctx context.Context, taskID uuid.UUID, result map[string]any) error
}

type ProjectCourseResultRepository interface {
	PersistProjectCourseResult(ctx context.Context, result map[string]any) error
}

// ResultPersister coordinates final result writes that must stay in core-api.
type ResultPersister struct {
	blogRepo          BlogResultRepository
	usageRepo         UsageRepository
	projectCourseRepo ProjectCourseResultRepository
}

// NewResultPersister creates a core-api owned result persister.
func NewResultPersister(blogRepo BlogResultRepository, usageRepo UsageRepository, projectCourseRepos ...ProjectCourseResultRepository) *ResultPersister {
	var projectCourseRepo ProjectCourseResultRepository
	if len(projectCourseRepos) > 0 {
		projectCourseRepo = projectCourseRepos[0]
	}
	return &ResultPersister{
		blogRepo:          blogRepo,
		usageRepo:         usageRepo,
		projectCourseRepo: projectCourseRepo,
	}
}

// PersistGenerationResult writes the final result into blogs first, then applies optional usage accounting.
func (p *ResultPersister) PersistGenerationResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error {
	if strings.HasPrefix(strings.TrimSpace(stringValue(result["task_subtype"])), "project_course_") {
		if p.projectCourseRepo == nil {
			return nil
		}
		if err := p.projectCourseRepo.PersistProjectCourseResult(ctx, result); err != nil {
			return err
		}
		return nil
	}
	if p.blogRepo != nil {
		if err := p.blogRepo.PersistGenerationResult(ctx, taskID, result); err != nil {
			return err
		}
	}
	if p.usageRepo != nil {
		return p.usageRepo.AccumulateTokens(ctx, taskID, result)
	}
	return nil
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
