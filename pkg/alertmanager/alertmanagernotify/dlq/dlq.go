// Package dlq provides a JSONL dead-letter sink and an idempotent replay
// ledger for alertmanager notification failures.
//
// It mirrors the durability guarantee previously provided by the Python
// orchestrator's RetryingSink/DLQSink/replay_dlq trio: terminal notify
// failures are persisted to disk so they survive process restarts and can
// be replayed without double-delivery.
//
// The JSONL rotation logic intentionally follows the style established by
// pkg/types/ruletypes/pilot_audit_sink_jsonl.go (50 MiB default, mutex-
// guarded append, timestamped rotated siblings). The two sinks remain
// independent for now; a shared rotation helper would be a sensible
// follow-up once a third consumer appears.
package dlq

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DefaultJSONLDeadLetterMaxSizeBytes is the production default rotation
// threshold (50 MiB), matching the pilot audit sink convention.
const DefaultJSONLDeadLetterMaxSizeBytes int64 = 50 * 1024 * 1024

// Entry is a single terminal notification failure recorded to disk. The
// schema is intentionally narrow: enough metadata to identify the event,
// the channel that failed, the payload that would have been delivered,
// and a textual reason for forensics. Anything richer (full alert state,
// stack traces) belongs in logs, not the replayable dead-letter store.
type Entry struct {
	EventID  string    `json:"event_id"`
	Channel  string    `json:"channel"`
	Payload  []byte    `json:"payload"`
	FailedAt time.Time `json:"failed_at"`
	Reason   string    `json:"reason,omitempty"`
}

// Sink is the narrow contract the alertmanager dispatcher depends on so
// terminal notify failures can be persisted to disk. JSONLDeadLetterSink
// is the production implementation; tests may provide an in-memory fake.
type Sink interface {
	Write(e *Entry) error
}

// JSONLDeadLetterSink appends Entry values as newline-delimited JSON to a
// file, rotating when the active file would exceed rotateBytes.
type JSONLDeadLetterSink struct {
	path        string
	rotateBytes int64

	mu      sync.Mutex
	f       *os.File
	written int64
}

// NewJSONLDeadLetterSink opens (creating if necessary) the sink at path
// with the given rotation threshold. The parent directory is created if
// it does not already exist. Pass DefaultJSONLDeadLetterMaxSizeBytes for
// the 50 MiB production default.
func NewJSONLDeadLetterSink(path string, rotateBytes int64) (*JSONLDeadLetterSink, error) {
	if path == "" {
		return nil, fmt.Errorf("dlq sink: path must not be empty")
	}
	if rotateBytes <= 0 {
		return nil, fmt.Errorf("dlq sink: rotateBytes must be positive, got %d", rotateBytes)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("dlq sink: mkdirall: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("dlq sink: open: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("dlq sink: stat: %w", err)
	}
	return &JSONLDeadLetterSink{
		path:        path,
		rotateBytes: rotateBytes,
		f:           f,
		written:     info.Size(),
	}, nil
}

// Write appends entry as one JSON line. If the resulting file would
// exceed the rotation threshold, the current file is rotated first.
// Concurrent Write calls are serialized.
func (s *JSONLDeadLetterSink) Write(e *Entry) error {
	if e == nil {
		return fmt.Errorf("dlq sink: entry must not be nil")
	}
	b, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("dlq sink: marshal: %w", err)
	}
	line := append(b, '\n')

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.f == nil {
		return fmt.Errorf("dlq sink: sink is closed")
	}

	// Rotate only when there is something to preserve and the incoming
	// write would push the file past the threshold. An empty file always
	// accepts the next write so we never produce zero-byte rotated files.
	if s.written > 0 && s.written+int64(len(line)) > s.rotateBytes {
		if err := s.rotateLocked(); err != nil {
			return fmt.Errorf("dlq sink: rotate: %w", err)
		}
	}

	n, err := s.f.Write(line)
	if err != nil {
		return fmt.Errorf("dlq sink: write: %w", err)
	}
	s.written += int64(n)
	return nil
}

// rotateLocked closes the active file, renames it to a timestamped
// sibling that matches the "<path>*" glob, then reopens a fresh primary
// file. Must be called under mu.
func (s *JSONLDeadLetterSink) rotateLocked() error {
	if err := s.f.Close(); err != nil {
		return fmt.Errorf("close active: %w", err)
	}
	rotated := fmt.Sprintf("%s.%s", s.path, time.Now().UTC().Format("20060102T150405.000000000Z"))
	if err := os.Rename(s.path, rotated); err != nil {
		// Best-effort: try to reopen the primary so the sink is still
		// usable even when rotation failed (e.g., disk transient).
		f, openErr := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if openErr == nil {
			s.f = f
			info, _ := f.Stat()
			if info != nil {
				s.written = info.Size()
			}
		}
		return fmt.Errorf("rename: %w", err)
	}
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("reopen primary: %w", err)
	}
	s.f = f
	s.written = 0
	return nil
}

// Close flushes and closes the active file. Subsequent Write calls
// return an error.
func (s *JSONLDeadLetterSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.f == nil {
		return nil
	}
	err := s.f.Close()
	s.f = nil
	return err
}
