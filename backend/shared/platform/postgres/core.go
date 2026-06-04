package postgres

import (
	"fmt"

	"gorm.io/gorm"

	"inkwords-backend/internal/infra/db"
)

// InitCore initializes the shared core database connection and returns the active GORM handle.
func InitCore(dsn string) (*gorm.DB, error) {
	if err := db.InitCoreDB(dsn); err != nil {
		return nil, fmt.Errorf("init core db: %w", err)
	}
	return db.DB, nil
}

// InitReview initializes the shared review database connection and returns the active GORM handle.
func InitReview(dsn string) (*gorm.DB, error) {
	if err := db.InitReviewDB(dsn); err != nil {
		return nil, fmt.Errorf("init review db: %w", err)
	}
	return db.DB, nil
}
