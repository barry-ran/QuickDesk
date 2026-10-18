package holiday

import (
	"fmt"
	"time"

	"github.com/6tail/lunar-go/calendar"
)

// LunarToSolar converts a lunar (农历) date to its solar date in the same
// lunar year. ok is false when the combination does not exist - a lunar month
// is 29 or 30 days, so "the 30th of the 12th month" is not a real date in
// every year - or when the inputs are out of range.
//
// Month 1..12 is a normal month; a negative month would mean a leap month, but
// policies never need that and it is rejected here.
func LunarToSolar(lunarYear, lunarMonth, lunarDay int) (solar time.Time, ok bool) {
	if lunarYear < 1900 || lunarYear > 2100 || lunarMonth < 1 || lunarMonth > 12 || lunarDay < 1 || lunarDay > 30 {
		return time.Time{}, false
	}
	// The upstream library panics on impossible dates; recover rather than
	// let a bad policy take the server down.
	defer func() {
		if r := recover(); r != nil {
			solar, ok = time.Time{}, false
		}
	}()
	s := calendar.NewLunarFromYmd(lunarYear, lunarMonth, lunarDay).GetSolar()
	return time.Date(s.GetYear(), time.Month(s.GetMonth()), s.GetDay(), 0, 0, 0, 0, time.UTC), true
}

// LunarToSolarClamped is LunarToSolar with a fallback for the one genuinely
// variable case: 除夕 (New Year's Eve) is the 30th of the 12th lunar month in
// years where that month has 30 days and the 29th when it has 29. Asking for
// the 30th therefore falls back to the 29th so a "除夕" entry keeps working
// every year instead of silently skipping the 29-day years.
func LunarToSolarClamped(lunarYear, lunarMonth, lunarDay int) (time.Time, bool) {
	if solar, ok := LunarToSolar(lunarYear, lunarMonth, lunarDay); ok {
		return solar, true
	}
	if lunarDay > 1 {
		return LunarToSolar(lunarYear, lunarMonth, lunarDay-1)
	}
	return time.Time{}, false
}

// FormatLunar is a small helper for error messages.
func FormatLunar(month, day int) string {
	return fmt.Sprintf("%02d-%02d", month, day)
}
