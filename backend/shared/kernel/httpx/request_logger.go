package httpx

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
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

	logger := slog.New(slog.NewJSONHandler(writer, nil))
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		logger.Info(
			"request_completed",
			"service", serviceName,
			"request_id", GetRequestID(c),
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(startedAt).Milliseconds(),
		)
	}
}
