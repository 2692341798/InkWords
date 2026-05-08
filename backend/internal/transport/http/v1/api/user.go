package api

import (
	"github.com/gin-gonic/gin"
	userdomain "inkwords-backend/internal/domain/user"
	"inkwords-backend/internal/infra/db"
	"inkwords-backend/internal/service"
)

type UserAPI struct {
	userService       *service.UserService
	userDomainHandler *userdomain.Handler
}

// NewUserAPI 创建 UserAPI 实例
func NewUserAPI(userService *service.UserService) *UserAPI {
	repo := userdomain.NewGormRepository(db.DB)
	domainService := userdomain.NewService(repo)

	return &UserAPI{
		userService:       userService,
		userDomainHandler: userdomain.NewHandler(domainService),
	}
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

func NewUserAPIWithDeps(userService *service.UserService, userDomainHandler *userdomain.Handler) *UserAPI {
	return &UserAPI{
		userService:       userService,
		userDomainHandler: userDomainHandler,
	}
}
