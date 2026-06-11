package handler

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"gorm.io/gorm"
)

// AdminStatsHandler exposes read-only observability endpoints used by the
// admin web dashboard. All of these are simple aggregate SQL queries on
// the operational tables 鈥?no WS / event subscription.
type AdminStatsHandler struct {
	devices   *service.DeviceService
	presence  *service.PresenceService
	metrics   *service.MetricsService
	startedAt time.Time
	db        *gorm.DB
}

func NewAdminStatsHandler(d *service.DeviceService, p *service.PresenceService, m *service.MetricsService, db *gorm.DB) *AdminStatsHandler {
	return &AdminStatsHandler{devices: d, presence: p, metrics: m, startedAt: time.Now().UTC(), db: db}
}

// GetStats handles GET /v1/admin/stats 鈥?the top-line numbers on the
// landing page: total users, total devices, online devices, new today.
func (h *AdminStatsHandler) GetStats(c *gin.Context) {
	ctx := c.Request.Context()
	var userCount, deviceCount int64
	h.db.WithContext(ctx).Model(&models.User{}).Count(&userCount)
	h.db.WithContext(ctx).Model(&models.Device{}).Count(&deviceCount)

	startToday := time.Now().UTC().Truncate(24 * time.Hour)
	usersNew, _ := countSince(ctx, h.db, &models.User{}, startToday)
	devicesNew, _ := h.devices.CountSince(ctx, startToday)
	todayActiveUsers, _ := h.countActiveUsersSince(ctx, startToday)

	// Online: scan each device row 鈥?cheap enough for a few thousand.
	onlineCount := h.countOnlineDevices(ctx)

	c.JSON(http.StatusOK, gin.H{
		"users_total":        userCount,
		"devices_total":      deviceCount,
		"devices_online":     onlineCount,
		"users_new_today":    usersNew,
		"devices_new_today":  devicesNew,
		"today_active_users": todayActiveUsers,
	})
}

func (h *AdminStatsHandler) countOnlineDevices(ctx context.Context) int64 {
	var devices []models.Device
	if err := h.db.WithContext(ctx).
		Select("device_id").
		Find(&devices).Error; err != nil {
		return 0
	}
	ids := make([]string, 0, len(devices))
	for _, d := range devices {
		ids = append(ids, d.DeviceID)
	}
	online := h.presence.BulkOnline(ctx, ids)
	var count int64
	for _, v := range online {
		if v {
			count++
		}
	}
	return count
}

func (h *AdminStatsHandler) countActiveUsersSince(ctx context.Context, since time.Time) (int64, error) {
	var n int64
	err := h.db.WithContext(ctx).
		Model(&models.ConnectionHistory{}).
		Where("created_at >= ?", since).
		Distinct("user_id").
		Count(&n).Error
	return n, err
}

// GetSystemStatus handles GET /v1/admin/system/status.
func (h *AdminStatsHandler) GetSystemStatus(c *gin.Context) {
	vm, _ := mem.VirtualMemory()
	cpuP, _ := cpu.Percent(0, false)
	var cpuPercent float64
	if len(cpuP) > 0 {
		cpuPercent = cpuP[0]
	}
	c.JSON(http.StatusOK, gin.H{
		"uptime_seconds": time.Since(h.startedAt).Seconds(),
		"go_version":     runtime.Version(),
		"num_goroutines": runtime.NumGoroutine(),
		"mem_used":       vm.Used,
		"mem_total":      vm.Total,
		"cpu_percent":    cpuPercent,
	})
}

// GetActivity handles GET /v1/admin/activity — a paged feed of connection
// history entries for dashboards.
func (h *AdminStatsHandler) GetActivity(c *gin.Context) {
	p := ParseCursor(c)
	cur, err := DecodeCursor(p.Cursor)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	q := h.db.WithContext(c.Request.Context()).Model(&models.ConnectionHistory{})
	var total int64
	q.Session(&gorm.Session{}).Count(&total)
	if cur.OffsetID > 0 {
		q = q.Where("id < ?", cur.OffsetID)
	}
	var rows []models.ConnectionHistory
	if err := q.Order("id DESC").Limit(p.Limit + 1).Find(&rows).Error; err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	next := ""
	if len(rows) > p.Limit {
		last := rows[p.Limit-1]
		next = EncodeCursor(CursorPayload{OffsetID: last.ID})
		rows = rows[:p.Limit]
	}
	c.JSON(http.StatusOK, NewCursorPage(rows, next, total))
}

// GetTrends returns daily counts for the last 7/30/etc days. Kept simple.
func (h *AdminStatsHandler) GetTrends(c *gin.Context) {
	rangeStr := c.DefaultQuery("range", "7d")
	days := 7
	switch rangeStr {
	case "24h":
		days = 1
	case "30d":
		days = 30
	}
	// Use a single SELECT with GROUP BY DATE to keep the dashboard snappy.
	type row struct {
		Day   time.Time `json:"day"`
		Count int64     `json:"count"`
	}
	var users, devices, conns []row
	start := time.Now().UTC().AddDate(0, 0, -days)
	h.db.WithContext(c.Request.Context()).Model(&models.User{}).
		Select("DATE(created_at) AS day, COUNT(*) AS count").
		Where("created_at >= ?", start).
		Group("day").
		Order("day ASC").
		Scan(&users)
	h.db.WithContext(c.Request.Context()).Model(&models.Device{}).
		Select("DATE(created_at) AS day, COUNT(*) AS count").
		Where("created_at >= ?", start).
		Group("day").
		Order("day ASC").
		Scan(&devices)
	h.db.WithContext(c.Request.Context()).Model(&models.ConnectionHistory{}).
		Select("DATE(created_at) AS day, COUNT(*) AS count").
		Where("created_at >= ?", start).
		Group("day").
		Order("day ASC").
		Scan(&conns)
	c.JSON(http.StatusOK, gin.H{
		"users":       users,
		"devices":     devices,
		"connections": conns,
	})
}

// GetConnections returns currently-active signaling sessions. For now we
// approximate "active" with the set of devices that have at least one WS
// presence signal.
func (h *AdminStatsHandler) GetConnections(c *gin.Context) {
	metrics := h.metrics.Snapshot()
	startToday := time.Now().UTC().Truncate(24 * time.Hour)
	todayConnections, _ := countSince(c.Request.Context(), h.db, &models.ConnectionHistory{}, startToday)
	var devs []models.Device
	h.db.WithContext(c.Request.Context()).Find(&devs)
	ids := make([]string, 0, len(devs))
	for _, d := range devs {
		ids = append(ids, d.DeviceID)
	}
	online := h.presence.BulkOnline(c.Request.Context(), ids)
	items := make([]gin.H, 0)
	for i := range devs {
		if !online[devs[i].DeviceID] {
			continue
		}
		items = append(items, gin.H{
			"device_id":    devs[i].DeviceID,
			"device_name": devs[i].DeviceName,
			"user_id":      devs[i].UserID,
			"last_seen_at": devs[i].LastSeenAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"items":                 items,
		"current_connections":   metrics.WebSocketConnections,
		"today_connections":     todayConnections,
		"websocket_connections": metrics.WebSocketConnections,
		"signal_hosts":          metrics.SignalHosts,
		"signal_clients":        metrics.SignalClients,
		"event_streams":         metrics.EventStreams,
		"api_requests_today":    metrics.APIRequestsToday,
	})
}

// countSince is a tiny helper so we don't repeat the same pattern for
// several models.
func countSince(ctx context.Context, db *gorm.DB, model interface{}, since time.Time) (int64, error) {
	var n int64
	err := db.WithContext(ctx).Model(model).Where("created_at >= ?", since).Count(&n).Error
	return n, err
}
