package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"
)

// Audit-archive read side.
//
// Retention (see audit_service.go) appends expired rows to a per-day JSONL file
// before deleting them, which keeps the live table bounded without losing the
// trail. The catch was that the archive was a file on the server: reading it
// meant SSHing in and grepping. This service answers the same question the live
// list does - "what happened" - against the archived day, with the same bounded
// reverse scan the log search uses.

const (
	// DefaultAuditArchiveLimit is how many entries a query returns when the
	// caller does not ask for a specific number.
	DefaultAuditArchiveLimit = 50
	// MaxAuditArchiveLimit bounds one response.
	MaxAuditArchiveLimit = 500
	// auditArchiveScanBudget bounds how much of one archive file a query reads.
	// Hitting it sets `truncated` rather than silently pretending the file ended.
	auditArchiveScanBudget = 32 << 20
	// auditArchiveDateLayout is the one date form the endpoint accepts and the
	// form encoded in the archive file name.
	auditArchiveDateLayout = "2006-01-02"
)

// ErrAuditArchiveNotConfigured means no AUDIT_ARCHIVE_DIR is set, so there is
// nothing to read.
var ErrAuditArchiveNotConfigured = errors.New("audit archive is not configured")

// ErrAuditArchiveNotFound means the requested day has no archive file: either
// nothing expired that day, or the day is not the one that was written.
var ErrAuditArchiveNotFound = errors.New("audit archive file not found")

// ErrAuditArchiveBadDate means the requested day is not a YYYY-MM-DD date, so
// it cannot name an archive file. Callers should surface it as a bad request
// rather than a server error: the caller's input is wrong, not the server.
var ErrAuditArchiveBadDate = errors.New("date must be YYYY-MM-DD")

// AuditArchiveQuery is one archive search. The zero value is "the newest 50
// entries from today's archive".
type AuditArchiveQuery struct {
	// Date selects the day (YYYY-MM-DD). Empty means today in UTC.
	Date string
	// Query is a case-insensitive substring across the row's string fields
	// (action, resource, details, ip, admin_username).
	Query string
	// Action matches the action column exactly.
	Action string
	// AdminUsername is a case-insensitive substring on the admin actor name.
	AdminUsername string
	// ActorUserID narrows to one end-user actor. Zero means "no filter".
	ActorUserID uint
	// Limit caps returned entries; 0 means DefaultAuditArchiveLimit.
	Limit int
}

// AuditArchiveResult is the query response, including what the scan cost so a
// caller can tell "nothing matched" from "the scan gave up".
type AuditArchiveResult struct {
	Date         string            `json:"date"`
	File         string            `json:"file"`
	Items        []models.AuditLog `json:"items"`
	ScannedBytes int64             `json:"scanned_bytes"`
	ScannedLines int               `json:"scanned_lines"`
	// Truncated reports that the byte budget ran out before the scan reached the
	// start of the file, so an empty result is not proof the event never happened.
	Truncated bool `json:"truncated"`
	// HasMore reports that the response was capped: either the limit was
	// reached or the budget ran out, so older matching entries may exist.
	HasMore bool `json:"has_more"`
}

// auditArchiveFileName is the per-day archive file name, matching what
// AuditService.archiveRows writes.
func auditArchiveFileName(date string) string {
	return fmt.Sprintf("audit-logs-%s.jsonl", date)
}

// QueryArchive scans one day's archive newest-first and returns the first
// `Limit` matching rows. It never touches the database, so it works even after
// the live rows have been purged.
func (s *AuditService) QueryArchive(ctx context.Context, q AuditArchiveQuery) (AuditArchiveResult, error) {
	var out AuditArchiveResult
	if s == nil || strings.TrimSpace(s.archiveDir) == "" {
		return out, ErrAuditArchiveNotConfigured
	}

	date := strings.TrimSpace(q.Date)
	if date == "" {
		date = time.Now().UTC().Format(auditArchiveDateLayout)
	} else if _, err := time.Parse(auditArchiveDateLayout, date); err != nil {
		return out, fmt.Errorf("%w: got %q", ErrAuditArchiveBadDate, date)
	}
	out.Date = date
	out.File = auditArchiveFileName(date)
	out.Items = []models.AuditLog{}

	limit := q.Limit
	if limit <= 0 {
		limit = DefaultAuditArchiveLimit
	}
	if limit > MaxAuditArchiveLimit {
		limit = MaxAuditArchiveLimit
	}

	path := filepath.Join(s.archiveDir, out.File)
	budget := int64(auditArchiveScanBudget)
	exhausted, err := forEachLineReverse(path, &budget, func(line string) bool {
		out.ScannedLines++
		var row models.AuditLog
		// A malformed line is skipped rather than failing the whole query: one
		// damaged entry must not make the rest of the day unreadable.
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return true
		}
		if !auditArchiveMatches(row, q) {
			return true
		}
		out.Items = append(out.Items, row)
		return len(out.Items) < limit
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return out, fmt.Errorf("%w: %s", ErrAuditArchiveNotFound, out.File)
		}
		return out, fmt.Errorf("read %s: %w", out.File, err)
	}
	out.Truncated = exhausted
	out.ScannedBytes = int64(auditArchiveScanBudget) - budget
	out.HasMore = out.Truncated || len(out.Items) >= limit
	return out, nil
}

// auditArchiveMatches applies the filters to one archived row.
func auditArchiveMatches(row models.AuditLog, q AuditArchiveQuery) bool {
	if a := strings.TrimSpace(q.Action); a != "" && row.Action != a {
		return false
	}
	if q.ActorUserID > 0 && row.ActorUserID != q.ActorUserID {
		return false
	}
	if u := strings.TrimSpace(q.AdminUsername); u != "" {
		if !strings.Contains(strings.ToLower(row.AdminUsername), strings.ToLower(u)) {
			return false
		}
	}
	if term := strings.TrimSpace(q.Query); term != "" {
		haystack := strings.ToLower(strings.Join([]string{
			row.Action,
			row.ResourceType,
			row.ResourceID,
			row.Details,
			row.IP,
			row.AdminUsername,
		}, "\n"))
		if !strings.Contains(haystack, strings.ToLower(term)) {
			return false
		}
	}
	return true
}
