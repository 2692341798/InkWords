package httpx

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	localWorkspaceContextKey = "local_workspace_id"
)

// LocalWorkspaceResolver resolves the one workspace controlled by the local installation.
// Implementations must not derive the value from an HTTP request.
type LocalWorkspaceResolver func(context.Context) (uuid.UUID, error)

// LocalWorkspaceContext resolves the local installation workspace and puts it in Gin context.
// Every local route uses this identity; request input cannot override it.
func LocalWorkspaceContext(resolve LocalWorkspaceResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		if resolve == nil {
			abortLocalWorkspaceUnavailable(c)
			return
		}

		workspaceID, err := resolve(c.Request.Context())
		if err != nil || workspaceID == uuid.Nil {
			abortLocalWorkspaceUnavailable(c)
			return
		}

		c.Set(localWorkspaceContextKey, workspaceID)
		c.Next()
	}
}

// GetLocalWorkspaceID returns the workspace previously resolved by LocalWorkspaceContext.
func GetLocalWorkspaceID(c *gin.Context) (uuid.UUID, error) {
	if c == nil {
		return uuid.Nil, errors.New("local workspace context is unavailable")
	}

	value, exists := c.Get(localWorkspaceContextKey)
	if !exists {
		return uuid.Nil, errors.New("local workspace context is unavailable")
	}

	workspaceID, ok := value.(uuid.UUID)
	if !ok || workspaceID == uuid.Nil {
		return uuid.Nil, errors.New("local workspace context is invalid")
	}
	return workspaceID, nil
}

func abortLocalWorkspaceUnavailable(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
		"code":    "LOCAL_WORKSPACE_UNAVAILABLE",
		"message": "本地工作区尚未准备完成，请检查数据库迁移后重试",
		"data":    nil,
	})
}
