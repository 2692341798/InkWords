package db

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
)

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

	log.Println("Database connection and migration successful")
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
	return database.AutoMigrate(
		&model.User{},
		&model.Blog{},
		&model.OAuthToken{},
		&model.UserPromptSettings{},
		&model.JobTask{},
		&model.JobTaskEvent{},
	)
}

func autoMigrateReview(database *gorm.DB) error {
	return database.AutoMigrate(
		&model.ReviewSession{},
		&model.ReviewTurn{},
	)
}

func InitDB(dsn string) error { return InitCoreDB(dsn) }
