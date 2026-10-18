package holiday

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// MaxCalendarBytes caps one downloaded year file. The real files are a few
// kilobytes; the cap is there so a misconfigured URL pointing at something
// large cannot fill the disk.
const MaxCalendarBytes = 1 << 20

// Fetcher keeps the override directory fresh by downloading the published
// holiday-cn year files.
//
// The alternative it replaces is an operator noticing that next year's State
// Council notice is out and hand-copying a JSON file onto the server. The
// fetcher keeps that path working - it writes the same files the loader reads -
// and adds the part a human forgets.
type Fetcher struct {
	// URL is either a template containing `{year}`, or a base URL to which
	// `/<year>.json` is appended.
	URL string
	// Dir is the override directory the loader reads. Required: a fetch with
	// nowhere to put the file is a configuration mistake, not a no-op.
	Dir string
	// Years decides which years to keep fresh. Nil means the current year and
	// the next one, which is the window an operator actually needs: the notice
	// for next year is published during this one.
	Years func(now time.Time) []int
	// Client is the HTTP client. Nil means a 15-second timeout.
	Client *http.Client
	// Now is a clock seam for tests. Nil means time.Now.
	Now func() time.Time
}

// NewFetcher builds a fetcher for one URL and cache directory.
func NewFetcher(url, dir string) *Fetcher {
	return &Fetcher{URL: strings.TrimSpace(url), Dir: strings.TrimSpace(dir)}
}

func (f *Fetcher) now() time.Time {
	if f.Now != nil {
		return f.Now()
	}
	return time.Now()
}

func (f *Fetcher) client() *http.Client {
	if f.Client != nil {
		return f.Client
	}
	return &http.Client{Timeout: 15 * time.Second}
}

// yearURL builds the URL for one year: `{year}` is substituted when present,
// otherwise the year is appended as holiday-cn names its files.
func (f *Fetcher) yearURL(year int) string {
	y := strconv.Itoa(year)
	if strings.Contains(f.URL, "{year}") {
		return strings.ReplaceAll(f.URL, "{year}", y)
	}
	return strings.TrimRight(f.URL, "/") + "/" + y + ".json"
}

// yearsToFetch is the window the fetcher maintains.
func (f *Fetcher) yearsToFetch(now time.Time) []int {
	if f.Years != nil {
		return f.Years(now)
	}
	return []int{now.Year(), now.Year() + 1}
}

// FetchYear downloads one year, and reports whether the cached file changed.
//
// The bytes are validated before anything is written: they have to parse into a
// usable calendar that actually contains the requested year. A redirect to an
// HTML error page, a truncated file or a misconfigured URL that answers every
// request with the same year all fail here, and the previous file stays
// untouched - a bad fetch must never be able to replace a good calendar.
func (f *Fetcher) FetchYear(ctx context.Context, year int) (bool, error) {
	if f.Dir == "" {
		return false, errors.New("no cache directory configured")
	}
	url := f.yearURL(year)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	resp, err := f.client().Do(req)
	if err != nil {
		return false, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("fetch %s: unexpected status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxCalendarBytes+1))
	if err != nil {
		return false, fmt.Errorf("fetch %s: %w", url, err)
	}
	if len(body) > MaxCalendarBytes {
		return false, fmt.Errorf("fetch %s: file exceeds %d bytes", url, MaxCalendarBytes)
	}

	probe := New()
	if err := probe.LoadFile(bytes.NewReader(body)); err != nil {
		return false, fmt.Errorf("fetch %s: %w", url, err)
	}
	if !containsYear(probe.Years(), year) {
		return false, fmt.Errorf("fetch %s: no %d dates in the file", url, year)
	}

	path := filepath.Join(f.Dir, "cn-"+strconv.Itoa(year)+".json")
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, body) {
		return false, nil
	}
	if err := os.MkdirAll(f.Dir, 0o750); err != nil {
		return false, err
	}
	// Write next to the target and rename, so a reader never sees a partial
	// file and a crash mid-write cannot leave a broken calendar behind.
	tmp := path + ".part"
	if err := os.WriteFile(tmp, body, 0o640); err != nil {
		return false, err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return false, err
	}
	return true, nil
}

func containsYear(years []int, want int) bool {
	for _, y := range years {
		if y == want {
			return true
		}
	}
	return false
}

// Refresh fetches every year in the window and returns the ones whose cached
// file changed. One year failing does not stop the others: a missing notice for
// next year must not stop this year's file from being repaired.
func (f *Fetcher) Refresh(ctx context.Context) ([]int, error) {
	var changed []int
	var errs []error
	for _, year := range f.yearsToFetch(f.now()) {
		updated, err := f.FetchYear(ctx, year)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if updated {
			changed = append(changed, year)
		}
	}
	return changed, errors.Join(errs...)
}

// Run refreshes on every tick until ctx is cancelled. `onChange` receives the
// years whose files changed so the caller can rebuild the effective calendar;
// it runs even when another year failed, because the successful ones are
// already on disk.
func (f *Fetcher) Run(ctx context.Context, interval time.Duration, onChange func([]int)) {
	if f == nil || interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			changed, err := f.Refresh(ctx)
			if err != nil {
				log.Printf("[holiday] calendar refresh: %v", err)
			}
			if len(changed) > 0 && onChange != nil {
				onChange(changed)
			}
		}
	}
}
