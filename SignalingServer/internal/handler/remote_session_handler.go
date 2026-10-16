package handler

import (
	"errors"
	"net/http"
	"strings"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RemoteSessionHandler serves GET /v1/me/remote-sessions: the owner's view of
// who is in their machines right now, and who has been.
//
// It is deliberately not /v1/me/sessions - that path is the caller's own login
// sessions, a different resource with a different lifetime, and overloading it
// would make "revoke this session" ambiguous.
type RemoteSessionHandler struct {
	sessions   *service.RemoteSessionService
	recordings *service.SessionRecordingService
	access     *service.DeviceAccessService
}

func NewRemoteSessionHandler(
	sessions *service.RemoteSessionService,
	recordings *service.SessionRecordingService,
	access *service.DeviceAccessService,
) *RemoteSessionHandler {
	return &RemoteSessionHandler{sessions: sessions, recordings: recordings, access: access}
}

func (h *RemoteSessionHandler) List(c *gin.Context) {
	uid := middleware.MustUserID(c)
	p := ParseCursor(c)
	cur, err := DecodeCursor(p.Cursor)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	// `status` is spelled as a word rather than reusing `active=true`: the list
	// answers "who is in right now" (active) and "what happened" (ended) with the
	// same shape, and a tri-state reads better than a query param whose absence
	// means "either".
	var active *bool
	switch strings.ToLower(strings.TrimSpace(c.Query("status"))) {
	case "", "all":
	case "active":
		open := true
		active = &open
	case "ended":
		closed := false
		active = &closed
	default:
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "status must be active, ended or all")
		return
	}
	rows, err := h.sessions.List(c.Request.Context(), service.RemoteSessionFilter{
		UserID:   uid,
		DeviceID: strings.TrimSpace(c.Query("device_id")),
		Active:   active,
		// One row past the page is how the handler learns a next page exists
		// without a second COUNT query.
		Limit:    p.Limit + 1,
		OffsetID: cur.OffsetID,
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

// Get answers "what happened in this session": the session itself plus the
// records that share its session id - the recordings the controller produced
// and the access-log entries the server wrote when it authenticated.
//
// The three tables always existed; what did not exist was a way to read them
// together, which is the difference between a session model and three unrelated
// logs that happen to carry the same string.
func (h *RemoteSessionHandler) Get(c *gin.Context) {
	uid := middleware.MustUserID(c)
	sessionID := strings.TrimSpace(c.Param("session_id"))
	if sessionID == "" {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "session_id is required")
		return
	}
	row, err := h.sessions.GetForUser(c.Request.Context(), uid, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Not "forbidden": a session id the caller does not own must not be
			// distinguishable from one that never existed.
			ProblemNotFound(c, ProblemCodeNotFound, "Session not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}

	recordings := []models.SessionRecording{}
	if h.recordings != nil {
		recordings, err = h.recordings.ListForSession(c.Request.Context(), uid, row.SessionID, 100)
		if err != nil {
			ProblemInternal(c, err.Error())
			return
		}
	}
	accessLog := []models.DeviceAccessLog{}
	if h.access != nil {
		accessLog, err = h.access.ListForSession(c.Request.Context(), row.DeviceID, row.SessionID, 100)
		if err != nil {
			ProblemInternal(c, err.Error())
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"session":    row,
		"recordings": recordings,
		"access_log": accessLog,
		"counts": gin.H{
			"recordings": len(recordings),
			"access_log": len(accessLog),
		},
	})
}
