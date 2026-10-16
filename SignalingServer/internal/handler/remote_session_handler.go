package handler

import (
	"net/http"
	"strings"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

// RemoteSessionHandler serves GET /v1/me/remote-sessions: the owner's view of
// who is in their machines right now, and who has been.
//
// It is deliberately not /v1/me/sessions - that path is the caller's own login
// sessions, a different resource with a different lifetime, and overloading it
// would make "revoke this session" ambiguous.
type RemoteSessionHandler struct {
	sessions *service.RemoteSessionService
}

func NewRemoteSessionHandler(sessions *service.RemoteSessionService) *RemoteSessionHandler {
	return &RemoteSessionHandler{sessions: sessions}
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
