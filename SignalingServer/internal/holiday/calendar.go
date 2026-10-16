// Package holiday loads the statutory holiday calendars that device access
// policies use to block (or allow) whole holiday periods without hand-listing
// every date each year.
//
// The on-disk format is the one published by the `holiday-cn` dataset, which
// mirrors the State Council's annual notice: a file per year with a list of
// `{name, date, isOffDay}` entries, plus the source URLs in `papers`. Keeping
// that format means an operator can drop next year's file in unchanged instead
// of editing code.
package holiday

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:embed cn-2026.json
var embedded embed.FS

// Day is one date in a holiday calendar. IsOffDay distinguishes an official
// day off from a make-up workday (调休上班), which the State Council arranges
// when a holiday runs across a weekend.
type Day struct {
	Name     string `json:"name"`
	Date     string `json:"date"`
	IsOffDay bool   `json:"isOffDay"`
}

// file is the on-disk shape of one calendar year.
type file struct {
	Year   int      `json:"year"`
	Papers []string `json:"papers"`
	Days   []Day    `json:"days"`
}

// Calendar is the union of every loaded year, indexed by date.
type Calendar struct {
	days  map[string]Day
	names map[string]struct{}
	years map[int]struct{}
}

// New returns an empty calendar.
func New() *Calendar {
	return &Calendar{
		days:  make(map[string]Day),
		names: make(map[string]struct{}),
		years: make(map[int]struct{}),
	}
}

// Len reports how many dates the calendar holds.
func (c *Calendar) Len() int {
	if c == nil {
		return 0
	}
	return len(c.days)
}

// Add merges parsed days into the calendar. It rejects a malformed date so a
// broken data file fails loudly at load time rather than silently matching
// nothing at evaluation time.
func (c *Calendar) Add(days []Day) error {
	for i, d := range days {
		date := strings.TrimSpace(d.Date)
		parsed, err := time.Parse("2006-01-02", date)
		if err != nil {
			return fmt.Errorf("holiday entry %d has date %q: %w", i, d.Date, err)
		}
		name := strings.TrimSpace(d.Name)
		if name == "" {
			return fmt.Errorf("holiday entry %d (%s) has no name", i, date)
		}
		c.days[date] = Day{Name: name, Date: date, IsOffDay: d.IsOffDay}
		c.names[name] = struct{}{}
		c.years[parsed.Year()] = struct{}{}
	}
	return nil
}

// LoadFile reads one holiday-cn style year file and merges it into c.
func (c *Calendar) LoadFile(r io.Reader) error {
	var f file
	dec := json.NewDecoder(r)
	// Unknown top-level fields are tolerated on purpose: the upstream
	// holiday-cn files carry `$schema` / `$id`, and operators should be able
	// to drop them in unchanged. Malformed *days* still fail in Add.
	if err := dec.Decode(&f); err != nil {
		return fmt.Errorf("parse holiday calendar: %w", err)
	}
	return c.Add(f.Days)
}

// LoadDir merges every *.json file in dir. Used for the operator override so
// next year's file can be dropped in without a rebuild.
func (c *Calendar) LoadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	loaded := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		f, err := os.Open(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		err = c.LoadFile(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
		loaded++
	}
	if loaded == 0 {
		return fmt.Errorf("no *.json holiday calendars found in %s", dir)
	}
	return nil
}

// Lookup returns the calendar entry for a date, if any.
func (c *Calendar) Lookup(date time.Time) (Day, bool) {
	if c == nil {
		return Day{}, false
	}
	d, ok := c.days[date.Format("2006-01-02")]
	return d, ok
}

// HasName reports whether any loaded year contains this holiday name. Policy
// validation uses it to reject a typo at write time instead of silently
// matching nothing forever.
func (c *Calendar) HasName(name string) bool {
	if c == nil {
		return false
	}
	_, ok := c.names[strings.TrimSpace(name)]
	return ok
}

// Years returns the loaded years in ascending order.
func (c *Calendar) Years() []int {
	if c == nil {
		return nil
	}
	out := make([]int, 0, len(c.years))
	for y := range c.years {
		out = append(out, y)
	}
	sort.Ints(out)
	return out
}

// Dates returns the dates of one named holiday in a year, filtered by whether
// they are days off. Results are sorted.
func (c *Calendar) Dates(year int, name string, offDay bool) []time.Time {
	if c == nil {
		return nil
	}
	want := strings.TrimSpace(name)
	prefix := fmt.Sprintf("%04d-", year)
	var out []time.Time
	for key, d := range c.days {
		if d.IsOffDay != offDay || d.Name != want || !strings.HasPrefix(key, prefix) {
			continue
		}
		if t, err := time.Parse("2006-01-02", key); err == nil {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out
}

var (
	builtinOnce sync.Once
	builtin     *Calendar
)

// Builtin returns the calendar embedded in the binary (currently only the
// 2026 CN arrangement, sourced from the State Council notice). It is parsed
// once and shared: callers must not mutate it.
func Builtin() *Calendar {
	builtinOnce.Do(func() {
		c := New()
		f, err := embedded.Open("cn-2026.json")
		if err != nil {
			panic(fmt.Sprintf("open embedded holiday calendar: %v", err))
		}
		defer f.Close()
		if err := c.LoadFile(f); err != nil {
			panic(fmt.Sprintf("parse embedded holiday calendar: %v", err))
		}
		builtin = c
	})
	return builtin
}

// Load builds the effective calendar: the embedded year(s) plus, when dir is
// set, every year file found there (an operator override for a year the binary
// does not ship yet).
func Load(dir string) (*Calendar, error) {
	c := New()
	if err := c.Add(builtinDays()); err != nil {
		return nil, err
	}
	if strings.TrimSpace(dir) != "" {
		if err := c.LoadDir(dir); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func builtinDays() []Day {
	src := Builtin()
	out := make([]Day, 0, len(src.days))
	for _, d := range src.days {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out
}
