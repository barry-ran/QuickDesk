package middleware

import (
	"log"
	"time"

	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware logs HTTP requests
func LoggerMiddleware(metrics *service.MetricsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Process request
		c.Next()

		// Log after processing
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		if metrics != nil {
			metrics.RecordHTTPRequest(path)
		}

		log.Printf("[%s] %s %s - %d (%v)",
			method, path, c.ClientIP(), statusCode, latency)
	}
}
