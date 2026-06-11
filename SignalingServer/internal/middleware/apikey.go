package middleware

import (
	"strings"

	"quickdesk/signaling/internal/httpx"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

// DynamicSettingsProvider is the interface APIKeyAuth needs to read live settings.
type DynamicSettingsProvider interface {
	GetAPIKey() string
	GetAllowedOrigins() []string
}

type APIKeyAuth struct {
	settings DynamicSettingsProvider
}

func NewAPIKeyAuth(settings DynamicSettingsProvider) *APIKeyAuth {
	return &APIKeyAuth{settings: settings}
}

// NewAPIKeyAuthStatic creates an APIKeyAuth with static values (for tests or fallback).
func NewAPIKeyAuthStatic(apiKey string, allowedOrigins []string) *APIKeyAuth {
	return &APIKeyAuth{settings: &staticProvider{apiKey: apiKey, origins: allowedOrigins}}
}

func (a *APIKeyAuth) Enabled() bool {
	return a.settings.GetAPIKey() != "" || len(a.settings.GetAllowedOrigins()) > 0
}

func (a *APIKeyAuth) validateAPIKey(c *gin.Context) bool {
	apiKey := a.settings.GetAPIKey()
	if apiKey == "" {
		return false
	}
	clientKey := c.GetHeader("X-API-Key")
	if clientKey == "" {
		clientKey = c.Query("api_key")
	}
	return clientKey == apiKey
}

func (a *APIKeyAuth) validateOrigin(c *gin.Context) bool {
	origins := a.settings.GetAllowedOrigins()
	if len(origins) == 0 {
		return false
	}
	reqOrigin := strings.TrimRight(strings.ToLower(c.GetHeader("Origin")), "/")
	if reqOrigin == "" {
		return false
	}
	for _, o := range origins {
		if strings.TrimRight(strings.ToLower(o), "/") == reqOrigin {
			return true
		}
	}
	return false
}

func (a *APIKeyAuth) Required() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !a.Enabled() {
			c.Next()
			return
		}
		if a.validateAPIKey(c) || a.validateOrigin(c) {
			c.Next()
			return
		}
		httpx.Forbidden(c, httpx.CodeForbidden, "Invalid or missing API key / origin not allowed")
	}
}

func (a *APIKeyAuth) ValidateRequest(c *gin.Context) bool {
	if !a.Enabled() {
		return true
	}
	return a.validateAPIKey(c) || a.validateOrigin(c)
}

// Ensure SettingsService satisfies the interface at compile time.
var _ DynamicSettingsProvider = (*service.SettingsService)(nil)

type staticProvider struct {
	apiKey  string
	origins []string
}

func (p *staticProvider) GetAPIKey() string          { return p.apiKey }
func (p *staticProvider) GetAllowedOrigins() []string { return p.origins }
