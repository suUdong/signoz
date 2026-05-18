package dlq

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ReplayLedger is an append-only set of event IDs that have already been
// processed by a replay loop. It is durable across restarts: on open the
// file is scanned and every non-empty line becomes a seen entry.
//
// The expected usage is: when replaying entries from a JSONLDeadLetterSink,
// call MarkIfNew(entry.EventID) and only re-deliver when it returns true.
// This guarantees idempotency even if replay crashes mid-batch.
type ReplayLedger struct {
	mu   sync.Mutex
	seen map[string]struct{}
	f    *os.File
	w    *bufio.Writer
}

// NewReplayLedger opens (creating if necessary) the ledger at path and
// rebuilds the in-memory seen set from the file's existing contents.
// The parent directory is created if it does not already exist.
func NewReplayLedger(path string) (*ReplayLedger, error) {
	if path == "" {
		return nil, fmt.Errorf("replay ledger: path must not be empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("replay ledger: mkdirall: %w", err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("replay ledger: open: %w", err)
	}

	seen := make(map[string]struct{})
	scanner := bufio.NewScanner(f)
	// Allow up to 1 MiB per line for safety. Event IDs are typically
	// short (UUID / hash) but we do not want a pathological entry to
	// silently truncate the ledger.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			seen[line] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("replay ledger: scan: %w", err)
	}

	// Seek to end so subsequent writes append, regardless of where the
	// scanner left the cursor.
	if _, err := f.Seek(0, 2); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("replay ledger: seek end: %w", err)
	}

	return &ReplayLedger{
		seen: seen,
		f:    f,
		w:    bufio.NewWriter(f),
	}, nil
}

// MarkIfNew returns true and durably records eventID when it has not
// been seen before. It returns false for empty IDs, for previously
// recorded IDs, on write failure, or when the ledger is closed.
func (l *ReplayLedger) MarkIfNew(eventID string) bool {
	if eventID == "" {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return false
	}
	if _, ok := l.seen[eventID]; ok {
		return false
	}
	if _, err := fmt.Fprintln(l.w, eventID); err != nil {
		return false
	}
	if err := l.w.Flush(); err != nil {
		return false
	}
	l.seen[eventID] = struct{}{}
	return true
}

// Has reports whether eventID is already recorded. Useful for read-only
// callers (e.g., metrics, dry-run replay).
func (l *ReplayLedger) Has(eventID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, ok := l.seen[eventID]
	return ok
}

// Close flushes pending writes and closes the underlying file. The
// ledger may not be used after Close.
func (l *ReplayLedger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return nil
	}
	flushErr := l.w.Flush()
	closeErr := l.f.Close()
	l.f = nil
	l.w = nil
	if flushErr != nil {
		return flushErr
	}
	return closeErr
}
