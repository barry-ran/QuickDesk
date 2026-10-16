package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"quickdesk/signaling/internal/holiday"
	"quickdesk/signaling/internal/models"
)

// ErrInvalidScheduledTaskCalendar covers a calendar `schedule` or the
// `timezone` it is evaluated in. It is a separate sentinel from
// ErrInvalidScheduledTaskSchedule because "run_at must be in the future" and
// "weekly@funday@09:00 is not a thing" deserve different messages in the API.
var ErrInvalidScheduledTaskCalendar = errors.New("invalid calendar schedule")

// The calendar cadences. This is deliberately a small, explicit DSL instead of
// a five-field cron expression: the tasks an operator schedules here are
// daily / weekly / monthly maintenance, and an explicit form is something the
// API can explain back in a 400 message instead of leaving them to count
// asterisks.
const (
	calendarScheduleDaily    = "daily"
	calendarScheduleWeekdays = "weekdays"
	calendarScheduleWeekly   = "weekly"
	calendarScheduleMonthly  = "monthly"
	calendarScheduleEvery    = "every"
)

// SupportedCalendarSchedules lists the accepted forms, for the API's discovery
// field. Kept next to the parser so the two cannot drift.
func SupportedCalendarSchedules() []string {
	return []string{
		"daily@HH:MM",
		"weekdays@HH:MM",
		"weekly@<dow>@HH:MM",
		"monthly@<dom>@HH:MM",
		"every@<N>@days@HH:MM",
	}
}

// calendarSchedule is a parsed recurrence. `next` turns it into the concrete
// instant the worker should run at, which is the only thing the row stores.
type calendarSchedule struct {
	kind   string
	hour   int
	minute int
	// dow and dom are only meaningful for the matching kind.
	dow time.Weekday
	dom int
	// everyDays is the interval for `every@<N>@days@HH:MM`.
	everyDays int
}

var calendarWeekdayNames = map[string]time.Weekday{
	"sun": time.Sunday, "sunday": time.Sunday,
	"mon": time.Monday, "monday": time.Monday,
	"tue": time.Tuesday, "tuesday": time.Tuesday,
	"wed": time.Wednesday, "wednesday": time.Wednesday,
	"thu": time.Thursday, "thursday": time.Thursday,
	"fri": time.Friday, "friday": time.Friday,
	"sat": time.Saturday, "saturday": time.Saturday,
}

// parseCalendarSchedule accepts `daily@05:30`, `weekdays@09:00`,
// `weekly@mon@09:00` (weekday names or 0..6, Sunday = 0),
// `monthly@1@02:00` (day 1..31) and `every@3@days@09:00`.
func parseCalendarSchedule(spec string) (calendarSchedule, error) {
	var out calendarSchedule
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return out, calendarFormatError(trimmed, "empty schedule")
	}
	parts := strings.Split(trimmed, "@")
	if len(parts) < 2 {
		return out, calendarFormatError(trimmed, "expected one of "+strings.Join(SupportedCalendarSchedules(), ", "))
	}
	out.kind = strings.ToLower(strings.TrimSpace(parts[0]))

	// `every@<N>@days@HH:MM` is the one form with a unit in the middle, so it is
	// parsed before the generic clock handling below.
	if out.kind == calendarScheduleEvery {
		if len(parts) != 4 {
			return out, calendarFormatError(trimmed, "expected every@<N>@days@HH:MM")
		}
		days, err := parseCalendarInterval(parts[1])
		if err != nil {
			return out, calendarFormatError(trimmed, err.Error())
		}
		unit := strings.ToLower(strings.TrimSpace(parts[2]))
		if unit != "days" {
			return out, calendarFormatError(trimmed,
				"only whole days are supported, so the unit must be `days`")
		}
		hour, minute, err := parseCalendarClock(strings.TrimSpace(parts[3]))
		if err != nil {
			return out, calendarFormatError(trimmed, err.Error())
		}
		out.everyDays = days
		out.hour, out.minute = hour, minute
		return out, nil
	}

	switch out.kind {
	case calendarScheduleDaily, calendarScheduleWeekdays, calendarScheduleWeekly, calendarScheduleMonthly:
	default:
		return out, calendarFormatError(trimmed,
			"unknown cadence "+strconv.Quote(out.kind)+"; expected daily, weekdays, weekly, monthly or every")
	}
	wantParts := 2
	if out.kind == calendarScheduleWeekly || out.kind == calendarScheduleMonthly {
		wantParts = 3
	}
	if len(parts) != wantParts {
		return out, calendarFormatError(trimmed, "expected one of "+strings.Join(SupportedCalendarSchedules(), ", "))
	}
	hour, minute, err := parseCalendarClock(strings.TrimSpace(parts[len(parts)-1]))
	if err != nil {
		return out, calendarFormatError(trimmed, err.Error())
	}
	out.hour, out.minute = hour, minute
	if out.kind == calendarScheduleDaily || out.kind == calendarScheduleWeekdays {
		return out, nil
	}
	selector := strings.ToLower(strings.TrimSpace(parts[1]))
	if out.kind == calendarScheduleWeekly {
		dow, err := parseCalendarWeekday(selector)
		if err != nil {
			return out, calendarFormatError(trimmed, err.Error())
		}
		out.dow = dow
		return out, nil
	}
	dom, err := parseCalendarMonthDay(selector)
	if err != nil {
		return out, calendarFormatError(trimmed, err.Error())
	}
	out.dom = dom
	return out, nil
}

func calendarFormatError(spec, reason string) error {
	return fmt.Errorf("%w: %q: %s", ErrInvalidScheduledTaskCalendar, spec, reason)
}

// parseCalendarClock reads HH:MM in 24-hour form.
func parseCalendarClock(value string) (int, int, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, 0, errors.New("time must be HH:MM")
	}
	hour, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, errors.New("hour must be 00..23")
	}
	minute, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, errors.New("minute must be 00..59")
	}
	return hour, minute, nil
}

func parseCalendarWeekday(value string) (time.Weekday, error) {
	if dow, ok := calendarWeekdayNames[value]; ok {
		return dow, nil
	}
	n, err := strconv.Atoi(value)
	if err == nil && n >= 0 && n <= 6 {
		return time.Weekday(n), nil
	}
	return 0, errors.New("weekday must be mon..sun or 0..6 (0 = Sunday)")
}

func parseCalendarMonthDay(value string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 || n > 31 {
		return 0, errors.New("monthly day of month must be 1..31")
	}
	return n, nil
}

// parseCalendarInterval reads the N of `every@N@days@HH:MM`. The ceiling exists
// so a typo cannot create a task whose next occurrence is decades away, which
// would look like a silently dead row.
func parseCalendarInterval(value string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n < 1 || n > 365 {
		return 0, errors.New("interval must be 1..365 days")
	}
	return n, nil
}

// next returns the first occurrence strictly after `after`, evaluated on the
// wall clock of loc. Wall-clock semantics are the point: a task written as
// "03:00" has to stay 03:00 local across a DST change, not drift to 02:00.
//
// The loops walk forward rather than solving arithmetically because a DST
// transition can normalise a candidate onto the wrong side of `after` (a
// skipped hour moves forward, an ambiguous one picks a side), and one extra
// comparison is cheaper than the special cases.
func (c calendarSchedule) next(after time.Time, loc *time.Location, anchor time.Time) time.Time {
	local := after.In(loc)
	switch c.kind {
	case calendarScheduleDaily:
		for day := 0; day <= 2; day++ {
			candidate := c.onDay(local.AddDate(0, 0, day), loc)
			if candidate.After(after) {
				return candidate
			}
		}
		return c.onDay(local.AddDate(0, 0, 3), loc)
	case calendarScheduleWeekdays:
		// Working days by the calendar only: statutory days off are a separate,
		// opt-in rule (`skip_holidays`), because "weekdays" and "not a national
		// holiday" are different questions and 调休 workdays are still weekdays.
		for day := 0; day <= 7; day++ {
			dayInLoc := local.AddDate(0, 0, day)
			switch dayInLoc.Weekday() {
			case time.Saturday, time.Sunday:
				continue
			}
			candidate := c.onDay(dayInLoc, loc)
			if candidate.After(after) {
				return candidate
			}
		}
		return c.onDay(local.AddDate(0, 0, 7), loc)
	case calendarScheduleEvery:
		// The phase is anchored to the task's creation date, not to the previous
		// run: "every 3 days" has to mean the same dates even if one occurrence
		// ran late, and a worker restart must not shift the series.
		days := c.everyDays
		if days <= 0 {
			days = 1
		}
		base := anchor
		if base.IsZero() {
			// Only possible for a row written outside the API; falling back to
			// "from now" keeps the task alive instead of failing it every tick.
			base = after
		}
		baseLocal := base.In(loc)
		baseDay := time.Date(baseLocal.Year(), baseLocal.Month(), baseLocal.Day(), 0, 0, 0, 0, loc)
		today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
		periods := civilDaysBetween(baseDay, today) / days
		if periods < 0 {
			periods = 0
		}
		for k := periods; k <= periods+2; k++ {
			candidate := c.onDay(baseDay.AddDate(0, 0, k*days), loc)
			if candidate.After(after) {
				return candidate
			}
		}
		return c.onDay(baseDay.AddDate(0, 0, (periods+3)*days), loc)
	case calendarScheduleWeekly:
		for day := 0; day <= 14; day++ {
			dayOfWeek := local.AddDate(0, 0, day)
			if dayOfWeek.Weekday() != c.dow {
				continue
			}
			candidate := c.onDay(dayOfWeek, loc)
			if candidate.After(after) {
				return candidate
			}
		}
		return c.onDay(local.AddDate(0, 0, 14), loc)
	case calendarScheduleMonthly:
		// Months are counted arithmetically, not with AddDate: adding a month to
		// the 31st overflows into the month after the short one (Jan 31 + 1mo
		// is Mar 3), which would skip February entirely.
		startYear, startMonth := local.Year(), int(local.Month())
		for monthsAhead := 0; monthsAhead <= 2; monthsAhead++ {
			year, month := offsetMonth(startYear, startMonth, monthsAhead)
			candidate := c.onMonth(year, time.Month(month), loc)
			if candidate.After(after) {
				return candidate
			}
		}
		year, month := offsetMonth(startYear, startMonth, 3)
		return c.onMonth(year, time.Month(month), loc)
	}
	// Unreachable: the parser only produces the kinds handled above.
	return local.Add(time.Minute)
}

// civilDaysBetween counts calendar days between two instants, comparing their
// civil date rather than their duration. A duration-based count is off by one
// across a DST change, because one of those days is 23 or 25 hours long.
func civilDaysBetween(from, to time.Time) int {
	a := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	b := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(b.Sub(a).Hours() / 24)
}

// advancePastHolidays moves an occurrence off a statutory day off, keeping the
// wall-clock time, and returns the first non-holiday instant at or after it.
//
// The walk is day by day because the Spring Festival break is more than a week
// long and hard-coding its length would be wrong every year. The bound is a
// safety net against a calendar file that marks every day as a holiday: a task
// that can never run is worse than one that runs on a holiday.
func advancePastHolidays(occurrence time.Time, loc *time.Location, calendar *holiday.Calendar) time.Time {
	if calendar == nil {
		return occurrence
	}
	for i := 0; i < 30; i++ {
		local := occurrence.In(loc)
		if day, ok := calendar.Lookup(local); !ok || !day.IsOffDay {
			return occurrence
		}
		occurrence = resolveWallClock(local.Year(), local.Month(), local.Day()+1, local.Hour(), local.Minute(), loc)
	}
	return occurrence
}

// onDay is the schedule's time of day on whichever calendar day `day` is.
func (c calendarSchedule) onDay(day time.Time, loc *time.Location) time.Time {
	return resolveWallClock(day.Year(), day.Month(), day.Day(), c.hour, c.minute, loc)
}

// onMonth clamps a day-of-month the month cannot hold, so `monthly@31` runs on
// the 30th in April and the 28th (29th in a leap year) in February instead of
// silently skipping those months. A maintenance task that must run every month
// is the more useful contract, and the docs state the clamp.
func (c calendarSchedule) onMonth(year int, month time.Month, loc *time.Location) time.Time {
	day := c.dom
	if last := daysInMonth(year, month); day > last {
		day = last
	}
	return resolveWallClock(year, month, day, c.hour, c.minute, loc)
}

func daysInMonth(year int, month time.Month) int {
	// Day 0 of the following month is the last day of this one.
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// offsetMonth adds whole months to a year/month pair, carrying into the year.
func offsetMonth(year, month, months int) (int, int) {
	index := year*12 + (month - 1) + months
	return index / 12, index%12 + 1
}

// resolveWallClock builds the instant for a local wall-clock time on the given
// day.
//
// The reason this is not a one-line time.Date: a DST spring-forward deletes an
// hour of wall-clock time, and Go normalises a value inside that gap
// *backwards* (US 2026-03-08 02:30 comes back as 01:30 -05:00). A task written
// as "02:30" would then fire an hour early, in the middle of the night, once a
// year. Stepping forward until the wall clock reaches the requested time lands
// on the first real instant at or after it - 03:00 for a one-hour gap - which
// is the rule cron and systemd timers use for a skipped hour. The loop exits on
// the first iteration on every ordinary day.
func resolveWallClock(year int, month time.Month, day, hour, minute int, loc *time.Location) time.Time {
	candidate := time.Date(year, month, day, hour, minute, 0, 0, loc)
	wanted := hour*60 + minute
	for i := 0; i < 181; i++ {
		local := candidate.In(loc)
		if local.Day() != day {
			// The normalisation rolled past midnight; nothing better to return.
			return candidate
		}
		if local.Hour()*60+local.Minute() >= wanted {
			return candidate
		}
		candidate = candidate.Add(time.Minute)
	}
	return candidate
}

// scheduledTaskLocation resolves the zone a calendar schedule is evaluated in.
// Empty means UTC, which is what every row written before this existed gets.
func scheduledTaskLocation(timezone string) (*time.Location, error) {
	trimmed := strings.TrimSpace(timezone)
	if trimmed == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%w: timezone %q is not a known IANA name", ErrInvalidScheduledTaskCalendar, trimmed)
	}
	return loc, nil
}

// scheduledTaskRecurs reports whether the task has another occurrence after the
// one being claimed: an interval task with a period, or a calendar task.
func scheduledTaskRecurs(task models.ScheduledTask) bool {
	return strings.TrimSpace(task.Schedule) != "" || task.RepeatSeconds > 0
}

// scheduledTaskNextOccurrence is the single place the two recurrence shapes
// meet, so the claim and the retry clamp cannot drift apart. For a calendar
// task it is computed from the task's own timezone, so "daily@05:30" keeps
// firing at 05:30 local even while the server's clock lives in UTC.
func scheduledTaskNextOccurrence(task models.ScheduledTask, after time.Time) (time.Time, error) {
	if strings.TrimSpace(task.Schedule) != "" {
		spec, err := parseCalendarSchedule(task.Schedule)
		if err != nil {
			return time.Time{}, err
		}
		loc, err := scheduledTaskLocation(task.Timezone)
		if err != nil {
			return time.Time{}, err
		}
		next := spec.next(after, loc, task.CreatedAt)
		if task.SkipHolidays {
			next = advancePastHolidays(next, loc, currentHolidayCalendar())
		}
		return next, nil
	}
	if task.RepeatSeconds > 0 {
		return after.Add(time.Duration(task.RepeatSeconds) * time.Second), nil
	}
	return time.Time{}, nil
}
