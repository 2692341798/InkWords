package db

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
)

const (
	coreVersionTable           = "inkwords_core_schema_migrations"
	reviewVersionTable         = "inkwords_review_schema_migrations"
	legacyCoreCleanupVersion   = int64(24)
	legacyReviewCleanupVersion = int64(25)
)

// These projections exist only so a brand-new database can run the historical
// migration chain that predates workspace ownership. Once the cleanup migration
// is recorded, AutoMigrate must never recreate these tables or columns.
type legacyOwnerBootstrap struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`
}

func (legacyOwnerBootstrap) TableName() string { return "users" }

type legacyBlogOwnerBootstrap struct {
	ID     uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID *uuid.UUID `gorm:"type:uuid;index:idx_user_parent_chapter"`
}

func (legacyBlogOwnerBootstrap) TableName() string { return "blogs" }

type legacyTaskOwnerBootstrap struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"`
	RequestedBy *uuid.UUID `gorm:"type:uuid;index:idx_job_tasks_requested_by"`
}

func (legacyTaskOwnerBootstrap) TableName() string { return "job_tasks" }

type legacyReviewOwnerBootstrap struct {
	ID     uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID *uuid.UUID `gorm:"type:uuid;index:idx_review_sessions_user_note_created"`
}

func (legacyReviewOwnerBootstrap) TableName() string { return "review_sessions" }

// DB 全局数据库实例
var DB *gorm.DB

func InitCoreDB(dsn string) error {
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		return err
	}

	err = autoMigrateCore(DB)
	if err != nil {
		log.Printf("Failed to auto migrate database: %v", err)
		return err
	}

	log.Println("Database connection and schema bootstrap successful; versioned core migrations run next")
	return nil
}

func InitReviewDB(dsn string) error {
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		return err
	}

	err = autoMigrateReview(DB)
	if err != nil {
		log.Printf("Failed to auto migrate database: %v", err)
		return err
	}

	log.Println("Database connection and migration successful")
	return nil
}

// autoMigrateCore/autoMigrateReview 将持久化模型集中在入口，避免启动路径和测试路径出现迁移清单漂移。
func autoMigrateCore(database *gorm.DB) error {
	cleaned, err := migrationApplied(database, coreVersionTable, legacyCoreCleanupVersion)
	if err != nil {
		return fmt.Errorf("inspect legacy core cleanup state: %w", err)
	}
	if !cleaned {
		if err := database.AutoMigrate(
			&legacyOwnerBootstrap{},
			&legacyBlogOwnerBootstrap{},
			&legacyTaskOwnerBootstrap{},
		); err != nil {
			return err
		}
	}
	return database.AutoMigrate(
		&model.Blog{},
		&model.JobTask{},
		&model.JobTaskEvent{},
	)
}

func autoMigrateReview(database *gorm.DB) error {
	cleaned, err := migrationApplied(database, reviewVersionTable, legacyReviewCleanupVersion)
	if err != nil {
		return fmt.Errorf("inspect legacy review cleanup state: %w", err)
	}
	if !cleaned {
		if err := database.AutoMigrate(&legacyReviewOwnerBootstrap{}); err != nil {
			return err
		}
	}
	return database.AutoMigrate(
		&model.ReviewSession{},
		&model.ReviewTurn{},
	)
}

func migrationApplied(database *gorm.DB, tableName string, version int64) (bool, error) {
	if !database.Migrator().HasTable(tableName) {
		return false, nil
	}
	var count int64
	if err := database.Table(tableName).
		Where("version_id = ? AND is_applied = ?", version, true).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count == 1, nil
}

func InitDB(dsn string) error { return InitCoreDB(dsn) }
