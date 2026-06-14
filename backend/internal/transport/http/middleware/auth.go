package middleware

import (
	"github.com/gin-gonic/gin"

	"inkwords-backend/shared/kernel/httpx"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "user_id"
)

// AuthMiddleware 创建身份验证中间件
func AuthMiddleware() gin.HandlerFunc {
	return httpx.AuthMiddleware()
}
