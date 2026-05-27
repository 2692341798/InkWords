package db

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
)

// DB 全局数据库实例
var DB *gorm.DB

// InitDB 初始化数据库连接并执行自动迁移
func InitDB(dsn string) error {
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		return err
	}

	err = autoMigrate(DB)
	if err != nil {
		log.Printf("Failed to auto migrate database: %v", err)
		return err
	}

	log.Println("Database connection and migration successful")
	return nil
}

// autoMigrate 将所有持久化模型集中在一个入口，避免启动路径和测试路径出现迁移清单漂移。
func autoMigrate(database *gorm.DB) error {
	return database.AutoMigrate(
		&model.User{},
		&model.Blog{},
		&model.OAuthToken{},
		&model.UserPromptSettings{},
		&model.ReviewSession{},
		&model.ReviewTurn{},
	)
}
