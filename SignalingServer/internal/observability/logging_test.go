package observability

import (
	"bytes"
	"errors"
	"testing"
)

// resetSinks clears the package-level sink list for the duration of a test, so
// attaching one does not leak into another.
func resetSinks(t *testing.T) {
	t.Helper()
	sinksMu.Lock()
	saved := sinks
	sinks = nil
	sinksMu.Unlock()
	t.Cleanup(func() {
		sinksMu.Lock()
		sinks = saved
		sinksMu.Unlock()
	})
}

// A sink attached after Configure must see the same bytes the file gets. That
// is the whole contract the log-shipping feature is built on.
func TestSinkFanoutMirrorsLinesToAttachedSinks(t *testing.T) {
	resetSinks(t)
	var primary, first, second bytes.Buffer
	AttachLogSink(&first)
	AttachLogSink(&second)

	line := "2026/10/16 08:00:00.123456 component=http event=request status=404\n"
	n, err := (&sinkFanout{primary: &primary}).Write([]byte(line))
	if err != nil || n != len(line) {
		t.Fatalf("write = %d,%v; want %d,nil", n, err, len(line))
	}
	for name, got := range map[string]string{
		"primary": primary.String(),
		"first":   first.String(),
		"second":  second.String(),
	} {
		if got != line {
			t.Fatalf("%s received %q, want %q", name, got, line)
		}
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("store is down") }

// A sink is a copy, so a broken store must not turn into a failed log call: the
// caller's line is already on stdout and in the rotating file.
func TestSinkFanoutSwallowsSinkErrors(t *testing.T) {
	resetSinks(t)
	AttachLogSink(brokenWriter{})
	var good bytes.Buffer
	AttachLogSink(&good)

	line := "component=instance_log_sink event=store_failed\n"
	n, err := (&sinkFanout{primary: &bytes.Buffer{}}).Write([]byte(line))
	if err != nil {
		t.Fatalf("write returned %v; a failing sink must not fail logging", err)
	}
	if n != len(line) {
		t.Fatalf("write = %d, want %d", n, len(line))
	}
	if good.String() != line {
		t.Fatalf("the healthy sink received %q, want %q - one broken sink must not stop the others", good.String(), line)
	}
}

func TestAttachLogSinkIgnoresNil(t *testing.T) {
	resetSinks(t)
	AttachLogSink(nil)
	sinksMu.RLock()
	defer sinksMu.RUnlock()
	if len(sinks) != 0 {
		t.Fatalf("attached sinks = %d, want 0: a nil sink is a no-op", len(sinks))
	}
}
