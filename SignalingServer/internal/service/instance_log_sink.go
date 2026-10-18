package service

import (
	"context"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Shipping bounds. The sink sits in the logging path, so every one of these
// exists to make "the log store is slow or down" cost the process nothing more
// than lost copies.
const (
	// DefaultInstanceLogBufferLines bounds how many lines may be queued before
	// the sink starts dropping. A request must never block on log shipping.
	DefaultInstanceLogBufferLines = 4096
	// instanceLogFlushInterval is how long a line may sit in memory before it is
	// written; it doubles as the retry interval when the database is unhappy.
	instanceLogFlushInterval = time.Second
	// instanceLogBatchLines bounds one insert statement.
	instanceLogBatchLines = 500
	// instanceLogMaxLineBytes truncates a single line. A log line is a
	// diagnostic, not a payload: shipping a multi-megabyte one would cost more
	// than it tells us.
	instanceLogMaxLineBytes = 4096
	// instanceLogPurgeBatch bounds one retention delete.
	instanceLogPurgeBatch = 5000
	// instanceLogFailureLogInterval throttles the line the sink logs when the
	// store refuses a batch, because that line is itself shippable.
	instanceLogFailureLogInterval = 30 * time.Second
	// instanceLogTimestampLayout matches the standard logger's prefix with
	// log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC, which is what Configure
	// sets.
	instanceLogTimestampLayout = "2006/01/02 15:04:05.000000"
)

// InstanceLogSink ships this process's log lines into the shared database so a
// search can cover every instance rather than only the machine that answered
// the request. It is an io.Writer, attached to the logger with
// observability.AttachLogSink after the database is open.
//
// Three properties matter more than throughput:
//
//   - It never blocks a caller. Lines go into a bounded channel; when that is
//     full the line is dropped and counted instead of stalling the request that
//     was being logged.
//   - It never fails a write. The authoritative copy of a line is still stdout
//     and the rotating file on this instance, so a broken database degrades to
//     "aggregation stopped" and the drop/stored counters say so.
//   - It is idempotent. Each shipped line carries a per-instance sequence
//     number and the insert ignores conflicts, so a batch that is retried after
//     a timeout that had actually committed does not duplicate lines.
type InstanceLogSink struct {
	db         *gorm.DB
	instanceID string
	source     string
	lines      chan models.InstanceLogEntry
	flushEvery time.Duration
	batchLines int

	seq     atomic.Int64
	dropped atomic.Int64
	stored  atomic.Int64
	// storeFailures counts failed inserts, and lastStoreFailureLog throttles the
	// line the sink logs about them. Without the throttle a store that is down
	// would log "store_failed" every tick - and because that line is itself a
	// log line the sink would try to ship it, once a second, forever.
	storeFailures       atomic.Int64
	lastStoreFailureLog atomic.Int64
	// chain serialises flushes so two callers can never insert the same batch.
	chain sync.Mutex
}

// NewInstanceLogSink returns a sink for one process. instanceID is the id the
// server logs and registers, so a shipped line can be traced back to a process.
func NewInstanceLogSink(db *gorm.DB, instanceID string) *InstanceLogSink {
	return &InstanceLogSink{
		db:         db,
		instanceID: strings.TrimSpace(instanceID),
		source:     DefaultLogSourceName,
		lines:      make(chan models.InstanceLogEntry, DefaultInstanceLogBufferLines),
		flushEvery: instanceLogFlushInterval,
		batchLines: instanceLogBatchLines,
	}
}

// WithBuffer overrides the queue depth. Tests use a tiny buffer to drive the
// drop path without writing thousands of lines.
func (s *InstanceLogSink) WithBuffer(lines int) *InstanceLogSink {
	if lines > 0 {
		s.lines = make(chan models.InstanceLogEntry, lines)
	}
	return s
}

// WithSource names the log source the shipped lines belong to.
func (s *InstanceLogSink) WithSource(source string) *InstanceLogSink {
	if trimmed := strings.TrimSpace(source); trimmed != "" {
		s.source = trimmed
	}
	return s
}

// WithFlushInterval overrides how long a line may wait before it is written.
func (s *InstanceLogSink) WithFlushInterval(d time.Duration) *InstanceLogSink {
	if d > 0 {
		s.flushEvery = d
	}
	return s
}

// Enqueue puts an already-identified line into the shipping queue. The file
// tailers use this rather than Write because a line they read already knows its
// identity: the file it came from, a generation that changes when that file is
// rotated or truncated, and the byte offset it started at. That tuple is what
// makes a re-read idempotent, so it must not be replaced by this process's own
// counter.
//
// Returns false when the queue is full; the caller does not need to react,
// because the drop is counted either way and the authoritative copy is still
// the file on disk.
func (s *InstanceLogSink) Enqueue(entry models.InstanceLogEntry) bool {
	if s == nil || s.db == nil || s.instanceID == "" {
		return false
	}
	if entry.InstanceID == "" {
		entry.InstanceID = s.instanceID
	}
	if entry.LoggedAt.IsZero() {
		entry.LoggedAt = time.Now().UTC()
	}
	entry.Line = truncateInstanceLogLine(entry.Line)
	select {
	case s.lines <- entry:
		return true
	default:
		s.dropped.Add(1)
		return false
	}
}

// Write implements io.Writer, so the sink can be attached to the logger. It
// always reports the whole buffer as written and never returns an error: a
// shipping problem is reported through Dropped/Stored, not by breaking the
// caller's log call.
func (s *InstanceLogSink) Write(p []byte) (int, error) {
	if s == nil || s.db == nil || s.instanceID == "" {
		return len(p), nil
	}
	now := time.Now().UTC()
	for _, raw := range strings.Split(string(p), "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		entry := models.InstanceLogEntry{
			InstanceID: s.instanceID,
			Seq:        s.seq.Add(1),
			Source:     s.source,
			LoggedAt:   parseInstanceLogTime(line, now),
			Line:       truncateInstanceLogLine(line),
		}
		select {
		case s.lines <- entry:
		default:
			// Full queue: drop the copy, keep the request moving. The counter is
			// reported by the search endpoint so "missing lines" is visible
			// rather than silent.
			s.dropped.Add(1)
		}
	}
	return len(p), nil
}

// Run flushes queued lines to the database until ctx is cancelled, then makes
// one last attempt so a graceful shutdown does not drop what is already queued.
func (s *InstanceLogSink) Run(ctx context.Context) {
	if s == nil || s.db == nil {
		return
	}
	ticker := time.NewTicker(s.flushEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			s.flush(ctx, s.batchLines)
			return
		case <-ticker.C:
			s.flush(ctx, s.batchLines)
		}
	}
}

// Flush writes up to batchLines queued lines and returns how many were stored.
// Exported so a test (and a shutdown path) can drain without waiting for a tick.
func (s *InstanceLogSink) Flush(ctx context.Context) int {
	if s == nil || s.db == nil {
		return 0
	}
	return s.flush(ctx, s.batchLines)
}

func (s *InstanceLogSink) flush(ctx context.Context, max int) int {
	s.chain.Lock()
	defer s.chain.Unlock()

	batch := make([]models.InstanceLogEntry, 0, max)
	for len(batch) < max {
		select {
		case entry := <-s.lines:
			batch = append(batch, entry)
		default:
			if len(batch) == 0 {
				return 0
			}
			return s.store(ctx, batch)
		}
	}
	if len(batch) == 0 {
		return 0
	}
	return s.store(ctx, batch)
}

func (s *InstanceLogSink) store(ctx context.Context, batch []models.InstanceLogEntry) int {
	// OnConflict DoNothing is what makes a re-sent batch idempotent: the unique
	// key is (instance_id, source, seq), so a retry inserts nothing new.
	res := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(batch, instanceLogBatchLines)
	if res.Error != nil {
		// Put nothing back: the file on this instance still has the lines, and
		// re-queueing a failing batch would just refill the buffer with rows the
		// database is already refusing.
		failures := s.storeFailures.Add(1)
		now := time.Now().UnixNano()
		if now-s.lastStoreFailureLog.Load() > int64(instanceLogFailureLogInterval) {
			s.lastStoreFailureLog.Store(now)
			log.Printf("component=instance_log_sink event=store_failed lines=%d failures=%d error=%q",
				len(batch), failures, res.Error.Error())
		}
		return 0
	}
	// The conflict clause means "inserted" and "attempted" differ on a re-sent
	// batch, so report what the database actually took.
	s.stored.Add(res.RowsAffected)
	return int(res.RowsAffected)
}

// Stored reports how many lines have been written to the shared store.
func (s *InstanceLogSink) Stored() int64 {
	if s == nil {
		return 0
	}
	return s.stored.Load()
}

// Dropped reports how many lines this process could not queue. It is the honest
// counterpart to Stored: a search that returns nothing must be able to tell
// "the line never existed" from "this instance dropped lines under load".
func (s *InstanceLogSink) Dropped() int64 {
	if s == nil {
		return 0
	}
	return s.dropped.Load()
}

// PurgeBefore deletes shipped lines older than cutoff in bounded batches and
// reports how many went. There is no archive step on purpose: unlike the audit
// trail, a shipped line is a copy whose original still lives in the rotating
// file on the instance that wrote it.
func (s *InstanceLogSink) PurgeBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	res := s.db.WithContext(ctx).Exec(
		"DELETE FROM instance_log_entries WHERE id IN (SELECT id FROM instance_log_entries WHERE logged_at < ? ORDER BY id ASC LIMIT ?)",
		cutoff.UTC(), instanceLogPurgeBatch)
	return res.RowsAffected, res.Error
}

// RunRetention purges shipped lines on a ticker until ctx is cancelled.
// retentionDays <= 0 or interval <= 0 disables it.
func (s *InstanceLogSink) RunRetention(ctx context.Context, interval time.Duration, retentionDays int) {
	if s == nil || s.db == nil || interval <= 0 || retentionDays <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
			total := int64(0)
			for {
				removed, err := s.PurgeBefore(ctx, cutoff)
				if err != nil {
					log.Printf("component=instance_log_sink event=retention_failed error=%q", err.Error())
					break
				}
				total += removed
				if removed < instanceLogPurgeBatch {
					break
				}
			}
			if total > 0 {
				log.Printf("component=instance_log_sink event=retention_purged rows=%d older_than_days=%d", total, retentionDays)
			}
		}
	}
}

// parseInstanceLogTime reads the timestamp the standard logger puts at the
// front of a line. Lines written by something else (a panic, a third-party
// writer) have no prefix; those fall back to "now", which is closer than
// dropping them.
func parseInstanceLogTime(line string, fallback time.Time) time.Time {
	if len(line) >= len(instanceLogTimestampLayout) {
		if parsed, err := time.Parse(instanceLogTimestampLayout, line[:len(instanceLogTimestampLayout)]); err == nil {
			return parsed.UTC()
		}
	}
	return fallback
}

// truncateInstanceLogLine bounds one line and keeps it valid UTF-8, because the
// shared column is `text` and PostgreSQL rejects invalid encoding outright.
func truncateInstanceLogLine(line string) string {
	if len(line) <= instanceLogMaxLineBytes {
		return line
	}
	return strings.ToValidUTF8(line[:instanceLogMaxLineBytes], "")
}
