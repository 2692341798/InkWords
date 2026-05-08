package auth

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) OAuthRedirect(c *gin.Context) {
	provider := c.Param("provider")

	authURL, err := h.service.GetAuthURL(provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": err.Error(), "data": nil})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

func (h *Handler) OAuthCallback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	if len(frontendURL) > 0 && frontendURL[len(frontendURL)-1] == '/' {
		frontendURL = frontendURL[:len(frontendURL)-1]
	}

	if code == "" {
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s/?error=%s", frontendURL, url.QueryEscape("code is required")))
		return
	}

	token, user, err := h.service.HandleCallback(c.Request.Context(), provider, code)
	if err != nil {
		if errors.Is(err, ErrEmailExistsBindRequired) && user != nil {
			ghID := ""
			if user.GithubID != nil {
				ghID = *user.GithubID
			}
			redirectURL := fmt.Sprintf("%s/?bind_required=true&email=%s&github_id=%s&avatar_url=%s&username=%s",
				frontendURL,
				url.QueryEscape(user.Email),
				url.QueryEscape(ghID),
				url.QueryEscape(user.AvatarURL),
				url.QueryEscape(user.Username))
			c.Redirect(http.StatusTemporaryRedirect, redirectURL)
			return
		}
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s/?error=%s", frontendURL, url.QueryEscape(err.Error())))
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s/?token=%s", frontendURL, token))
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数格式错误"})
		return
	}

	token, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "注册成功", "data": gin.H{"token": token}})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": err.Error(), "data": nil})
		return
	}

	token, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, req.CaptchaID, req.CaptchaValue)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": err.Error(), "data": nil})
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

func (h *Handler) BindGithub(c *gin.Context) {
	var req BindGithubRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "message": "参数格式错误", "data": nil})
		return
	}

	token, user, err := h.service.BindGithub(c.Request.Context(), req.Email, req.Password, req.GithubID, req.Username, req.AvatarURL)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": err.Error(), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "绑定成功",
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

func (h *Handler) GetCaptcha(c *gin.Context) {
	id, b64s, err := h.service.GenerateCaptcha()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": http.StatusInternalServerError, "message": "验证码生成失败", "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "success", "data": gin.H{"captcha_id": id, "image": b64s}})
}

