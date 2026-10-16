package service

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// countingAction records how many times it ran, so a test can tell "the claim
// handed this task to exactly one worker" from "it ran twice".
type countingAction struct {
	mu    sync.Mutex
	runs  int
	tasks []uint
	err   error
}

func (a *countingAction) Run(_ context.Context, task models.ScheduledTask) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.runs++
	a.tasks = append(a.tasks, task.ID)
	return a.err
}

func (a *countingAction) count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.runs
}

// newScheduledTaskTestDB gives the scheduler tests a real Postgres, because the
// claim is `FOR UPDATE SKIP LOCKED`.
func newScheduledTaskTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the scheduler tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.ScheduledTask{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("TRUNCATE scheduled_tasks").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

func seedScheduledTask(t *testing.T, db *gorm.DB, row models.ScheduledTask) models.ScheduledTask {
	t.Helper()
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed task: %v", err)
	}
	return row
}

func loadScheduledTask(t *testing.T, db *gorm.DB, id uint) models.ScheduledTask {
	t.Helper()
	var row models.ScheduledTask
	if err := db.First(&row, id).Error; err != nil {
		t.Fatalf("load task: %v", err)
	}
	return row
}

func TestScheduledTaskCreateValidation(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	svc := NewScheduledTaskService(db).RegisterAction("probe", &countingAction{})
	ctx := context.Background()
	future := time.Now().Add(time.Hour)

	if _, err := svc.Create(ctx, 1, ScheduledTaskSpec{Kind: "nope", RunAt: future}); !errors.Is(err, ErrInvalidScheduledTaskKind) {
		t.Fatalf("unknown kind err = %v, want ErrInvalidScheduledTaskKind", err)
	}
	if _, err := svc.Create(ctx, 1, ScheduledTaskSpec{Kind: "probe", RunAt: time.Now().Add(-time.Minute)}); !errors.Is(err, ErrInvalidScheduledTaskSchedule) {
		t.Fatalf("past run_at err = %v, want ErrInvalidScheduledTaskSchedule", err)
	}
	if _, err := svc.Create(ctx, 1, ScheduledTaskSpec{Kind: "probe", RunAt: future, RepeatSeconds: 30}); !errors.Is(err, ErrInvalidScheduledTaskSchedule) {
		t.Fatalf("repeat below the minimum err = %v, want ErrInvalidScheduledTaskSchedule", err)
	}
	if _, err := svc.Create(ctx, 1, ScheduledTaskSpec{Kind: "probe", RunAt: future, MaxAttempts: 11}); !errors.Is(err, ErrInvalidScheduledTaskMaxAttempts) {
		t.Fatalf("max_attempts above the cap err = %v, want ErrInvalidScheduledTaskMaxAttempts", err)
	}
	row, err := svc.Create(ctx, 1, ScheduledTaskSpec{Kind: "probe", RunAt: future, RepeatSeconds: 60})
	if err != nil {
		t.Fatalf("valid create: %v", err)
	}
	if !row.Enabled || row.RepeatSeconds != 60 || row.MaxAttempts != DefaultScheduledTaskMaxAttempts {
		t.Fatalf("created row = %+v, want an enabled 60s repeat", row)
	}
}

// A calendar task derives its own first occurrence, so the caller never has to
// compute "the next 05:30 in Shanghai" by hand - the reason the field exists.
func TestScheduledTaskCalendarCreateDerivesTheFirstRun(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	svc := NewScheduledTaskService(db).RegisterAction("probe", &countingAction{})
	ctx := context.Background()

	row, err := svc.Create(ctx, 4, ScheduledTaskSpec{
		Kind:     "probe",
		Schedule: "daily@05:30",
		Timezone: "Asia/Shanghai",
	})
	if err != nil {
		t.Fatalf("create calendar task: %v", err)
	}
	if row.Schedule != "daily@05:30" || row.Timezone != "Asia/Shanghai" {
		t.Fatalf("row = %+v, want the schedule and zone persisted", row)
	}
	if row.RepeatSeconds != 0 {
		t.Fatalf("repeat_seconds = %d, want 0: a calendar task's period is the calendar", row.RepeatSeconds)
	}
	if !row.NextRunAt.After(time.Now().UTC()) {
		t.Fatalf("next_run_at = %v, want a future occurrence", row.NextRunAt)
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	local := row.NextRunAt.In(loc)
	if local.Hour() != 5 || local.Minute() != 30 {
		t.Fatalf("first occurrence = %v (%02d:%02d local), want 05:30 in Shanghai",
			local, local.Hour(), local.Minute())
	}
	// And the row survives a round trip through the database with both new
	// columns intact, not just in the struct Create returned.
	stored := loadScheduledTask(t, db, row.ID)
	if stored.Schedule != "daily@05:30" || stored.Timezone != "Asia/Shanghai" {
		t.Fatalf("stored row = %+v, want the calendar fields persisted", stored)
	}
}

// A calendar task advances to the next occurrence, not to a fixed period: the
// whole point is that tomorrow's run is tomorrow's calendar time.
func TestScheduledTaskCalendarAdvancesToTheNextOccurrence(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	action := &countingAction{}
	svc := NewScheduledTaskService(db).RegisterAction("probe", action)
	ctx := context.Background()

	now := time.Now().UTC()
	task := seedScheduledTask(t, db, models.ScheduledTask{
		UserID:    9,
		Kind:      "probe",
		Schedule:  "daily@00:00",
		Timezone:  "UTC",
		NextRunAt: now.Add(-time.Minute),
		Enabled:   true,
	})
	if ran, err := svc.RunDue(ctx, now); err != nil || ran != 1 {
		t.Fatalf("run due = %d, %v; want 1, nil", ran, err)
	}
	row := loadScheduledTask(t, db, task.ID)
	if !row.Enabled || row.LastStatus != ScheduledTaskStatusOK {
		t.Fatalf("row = %+v, want an enabled task that ran ok", row)
	}
	// The next midnight, computed independently here so the test does not just
	// restate the implementation.
	want := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if !want.After(now) {
		want = want.AddDate(0, 0, 1)
	}
	if !row.NextRunAt.Equal(want) {
		t.Fatalf("next_run_at = %v, want the next midnight %v", row.NextRunAt, want)
	}
	if action.count() != 1 {
		t.Fatalf("action ran %d times, want 1", action.count())
	}
}

// A failing calendar task is retried inside the occurrence it belongs to: the
// retry runs minutes from now, not at tomorrow's calendar time, and it stays
// enabled so the retry can actually happen.
func TestScheduledTaskCalendarRetryRunsBeforeTheNextOccurrence(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	action := &flakyAction{failures: 99}
	svc := NewScheduledTaskService(db).RegisterAction("probe", action)
	ctx := context.Background()

	now := time.Now().UTC()
	task := seedScheduledTask(t, db, models.ScheduledTask{
		UserID:      3,
		Kind:        "probe",
		Schedule:    "daily@00:00",
		Timezone:    "UTC",
		NextRunAt:   now.Add(-time.Second),
		Enabled:     true,
		MaxAttempts: 3,
	})
	if n, err := svc.RunDue(ctx, now); err != nil || n != 1 {
		t.Fatalf("first pass = %d,%v; want 1,nil", n, err)
	}
	row := loadScheduledTask(t, db, task.ID)
	if !row.Enabled || row.Attempts != 1 || row.LastStatus != ScheduledTaskStatusFailed {
		t.Fatalf("row after the failure = %+v, want enabled, failed, attempts=1", row)
	}
	if !row.NextRunAt.After(now) || row.NextRunAt.After(now.Add(5*time.Minute)) {
		t.Fatalf("retry at %v, want it minutes from now rather than at the next occurrence", row.NextRunAt)
	}
	if row.LastError == "" {
		t.Fatal("last_error is empty; a retried failure must stay readable")
	}
}

// A calendar task that names something the parser cannot read must be parked,
// not retried forever: it was written by a hand-edit or an older writer, and
// the worker would otherwise claim it on every tick.
func TestScheduledTaskCalendarWithUnusableScheduleIsParked(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	action := &countingAction{}
	svc := NewScheduledTaskService(db).RegisterAction("probe", action)
	ctx := context.Background()

	now := time.Now().UTC()
	task := seedScheduledTask(t, db, models.ScheduledTask{
		UserID:    5,
		Kind:      "probe",
		Schedule:  "fortnightly@mon@09:00",
		NextRunAt: now.Add(-time.Minute),
		Enabled:   true,
	})
	if ran, err := svc.RunDue(ctx, now); err != nil || ran != 0 {
		t.Fatalf("run due = %d, %v; want 0, nil - an unreadable schedule must not run", ran, err)
	}
	if action.count() != 0 {
		t.Fatalf("action ran %d times, want 0", action.count())
	}
	row := loadScheduledTask(t, db, task.ID)
	if row.Enabled {
		t.Fatal("a task with an unusable schedule must be disabled, not left due")
	}
	if row.LastStatus != ScheduledTaskStatusFailed || row.LastError == "" {
		t.Fatalf("row = %+v, want the failure and its reason recorded", row)
	}
	// And it stays parked: a second pass has nothing to claim.
	if ran, err := svc.RunDue(ctx, now.Add(time.Minute)); err != nil || ran != 0 {
		t.Fatalf("second pass = %d, %v; want 0, nil", ran, err)
	}
}

// The two ways of saying "when" are mutually exclusive, and an unknown zone is
// a client error rather than a stored row that fires at the wrong time.
func TestScheduledTaskCreateRejectsMixedOrBrokenCalendarSchedules(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	svc := NewScheduledTaskService(db).RegisterAction("probe", &countingAction{})
	ctx := context.Background()
	future := time.Now().Add(time.Hour)

	cases := []struct {
		name string
		spec ScheduledTaskSpec
	}{
		{name: "run_at and schedule together", spec: ScheduledTaskSpec{
			Kind: "probe", RunAt: future, Schedule: "daily@05:30",
		}},
		{name: "repeat_seconds with schedule", spec: ScheduledTaskSpec{
			Kind: "probe", Schedule: "daily@05:30", RepeatSeconds: 60,
		}},
		{name: "timezone without schedule", spec: ScheduledTaskSpec{
			Kind: "probe", RunAt: future, Timezone: "Asia/Shanghai",
		}},
		{name: "unknown cadence", spec: ScheduledTaskSpec{
			Kind: "probe", Schedule: "fortnightly@mon@09:00",
		}},
		{name: "impossible clock", spec: ScheduledTaskSpec{
			Kind: "probe", Schedule: "daily@25:00",
		}},
		{name: "unknown timezone", spec: ScheduledTaskSpec{
			Kind: "probe", Schedule: "daily@05:30", Timezone: "Mars/Olympus",
		}},
		{name: "skip_holidays without a calendar schedule", spec: ScheduledTaskSpec{
			Kind: "probe", RunAt: future, SkipHolidays: true,
		}},
		{name: "skip_holidays on an interval task", spec: ScheduledTaskSpec{
			Kind: "probe", RunAt: future, RepeatSeconds: 3600, SkipHolidays: true,
		}},
	}
	for _, tc := range cases {
		if _, err := svc.Create(ctx, 1, tc.spec); !errors.Is(err, ErrInvalidScheduledTaskCalendar) {
			t.Fatalf("%s: err = %v, want ErrInvalidScheduledTaskCalendar", tc.name, err)
		}
	}
}

// A one-shot task fires once, records what happened, and does not stay armed.
func TestScheduledTaskRunDueFiresOnceAndRecordsOutcome(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	action := &countingAction{}
	svc := NewScheduledTaskService(db).RegisterAction("probe", action)
	ctx := context.Background()

	// Seeded already due: Create refuses a past run_at on purpose, so a test
	// that wants a due task writes the row directly.
	task := seedScheduledTask(t, db, models.ScheduledTask{
		UserID: 7, Kind: "probe", NextRunAt: time.Now().Add(-time.Minute), Enabled: true,
	})

	ran, err := svc.RunDue(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("run due: %v", err)
	}
	if ran != 1 || action.count() != 1 {
		t.Fatalf("ran=%d action.count=%d, want 1 and 1", ran, action.count())
	}
	after := loadScheduledTask(t, db, task.ID)
	if after.LastStatus != ScheduledTaskStatusOK || after.LastRunAt == nil {
		t.Fatalf("outcome = %+v, want ok with last_run_at set", after)
	}
	if after.Enabled {
		t.Fatal("a one-shot task must be disabled after it runs")
	}

	// Second pass: nothing due, nothing runs.
	if ran, err = svc.RunDue(ctx, time.Now().UTC()); err != nil {
		t.Fatalf("second run due: %v", err)
	}
	if ran != 0 || action.count() != 1 {
		t.Fatalf("second pass ran %d (action count %d), want 0 and 1", ran, action.count())
	}
}

// The claim is the whole safety story: with several workers racing, each due
// task must be handed to exactly one of them.
func TestScheduledTaskClaimIsExclusiveAcrossWorkers(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	action := &countingAction{}
	svc := NewScheduledTaskService(db).RegisterAction("probe", action)
	ctx := context.Background()

	const rows = 60
	for i := 0; i < rows; i++ {
		seedScheduledTask(t, db, models.ScheduledTask{
			UserID: 7, Kind: "probe", NextRunAt: time.Now().Add(-time.Minute), Enabled: true,
		})
	}

	var wg sync.WaitGroup
	for w := 0; w < 6; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				n, err := svc.RunDue(ctx, time.Now().UTC())
				if err != nil || n == 0 {
					return
				}
			}
		}()
	}
	wg.Wait()

	if action.count() != rows {
		t.Fatalf("action ran %d times for %d tasks; the claim handed some out twice", action.count(), rows)
	}
	var stillEnabled int64
	if err := db.Model(&models.ScheduledTask{}).Where("enabled = TRUE").Count(&stillEnabled).Error; err != nil {
		t.Fatalf("count enabled: %v", err)
	}
	if stillEnabled != 0 {
		t.Fatalf("%d one-shot tasks are still enabled", stillEnabled)
	}
}

// A recurring task reschedules from now rather than replaying every period the
// worker was down for, and stays armed.
func TestScheduledTaskRecurringReschedulesFromNow(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	action := &countingAction{}
	svc := NewScheduledTaskService(db).RegisterAction("probe", action)
	ctx := context.Background()

	task := seedScheduledTask(t, db, models.ScheduledTask{
		UserID: 7, Kind: "probe",
		// Ten periods in the past: the scheduler must fire once, not ten times.
		NextRunAt:     time.Now().Add(-10 * time.Minute),
		RepeatSeconds: 60,
		Enabled:       true,
	})
	now := time.Now().UTC()
	if ran, err := svc.RunDue(ctx, now); err != nil || ran != 1 {
		t.Fatalf("run due = %d, %v; want 1, nil", ran, err)
	}
	after := loadScheduledTask(t, db, task.ID)
	if !after.Enabled {
		t.Fatal("a recurring task must stay enabled")
	}
	if after.NextRunAt.Before(now.Add(30 * time.Second)) {
		t.Fatalf("next_run_at = %s, want it rescheduled roughly one period from now", after.NextRunAt)
	}
	if action.count() != 1 {
		t.Fatalf("action ran %d times, want exactly 1 (missed periods are skipped, not replayed)", action.count())
	}
}

// A failing action is recorded, not retried in place: the schedule has already
// moved on, so the operator reads last_error instead of watching a hot loop.
func TestScheduledTaskFailureIsRecorded(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	svc := NewScheduledTaskService(db).RegisterAction("boom", &countingAction{err: errors.New("redis is down")})
	ctx := context.Background()

	task := seedScheduledTask(t, db, models.ScheduledTask{
		UserID: 7, Kind: "boom", NextRunAt: time.Now().Add(-time.Minute), Enabled: true,
	})
	if _, err := svc.RunDue(ctx, time.Now().UTC()); err != nil {
		t.Fatalf("run due: %v", err)
	}
	after := loadScheduledTask(t, db, task.ID)
	if after.LastStatus != ScheduledTaskStatusFailed {
		t.Fatalf("last_status = %q, want failed", after.LastStatus)
	}
	if after.LastError != "redis is down" {
		t.Fatalf("last_error = %q, want the action's message", after.LastError)
	}
}

// Tasks are owner-scoped: another user's id is invisible, not deletable.
func TestScheduledTaskListAndDeleteAreOwnerScoped(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	svc := NewScheduledTaskService(db).RegisterAction("probe", &countingAction{})
	ctx := context.Background()

	mine := seedScheduledTask(t, db, models.ScheduledTask{
		UserID: 1, Kind: "probe", NextRunAt: time.Now().Add(time.Hour), Enabled: true,
	})
	theirs := seedScheduledTask(t, db, models.ScheduledTask{
		UserID: 2, Kind: "probe", NextRunAt: time.Now().Add(time.Hour), Enabled: true,
	})

	rows, err := svc.List(ctx, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != mine.ID {
		t.Fatalf("list for user 1 returned %+v, want only their own task", rows)
	}
	if _, err := svc.Get(ctx, 1, theirs.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("cross-user get err = %v, want ErrRecordNotFound", err)
	}
	if err := svc.Delete(ctx, 1, theirs.ID); err != nil {
		t.Fatalf("cross-user delete: %v", err)
	}
	if after := loadScheduledTask(t, db, theirs.ID); after.ID != theirs.ID {
		t.Fatal("another user's task was deleted")
	}
}

// flakyAction fails its first `failures` runs and then succeeds, so a test can
// watch one occurrence get retried instead of losing its only chance.
type flakyAction struct {
	mu       sync.Mutex
	failures int
	runs     int
}

func (a *flakyAction) Run(_ context.Context, _ models.ScheduledTask) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.runs++
	if a.runs <= a.failures {
		return errors.New("boom")
	}
	return nil
}

func (a *flakyAction) count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.runs
}

// A one-shot task that fails once must not lose its only chance: the retry
// budget re-arms it after a short backoff, and a later success clears the
// counter.
func TestScheduledTaskRetriesFailedOneShotThenSucceeds(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	action := &flakyAction{failures: 1}
	svc := NewScheduledTaskService(db).RegisterAction("probe", action)
	ctx := context.Background()

	now := time.Now().UTC()
	task := seedScheduledTask(t, db, models.ScheduledTask{
		UserID:      1,
		Kind:        "probe",
		NextRunAt:   now.Add(-time.Second),
		Enabled:     true,
		MaxAttempts: 3,
	})

	if n, err := svc.RunDue(ctx, now); err != nil || n != 1 {
		t.Fatalf("first RunDue = %d,%v, want 1,nil", n, err)
	}
	row := loadScheduledTask(t, db, task.ID)
	if !row.Enabled || row.Attempts != 1 || row.LastStatus != ScheduledTaskStatusFailed {
		t.Fatalf("after the first failure row = %+v, want enabled with attempts=1", row)
	}
	if d := row.NextRunAt.Sub(now); d <= 0 || d > time.Minute {
		t.Fatalf("retry scheduled %v after the failure, want a short backoff inside a minute", d)
	}

	if n, err := svc.RunDue(ctx, row.NextRunAt.Add(time.Second)); err != nil || n != 1 {
		t.Fatalf("retry RunDue = %d,%v, want 1,nil", n, err)
	}
	row = loadScheduledTask(t, db, task.ID)
	if row.Enabled || row.Attempts != 0 || row.LastStatus != ScheduledTaskStatusOK {
		t.Fatalf("after the retry succeeded row = %+v, want finished with the counter cleared", row)
	}
	if action.count() != 2 {
		t.Fatalf("action ran %d times, want 2", action.count())
	}
}

// The budget is finite: once it is spent the one-shot parks with the error
// instead of spinning, and no later pass picks it up again.
func TestScheduledTaskParksOneShotWhenAttemptsRunOut(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	action := &flakyAction{failures: 99}
	svc := NewScheduledTaskService(db).RegisterAction("probe", action)
	ctx := context.Background()

	now := time.Now().UTC()
	task := seedScheduledTask(t, db, models.ScheduledTask{
		UserID:      1,
		Kind:        "probe",
		NextRunAt:   now.Add(-time.Second),
		Enabled:     true,
		MaxAttempts: 2,
	})

	if n, _ := svc.RunDue(ctx, now); n != 1 {
		t.Fatalf("first pass ran %d tasks, want 1", n)
	}
	row := loadScheduledTask(t, db, task.ID)
	if !row.Enabled || row.Attempts != 1 {
		t.Fatalf("after the first failure row = %+v, want enabled with attempts=1", row)
	}

	second := row.NextRunAt.Add(time.Second)
	if n, _ := svc.RunDue(ctx, second); n != 1 {
		t.Fatalf("second pass ran %d tasks, want 1", n)
	}
	row = loadScheduledTask(t, db, task.ID)
	if row.Enabled || row.Attempts != 2 || row.LastStatus != ScheduledTaskStatusFailed {
		t.Fatalf("exhausted row = %+v, want disabled with attempts=2", row)
	}

	if n, _ := svc.RunDue(ctx, second.Add(time.Hour)); n != 0 {
		t.Fatalf("a parked task ran again (%d tasks), want 0", n)
	}
	if action.count() != 2 {
		t.Fatalf("action ran %d times, want exactly the budget", action.count())
	}
}

// The default budget is 1, which is the old "fire once, record the error"
// behaviour: no re-arm, the row parks immediately.
func TestScheduledTaskWithoutRetryBudgetDoesNotRetry(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	action := &flakyAction{failures: 99}
	svc := NewScheduledTaskService(db).RegisterAction("probe", action)
	ctx := context.Background()

	now := time.Now().UTC()
	task := seedScheduledTask(t, db, models.ScheduledTask{
		UserID:      1,
		Kind:        "probe",
		NextRunAt:   now.Add(-time.Second),
		Enabled:     true,
		MaxAttempts: 1,
	})

	if n, _ := svc.RunDue(ctx, now); n != 1 {
		t.Fatalf("first pass ran %d tasks, want 1", n)
	}
	row := loadScheduledTask(t, db, task.ID)
	if row.Enabled || row.Attempts != 1 || row.LastStatus != ScheduledTaskStatusFailed {
		t.Fatalf("no-retry row = %+v, want parked after a single attempt", row)
	}
	if action.count() != 1 {
		t.Fatalf("action ran %d times, want 1", action.count())
	}
}

// A recurring task's retry must run sooner than (or at) its next period - it
// exists to recover the failed occurrence, not to drift the schedule.
func TestScheduledTaskRecurringRetryRunsBeforeTheNextPeriod(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	action := &flakyAction{failures: 99}
	svc := NewScheduledTaskService(db).RegisterAction("probe", action)
	ctx := context.Background()

	now := time.Now().UTC()
	task := seedScheduledTask(t, db, models.ScheduledTask{
		UserID:        1,
		Kind:          "probe",
		NextRunAt:     now.Add(-time.Second),
		RepeatSeconds: 3600,
		Enabled:       true,
		MaxAttempts:   3,
	})

	if n, _ := svc.RunDue(ctx, now); n != 1 {
		t.Fatalf("first pass ran %d tasks, want 1", n)
	}
	row := loadScheduledTask(t, db, task.ID)
	if !row.Enabled || row.Attempts != 1 {
		t.Fatalf("row after the failure = %+v, want enabled with attempts=1", row)
	}
	if !row.NextRunAt.Before(now.Add(3600 * time.Second)) {
		t.Fatalf("retry at %v is not before the hourly period end", row.NextRunAt)
	}
	if row.LastError == "" {
		t.Fatal("last_error is empty; a retried failure must still be readable")
	}
}

// capturingNotifier records the events the scheduler publishes, so a test can
// tell "gave up" from "retrying" without a Redis-backed bus.
type capturingNotifier struct {
	mu     sync.Mutex
	events []Event
}

func (n *capturingNotifier) Publish(_ context.Context, evt Event) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.events = append(n.events, evt)
}

func (n *capturingNotifier) all() []Event {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]Event(nil), n.events...)
}

// Alerting has to fire exactly once, when the task gives up. Emitting on every
// retry would train the owner to ignore it, and emitting on success would be
// worse.
func TestScheduledTaskPublishesOnlyWhenItGivesUp(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	action := &flakyAction{failures: 99}
	notifier := &capturingNotifier{}
	svc := NewScheduledTaskService(db).RegisterAction("probe", action).WithNotifier(notifier)
	ctx := context.Background()

	now := time.Now().UTC()
	task := seedScheduledTask(t, db, models.ScheduledTask{
		UserID:      7,
		Kind:        "probe",
		NextRunAt:   now.Add(-time.Second),
		Enabled:     true,
		MaxAttempts: 2,
	})

	// First failure still has budget: the task is retried and nothing is posted.
	if n, _ := svc.RunDue(ctx, now); n != 1 {
		t.Fatalf("first pass ran %d tasks, want 1", n)
	}
	if got := notifier.all(); len(got) != 0 {
		t.Fatalf("published %d events on a retryable failure, want 0", len(got))
	}

	// Second failure spends the budget: exactly one terminal event.
	row := loadScheduledTask(t, db, task.ID)
	if n, _ := svc.RunDue(ctx, row.NextRunAt.Add(time.Second)); n != 1 {
		t.Fatalf("second pass ran %d tasks, want 1", n)
	}
	events := notifier.all()
	if len(events) != 1 {
		t.Fatalf("published %d events after giving up, want exactly 1", len(events))
	}
	if events[0].Type != EventScheduledTaskFailed || events[0].UserID != 7 {
		t.Fatalf("event = %+v, want scheduled_task.failed for user 7", events[0])
	}
	if events[0].Data["kind"] != "probe" || events[0].Data["attempts"] != 2 {
		t.Fatalf("event data = %+v, want kind=probe attempts=2", events[0].Data)
	}
}

// A task that succeeds must not notify at all.
func TestScheduledTaskSuccessPublishesNothing(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	action := &flakyAction{}
	notifier := &capturingNotifier{}
	svc := NewScheduledTaskService(db).RegisterAction("probe", action).WithNotifier(notifier)
	ctx := context.Background()

	now := time.Now().UTC()
	seedScheduledTask(t, db, models.ScheduledTask{
		UserID:      7,
		Kind:        "probe",
		NextRunAt:   now.Add(-time.Second),
		Enabled:     true,
		MaxAttempts: 3,
	})

	if n, _ := svc.RunDue(ctx, now); n != 1 {
		t.Fatalf("pass ran %d tasks, want 1", n)
	}
	if got := notifier.all(); len(got) != 0 {
		t.Fatalf("published %d events for a successful run, want 0", len(got))
	}
}

// The event has to be subscribable, or DispatchEvent would drop it and the
// alert would exist only in the audit trail.
func TestScheduledTaskFailedIsSubscribable(t *testing.T) {
	if !IsSupportedUserNotificationEvent(EventScheduledTaskFailed) {
		t.Fatal("scheduled_task.failed is not in SupportedUserNotificationEvents")
	}
}
