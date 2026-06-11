package handler

import (
	"quickdesk/signaling/internal/httpx"

	"github.com/gin-gonic/gin"
)

// Re-export the RFC 7807 helpers from httpx so handler code can call them
// without an extra import. Keeps the old "ProblemXxx" spelling that the
// rest of the handler package already uses.

// Error code aliases.
const (
	ProblemCodeValidationFailed = httpx.CodeValidationFailed
	ProblemCodeInvalidRequest   = httpx.CodeInvalidRequest
	ProblemCodeUnauthorized     = httpx.CodeUnauthorized
	ProblemCodeForbidden        = httpx.CodeForbidden
	ProblemCodeNotFound         = httpx.CodeNotFound
	ProblemCodeConflict         = httpx.CodeConflict
	ProblemCodeRateLimited      = httpx.CodeRateLimited
	ProblemCodeInternalError    = httpx.CodeInternalError

	ProblemCodeTokenExpired       = httpx.CodeTokenExpired
	ProblemCodeTokenInvalid       = httpx.CodeTokenInvalid
	ProblemCodeRefreshInvalid     = httpx.CodeRefreshInvalid
	ProblemCodeInvalidCredentials = httpx.CodeInvalidCredentials
	ProblemCodeAccountDisabled    = httpx.CodeAccountDisabled

	ProblemCodeDeviceNotFound   = httpx.CodeDeviceNotFound
	ProblemCodeHostOffline      = httpx.CodeHostOffline
	ProblemCodeInvalidCode      = httpx.CodeInvalidCode
	ProblemCodeTooManyAttempts  = httpx.CodeTooManyAttempts
	ProblemCodeDeviceSecretBad  = httpx.CodeDeviceSecretBad
	ProblemCodeProvisionFailed  = httpx.CodeProvisionFailed
	ProblemCodePeerDisconnected = httpx.CodePeerDisconnected

	ProblemCodeSmsDisabled   = httpx.CodeSmsDisabled
	ProblemCodeSmsRateLimit  = httpx.CodeSmsRateLimit
	ProblemCodeSmsCodeBad    = httpx.CodeSmsCodeBad
	ProblemCodeSmsCodeExpire = httpx.CodeSmsCodeExpire
)

func WriteProblem(c *gin.Context, status int, code, title, detail string) {
	httpx.WriteProblem(c, status, code, title, detail)
}
func WriteProblemJSON(c *gin.Context, status int, code, title, detail string, extra gin.H) {
	httpx.WriteProblemJSON(c, status, code, title, detail, extra)
}
func ProblemBadRequest(c *gin.Context, code, detail string)    { httpx.BadRequest(c, code, detail) }
func ProblemUnauthorized(c *gin.Context, code, detail string)  { httpx.Unauthorized(c, code, detail) }
func ProblemForbidden(c *gin.Context, code, detail string)     { httpx.Forbidden(c, code, detail) }
func ProblemNotFound(c *gin.Context, code, detail string)      { httpx.NotFound(c, code, detail) }
func ProblemConflict(c *gin.Context, code, detail string)      { httpx.Conflict(c, code, detail) }
func ProblemTooManyRequests(c *gin.Context, code, detail string, retry int) {
	httpx.TooManyRequests(c, code, detail, retry)
}
func ProblemInternal(c *gin.Context, detail string) { httpx.Internal(c, detail) }
