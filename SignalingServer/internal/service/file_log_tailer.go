package service

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"quickdesk/signaling/internal/models"
)

// Log source names for the files another process writes. They match the names
// the local scan uses for the same files, so `source` means the same thing
// whichever scope a caller picked.
const (
	PostgresLogSourceName = "postgres.log"
	RedisLogSourceName    = "redis.log"
)

// Tailer bounds.
const (
	// DefaultFileLogPollInterval is how often a tailed file is checked. A log
	// line is a diagnostic; a couple of seconds of lag is invisible next to the
	// cost of polling hard on a database that is already in trouble.
	DefaultFileLogPollInterval = 2 * time.Second
	// fileLogTailerMaxLinesPerTick bounds one poll. A file that grew by more
	// than this during a stall is picked up over the following ticks rather than
	// in one unbounded read, which keeps a burst from flooding the sink queue.
	fileLogTailerMaxLinesPerTick = 2000
	// fileLogTailerSeqScale separates sequence numbers of different file
	// generations. The sequence of a line is
	// `generation*fileLogTailerSeqScale + offset`: offsets restart at zero after
	// a rotation, so without the generation every post-rotation line would
	// collide with a pre-rotation one and be dropped as a duplicate.
	fileLogTailerSeqScale = int64(1_000_000_000_000)
	// fileLogTailerFailureLogInterval throttles the line the tailer logs when it
	// cannot read the file, because that line is itself shippable.
	fileLogTailerFailureLogInterval = 30 * time.Second
)

// FileLogTailer ships the lines another process appends to a plain-text log
// file - the PostgreSQL and Redis logs - into the same shared store the
// application log goes to, so a `scope=all` search covers every instance's
// database log as well.
//
// It reads; it never writes or rotates the file it follows. The offset is the
// only cursor, and the identity of a line is `(source, generation, offset)`
// rather than the offset alone: log rotation reuses offsets, so treating the
// offset as the identity would silently drop every line written after a
// rotation as "already seen".
type FileLogTailer struct {
	sink      *InstanceLogSink
	path      string
	source    string
	pollEvery time.Duration
	maxLines  int

	// State describing the file as it was last seen.
	device     uint64
	inode      uint64
	generation int64
	offset     int64
	attached   bool

	readFailures       atomic.Int64
	lastReadFailureLog atomic.Int64
	rotations          atomic.Int64
	shipped            atomic.Int64
}

// NewFileLogTailer follows path and ships its lines under source. The tailer
// starts at the current end of the file: an operator wants the lines written
// while the server runs, not a replay of whatever history the database happened
// to leave on disk.
func NewFileLogTailer(sink *InstanceLogSink, path, source string) *FileLogTailer {
	return &FileLogTailer{
		sink:      sink,
		path:      strings.TrimSpace(path),
		source:    strings.TrimSpace(source),
		pollEvery: DefaultFileLogPollInterval,
		maxLines:  fileLogTailerMaxLinesPerTick,
	}
}

// WithPollInterval overrides how often the file is checked.
func (t *FileLogTailer) WithPollInterval(d time.Duration) *FileLogTailer {
	if t != nil && d > 0 {
		t.pollEvery = d
	}
	return t
}

// Shipped reports how many lines this tailer has handed to the sink. It is not
// the same as stored: the sink may still drop them, and it reports that itself.
func (t *FileLogTailer) Shipped() int64 {
	if t == nil {
		return 0
	}
	return t.shipped.Load()
}

// Rotations reports how many times the followed file was replaced or truncated.
func (t *FileLogTailer) Rotations() int64 {
	if t == nil {
		return 0
	}
	return t.rotations.Load()
}

// Run polls until ctx is cancelled, then makes one last pass so a graceful
// shutdown picks up whatever was appended since the previous tick.
func (t *FileLogTailer) Run(ctx context.Context) {
	if t == nil || t.sink == nil || t.path == "" {
		return
	}
	ticker := time.NewTicker(t.pollEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_, _ = t.Poll(context.Background())
			return
		case <-ticker.C:
			if _, err := t.Poll(ctx); err != nil {
				t.logReadFailure(err)
			}
		}
	}
}

// Poll reads whatever complete lines have appeared since the previous call and
// returns how many were handed to the sink. A missing file is not an error: the
// configured path may simply not exist yet, and the tailer keeps its state so
// that the file appearing later is followed from its start.
func (t *FileLogTailer) Poll(ctx context.Context) (int, error) {
	if t == nil || t.sink == nil || t.path == "" {
		return 0, nil
	}
	if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return 0, err
	}

	info, err := os.Stat(t.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}

	device, inode := fileLogIdentity(info)
	if !t.attached {
		t.device, t.inode = device, inode
		t.generation = 1
		t.offset = info.Size()
		t.attached = true
		return 0, nil
	}

	// A different file behind the same path, or a file that shrank, both mean
	// the bytes at a given offset are no longer the bytes they were. Bumping the
	// generation is what keeps the new lines from colliding with the old ones.
	if device != t.device || inode != t.inode || info.Size() < t.offset {
		t.generation++
		t.offset = 0
		t.device, t.inode = device, inode
		t.rotations.Add(1)
	}
	if info.Size() <= t.offset {
		return 0, nil
	}

	file, err := os.Open(t.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Rotated away between the stat and the open; the next poll sees the
			// new file and bumps the generation then.
			return 0, nil
		}
		return 0, err
	}
	defer file.Close()
	if _, err := file.Seek(t.offset, io.SeekStart); err != nil {
		return 0, err
	}

	reader := bufio.NewReaderSize(file, 64<<10)
	now := time.Now().UTC()
	cursor := t.offset
	shipped := 0
	for shipped < t.maxLines {
		lineStart := cursor
		// The read error is deliberately ignored: io.EOF is the ordinary way a
		// read stops at the end of the file, and the suffix check below is what
		// decides whether the bytes form a whole line.
		raw, _ := reader.ReadString('\n')
		if !strings.HasSuffix(raw, "\n") {
			// Either the file ended or the writer is mid-line. Leaving the offset
			// at the start of this fragment is what guarantees the store only
			// ever holds whole lines: a half-written line in the table would be a
			// line that never existed.
			break
		}
		cursor += int64(len(raw))
		line := strings.TrimRight(raw, "\n")
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) != "" {
			t.sink.Enqueue(models.InstanceLogEntry{
				Source:   t.source,
				Seq:      t.generation*fileLogTailerSeqScale + lineStart,
				LoggedAt: parseInstanceLogTime(line, now),
				Line:     line,
			})
			shipped++
		}
	}
	t.offset = cursor
	if shipped > 0 {
		t.shipped.Add(int64(shipped))
	}
	return shipped, nil
}

func (t *FileLogTailer) logReadFailure(err error) {
	failures := t.readFailures.Add(1)
	now := time.Now().UnixNano()
	if now-t.lastReadFailureLog.Load() < int64(fileLogTailerFailureLogInterval) {
		return
	}
	t.lastReadFailureLog.Store(now)
	log.Printf("component=file_log_tailer event=read_failed source=%s path=%q failures=%d error=%q",
		t.source, t.path, failures, err.Error())
}
