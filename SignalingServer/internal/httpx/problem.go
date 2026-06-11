// Package httpx holds leaf-level HTTP helpers shared by both the handler
// and middleware packages. Nothing in here may import other internal
// packages — that keeps middleware → httpx → (nothing) acyclic.
package httpx

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RFC 7807 problem details helpers. See docs §2.3.

const problemTypeBase = "https://quickdesk.io/problems/"

// Stable machine-readable error codes. Keep in sync with docs §2.15.
const (
	// Generic
	CodeValidationFailed = "VALIDATION_FAILED"
	CodeInvalidRequest   = "INVALID_REQUEST"
	CodeUnauthorized     = "UNAUTHORIZED"
	CodeForbidden        = "FORBIDDEN"
	CodeNotFound         = "NOT_FOUND"
	CodeConflict         = "CONFLICT"
	CodeRateLimited      = "RATE_LIMITED"
	CodeInternalError    = "INTERNAL_ERROR"

	// Auth
	CodeTokenExpired       = "TOKEN_EXPIRED"
	CodeTokenInvalid       = "TOKEN_INVALID"
	CodeRefreshInvalid     = "REFRESH_INVALID"
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
	CodeAccountDisabled    = "ACCOUNT_DISABLED"

	// Device / signaling
	CodeDeviceNotFound   = "DEVICE_NOT_FOUND"
	CodeHostOffline      = "HOST_OFFLINE"
	CodeInvalidCode      = "INVALID_CODE"
	CodeTooManyAttempts  = "TOO_MANY_ATTEMPTS"
	CodeDeviceSecretBad  = "DEVICE_SECRET_INVALID"
	CodeProvisionFailed  = "PROVISION_FAILED"
	CodePeerDisconnected = "PEER_DISCONNECTED"

	// SMS
	CodeSmsDisabled   = "SMS_DISABLED"
	CodeSmsRateLimit  = "SMS_RATE_LIMIT"
	CodeSmsCodeBad    = "SMS_CODE_INVALID"
	CodeSmsCodeExpire = "SMS_CODE_EXPIRED"
)

// Problem is the RFC 7807 wire payload with our extensions.
type Problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
	Code     string `json:"code"`
	TraceID  string `json:"trace_id,omitempty"`
}

// WriteProblem aborts the request with a problem+json response.
func WriteProblem(c *gin.Context, status int, code, title, detail string) {
	WriteProblemJSON(c, status, code, title, detail, nil)
}

func WriteProblemJSON(c *gin.Context, status int, code, title, detail string, extra gin.H) {
	p := Problem{
		Type:     problemTypeBase + slugifyCode(code),
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: c.Request.URL.Path,
		Code:     code,
		TraceID:  c.GetString("request_id"),
	}
	body := gin.H{
		"type":     p.Type,
		"title":    p.Title,
		"status":   p.Status,
		"code":     p.Code,
		"trace_id": p.TraceID,
		"instance": p.Instance,
	}
	if p.Detail != "" {
		body["detail"] = p.Detail
	}
	for k, v := range extra {
		body[k] = v
	}
	c.Header("Content-Type", "application/problem+json")
	c.AbortWithStatusJSON(status, body)
}

// Shorthands per common statuses.

func BadRequest(c *gin.Context, code, detail string) {
	WriteProblem(c, http.StatusBadRequest, code, "Bad request", detail)
}

func Unauthorized(c *gin.Context, code, detail string) {
	WriteProblem(c, http.StatusUnauthorized, code, "Unauthorized", detail)
}

func Forbidden(c *gin.Context, code, detail string) {
	WriteProblem(c, http.StatusForbidden, code, "Forbidden", detail)
}

func NotFound(c *gin.Context, code, detail string) {
	WriteProblem(c, http.StatusNotFound, code, "Not found", detail)
}

func Conflict(c *gin.Context, code, detail string) {
	WriteProblem(c, http.StatusConflict, code, "Conflict", detail)
}

func TooManyRequests(c *gin.Context, code, detail string, retryAfterSec int) {
	if retryAfterSec > 0 {
		c.Header("Retry-After", fmt.Sprintf("%d", retryAfterSec))
	}
	WriteProblem(c, http.StatusTooManyRequests, code, "Too many requests", detail)
}

func Internal(c *gin.Context, detail string) {
	WriteProblem(c, http.StatusInternalServerError, CodeInternalError, "Internal server error", detail)
}

func slugifyCode(code string) string {
	out := make([]byte, 0, len(code))
	for i := 0; i < len(code); i++ {
		ch := code[i]
		switch {
		case ch >= 'A' && ch <= 'Z':
			out = append(out, ch-'A'+'a')
		case ch == '_':
			out = append(out, '-')
		default:
			out = append(out, ch)
		}
	}
	return string(out)
}
