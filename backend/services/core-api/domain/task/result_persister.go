package task

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// BlogResultRepository defines the core-api owned write boundary for persisting final generation output into blogs.
type BlogResultRepository interface {
	PersistGenerationResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error
}

// TextbookSampleResultRepository is the textbook-owned boundary for one validated candidate.
type TextbookSampleResultRepository interface {
	PersistTextbookSampleResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error
}

// TextbookSourceImportResultRepository owns the core-side write from one
// parser result into immutable source snapshots and citeable chunks.
type TextbookSourceImportResultRepository interface {
	PersistTextbookSourceImportResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error
}

// ResultPersister coordinates final result writes that must stay in core-api.
type ResultPersister struct {
	blogRepo                 BlogResultRepository
	textbookSampleRepo       TextbookSampleResultRepository
	textbookSourceImportRepo TextbookSourceImportResultRepository
}

// WithTextbookSampleRepository enables a dedicated result path without widening legacy blog writes.
func (p *ResultPersister) WithTextbookSampleRepository(repo TextbookSampleResultRepository) *ResultPersister {
	p.textbookSampleRepo = repo
	return p
}

func (p *ResultPersister) WithTextbookSourceImportRepository(repo TextbookSourceImportResultRepository) *ResultPersister {
	p.textbookSourceImportRepo = repo
	return p
}

// NewResultPersister creates a core-api owned result persister.
func NewResultPersister(blogRepo BlogResultRepository) *ResultPersister {
	return &ResultPersister{blogRepo: blogRepo}
}

// PersistGenerationResult writes the final result into its authoritative domain projection.
func (p *ResultPersister) PersistGenerationResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error {
	subtype := strings.TrimSpace(stringValue(result["task_subtype"]))
	if subtype == sharedtextbook.TextbookSampleGenerationTaskSubtype {
		if p.textbookSampleRepo == nil {
			return fmt.Errorf("textbook sample result repository is not configured")
		}
		return p.textbookSampleRepo.PersistTextbookSampleResult(ctx, taskID, result)
	}
	if p.blogRepo != nil {
		if err := p.blogRepo.PersistGenerationResult(ctx, taskID, result); err != nil {
			return err
		}
	}
	return nil
}

// PersistParseResult handles only explicitly typed parser results. Generic
// legacy parsing remains a transient helper and cannot create textbook facts.
func (p *ResultPersister) PersistParseResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error {
	subtype := strings.TrimSpace(stringValue(result["task_subtype"]))
	if subtype != sharedtextbook.TextbookSourceImportTaskSubtype && subtype != sharedtextbook.TextbookOfficialWebImportTaskSubtype {
		return nil
	}
	if p.textbookSourceImportRepo == nil {
		return fmt.Errorf("textbook source import result repository is not configured")
	}
	return p.textbookSourceImportRepo.PersistTextbookSourceImportResult(ctx, taskID, result)
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
