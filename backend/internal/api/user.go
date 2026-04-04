package api

import (
	"net/http"

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
			"subscription_tier":   user.SubscriptionTier,
			"tokens_used":         user.TokensUsed,
			"token_limit":         tokenLimit,
			"connected_platforms": connectedPlatforms,
		},
	})
}
