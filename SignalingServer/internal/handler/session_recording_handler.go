package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SessionRecordingHandler serves the user-scoped recording index under
// /v1/me/recordings. Media files are produced by the control engine; this
// surface owns discovery, ownership, and lifecycle metadata.
type SessionRecordingHandler struct {
	recordings *service.SessionRecordingService
}

func NewSessionRecordingHandler(recordings *service.SessionRecordingService) *SessionRecordingHandler {
	return &SessionRecordingHandler{recordings: recordings}
}

type createSessionRecordingReq struct {
	DeviceID        string     `json:"device_id" binding:"required"`
	DeviceName      string     `json:"device_name"`
	SessionID       string     `json:"session_id"`
	Status          string     `json:"status" binding:"required"`
	Format          string     `json:"format"`
	ArtifactPath    string     `json:"artifact_path"`
	ArtifactURL     string     `json:"artifact_url"`
	ThumbnailPath   string     `json:"thumbnail_path"`
	DurationSeconds int        `json:"duration_seconds"`
	SizeBytes       int64      `json:"size_bytes"`
	StartedAt       *time.Time `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at"`
}

func (h *SessionRecordingHandler) Create(c *gin.Context) {
	uid := middleware.MustUserID(c)
	var req createSessionRecordingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	var startedAt time.Time
	if req.StartedAt != nil {
		startedAt = *req.StartedAt
	}
	row, err := h.recordings.Create(c.Request.Context(), uid, service.CreateSessionRecordingInput{
		DeviceID:        req.DeviceID,
		DeviceName:      req.DeviceName,
		SessionID:       req.SessionID,
		Status:          req.Status,
		Format:          req.Format,
		ArtifactPath:    req.ArtifactPath,
		ArtifactURL:     req.ArtifactURL,
		ThumbnailPath:   req.ThumbnailPath,
		DurationSeconds: req.DurationSeconds,
		SizeBytes:       req.SizeBytes,
		StartedAt:       startedAt,
		EndedAt:         req.EndedAt,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidRecordingStatus) {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
			return
		}
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	c.JSON(http.StatusCreated, row)
}

func (h *SessionRecordingHandler) List(c *gin.Context) {
	uid := middleware.MustUserID(c)
	p := ParseCursor(c)
	cur, err := DecodeCursor(p.Cursor)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	rows, err := h.recordings.List(c.Request.Context(), uid, service.SessionRecordingListParams{
		BeforeID: cur.OffsetID,
		Limit:    p.Limit + 1,
	})
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	var next string
	if len(rows) > p.Limit {
		next = EncodeCursor(CursorPayload{OffsetID: rows[p.Limit-1].ID})
		rows = rows[:p.Limit]
	}
	c.JSON(http.StatusOK, NewCursorPage(rows, next, -1))
}

func (h *SessionRecordingHandler) Get(c *gin.Context) {
	uid := middleware.MustUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid recording id")
		return
	}
	row, err := h.recordings.Get(c.Request.Context(), uid, uint(id))
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

func (h *SessionRecordingHandler) Delete(c *gin.Context) {
	uid := middleware.MustUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid recording id")
		return
	}
	if err := h.recordings.Delete(c.Request.Context(), uid, uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Recording not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
