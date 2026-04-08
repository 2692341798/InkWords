package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"inkwords-backend/internal/service"
)

type UserAPI struct {
	userService *service.UserService
}

// NewUserAPI 创建 UserAPI 实例
func NewUserAPI(userService *service.UserService) *UserAPI {
	return &UserAPI{
		userService: userService,
	}
}

// GetProfile 获取个人中心配置与额度
func (a *UserAPI) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    http.StatusUnauthorized,
			"message": "未授权的访问",
			"data":    nil,
		})
		return
	}

	uid, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "用户 ID 类型错误",
			"data":    nil,
		})
		return
	}

	user, err := a.userService.GetUserByID(uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    http.StatusNotFound,
			"message": "用户不存在",
			"data":    nil,
		})
		return
	}

	// 读取数据库中的 TokenLimit，如果没有设置，则默认 100000
	tokenLimit := user.TokenLimit
	if tokenLimit == 0 {
		tokenLimit = 100000
	}

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
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "请求参数格式错误", "data": nil})
		return
	}

	if len(req.Username) < 2 || len(req.Username) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "用户名长度必须在 2 到 20 个字符之间", "data": nil})
		return
	}

	if err := a.userService.UpdateUsername(uid, req.Username); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "更新配置失败", "data": nil})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "未上传文件", "data": nil})
		return
	}

	// 2MB limit
	if file.Size > 2*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "文件过大，最大限制 2MB", "data": nil})
		return
	}

	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s_%d%s", uid.String(), time.Now().Unix(), ext)
	uploadDir := "./uploads/avatars"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "创建上传目录失败", "data": nil})
		return
	}

	savePath := filepath.Join(uploadDir, filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "保存文件失败", "data": nil})
		return
	}

	avatarURL := "/uploads/avatars/" + filename
	if err := a.userService.UpdateAvatarURL(uid, avatarURL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "更新用户头像失败", "data": nil})
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

	user, err := a.userService.GetUserByID(uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusNotFound, "message": "用户不存在", "data": nil})
		return
	}

	totalArticles, totalWords, stackMap, err := a.userService.GetUserStats(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "获取统计数据失败", "data": nil})
		return
	}

	var techStackStats []TechStackStat
	for k, v := range stackMap {
		techStackStats = append(techStackStats, TechStackStat{Name: k, Count: v})
	}

	sort.Slice(techStackStats, func(i, j int) bool {
		return techStackStats[i].Count > techStackStats[j].Count
	})

	if len(techStackStats) > 20 {
		techStackStats = techStackStats[:20]
	}

	// 计算预估费用
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
