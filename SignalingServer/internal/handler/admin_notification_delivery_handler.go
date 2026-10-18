package handler

import (
	"fmt"
	"net/http"
	"time"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminNotificationDeliveryHandler exposes the user-notification outbox to
// administrators:
//
//	GET  /v1/admin/notification-deliveries:stats
//	POST /v1/admin/notification-deliveries:sweep
//
// The periodic retention worker calls the same service method as `:sweep`, so
// an operator can run the purge on demand (and see what it did) instead of
// waiting for the next tick.
type AdminNotificationDeliveryHandler struct {
	deliveries           *service.NotificationDeliveryService
	audit                *service.AuditService
	retentionDays        int
	archiveRetentionDays int
}

func NewAdminNotificationDeliveryHandler(
	deliveries *service.NotificationDeliveryService,
	audit *service.AuditService,
	retentionDays, archiveRetentionDays int,
) *AdminNotificationDeliveryHandler {
	return &AdminNotificationDeliveryHandler{
		deliveries:           deliveries,
		audit:                audit,
		retentionDays:        retentionDays,
		archiveRetentionDays: archiveRetentionDays,
	}
}

// Stats reports how many outbox rows sit in each delivery state.
func (h *AdminNotificationDeliveryHandler) Stats(c *gin.Context) {
	if h.deliveries == nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "delivery store is not configured")
		return
	}
	counts, err := h.deliveries.CountByStatus(c.Request.Context())
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	var total int64
	for _, n := range counts {
		total += n
	}
	// The confirmation half of the loop: `acked` is how many rows somebody
	// confirmed, `unacked` is what endpoints accepted but nobody did. A channel
	// that returns 200 and drops the event only shows up in the second number.
	acked, unacked, err := h.deliveries.CountAcknowledgements(c.Request.Context())
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"total":                  total,
		"by_status":              counts,
		"acked":                  acked,
		"unacked":                unacked,
		"retention_days":         h.retentionDays,
		"archive_retention_days": h.archiveRetentionDays,
		"archive_dir":            h.deliveries.ArchiveDir(),
	})
}

// notificationSweepReq lets an operator run one purge with a different window
// than the configured retention (handy for testing without changing config).
type notificationSweepReq struct {
	OlderThanDays        *int `json:"older_than_days"`
	ArchiveRetentionDays *int `json:"archive_retention_days"`
}

// Sweep runs the retention pass on demand.
func (h *AdminNotificationDeliveryHandler) Sweep(c *gin.Context) {
	if h.deliveries == nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "delivery store is not configured")
		return
	}
	var req notificationSweepReq
	// An empty body means "use the configured retention"; only parse when the
	// caller actually sent something.
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
			return
		}
	}

	days := h.retentionDays
	if req.OlderThanDays != nil {
		days = *req.OlderThanDays
	}
	archiveDays := h.archiveRetentionDays
	if req.ArchiveRetentionDays != nil {
		archiveDays = *req.ArchiveRetentionDays
	}
	if days <= 0 && archiveDays <= 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest,
			"retention is not configured; set NOTIFICATION_RETENTION_DAYS / NOTIFICATION_ARCHIVE_RETENTION_DAYS or pass older_than_days / archive_retention_days")
		return
	}

	var purged int64
	cutoff := ""
	if days > 0 {
		cut := time.Now().UTC().AddDate(0, 0, -days)
		cutoff = cut.Format(time.RFC3339)
		removed, err := h.deliveries.PurgeTerminal(c.Request.Context(), cut)
		if err != nil {
			// An archive failure lands here: nothing was deleted, so say so.
			ProblemInternal(c, "purge failed, nothing deleted: "+err.Error())
			return
		}
		purged = removed
	}

	archiveDir := h.deliveries.ArchiveDir()
	// The daily archive files are a second, independent bound: rows may be kept
	// forever while the JSONL they were purged into is not.
	var archiveResult service.ArchiveSweepResult
	if archiveDays > 0 {
		if archiveDir == "" {
			ProblemBadRequest(c, ProblemCodeInvalidRequest,
				"archive retention was requested but NOTIFICATION_ARCHIVE_DIR is not configured")
			return
		}
		result, err := h.deliveries.SweepArchiveDetailed(archiveDays)
		if err != nil {
			ProblemInternal(c, "archive sweep failed: "+err.Error())
			return
		}
		archiveResult = result
	}
	if h.audit != nil {
		h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "notification_delivery.sweep",
			"notification_delivery", "",
			fmt.Sprintf("older_than_days=%d cutoff=%s purged=%d archive_retention_days=%d archive_removed=%d archive_moved=%d archive_deleted=%d archive=%s cold_archive=%s",
				days, cutoff, purged, archiveDays, archiveResult.Removed,
				archiveResult.Moved, archiveResult.Deleted, archiveDir, h.deliveries.ArchiveColdDir()),
			c.ClientIP())
	}
	c.JSON(http.StatusOK, gin.H{
		"purged":                 purged,
		"older_than_days":        days,
		"cutoff":                 cutoff,
		"archive_retention_days": archiveDays,
		"archive_removed":        archiveResult.Removed,
		"archive_moved":          archiveResult.Moved,
		"archive_deleted":        archiveResult.Deleted,
		"archive_dir":            archiveDir,
		"archive_cold_dir":       h.deliveries.ArchiveColdDir(),
	})
}
