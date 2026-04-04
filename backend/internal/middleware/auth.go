package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"inkwords-backend/pkg/jwt"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "user_id"
)

// AuthMiddleware 创建身份验证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorizationHeader := c.GetHeader(authorizationHeaderKey)

		if len(authorizationHeader) == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "authorization header is not provided",
				"data":    nil,
			})
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "invalid authorization header format",
				"data":    nil,
			})
			return
		}

		authorizationType := strings.ToLower(fields[0])
		if authorizationType != authorizationTypeBearer {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "unsupported authorization type",
				"data":    nil,
			})
			return
		}

		accessToken := fields[1]
		claims, err := jwt.ParseToken(accessToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}

		// 将 user_id 存储在上下文中供后续处理使用
		c.Set(authorizationPayloadKey, claims.UserID)
		c.Next()
	}
}
