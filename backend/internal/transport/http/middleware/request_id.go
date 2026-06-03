package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	requestIDHeader     = "X-Request-ID"
	requestIDContextKey = "request_id"
)

// RequestID ensures every request carries the same request id through headers and Gin context.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(requestIDHeader))
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Set(requestIDContextKey, requestID)
		c.Writer.Header().Set(requestIDHeader, requestID)
		c.Next()
	}
}

// GetRequestID returns the request id previously injected by RequestID.
func GetRequestID(c *gin.Context) string {
	if value, ok := c.Get(requestIDContextKey); ok {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return c.Writer.Header().Get(requestIDHeader)
}
