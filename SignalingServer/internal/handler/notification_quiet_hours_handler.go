package handler

import (
	"errors"
	"log"
	"net/http"
	"time"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

// NotificationQuietHoursHandler serves the per-user quiet-hours setting:
//
//	GET /v1/me/notification-quiet-hours
//	PUT /v1/me/notification-quiet-hours
//
// The setting is what the delivery worker asks before it posts anything, so a
// maintenance window or a snooze stops the alerts at the source instead of
// hoping every rule remembers to check the clock.
type NotificationQuietHoursHandler struct {
	quiet      *service.NotificationQuietHoursService
	deliveries *service.NotificationDeliveryService
}

func NewNotificationQuietHoursHandler(
	quiet *service.NotificationQuietHoursService,
	deliveries *service.NotificationDeliveryService,
) *NotificationQuietHoursHandler {
	return &NotificationQuietHoursHandler{quiet: quiet, deliveries: deliveries}
}

// notificationQuietHoursReq is a full replace, like the offline-alert rule:
// omitting `snoozed_until` clears the snooze, omitting `quiet_hours` clears
// the recurring windows.
type notificationQuietHoursReq struct {
	SnoozedUntil *time.Time             `json:"snoozed_until"`
	QuietHours   []service.AccessWindow `json:"quiet_hours"`
	Timezone     string                 `json:"timezone"`
}

func (h *NotificationQuietHoursHandler) Get(c *gin.Context) {
	if h.quiet == nil {
		ProblemInternal(c, "quiet hours are not configured")
		return
	}
	row, err := h.quiet.Get(c.Request.Context(), middleware.MustUserID(c))
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, service.QuietHoursView(row, time.Now()))
}

func (h *NotificationQuietHoursHandler) Put(c *gin.Context) {
	if h.quiet == nil {
		ProblemInternal(c, "quiet hours are not configured")
		return
	}
	var req notificationQuietHoursReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid JSON body")
		return
	}
	row, err := h.quiet.Put(c.Request.Context(), middleware.MustUserID(c), service.NotificationQuietHoursInput{
		SnoozedUntil: req.SnoozedUntil,
		Windows:      req.QuietHours,
		Timezone:     req.Timezone,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidQuietHours) {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	now := time.Now()
	view := service.QuietHoursView(row, now)
	// Alerts already parked by the previous setting follow the new one: a
	// cleared snooze releases them straight away (rather than leaving them to
	// sleep until the deadline that no longer exists), and a shortened quiet
	// period pulls them forward. Only rows this setting parked are touched.
	if h.deliveries != nil {
		at := now.UTC()
		if view.ActiveUntil != nil {
			at = view.ActiveUntil.UTC()
		}
		if _, err := h.deliveries.ReclockHeld(c.Request.Context(), middleware.MustUserID(c), at, view.Active); err != nil {
			// The setting is saved; a re-clock failure only means the held rows
			// keep their old schedule until the next write.
			log.Printf("[notification] reclocking held deliveries failed: %v", err)
		}
	}
	c.JSON(http.StatusOK, view)
}
