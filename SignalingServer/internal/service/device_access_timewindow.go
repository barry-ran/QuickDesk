package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"quickdesk/signaling/internal/holiday"
)

// ErrInvalidAccessWindow means an allowed-access window is malformed (bad day,
// bad "HH:MM" time, or no start/end).
var ErrInvalidAccessWindow = errors.New("access windows need days 0-6 and HH:MM start/end times")

// ErrInvalidTimezone means the IANA timezone could not be loaded.
var ErrInvalidTimezone = errors.New("timezone must be an IANA name like Asia/Shanghai")

// ErrInvalidAccessException means an exception date is malformed (bad date or
// range, unknown mode/repeat/holiday, mixed sources) or two exceptions with
// different modes cover the same day.
var ErrInvalidAccessException = errors.New("access exception dates need exactly one of date/end_date, holiday, or repeat \"lunar\" with lunar_month/lunar_day; mode \"allow\" or \"deny\"; and no overlapping allow/deny entries")

// Exception-date modes.
const (
	AccessExceptionModeAllow = "allow"
	AccessExceptionModeDeny  = "deny"
)

// AccessExceptionRepeatYearly makes an exception recur every year on the same
// month-day(s), so "Spring Festival" or "New Year" is one entry instead of one
// per year.
const AccessExceptionRepeatYearly = "yearly"

// AccessExceptionRepeatLunar makes an exception follow the lunar calendar
// instead of a fixed month-day, so 春节 (正月初一), 端午 (五月初五) and 中秋
// (八月十五) are one entry that is correct every year. Requires LunarMonth and
// LunarDay.
const AccessExceptionRepeatLunar = "lunar"

// AccessExceptionHolidayDay selects which dates of a named statutory holiday
// an entry covers: the days off (default) or the make-up workdays (调休上班).
const (
	AccessExceptionHolidayDayOff  = "off"
	AccessExceptionHolidayDayWork = "work"
)

var (
	policyCalendarMu sync.RWMutex
	policyCalendar   = holiday.Builtin()
)

// SetHolidayCalendar replaces the process-wide statutory holiday calendar used
// to resolve `holiday` exception entries. main calls it once at startup with
// the configured calendar; tests may swap it to make results deterministic.
// A nil calendar is ignored so a failed load cannot silently empty the data.
func SetHolidayCalendar(c *holiday.Calendar) {
	if c == nil {
		return
	}
	policyCalendarMu.Lock()
	policyCalendar = c
	policyCalendarMu.Unlock()
}

func currentHolidayCalendar() *holiday.Calendar {
	policyCalendarMu.RLock()
	defer policyCalendarMu.RUnlock()
	return policyCalendar
}

// AccessExceptionDate is a calendar override, evaluated in the policy's
// timezone. "deny" blocks access for the covered day(s) (holidays,
// maintenance); "allow" permits them and overrides the weekly windows (a
// one-off work day). Only the *time* rule is decided here - remote_disabled
// and the IP allow list are still enforced.
//
// Exactly one source may be set:
//
//   - Date (+ optional EndDate): a fixed date or inclusive range, optionally
//     recurring every year on the same month-day(s) via Repeat:"yearly", which
//     may wrap the new year (e.g. 12-30 through 01-02);
//   - Holiday: the name of an entry in the statutory holiday calendar
//     (e.g. "春节"). HolidayDay picks the days off (default) or the make-up
//     workdays, so both "放假别连我机器" and "调休那天要开工" are one entry;
//   - Repeat:"lunar" with LunarMonth/LunarDay: a date on the lunar calendar
//     (春节 1/1, 端午 5/5, 中秋 8/15) that resolves to a different solar date
//     each year, so it never needs an annual edit.
type AccessExceptionDate struct {
	Date    string `json:"date,omitempty"`
	EndDate string `json:"end_date,omitempty"`
	Mode    string `json:"mode"`
	Repeat  string `json:"repeat,omitempty"`
	// Holiday names a statutory holiday in the configured calendar (e.g.
	// "春节", "国庆节"). Mutually exclusive with Date and lunar fields.
	Holiday string `json:"holiday,omitempty"`
	// HolidayDay is "off" (default) or "work" for a Holiday entry.
	HolidayDay string `json:"holiday_day,omitempty"`
	// LunarMonth/LunarDay describe a lunar date when Repeat is "lunar".
	LunarMonth int `json:"lunar_month,omitempty"`
	LunarDay   int `json:"lunar_day,omitempty"`
}

// AccessWindow is one recurring window in which remote access is allowed.
//
// Days use Go's weekday order, 0 = Sunday … 6 = Saturday; an empty list means
// every day. Start and End are wall-clock "HH:MM" in the policy's timezone.
// Start > End means the window wraps past midnight, and the day filter then
// applies to the day the window *starts* (so `{days:[5],start:"22:00",end:"06:00"}`
// covers Friday 22:00 through Saturday 06:00).
type AccessWindow struct {
	Days  []int  `json:"days,omitempty"`
	Start string `json:"start"`
	End   string `json:"end"`
}

// ValidateAccessWindows rejects malformed windows. An empty list is valid and
// means "no time restriction".
func ValidateAccessWindows(windows []AccessWindow) error {
	for i, w := range windows {
		for _, d := range w.Days {
			if d < 0 || d > 6 {
				return fmt.Errorf("%w: window %d has day %d (want 0-6)", ErrInvalidAccessWindow, i, d)
			}
		}
		if _, err := parseClock(w.Start); err != nil {
			return fmt.Errorf("%w: window %d start %q", ErrInvalidAccessWindow, i, w.Start)
		}
		if _, err := parseClock(w.End); err != nil {
			return fmt.Errorf("%w: window %d end %q", ErrInvalidAccessWindow, i, w.End)
		}
	}
	return nil
}

// ValidateTimezone accepts an empty string (meaning the server's local zone)
// or an IANA location name.
func ValidateTimezone(name string) error {
	if strings.TrimSpace(name) == "" {
		return nil
	}
	if _, err := time.LoadLocation(name); err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidTimezone, name)
	}
	return nil
}

// ValidateAccessExceptions rejects malformed exception entries and refuses to
// accept an allow and a deny entry that cover the same day, so a policy can
// never be ambiguous about whether a given day is allowed or denied. Entries
// with the same mode may overlap (harmless duplication).
//
// Overlap is checked by resolving every entry into concrete dates around the
// current year. That makes holiday and lunar entries comparable with fixed
// dates - which a pure month-day comparison could not do - while keeping the
// existing behaviour for date and yearly entries.
func ValidateAccessExceptions(exceptions []AccessExceptionDate) error {
	for i, e := range exceptions {
		if err := validateAccessExceptionShape(e); err != nil {
			return fmt.Errorf("%w: exception %d %s", ErrInvalidAccessException, i, err)
		}
	}

	refYear := time.Now().UTC().Year()
	years := overlapYears(exceptions, refYear)
	sets := make([]map[string]struct{}, len(exceptions))
	for i, e := range exceptions {
		sets[i] = exceptionDateSet(e, years)
	}
	for i := 0; i < len(exceptions); i++ {
		for j := i + 1; j < len(exceptions); j++ {
			if sameMode(exceptions[i].Mode, exceptions[j].Mode) {
				continue
			}
			if dateSetsIntersect(sets[i], sets[j]) {
				return fmt.Errorf("%w: exceptions %d and %d both cover a day with different modes",
					ErrInvalidAccessException, i, j)
			}
		}
	}
	return nil
}

// validateAccessExceptionShape checks one entry in isolation: exactly one
// source, a known mode/repeat, and a sane range.
func validateAccessExceptionShape(e AccessExceptionDate) error {
	mode := strings.ToLower(strings.TrimSpace(e.Mode))
	if mode != AccessExceptionModeAllow && mode != AccessExceptionModeDeny {
		return fmt.Errorf("mode %q", e.Mode)
	}
	repeat := strings.ToLower(strings.TrimSpace(e.Repeat))
	if repeat != "" && repeat != AccessExceptionRepeatYearly && repeat != AccessExceptionRepeatLunar {
		return fmt.Errorf("repeat %q", e.Repeat)
	}
	holidayName := strings.TrimSpace(e.Holiday)
	lunar := repeat == AccessExceptionRepeatLunar

	switch {
	case lunar:
		if strings.TrimSpace(e.Date) != "" || strings.TrimSpace(e.EndDate) != "" {
			return errors.New("has both lunar and date fields")
		}
		if holidayName != "" {
			return errors.New("has both lunar and holiday fields")
		}
		if e.LunarMonth < 1 || e.LunarMonth > 12 {
			return fmt.Errorf("lunar_month %d", e.LunarMonth)
		}
		if e.LunarDay < 1 || e.LunarDay > 30 {
			return fmt.Errorf("lunar_day %d", e.LunarDay)
		}
		if strings.TrimSpace(e.HolidayDay) != "" {
			return errors.New("holiday_day is only valid with holiday")
		}
		return nil
	case holidayName != "":
		if strings.TrimSpace(e.Date) != "" || strings.TrimSpace(e.EndDate) != "" {
			return errors.New("has both holiday and date fields")
		}
		if repeat != "" {
			return errors.New("holiday entries recur by definition; drop repeat")
		}
		if e.LunarMonth != 0 || e.LunarDay != 0 {
			return errors.New("has both holiday and lunar fields")
		}
		day := strings.ToLower(strings.TrimSpace(e.HolidayDay))
		if day != "" && day != AccessExceptionHolidayDayOff && day != AccessExceptionHolidayDayWork {
			return fmt.Errorf("holiday_day %q", e.HolidayDay)
		}
		// An unknown name would silently never match, so reject it while the
		// calendar has data. A calendar with no years loaded cannot judge.
		cal := currentHolidayCalendar()
		if cal.Len() > 0 && !cal.HasName(holidayName) {
			return fmt.Errorf("unknown holiday %q", holidayName)
		}
		return nil
	default:
		if e.LunarMonth != 0 || e.LunarDay != 0 {
			return errors.New("lunar_month/lunar_day need repeat \"lunar\"")
		}
		if strings.TrimSpace(e.HolidayDay) != "" {
			return errors.New("holiday_day is only valid with holiday")
		}
		start, err := parseExceptionDate(e.Date)
		if err != nil {
			return fmt.Errorf("date %q", e.Date)
		}
		end := start
		if strings.TrimSpace(e.EndDate) != "" {
			if end, err = parseExceptionDate(e.EndDate); err != nil {
				return fmt.Errorf("end_date %q", e.EndDate)
			}
		}
		// A yearly exception may wrap the new year, so only a non-recurring
		// range has to be ordered.
		if repeat == "" && end.Before(start) {
			return fmt.Errorf("end_date %s is before date %s", e.EndDate, e.Date)
		}
		return nil
	}
}

// AccessExceptionMatch reports whether the policy's local calendar date for now
// is covered by an exception, and if so whether it allows access. matched=false
// means "no exception today", so the caller should fall back to the weekly
// windows.
func AccessExceptionMatch(exceptions []AccessExceptionDate, loc *time.Location, now time.Time) (matched bool, allowed bool, reason string) {
	if len(exceptions) == 0 {
		return false, false, ""
	}
	if loc == nil {
		loc = time.UTC
	}
	today := now.In(loc)
	day := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)

	// Deny is scanned first as a defensive tie-break; validation already
	// rejects allow/deny overlaps, but a malformed stored policy should not
	// silently become an access grant.
	for _, e := range exceptions {
		if sameMode(e.Mode, AccessExceptionModeDeny) && exceptionCovers(e, day) {
			return true, false, "today is blocked by an access exception date"
		}
	}
	for _, e := range exceptions {
		if sameMode(e.Mode, AccessExceptionModeAllow) && exceptionCovers(e, day) {
			return true, true, ""
		}
	}
	return false, false, ""
}

func sameMode(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// exceptionCovers reports whether the (midnight, UTC-normalised) day falls in
// the exception, honouring yearly recurrence, the lunar calendar, statutory
// holidays and new-year wrapping.
func exceptionCovers(e AccessExceptionDate, day time.Time) bool {
	if name := strings.TrimSpace(e.Holiday); name != "" {
		entry, ok := currentHolidayCalendar().Lookup(day)
		if !ok || entry.Name != name {
			return false
		}
		wantOff := !strings.EqualFold(strings.TrimSpace(e.HolidayDay), AccessExceptionHolidayDayWork)
		return entry.IsOffDay == wantOff
	}
	if strings.EqualFold(strings.TrimSpace(e.Repeat), AccessExceptionRepeatLunar) {
		// The solar year and the lunar year differ for dates before the lunar
		// new year, so try the neighbouring lunar years: 12/29 of lunar 2026
		// lands in solar February 2027.
		for _, y := range []int{day.Year() - 1, day.Year(), day.Year() + 1} {
			if solar, ok := holiday.LunarToSolarClamped(y, e.LunarMonth, e.LunarDay); ok && solar.Equal(day) {
				return true
			}
		}
		return false
	}
	start, err := parseExceptionDate(e.Date)
	if err != nil {
		return false
	}
	end := start
	if strings.TrimSpace(e.EndDate) != "" {
		if end, err = parseExceptionDate(e.EndDate); err != nil {
			return false
		}
	}
	if strings.EqualFold(strings.TrimSpace(e.Repeat), AccessExceptionRepeatYearly) {
		return inMonthDayRange(monthDayKey(start), monthDayKey(end), monthDayKey(day))
	}
	return !day.Before(start) && !day.After(end)
}

// overlapYears collects the years an overlap check has to consider: the
// reference year and its neighbours (so a range that wraps the new year is
// seen from both sides), every year the holiday calendar knows about, and the
// explicit years of fixed-range entries.
func overlapYears(exceptions []AccessExceptionDate, refYear int) []int {
	set := map[int]struct{}{refYear - 1: {}, refYear: {}, refYear + 1: {}}
	for _, y := range currentHolidayCalendar().Years() {
		set[y] = struct{}{}
	}
	for _, e := range exceptions {
		for _, raw := range []string{e.Date, e.EndDate} {
			if strings.TrimSpace(raw) == "" {
				continue
			}
			if d, err := parseExceptionDate(raw); err == nil {
				set[d.Year()] = struct{}{}
			}
		}
	}
	out := make([]int, 0, len(set))
	for y := range set {
		out = append(out, y)
	}
	sort.Ints(out)
	return out
}

// exceptionDateSet resolves an entry into every concrete calendar date it
// covers across the given years.
func exceptionDateSet(e AccessExceptionDate, years []int) map[string]struct{} {
	set := make(map[string]struct{})
	add := func(d time.Time) { set[d.Format("2006-01-02")] = struct{}{} }

	if name := strings.TrimSpace(e.Holiday); name != "" {
		offDay := !strings.EqualFold(strings.TrimSpace(e.HolidayDay), AccessExceptionHolidayDayWork)
		cal := currentHolidayCalendar()
		for _, y := range years {
			for _, d := range cal.Dates(y, name, offDay) {
				add(d)
			}
		}
		return set
	}
	if strings.EqualFold(strings.TrimSpace(e.Repeat), AccessExceptionRepeatLunar) {
		for _, y := range years {
			if d, ok := holiday.LunarToSolarClamped(y, e.LunarMonth, e.LunarDay); ok {
				add(d)
			}
		}
		return set
	}
	start, err := parseExceptionDate(e.Date)
	if err != nil {
		return set
	}
	end := start
	if strings.TrimSpace(e.EndDate) != "" {
		if parsed, err := parseExceptionDate(e.EndDate); err == nil {
			end = parsed
		}
	}
	if strings.EqualFold(strings.TrimSpace(e.Repeat), AccessExceptionRepeatYearly) {
		startKey, endKey := monthDayKey(start), monthDayKey(end)
		for _, y := range years {
			for d := time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC); d.Year() == y; d = d.AddDate(0, 0, 1) {
				if inMonthDayRange(startKey, endKey, monthDayKey(d)) {
					add(d)
				}
			}
		}
		return set
	}
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		add(d)
	}
	return set
}

func dateSetsIntersect(a, b map[string]struct{}) bool {
	if len(a) > len(b) {
		a, b = b, a
	}
	for k := range a {
		if _, ok := b[k]; ok {
			return true
		}
	}
	return false
}

// parseExceptionDate parses YYYY-MM-DD into a date-only UTC time so range
// comparisons never depend on a clock or a location.
func parseExceptionDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(value))
}

// monthDayKey returns month*100+day, a sortable stand-in for "day of year".
func monthDayKey(t time.Time) int {
	return int(t.Month())*100 + t.Day()
}

// inMonthDayRange reports membership in a month-day range that may wrap the new
// year (start > end).
func inMonthDayRange(start, end, md int) bool {
	if start <= end {
		return md >= start && md <= end
	}
	return md >= start || md <= end
}

// EvaluateAccessWindowsAt reports whether now falls inside any window. An
// empty list allows everything; a nil location falls back to UTC.
func EvaluateAccessWindowsAt(windows []AccessWindow, loc *time.Location, now time.Time) (bool, string) {
	if len(windows) == 0 {
		return true, ""
	}
	if loc == nil {
		loc = time.UTC
	}
	local := now.In(loc)
	for _, w := range windows {
		if windowAllows(w, local) {
			return true, ""
		}
	}
	return false, "outside the allowed access window"
}

// LocationForPolicy resolves the policy timezone, falling back to the
// server's local zone when unset and to UTC when an unknown zone somehow
// slipped past validation.
func LocationForPolicy(name string) *time.Location {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return time.Local
	}
	loc, err := time.LoadLocation(trimmed)
	if err != nil {
		return time.UTC
	}
	return loc
}

func windowAllows(w AccessWindow, local time.Time) bool {
	start, err := parseClock(w.Start)
	if err != nil {
		return false
	}
	end, err := parseClock(w.End)
	if err != nil {
		return false
	}
	today := int(local.Weekday())
	yesterday := (today + 6) % 7
	cur := local.Hour()*60 + local.Minute()

	switch {
	case start == end:
		// Same start and end reads as "all day" for the selected days.
		return dayMatches(w.Days, today)
	case start < end:
		return dayMatches(w.Days, today) && cur >= start && cur < end
	default:
		// Wraps past midnight.
		if dayMatches(w.Days, today) && cur >= start {
			return true
		}
		return dayMatches(w.Days, yesterday) && cur < end
	}
}

func dayMatches(days []int, day int) bool {
	if len(days) == 0 {
		return true
	}
	for _, d := range days {
		if d == day {
			return true
		}
	}
	return false
}

// parseClock turns "HH:MM" into minutes since midnight.
func parseClock(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	t, err := time.Parse("15:04", trimmed)
	if err != nil {
		return 0, err
	}
	return t.Hour()*60 + t.Minute(), nil
}
