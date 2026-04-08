package service

import (
	"encoding/json"
	"errors"
	"inkwords-backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserService 处理用户相关的业务逻辑
type UserService struct {
	db *gorm.DB
}

// NewUserService 创建 UserService 实例
func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

// GetUserByID 根据 ID 获取用户信息
func (s *UserService) GetUserByID(uid uuid.UUID) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUsername 更新用户名
func (s *UserService) UpdateUsername(uid uuid.UUID, username string) error {
	if len(username) < 2 || len(username) > 20 {
		return errors.New("用户名长度必须在 2 到 20 个字符之间")
	}
	return s.db.Model(&model.User{}).Where("id = ?", uid).Update("username", username).Error
}

// CheckQuota 检查用户的 Token 是否超额
func (s *UserService) CheckQuota(uid uuid.UUID) error {
	user, err := s.GetUserByID(uid)
	if err != nil {
		return err
	}
	
	limit := user.TokenLimit
	if limit == 0 {
		limit = 100000
	}

	if user.TokensUsed >= limit {
		return errors.New("您的 Token 额度已耗尽，请升级订阅或联系管理员")
	}

	return nil
}
func (s *UserService) UpdateAvatarURL(uid uuid.UUID, avatarURL string) error {
	return s.db.Model(&model.User{}).Where("id = ?", uid).Update("avatar_url", avatarURL).Error
}

// GetUserStats 获取用户 Dashboard 统计数据
func (s *UserService) GetUserStats(uid uuid.UUID) (int64, int64, map[string]int, error) {
	var user model.User
	if err := s.db.First(&user, "id = ?", uid).Error; err != nil {
		return 0, 0, nil, err
	}

	var totalArticles int64
	var totalWords int64

	// Exclude parent nodes (where parent_id is null) when counting articles
	s.db.Model(&model.Blog{}).Where("user_id = ? AND status = 1 AND parent_id IS NOT NULL", uid).Count(&totalArticles)

	type Result struct {
		TotalWords int64
	}
	var res Result
	s.db.Model(&model.Blog{}).Select("sum(word_count) as total_words").Where("user_id = ? AND status = 1 AND parent_id IS NOT NULL", uid).Scan(&res)
	totalWords = res.TotalWords

	var blogs []model.Blog
	// Exclude parent nodes when aggregating tech stacks
	s.db.Where("user_id = ? AND status = 1 AND parent_id IS NOT NULL AND tech_stacks IS NOT NULL", uid).Find(&blogs)

	stackMap := make(map[string]int)
	for _, blog := range blogs {
		var stacks []string
		if len(blog.TechStacks) > 0 {
			if err := json.Unmarshal(blog.TechStacks, &stacks); err == nil {
				for _, stack := range stacks {
					stackMap[stack]++
				}
			}
		}
	}

	return totalArticles, totalWords, stackMap, nil
}
