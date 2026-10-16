package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Log aggregation for a single-node deployment, part one: being able to ask the
// running server what happened.
//
// The application log is a rotating file (`signaling.log`, `.1`, `.2`, …) that
// the logger keeps bounded, so it is already the aggregate - what was missing
// was any way to read it other than SSHing in and running grep. This service
// answers that with a bounded reverse scan: newest lines first, a byte budget
// so a large log cannot be pulled into memory, and the same filters an operator
// would otherwise type by hand.
//
// Reading backwards rather than tailing the whole file matters: the interesting
// question is almost always "what just happened", and the newest line is at the
// end of a file that may be tens of megabytes.

const (
	// DefaultLogQueryLimit is how many matching lines a query returns when the
	// caller does not ask for a specific number.
	DefaultLogQueryLimit = 200
	// MaxLogQueryLimit bounds one response. The admin UI shows a page of lines,
	// not a download.
	MaxLogQueryLimit = 2000
	// logBlockSize is the reverse-scan read size. One block is a handful of
	// lines; larger blocks waste reads on a small limit, smaller ones cost
	// syscalls.
	logBlockSize = 64 * 1024
	// logScanBudget bounds how much of the log one query reads. Hitting it sets
	// `truncated` rather than silently pretending the log ends there.
	logScanBudget = 8 << 20
)

// LogQuery is one admin log search. The zero value is "newest 200 lines".
type LogQuery struct {
	// Query is a case-insensitive substring match over the whole line.
	Query string
	// Match is a full-text expression with the same grammar the aggregated
	// store gets from `websearch_to_tsquery('simple', …)`: words are ANDed,
	// "quoted text" is a phrase, `or` alternates and a leading `-` negates.
	// It is a separate mode from Query on purpose: substring finds a request
	// id, full text finds words without matching them apart.
	Match string
	// Component matches the `component=...` field the structured log helpers
	// write, as a whole token (`component=api` does not match `component=api_auth`).
	Component string
	// Since and Until bound the lines by their leading timestamp. The logger
	// writes UTC, so the bounds are compared in UTC.
	Since *time.Time
	Until *time.Time
	// Limit caps returned lines; 0 means DefaultLogQueryLimit.
	Limit int
	// File selects which configured log source to scan. Empty means the
	// application log (DefaultLogSourceName) when it is configured.
	File string
}

// Paging back through history is done with `Until`, not a second scan
// direction: take the oldest timestamp from one page and ask again with
// `until` set to it. One reverse scan is easier to keep correct than two.

// LogEntry is one matching line.
type LogEntry struct {
	// Timestamp is RFC3339 UTC, or empty when the line has no parseable prefix.
	Timestamp string `json:"timestamp,omitempty"`
	Line      string `json:"line"`
	// File names the source the line came from (`signaling.log`, `postgres.log`,
	// …). It is set on every entry so a search across all sources can still say
	// which file each hit belongs to.
	File string `json:"file,omitempty"`
}

// LogFileInfo describes one file the query walked.
type LogFileInfo struct {
	Name  string `json:"name"`
	Bytes int64  `json:"bytes"`
	// path is where the file actually lives. It is unexported so the JSON
	// response keeps reporting a stable logical name and never leaks a host
	// path.
	path string
}

// LogQueryResult is the query response, including what it cost to produce so a
// caller can tell "nothing matched" from "the scan gave up".
type LogQueryResult struct {
	Entries []LogEntry    `json:"items"`
	Files   []LogFileInfo `json:"files"`
	// Match echoes the full-text expression that was applied, so "nothing
	// matched" can be told from "the mode I asked for was not used".
	Match string `json:"match,omitempty"`
	// ScannedBytes is how much of the log this query read.
	ScannedBytes int64 `json:"scanned_bytes"`
	// ScannedLines counts lines the scan looked at, matched or not.
	ScannedLines int `json:"scanned_lines"`
	// Truncated reports that the byte budget ran out before the scan reached
	// the oldest file, so an empty result is not proof the event never happened.
	Truncated bool `json:"truncated"`
}

// LogQueryService reads one or more rotating log files. The application log is
// always the default source; PostgreSQL and Redis logs can be registered as
// extra sources so "what happened around 3pm" is answerable without first
// downloading each file and grepping locally.
type LogQueryService struct {
	dir     string
	sources map[string]string
	order   []string
	// scanBudget bounds how many bytes one query reads. It defaults to
	// logScanBudget and exists as a field so tests can drive the truncation
	// path without writing eight megabytes of fixture data.
	scanBudget int64
}

// DefaultLogSourceName is the application log's source name. It is also the
// file the search endpoint uses when the caller does not pass `file=`.
const DefaultLogSourceName = "signaling.log"

// AllLogSources is the reserved `file=` value that scans every registered
// source in one pass with one shared byte budget. Reading the sources one at a
// time was the practical complaint: an incident usually spans the application
// log and the database log, and making the operator guess which one to look at
// first is work the server can do.
const AllLogSources = "all"

// Log search scopes. `local` reads the files on the instance that answered the
// request, which is what the endpoint always did; `all` reads the lines every
// instance shipped into the shared database, which is what makes a
// multi-instance deployment answerable from one place.
const (
	LogScopeLocal = "local"
	LogScopeAll   = "all"
)

// NewLogQueryService points the service at the directory the logger writes to.
func NewLogQueryService(dir string) *LogQueryService {
	s := &LogQueryService{dir: dir, sources: map[string]string{}, scanBudget: logScanBudget}
	if strings.TrimSpace(dir) != "" {
		s.AddFile(DefaultLogSourceName, filepath.Join(dir, DefaultLogSourceName))
	}
	return s
}

// AddFile registers one more searchable log source. Registering the same name
// twice replaces the path without duplicating the source in the order, so a
// caller cannot accidentally scan a file twice.
func (s *LogQueryService) AddFile(name, path string) *LogQueryService {
	if s == nil {
		return s
	}
	name = strings.TrimSpace(name)
	path = strings.TrimSpace(path)
	if name == "" || path == "" {
		return s
	}
	if s.sources == nil {
		s.sources = map[string]string{}
	}
	if _, exists := s.sources[name]; !exists {
		s.order = append(s.order, name)
	}
	s.sources[name] = path
	return s
}

// LogFileSource is one registered source, in registration order.
type LogFileSource struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Sources lists the registered sources in registration order.
func (s *LogQueryService) Sources() []LogFileSource {
	if s == nil {
		return nil
	}
	out := make([]LogFileSource, 0, len(s.order))
	for _, name := range s.order {
		out = append(out, LogFileSource{Name: name, Path: s.sources[name]})
	}
	return out
}

// LogDir reports the directory being read ("" when not configured).
func (s *LogQueryService) LogDir() string {
	if s == nil {
		return ""
	}
	return s.dir
}

// Query walks the log newest-file-first and returns the first `limit` matching
// lines. It never returns an error for a missing log: a server that has not
// written anything yet simply has nothing to show.
func (s *LogQueryService) Query(ctx context.Context, q LogQuery) (LogQueryResult, error) {
	var out LogQueryResult
	if s == nil || len(s.order) == 0 {
		out.Entries = []LogEntry{}
		out.Files = []LogFileInfo{}
		return out, nil
	}
	limit := q.Limit
	if limit <= 0 {
		limit = DefaultLogQueryLimit
	}
	if limit > MaxLogQueryLimit {
		limit = MaxLogQueryLimit
	}
	// Compile the full-text expression once for the whole scan rather than per
	// candidate line.
	match := parseLogMatch(q.Match)
	if strings.TrimSpace(q.Match) != "" {
		out.Match = strings.TrimSpace(q.Match)
	}

	// `file=all` scans every registered source; anything else resolves to one.
	var names []string
	if strings.TrimSpace(q.File) == AllLogSources {
		names = append(names, s.order...)
	} else {
		sourceName, _, ok := s.resolveSource(q.File)
		if !ok {
			out.Entries = []LogEntry{}
			out.Files = []LogFileInfo{}
			return out, nil
		}
		names = []string{sourceName}
	}

	out.Entries = make([]LogEntry, 0, min(limit, 64))

	budget := s.scanBudget
	if budget <= 0 {
		budget = logScanBudget
	}
	totalBudget := budget
	for _, name := range names {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		if len(out.Entries) >= limit {
			break
		}
		if budget <= 0 {
			out.Truncated = true
			break
		}
		if err := s.scanSource(name, q, match, limit, &budget, &out); err != nil {
			return out, err
		}
		if out.Truncated {
			break
		}
	}
	out.ScannedBytes = totalBudget - budget
	return out, nil
}

// scanSource walks one source's rotated files newest-first, appending matches
// to out and spending from the shared budget. The budget stays shared across
// sources on purpose: "search everything" must still be one bounded read, not
// one full read per file.
func (s *LogQueryService) scanSource(
	sourceName string,
	q LogQuery,
	match logMatchQuery,
	limit int,
	budget *int64,
	out *LogQueryResult,
) error {
	files := logFilesFor(sourceName, s.sources[sourceName])
	out.Files = append(out.Files, files...)
	for _, file := range files {
		if len(out.Entries) >= limit || *budget <= 0 {
			return nil
		}
		exhausted, err := forEachLineReverse(file.path, budget, func(line string) bool {
			out.ScannedLines++
			if !logLineMatches(line, q, match) {
				return true
			}
			entry := logEntry(line)
			entry.File = sourceName
			out.Entries = append(out.Entries, entry)
			return len(out.Entries) < limit
		})
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("read %s: %w", file.Name, err)
		}
		if exhausted {
			// The budget ran out inside this source: stop the whole scan rather
			// than silently continuing with a different file.
			out.Truncated = true
			return nil
		}
	}
	return nil
}

// resolveSource picks the file to scan. An empty name means the application
// log when it is registered, otherwise the first source; an unknown name
// resolves to nothing rather than silently scanning a different file.
func (s *LogQueryService) resolveSource(name string) (string, string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		if _, ok := s.sources[DefaultLogSourceName]; ok {
			name = DefaultLogSourceName
		} else if len(s.order) > 0 {
			name = s.order[0]
		}
	}
	path, ok := s.sources[name]
	if !ok {
		return "", "", false
	}
	return name, path, true
}

// logFilesFor lists the current log file and its rotated siblings, newest
// first. The rotating writer names them `<name>`, `<name>.1`, … with `.1` being
// the most recent rotation, so the same rule works for the application log and
// for an externally rotated PostgreSQL or Redis log.
func logFilesFor(sourceName, sourcePath string) []LogFileInfo {
	out := make([]LogFileInfo, 0, 4)
	collect := func(reportName, path string) {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			return
		}
		out = append(out, LogFileInfo{Name: reportName, Bytes: info.Size(), path: path})
	}
	collect(sourceName, sourcePath)
	for i := 1; i <= 64; i++ {
		path := fmt.Sprintf("%s.%d", sourcePath, i)
		if _, err := os.Stat(path); err != nil {
			break
		}
		collect(fmt.Sprintf("%s.%d", sourceName, i), path)
	}
	return out
}

// forEachLineReverse calls fn for every line of path from the last one to the
// first, spending at most *budget bytes. fn returning false stops the walk.
//
// The bool result reports that the walk stopped because the budget ran out
// before reaching the start of the file - the caller has to be able to tell
// "the log has nothing older" from "I stopped looking", because an empty result
// from a truncated scan is not evidence that an event never happened.
//
// The scan works on fixed-size blocks read from the end: the first fragment of
// a block is the tail of a line whose beginning lives in the previous block, so
// it is carried over instead of being handed out half-formed.
func forEachLineReverse(path string, budget *int64, fn func(line string) bool) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return false, err
	}
	size := info.Size()
	offset := size
	var carry []byte
	for offset > 0 {
		if *budget <= 0 {
			return true, nil
		}
		readSize := int64(logBlockSize)
		if readSize > offset {
			readSize = offset
		}
		// Never read past the budget: "scanned_bytes <= budget" is then an
		// invariant a caller can rely on, instead of a bound that can overshoot
		// by a whole block.
		if readSize > *budget {
			readSize = *budget
		}
		if readSize <= 0 {
			return true, nil
		}
		offset -= readSize
		buf := make([]byte, readSize)
		if _, err := f.ReadAt(buf, offset); err != nil && err != io.EOF {
			return false, err
		}
		*budget -= readSize

		data := buf
		if len(carry) > 0 {
			data = append(append([]byte(nil), buf...), carry...)
		}
		parts := bytes.Split(data, []byte{'\n'})
		if offset > 0 {
			carry = append([]byte(nil), parts[0]...)
			parts = parts[1:]
		} else {
			carry = nil
		}
		for i := len(parts) - 1; i >= 0; i-- {
			line := string(parts[i])
			if line == "" {
				continue
			}
			if !fn(line) {
				return false, nil
			}
		}
	}
	if len(carry) > 0 && *budget > 0 {
		// The very first line of the file never had a preceding newline.
		fn(string(carry))
	}
	return false, nil
}

// logLineMatches applies the query filters to one line.
func logLineMatches(line string, q LogQuery, match logMatchQuery) bool {
	if q.Match != "" && !match.matches(line) {
		return false
	}
	if q.Query != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(q.Query)) {
		return false
	}
	if q.Component != "" && !logLineHasComponent(line, q.Component) {
		return false
	}
	if q.Since != nil || q.Until != nil {
		ts, ok := parseLogTimestamp(line)
		if !ok {
			// A line without a timestamp cannot be placed in time, so a time
			// filter excludes it rather than guessing.
			return false
		}
		if q.Since != nil && ts.Before(q.Since.UTC()) {
			return false
		}
		if q.Until != nil && ts.After(q.Until.UTC()) {
			return false
		}
	}
	return true
}

// logLineHasComponent looks for `component=<value>` as a whole token, so
// `component=api` does not match a `component=api_auth` line.
func logLineHasComponent(line, component string) bool {
	needle := "component=" + component
	for start := 0; start < len(line); {
		i := strings.Index(line[start:], needle)
		if i < 0 {
			return false
		}
		at := start + i
		end := at + len(needle)
		startsToken := at == 0 || line[at-1] == ' '
		endsToken := end == len(line) || line[end] == ' '
		if startsToken && endsToken {
			return true
		}
		start = end
	}
	return false
}

// parseLogTimestamp reads the prefix the standard logger writes
// (`2006/01/02 15:04:05.000000`, UTC).
func parseLogTimestamp(line string) (time.Time, bool) {
	const withMicros = "2006/01/02 15:04:05.000000"
	const withSeconds = "2006/01/02 15:04:05"
	if len(line) >= len(withMicros) {
		if ts, err := time.Parse(withMicros, line[:len(withMicros)]); err == nil {
			return ts.UTC(), true
		}
	}
	if len(line) >= len(withSeconds) {
		if ts, err := time.Parse(withSeconds, line[:len(withSeconds)]); err == nil {
			return ts.UTC(), true
		}
	}
	return time.Time{}, false
}

// logEntry renders one line, keeping the timestamp as a real field so a client
// does not have to re-parse the logger's format.
func logEntry(line string) LogEntry {
	entry := LogEntry{Line: line}
	if ts, ok := parseLogTimestamp(line); ok {
		entry.Timestamp = ts.Format(time.RFC3339)
	}
	return entry
}
