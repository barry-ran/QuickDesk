package handler

import (
	"net/http"
	"quickdesk/signaling/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UserDeviceHandler manages user-device binding operations.
type UserDeviceHandler struct {
	db           *gorm.DB
	syncNotifier func(userID uint, msg interface{})
}

// NewUserDeviceHandler creates a new UserDeviceHandler.
func NewUserDeviceHandler(db *gorm.DB) *UserDeviceHandler {
	return &UserDeviceHandler{db: db}
}

// UnbindDevice handles POST /api/v1/user/devices/unbind
func (h *UserDeviceHandler) UnbindDevice(c *gin.Context) {
	var req struct {
		DeviceID string `json:"device_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userIDVal, _ := c.Get("authed_user_id")
	authedUserID := userIDVal.(uint)

	var binding models.UserDevice
	if result := h.db.Where("user_id = ? AND device_id = ? AND status = ?", authedUserID, req.DeviceID, true).First(&binding); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "绑定记录不存在"})
		return
	}

	binding.Status = false
	h.db.Save(&binding)
	recomputeDeviceCount(h.db, authedUserID)

	c.JSON(http.StatusOK, gin.H{"message": "设备解绑成功"})
}

// GetUserDevices handles GET /api/v1/user/devices
// Returns all devices bound to a user (from the devices table),
// enriched with remark from user_devices, access_code and online status.
func (h *UserDeviceHandler) GetUserDevices(c *gin.Context) {
	userID, _ := c.Get("authed_user_id")

	var devices []models.Device
	if result := h.db.Where("user_id = ?", userID).Find(&devices); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// Build a map of device_id → UserDevice for remark lookup
	var userDevices []models.UserDevice
	h.db.Where("user_id = ? AND status = ?", userID, true).Find(&userDevices)
	udMap := make(map[string]models.UserDevice)
	for _, ud := range userDevices {
		udMap[ud.DeviceID] = ud
	}

	type DeviceInfo struct {
		models.Device
		Remark string `json:"remark"`
	}

	result := make([]DeviceInfo, 0, len(devices))
	for _, d := range devices {
		info := DeviceInfo{Device: d}
		if ud, ok := udMap[d.DeviceID]; ok {
			info.Remark = ud.DeviceName
		}
		result = append(result, info)
	}

	c.JSON(http.StatusOK, gin.H{"devices": result, "count": len(result)})
}

// GetUserDeviceLogs handles GET /api/v1/user/devices/logs
// Returns connection history for a user over the last 3 days.
func (h *UserDeviceHandler) GetUserDeviceLogs(c *gin.Context) {
	userID, _ := c.Get("authed_user_id")

	threeDaysAgo := time.Now().AddDate(0, 0, -3)
	var logs []models.ConnectionHistory
	if result := h.db.Where("user_id = ? AND created_at >= ?", userID, threeDaysAgo).
		Order("created_at DESC").
		Limit(100).
		Find(&logs); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs, "count": len(logs)})
}

// RecordConnection handles POST /api/v1/user/devices/record
// Called by WebClient after a connection attempt to persist its result.
func (h *UserDeviceHandler) RecordConnection(c *gin.Context) {
	var req struct {
		DeviceID string `json:"device_id" binding:"required"`
		Duration int    `json:"duration"` // seconds
		Status   string `json:"status"`   // success / failed / timeout
		ErrorMsg string `json:"error_msg"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userIDVal, _ := c.Get("authed_user_id")
	authedUserID := userIDVal.(uint)

	entry := models.ConnectionHistory{
		UserID:    authedUserID,
		DeviceID:  req.DeviceID,
		ConnectIP: c.ClientIP(),
		Status:    req.Status,
		ErrorMsg:  req.ErrorMsg,
		Duration:  req.Duration,
	}
	h.db.Create(&entry)

	if req.Status == "success" {
		h.db.Model(&models.UserDevice{}).
			Where("user_id = ? AND device_id = ? AND status = ?", authedUserID, req.DeviceID, true).
			Updates(map[string]interface{}{
				"last_connect":  time.Now(),
				"connect_count": gorm.Expr("connect_count + 1"),
			})
	}

	c.JSON(http.StatusOK, gin.H{"message": "连接记录已保存"})
}

// GetAllBindings handles GET /admin/device-bindings — admin only.
func (h *UserDeviceHandler) GetAllBindings(c *gin.Context) {
	var bindings []models.UserDevice
	if result := h.db.Preload("User").Find(&bindings); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bindings": bindings, "count": len(bindings)})
}

// recomputeDeviceCount recalculates the user's device_count from the actual user_devices table.
func recomputeDeviceCount(db *gorm.DB, userID uint) {
	var count int64
	db.Model(&models.UserDevice{}).Where("user_id = ? AND status = ?", userID, true).Count(&count)
	db.Model(&models.User{}).Where("id = ?", userID).Update("device_count", count)
}

// logConnection persists a connection event to connection_histories.
func (h *UserDeviceHandler) logConnection(userID uint, deviceID, deviceName, status, errorMsg, connectIP string) {
	entry := models.ConnectionHistory{
		UserID:     userID,
		DeviceID:   deviceID,
		DeviceName: deviceName,
		ConnectIP:  connectIP,
		Status:     status,
		ErrorMsg:   errorMsg,
	}
	h.db.Create(&entry)
}

// SetSyncNotifier sets the callback used to push sync messages to connected clients.
func (h *UserDeviceHandler) SetSyncNotifier(fn func(userID uint, msg interface{})) {
	h.syncNotifier = fn
}

// notifySync pushes a sync message to the user's connected clients if a notifier is set.
func (h *UserDeviceHandler) notifySync(userID uint, msg interface{}) {
	if h.syncNotifier != nil {
		h.syncNotifier(userID, msg)
	}
}

// AutoBindDevice handles POST /api/v1/user/devices/auto-bind
// Host calls this when the user logs in to automatically bind the device.
func (h *UserDeviceHandler) AutoBindDevice(c *gin.Context) {
	var req struct {
		DeviceID string `json:"device_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userIDVal, _ := c.Get("authed_user_id")
	authedUserID := userIDVal.(uint)

	var device models.Device
	if result := h.db.Where("device_id = ?", req.DeviceID).First(&device); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "设备不存在"})
		return
	}

	if device.UserID != nil && *device.UserID != 0 && *device.UserID != authedUserID {
		c.JSON(http.StatusConflict, gin.H{"error": "设备已绑定其他用户"})
		return
	}

	// Update device ownership
	h.db.Model(&models.Device{}).Where("device_id = ?", req.DeviceID).Update("user_id", authedUserID)

	// Upsert UserDevice: reactivate if exists but inactive, create otherwise
	var existing models.UserDevice
	result := h.db.Where("user_id = ? AND device_id = ?", authedUserID, req.DeviceID).First(&existing)
	if result.Error == nil {
		// Record exists — reactivate if needed
		if !existing.Status {
			existing.Status = true
		}
		existing.BindType = "auto"
		existing.LastConnect = time.Now()
		existing.ConnectCount++
		h.db.Save(&existing)
	} else {
		existing = models.UserDevice{
			UserID:       authedUserID,
			DeviceID:     req.DeviceID,
			DeviceName:   "设备-" + req.DeviceID,
			BindType:     "auto",
			Status:       true,
			LastConnect:  time.Now(),
			ConnectCount: 1,
		}
		h.db.Create(&existing)
	}

	recomputeDeviceCount(h.db, authedUserID)

	c.JSON(http.StatusOK, gin.H{"message": "设备自动绑定成功", "binding": existing})
}

// UpdateAccessCode handles PUT /api/v1/user/devices/:device_id/access-code
// Host updates the device access code.
func (h *UserDeviceHandler) UpdateAccessCode(c *gin.Context) {
	deviceID := c.Param("device_id")
	var req struct {
		AccessCode string `json:"access_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userIDVal, _ := c.Get("authed_user_id")
	authedUserID := userIDVal.(uint)

	var device models.Device
	if result := h.db.Where("device_id = ?", deviceID).First(&device); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "设备不存在"})
		return
	}

	if device.UserID != nil && *device.UserID != 0 && *device.UserID != authedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此设备"})
		return
	}

	h.db.Model(&models.Device{}).Where("device_id = ?", deviceID).Update("access_code", req.AccessCode)

	h.notifySync(authedUserID, gin.H{
		"type":        "device_access_code_changed",
		"device_id":   deviceID,
		"access_code": req.AccessCode,
	})

	c.JSON(http.StatusOK, gin.H{"message": "访问码更新成功"})
}

// UpdateDeviceRemark handles PUT /api/v1/user/devices/:device_id/remark
// Sets the user-defined remark for a device.
func (h *UserDeviceHandler) UpdateDeviceRemark(c *gin.Context) {
	deviceID := c.Param("device_id")
	var req struct {
		Remark string `json:"remark" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userIDVal, _ := c.Get("authed_user_id")
	authedUserID := userIDVal.(uint)

	var binding models.UserDevice
	if result := h.db.Where("user_id = ? AND device_id = ? AND status = ?", authedUserID, deviceID, true).First(&binding); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "绑定记录不存在"})
		return
	}

	binding.DeviceName = req.Remark
	h.db.Save(&binding)

	h.notifySync(authedUserID, gin.H{
		"type":      "device_remark_changed",
		"device_id": deviceID,
		"remark":    req.Remark,
	})

	c.JSON(http.StatusOK, gin.H{"message": "备注更新成功"})
}

// GetFavorites handles GET /api/v1/user/favorites
// Returns all favorites for the authenticated user.
func (h *UserDeviceHandler) GetFavorites(c *gin.Context) {
	userIDVal, _ := c.Get("authed_user_id")
	authedUserID := userIDVal.(uint)

	var favorites []models.UserFavorite
	if result := h.db.Where("user_id = ?", authedUserID).Order("updated_at DESC").Find(&favorites); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"favorites": favorites, "count": len(favorites)})
}

// AddFavorite handles POST /api/v1/user/favorites
// Adds or updates a favorite device for quick access.
func (h *UserDeviceHandler) AddFavorite(c *gin.Context) {
	var req struct {
		DeviceID       string `json:"device_id" binding:"required"`
		DeviceName     string `json:"device_name"`
		AccessPassword string `json:"access_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userIDVal, _ := c.Get("authed_user_id")
	authedUserID := userIDVal.(uint)

	var favorite models.UserFavorite
	result := h.db.Where("user_id = ? AND device_id = ?", authedUserID, req.DeviceID).First(&favorite)
	if result.Error == nil {
		// Already exists — update it
		if req.DeviceName != "" {
			favorite.DeviceName = req.DeviceName
		}
		if req.AccessPassword != "" {
			favorite.AccessPassword = req.AccessPassword
		}
		h.db.Save(&favorite)
	} else {
		favorite = models.UserFavorite{
			UserID:         authedUserID,
			DeviceID:       req.DeviceID,
			DeviceName:     req.DeviceName,
			AccessPassword: req.AccessPassword,
		}
		if result := h.db.Create(&favorite); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "添加收藏失败: " + result.Error.Error()})
			return
		}
	}

	h.notifySync(authedUserID, gin.H{
		"type":     "favorite_added",
		"favorite": favorite,
	})

	c.JSON(http.StatusOK, gin.H{"message": "收藏成功", "favorite": favorite})
}

// UpdateFavorite handles PUT /api/v1/user/favorites/:device_id
// Updates an existing favorite's fields.
func (h *UserDeviceHandler) UpdateFavorite(c *gin.Context) {
	deviceID := c.Param("device_id")
	var req struct {
		DeviceName     string `json:"device_name"`
		AccessPassword string `json:"access_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userIDVal, _ := c.Get("authed_user_id")
	authedUserID := userIDVal.(uint)

	var favorite models.UserFavorite
	if result := h.db.Where("user_id = ? AND device_id = ?", authedUserID, deviceID).First(&favorite); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "收藏记录不存在"})
		return
	}

	if req.DeviceName != "" {
		favorite.DeviceName = req.DeviceName
	}
	if req.AccessPassword != "" {
		favorite.AccessPassword = req.AccessPassword
	}
	h.db.Save(&favorite)

	h.notifySync(authedUserID, gin.H{
		"type":     "favorite_updated",
		"favorite": favorite,
	})

	c.JSON(http.StatusOK, gin.H{"message": "收藏更新成功", "favorite": favorite})
}

// RemoveFavorite handles DELETE /api/v1/user/favorites/:device_id
// Removes a favorite device.
func (h *UserDeviceHandler) RemoveFavorite(c *gin.Context) {
	deviceID := c.Param("device_id")
	userIDVal, _ := c.Get("authed_user_id")
	authedUserID := userIDVal.(uint)

	result := h.db.Where("user_id = ? AND device_id = ?", authedUserID, deviceID).Delete(&models.UserFavorite{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "收藏记录不存在"})
		return
	}

	h.notifySync(authedUserID, gin.H{
		"type":      "favorite_removed",
		"device_id": deviceID,
	})

	c.JSON(http.StatusOK, gin.H{"message": "取消收藏成功"})
}

// DeviceLogin handles POST /api/v1/user/devices/:device_id/login
// Called when a user logs in on a device, marks the device as actively logged in.
func (h *UserDeviceHandler) DeviceLogin(c *gin.Context) {
	deviceID := c.Param("device_id")
	userIDVal, _ := c.Get("authed_user_id")
	authedUserID := userIDVal.(uint)

	var device models.Device
	if result := h.db.Where("device_id = ?", deviceID).First(&device); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "设备不存在"})
		return
	}

	if device.UserID != nil && *device.UserID != 0 && *device.UserID != authedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此设备"})
		return
	}

	h.db.Model(&models.Device{}).Where("device_id = ?", deviceID).Update("logged_in", true)

	h.notifySync(authedUserID, gin.H{
		"type":      "device_logged_in",
		"device_id": deviceID,
	})

	c.JSON(http.StatusOK, gin.H{"message": "设备登录状态已更新"})
}

// DeviceLogout handles POST /api/v1/user/devices/:device_id/logout
// Called when a user logs out on a device, marks the device as not logged in.
func (h *UserDeviceHandler) DeviceLogout(c *gin.Context) {
	deviceID := c.Param("device_id")
	userIDVal, _ := c.Get("authed_user_id")
	authedUserID := userIDVal.(uint)

	var device models.Device
	if result := h.db.Where("device_id = ?", deviceID).First(&device); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "设备不存在"})
		return
	}

	if device.UserID != nil && *device.UserID != 0 && *device.UserID != authedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此设备"})
		return
	}

	h.db.Model(&models.Device{}).Where("device_id = ?", deviceID).Update("logged_in", false)

	h.notifySync(authedUserID, gin.H{
		"type":      "device_logged_out",
		"device_id": deviceID,
	})

	c.JSON(http.StatusOK, gin.H{"message": "设备登出状态已更新"})
}
