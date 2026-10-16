package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

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
	devices    *service.DeviceService
	revoker    SessionRevoker
	audit      *service.AuditService
}

func NewRemoteSessionHandler(
	sessions *service.RemoteSessionService,
	recordings *service.SessionRecordingService,
	access *service.DeviceAccessService,
	devices *service.DeviceService,
	revoker SessionRevoker,
	audit *service.AuditService,
) *RemoteSessionHandler {
	return &RemoteSessionHandler{
		sessions: sessions, recordings: recordings, access: access, devices: devices,
		revoker: revoker, audit: audit,
	}
}

// SessionRevoker closes the live signaling socket behind a session id, wherever
// in the fleet it happens to live. It is implemented by RealtimeHandler, which
// is the only thing that knows the socket topology; the read paths depend on the
// narrow interface so they can be built and tested without a live WebSocket.
type SessionRevoker interface {
	// RevokeSession closes the socket on this instance and reports whether it
	// found one.
	RevokeSession(sessionID string) bool
	// AskInstanceRevoke asks the instance that owns a session to close it. It
	// only publishes the request; the caller confirms by watching the row.
	AskInstanceRevoke(ctx context.Context, instanceID, sessionID string) error
	// InstanceID is this process's identity, the value session rows record.
	InstanceID() string
}

// sessionRevokeTimeout bounds how long a cross-instance revoke waits for the
// owning instance to actually close the socket. Generous for one round trip and
// a socket teardown; short enough that an operator is not left waiting when the
// owning instance is gone.
const sessionRevokeTimeout = 3 * time.Second

// revokeSessionSocket is the shared body of the owner and administrator revoke
// actions. Both callers have already loaded the row and established that the
// caller may act on it, so the only work left is closing the socket - and
// saying plainly when there was nothing to close, rather than reporting a
// success that did not happen.
func revokeSessionSocket(
	c *gin.Context,
	sessions *service.RemoteSessionService,
	revoker SessionRevoker,
	row *models.RemoteSession,
) bool {
	if revoker == nil {
		ProblemInternal(c, "session revocation is not configured")
		return false
	}
	if !row.Active {
		ProblemConflict(c, ProblemCodeConflict, "Session is already over")
		return false
	}
	// This instance first: it is the common case, and it costs nothing.
	if revoker.RevokeSession(row.SessionID) {
		return true
	}
	// The socket is elsewhere. If the row names its instance, ask that one and
	// wait for the row to actually close: a published command that nobody acted
	// on must not come back as a success. Rows with no instance id predate the
	// column, and a row naming this instance that we could not find means the
	// socket is already gone - both are "not connected here".
	if row.InstanceID == "" || row.InstanceID == revoker.InstanceID() {
		ProblemConflict(c, ProblemCodeConflict, "Session is not connected to this instance")
		return false
	}
	if err := revoker.AskInstanceRevoke(c.Request.Context(), row.InstanceID, row.SessionID); err != nil {
		ProblemInternal(c, err.Error())
		return false
	}
	if waitForSessionEnd(c.Request.Context(), sessions, row.SessionID, sessionRevokeTimeout) {
		return true
	}
	ProblemConflict(c, ProblemCodeConflict,
		"The instance holding the session did not close it in time")
	return false
}

// waitForSessionEnd polls the session row until it is no longer active. The
// database is the only shared truth about whether a socket closed: the
// instance that owns it stamps the row in its own teardown, and that stamp is
// what turns "we asked politely" into "it happened".
func waitForSessionEnd(ctx context.Context, sessions *service.RemoteSessionService, sessionID string, timeout time.Duration) bool {
	if sessions == nil {
		return false
	}
	deadline := time.Now().Add(timeout)
	for {
		if row, err := sessions.Get(ctx, sessionID); err == nil && !row.Active {
			return true
		}
		if !time.Now().Before(deadline) {
			return false
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// maxSessionsPerDetailPage bounds how many joined rows one list page may carry.
// The arrays are a page of the related records, not a promise of completeness;
// a client that needs one session in full reads its detail.
const maxSessionsPerDetailPage = 500

// parseStatusFilter reads ?status= the same way for every session list.
//
// It is spelled as a word rather than reusing `active=true`: the lists answer
// "who is in right now" (active) and "what happened" (ended) with the same
// shape, and a tri-state reads better than a query param whose absence means
// "either".
func parseStatusFilter(c *gin.Context) (*bool, bool) {
	switch strings.ToLower(strings.TrimSpace(c.Query("status"))) {
	case "", "all":
		return nil, true
	case "active":
		open := true
		return &open, true
	case "ended":
		closed := false
		return &closed, true
	default:
		return nil, false
	}
}

// detailEnvelope is the one shape a session-with-its-records is ever returned
// in, so the detail read and the device-scoped list cannot drift apart.
func detailEnvelope(
	session *models.RemoteSession,
	recordings []models.SessionRecording,
	accessLog []models.DeviceAccessLog,
) gin.H {
	return gin.H{
		"session":    session,
		"recordings": recordings,
		"access_log": accessLog,
		"counts": gin.H{
			"recordings": len(recordings),
			"access_log": len(accessLog),
		},
	}
}

func (h *RemoteSessionHandler) List(c *gin.Context) {
	uid := middleware.MustUserID(c)
	p := ParseCursor(c)
	cur, err := DecodeCursor(p.Cursor)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	active, ok := parseStatusFilter(c)
	if !ok {
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
	c.JSON(http.StatusOK, detailEnvelope(row, recordings, accessLog))
}

// ListForDevice handles GET /v1/me/devices/:device_id/sessions: that machine's
// recent control sessions, each already joined with the records that share its
// id - the shape a device page needs, so a client does not have to know a
// session id (or make a second request per row) to reach the join.
//
// A device that is not the caller's is a 403 here rather than the 404 the
// detail read uses: the caller named a device, and this mirrors
// /v1/me/devices/:device_id/access-logs. A session id is different - it is
// guessable-looking and belongs to nobody in particular until looked up, so
// there a 404 keeps a caller from probing which ids exist.
func (h *RemoteSessionHandler) ListForDevice(c *gin.Context) {
	uid := middleware.MustUserID(c)
	deviceID := strings.TrimSpace(c.Param("device_id"))
	if deviceID == "" {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "device_id is required")
		return
	}
	if h.devices == nil {
		ProblemInternal(c, "device lookup is not configured")
		return
	}
	device, err := h.devices.GetByDeviceID(c.Request.Context(), deviceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeDeviceNotFound, "Device not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	if device.UserID == nil || *device.UserID != uid {
		ProblemForbidden(c, ProblemCodeForbidden, "Device is not owned by this user")
		return
	}

	active, ok := parseStatusFilter(c)
	if !ok {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "status must be active, ended or all")
		return
	}
	p := ParseCursor(c)
	cur, err := DecodeCursor(p.Cursor)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	rows, err := h.sessions.List(c.Request.Context(), service.RemoteSessionFilter{
		UserID:   uid,
		DeviceID: deviceID,
		Active:   active,
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

	details, err := h.attachRelated(c, uid, deviceID, rows)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, NewCursorPage(details, next, -1))
}

// attachRelated joins a page of sessions with their recordings and access-log
// rows in two batched queries and groups them in memory.
//
// Two queries per page rather than two per row is the whole reason this method
// exists: a device page shows fifty sessions, and fifty round trips to paint one
// screen is how a control plane gets a reputation for being slow.
func (h *RemoteSessionHandler) attachRelated(
	c *gin.Context,
	uid uint,
	deviceID string,
	sessions []models.RemoteSession,
) ([]gin.H, error) {
	details := make([]gin.H, 0, len(sessions))
	if len(sessions) == 0 {
		return details, nil
	}
	ids := make([]string, 0, len(sessions))
	for _, row := range sessions {
		if row.SessionID != "" {
			ids = append(ids, row.SessionID)
		}
	}
	recordings := []models.SessionRecording{}
	if h.recordings != nil && len(ids) > 0 {
		var err error
		recordings, err = h.recordings.ListForSessions(c.Request.Context(), uid, ids, maxSessionsPerDetailPage)
		if err != nil {
			return nil, err
		}
	}
	accessLog := []models.DeviceAccessLog{}
	if h.access != nil && len(ids) > 0 {
		var err error
		accessLog, err = h.access.ListForSessions(c.Request.Context(), deviceID, ids, maxSessionsPerDetailPage)
		if err != nil {
			return nil, err
		}
	}
	recordingsBySession := map[string][]models.SessionRecording{}
	for _, row := range recordings {
		recordingsBySession[row.SessionID] = append(recordingsBySession[row.SessionID], row)
	}
	accessBySession := map[string][]models.DeviceAccessLog{}
	for _, row := range accessLog {
		accessBySession[row.SessionID] = append(accessBySession[row.SessionID], row)
	}
	for _, row := range sessions {
		sessionRecordings := recordingsBySession[row.SessionID]
		if sessionRecordings == nil {
			sessionRecordings = []models.SessionRecording{}
		}
		sessionAccess := accessBySession[row.SessionID]
		if sessionAccess == nil {
			sessionAccess = []models.DeviceAccessLog{}
		}
		details = append(details, detailEnvelope(&row, sessionRecordings, sessionAccess))
	}
	return details, nil
}

// Revoke handles POST /v1/me/remote-sessions/:session_id/revoke: the owner ends
// a session on their own machine.
//
// Ownership is checked with the same read the detail endpoint uses, so a
// session belonging to somebody else is a 404 for the same reason - the caller
// learns nothing about which ids exist. The row then has to be live and
// connected here; an already-finished or not-here session is a 409.
func (h *RemoteSessionHandler) Revoke(c *gin.Context) {
	uid := middleware.MustUserID(c)
	sessionID := strings.TrimSpace(c.Param("session_id"))
	if sessionID == "" {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "session_id is required")
		return
	}
	row, err := h.sessions.GetForUser(c.Request.Context(), uid, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Session not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	if !revokeSessionSocket(c, h.sessions, h.revoker, row) {
		return
	}
	if h.audit != nil {
		h.audit.LogActor(c.Request.Context(), uid, "session.revoke", "remote_session", row.SessionID,
			"device="+row.DeviceID+" client="+row.ClientID, c.ClientIP())
	}
	c.JSON(http.StatusOK, gin.H{
		"session_id": row.SessionID,
		"device_id":  row.DeviceID,
		"revoked":    true,
	})
}
