package handler

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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
	audit      *service.AuditService
	db         *gorm.DB
	files      map[string]string
	logQuery   *service.LogQueryService
	aggregated *service.AggregatedLogQueryService
}

func NewAdminLogsHandler(
	audit *service.AuditService,
	db *gorm.DB,
	logDir, postgresLogFile, redisLogFile string,
	logQuery *service.LogQueryService,
	aggregated *service.AggregatedLogQueryService,
) *AdminLogsHandler {
	files := map[string]string{}
	if strings.TrimSpace(logDir) != "" {
		files["signaling.log"] = filepath.Join(logDir, "signaling.log")
	}
	if postgresLogFile != "" {
		files["postgres.log"] = postgresLogFile
	}
	if redisLogFile != "" {
		files["redis.log"] = redisLogFile
	}
	// Make every listed file searchable too. Before this the search only ever
	// scanned the application log, so a configured PostgreSQL or Redis log
	// could be downloaded but not queried in place.
	if logQuery != nil {
		for name, path := range files {
			logQuery.AddFile(name, path)
		}
	}
	return &AdminLogsHandler{
		audit:      audit,
		db:         db,
		files:      files,
		logQuery:   logQuery,
		aggregated: aggregated,
	}
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

// Search answers `GET /v1/admin/system/logs:search`.
//
//	?q=          case-insensitive substring over the whole line
//	&match=      full-text expression (words ANDed, "a phrase", or, -negation)
//	&component=  whole-token match on the structured `component=` field
//	&since=&until=  RFC3339 bounds on the line's own timestamp (the logger
//	                writes UTC)
//	&limit=      default 200, clamped to 2000
//	&scope=      `local` (default) reads this instance's files; `all` reads the
//	             lines every instance shipped into the shared database, so a
//	             multi-instance deployment can be searched from one place.
//	             `match=` works on both: the file scanner compiles the same
//	             expression itself instead of deferring to Postgres
//	&instance=   with scope=all, narrow to one instance id
//
// Downloading the whole file works too, but it means pulling tens of megabytes
// through a browser to answer "what happened around 3pm". This reads the file
// backwards with a byte budget, so the newest matching lines come back without
// moving the whole log. The search itself is audited: "who looked at the logs,
// and for what" is worth knowing.
func (h *AdminLogsHandler) Search(c *gin.Context) {
	admin, ok := h.requireSuperAdmin(c)
	if !ok {
		return
	}
	if h.logQuery == nil || len(h.logQuery.Sources()) == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "log directory is not configured")
		return
	}
	scope := strings.ToLower(strings.TrimSpace(c.Query("scope")))
	if scope == "" {
		scope = service.LogScopeLocal
	}
	if scope != service.LogScopeLocal && scope != service.LogScopeAll {
		ProblemBadRequest(c, ProblemCodeInvalidRequest,
			fmt.Sprintf("scope must be %q or %q", service.LogScopeLocal, service.LogScopeAll))
		return
	}
	file := strings.TrimSpace(c.Query("file"))
	if file == "" && scope == service.LogScopeLocal {
		file = service.DefaultLogSourceName
	}
	if scope == service.LogScopeLocal && file != service.AllLogSources {
		if _, ok := h.files[file]; !ok {
			ProblemBadRequest(c, ProblemCodeInvalidRequest,
				fmt.Sprintf("unknown log file %q (use %q to search every configured source)",
					file, service.AllLogSources))
			return
		}
	}
	limit := service.DefaultLogQueryLimit
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "limit must be a number")
			return
		}
		if parsed < 0 {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "limit must be zero (default) or positive")
			return
		}
		if parsed > 0 {
			// Clamp rather than reject: the caller asked for lines, and the
			// response reports the effective cap so nothing is hidden.
			limit = min(parsed, service.MaxLogQueryLimit)
		}
	}
	since, err := parseOptionalRFC3339("since", c.Query("since"))
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	until, err := parseOptionalRFC3339("until", c.Query("until"))
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if since != nil && until != nil && since.After(*until) {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "since must not be later than until")
		return
	}
	if scope == service.LogScopeAll {
		if h.aggregated == nil {
			ProblemBadRequest(c, ProblemCodeInvalidRequest,
				"cross-instance log aggregation is not enabled; set INSTANCE_LOG_SINK=true to ship this instance's log into the shared store")
			return
		}
		result, err := h.aggregated.Query(c.Request.Context(), service.AggregatedLogQuery{
			Query:     strings.TrimSpace(c.Query("q")),
			Match:     strings.TrimSpace(c.Query("match")),
			Component: strings.TrimSpace(c.Query("component")),
			Since:     since,
			Until:     until,
			Limit:     limit,
			Instance:  strings.TrimSpace(c.Query("instance")),
			Source:    file,
		})
		if err != nil {
			ProblemInternal(c, err.Error())
			return
		}
		filter := fmt.Sprintf("scope=%q q=%q match=%q component=%q instance=%q file=%q limit=%d truncated=%t instances=%d",
			scope, c.Query("q"), c.Query("match"), c.Query("component"), c.Query("instance"), file, limit,
			result.Truncated, len(result.Instances))
		h.audit.Log(c.Request.Context(), admin.ID, admin.Username, "log.search", "log_store", "aggregated", filter, c.ClientIP())
		c.JSON(http.StatusOK, gin.H{
			"items":         result.Entries,
			"count":         len(result.Entries),
			"limit":         limit,
			"scope":         scope,
			"file":          file,
			"instance":      strings.TrimSpace(c.Query("instance")),
			"match":         result.Match,
			"instances":     result.Instances,
			"truncated":     result.Truncated,
			"stored_lines":  result.Stored,
			"dropped_lines": result.Dropped,
		})
		return
	}

	result, err := h.logQuery.Query(c.Request.Context(), service.LogQuery{
		Query:     strings.TrimSpace(c.Query("q")),
		Match:     strings.TrimSpace(c.Query("match")),
		Component: strings.TrimSpace(c.Query("component")),
		Since:     since,
		Until:     until,
		Limit:     limit,
		File:      file,
	})
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	filter := fmt.Sprintf("file=%q q=%q match=%q component=%q limit=%d scanned_bytes=%d truncated=%t",
		file, c.Query("q"), c.Query("match"), c.Query("component"), limit, result.ScannedBytes, result.Truncated)
	h.audit.Log(c.Request.Context(), admin.ID, admin.Username, "log.search", "log_file", file, filter, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"items":         result.Entries,
		"count":         len(result.Entries),
		"limit":         limit,
		"scope":         scope,
		"file":          file,
		"match":         result.Match,
		"files":         result.Files,
		"scanned_bytes": result.ScannedBytes,
		"scanned_lines": result.ScannedLines,
		"truncated":     result.Truncated,
	})
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
	actorUserID, err := parseActorUserID(c.Query("actor_user_id"))
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	logs, _, err := h.audit.List(c.Request.Context(), service.AuditListParams{
		Action: c.Query("action"), AdminUsername: c.Query("admin"),
		Username:    strings.TrimSpace(c.Query("username")),
		ActorUserID: actorUserID,
		DateFrom:    c.Query("date_from"), DateTo: c.Query("date_to"), Limit: 100000,
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
	// actor_user_id belongs in the export for the same reason it exists in the
	// table: `recording.delete` is written by owners and admins alike, and
	// without this column the CSV cannot tell the two apart.
	_ = writer.Write([]string{"id", "admin_id", "admin_username", "actor_user_id", "action", "resource_type", "resource_id", "details", "ip", "created_at"})
	for _, entry := range logs {
		_ = writer.Write([]string{fmt.Sprint(entry.ID), fmt.Sprint(entry.AdminID), entry.AdminUsername, fmt.Sprint(entry.ActorUserID), entry.Action, entry.ResourceType, entry.ResourceID, entry.Details, entry.IP, entry.CreatedAt.UTC().Format(time.RFC3339)})
	}
	writer.Flush()
}
