package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminUsersHandler serves /v1/admin/users/* — CRUD over business users
// plus the batch / session-revoke / device-count endpoints.
type AdminUsersHandler struct {
	users    *service.UserService
	tokens   *service.TokenService
	bus      *service.EventBus
	audit    *service.AuditService
	presence *service.PresenceService
	db       *gorm.DB
}

func NewAdminUsersHandler(
	users *service.UserService,
	tokens *service.TokenService,
	bus *service.EventBus,
	audit *service.AuditService,
	presence *service.PresenceService,
	db *gorm.DB,
) *AdminUsersHandler {
	return &AdminUsersHandler{users: users, tokens: tokens, bus: bus, audit: audit, presence: presence, db: db}
}

// List handles GET /v1/admin/users with cursor-based pagination (§3.1).
// Supported filters: search, level, status, channel_type.
func (h *AdminUsersHandler) List(c *gin.Context) {
	p := ParseCursor(c)
	allowedSorts := map[string]bool{
		"id": true, "created_at": true, "updated_at": true,
		"username": true, "level": true, "device_count": true,
	}
	if !allowedSorts[p.Sort] {
		p.Sort = "id"
	}

	cur, err := DecodeCursor(p.Cursor)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}

	var statusFilter *bool
	if s := c.Query("status"); s == "true" || s == "false" {
		v := s == "true"
		statusFilter = &v
	}

	// +1 lookahead to know whether there's a next page.
	users, total, err := h.users.AdminList(c.Request.Context(), service.UserAdminListParams{
		AfterID:     cur.OffsetID,
		Limit:       p.Limit + 1,
		Sort:        p.Sort,
		Order:       p.Order,
		Search:      p.Search,
		Level:       c.Query("level"),
		Status:      statusFilter,
		ChannelType: c.Query("channel_type"),
	})
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	next := ""
	if len(users) > p.Limit {
		last := users[p.Limit-1]
		next = EncodeCursor(CursorPayload{OffsetID: last.ID})
		users = users[:p.Limit]
	}
	c.JSON(http.StatusOK, NewCursorPage(users, next, total))
}

// Get handles GET /v1/admin/users/:id.
func (h *AdminUsersHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid id")
		return
	}
	u, err := h.users.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		ProblemNotFound(c, ProblemCodeNotFound, "User not found")
		return
	}
	c.JSON(http.StatusOK, u)
}

// GetDetails bundles the user profile with their active devices +
// sessions + recent connection history (§2.2 admin users/:id/details).
//
// `devices` is enriched: we join UserDevice (binding metadata: remark,
// first_bound_at, last_connect_at, connect_count) with Device (hardware
// metadata: device_uuid, os, os_version, app_version, device_name,
// access_code, last_seen_at) and presence (online, logged_in derived).
// The admin web's UserDetailPage table reads device_uuid / os / online /
// last_seen_at from this enriched shape.
func (h *AdminUsersHandler) GetDetails(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid id")
		return
	}
	u, err := h.users.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		ProblemNotFound(c, ProblemCodeNotFound, "User not found")
		return
	}
	var userDevices []models.UserDevice
	h.db.WithContext(c.Request.Context()).
		Where("user_id = ? AND status = ?", u.ID, true).
		Find(&userDevices)

	// Enrich each binding with the live Device row + presence.
	deviceIDs := make([]string, 0, len(userDevices))
	for _, ud := range userDevices {
		deviceIDs = append(deviceIDs, ud.DeviceID)
	}
	devicesByID := map[string]*models.Device{}
	if len(deviceIDs) > 0 {
		var devs []models.Device
		h.db.WithContext(c.Request.Context()).
			Where("device_id IN ?", deviceIDs).
			Find(&devs)
		for i := range devs {
			devicesByID[devs[i].DeviceID] = &devs[i]
		}
	}
	online := map[string]bool{}
	if h.presence != nil && len(deviceIDs) > 0 {
		online = h.presence.BulkOnline(c.Request.Context(), deviceIDs)
	}
	enrichedDevices := make([]gin.H, 0, len(userDevices))
	for _, ud := range userDevices {
		row := gin.H{
			"device_id":        ud.DeviceID,
			"remark":           ud.Remark,
			"first_bound_at":   ud.FirstBoundAt,
			"last_connect_at":  ud.LastConnectAt,
			"connect_count":    ud.ConnectCount,
			"status":           ud.Status,
		}
		if d := devicesByID[ud.DeviceID]; d != nil {
			row["device_uuid"]  = d.DeviceUUID
			row["device_name"] = d.DeviceName
			row["os"]           = d.OS
			row["os_version"]   = d.OSVersion
			row["app_version"]  = d.AppVersion
			row["access_code"]  = d.AccessCode
			row["last_seen_at"] = d.LastSeenAt
			isOnline := online[d.DeviceID]
			row["online"]       = isOnline
			row["logged_in"]    = d.LoggedIn && isOnline
		}
		enrichedDevices = append(enrichedDevices, row)
	}

	var history []models.ConnectionHistory
	h.db.WithContext(c.Request.Context()).
		Where("user_id = ?", u.ID).
		Order("created_at DESC").
		Limit(50).
		Find(&history)

	// Active sessions (refresh-token families) with their captured
	// user-agent / IP / timestamps.
	sessions := make([]gin.H, 0)
	if views, err := h.tokens.ListSessionsWithMeta(c.Request.Context(), service.ScopeUser, u.ID); err == nil {
		for _, v := range views {
			sessions = append(sessions, gin.H{
				"id":         v.FamilyID,
				"user_agent": v.Meta.UserAgent,
				"ip":         v.Meta.IP,
				"last_seen":  v.Meta.LastSeen,
				"created_at": v.Meta.CreatedAt,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"user":               u,
		"devices":            enrichedDevices,
		"sessions":           sessions,
		"connection_history": history,
	})
}

// Create handles POST /v1/admin/users.
type adminCreateUserReq struct {
	Username    string `json:"username" binding:"required"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
	Password    string `json:"password" binding:"required"`
	Level       string `json:"level"`
	ChannelType string `json:"channel_type"`
}

func (h *AdminUsersHandler) Create(c *gin.Context) {
	var req adminCreateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	u, err := h.users.Register(c.Request.Context(), service.RegisterInput{
		Username: req.Username,
		Password: req.Password,
		Phone:    req.Phone,
		Email:    req.Email,
	})
	if err != nil {
		writeUserErrorProblem(c, err)
		return
	}
	// Apply optional admin-only fields.
	updates := map[string]interface{}{}
	if req.Level != "" {
		updates["level"] = req.Level
	}
	if req.ChannelType != "" {
		updates["channel_type"] = req.ChannelType
	}
	if len(updates) > 0 {
		h.db.WithContext(c.Request.Context()).Model(u).Updates(updates)
	}
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "user.create", "user", formatUint(u.ID), "", c.ClientIP())
	c.JSON(http.StatusCreated, u)
}

// Patch handles PATCH /v1/admin/users/:id.
type adminPatchUserReq struct {
	Username    *string `json:"username"`
	Phone       *string `json:"phone"`
	Email       *string `json:"email"`
	Password    *string `json:"password"`
	Level       *string `json:"level"`
	DeviceCount *int    `json:"device_count"`
	ChannelType *string `json:"channel_type"`
	Status      *bool   `json:"status"`
}

func (h *AdminUsersHandler) Patch(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid id")
		return
	}
	u, err := h.users.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		ProblemNotFound(c, ProblemCodeNotFound, "User not found")
		return
	}
	var req adminPatchUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}

	// Password change uses the service helper so complexity is enforced.
	if req.Password != nil && *req.Password != "" {
		if err := service.ValidatePassword(*req.Password); err != nil {
			ProblemBadRequest(c, "PASSWORD_WEAK", err.Error())
			return
		}
		// Service-level ResetPasswordByPhone isn't helpful here — run
		// inline.
		if err := h.db.WithContext(c.Request.Context()).Model(u).Update("password", hashPassword(*req.Password)).Error; err != nil {
			ProblemInternal(c, err.Error())
			return
		}
		// Revoke every active session for this user so they're forced to
		// log in again with the new password (§2.16).
		h.tokens.RevokeAllForSubject(c.Request.Context(), service.ScopeUser, u.ID)
		h.bus.Publish(c.Request.Context(), service.Event{
			Type:   service.EventSessionRevoked,
			UserID: u.ID,
			Data:   map[string]interface{}{"reason": "admin_password_reset"},
		})
	}

	updates := map[string]interface{}{}
	if req.Username != nil {
		updates["username"] = *req.Username
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.Level != nil {
		updates["level"] = *req.Level
	}
	if req.DeviceCount != nil {
		updates["device_count"] = *req.DeviceCount
	}
	if req.ChannelType != nil {
		updates["channel_type"] = *req.ChannelType
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if len(updates) > 0 {
		if err := h.db.WithContext(c.Request.Context()).Model(u).Updates(updates).Error; err != nil {
			ProblemInternal(c, err.Error())
			return
		}
	}
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "user.update", "user", formatUint(u.ID), "", c.ClientIP())
	u, _ = h.users.GetByID(c.Request.Context(), u.ID)
	c.JSON(http.StatusOK, u)
}

// Delete handles DELETE /v1/admin/users/:id.
func (h *AdminUsersHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid id")
		return
	}
	if err := h.users.Delete(c.Request.Context(), uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "User not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	// Revoke any lingering sessions.
	h.tokens.RevokeAllForSubject(c.Request.Context(), service.ScopeUser, uint(id))
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "user.delete", "user", c.Param("id"), "", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// RevokeSessions handles POST /v1/admin/users/:id/sessions:revoke.
func (h *AdminUsersHandler) RevokeSessions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid id")
		return
	}
	h.tokens.RevokeAllForSubject(c.Request.Context(), service.ScopeUser, uint(id))
	h.bus.Publish(c.Request.Context(), service.Event{
		Type:   service.EventSessionRevoked,
		UserID: uint(id),
		Data:   map[string]interface{}{"reason": "admin_forced"},
	})
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "user.sessions.revoke", "user", c.Param("id"), "", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// PatchDeviceCount handles PATCH /v1/admin/users/:id/device-count.
func (h *AdminUsersHandler) PatchDeviceCount(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid id")
		return
	}
	var req struct {
		DeviceCount int `json:"device_count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if err := h.db.WithContext(c.Request.Context()).Model(&models.User{}).
		Where("id = ?", id).
		Update("device_count", req.DeviceCount).Error; err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "user.device_count", "user", c.Param("id"), "", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Batch handles POST /v1/admin/users:batch.
type adminUsersBatchReq struct {
	IDs []uint `json:"ids" binding:"required"`
	Op  string `json:"op"  binding:"required"`
	// For op=set_level.
	Level string `json:"level"`
}

func (h *AdminUsersHandler) Batch(c *gin.Context) {
	var req adminUsersBatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if len(req.IDs) == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "ids required")
		return
	}
	switch req.Op {
	case "enable":
		if err := h.db.WithContext(c.Request.Context()).Model(&models.User{}).Where("id IN ?", req.IDs).Update("status", true).Error; err != nil {
			ProblemInternal(c, err.Error())
			return
		}
	case "disable":
		if err := h.db.WithContext(c.Request.Context()).Model(&models.User{}).Where("id IN ?", req.IDs).Update("status", false).Error; err != nil {
			ProblemInternal(c, err.Error())
			return
		}
		for _, id := range req.IDs {
			h.tokens.RevokeAllForSubject(c.Request.Context(), service.ScopeUser, id)
		}
	case "delete":
		for _, id := range req.IDs {
			_ = h.users.Delete(c.Request.Context(), id)
			h.tokens.RevokeAllForSubject(c.Request.Context(), service.ScopeUser, id)
		}
	case "set_level":
		if req.Level == "" {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "level required")
			return
		}
		if err := h.db.WithContext(c.Request.Context()).Model(&models.User{}).Where("id IN ?", req.IDs).Update("level", req.Level).Error; err != nil {
			ProblemInternal(c, err.Error())
			return
		}
	default:
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "unknown op")
		return
	}
	details, _ := json.Marshal(req)
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "user.batch", "user", "", string(details), c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
