// Package observability configures process logging and provides small,
// dependency-free helpers for structured production diagnostics.
package observability

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Config controls the persistent application log. Docker deployments should
// use /data/logs so logs survive image/container replacement with DB data.
type Config struct {
	Dir        string
	MaxSizeMB  int
	MaxBackups int
}

// Configure mirrors the standard library logger to stdout and a bounded,
// rotated file. Standard log.Printf callers remain supported while new code
// should prefer Event for stable key=value business diagnostics.
func Configure(cfg Config) (string, error) {
	if cfg.Dir == "" {
		cfg.Dir = "logs"
	}
	if cfg.MaxSizeMB <= 0 {
		cfg.MaxSizeMB = 50
	}
	if cfg.MaxBackups <= 0 {
		cfg.MaxBackups = 5
	}
	if err := os.MkdirAll(cfg.Dir, 0750); err != nil {
		return "", fmt.Errorf("create log directory: %w", err)
	}

	path := filepath.Join(cfg.Dir, "signaling.log")
	writer, err := newRotatingWriter(path, int64(cfg.MaxSizeMB)*1024*1024, cfg.MaxBackups)
	if err != nil {
		return "", err
	}
	log.SetOutput(io.MultiWriter(os.Stdout, writer))
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC)
	return path, nil
}

// Event writes one line with a stable component/event prefix and sorted
// key=value fields. Never pass credentials, access codes, tokens or secrets.
func Event(component, name string, fields map[string]interface{}) {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys)+2)
	parts = append(parts, "component="+component, "event="+name)
	for _, key := range keys {
		parts = append(parts, key+"="+formatValue(fields[key]))
	}
	log.Print(strings.Join(parts, " "))
}

func formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strconvQuote(v)
	case time.Duration:
		return v.String()
	default:
		body, err := json.Marshal(v)
		if err != nil {
			return strconvQuote(fmt.Sprint(v))
		}
		return string(body)
	}
}

func strconvQuote(value string) string {
	body, _ := json.Marshal(value)
	return string(body)
}

type rotatingWriter struct {
	mu         sync.Mutex
	path       string
	maxSize    int64
	maxBackups int
	file       *os.File
	size       int64
}

func newRotatingWriter(path string, maxSize int64, maxBackups int) (*rotatingWriter, error) {
	writer := &rotatingWriter{path: path, maxSize: maxSize, maxBackups: maxBackups}
	if err := writer.open(); err != nil {
		return nil, err
	}
	return writer, nil
}

func (w *rotatingWriter) open() error {
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		return fmt.Errorf("open log file %s: %w", w.path, err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return err
	}
	w.file = file
	w.size = info.Size()
	return nil
}

func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.maxSize > 0 && w.size+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *rotatingWriter) rotate() error {
	if err := w.file.Close(); err != nil {
		return err
	}
	for i := w.maxBackups - 1; i >= 1; i-- {
		old := fmt.Sprintf("%s.%d", w.path, i)
		newer := fmt.Sprintf("%s.%d", w.path, i+1)
		if i == w.maxBackups-1 {
			_ = os.Remove(newer)
		}
		if _, err := os.Stat(old); err == nil {
			if err := os.Rename(old, newer); err != nil {
				return err
			}
		}
	}
	if err := os.Rename(w.path, w.path+".1"); err != nil && !os.IsNotExist(err) {
		return err
	}
	return w.open()
}
