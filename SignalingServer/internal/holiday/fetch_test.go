package holiday

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// oneYear is a minimal holiday-cn style file: the loader only needs named,
// parseable days.
func oneYear(year int) string {
	return fmt.Sprintf(
		`{"$schema":"https://example.invalid/schema.json","year":%d,"papers":[],"days":[`+
			`{"name":"New Year","date":"%04d-01-01","isOffDay":true},`+
			`{"name":"New Year","date":"%04d-01-02","isOffDay":false}]}`,
		year, year, year)
}

func TestFetcherWritesTheFileAndSkipsUnchangedContent(t *testing.T) {
	year := time.Now().Year() + 1
	dir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != fmt.Sprintf("/%d.json", year) {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(oneYear(year)))
	}))
	defer server.Close()

	fetcher := NewFetcher(server.URL, dir)
	changed, err := fetcher.FetchYear(context.Background(), year)
	if err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	if !changed {
		t.Fatal("the first fetch reported no change")
	}
	path := filepath.Join(dir, fmt.Sprintf("cn-%d.json", year))
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the fetched file is not on disk: %v", err)
	}

	// The point of the whole exercise: the loader the policies use sees the
	// fetched year without anyone copying a file in.
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !containsYear(loaded.Years(), year) {
		t.Fatalf("loaded years = %v, want them to include %d", loaded.Years(), year)
	}

	changed, err = fetcher.FetchYear(context.Background(), year)
	if err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if changed {
		t.Fatal("an identical file was reported as a change")
	}
}

// A bad fetch must never be able to replace a good calendar.
func TestFetcherKeepsTheCachedFileWhenTheFetchIsBad(t *testing.T) {
	year := time.Now().Year() + 1
	dir := t.TempDir()
	path := filepath.Join(dir, fmt.Sprintf("cn-%d.json", year))
	good := oneYear(year)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(good), 0o640); err != nil {
		t.Fatalf("seed: %v", err)
	}

	bodies := map[string]string{
		"not json":     "<html>maintenance</html>",
		"wrong year":   oneYear(year + 1),
		"empty days":   fmt.Sprintf(`{"year":%d,"days":[]}`, year),
		"broken entry": fmt.Sprintf(`{"year":%d,"days":[{"name":"x","date":"not-a-date"}]}`, year),
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(body))
			}))
			defer server.Close()

			if _, err := NewFetcher(server.URL, dir).FetchYear(context.Background(), year); err == nil {
				t.Fatal("a bad body was accepted")
			}
			onDisk, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read back: %v", err)
			}
			if string(onDisk) != good {
				t.Fatal("the cached calendar was overwritten by a bad fetch")
			}
		})
	}

	t.Run("non-200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "nope", http.StatusInternalServerError)
		}))
		defer server.Close()
		if _, err := NewFetcher(server.URL, dir).FetchYear(context.Background(), year); err == nil {
			t.Fatal("a 500 response was accepted")
		}
	})
}

func TestFetcherRejectsAnOversizedFile(t *testing.T) {
	year := time.Now().Year() + 1
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", MaxCalendarBytes+10)))
	}))
	defer server.Close()

	if _, err := NewFetcher(server.URL, t.TempDir()).FetchYear(context.Background(), year); err == nil {
		t.Fatal("an oversized file was accepted")
	} else if !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetcherYearURL(t *testing.T) {
	cases := []struct {
		base string
		year int
		want string
	}{
		{"https://example.test/holiday-cn", 2027, "https://example.test/holiday-cn/2027.json"},
		{"https://example.test/holiday-cn/", 2027, "https://example.test/holiday-cn/2027.json"},
		{"https://example.test/{year}.json", 2027, "https://example.test/2027.json"},
	}
	for _, tc := range cases {
		if got := NewFetcher(tc.base, "/tmp").yearURL(tc.year); got != tc.want {
			t.Errorf("yearURL(%q, %d) = %q, want %q", tc.base, tc.year, got, tc.want)
		}
	}
}

func TestFetcherNeedsACacheDirectory(t *testing.T) {
	if _, err := NewFetcher("https://example.test", "").FetchYear(context.Background(), 2027); err == nil {
		t.Fatal("a fetch with no cache directory was accepted")
	}
}

// The periodic path: Run refreshes on every tick and reports the years it
// changed.
func TestFetcherRunRefreshesOnItsInterval(t *testing.T) {
	dir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(oneYear(2026)))
	}))
	defer server.Close()

	fetcher := NewFetcher(server.URL, dir)
	fetcher.Years = func(time.Time) []int { return []int{2026} }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	seen := make(chan []int, 1)
	go fetcher.Run(ctx, 30*time.Millisecond, func(changed []int) {
		select {
		case seen <- changed:
		default:
		}
	})

	select {
	case changed := <-seen:
		if len(changed) != 1 || changed[0] != 2026 {
			t.Fatalf("onChange reported %v, want [2026]", changed)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the fetcher never refreshed")
	}
	if _, err := os.Stat(filepath.Join(dir, "cn-2026.json")); err != nil {
		t.Fatalf("the periodic refresh wrote no file: %v", err)
	}
}
