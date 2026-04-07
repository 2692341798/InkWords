package api

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

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

	authURL, err := a.authService.GetAuthURL(provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// OAuthCallback 第三方登录回调
func (a *AuthAPI) OAuthCallback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	// Ensure no trailing slash for the base URL to prevent double slashes
	if len(frontendURL) > 0 && frontendURL[len(frontendURL)-1] == '/' {
		frontendURL = frontendURL[:len(frontendURL)-1]
	}

	if code == "" {
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s/?error=%s", frontendURL, url.QueryEscape("code is required")))
		return
	}

	token, _, err := a.authService.HandleCallback(c.Request.Context(), provider, code)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s/?error=%s", frontendURL, url.QueryEscape(err.Error())))
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s/?token=%s", frontendURL, token))
}

// Register 用户注册
func (a *AuthAPI) Register(c *gin.Context) {
	var req struct {
		Username     string `json:"username" binding:"required"`
		Email        string `json:"email" binding:"required,email"`
		Password     string `json:"password" binding:"required"`
		CaptchaID    string `json:"captcha_id" binding:"required"`
		CaptchaValue string `json:"captcha_value" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数格式错误"})
		return
	}

	token, err := a.authService.Register(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "注册成功", "data": gin.H{"token": token}})
}

// GetCaptcha 获取图形验证码
func (a *AuthAPI) GetCaptcha(c *gin.Context) {
	id, b64s, err := a.authService.GenerateCaptcha()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "验证码生成失败",
			"data":    nil,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "success",
		"data": gin.H{
			"captcha_id": id,
			"image":      b64s,
		},
	})
}

// Login 用户登录
func (a *AuthAPI) Login(c *gin.Context) {
	var req struct {
		Email        string `json:"email" binding:"required,email"`
		Password     string `json:"password" binding:"required"`
		CaptchaID    string `json:"captcha_id"`
		CaptchaValue string `json:"captcha_value"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	token, user, err := a.authService.Login(req.Email, req.Password, req.CaptchaID, req.CaptchaValue)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    http.StatusUnauthorized,
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
