package middleware

import (
	"net"
	"strings"

	"quickdesk/signaling/internal/httpx"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

// IPWhitelistMiddleware restricts admin traffic to a whitelist of exact
// IPs / CIDR blocks stored in settings.AdminIPWhitelist (newline-separated).
// An empty whitelist disables the check entirely. All rejections are
// emitted as RFC 7807 problem details per §2.3.
func IPWhitelistMiddleware(settingsService *service.SettingsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		settings := settingsService.Get()
		whitelist := strings.TrimSpace(settings.AdminIPWhitelist)
		if whitelist == "" {
			c.Next()
			return
		}

		clientIP := net.ParseIP(c.ClientIP())
		if clientIP == nil {
			httpx.Forbidden(c, httpx.CodeForbidden, "Invalid client IP")
			return
		}

		for _, entry := range strings.Split(whitelist, "\n") {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}
			if strings.Contains(entry, "/") {
				_, cidr, err := net.ParseCIDR(entry)
				if err == nil && cidr.Contains(clientIP) {
					c.Next()
					return
				}
				continue
			}
			if net.ParseIP(entry) != nil && entry == clientIP.String() {
				c.Next()
				return
			}
		}

		httpx.Forbidden(c, httpx.CodeForbidden, "Client IP not allowed")
	}
}
