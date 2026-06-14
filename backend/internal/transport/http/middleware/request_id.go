package middleware

import (
	"github.com/gin-gonic/gin"

	"inkwords-backend/shared/kernel/httpx"
)

const (
	requestIDHeader     = "X-Request-ID"
	requestIDContextKey = "request_id"
)

// RequestID ensures every request carries the same request id through headers and Gin context.
func RequestID() gin.HandlerFunc {
	return httpx.RequestID()
}

// GetRequestID returns the request id previously injected by RequestID.
func GetRequestID(c *gin.Context) string {
	return httpx.GetRequestID(c)
}
