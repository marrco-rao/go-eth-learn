package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"ntf-auction-backend/internal/logger"
)

func RequestLogger(l *logger.DBLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		cost := time.Since(start)
		message := fmt.Sprintf("request completed in %s", cost.String())
		if status >= 500 {
			l.Error(c.Request.Context(), message, c.Request.Method, c.FullPath(), status)
			return
		}
		l.Info(c.Request.Context(), message, c.Request.Method, c.FullPath(), status)
	}
}
