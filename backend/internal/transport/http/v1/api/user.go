package api

import (
	"github.com/gin-gonic/gin"
	userdomain "inkwords-backend/internal/domain/user"
	"inkwords-backend/internal/service"
)

// UserAPI adapts user-related HTTP routes onto the user domain handler.
type UserAPI struct {
	userService       *service.UserService
	userDomainHandler *userdomain.Handler
}

// GetProfile 获取个人中心配置与额度
func (a *UserAPI) GetProfile(c *gin.Context) {
	a.userDomainHandler.GetProfile(c)
}

// UpdateProfile 更新个人信息
func (a *UserAPI) UpdateProfile(c *gin.Context) {
	a.userDomainHandler.UpdateProfile(c)
}

// UploadAvatar 上传头像
func (a *UserAPI) UploadAvatar(c *gin.Context) {
	a.userDomainHandler.UploadAvatar(c)
}

// GetUserStats 获取用户 Dashboard 统计数据
func (a *UserAPI) GetUserStats(c *gin.Context) {
	a.userDomainHandler.GetUserStats(c)
}

// GetPromptSettings proxies prompt-settings reads to the user domain handler.
func (a *UserAPI) GetPromptSettings(c *gin.Context) {
	a.userDomainHandler.GetPromptSettings(c)
}

// UpdatePromptSettings proxies prompt-settings updates to the user domain handler.
func (a *UserAPI) UpdatePromptSettings(c *gin.Context) {
	a.userDomainHandler.UpdatePromptSettings(c)
}

// NewUserAPIWithDeps creates a UserAPI with explicitly injected dependencies.
func NewUserAPIWithDeps(userService *service.UserService, userDomainHandler *userdomain.Handler) *UserAPI {
	return &UserAPI{
		userService:       userService,
		userDomainHandler: userDomainHandler,
	}
}
