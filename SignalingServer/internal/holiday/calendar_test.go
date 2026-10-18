package holiday

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func day(t *testing.T, ymd string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", ymd)
	if err != nil {
		t.Fatalf("parse %s: %v", ymd, err)
	}
	return parsed
}

// The embedded data is the 2026 arrangement from the State Council notice. The
// assertions below are the load-bearing dates: the Spring Festival span (which
// is what "放假别连我机器" keys off), its make-up workdays, and the three fixed
// holidays that are easy to get wrong.
func TestBuiltinCalendar2026(t *testing.T) {
	c := Builtin()
	if c.Len() != 39 {
		t.Fatalf("builtin entries = %d, want 39", c.Len())
	}
	if years := c.Years(); len(years) != 1 || years[0] != 2026 {
		t.Fatalf("builtin years = %v, want [2026]", years)
	}

	cases := []struct {
		date  string
		name  string
		isOff bool
	}{
		{"2026-01-01", "元旦", true},
		{"2026-01-04", "元旦", false}, // 补班
		{"2026-02-14", "春节", false}, // 补班
		{"2026-02-15", "春节", true},  // 放假第一天
		{"2026-02-17", "春节", true},  // 正月初一
		{"2026-02-23", "春节", true},  // 放假最后一天
		{"2026-02-28", "春节", false}, // 补班
		{"2026-06-19", "端午节", true},
		{"2026-09-25", "中秋节", true},
		{"2026-10-01", "国庆节", true},
		{"2026-10-10", "国庆节", false}, // 补班
	}
	for _, tc := range cases {
		got, ok := c.Lookup(day(t, tc.date))
		if !ok {
			t.Fatalf("%s missing from the calendar", tc.date)
		}
		if got.Name != tc.name || got.IsOffDay != tc.isOff {
			t.Fatalf("%s = {%s off=%v}, want {%s off=%v}",
				tc.date, got.Name, got.IsOffDay, tc.name, tc.isOff)
		}
	}

	// The calendar lists every covered date literally; it must not infer the
	// days around a holiday.
	if _, ok := c.Lookup(day(t, "2026-02-24")); ok {
		t.Fatal("2026-02-24 is the first working day after the break and must not be listed")
	}
	if _, ok := c.Lookup(day(t, "2026-07-01")); ok {
		t.Fatal("an ordinary day must not be in the calendar")
	}
	if !c.HasName("春节") || c.HasName("元宵节") {
		t.Fatal("HasName should accept loaded names only")
	}
}

func TestCalendarDatesGroupsByOffDay(t *testing.T) {
	c := Builtin()

	off := c.Dates(2026, "春节", true)
	if len(off) != 9 {
		t.Fatalf("春节 off days = %d, want 9", len(off))
	}
	if got := off[0].Format("2006-01-02"); got != "2026-02-15" {
		t.Fatalf("first 春节 off day = %s, want 2026-02-15", got)
	}
	if got := off[len(off)-1].Format("2006-01-02"); got != "2026-02-23" {
		t.Fatalf("last 春节 off day = %s, want 2026-02-23", got)
	}

	work := c.Dates(2026, "春节", false)
	if len(work) != 2 {
		t.Fatalf("春节 make-up workdays = %d, want 2", len(work))
	}
	if got := work[0].Format("2006-01-02"); got != "2026-02-14" {
		t.Fatalf("first 春节 workday = %s, want 2026-02-14", got)
	}
}

func TestLoadDirMergesYearsAndRejectsJunk(t *testing.T) {
	dir := t.TempDir()
	next := `{
	  "year": 2027,
	  "days": [
	    {"name": "元旦", "date": "2027-01-01", "isOffDay": true},
	    {"name": "元旦", "date": "2027-01-02", "isOffDay": true}
	  ]
	}`
	if err := os.WriteFile(filepath.Join(dir, "2027.json"), []byte(next), 0o600); err != nil {
		t.Fatalf("write override: %v", err)
	}

	cal, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cal.Len() != 41 { // 39 embedded + 2 override
		t.Fatalf("merged entries = %d, want 41", cal.Len())
	}
	if years := cal.Years(); len(years) != 2 || years[0] != 2026 || years[1] != 2027 {
		t.Fatalf("merged years = %v, want [2026 2027]", years)
	}
	if _, ok := cal.Lookup(day(t, "2027-01-01")); !ok {
		t.Fatal("override year was not merged")
	}

	bad := t.TempDir()
	if err := os.WriteFile(filepath.Join(bad, "broken.json"), []byte(`{"days":[{"name":"x","date":"not-a-date"}]}`), 0o600); err != nil {
		t.Fatalf("write broken file: %v", err)
	}
	if _, err := Load(bad); err == nil {
		t.Fatal("a malformed override file must fail the load, not be ignored")
	}

	empty := t.TempDir()
	if _, err := Load(empty); err == nil || !strings.Contains(err.Error(), "no *.json") {
		t.Fatalf("an empty override dir should be reported, got %v", err)
	}
}

func TestCalendarAddRejectsNamelessEntries(t *testing.T) {
	c := New()
	if err := c.Add([]Day{{Name: "  ", Date: "2026-01-01"}}); err == nil {
		t.Fatal("a nameless entry must be rejected")
	}
}
