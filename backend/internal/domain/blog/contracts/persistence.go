package contracts

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"inkwords-backend/internal/model"
)

// Chapter represents a single chapter in the generated outline.
type Chapter struct {
	ID      string   `json:"id,omitempty"`
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Sort    int      `json:"sort"`
	Files   []string `json:"files"`
	Action  string   `json:"action,omitempty"`
}

// GeneratedBlogPersistenceInput contains the persisted business facts produced
// by GeneratorService after LLM generation has completed.
type GeneratedBlogPersistenceInput struct {
	UserID     uuid.UUID
	Title      string
	Content    string
	SourceType string
	WordCount  int
	TechStacks datatypes.JSON
}

// GeneratedBlogPersistence defines the explicit persistence boundary for
// GeneratorService when it needs to store the final generated blog outcome.
type GeneratedBlogPersistence interface {
	SaveGeneratedBlog(ctx context.Context, input GeneratedBlogPersistenceInput) error
}

// ContinuePersistence captures continue flow reads and final blog updates.
type ContinuePersistence interface {
	LoadContinueBlog(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) (model.Blog, error)
	SaveContinuedBlog(ctx context.Context, blog model.Blog, updatedContent string) error
}

// SeriesChapterPersistenceInput describes the persisted business facts for a
// completed series chapter.
type SeriesChapterPersistenceInput struct {
	UserID     uuid.UUID
	ParentID   uuid.UUID
	BlogID     uuid.UUID
	Chapter    Chapter
	SourceType string
	Content    string
	WordCount  int
	TechStacks datatypes.JSON
}

// SeriesDraftPreflightInput describes the context needed to prepare series
// parent/draft blogs before generation starts.
type SeriesDraftPreflightInput struct {
	UserID      uuid.UUID
	ParentID    uuid.UUID
	ParentTitle string
	SourceType  string
	GitURL      string
	Outline     []Chapter
}

// SeriesPersistence captures series-generation business fact writes that still
// belong to core-api owned blog tables.
type SeriesPersistence interface {
	EnsureSeriesParentAndDrafts(ctx context.Context, input SeriesDraftPreflightInput) ([]Chapter, error)
	SaveSeriesChapter(ctx context.Context, input SeriesChapterPersistenceInput) error
	MarkSeriesChapterFailed(ctx context.Context, userID uuid.UUID, blogID uuid.UUID) error
	SaveSeriesIntro(ctx context.Context, parentID uuid.UUID, content string) error
	MarkSeriesIntroFailed(ctx context.Context, parentID uuid.UUID) error
	LoadSeriesOldContent(ctx context.Context, blogID uuid.UUID) (string, error)
	UpdateSkippedSeriesChapterMeta(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, chapter Chapter) error
}

