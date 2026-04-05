package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func RequestContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}

		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			userID = "demo-user"
		}

		sessionID := c.GetHeader("X-Session-ID")
		if sessionID == "" {
			sessionID = c.Param("session_id")
		}

		c.Set("request_id", requestID)
		c.Set("user_id", userID)
		if sessionID != "" {
			c.Set("session_id", sessionID)
		}
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}
