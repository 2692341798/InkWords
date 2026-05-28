package api

import (
	"github.com/gin-gonic/gin"

	authdomain "inkwords-backend/internal/domain/auth"
)

type AuthAPI struct {
	authDomainHandler *authdomain.Handler
}

func NewAuthAPIWithDeps(authDomainHandler *authdomain.Handler) *AuthAPI {
	return &AuthAPI{
		authDomainHandler: authDomainHandler,
	}
}

// OAuthRedirect 重定向到第三方授权页面
func (a *AuthAPI) OAuthRedirect(c *gin.Context) {
	a.authDomainHandler.OAuthRedirect(c)
}

// OAuthCallback 第三方登录回调
func (a *AuthAPI) OAuthCallback(c *gin.Context) {
	a.authDomainHandler.OAuthCallback(c)
}

// Register 用户注册
func (a *AuthAPI) Register(c *gin.Context) {
	a.authDomainHandler.Register(c)
}

// BindGithub 绑定 GitHub 账号
func (a *AuthAPI) BindGithub(c *gin.Context) {
	a.authDomainHandler.BindGithub(c)
}
func (a *AuthAPI) GetCaptcha(c *gin.Context) {
	a.authDomainHandler.GetCaptcha(c)
}

// Login 用户登录
func (a *AuthAPI) Login(c *gin.Context) {
	a.authDomainHandler.Login(c)
}
