package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ScheduledTaskHandler serves the owner-scheduled control-plane actions:
//
//	GET    /v1/me/scheduled-tasks
//	POST   /v1/me/scheduled-tasks
//	DELETE /v1/me/scheduled-tasks/:id
//
// It is a schedule-management surface, not an execution one: the worker runs
// due tasks in the background, and `last_status`/`last_error` are how a caller
// finds out what happened.
type ScheduledTaskHandler struct {
	tasks *service.ScheduledTaskService
}

func NewScheduledTaskHandler(tasks *service.ScheduledTaskService) *ScheduledTaskHandler {
	return &ScheduledTaskHandler{tasks: tasks}
}

type scheduledTaskReq struct {
	Kind string `json:"kind" binding:"required"`
	// RunAt is RFC3339 and must be in the future. It is required unless
	// Schedule is given, which derives every occurrence itself.
	RunAt string `json:"run_at"`
	// RepeatSeconds is 0 for a one-shot task, or 60..31536000 to repeat.
	RepeatSeconds int `json:"repeat_seconds"`
	// MaxAttempts is how many times one occurrence may be attempted (1..10).
	// 1, the default, means "no retry".
	MaxAttempts int `json:"max_attempts"`
	// Schedule is an optional calendar recurrence: `daily@HH:MM`,
	// `weekdays@HH:MM`, `weekly@<dow>@HH:MM`, `monthly@<dom>@HH:MM` or
	// `every@<N>@days@HH:MM`.
	Schedule string `json:"schedule"`
	// Timezone is the IANA zone Schedule is evaluated in; empty means UTC.
	Timezone string `json:"timezone"`
	// SkipHolidays moves a calendar occurrence off a statutory day off. Only
	// meaningful together with Schedule.
	SkipHolidays bool `json:"skip_holidays"`
}

// scheduledTaskView is the wire shape. The outcome fields are exposed so an
// owner can see that a task fired and what it did without reading server logs.
type scheduledTaskView struct {
	ID            uint       `json:"id"`
	Kind          string     `json:"kind"`
	NextRunAt     time.Time  `json:"next_run_at"`
	RepeatSeconds int        `json:"repeat_seconds"`
	Enabled       bool       `json:"enabled"`
	Schedule      string     `json:"schedule,omitempty"`
	Timezone      string     `json:"timezone,omitempty"`
	SkipHolidays  bool       `json:"skip_holidays"`
	MaxAttempts   int        `json:"max_attempts"`
	Attempts      int        `json:"attempts"`
	LastRunAt     *time.Time `json:"last_run_at,omitempty"`
	LastStatus    string     `json:"last_status,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func scheduledTaskViewOf(row models.ScheduledTask) scheduledTaskView {
	return scheduledTaskView{
		ID:            row.ID,
		Kind:          row.Kind,
		NextRunAt:     row.NextRunAt,
		RepeatSeconds: row.RepeatSeconds,
		Enabled:       row.Enabled,
		Schedule:      row.Schedule,
		Timezone:      row.Timezone,
		SkipHolidays:  row.SkipHolidays,
		MaxAttempts:   row.MaxAttempts,
		Attempts:      row.Attempts,
		LastRunAt:     row.LastRunAt,
		LastStatus:    row.LastStatus,
		LastError:     row.LastError,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

func (h *ScheduledTaskHandler) List(c *gin.Context) {
	if h.tasks == nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "scheduled tasks are not configured")
		return
	}
	rows, err := h.tasks.List(c.Request.Context(), middleware.MustUserID(c))
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	items := make([]scheduledTaskView, 0, len(rows))
	for _, row := range rows {
		items = append(items, scheduledTaskViewOf(row))
	}
	c.JSON(http.StatusOK, gin.H{
		"items":               items,
		"supported_kinds":     h.tasks.SupportedKinds(),
		"supported_schedules": service.SupportedCalendarSchedules(),
	})
}

func (h *ScheduledTaskHandler) Create(c *gin.Context) {
	if h.tasks == nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "scheduled tasks are not configured")
		return
	}
	var req scheduledTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	// run_at and schedule are two ways to say when; the service rejects being
	// given both, and here a syntactically broken run_at is caught before it
	// reaches it.
	var runAt time.Time
	if strings.TrimSpace(req.RunAt) != "" {
		parsed, err := time.Parse(time.RFC3339, req.RunAt)
		if err != nil {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "run_at must be RFC3339, e.g. 2026-10-17T03:00:00Z")
			return
		}
		runAt = parsed
	} else if strings.TrimSpace(req.Schedule) == "" {
		ProblemBadRequest(c, ProblemCodeInvalidRequest,
			"run_at is required unless schedule is given (one of: "+strings.Join(service.SupportedCalendarSchedules(), ", ")+")")
		return
	}
	row, err := h.tasks.Create(c.Request.Context(), middleware.MustUserID(c), service.ScheduledTaskSpec{
		Kind:          req.Kind,
		RunAt:         runAt,
		RepeatSeconds: req.RepeatSeconds,
		MaxAttempts:   req.MaxAttempts,
		Schedule:      req.Schedule,
		Timezone:      req.Timezone,
		SkipHolidays:  req.SkipHolidays,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidScheduledTaskKind) ||
			errors.Is(err, service.ErrInvalidScheduledTaskSchedule) ||
			errors.Is(err, service.ErrInvalidScheduledTaskMaxAttempts) ||
			errors.Is(err, service.ErrInvalidScheduledTaskCalendar) {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, scheduledTaskViewOf(*row))
}

func (h *ScheduledTaskHandler) Delete(c *gin.Context) {
	if h.tasks == nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "scheduled tasks are not configured")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid scheduled task id")
		return
	}
	uid := middleware.MustUserID(c)
	// Resolve first so deleting someone else's id is a 404 rather than a
	// silent success, and deleting an id that never existed is honest too.
	if _, err := h.tasks.Get(c.Request.Context(), uid, uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Scheduled task not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	if err := h.tasks.Delete(c.Request.Context(), uid, uint(id)); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
