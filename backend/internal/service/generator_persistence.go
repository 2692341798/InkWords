package service

import (
	blogdomain "inkwords-backend/internal/domain/blog"
	blogcontracts "inkwords-backend/internal/domain/blog/contracts"
	"gorm.io/gorm"
)

type GeneratedBlogPersistence = blogcontracts.GeneratedBlogPersistence
type GeneratedBlogPersistenceInput = blogcontracts.GeneratedBlogPersistenceInput

func newDatabaseGeneratedBlogPersistence(database *gorm.DB) GeneratedBlogPersistence {
	return blogdomain.NewGeneratedBlogPersistence(database)
}
