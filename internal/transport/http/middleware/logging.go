package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"knowflow/internal/platform/observability"
)

func Logging(logger *observability.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		entry := slog.Default()
		if logger != nil {
			entry = logger.Logger
		}

		entry.Info("http request",
			"path", c.FullPath(),
			"request_id", c.GetString("request_id"),
			"user_id", c.GetString("user_id"),
			"session_id", c.GetString("session_id"),
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
		)
	}
}
