package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// Scheduled-task outcome values written to ScheduledTask.LastStatus.
const (
	ScheduledTaskStatusOK     = "ok"
	ScheduledTaskStatusFailed = "failed"
)

// Schedule bounds. A repeat interval below a minute would let one task occupy
// the worker, and a year is already longer than any maintenance cadence worth
// configuring.
const (
	MinScheduledTaskRepeatSeconds = 60
	MaxScheduledTaskRepeatSeconds = 365 * 24 * 60 * 60
	// DefaultScheduledTaskMaxAttempts is "no retry": one attempt per occurrence,
	// which is how the scheduler behaved before retries existed.
	DefaultScheduledTaskMaxAttempts = 1
	MinScheduledTaskMaxAttempts     = 1
	// MaxScheduledTaskMaxAttempts bounds the retry budget. A task that keeps
	// failing is a bug or an outage; hammering it forever helps neither.
	MaxScheduledTaskMaxAttempts = 10
	// scheduledTaskRetryBackoffCap bounds the per-retry delay.
	scheduledTaskRetryBackoffCap = 10 * time.Minute
	// scheduledTaskBatch bounds one worker pass.
	scheduledTaskBatch = 100
)

var (
	// ErrInvalidScheduledTaskKind means the kind has no registered action, so
	// scheduling it would create a row nothing can ever run.
	ErrInvalidScheduledTaskKind = errors.New("unknown scheduled task kind")
	// ErrInvalidScheduledTaskSchedule covers a run_at that is absent, in the
	// past, or a repeat interval outside the accepted range.
	ErrInvalidScheduledTaskSchedule = errors.New("scheduled task needs a future run_at and repeat_seconds of 0 or 60..31536000")
	// ErrInvalidScheduledTaskMaxAttempts covers a retry budget outside 1..10.
	ErrInvalidScheduledTaskMaxAttempts = errors.New("scheduled task max_attempts must be between 1 and 10")
)

// ScheduledTaskSpec is one create request: either an absolute schedule
// (`RunAt` + `RepeatSeconds`) or a calendar `Schedule` in a `Timezone`. The two
// shapes are mutually exclusive, which Create enforces rather than silently
// picking one.
type ScheduledTaskSpec struct {
	Kind          string
	RunAt         time.Time
	RepeatSeconds int
	MaxAttempts   int
	// Schedule is a calendar recurrence: `daily@HH:MM`, `weekdays@HH:MM`,
	// `weekly@<dow>@HH:MM`, `monthly@<dom>@HH:MM` or `every@<N>@days@HH:MM`.
	Schedule string
	// Timezone is the IANA zone Schedule is evaluated in; empty means UTC.
	Timezone string
	// SkipHolidays moves a calendar occurrence off a statutory day off, using
	// the same calendar the device access policy resolves holidays with.
	SkipHolidays bool
}

// ScheduledTaskAction is the server-side work one kind performs. Actions must
// be safe to run unattended and without a request context: the worker calls
// them with the task's owner as the only identity.
type ScheduledTaskAction interface {
	Run(ctx context.Context, task models.ScheduledTask) error
}

// ScheduledTaskNotifier receives the events the scheduler emits. It is an
// interface rather than *EventBus so a test can capture what was published
// without standing up a Redis-backed bus.
type ScheduledTaskNotifier interface {
	Publish(ctx context.Context, evt Event)
}

// ScheduledTaskService owns owner-scheduled control-plane actions: the row,
// the due-claim, and the execution. It deliberately starts with kinds whose
// whole effect lives in the control plane, so a task cannot silently depend on
// a client that is not there.
type ScheduledTaskService struct {
	db       *gorm.DB
	actions  map[string]ScheduledTaskAction
	audit    *AuditService
	notifier ScheduledTaskNotifier
}

func NewScheduledTaskService(db *gorm.DB) *ScheduledTaskService {
	return &ScheduledTaskService{db: db, actions: map[string]ScheduledTaskAction{}}
}

// WithAudit records one audit row per run. Optional: a deployment without an
// audit service still gets the task's own last_status/last_error.
func (s *ScheduledTaskService) WithAudit(a *AuditService) *ScheduledTaskService {
	s.audit = a
	return s
}

// WithNotifier emits a `scheduled_task.failed` event when a task spends its
// retry budget, so the owner's notification channels can tell them a scheduled
// action never succeeded. Optional: without it the failure is still on the row
// and in the audit trail.
func (s *ScheduledTaskService) WithNotifier(n ScheduledTaskNotifier) *ScheduledTaskService {
	s.notifier = n
	return s
}

// RegisterAction wires a kind to its work. Called at startup so the API's
// supported-kind list and the worker's dispatch table cannot disagree.
func (s *ScheduledTaskService) RegisterAction(kind string, action ScheduledTaskAction) *ScheduledTaskService {
	s.actions[strings.TrimSpace(kind)] = action
	return s
}

// SupportedKinds lists the registered kinds in a stable order.
func (s *ScheduledTaskService) SupportedKinds() []string {
	out := make([]string, 0, len(s.actions))
	for kind := range s.actions {
		out = append(out, kind)
	}
	sort.Strings(out)
	return out
}

// Create schedules one action for the user. The kind has to be registered and
// the schedule has to describe a future occurrence: a task that is already due
// is almost always a timezone or clock mistake, and the worker would otherwise
// run it before the caller saw the response.
func (s *ScheduledTaskService) Create(
	ctx context.Context,
	userID uint,
	spec ScheduledTaskSpec,
) (*models.ScheduledTask, error) {
	kind := strings.TrimSpace(spec.Kind)
	if _, ok := s.actions[kind]; !ok {
		return nil, fmt.Errorf("%w: %q (supported: %s)", ErrInvalidScheduledTaskKind, kind,
			strings.Join(s.SupportedKinds(), ", "))
	}
	maxAttempts := spec.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = DefaultScheduledTaskMaxAttempts
	}
	if maxAttempts < MinScheduledTaskMaxAttempts || maxAttempts > MaxScheduledTaskMaxAttempts {
		return nil, fmt.Errorf("%w: max_attempts is %d", ErrInvalidScheduledTaskMaxAttempts, maxAttempts)
	}
	now := time.Now().UTC()
	schedule := strings.TrimSpace(spec.Schedule)
	timezone := strings.TrimSpace(spec.Timezone)
	row := &models.ScheduledTask{
		UserID:      userID,
		Kind:        kind,
		Enabled:     true,
		MaxAttempts: maxAttempts,
	}
	if spec.SkipHolidays && schedule == "" {
		// An interval task has no calendar day to skip, and accepting the flag
		// would create a task whose stored row claims a rule that never applies.
		return nil, fmt.Errorf("%w: skip_holidays is only meaningful with schedule", ErrInvalidScheduledTaskCalendar)
	}
	if schedule != "" {
		// A calendar task derives every occurrence, the first one included, so a
		// caller-supplied run_at would be a second, contradictory schedule.
		if !spec.RunAt.IsZero() {
			return nil, fmt.Errorf("%w: run_at and schedule are mutually exclusive", ErrInvalidScheduledTaskCalendar)
		}
		if spec.RepeatSeconds != 0 {
			return nil, fmt.Errorf("%w: repeat_seconds must be 0 with schedule, because the calendar recurrence is the period", ErrInvalidScheduledTaskCalendar)
		}
		parsed, err := parseCalendarSchedule(schedule)
		if err != nil {
			return nil, err
		}
		loc, err := scheduledTaskLocation(timezone)
		if err != nil {
			return nil, err
		}
		row.Schedule = schedule
		row.Timezone = timezone
		row.SkipHolidays = spec.SkipHolidays
		row.CreatedAt = now
		next := parsed.next(now, loc, row.CreatedAt)
		if row.SkipHolidays {
			next = advancePastHolidays(next, loc, currentHolidayCalendar())
		}
		row.NextRunAt = next
	} else {
		if timezone != "" {
			return nil, fmt.Errorf("%w: timezone is only meaningful with schedule", ErrInvalidScheduledTaskCalendar)
		}
		if spec.RunAt.IsZero() || !spec.RunAt.After(now) {
			return nil, fmt.Errorf("%w: run_at must be in the future", ErrInvalidScheduledTaskSchedule)
		}
		if spec.RepeatSeconds != 0 &&
			(spec.RepeatSeconds < MinScheduledTaskRepeatSeconds || spec.RepeatSeconds > MaxScheduledTaskRepeatSeconds) {
			return nil, fmt.Errorf("%w: repeat_seconds is %d", ErrInvalidScheduledTaskSchedule, spec.RepeatSeconds)
		}
		row.NextRunAt = spec.RunAt.UTC()
		row.RepeatSeconds = spec.RepeatSeconds
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// List returns the user's tasks, soonest first, so the UI can show what is
// about to happen.
func (s *ScheduledTaskService) List(ctx context.Context, userID uint) ([]models.ScheduledTask, error) {
	var rows []models.ScheduledTask
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("next_run_at, id").
		Find(&rows).Error
	return rows, err
}

// Get is the ownership-scoped read used by delete.
func (s *ScheduledTaskService) Get(ctx context.Context, userID, id uint) (*models.ScheduledTask, error) {
	var row models.ScheduledTask
	if err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// Delete removes one of the user's tasks. Filtering on user_id is what makes
// another user's id a 404 rather than a deletion.
func (s *ScheduledTaskService) Delete(ctx context.Context, userID, id uint) error {
	return s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.ScheduledTask{}).Error
}

// RunDue claims and runs every task that is due at `now`, returning how many it
// ran. Run is exported so maintenance commands and tests can drive the
// scheduler without waiting for the ticker.
func (s *ScheduledTaskService) RunDue(ctx context.Context, now time.Time) (int, error) {
	claimed, err := s.claimDue(ctx, now)
	if err != nil {
		return 0, err
	}
	for _, task := range claimed {
		s.runOne(ctx, task, now)
	}
	return len(claimed), nil
}

// claimDue atomically takes the due tasks and advances their schedule.
//
// The advance happens inside the claiming transaction, before the action runs,
// so a second worker (or a second instance) can never fire the same occurrence
// twice. The trade-off is at-most-once: a crash in the window between the claim
// and the action skips that occurrence. That is the right side to err on for a
// scheduler - a duplicate "sign out everywhere" is surprising, a skipped one is
// visible in last_status - and recurring tasks simply run again next period.
//
// A recurring task whose worker was down for several periods fires once and is
// rescheduled from `now`, rather than replaying every missed occurrence. The
// same rule covers calendar schedules, where "next" is recomputed from `now` in
// the task's own timezone instead of by adding a fixed period.
func (s *ScheduledTaskService) claimDue(ctx context.Context, now time.Time) ([]models.ScheduledTask, error) {
	var claimed []models.ScheduledTask
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw(`
			SELECT * FROM scheduled_tasks
			WHERE enabled = TRUE AND next_run_at <= ?
			ORDER BY next_run_at, id
			LIMIT ?
			FOR UPDATE SKIP LOCKED`, now, scheduledTaskBatch).Scan(&claimed).Error; err != nil {
			return err
		}
		runnable := claimed[:0]
		for i := range claimed {
			updates := map[string]interface{}{"updated_at": now}
			switch {
			case !scheduledTaskRecurs(claimed[i]):
				// A one-shot task has no next occurrence: disable it and keep
				// the row so "it ran" stays visible.
				updates["enabled"] = false
				runnable = append(runnable, claimed[i])
			default:
				next, err := scheduledTaskNextOccurrence(claimed[i], now)
				if err != nil {
					// A recurrence we cannot compute would stay due and be
					// claimed on every tick, so park the row instead of
					// looping on it. The action does not run: nobody can say
					// when this task was supposed to fire.
					log.Printf("[scheduled-task] task %d has an unusable schedule, disabling it: %v", claimed[i].ID, err)
					updates["enabled"] = false
					updates["last_status"] = ScheduledTaskStatusFailed
					updates["last_error"] = truncateScheduledTaskError(err.Error())
					break
				}
				updates["next_run_at"] = next
				runnable = append(runnable, claimed[i])
			}
			if err := tx.Model(&models.ScheduledTask{}).
				Where("id = ?", claimed[i].ID).
				Updates(updates).Error; err != nil {
				return err
			}
		}
		claimed = runnable
		return nil
	})
	return claimed, err
}

// runOne executes one claimed task and records the outcome, re-arming the task
// when the failure is retryable.
//
// The claim already moved the schedule forward (or disabled a one-shot), so a
// retry is expressed by writing a *new* next_run_at from here. Two rules keep
// that honest: the retry never pushes a recurring task past its own next
// period, and the retry budget is bounded per occurrence, so a task that keeps
// failing eventually parks with last_error instead of spinning.
func (s *ScheduledTaskService) runOne(ctx context.Context, task models.ScheduledTask, now time.Time) {
	action, ok := s.actions[task.Kind]
	status := ScheduledTaskStatusOK
	message := ""
	if !ok {
		status = ScheduledTaskStatusFailed
		message = "no action registered for kind " + task.Kind
	} else if err := action.Run(ctx, task); err != nil {
		status = ScheduledTaskStatusFailed
		message = truncateScheduledTaskError(err.Error())
	}

	attempted := task.Attempts + 1
	updates := map[string]interface{}{
		"last_run_at": now,
		"last_status": status,
		"last_error":  message,
		"updated_at":  now,
	}
	giveUp := false
	switch {
	case status == ScheduledTaskStatusOK:
		// Success ends the occurrence: clear the counter so the next period (or
		// an operator re-running it) starts fresh.
		updates["attempts"] = 0
	case attempted >= scheduledTaskMaxAttempts(task):
		// Out of attempts. A one-shot stays disabled and keeps the count so the
		// operator can see it tried; a recurring task keeps the occurrence the
		// claim already advanced and starts the next one with a clean counter.
		giveUp = true
		if scheduledTaskRecurs(task) {
			updates["attempts"] = 0
		} else {
			updates["attempts"] = attempted
		}
	default:
		retryAt := now.Add(scheduledTaskRetryBackoff(attempted))
		retryable := true
		if scheduledTaskRecurs(task) {
			// The retry exists to run sooner, not to drift the schedule, so it
			// is clamped to the occurrence the claim already advanced to.
			periodEnd, err := scheduledTaskNextOccurrence(task, now)
			if err != nil {
				// A recurrence we cannot compute cannot be re-armed safely;
				// park it instead of spinning on it.
				retryable = false
				message = truncateScheduledTaskError(message + " (retry skipped: " + err.Error() + ")")
			} else if retryAt.After(periodEnd) {
				retryAt = periodEnd
			}
		}
		if retryable {
			updates["next_run_at"] = retryAt
			updates["attempts"] = attempted
			updates["enabled"] = true
		} else {
			giveUp = true
			if scheduledTaskRecurs(task) {
				updates["attempts"] = 0
			} else {
				updates["attempts"] = attempted
			}
		}
	}
	// The message may have been extended above by a skipped retry.
	updates["last_error"] = message

	if err := s.db.WithContext(ctx).Model(&models.ScheduledTask{}).
		Where("id = ?", task.ID).
		Updates(updates).Error; err != nil {
		log.Printf("[scheduled-task] record outcome for task %d failed: %v", task.ID, err)
	}
	if s.audit != nil {
		s.audit.LogActor(ctx, task.UserID, "scheduled_task.run", "scheduled_task",
			fmt.Sprintf("%d", task.ID),
			fmt.Sprintf("kind=%s status=%s attempt=%d/%d error=%q",
				task.Kind, status, attempted, scheduledTaskMaxAttempts(task), message),
			"")
	}
	// Only the terminal failure is an event: the retries are visible on the row
	// and in the audit trail, and alerting on each one would bury the signal.
	if giveUp && s.notifier != nil {
		s.notifier.Publish(ctx, Event{
			Type:   EventScheduledTaskFailed,
			UserID: task.UserID,
			Data: map[string]interface{}{
				"task_id":  task.ID,
				"kind":     task.Kind,
				"status":   status,
				"attempts": attempted,
				"error":    message,
			},
		})
	}
}

// truncateScheduledTaskError keeps a message inside the column's 255 bytes, so
// a long error is truncated in the row instead of failing the write.
func truncateScheduledTaskError(message string) string {
	if len(message) > 255 {
		return message[:255]
	}
	return message
}

// scheduledTaskMaxAttempts treats an unset budget as "no retry", so rows
// written before the column existed keep their old behaviour.
func scheduledTaskMaxAttempts(task models.ScheduledTask) int {
	if task.MaxAttempts < MinScheduledTaskMaxAttempts {
		return DefaultScheduledTaskMaxAttempts
	}
	return task.MaxAttempts
}

// scheduledTaskRetryBackoff grows 30s, 1m, 2m, 4m, 8m and then holds at the cap.
// The first retry is deliberately quick: the common failure is a blip, and a
// one-shot task has no next period to fall back on.
func scheduledTaskRetryBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	backoff := 30 * time.Second * time.Duration(1<<uint(min(attempt-1, 5)))
	if backoff > scheduledTaskRetryBackoffCap {
		return scheduledTaskRetryBackoffCap
	}
	return backoff
}

// Run drives RunDue on a ticker until ctx is cancelled. interval <= 0 disables
// the worker entirely, for a deployment that never schedules anything; a tick
// with nothing due is one indexed query.
func (s *ScheduledTaskService) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.RunDue(ctx, time.Now().UTC()); err != nil {
				log.Printf("[scheduled-task] run due tasks failed: %v", err)
			}
		}
	}
}

// SignOutAllSessionsAction revokes every active session family of the task's
// owner. It is the first registered kind because the whole effect lives in the
// control plane: no controlled machine and no controller client has to
// cooperate, so it can be scheduled and verified today.
type SignOutAllSessionsAction struct {
	tokens *TokenService
	bus    *EventBus
}

func NewSignOutAllSessionsAction(tokens *TokenService, bus *EventBus) *SignOutAllSessionsAction {
	return &SignOutAllSessionsAction{tokens: tokens, bus: bus}
}

func (a *SignOutAllSessionsAction) Run(ctx context.Context, task models.ScheduledTask) error {
	if a.tokens == nil {
		return errors.New("token service is not configured")
	}
	a.tokens.RevokeAllForSubject(ctx, ScopeUser, task.UserID)
	if a.bus != nil {
		a.bus.Publish(ctx, Event{
			Type:   EventSessionRevoked,
			UserID: task.UserID,
			Data: map[string]interface{}{
				"reason":  "scheduled_task",
				"task_id": task.ID,
			},
		})
	}
	return nil
}
