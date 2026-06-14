package middleware

import (
	"io"
	"os"

	"github.com/gin-gonic/gin"

	"inkwords-backend/shared/kernel/httpx"
)

// RequestLogger emits one structured access log entry after each HTTP request completes.
func RequestLogger(serviceName string) gin.HandlerFunc {
	return RequestLoggerWithWriter(serviceName, os.Stdout)
}

// RequestLoggerWithWriter allows tests to capture structured request logs without touching global stdout.
func RequestLoggerWithWriter(serviceName string, writer io.Writer) gin.HandlerFunc {
	if writer == nil {
		writer = os.Stdout
	}

	return httpx.RequestLoggerWithWriter(serviceName, writer)
}
