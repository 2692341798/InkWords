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

	// 执行自动迁移
	err = DB.AutoMigrate(
		&model.User{},
		&model.VerificationCode{},
		&model.Blog{},
		&model.OAuthToken{},
	)
	if err != nil {
		log.Printf("Failed to auto migrate database: %v", err)
		return err
	}

	log.Println("Database connection and migration successful")
	return nil
}
