package ruletypes

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap" //nolint:depguard
)

// DefaultPilotAuditJSONLMaxSizeBytes is the production default rotation threshold (50 MiB).
const DefaultPilotAuditJSONLMaxSizeBytes int64 = 50 * 1024 * 1024

// PilotAuditEventJSONLSink writes pilot audit events as newline-delimited JSON
// to a local file, rotating when the file exceeds maxSizeBytes.
type PilotAuditEventJSONLSink struct {
	mu           sync.Mutex
	path         string
	maxSizeBytes int64
}

// NewPilotAuditEventJSONLSink constructs a sink that appends events to path.
// maxSizeBytes controls the rotation threshold; pass DefaultPilotAuditJSONLMaxSizeBytes
// for the 50 MiB production default.
func NewPilotAuditEventJSONLSink(path string, maxSizeBytes int64) (*PilotAuditEventJSONLSink, error) {
	if path == "" {
		return nil, fmt.Errorf("pilot audit JSONL sink: path must not be empty")
	}
	if maxSizeBytes <= 0 {
		return nil, fmt.Errorf("pilot audit JSONL sink: maxSizeBytes must be positive, got %d", maxSizeBytes)
	}
	return &PilotAuditEventJSONLSink{
		path:         path,
		maxSizeBytes: maxSizeBytes,
	}, nil
}

// Record validates event, then appends it as a JSONL line, rotating the file
// if the write would exceed maxSizeBytes. Validation failures are dropped with
// a warning log; they do not propagate as errors so callers are never blocked.
func (s *PilotAuditEventJSONLSink) Record(_ context.Context, event PilotAuditEvent) error {
	if err := ValidatePilotAuditEvent(event); err != nil {
		zap.L().Warn("pilot audit event invalid; dropping", zap.Error(err)) //nolint:depguard
		return nil
	}

	line, err := s.marshalLine(event)
	if err != nil {
		return fmt.Errorf("pilot audit JSONL sink: marshal: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("pilot audit JSONL sink: mkdirall: %w", err)
	}

	if err := s.maybeRotate(int64(len(line))); err != nil {
		return fmt.Errorf("pilot audit JSONL sink: rotate: %w", err)
	}

	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("pilot audit JSONL sink: open: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(line); err != nil {
		return fmt.Errorf("pilot audit JSONL sink: write: %w", err)
	}
	return nil
}

// marshalLine encodes event as a JSON object followed by a newline.
func (s *PilotAuditEventJSONLSink) marshalLine(event PilotAuditEvent) ([]byte, error) {
	b, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// maybeRotate renames the active file to a timestamped backup when the file
// already has content and adding incoming bytes would exceed maxSizeBytes.
// A fresh empty file is never rotated; it always accepts the next write.
// Must be called under mu.
func (s *PilotAuditEventJSONLSink) maybeRotate(incoming int64) error {
	info, err := os.Stat(s.path)
	if os.IsNotExist(err) {
		return nil // file does not exist yet, nothing to rotate
	}
	if err != nil {
		return err
	}
	current := info.Size()
	if current == 0 {
		return nil // empty file: always accept the first write
	}
	if current+incoming <= s.maxSizeBytes {
		return nil
	}
	rotated := fmt.Sprintf("%s.%s", s.path, time.Now().UTC().Format("20060102T150405.000000000Z"))
	return os.Rename(s.path, rotated)
}
