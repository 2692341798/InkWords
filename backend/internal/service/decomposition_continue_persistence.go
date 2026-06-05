package service

import (
	blogdomain "inkwords-backend/internal/domain/blog"
	blogcontracts "inkwords-backend/internal/domain/blog/contracts"
	"gorm.io/gorm"
)

type ContinuePersistence = blogcontracts.ContinuePersistence

// NewGormContinuePersistence keeps the legacy constructor stable while routing
// the default production adapter to blog-domain.
func NewGormContinuePersistence(database *gorm.DB) ContinuePersistence {
	return blogdomain.NewContinuePersistence(database)
}
