package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"inkwords-backend/internal/db"
	"inkwords-backend/internal/model"
)

type UserAPI struct{}

func NewUserAPI() *UserAPI {
	return &UserAPI{}
}

// GetProfile 获取个人中心配置与额度
func (a *UserAPI) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    http.StatusUnauthorized,
			"message": "unauthorized",
			"data":    nil,
		})
		return
	}

	uid, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "invalid user id type",
			"data":    nil,
		})
		return
	}

	var user model.User
	if err := db.DB.First(&user, "id = ?", uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    http.StatusNotFound,
			"message": "user not found",
			"data":    nil,
		})
		return
	}

	// 假定默认额度
	tokenLimit := 100000

	// 连接平台逻辑
	connectedPlatforms := make([]string, 0)
	if user.GithubID != nil && *user.GithubID != "" {
		connectedPlatforms = append(connectedPlatforms, "github")
	}
	if user.WechatOpenID != nil && *user.WechatOpenID != "" {
		connectedPlatforms = append(connectedPlatforms, "wechat")
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "success",
		"data": gin.H{
			"username":            user.Username,
			"email":               user.Email,
			"avatar_url":          user.AvatarURL,
			"subscription_tier":   user.SubscriptionTier,
			"tokens_used":         user.TokensUsed,
			"token_limit":         tokenLimit,
			"connected_platforms": connectedPlatforms,
		},
	})
}

// UpdateProfile 更新个人信息
func (a *UserAPI) UpdateProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	var req struct {
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "invalid request body", "data": nil})
		return
	}

	if err := db.DB.Model(&model.User{}).Where("id = ?", uid).Update("username", req.Username).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "failed to update profile", "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "success", "data": gin.H{"username": req.Username}})
}

// UploadAvatar 上传头像
func (a *UserAPI) UploadAvatar(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	file, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "no file uploaded", "data": nil})
		return
	}

	// 2MB limit
	if file.Size > 2*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "file too large, max 2MB", "data": nil})
		return
	}

	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s_%d%s", uid.String(), time.Now().Unix(), ext)
	uploadDir := "./uploads/avatars"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "failed to create upload directory", "data": nil})
		return
	}

	savePath := filepath.Join(uploadDir, filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "failed to save file", "data": nil})
		return
	}

	avatarURL := "/uploads/avatars/" + filename
	if err := db.DB.Model(&model.User{}).Where("id = ?", uid).Update("avatar_url", avatarURL).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "failed to update user avatar", "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "success", "data": gin.H{"avatar_url": avatarURL}})
}

// TechStackStat 包含技术栈名称及次数
type TechStackStat struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// GetUserStats 获取用户 Dashboard 统计数据
func (a *UserAPI) GetUserStats(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	var user model.User
	if err := db.DB.First(&user, "id = ?", uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusNotFound, "message": "user not found", "data": nil})
		return
	}

	var totalArticles int64
	var totalWords int64

	// Exclude parent nodes (where parent_id is null) when counting articles
	db.DB.Model(&model.Blog{}).Where("user_id = ? AND status = 1 AND parent_id IS NOT NULL", uid).Count(&totalArticles)

	type Result struct {
		TotalWords int64
	}
	var res Result
	db.DB.Model(&model.Blog{}).Select("sum(word_count) as total_words").Where("user_id = ? AND status = 1 AND parent_id IS NOT NULL", uid).Scan(&res)
	totalWords = res.TotalWords

	var blogs []model.Blog
	// Exclude parent nodes when aggregating tech stacks
	db.DB.Where("user_id = ? AND status = 1 AND parent_id IS NOT NULL AND tech_stacks IS NOT NULL", uid).Find(&blogs)

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

	var techStackStats []TechStackStat
	for k, v := range stackMap {
		techStackStats = append(techStackStats, TechStackStat{Name: k, Count: v})
	}

	// 计算预估费用
	// 根据 DeepSeek V3 (deepseek-chat) 收费标准，粗略估算输入和输出混合：
	// 输入：缓存命中 0.2元/百万token，未命中 2元/百万token
	// 输出：3元/百万token
	// 此处采用一个平均经验值，比如假设 70% 是输入（未命中占大头），30% 是输出，综合单价约为：2.3元/百万token
	estimatedCost := (float64(user.TokensUsed) / 1000000.0) * 2.3

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "success",
		"data": gin.H{
			"tokens_used":      user.TokensUsed,
			"estimated_cost":   estimatedCost,
			"total_articles":   totalArticles,
			"total_words":      totalWords,
			"tech_stack_stats": techStackStats,
		},
	})
}
