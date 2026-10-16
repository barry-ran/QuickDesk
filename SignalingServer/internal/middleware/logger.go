package middleware

import (
	"time"

	"quickdesk/signaling/internal/observability"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware emits only failed or slow HTTP requests. Successful hot
// paths such as host heartbeat are intentionally omitted to keep production
// logs useful during incident analysis.
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
			// FullPath is the matched pattern; it is empty for a request that
			// matched nothing, which the metrics service records as
			// "unmatched" instead of inventing a series per bogus path.
			metrics.RecordHTTPRequest(method, path, c.FullPath(), statusCode)
		}

		if statusCode >= 400 || latency >= time.Second {
			requestID, _ := c.Get("request_id")
			observability.Event("http", "request", map[string]interface{}{
				"client_ip":  c.ClientIP(),
				"latency_ms": latency.Milliseconds(),
				"method":     method,
				"path":       path,
				"request_id": requestID,
				"status":     statusCode,
			})
		}
	}
}
