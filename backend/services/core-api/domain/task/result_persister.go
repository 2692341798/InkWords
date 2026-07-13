package task

import (
	"context"
	"fmt"
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

type ProjectCourseGenerationResultRepository interface {
	PersistProjectCourseGenerationResult(ctx context.Context, result map[string]any) error
}

type ProjectCourseBlogResultRepository interface {
	PersistProjectCourseGenerationBlogs(ctx context.Context, taskID uuid.UUID, result map[string]any) error
}

// ResultPersister coordinates final result writes that must stay in core-api.
type ResultPersister struct {
	blogRepo                    BlogResultRepository
	usageRepo                   UsageRepository
	projectCourseRepo           ProjectCourseResultRepository
	projectCourseGenerationRepo ProjectCourseGenerationResultRepository
	projectCourseBlogRepo       ProjectCourseBlogResultRepository
}

// NewResultPersister creates a core-api owned result persister.
func NewResultPersister(blogRepo BlogResultRepository, usageRepo UsageRepository, projectCourseRepos ...ProjectCourseResultRepository) *ResultPersister {
	var projectCourseRepo ProjectCourseResultRepository
	if len(projectCourseRepos) > 0 {
		projectCourseRepo = projectCourseRepos[0]
	}
	var projectCourseGenerationRepo ProjectCourseGenerationResultRepository
	if projectCourseRepo != nil {
		projectCourseGenerationRepo, _ = projectCourseRepo.(ProjectCourseGenerationResultRepository)
	}
	var projectCourseBlogRepo ProjectCourseBlogResultRepository
	if blogRepo != nil {
		projectCourseBlogRepo, _ = blogRepo.(ProjectCourseBlogResultRepository)
	}
	return &ResultPersister{
		blogRepo:                    blogRepo,
		usageRepo:                   usageRepo,
		projectCourseRepo:           projectCourseRepo,
		projectCourseGenerationRepo: projectCourseGenerationRepo,
		projectCourseBlogRepo:       projectCourseBlogRepo,
	}
}

// PersistGenerationResult writes the final result into blogs first, then applies optional usage accounting.
func (p *ResultPersister) PersistGenerationResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error {
	subtype := strings.TrimSpace(stringValue(result["task_subtype"]))
	if subtype == ProjectCourseAnalyzeTaskSubtype {
		if p.projectCourseRepo == nil {
			return nil
		}
		if err := p.projectCourseRepo.PersistProjectCourseResult(ctx, result); err != nil {
			return err
		}
		return nil
	}
	if subtype == ProjectCourseGenerateTaskSubtype {
		if p.projectCourseGenerationRepo == nil {
			return fmt.Errorf("project course generation result repository is not configured")
		}
		if err := p.projectCourseGenerationRepo.PersistProjectCourseGenerationResult(ctx, result); err != nil {
			return err
		}
		if p.projectCourseBlogRepo == nil {
			return fmt.Errorf("project course blog result repository is not configured")
		}
		return p.projectCourseBlogRepo.PersistProjectCourseGenerationBlogs(ctx, taskID, result)
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
