package models

import "time"

// ScheduledTaskKind values. The kind decides which server-side action the
// worker runs when the task comes due; the set is deliberately small and
// explicit, because a task kind is a thing an operator can schedule against
// their own account and every one of them has to be safe to run unattended.
const (
	// ScheduledTaskSignOutAllSessions revokes every active session family of
	// the task's owner: "log out everywhere at 22:00, and every night after".
	// It is the first kind because the whole action lives in the control plane
	// - no controlled machine or controller client has to cooperate - so it can
	// be built and verified without waiting on client work.
	ScheduledTaskSignOutAllSessions = "sign_out_all_sessions"
)

// ScheduledTask is one owner-scheduled control-plane action.
//
// The row carries both the schedule and the outcome, so an operator can see
// when a task last ran and what happened without a second table. `next_run_at`
// is the only field the worker sorts on; a one-shot task (RepeatSeconds == 0)
// is disabled after it runs rather than deleted, so the history of "it ran" is
// preserved.
type ScheduledTask struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	UserID uint   `gorm:"not null;index" json:"user_id"`
	Kind   string `gorm:"size:64;not null" json:"kind"`

	// NextRunAt is when the worker may run this task.
	NextRunAt time.Time `gorm:"not null;index" json:"next_run_at"`
	// RepeatSeconds is the interval between runs; 0 means "run once".
	RepeatSeconds int  `gorm:"not null;default:0" json:"repeat_seconds"`
	Enabled       bool `gorm:"not null;default:true" json:"enabled"`
	// Schedule is an optional calendar recurrence: `daily@HH:MM`,
	// `weekdays@HH:MM`, `weekly@<dow>@HH:MM`, `monthly@<dom>@HH:MM` or
	// `every@<N>@days@HH:MM`. When it is set it *is* the recurrence, so
	// RepeatSeconds must be 0 and NextRunAt is derived from it after every run
	// rather than by adding a fixed period.
	Schedule string `gorm:"size:64;not null;default:''" json:"schedule"`
	// SkipHolidays moves a calendar occurrence off a statutory day off (the
	// same calendar the device access policy uses), keeping the wall-clock time.
	// It is rejected without a Schedule because an interval task has no calendar
	// day to skip.
	SkipHolidays bool `gorm:"not null;default:false" json:"skip_holidays"`
	// Timezone is the IANA zone the calendar schedule is evaluated in; empty
	// means UTC. It is ignored for an interval task.
	Timezone string `gorm:"size:64;not null;default:''" json:"timezone"`
	// MaxAttempts is how many times one occurrence may be attempted: 1 means
	// "no retry" (the historical behaviour), 2..10 re-arms a failed task after a
	// short backoff instead of waiting a whole period or giving up outright.
	MaxAttempts int `gorm:"not null;default:1" json:"max_attempts"`
	// Attempts counts attempts made for the current occurrence. It is reset to
	// 0 on success and, for a recurring task, when retries are exhausted so the
	// next period starts fresh.
	Attempts int `gorm:"not null;default:0" json:"attempts"`

	LastRunAt  *time.Time `json:"last_run_at"`
	LastStatus string     `gorm:"size:16;not null;default:''" json:"last_status"`
	LastError  string     `gorm:"size:255;not null;default:''" json:"last_error"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName overrides the default table name.
func (ScheduledTask) TableName() string {
	return "scheduled_tasks"
}
