package service

import (
	"errors"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"
)

func mustLoad(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("load %s: %v", name, err)
	}
	return loc
}

func TestValidateAccessWindows(t *testing.T) {
	tests := []struct {
		name    string
		windows []AccessWindow
		wantErr bool
	}{
		{"empty means any time", nil, false},
		{"weekday office hours", []AccessWindow{{Days: []int{1, 2, 3, 4, 5}, Start: "09:00", End: "18:00"}}, false},
		{"every day when days omitted", []AccessWindow{{Start: "00:00", End: "23:59"}}, false},
		{"wrapping window", []AccessWindow{{Days: []int{5}, Start: "22:00", End: "06:00"}}, false},
		{"day out of range", []AccessWindow{{Days: []int{7}, Start: "09:00", End: "18:00"}}, true},
		{"negative day", []AccessWindow{{Days: []int{-1}, Start: "09:00", End: "18:00"}}, true},
		{"bad start", []AccessWindow{{Start: "9am", End: "18:00"}}, true},
		{"hour out of range", []AccessWindow{{Start: "25:00", End: "18:00"}}, true},
		{"missing end", []AccessWindow{{Start: "09:00"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAccessWindows(tt.windows)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateAccessWindows(%v) err = %v, wantErr %v", tt.windows, err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, ErrInvalidAccessWindow) {
				t.Fatalf("error should wrap ErrInvalidAccessWindow, got %v", err)
			}
		})
	}
}

func TestValidateTimezone(t *testing.T) {
	for _, ok := range []string{"", "  ", "Asia/Shanghai", "UTC", "America/New_York"} {
		if err := ValidateTimezone(ok); err != nil {
			t.Errorf("ValidateTimezone(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{"Mars/Phobos", "GMT+8", "Shanghai"} {
		if err := ValidateTimezone(bad); !errors.Is(err, ErrInvalidTimezone) {
			t.Errorf("ValidateTimezone(%q) = %v, want ErrInvalidTimezone", bad, err)
		}
	}
}

func TestEvaluateAccessWindowsAt(t *testing.T) {
	shanghai := mustLoad(t, "Asia/Shanghai")
	officeHours := []AccessWindow{{Days: []int{1, 2, 3, 4, 5}, Start: "09:00", End: "18:00"}}

	at := func(loc *time.Location, year int, month time.Month, day, hour, min int) time.Time {
		return time.Date(year, month, day, hour, min, 0, 0, loc)
	}

	tests := []struct {
		name    string
		windows []AccessWindow
		now     time.Time
		want    bool
	}{
		{"no windows allows anything", nil, at(shanghai, 2026, 10, 19, 3, 0), true},
		{"inside weekday window", officeHours, at(shanghai, 2026, 10, 19, 10, 0), true},
		{"before window", officeHours, at(shanghai, 2026, 10, 19, 8, 59), false},
		{"end is exclusive", officeHours, at(shanghai, 2026, 10, 19, 18, 0), false},
		{"weekend is excluded", officeHours, at(shanghai, 2026, 10, 17, 10, 0), false},
		{"same instant in another zone", officeHours, at(time.UTC, 2026, 10, 19, 3, 0), true}, // 11:00 上海
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, reason := EvaluateAccessWindowsAt(tt.windows, shanghai, tt.now)
			if allowed != tt.want {
				t.Fatalf("allowed = %v (reason %q), want %v", allowed, reason, tt.want)
			}
			if !allowed && reason == "" {
				t.Fatal("denied window should explain itself")
			}
		})
	}
}

// A window that crosses midnight is anchored on the day it starts: Friday
// 22:00-06:00 covers Friday night *and* Saturday morning.
func TestEvaluateAccessWindowsAtWrapsMidnight(t *testing.T) {
	loc := mustLoad(t, "Asia/Shanghai")
	fridayNight := []AccessWindow{{Days: []int{5}, Start: "22:00", End: "06:00"}}

	cases := []struct {
		when time.Time
		want bool
	}{
		{time.Date(2026, 10, 23, 22, 30, 0, 0, loc), true},  // Friday 22:30
		{time.Date(2026, 10, 24, 5, 30, 0, 0, loc), true},   // Saturday 05:30
		{time.Date(2026, 10, 24, 7, 0, 0, 0, loc), false},   // Saturday 07:00
		{time.Date(2026, 10, 24, 22, 30, 0, 0, loc), false}, // Saturday night is a different start day
		{time.Date(2026, 10, 23, 21, 59, 0, 0, loc), false},
	}
	for _, c := range cases {
		allowed, _ := EvaluateAccessWindowsAt(fridayNight, loc, c.when)
		if allowed != c.want {
			t.Errorf("window at %s = %v, want %v", c.when.Format(time.RFC3339), allowed, c.want)
		}
	}
}

func TestEvaluateDeviceAccessPolicyAtHonoursWindows(t *testing.T) {
	loc := mustLoad(t, "Asia/Shanghai")
	policy := &models.DeviceAccessPolicy{
		AllowIPs:       `["127.0.0.0/8"]`,
		AllowedWindows: `[{"days":[1],"start":"09:00","end":"18:00"}]`,
		Timezone:       "Asia/Shanghai",
	}

	mondayInside := time.Date(2026, 10, 19, 10, 0, 0, 0, loc)
	if allowed, reason := EvaluateDeviceAccessPolicyAt(policy, "127.0.0.1", mondayInside); !allowed {
		t.Fatalf("inside window should be allowed, got %q", reason)
	}

	mondayOutside := time.Date(2026, 10, 19, 20, 0, 0, 0, loc)
	allowed, reason := EvaluateDeviceAccessPolicyAt(policy, "127.0.0.1", mondayOutside)
	if allowed {
		t.Fatal("outside window must be denied")
	}
	if reason != "outside the allowed access window" {
		t.Fatalf("unexpected reason %q", reason)
	}

	// The IP rule still wins when both would deny: it is checked first.
	if _, reason := EvaluateDeviceAccessPolicyAt(policy, "10.0.0.1", mondayInside); reason != "client IP is not in the device allow list" {
		t.Fatalf("expected the IP reason first, got %q", reason)
	}
}

func TestValidateAccessExceptions(t *testing.T) {
	tests := []struct {
		name    string
		input   []AccessExceptionDate
		wantErr bool
	}{
		{"empty means no exceptions", nil, false},
		{"deny a holiday", []AccessExceptionDate{{Date: "2026-02-17", Mode: "deny"}}, false},
		{"allow a one-off work day", []AccessExceptionDate{{Date: "2026-10-17", Mode: "allow"}}, false},
		{"mode is case/space tolerant", []AccessExceptionDate{{Date: " 2026-10-17 ", Mode: " ALLOW "}}, false},
		{"two different dates", []AccessExceptionDate{{Date: "2026-02-17", Mode: "deny"}, {Date: "2026-02-18", Mode: "deny"}}, false},
		{"bad date", []AccessExceptionDate{{Date: "2026-13-40", Mode: "deny"}}, true},
		{"date with time", []AccessExceptionDate{{Date: "2026-10-17T00:00:00Z", Mode: "deny"}}, true},
		{"unknown mode", []AccessExceptionDate{{Date: "2026-10-17", Mode: "block"}}, true},
		{"empty mode", []AccessExceptionDate{{Date: "2026-10-17"}}, true},
		{"same date twice is ambiguous", []AccessExceptionDate{{Date: "2026-10-17", Mode: "allow"}, {Date: "2026-10-17", Mode: "deny"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAccessExceptions(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateAccessExceptions(%v) err = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !errors.Is(err, ErrInvalidAccessException) {
				t.Fatalf("error should wrap ErrInvalidAccessException, got %v", err)
			}
		})
	}
}

func TestAccessExceptionMatch(t *testing.T) {
	shanghai := mustLoad(t, "Asia/Shanghai")
	exceptions := []AccessExceptionDate{
		{Date: "2026-10-19", Mode: "deny"},
		{Date: "2026-10-17", Mode: "allow"},
	}

	// 2026-10-19 10:00 Shanghai is inside the deny date.
	if matched, allowed, reason := AccessExceptionMatch(exceptions, shanghai, time.Date(2026, 10, 19, 10, 0, 0, 0, shanghai)); !matched || allowed || reason == "" {
		t.Fatalf("deny date should match and deny, got matched=%v allowed=%v reason=%q", matched, allowed, reason)
	}
	// Same instant expressed in UTC must still land on the same local date.
	utc := time.Date(2026, 10, 19, 2, 0, 0, 0, time.UTC)
	if matched, allowed, _ := AccessExceptionMatch(exceptions, shanghai, utc); !matched || allowed {
		t.Fatalf("deny date must be evaluated in the policy timezone, got matched=%v allowed=%v", matched, allowed)
	}
	// An allow date matches and permits.
	if matched, allowed, _ := AccessExceptionMatch(exceptions, shanghai, time.Date(2026, 10, 17, 9, 0, 0, 0, shanghai)); !matched || !allowed {
		t.Fatal("allow date should match and permit")
	}
	// A date with no exception does not match, so callers fall back to windows.
	if matched, _, _ := AccessExceptionMatch(exceptions, shanghai, time.Date(2026, 10, 18, 9, 0, 0, 0, shanghai)); matched {
		t.Fatal("a date with no exception must not match")
	}
	// The local-date boundary matters: 2026-10-16 20:00 UTC is already
	// 2026-10-17 04:00 in Shanghai, so the allow date applies.
	if matched, allowed, _ := AccessExceptionMatch(exceptions, shanghai, time.Date(2026, 10, 16, 20, 0, 0, 0, time.UTC)); !matched || !allowed {
		t.Fatal("exception must be anchored to the local calendar date, not UTC")
	}
}

func TestEvaluateDeviceAccessPolicyWithExceptionDates(t *testing.T) {
	shanghai := mustLoad(t, "Asia/Shanghai")
	window := `[{"days":[1],"start":"09:00","end":"18:00"}]`

	// Monday-only windows, but Monday 2026-10-19 is a company holiday.
	holiday := &models.DeviceAccessPolicy{
		AllowedWindows: window,
		Timezone:       "Asia/Shanghai",
		ExceptionDates: `[{"date":"2026-10-19","mode":"deny"}]`,
	}
	mondayInside := time.Date(2026, 10, 19, 10, 0, 0, 0, shanghai)
	allowed, reason := EvaluateDeviceAccessPolicyAt(holiday, "127.0.0.1", mondayInside)
	if allowed {
		t.Fatal("a deny exception must block the day even inside the window")
	}
	if reason != "today is blocked by an access exception date" {
		t.Fatalf("unexpected reason %q", reason)
	}

	// A different Monday with no exception still follows the window.
	otherMonday := time.Date(2026, 10, 26, 10, 0, 0, 0, shanghai)
	if allowed, reason := EvaluateDeviceAccessPolicyAt(holiday, "127.0.0.1", otherMonday); !allowed {
		t.Fatalf("a Monday without an exception should stay allowed, got %q", reason)
	}

	// An allow exception on a Saturday overrides the Monday-only windows.
	workDay := &models.DeviceAccessPolicy{
		AllowedWindows: window,
		Timezone:       "Asia/Shanghai",
		ExceptionDates: `[{"date":"2026-10-17","mode":"allow"}]`,
	}
	saturday := time.Date(2026, 10, 17, 10, 0, 0, 0, shanghai)
	if allowed, reason := EvaluateDeviceAccessPolicyAt(workDay, "127.0.0.1", saturday); !allowed {
		t.Fatalf("an allow exception must override the windows, got %q", reason)
	}
	// ...but the next Saturday with no exception is denied again.
	if _, reason := EvaluateDeviceAccessPolicyAt(workDay, "127.0.0.1", time.Date(2026, 10, 24, 10, 0, 0, 0, shanghai)); reason != "outside the allowed access window" {
		t.Fatalf("the following Saturday should fall back to the windows, got %q", reason)
	}

	// The stronger controls still win over an allow exception.
	locked := &models.DeviceAccessPolicy{
		RemoteDisabled: true,
		Timezone:       "Asia/Shanghai",
		ExceptionDates: `[{"date":"2026-10-17","mode":"allow"}]`,
	}
	if _, reason := EvaluateDeviceAccessPolicyAt(locked, "127.0.0.1", saturday); reason != "remote access is disabled for this device" {
		t.Fatalf("remote_disabled must beat an allow exception, got %q", reason)
	}
	ipLocked := &models.DeviceAccessPolicy{
		AllowIPs:       `["10.211.55.0/24"]`,
		Timezone:       "Asia/Shanghai",
		ExceptionDates: `[{"date":"2026-10-17","mode":"allow"}]`,
	}
	if _, reason := EvaluateDeviceAccessPolicyAt(ipLocked, "192.168.1.9", saturday); reason != "client IP is not in the device allow list" {
		t.Fatalf("the IP allow list must beat an allow exception, got %q", reason)
	}
}

func TestValidateAccessExceptionRangesAndRepeat(t *testing.T) {
	tests := []struct {
		name    string
		input   []AccessExceptionDate
		wantErr bool
	}{
		{"single day still works", []AccessExceptionDate{{Date: "2026-10-17", Mode: "deny"}}, false},
		{"inclusive range", []AccessExceptionDate{{Date: "2026-02-15", EndDate: "2026-02-21", Mode: "deny"}}, false},
		{"range end equals start", []AccessExceptionDate{{Date: "2026-02-15", EndDate: "2026-02-15", Mode: "deny"}}, false},
		{"yearly single day", []AccessExceptionDate{{Date: "2026-02-17", Mode: "deny", Repeat: "yearly"}}, false},
		{"yearly range may wrap the new year", []AccessExceptionDate{{Date: "2026-12-30", EndDate: "2026-01-02", Mode: "deny", Repeat: "yearly"}}, false},
		{"non-yearly range must be ordered", []AccessExceptionDate{{Date: "2026-10-07", EndDate: "2026-10-01", Mode: "deny"}}, true},
		{"bad end date", []AccessExceptionDate{{Date: "2026-10-01", EndDate: "2026-10-40", Mode: "deny"}}, true},
		{"unknown repeat", []AccessExceptionDate{{Date: "2026-10-01", Mode: "deny", Repeat: "monthly"}}, true},
		{"same-mode overlap is harmless", []AccessExceptionDate{
			{Date: "2026-10-01", EndDate: "2026-10-07", Mode: "deny"},
			{Date: "2026-10-05", Mode: "deny"},
		}, false},
		{"allow/deny overlap is rejected", []AccessExceptionDate{
			{Date: "2026-10-01", EndDate: "2026-10-07", Mode: "deny"},
			{Date: "2026-10-05", Mode: "allow"},
		}, true},
		{"disjoint allow/deny is fine", []AccessExceptionDate{
			{Date: "2026-10-01", EndDate: "2026-10-07", Mode: "deny"},
			{Date: "2026-10-10", Mode: "allow"},
		}, false},
		{"yearly deny overlapping a dated allow is rejected", []AccessExceptionDate{
			{Date: "2026-02-15", EndDate: "2026-02-21", Mode: "deny", Repeat: "yearly"},
			{Date: "2027-02-18", Mode: "allow"},
		}, true},
		{"two yearly ranges that do not touch are fine", []AccessExceptionDate{
			{Date: "2026-02-01", EndDate: "2026-02-07", Mode: "deny", Repeat: "yearly"},
			{Date: "2026-03-01", EndDate: "2026-03-07", Mode: "allow", Repeat: "yearly"},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAccessExceptions(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateAccessExceptions(%v) err = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestAccessExceptionMatchRangesAndRepeat(t *testing.T) {
	shanghai := mustLoad(t, "Asia/Shanghai")
	at := func(y int, m time.Month, d int) time.Time {
		return time.Date(y, m, d, 9, 0, 0, 0, shanghai)
	}

	// An inclusive range covers the middle days but nothing outside it.
	festival := []AccessExceptionDate{{Date: "2026-02-15", EndDate: "2026-02-21", Mode: "deny"}}
	for _, d := range []int{15, 18, 21} {
		if matched, allowed, _ := AccessExceptionMatch(festival, shanghai, at(2026, time.February, d)); !matched || allowed {
			t.Fatalf("2026-02-%02d should be denied by the range", d)
		}
	}
	for _, d := range []int{14, 22} {
		if matched, _, _ := AccessExceptionMatch(festival, shanghai, at(2026, time.February, d)); matched {
			t.Fatalf("2026-02-%02d must be outside the range", d)
		}
	}

	// The same range, recurring yearly, blocks the same days in another year.
	annual := []AccessExceptionDate{{Date: "2026-02-15", EndDate: "2026-02-21", Mode: "deny", Repeat: "yearly"}}
	if matched, allowed, _ := AccessExceptionMatch(annual, shanghai, at(2030, time.February, 18)); !matched || allowed {
		t.Fatal("a yearly exception must fire in later years")
	}
	if matched, _, _ := AccessExceptionMatch(annual, shanghai, at(2030, time.February, 25)); matched {
		t.Fatal("a yearly exception must not fire outside its month-day range")
	}

	// A yearly range that wraps the new year covers late December and early
	// January, across two different calendar years.
	newYear := []AccessExceptionDate{{Date: "2026-12-30", EndDate: "2026-01-02", Mode: "allow", Repeat: "yearly"}}
	for _, when := range []time.Time{at(2026, time.December, 31), at(2027, time.January, 1)} {
		if matched, allowed, _ := AccessExceptionMatch(newYear, shanghai, when); !matched || !allowed {
			t.Fatalf("%s should be covered by the wrapping yearly range", when.Format("2006-01-02"))
		}
	}
	if matched, _, _ := AccessExceptionMatch(newYear, shanghai, at(2027, time.January, 5)); matched {
		t.Fatal("a wrapping yearly range must not cover days after its end")
	}
}

func TestEvaluateDeviceAccessPolicyWithYearlyExceptionRange(t *testing.T) {
	shanghai := mustLoad(t, "Asia/Shanghai")
	// Weekdays 09:00-18:00, but every year the Spring Festival week is closed.
	policy := &models.DeviceAccessPolicy{
		AllowedWindows: `[{"days":[1,2,3,4,5],"start":"09:00","end":"18:00"}]`,
		Timezone:       "Asia/Shanghai",
		ExceptionDates: `[{"date":"2026-02-15","end_date":"2026-02-21","mode":"deny","repeat":"yearly"}]`,
	}
	// 2026-02-18 is a Wednesday inside the window, but the festival blocks it.
	wednesday := time.Date(2026, time.February, 18, 10, 0, 0, 0, shanghai)
	if allowed, reason := EvaluateDeviceAccessPolicyAt(policy, "127.0.0.1", wednesday); allowed {
		t.Fatal("the festival range must block a weekday inside the window")
	} else if reason != "today is blocked by an access exception date" {
		t.Fatalf("unexpected reason %q", reason)
	}
	// The next year's festival week is blocked too.
	nextYear := time.Date(2027, time.February, 17, 10, 0, 0, 0, shanghai)
	if allowed, _ := EvaluateDeviceAccessPolicyAt(policy, "127.0.0.1", nextYear); allowed {
		t.Fatal("a yearly exception must recur")
	}
	// A weekday in March still follows the weekly window.
	march := time.Date(2026, time.March, 4, 10, 0, 0, 0, shanghai)
	if allowed, reason := EvaluateDeviceAccessPolicyAt(policy, "127.0.0.1", march); !allowed {
		t.Fatalf("a March weekday should stay allowed, got %q", reason)
	}
}

// Named statutory holidays let a policy say "放假期间别连我机器" without
// re-listing the dates every year - the calendar data carries the span.
func TestAccessExceptionMatchHolidayCalendar(t *testing.T) {
	shanghai := mustLoad(t, "Asia/Shanghai")
	denyFestival := []AccessExceptionDate{{Holiday: "春节", Mode: "deny"}}

	// The Spring Festival break, 2026-02-15 .. 02-23.
	for _, ymd := range []string{"2026-02-15", "2026-02-17", "2026-02-23"} {
		d := mustDate(t, shanghai, ymd)
		matched, allowed, reason := AccessExceptionMatch(denyFestival, shanghai, d)
		if !matched || allowed || reason == "" {
			t.Fatalf("%s should be blocked by the festival, got matched=%v allowed=%v reason=%q",
				ymd, matched, allowed, reason)
		}
	}
	// A day just outside the break must not be swept up by the name match.
	if matched, _, _ := AccessExceptionMatch(denyFestival, shanghai, mustDate(t, shanghai, "2026-02-24")); matched {
		t.Fatal("2026-02-24 is a normal working day and must not match the festival")
	}
	// The make-up workdays are a separate selector, so a plain holiday entry
	// does not silently cover them.
	if matched, _, _ := AccessExceptionMatch(denyFestival, shanghai, mustDate(t, shanghai, "2026-02-14")); matched {
		t.Fatal("a '-off' holiday entry must not cover a make-up workday")
	}

	// The 调休 workday can be allowed explicitly, which is the "周六但要开工"
	// case that a weekly window would otherwise block.
	allowMakeup := []AccessExceptionDate{
		{Holiday: "春节", HolidayDay: AccessExceptionHolidayDayWork, Mode: "allow"},
	}
	if matched, allowed, _ := AccessExceptionMatch(allowMakeup, shanghai, mustDate(t, shanghai, "2026-02-14")); !matched || !allowed {
		t.Fatalf("the 春节 make-up workday should be allowed, got matched=%v allowed=%v", matched, allowed)
	}
	if matched, _, _ := AccessExceptionMatch(allowMakeup, shanghai, mustDate(t, shanghai, "2026-02-17")); matched {
		t.Fatal("a '-work' entry must not cover the days off")
	}
}

// A lunar entry resolves to a different solar date every year, so 春节/端午/中秋
// never need an annual edit. The multi-year lookup also has to handle dates
// that belong to the previous lunar year (everything before 春节).
func TestAccessExceptionMatchLunar(t *testing.T) {
	shanghai := mustLoad(t, "Asia/Shanghai")

	dragonBoat := []AccessExceptionDate{{
		Repeat: AccessExceptionRepeatLunar, LunarMonth: 5, LunarDay: 5, Mode: "deny",
	}}
	for _, ymd := range []string{"2026-06-19", "2027-06-09", "2028-05-28"} {
		d := mustDate(t, shanghai, ymd)
		if matched, allowed, _ := AccessExceptionMatch(dragonBoat, shanghai, d); !matched || allowed {
			t.Fatalf("端午 %s should be denied, got matched=%v allowed=%v", ymd, matched, allowed)
		}
	}
	if matched, _, _ := AccessExceptionMatch(dragonBoat, shanghai, mustDate(t, shanghai, "2026-06-20")); matched {
		t.Fatal("the day after 端午 must not match")
	}

	// 除夕: the 29th of the 12th lunar month, which lands in solar February of
	// the *following* year - exactly the case a naive day.Year() lookup breaks.
	newYearsEve := []AccessExceptionDate{{
		Repeat: AccessExceptionRepeatLunar, LunarMonth: 12, LunarDay: 29, Mode: "deny",
	}}
	// Solar 2027 is lunar 2026 for the 除夕 date, so the solar year and the
	// lunar year differ here; 2026-02-16 is the same day of the previous lunar
	// year, and both must match.
	for _, ymd := range []string{"2027-02-05", "2026-02-16"} {
		if matched, allowed, _ := AccessExceptionMatch(newYearsEve, shanghai, mustDate(t, shanghai, ymd)); !matched || allowed {
			t.Fatalf("除夕 %s should be denied, got matched=%v allowed=%v", ymd, matched, allowed)
		}
	}
	// The days either side of 除夕 are not the 29th of the 12th month.
	for _, ymd := range []string{"2027-02-04", "2026-02-15"} {
		if matched, _, _ := AccessExceptionMatch(newYearsEve, shanghai, mustDate(t, shanghai, ymd)); matched {
			t.Fatalf("%s must not match the 除夕 entry", ymd)
		}
	}
}

func TestValidateAccessExceptionsHolidayAndLunar(t *testing.T) {
	tests := []struct {
		name    string
		input   []AccessExceptionDate
		wantErr bool
	}{
		{"named holiday", []AccessExceptionDate{{Holiday: "春节", Mode: "deny"}}, false},
		{"named holiday workdays", []AccessExceptionDate{{Holiday: "春节", HolidayDay: "work", Mode: "allow"}}, false},
		{"holiday_day off is explicit", []AccessExceptionDate{{Holiday: "国庆节", HolidayDay: "off", Mode: "deny"}}, false},
		{"lunar festival", []AccessExceptionDate{{Repeat: "lunar", LunarMonth: 8, LunarDay: 15, Mode: "deny"}}, false},
		{"unknown holiday", []AccessExceptionDate{{Holiday: "元宵节", Mode: "deny"}}, true},
		{"holiday plus date", []AccessExceptionDate{{Holiday: "春节", Date: "2026-02-17", Mode: "deny"}}, true},
		{"holiday plus repeat", []AccessExceptionDate{{Holiday: "春节", Repeat: "yearly", Mode: "deny"}}, true},
		{"bad holiday_day", []AccessExceptionDate{{Holiday: "春节", HolidayDay: "sometimes", Mode: "deny"}}, true},
		{"lunar without repeat", []AccessExceptionDate{{LunarMonth: 5, LunarDay: 5, Mode: "deny"}}, true},
		{"lunar repeat without day", []AccessExceptionDate{{Repeat: "lunar", LunarMonth: 5, Mode: "deny"}}, true},
		{"lunar repeat with bad month", []AccessExceptionDate{{Repeat: "lunar", LunarMonth: 13, LunarDay: 1, Mode: "deny"}}, true},
		{"lunar repeat with date", []AccessExceptionDate{{Repeat: "lunar", Date: "2026-06-19", LunarMonth: 5, LunarDay: 5, Mode: "deny"}}, true},
		{"holiday_day on a date entry", []AccessExceptionDate{{Date: "2026-06-19", HolidayDay: "work", Mode: "deny"}}, true},
		// Overlap is detected against the resolved calendar dates, so a fixed
		// allow inside the blocked festival is caught.
		{"allow inside holiday deny", []AccessExceptionDate{
			{Holiday: "春节", Mode: "deny"},
			{Date: "2026-02-17", Mode: "allow"},
		}, true},
		{"allow on a make-up workday is fine", []AccessExceptionDate{
			{Holiday: "春节", Mode: "deny"},
			{Holiday: "春节", HolidayDay: "work", Mode: "allow"},
		}, false},
		{"lunar deny overlapping a fixed allow", []AccessExceptionDate{
			{Repeat: "lunar", LunarMonth: 5, LunarDay: 5, Mode: "deny"},
			{Date: "2026-06-19", Mode: "allow"},
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAccessExceptions(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateAccessExceptions(%v) err = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !errors.Is(err, ErrInvalidAccessException) {
				t.Fatalf("error should wrap ErrInvalidAccessException, got %v", err)
			}
		})
	}
}

// A policy can combine a weekly window with a named holiday, which is the
// intended real deployment shape: Mondays 09:00-18:00, closed for 春节.
func TestEvaluateDeviceAccessPolicyAtHonoursHolidayCalendar(t *testing.T) {
	shanghai := mustLoad(t, "Asia/Shanghai")
	policy := &models.DeviceAccessPolicy{
		AllowedWindows: `[{"days":[1,2,3,4,5,6],"start":"09:00","end":"18:00"}]`,
		Timezone:       "Asia/Shanghai",
		ExceptionDates: `[{"holiday":"春节","mode":"deny"}]`,
	}

	// 2026-02-18 is a Wednesday inside the weekly window, blocked by 春节.
	inside := time.Date(2026, time.February, 18, 10, 0, 0, 0, shanghai)
	allowed, reason := EvaluateDeviceAccessPolicyAt(policy, "127.0.0.1", inside)
	if allowed {
		t.Fatal("the Spring Festival must block a weekday inside the window")
	}
	if reason != "today is blocked by an access exception date" {
		t.Fatalf("unexpected reason %q", reason)
	}

	// 2026-03-04 is a normal Wednesday, so the weekly window applies again.
	normal := time.Date(2026, time.March, 4, 10, 0, 0, 0, shanghai)
	if allowed, reason := EvaluateDeviceAccessPolicyAt(policy, "127.0.0.1", normal); !allowed {
		t.Fatalf("a normal weekday should be allowed, got %q", reason)
	}
}

func mustDate(t *testing.T, loc *time.Location, ymd string) time.Time {
	t.Helper()
	parsed, err := time.ParseInLocation("2006-01-02", ymd, loc)
	if err != nil {
		t.Fatalf("parse %s: %v", ymd, err)
	}
	return parsed
}
