package task

import (
	"context"

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

// ResultPersister coordinates final result writes that must stay in core-api.
type ResultPersister struct {
	blogRepo  BlogResultRepository
	usageRepo UsageRepository
}

// NewResultPersister creates a core-api owned result persister.
func NewResultPersister(blogRepo BlogResultRepository, usageRepo UsageRepository) *ResultPersister {
	return &ResultPersister{
		blogRepo:  blogRepo,
		usageRepo: usageRepo,
	}
}

// PersistGenerationResult writes the final result into blogs first, then applies optional usage accounting.
func (p *ResultPersister) PersistGenerationResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error {
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
