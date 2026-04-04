package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"inkwords-backend/internal/service"
)

type AuthAPI struct {
	authService *service.AuthService
}

func NewAuthAPI() *AuthAPI {
	return &AuthAPI{
		authService: service.NewAuthService(),
	}
}

// OAuthRedirect 重定向到第三方授权页面
func (a *AuthAPI) OAuthRedirect(c *gin.Context) {
	provider := c.Param("provider")

	url, err := a.authService.GetAuthURL(provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}

// OAuthCallback 第三方登录回调
func (a *AuthAPI) OAuthCallback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "code is required",
			"data":    nil,
		})
		return
	}

	token, user, err := a.authService.HandleCallback(provider, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "success",
		"data": gin.H{
			"token": token,
			"user": gin.H{
				"id":                user.ID,
				"username":          user.Username,
				"avatar_url":        user.AvatarURL,
				"subscription_tier": user.SubscriptionTier,
				"tokens_used":       user.TokensUsed,
			},
		},
	})
}
