package handler

import (
	"net/http"

	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminAuditHandler serves GET /v1/admin/audit-logs.
type AdminAuditHandler struct {
	audit *service.AuditService
}

func NewAdminAuditHandler(a *service.AuditService) *AdminAuditHandler {
	return &AdminAuditHandler{audit: a}
}

func (h *AdminAuditHandler) List(c *gin.Context) {
	p := ParseCursor(c)
	cur, err := DecodeCursor(p.Cursor)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	logs, total, err := h.audit.List(c.Request.Context(), service.AuditListParams{
		AfterID:       cur.OffsetID,
		Limit:         p.Limit + 1,
		Action:        c.Query("action"),
		AdminUsername: c.Query("admin"),
		DateFrom:      c.Query("date_from"),
		DateTo:        c.Query("date_to"),
	})
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	next := ""
	if len(logs) > p.Limit {
		last := logs[p.Limit-1]
		next = EncodeCursor(CursorPayload{OffsetID: last.ID})
		logs = logs[:p.Limit]
	}
	c.JSON(http.StatusOK, NewCursorPage(logs, next, total))
}
