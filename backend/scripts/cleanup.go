//go:build tools
// +build tools

package main

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=inkwords port=5432 sslmode=disable"
		log.Println("Using default DSN:", dsn)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 查找在同一 ParentID 下存在重复 Title 的子节点
	var dirtyParents []string
	err = db.Table("blogs").
		Select("parent_id").
		Where("parent_id IS NOT NULL").
		Group("parent_id, title").
		Having("COUNT(id) > 1").
		Pluck("parent_id", &dirtyParents).Error

	if err != nil {
		log.Fatalf("Failed to query dirty data: %v", err)
	}

	if len(dirtyParents) == 0 {
		log.Println("没有发现重复的脏数据。")
		return
	}

	log.Printf("发现 %d 个受影响的父节点，正在清理...", len(dirtyParents))

	// 删除受影响的子节点
	if err := db.Where("parent_id IN ?", dirtyParents).Delete(&model.Blog{}).Error; err != nil {
		log.Fatalf("清理子节点失败: %v", err)
	}

	// 删除受影响的父节点
	if err := db.Where("id IN ?", dirtyParents).Delete(&model.Blog{}).Error; err != nil {
		log.Fatalf("清理父节点失败: %v", err)
	}

	log.Println("清理完成！已移除脏数据的父节点及其下所有子节点，请刷新页面重新生成。")
}
