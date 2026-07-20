package handler

import (
	"errors"
	"net/http"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/observability"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DeviceHandler serves the user-scoped device surface under /v1/me/*:
//
//	GET    /v1/me/devices
//	POST   /v1/me/devices                    (bind/takeover)
//	GET    /v1/me/devices/:device_id
//	PATCH  /v1/me/devices/:device_id         (remark / device_name)
//	DELETE /v1/me/devices/:device_id         (unbind)
//	DELETE /v1/me/devices/:device_id/session (clear logged_in)
//	GET    /v1/me/connections
//	POST   /v1/me/connections
//	GET    /v1/me/favorites
//	POST   /v1/me/favorites
//	PATCH  /v1/me/favorites/:device_id
//	DELETE /v1/me/favorites/:device_id
//
// All routes require middleware.UserAuth.Required(). Every mutation that
// observers care about publishes an event through EventBus so the realtime
// WS / webhook / audit subscribers stay up to date.
type DeviceHandler struct {
	devices     *service.DeviceService
	favorites   *service.FavoriteService
	connections *service.ConnectionService
	presence    *service.PresenceService
	bus         *service.EventBus
	db          *gorm.DB // for UserDevice remark lookups alongside devices
}

func NewDeviceHandler(
	devices *service.DeviceService,
	favorites *service.FavoriteService,
	connections *service.ConnectionService,
	presence *service.PresenceService,
	bus *service.EventBus,
	db *gorm.DB,
) *DeviceHandler {
	return &DeviceHandler{
		devices:     devices,
		favorites:   favorites,
		connections: connections,
		presence:    presence,
		bus:         bus,
		db:          db,
	}
}

// -----------------------------------------------------------------------
// Envelope helpers
// -----------------------------------------------------------------------

// deviceItem is the per-device wire shape used by GET /v1/me/devices (搂2.2).
// `online` and `logged_in` are derived from PresenceService at response
// time; they are never persisted on the device row.
type deviceItem struct {
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	Remark     string `json:"remark"`
	Online     bool   `json:"online"`
	LoggedIn   bool   `json:"logged_in"`
	AccessCode string `json:"access_code"`
	OS         string `json:"os"`
	OSVersion  string `json:"os_version"`
	AppVersion string `json:"app_version"`
	LastSeenAt string `json:"last_seen_at,omitempty"`
}

func (h *DeviceHandler) toDeviceItem(d *models.Device, remark string, online bool) deviceItem {
	lastSeen := ""
	if !d.LastSeenAt.IsZero() {
		lastSeen = d.LastSeenAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	return deviceItem{
		DeviceID:   d.DeviceID,
		DeviceName: d.DeviceName,
		Remark:     remark,
		Online:     online,
		LoggedIn:   d.LoggedIn && online,
		AccessCode: d.AccessCode,
		OS:         d.OS,
		OSVersion:  d.OSVersion,
		AppVersion: d.AppVersion,
		LastSeenAt: lastSeen,
	}
}

// -----------------------------------------------------------------------
// Listing / detail
// -----------------------------------------------------------------------

// ListMine handles GET /v1/me/devices. Cursor pagination is keyed on the
// device's internal id (DESC) 鈥?the caller opaquely round-trips the cursor.
func (h *DeviceHandler) ListMine(c *gin.Context) {
	uid := middleware.MustUserID(c)

	devs, err := h.devices.ListByUser(c.Request.Context(), uid)
	if err != nil {
		ProblemInternal(c, "Failed to list devices")
		return
	}

	// Fetch each device's per-user remark in one query.
	remarks := map[string]string{}
	if len(devs) > 0 {
		ids := make([]string, 0, len(devs))
		for _, d := range devs {
			ids = append(ids, d.DeviceID)
		}
		var uds []models.UserDevice
		h.db.WithContext(c.Request.Context()).
			Where("user_id = ? AND device_id IN ?", uid, ids).
			Find(&uds)
		for _, u := range uds {
			remarks[u.DeviceID] = u.Remark
		}
	}

	// Bulk presence lookup.
	ids := make([]string, 0, len(devs))
	for _, d := range devs {
		ids = append(ids, d.DeviceID)
	}
	onlineMap := h.presence.BulkOnline(c.Request.Context(), ids)

	items := make([]deviceItem, 0, len(devs))
	for i := range devs {
		items = append(items, h.toDeviceItem(&devs[i], remarks[devs[i].DeviceID], onlineMap[devs[i].DeviceID]))
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

// GetOne handles GET /v1/me/devices/:device_id.
func (h *DeviceHandler) GetOne(c *gin.Context) {
	uid := middleware.MustUserID(c)
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
	if d.UserID == nil || *d.UserID != uid {
		ProblemForbidden(c, ProblemCodeForbidden, "Device is not owned by this user")
		return
	}

	var ud models.UserDevice
	h.db.WithContext(c.Request.Context()).
		Where("user_id = ? AND device_id = ?", uid, deviceID).
		First(&ud)

	online := h.presence.IsOnline(c.Request.Context(), deviceID)
	c.JSON(http.StatusOK, h.toDeviceItem(d, ud.Remark, online))
}

// -----------------------------------------------------------------------
// Bind / unbind / clear session
// -----------------------------------------------------------------------

type bindDeviceReq struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// Bind handles POST /v1/me/devices.
func (h *DeviceHandler) Bind(c *gin.Context) {
	uid := middleware.MustUserID(c)
	var req bindDeviceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	result, err := h.devices.BindToUser(c.Request.Context(), req.DeviceID, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeDeviceNotFound, "Device not registered")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}

	online := h.presence.IsOnline(c.Request.Context(), req.DeviceID)
	var ud models.UserDevice
	h.db.WithContext(c.Request.Context()).
		Where("user_id = ? AND device_id = ?", uid, req.DeviceID).
		First(&ud)
	item := h.toDeviceItem(result.Device, ud.Remark, online)
	deviceEventData := map[string]interface{}{
		"device_id":    item.DeviceID,
		"device_name":  item.DeviceName,
		"remark":       item.Remark,
		"online":       item.Online,
		"logged_in":    item.LoggedIn,
		"os":           item.OS,
		"os_version":   item.OSVersion,
		"app_version":  item.AppVersion,
		"last_seen_at": item.LastSeenAt,
	}

	// Event fan-out. Takeover → (ownership.lost to old owner) + (added to new).
	// Plain bind → device.bound to new owner. The new-owner event carries
	// the non-secret DeviceItem fields so clients can converge via ordered
	// WS patches without issuing a racing HTTP list fetch.
	if result.PreviousOwner != nil {
		observability.Event("device", "ownership_transferred", map[string]interface{}{
			"device_id":     req.DeviceID,
			"new_user_id":   uid,
			"previous_user": *result.PreviousOwner,
			"request_id":    c.GetString("request_id"),
		})
		h.bus.Publish(c.Request.Context(), service.Event{
			Type:     service.EventDeviceOwnershipLost,
			UserID:   *result.PreviousOwner,
			DeviceID: req.DeviceID,
			Data: map[string]interface{}{
				"device_id": req.DeviceID,
				"new_owner": uid,
			},
		})
		if !result.AlreadyOwned {
			h.bus.Publish(c.Request.Context(), service.Event{
				Type:     service.EventDeviceAdded,
				UserID:   uid,
				DeviceID: req.DeviceID,
				Data:     deviceEventData,
			})
		}
	} else if result.AlreadyOwned {
		h.bus.Publish(c.Request.Context(), service.Event{
			Type:     service.EventDeviceSessionUpdated,
			UserID:   uid,
			DeviceID: req.DeviceID,
			Data: map[string]interface{}{
				"device_id": req.DeviceID,
				"online":    item.Online,
				"logged_in": item.LoggedIn,
			},
		})
	} else {
		observability.Event("device", "bound", map[string]interface{}{
			"device_id":  req.DeviceID,
			"request_id": c.GetString("request_id"),
			"user_id":    uid,
		})
		h.bus.Publish(c.Request.Context(), service.Event{
			Type:     service.EventDeviceBound,
			UserID:   uid,
			DeviceID: req.DeviceID,
			Data:     deviceEventData,
		})
	}

	c.JSON(http.StatusOK, item)
}

// Unbind handles DELETE /v1/me/devices/:device_id.
func (h *DeviceHandler) Unbind(c *gin.Context) {
	uid := middleware.MustUserID(c)
	deviceID := c.Param("device_id")
	if err := h.devices.UnbindFromUser(c.Request.Context(), deviceID, uid); err != nil {
		h.writeDeviceErr(c, err)
		return
	}
	h.bus.Publish(c.Request.Context(), service.Event{
		Type:     service.EventDeviceUnbound,
		UserID:   uid,
		DeviceID: deviceID,
		Data:     map[string]interface{}{"device_id": deviceID},
	})
	observability.Event("device", "unbound", map[string]interface{}{
		"device_id":  deviceID,
		"request_id": c.GetString("request_id"),
		"user_id":    uid,
	})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ClearSession handles DELETE /v1/me/devices/:device_id/session.
func (h *DeviceHandler) ClearSession(c *gin.Context) {
	uid := middleware.MustUserID(c)
	deviceID := c.Param("device_id")
	if err := h.devices.ClearSession(c.Request.Context(), deviceID, uid); err != nil {
		h.writeDeviceErr(c, err)
		return
	}
	h.bus.Publish(c.Request.Context(), service.Event{
		Type:     service.EventDeviceSessionUpdated,
		UserID:   uid,
		DeviceID: deviceID,
		Data: map[string]interface{}{
			"device_id": deviceID,
			"logged_in": false,
		},
	})
	observability.Event("device", "session_cleared", map[string]interface{}{
		"device_id":  deviceID,
		"request_id": c.GetString("request_id"),
		"user_id":    uid,
	})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// -----------------------------------------------------------------------
// PATCH metadata
// -----------------------------------------------------------------------

type patchDeviceReq struct {
	DeviceName *string `json:"device_name"`
	Remark     *string `json:"remark"`
}

func (h *DeviceHandler) Patch(c *gin.Context) {
	uid := middleware.MustUserID(c)
	deviceID := c.Param("device_id")
	var req patchDeviceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if req.DeviceName == nil && req.Remark == nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "no patchable fields supplied")
		return
	}
	if err := h.devices.PatchMeta(c.Request.Context(), deviceID, uid, service.PatchMetaInput{
		DeviceName: req.DeviceName,
		Remark:     req.Remark,
	}); err != nil {
		h.writeDeviceErr(c, err)
		return
	}
	if req.DeviceName != nil {
		h.bus.Publish(c.Request.Context(), service.Event{
			Type:     service.EventDeviceNameChanged,
			UserID:   uid,
			DeviceID: deviceID,
			Data: map[string]interface{}{
				"device_id":   deviceID,
				"device_name": *req.DeviceName,
			},
		})
	}
	if req.Remark != nil {
		h.bus.Publish(c.Request.Context(), service.Event{
			Type:     service.EventDeviceRemarkChanged,
			UserID:   uid,
			DeviceID: deviceID,
			Data: map[string]interface{}{
				"device_id": deviceID,
				"remark":    *req.Remark,
			},
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// writeDeviceErr maps DeviceService sentinel errors to RFC7807.
func (h *DeviceHandler) writeDeviceErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		ProblemNotFound(c, ProblemCodeDeviceNotFound, "Device not found")
	case errors.Is(err, service.ErrDeviceNotOwned):
		ProblemForbidden(c, ProblemCodeForbidden, "Device is not owned by this user")
	default:
		ProblemInternal(c, err.Error())
	}
}

// -----------------------------------------------------------------------
// Connections
// -----------------------------------------------------------------------

type recordConnectionReq struct {
	DeviceID   string `json:"device_id" binding:"required"`
	DeviceName string `json:"device_name"`
	Duration   int    `json:"duration"`
	Status     string `json:"status" binding:"required"`
	ErrorMsg   string `json:"error_msg"`
}

func (h *DeviceHandler) RecordConnection(c *gin.Context) {
	uid := middleware.MustUserID(c)
	var req recordConnectionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	row, err := h.connections.Record(c.Request.Context(), uid, service.RecordInput{
		DeviceID:   req.DeviceID,
		DeviceName: req.DeviceName,
		ConnectIP:  c.ClientIP(),
		Duration:   req.Duration,
		Status:     req.Status,
		ErrorMsg:   req.ErrorMsg,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidConnectionStatus) {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, row)
}

func (h *DeviceHandler) ListConnections(c *gin.Context) {
	uid := middleware.MustUserID(c)
	p := ParseCursor(c)
	cur, err := DecodeCursor(p.Cursor)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	params := service.ConnectionListParams{Limit: p.Limit + 1}
	if cur.OffsetAt != "" {
		if t, perr := parseTime(cur.OffsetAt); perr == nil {
			params.Before = t
		}
	}
	rows, err := h.connections.List(c.Request.Context(), uid, params)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	var next string
	if len(rows) > p.Limit {
		last := rows[p.Limit-1]
		next = EncodeCursor(CursorPayload{OffsetAt: last.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z")})
		rows = rows[:p.Limit]
	}
	c.JSON(http.StatusOK, gin.H{"items": rows, "next_cursor": next})
}

// -----------------------------------------------------------------------
// Favorites
// -----------------------------------------------------------------------

type addFavoriteReq struct {
	DeviceID       string `json:"device_id" binding:"required"`
	DeviceName     string `json:"device_name"`
	AccessPassword string `json:"access_password"`
}

type patchFavoriteReq struct {
	DeviceName     *string `json:"device_name"`
	AccessPassword *string `json:"access_password"`
}

func (h *DeviceHandler) ListFavorites(c *gin.Context) {
	uid := middleware.MustUserID(c)
	items, err := h.favorites.List(c.Request.Context(), uid)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *DeviceHandler) AddFavorite(c *gin.Context) {
	uid := middleware.MustUserID(c)
	var req addFavoriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	fav, err := h.favorites.Add(c.Request.Context(), uid, service.FavoriteInput{
		DeviceID:       req.DeviceID,
		DeviceName:     req.DeviceName,
		AccessPassword: req.AccessPassword,
	})
	if err != nil {
		if errors.Is(err, service.ErrFavoriteExists) {
			ProblemConflict(c, ProblemCodeConflict, err.Error())
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	h.bus.Publish(c.Request.Context(), service.Event{
		Type:     service.EventFavoriteAdded,
		UserID:   uid,
		DeviceID: req.DeviceID,
		Data: map[string]interface{}{
			"device_id":   req.DeviceID,
			"device_name": req.DeviceName,
		},
	})
	c.JSON(http.StatusOK, fav)
}

func (h *DeviceHandler) UpdateFavorite(c *gin.Context) {
	uid := middleware.MustUserID(c)
	deviceID := c.Param("device_id")
	var req patchFavoriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	fav, err := h.favorites.Update(c.Request.Context(), uid, deviceID, service.FavoritePatch{
		DeviceName:     req.DeviceName,
		AccessPassword: req.AccessPassword,
	})
	if err != nil {
		if errors.Is(err, service.ErrFavoriteNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, err.Error())
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	h.bus.Publish(c.Request.Context(), service.Event{
		Type:     service.EventFavoriteUpdated,
		UserID:   uid,
		DeviceID: deviceID,
		Data: map[string]interface{}{
			"device_id": deviceID,
		},
	})
	c.JSON(http.StatusOK, fav)
}

func (h *DeviceHandler) DeleteFavorite(c *gin.Context) {
	uid := middleware.MustUserID(c)
	deviceID := c.Param("device_id")
	if err := h.favorites.Delete(c.Request.Context(), uid, deviceID); err != nil {
		if errors.Is(err, service.ErrFavoriteNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, err.Error())
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	h.bus.Publish(c.Request.Context(), service.Event{
		Type:     service.EventFavoriteRemoved,
		UserID:   uid,
		DeviceID: deviceID,
		Data:     map[string]interface{}{"device_id": deviceID},
	})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
