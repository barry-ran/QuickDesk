package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminWebhooksHandler serves /v1/admin/webhooks/*. Per §2.17 M4 webhooks
// only subscribe to system-scope events; user-scope events (device.*,
// favorite.*) are NOT delivered to webhooks. That scoping happens inside
// the event bus subscriber; this handler is just CRUD.
type AdminWebhooksHandler struct {
	webhooks *service.WebhookService
	audit    *service.AuditService
}

func NewAdminWebhooksHandler(w *service.WebhookService, a *service.AuditService) *AdminWebhooksHandler {
	return &AdminWebhooksHandler{webhooks: w, audit: a}
}

func (h *AdminWebhooksHandler) List(c *gin.Context) {
	items, err := h.webhooks.GetAll(c.Request.Context())
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

type webhookCreateReq struct {
	Name    string   `json:"name" binding:"required"`
	URL     string   `json:"url" binding:"required"`
	Secret  string   `json:"secret"`
	Events  []string `json:"events" binding:"required"`
	Enabled bool     `json:"enabled"`
}

func (h *AdminWebhooksHandler) Create(c *gin.Context) {
	var req webhookCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	eventsJSON, _ := json.Marshal(req.Events)
	w := &models.Webhook{
		Name:    req.Name,
		URL:     req.URL,
		Secret:  req.Secret,
		Events:  string(eventsJSON),
		Enabled: req.Enabled,
	}
	if err := h.webhooks.Create(c.Request.Context(), w); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "webhook.create", "webhook", strconv.FormatUint(uint64(w.ID), 10), "", c.ClientIP())
	c.JSON(http.StatusCreated, w)
}

func (h *AdminWebhooksHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid id")
		return
	}
	w, err := h.webhooks.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Webhook not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, w)
}

type webhookPatchReq struct {
	Name    *string  `json:"name"`
	URL     *string  `json:"url"`
	Secret  *string  `json:"secret"`
	Events  []string `json:"events"`
	Enabled *bool    `json:"enabled"`
}

func (h *AdminWebhooksHandler) Patch(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid id")
		return
	}
	w, err := h.webhooks.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		ProblemNotFound(c, ProblemCodeNotFound, "Webhook not found")
		return
	}
	var req webhookPatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if req.Name != nil {
		w.Name = *req.Name
	}
	if req.URL != nil {
		w.URL = *req.URL
	}
	if req.Secret != nil {
		w.Secret = *req.Secret
	}
	if req.Events != nil {
		eventsJSON, _ := json.Marshal(req.Events)
		w.Events = string(eventsJSON)
	}
	if req.Enabled != nil {
		w.Enabled = *req.Enabled
	}
	if err := h.webhooks.Update(c.Request.Context(), w); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "webhook.update", "webhook", c.Param("id"), "", c.ClientIP())
	c.JSON(http.StatusOK, w)
}

func (h *AdminWebhooksHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid id")
		return
	}
	if err := h.webhooks.Delete(c.Request.Context(), uint(id)); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "webhook.delete", "webhook", c.Param("id"), "", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Test implements POST /v1/admin/webhooks/:id/test — sends a synthetic
// event to let operators verify the endpoint without waiting for a real
// trigger.
func (h *AdminWebhooksHandler) Test(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid id")
		return
	}
	w, err := h.webhooks.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		ProblemNotFound(c, ProblemCodeNotFound, "Webhook not found")
		return
	}
	h.webhooks.Dispatch("webhook.test", gin.H{
		"webhook_id": w.ID,
		"message":    "This is a synthetic test event",
	})
	c.JSON(http.StatusOK, gin.H{"status": "dispatched"})
}
