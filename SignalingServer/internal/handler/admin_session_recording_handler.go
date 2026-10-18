package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminSessionRecordingHandler serves the cross-user recording index under
// /v1/admin/recordings. It exposes read + delete only; recording metadata is
// still produced by the user API and the control engine.
type AdminSessionRecordingHandler struct {
	recordings    *service.SessionRecordingService
	audit         *service.AuditService
	lifecycle     *service.RecordingLifecycle
	retentionDays int
	staleMinutes  int
}

func NewAdminSessionRecordingHandler(
	recordings *service.SessionRecordingService,
	audit *service.AuditService,
	lifecycle *service.RecordingLifecycle,
	retentionDays int,
	staleMinutes int,
) *AdminSessionRecordingHandler {
	return &AdminSessionRecordingHandler{
		recordings:    recordings,
		audit:         audit,
		lifecycle:     lifecycle,
		retentionDays: retentionDays,
		staleMinutes:  staleMinutes,
	}
}

// Storage reports how much recording media is retained.
func (h *AdminSessionRecordingHandler) Storage(c *gin.Context) {
	usage, err := h.recordings.ArtifactUsage(c.Request.Context())
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"artifact_count":      usage.Count,
		"total_bytes":         usage.TotalBytes,
		"retention_days":      h.retentionDays,
		"stale_after_minutes": h.staleMinutes,
	})
}

// Sweep runs the retention/stale pass on demand. The periodic runner calls the
// same code path, so an admin can verify behaviour without waiting for a tick.
func (h *AdminSessionRecordingHandler) Sweep(c *gin.Context) {
	if h.lifecycle == nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "recording lifecycle is not configured")
		return
	}
	result, err := h.lifecycle.SweepOnce(c.Request.Context())
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	if h.audit != nil {
		h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "recording.sweep",
			"session_recording", "",
			fmt.Sprintf("stale=%d purged=%d freed_bytes=%d orphans=%d uploads=%d",
				result.StaleMarkedFailed, result.ArtifactsPurged, result.BytesFreed,
				result.OrphansRemoved, result.UploadsReaped),
			c.ClientIP())
	}
	c.JSON(http.StatusOK, gin.H{
		"stale_marked_failed": result.StaleMarkedFailed,
		"artifacts_purged":    result.ArtifactsPurged,
		"bytes_freed":         result.BytesFreed,
		"orphans_removed":     result.OrphansRemoved,
		"uploads_reaped":      result.UploadsReaped,
	})
}

func (h *AdminSessionRecordingHandler) List(c *gin.Context) {
	p := ParseCursor(c)
	cur, err := DecodeCursor(p.Cursor)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	var userID uint
	if raw := c.Query("user_id"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid user_id")
			return
		}
		userID = uint(parsed)
	}
	rows, total, err := h.recordings.ListAdmin(c.Request.Context(), service.SessionRecordingAdminListParams{
		AfterID:  cur.OffsetID,
		Limit:    p.Limit + 1,
		UserID:   userID,
		DeviceID: c.Query("device_id"),
		Status:   c.Query("status"),
		Search:   p.Search,
	})
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	next := ""
	if len(rows) > p.Limit {
		next = EncodeCursor(CursorPayload{OffsetID: rows[p.Limit-1].ID})
		rows = rows[:p.Limit]
	}
	c.JSON(http.StatusOK, NewCursorPage(rows, next, total))
}

func (h *AdminSessionRecordingHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid recording id")
		return
	}
	row, err := h.recordings.GetAdmin(c.Request.Context(), uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Recording not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, row)
}

func (h *AdminSessionRecordingHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid recording id")
		return
	}
	if err := h.recordings.DeleteAdmin(c.Request.Context(), uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Recording not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	if h.audit != nil {
		h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "recording.delete", "session_recording", c.Param("id"), "", c.ClientIP())
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
