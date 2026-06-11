package handler

import (
	"errors"
	"time"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// userJSON is the canonical wire shape for a user object across all
// handlers (auth response, /v1/me, admin views).
func userJSON(u *models.User) gin.H {
	if u == nil {
		return nil
	}
	return gin.H{
		"id":           u.ID,
		"username":     u.Username,
		"phone":        u.Phone,
		"email":        u.Email,
		"level":        u.Level,
		"device_count": u.DeviceCount,
		"channel_type": u.ChannelType,
		"status":       u.Status,
		"created_at":   u.CreatedAt,
		"updated_at":   u.UpdatedAt,
	}
}

// writeUserErrorProblem maps service.UserService errors onto RFC 7807.
func writeUserErrorProblem(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUsernameTaken):
		ProblemConflict(c, "USERNAME_EXISTS", "Username is already taken")
	case errors.Is(err, service.ErrPhoneTaken):
		ProblemConflict(c, "PHONE_EXISTS", "Phone is already registered")
	case errors.Is(err, service.ErrEmailTaken):
		ProblemConflict(c, "EMAIL_EXISTS", "Email is already registered")
	case errors.Is(err, service.ErrPhoneInvalid):
		ProblemBadRequest(c, "PHONE_INVALID", "Invalid phone format")
	case errors.Is(err, service.ErrPasswordWeak):
		ProblemBadRequest(c, "PASSWORD_WEAK", "Password must be ≥8 chars and contain a letter and a digit")
	case errors.Is(err, service.ErrBadCredentials):
		ProblemUnauthorized(c, ProblemCodeInvalidCredentials, "Invalid credentials")
	case errors.Is(err, service.ErrAccountDisabled):
		ProblemForbidden(c, ProblemCodeAccountDisabled, "Account is disabled")
	case errors.Is(err, service.ErrUserNotFound):
		ProblemNotFound(c, "USER_NOT_FOUND", "User not found")
	case errors.Is(err, service.ErrOldPasswordWrong):
		ProblemBadRequest(c, "PASSWORD_WRONG", "Old password is incorrect")
	default:
		ProblemInternal(c, err.Error())
	}
}

// writeSmsProblem maps SmsService errors to RFC 7807.
func writeSmsProblem(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrSmsDisabled):
		ProblemConflict(c, ProblemCodeSmsDisabled, "SMS service not configured")
	case errors.Is(err, service.ErrSmsRateLimit):
		ProblemTooManyRequests(c, ProblemCodeSmsRateLimit, "SMS send rate exceeded; retry later", 60)
	case errors.Is(err, service.ErrSmsDaily):
		ProblemTooManyRequests(c, ProblemCodeSmsRateLimit, "Daily SMS quota exceeded", 0)
	case errors.Is(err, service.ErrSmsCodeExpired):
		ProblemBadRequest(c, ProblemCodeSmsCodeExpire, "SMS code expired")
	case errors.Is(err, service.ErrSmsCodeWrong):
		ProblemBadRequest(c, ProblemCodeSmsCodeBad, "SMS code mismatch")
	case errors.Is(err, service.ErrSmsCodeAttempts):
		ProblemForbidden(c, ProblemCodeTooManyAttempts, "Too many SMS verification attempts")
	default:
		ProblemInternal(c, err.Error())
	}
}

// parseTime accepts a small handful of RFC3339-ish timestamp shapes used by
// our cursor payloads. Returns the zero Time on failure so callers can
// silently skip an invalid bound.
func parseTime(raw string) (t time.Time, err error) {
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
	} {
		if t, err = time.Parse(layout, raw); err == nil {
			return t, nil
		}
	}
	return time.Time{}, err
}

// hashPassword is a thin bcrypt wrapper used by admin handlers that set a
// user password directly (no policy check; callers run ValidatePassword
// first).
func hashPassword(p string) string {
	b, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(b)
}
