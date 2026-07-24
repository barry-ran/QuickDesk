package handler

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"quickdesk/signaling/internal/httpx"
	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminLogsHandler exposes an explicit allow-list of operational log files.
// PostgreSQL and Redis logs are opt-in because they are normally owned by
// separate containers; set POSTGRES_LOG_FILE / REDIS_LOG_FILE to bind them.
type AdminLogsHandler struct {
	audit *service.AuditService
	db    *gorm.DB
	files map[string]string
}

func NewAdminLogsHandler(audit *service.AuditService, db *gorm.DB, logDir, postgresLogFile, redisLogFile string) *AdminLogsHandler {
	files := map[string]string{
		"signaling.log": filepath.Join(logDir, "signaling.log"),
	}
	if postgresLogFile != "" {
		files["postgres.log"] = postgresLogFile
	}
	if redisLogFile != "" {
		files["redis.log"] = redisLogFile
	}
	return &AdminLogsHandler{audit: audit, db: db, files: files}
}

func (h *AdminLogsHandler) requireSuperAdmin(c *gin.Context) (*models.AdminUser, bool) {
	var admin models.AdminUser
	if err := h.db.WithContext(c.Request.Context()).First(&admin, middleware.MustAdminID(c)).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemInternal(c, err.Error())
		} else {
			httpx.Forbidden(c, httpx.CodeForbidden, "Super administrator access is required")
		}
		return nil, false
	}
	if admin.Role != "super_admin" {
		httpx.Forbidden(c, httpx.CodeForbidden, "Super administrator access is required")
		return nil, false
	}
	return &admin, true
}

func (h *AdminLogsHandler) List(c *gin.Context) {
	if _, ok := h.requireSuperAdmin(c); !ok {
		return
	}
	items := make([]gin.H, 0, len(h.files))
	for name, path := range h.files {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		items = append(items, gin.H{"name": name, "size_bytes": info.Size(), "modified_at": info.ModTime()})
	}
	sort.Slice(items, func(i, j int) bool { return items[i]["name"].(string) < items[j]["name"].(string) })
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *AdminLogsHandler) Download(c *gin.Context) {
	admin, ok := h.requireSuperAdmin(c)
	if !ok {
		return
	}
	name := c.Param("name")
	path, found := h.files[name]
	if !found {
		ProblemNotFound(c, ProblemCodeNotFound, "Log file not configured")
		return
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		ProblemNotFound(c, ProblemCodeNotFound, "Log file unavailable")
		return
	}
	h.audit.Log(c.Request.Context(), admin.ID, admin.Username, "log.download", "log_file", name, fmt.Sprintf("size_bytes=%d", info.Size()), c.ClientIP())
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	c.File(path)
}

func (h *AdminLogsHandler) ExportAudit(c *gin.Context) {
	admin, ok := h.requireSuperAdmin(c)
	if !ok {
		return
	}
	format := c.DefaultQuery("format", "csv")
	if format != "csv" && format != "json" {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "format must be csv or json")
		return
	}
	logs, _, err := h.audit.List(c.Request.Context(), service.AuditListParams{
		Action: c.Query("action"), AdminUsername: c.Query("admin"),
		DateFrom: c.Query("date_from"), DateTo: c.Query("date_to"), Limit: 100000,
	})
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	h.audit.Log(c.Request.Context(), admin.ID, admin.Username, "audit.export", "audit_log", "", "format="+format, c.ClientIP())
	stamp := time.Now().UTC().Format("20060102T150405Z")
	if format == "json" {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", "audit-logs-"+stamp+".json"))
		c.JSON(http.StatusOK, logs)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", "audit-logs-"+stamp+".csv"))
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"id", "admin_id", "admin_username", "action", "resource_type", "resource_id", "details", "ip", "created_at"})
	for _, entry := range logs {
		_ = writer.Write([]string{fmt.Sprint(entry.ID), fmt.Sprint(entry.AdminID), entry.AdminUsername, entry.Action, entry.ResourceType, entry.ResourceID, entry.Details, entry.IP, entry.CreatedAt.UTC().Format(time.RFC3339)})
	}
	writer.Flush()
}
