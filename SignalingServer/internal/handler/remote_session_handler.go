package handler

import (
	"context"
	"errors"
	"fmt"
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

// revokeOutcome says what happened to one revoke attempt. It exists so the
// single-session action and the batch action can share one implementation: the
// first turns the outcome into a problem response, the second reports it per
// row and keeps going.
type revokeOutcome int

const (
	revokeDone revokeOutcome = iota
	revokeNotConfigured
	revokeAlreadyOver
	revokeNotConnectedHere
	revokePublishFailed
	revokeNotConfirmed
)

// revokeOne closes one session - here if this instance holds it, otherwise by
// asking the instance that does - and reports the outcome plus a detail string.
// It never writes a response, because a batch caller needs every row's result
// rather than the first failure.
func revokeOne(
	ctx context.Context,
	sessions *service.RemoteSessionService,
	revoker SessionRevoker,
	row *models.RemoteSession,
) (revokeOutcome, string) {
	if revoker == nil {
		return revokeNotConfigured, "session revocation is not configured"
	}
	if !row.Active {
		return revokeAlreadyOver, "Session is already over"
	}
	// This instance first: it is the common case, and it costs nothing.
	if revoker.RevokeSession(row.SessionID) {
		// The socket is closed by now, but the row is stamped by that socket's
		// teardown a moment later. Confirm the stamp too, so "200 means it
		// happened" holds for the local path exactly as it does for the remote
		// one - a caller that reads the ended list straight away must find it.
		// A caller with no session store to read (a unit test) takes the socket
		// close as the answer.
		if sessions == nil || waitForSessionEnd(ctx, sessions, row.SessionID, sessionRevokeTimeout) {
			return revokeDone, ""
		}
		return revokeNotConfirmed, "The session socket closed but its row was not updated in time"
	}
	// The socket is elsewhere. If the row names its instance, ask that one and
	// wait for the row to actually close: a published command that nobody acted
	// on must not come back as a success. Rows with no instance id predate the
	// column, and a row naming this instance that we could not find means the
	// socket is already gone - both are "not connected here".
	if row.InstanceID == "" || row.InstanceID == revoker.InstanceID() {
		return revokeNotConnectedHere, "Session is not connected to this instance"
	}
	if err := revoker.AskInstanceRevoke(ctx, row.InstanceID, row.SessionID); err != nil {
		return revokePublishFailed, err.Error()
	}
	if waitForSessionEnd(ctx, sessions, row.SessionID, sessionRevokeTimeout) {
		return revokeDone, ""
	}
	return revokeNotConfirmed, "The instance holding the session did not close it in time"
}

// revokeSessionSocket is the single-session body of the owner and administrator
// revoke actions: it runs revokeOne and turns a failure into the problem the
// caller should see - 500 when the command never left, 409 when it could not be
// carried out - rather than reporting a success that did not happen.
func revokeSessionSocket(
	c *gin.Context,
	sessions *service.RemoteSessionService,
	revoker SessionRevoker,
	row *models.RemoteSession,
) bool {
	outcome, detail := revokeOne(c.Request.Context(), sessions, revoker, row)
	switch outcome {
	case revokeDone:
		return true
	case revokeNotConfigured, revokePublishFailed:
		ProblemInternal(c, detail)
	default:
		ProblemConflict(c, ProblemCodeConflict, detail)
	}
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

// maxBatchRevokeSessions bounds one device-wide revoke. It matches the list page
// cap: a machine with more concurrent controllers than this is not a shape the
// product has, and an unbounded sweep would let one request hold an operator for
// a long time. When the cap is hit the response says so rather than pretending
// the device is now empty.
const maxBatchRevokeSessions = 200

// revokeDeviceSessions closes every live session on one device and reports the
// outcome per session. It is shared by the owner and the administrator route:
// both have already established that the caller may act on the device, so all
// that is left is doing the work and being honest about partial success.
//
// A batch does not fail as a whole. One session on a dead instance must not stop
// the other nine from being kicked, so each row carries its own result and the
// caller sees exactly which ones did not go.
func revokeDeviceSessions(
	ctx context.Context,
	sessions *service.RemoteSessionService,
	revoker SessionRevoker,
	deviceID string,
	limit int,
) (revoked int, failed int, truncated bool, results []gin.H, err error) {
	if limit <= 0 {
		limit = maxBatchRevokeSessions
	}
	open := true
	rows, err := sessions.List(ctx, service.RemoteSessionFilter{
		DeviceID: deviceID,
		Active:   &open,
		// One row past the cap, so "there are more" is answerable without a
		// second count.
		Limit: limit + 1,
	})
	if err != nil {
		return 0, 0, false, nil, err
	}
	if len(rows) > limit {
		rows = rows[:limit]
		truncated = true
	}
	results = make([]gin.H, 0, len(rows))
	for i := range rows {
		outcome, detail := revokeOne(ctx, sessions, revoker, &rows[i])
		entry := gin.H{"session_id": rows[i].SessionID}
		if outcome == revokeDone {
			revoked++
			entry["revoked"] = true
		} else {
			failed++
			entry["revoked"] = false
			entry["reason"] = detail
		}
		results = append(results, entry)
	}
	return revoked, failed, truncated, results, nil
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

// RevokeForDevice handles POST /v1/me/devices/:device_id/sessions:revoke: the
// owner's "throw everyone out of this machine".
//
// Ownership answers follow the sibling routes - 404 for an unknown device, 403
// for somebody else's - and the response reports the outcome per session rather
// than a single pass/fail, because a batch with one unreachable session is a
// normal outcome and the operator needs to see which one it was.
func (h *RemoteSessionHandler) RevokeForDevice(c *gin.Context) {
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
	revoked, failed, truncated, results, err := revokeDeviceSessions(
		c.Request.Context(), h.sessions, h.revoker, deviceID, maxBatchRevokeSessions)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	if h.audit != nil && revoked > 0 {
		h.audit.LogActor(c.Request.Context(), uid, "session.revoke_batch", "device", deviceID,
			fmt.Sprintf("revoked=%d failed=%d truncated=%t", revoked, failed, truncated),
			c.ClientIP())
	}
	c.JSON(http.StatusOK, gin.H{
		"device_id": deviceID,
		"revoked":   revoked,
		"failed":    failed,
		"truncated": truncated,
		"sessions":  results,
	})
}
