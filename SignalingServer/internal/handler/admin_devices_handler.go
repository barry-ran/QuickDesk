package handler

import (
	"errors"
	"net/http"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminDevicesHandler serves /v1/admin/devices/* 鈥?list, detail, force
// unbind, secret rotation, and hard delete.
type AdminDevicesHandler struct {
	devices  *service.DeviceService
	presence *service.PresenceService
	bus      *service.EventBus
	audit    *service.AuditService
	db       *gorm.DB
}

func NewAdminDevicesHandler(
	devices *service.DeviceService,
	presence *service.PresenceService,
	bus *service.EventBus,
	audit *service.AuditService,
	db *gorm.DB,
) *AdminDevicesHandler {
	return &AdminDevicesHandler{
		devices:  devices,
		presence: presence,
		bus:      bus,
		audit:    audit,
		db:       db,
	}
}

func (h *AdminDevicesHandler) List(c *gin.Context) {
	p := ParseCursor(c)
	cur, err := DecodeCursor(p.Cursor)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	allowedSorts := map[string]bool{
		"id": true, "created_at": true, "updated_at": true,
		"last_seen_at": true, "device_id": true,
	}
	if !allowedSorts[p.Sort] {
		p.Sort = "id"
	}
	devs, total, err := h.devices.ListAdmin(c.Request.Context(), service.DeviceAdminListParams{
		AfterID: cur.OffsetID,
		Limit:   p.Limit + 1,
		Sort:    p.Sort,
		Order:   p.Order,
		Search:  p.Search,
		OS:      c.Query("os"),
	})
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	next := ""
	if len(devs) > p.Limit {
		last := devs[p.Limit-1]
		next = EncodeCursor(CursorPayload{OffsetID: last.ID})
		devs = devs[:p.Limit]
	}

	ids := make([]string, 0, len(devs))
	for _, d := range devs {
		ids = append(ids, d.DeviceID)
	}
	online := h.presence.BulkOnline(c.Request.Context(), ids)
	items := make([]gin.H, 0, len(devs))
	for i := range devs {
		d := &devs[i]
		items = append(items, deviceAdminJSON(d, online[d.DeviceID]))
	}
	c.JSON(http.StatusOK, NewCursorPage(items, next, total))
}

// deviceAdminJSON adds the cross-cutting derived fields admin views care
// about. `logged_in` is the *derived* value (intent AND online), matching
// the shape used elsewhere in the v1 API (§2.2).
func deviceAdminJSON(d *models.Device, online bool) gin.H {
	out := gin.H{
		"id":           d.ID,
		"device_id":    d.DeviceID,
		"device_uuid":  d.DeviceUUID,
		"device_name":  d.DeviceName,
		"os":           d.OS,
		"os_version":   d.OSVersion,
		"app_version":  d.AppVersion,
		"user_id":      d.UserID,
		"access_code":  d.AccessCode,
		"online":       online,
		"logged_in":    d.LoggedIn && online,
		"last_seen_at": d.LastSeenAt,
		"created_at":   d.CreatedAt,
		"updated_at":   d.UpdatedAt,
	}
	if d.UserID != nil {
		out["user"] = d.User
	}
	return out
}

func (h *AdminDevicesHandler) Get(c *gin.Context) {
	deviceID := c.Param("device_id")
	d, err := h.devices.GetByDeviceIDWithUser(c.Request.Context(), deviceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeDeviceNotFound, "Device not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	online := h.presence.IsOnline(c.Request.Context(), deviceID)
	out := deviceAdminJSON(d, online)
	var history []models.ConnectionHistory
	h.db.WithContext(c.Request.Context()).
		Where("device_id = ?", deviceID).
		Order("created_at DESC").
		Limit(50).
		Find(&history)
	out["connection_history"] = history
	c.JSON(http.StatusOK, out)
}

func (h *AdminDevicesHandler) Delete(c *gin.Context) {
	deviceID := c.Param("device_id")
	d, err := h.devices.GetByDeviceID(c.Request.Context(), deviceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeDeviceNotFound, "Device not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	owner := d.UserID
	if err := h.devices.Delete(c.Request.Context(), deviceID); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	if owner != nil {
		h.bus.Publish(c.Request.Context(), service.Event{
			Type:     service.EventDeviceUnbound,
			UserID:   *owner,
			DeviceID: deviceID,
			Data:     map[string]interface{}{"device_id": deviceID, "reason": "admin_delete"},
		})
	}
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "device.delete", "device", deviceID, "", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ForceUnbind handles POST /v1/admin/devices/:device_id/unbind.
func (h *AdminDevicesHandler) ForceUnbind(c *gin.Context) {
	deviceID := c.Param("device_id")
	prev, err := h.devices.AdminForceUnbind(c.Request.Context(), deviceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeDeviceNotFound, "Device not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	if prev != nil {
		h.bus.Publish(c.Request.Context(), service.Event{
			Type:     service.EventDeviceOwnershipLost,
			UserID:   *prev,
			DeviceID: deviceID,
			Data:     map[string]interface{}{"device_id": deviceID, "reason": "admin_unbind"},
		})
	}
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "device.unbind", "device", deviceID, "", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// RotateSecret handles POST /v1/admin/devices/:device_id/secret:rotate.
func (h *AdminDevicesHandler) RotateSecret(c *gin.Context) {
	deviceID := c.Param("device_id")
	plaintext, err := h.devices.RotateSecret(c.Request.Context(), deviceID)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	// Publish the system-scope event so audit + webhook subscribers see
	// it; user-scope routing is intentionally absent (the host is
	// expected to discover the rotation via the next 401 from a
	// device-secret-protected endpoint and re-provision).
	h.bus.Publish(c.Request.Context(), service.Event{
		Type:     service.EventDeviceSecretRotated,
		DeviceID: deviceID,
		Data:     map[string]interface{}{"device_id": deviceID},
	})
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "device.secret.rotate", "device", deviceID, "", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{
		"device_id":     deviceID,
		"device_secret": plaintext, // returned exactly once
	})
}

// Batch handles POST /v1/admin/devices:batch (delete / assign-group / remove-group).
type adminDevicesBatchReq struct {
	IDs     []string `json:"ids" binding:"required"`
	Op      string   `json:"op"  binding:"required"`
	GroupID uint     `json:"group_id"`
}

func (h *AdminDevicesHandler) Batch(c *gin.Context, groups *service.DeviceGroupService) {
	var req adminDevicesBatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if len(req.IDs) == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "ids required")
		return
	}
	switch req.Op {
	case "delete":
		for _, id := range req.IDs {
			_ = h.devices.Delete(c.Request.Context(), id)
		}
	case "assign_group":
		if req.GroupID == 0 {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "group_id required")
			return
		}
		if err := groups.AddDevices(c.Request.Context(), req.GroupID, req.IDs); err != nil {
			ProblemInternal(c, err.Error())
			return
		}
	case "remove_group":
		if req.GroupID == 0 {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "group_id required")
			return
		}
		if err := groups.RemoveDevices(c.Request.Context(), req.GroupID, req.IDs); err != nil {
			ProblemInternal(c, err.Error())
			return
		}
	default:
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "unknown op")
		return
	}
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "device.batch", "device", "", req.Op, c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ListBindings handles GET /v1/admin/device-bindings — paged list of every
// active user↔device binding (user_devices rows with status=true), joined
// with the parent user and device for admin display. Cursor pagination on
// user_devices.id DESC.
func (h *AdminDevicesHandler) ListBindings(c *gin.Context) {
	p := ParseCursor(c)
	cur, err := DecodeCursor(p.Cursor)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	q := h.db.WithContext(c.Request.Context()).
		Model(&models.UserDevice{}).
		Where("status = ?", true)

	if p.Search != "" {
		like := "%" + p.Search + "%"
		q = q.Where(
			"device_id LIKE ? OR user_id IN (SELECT id FROM users WHERE username LIKE ? OR phone LIKE ? OR email LIKE ?)",
			like, like, like, like,
		)
	}
	if uid := c.Query("user_id"); uid != "" {
		q = q.Where("user_id = ?", uid)
	}
	if did := c.Query("device_id"); did != "" {
		q = q.Where("device_id = ?", did)
	}

	var total int64
	q.Session(&gorm.Session{}).Count(&total)

	if cur.OffsetID > 0 {
		q = q.Where("id < ?", cur.OffsetID)
	}

	var rows []models.UserDevice
	if err := q.Order("id DESC").
		Limit(p.Limit + 1).
		Preload("User").
		Find(&rows).Error; err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	next := ""
	if len(rows) > p.Limit {
		last := rows[p.Limit-1]
		next = EncodeCursor(CursorPayload{OffsetID: last.ID})
		rows = rows[:p.Limit]
	}

	// Enrich each binding with the device row so operators can see
	// device_name / online / logged_in without a separate lookup.
	deviceIDs := make([]string, 0, len(rows))
	for _, r := range rows {
		deviceIDs = append(deviceIDs, r.DeviceID)
	}
	var devs []models.Device
	if len(deviceIDs) > 0 {
		h.db.WithContext(c.Request.Context()).
			Where("device_id IN ?", deviceIDs).
			Find(&devs)
	}
	deviceByID := make(map[string]*models.Device, len(devs))
	for i := range devs {
		deviceByID[devs[i].DeviceID] = &devs[i]
	}
	online := h.presence.BulkOnline(c.Request.Context(), deviceIDs)

	items := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		d := deviceByID[r.DeviceID]
		entry := gin.H{
			"id":              r.ID,
			"user_id":         r.UserID,
			"user":            r.User,
			"device_id":       r.DeviceID,
			"remark":          r.Remark,
			"first_bound_at":  r.FirstBoundAt,
			"last_connect_at": r.LastConnectAt,
			"connect_count":   r.ConnectCount,
			"created_at":      r.CreatedAt,
			"updated_at":      r.UpdatedAt,
		}
		if d != nil {
			entry["device"] = gin.H{
				"device_name":  d.DeviceName,
				"os":           d.OS,
				"os_version":   d.OSVersion,
				"app_version":  d.AppVersion,
				"online":       online[d.DeviceID],
				"logged_in":    d.LoggedIn && online[d.DeviceID],
				"last_seen_at": d.LastSeenAt,
			}
		}
		items = append(items, entry)
	}
	c.JSON(http.StatusOK, NewCursorPage(items, next, total))
}
