package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminRemoteSessionHandler serves the cross-user session console under
// /v1/admin/remote-sessions: which controller is in which machine across every
// account on the fleet, and what a session produced.
//
// It is the admin counterpart of /v1/me/remote-sessions and deliberately reuses
// the same joined shape (see detailEnvelope) so an operator's console and a
// user's device page render the same data. There is no owner filter here - that
// is the point of the endpoint - so authorization is the admin token plus the
// same IP allow-list every /v1/admin route already carries. It reads only; a
// session is still created and ended by the signaling socket that owns it.
type AdminRemoteSessionHandler struct {
	sessions   *service.RemoteSessionService
	recordings *service.SessionRecordingService
	access     *service.DeviceAccessService
	revoker    SessionRevoker
	audit      *service.AuditService
}

func NewAdminRemoteSessionHandler(
	sessions *service.RemoteSessionService,
	recordings *service.SessionRecordingService,
	access *service.DeviceAccessService,
	revoker SessionRevoker,
	audit *service.AuditService,
) *AdminRemoteSessionHandler {
	return &AdminRemoteSessionHandler{
		sessions: sessions, recordings: recordings, access: access,
		revoker: revoker, audit: audit,
	}
}

// List handles GET /v1/admin/remote-sessions.
//
// Filters: `status` (active/ended/all), `device_id`, `user_id`, and the usual
// cursor/limit. Each item arrives already joined with its recordings and access
// log, the same shape /v1/me/devices/:device_id/sessions returns, so the console
// can paint one page without a second request per row.
func (h *AdminRemoteSessionHandler) List(c *gin.Context) {
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
	var userID uint
	if raw := strings.TrimSpace(c.Query("user_id")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid user_id")
			return
		}
		userID = uint(parsed)
	}
	rows, err := h.sessions.List(c.Request.Context(), service.RemoteSessionFilter{
		UserID:   userID,
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
	details, err := h.attachRelated(c, rows)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, NewCursorPage(details, next, -1))
}

// Get handles GET /v1/admin/remote-sessions/:session_id.
//
// An unknown session id is a 404 problem. Unlike the owner-scoped read there is
// no "somebody else's session" case to hide: the caller is authorised to see
// every session, so the only reason a lookup fails is that the id does not
// exist.
func (h *AdminRemoteSessionHandler) Get(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Param("session_id"))
	if sessionID == "" {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "session_id is required")
		return
	}
	row, err := h.sessions.Get(c.Request.Context(), sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Session not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	recordings := []models.SessionRecording{}
	if h.recordings != nil {
		recordings, err = h.recordings.ListForSessionIDs(c.Request.Context(), []string{row.SessionID}, 100)
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

// Revoke handles POST /v1/admin/remote-sessions/:session_id/revoke: an
// administrator ends one live control session on any account's machine.
//
// Unlike the owner read there is no ownership to hide: the caller may act on
// every session, so an unknown id is simply a 404 and a session that is over or
// not connected here is a 409. The audit row is written only after the socket
// actually closed, so "revoked" in the trail always means it happened.
func (h *AdminRemoteSessionHandler) Revoke(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Param("session_id"))
	if sessionID == "" {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "session_id is required")
		return
	}
	row, err := h.sessions.Get(c.Request.Context(), sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Session not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	if !revokeSessionSocket(c, h.revoker, row) {
		return
	}
	if h.audit != nil {
		h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "session.revoke", "remote_session", row.SessionID,
			"device="+row.DeviceID+" user="+strconv.FormatUint(uint64(row.UserID), 10)+" client="+row.ClientID,
			c.ClientIP())
	}
	c.JSON(http.StatusOK, gin.H{
		"session_id": row.SessionID,
		"device_id":  row.DeviceID,
		"user_id":    row.UserID,
		"revoked":    true,
	})
}

// attachRelated is the admin twin of RemoteSessionHandler.attachRelated: it
// joins a page of sessions with their recordings and access-log rows in two
// batched queries. The only difference is the missing owner/device filter,
// because an admin page mixes both.
func (h *AdminRemoteSessionHandler) attachRelated(
	c *gin.Context,
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
		recordings, err = h.recordings.ListForSessionIDs(c.Request.Context(), ids, maxSessionsPerDetailPage)
		if err != nil {
			return nil, err
		}
	}
	accessLog := []models.DeviceAccessLog{}
	if h.access != nil && len(ids) > 0 {
		var err error
		accessLog, err = h.access.ListForSessionIDs(c.Request.Context(), ids, maxSessionsPerDetailPage)
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
