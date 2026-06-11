package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware returns a Gin middleware that sets CORS headers dynamically.
// It reads allowed origins from the DynamicSettingsProvider on each request,
// so changes in the admin panel take effect immediately.
func CORSMiddleware(settings DynamicSettingsProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := strings.TrimRight(strings.ToLower(c.GetHeader("Origin")), "/")
		if origin == "" {
			c.Next()
			return
		}

		origins := settings.GetAllowedOrigins()
		if len(origins) == 0 {
			c.Next()
			return
		}

		allowed := false
		for _, o := range origins {
			if strings.TrimRight(strings.ToLower(o), "/") == origin {
				allowed = true
				break
			}
		}

		if !allowed {
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Origin", c.GetHeader("Origin"))
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "X-API-Key, Authorization, Content-Type")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
