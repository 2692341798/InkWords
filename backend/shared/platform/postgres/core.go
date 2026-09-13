package postgres

import (
	"context"
	"fmt"
	"log"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"inkwords-backend/internal/infra/db"
	"inkwords-backend/shared/platform/postgres/migrations"
)

// InitCore initializes the shared core database connection and returns the active GORM handle.
func InitCore(dsn string) (*gorm.DB, error) {
	if err := db.InitCoreDB(dsn); err != nil {
		return nil, fmt.Errorf("init core db: %w", err)
	}

	sqlDB, err := db.DB.DB()
	if err != nil {
		return nil, fmt.Errorf("get core sql db for migrations: %w", err)
	}
	results, err := migrations.UpCore(context.Background(), sqlDB)
	if err != nil {
		return nil, fmt.Errorf("run core migrations: %w", err)
	}
	if len(results) > 0 {
		log.Printf("Applied %d versioned core migration(s)", len(results))
	}
	return db.DB, nil
}

// OpenExisting connects to a schema owned by another local service without
// running legacy AutoMigrate or versioned migrations. Review-service uses this
// only to resolve the installation's stable workspace identity from core data.
func OpenExisting(dsn string) (*gorm.DB, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("database dsn is required")
	}
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open existing database: %w", err)
	}
	return database, nil
}

// InitReview initializes the shared review database connection and returns the active GORM handle.
func InitReview(dsn string) (*gorm.DB, error) {
	if err := db.InitReviewDB(dsn); err != nil {
		return nil, fmt.Errorf("init review db: %w", err)
	}
	sqlDB, err := db.DB.DB()
	if err != nil {
		return nil, fmt.Errorf("get review sql db for migrations: %w", err)
	}
	results, err := migrations.UpReview(context.Background(), sqlDB)
	if err != nil {
		return nil, fmt.Errorf("run review migrations: %w", err)
	}
	if len(results) > 0 {
		log.Printf("Applied %d versioned review migration(s)", len(results))
	}
	return db.DB, nil
}
