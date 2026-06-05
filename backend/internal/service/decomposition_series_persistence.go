package service

import (
	blogdomain "inkwords-backend/internal/domain/blog"
	blogcontracts "inkwords-backend/internal/domain/blog/contracts"
	"gorm.io/gorm"
)

type SeriesChapterPersistenceInput = blogcontracts.SeriesChapterPersistenceInput
type SeriesDraftPreflightInput = blogcontracts.SeriesDraftPreflightInput
type SeriesPersistence = blogcontracts.SeriesPersistence

// NewGormSeriesPersistence keeps the legacy constructor stable while routing
// the default production adapter to blog-domain.
func NewGormSeriesPersistence(database *gorm.DB) SeriesPersistence {
	return blogdomain.NewSeriesPersistence(database)
}
