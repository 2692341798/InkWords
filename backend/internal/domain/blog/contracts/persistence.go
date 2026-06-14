package contracts

import sharedblog "inkwords-backend/shared/kernel/blog"

// Chapter represents a single chapter in the generated outline.
type Chapter = sharedblog.Chapter

// GeneratedBlogPersistenceInput contains the persisted business facts produced
// by GeneratorService after LLM generation has completed.
type GeneratedBlogPersistenceInput = sharedblog.GeneratedBlogPersistenceInput

// GeneratedBlogPersistence defines the explicit persistence boundary for
// GeneratorService when it needs to store the final generated blog outcome.
type GeneratedBlogPersistence = sharedblog.GeneratedBlogPersistence

// ContinueBlog is the minimal blog projection needed by the continue flow.
type ContinueBlog = sharedblog.ContinueBlog

// ContinuePersistence captures continue flow reads and final blog updates.
type ContinuePersistence = sharedblog.ContinuePersistence

// SeriesChapterPersistenceInput describes the persisted business facts for a
// completed series chapter.
type SeriesChapterPersistenceInput = sharedblog.SeriesChapterPersistenceInput

// SeriesDraftPreflightInput describes the context needed to prepare series
// parent/draft blogs before generation starts.
type SeriesDraftPreflightInput = sharedblog.SeriesDraftPreflightInput

// SeriesPersistence captures series-generation business fact writes that still
// belong to core-api owned blog tables.
type SeriesPersistence = sharedblog.SeriesPersistence
