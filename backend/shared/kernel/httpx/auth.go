package httpx

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"inkwords-backend/pkg/jwt"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "user_id"
)

// AuthMiddleware validates bearer JWTs and exposes user_id in Gin context.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorizationHeader := c.GetHeader(authorizationHeaderKey)

		if len(authorizationHeader) == 0 {
			// Why: local Docker testing needs a stable existing user so quota checks
			// and persisted data keep the same owner across requests.
			if configuredID := strings.TrimSpace(os.Getenv("DEV_AUTH_USER_ID")); configuredID != "" {
				userID, err := uuid.Parse(configuredID)
				if err != nil {
					log.Printf("AuthMiddleware: invalid DEV_AUTH_USER_ID for path %s", c.Request.URL.Path)
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
						"code":    http.StatusInternalServerError,
						"message": "development authentication is misconfigured",
						"data":    nil,
					})
					return
				}
				c.Set(authorizationPayloadKey, userID)
				c.Next()
				return
			}
			log.Printf("AuthMiddleware: empty header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "authorization header is not provided",
				"data":    nil,
			})
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			log.Printf("AuthMiddleware: invalid header format: %s", authorizationHeader)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "invalid authorization header format",
				"data":    nil,
			})
			return
		}

		authorizationType := strings.ToLower(fields[0])
		if authorizationType != authorizationTypeBearer {
			log.Printf("AuthMiddleware: unsupported auth type: %s", authorizationType)
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
			log.Printf("AuthMiddleware: ParseToken failed: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}

		c.Set(authorizationPayloadKey, claims.UserID)
		c.Next()
	}
}
