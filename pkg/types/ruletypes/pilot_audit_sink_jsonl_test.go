package ruletypes

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// jsonlValidTestEvent returns a PilotAuditEvent that passes ValidatePilotAuditEvent.
// pilotAuditSinkTestEvent (shared helper) omits RequestContext fields required by the
// validator, so JSONL-sink tests use this local helper instead.
func jsonlValidTestEvent() PilotAuditEvent {
	e := pilotAuditSinkTestEvent()
	e.RequestContext = PilotAuditRequestContext{
		IncidentID:  "INC-001",
		ServiceName: "test-service",
	}
	return e
}

// TestJSONLSinkValidEventAppendsLine verifies that a valid event grows the file
// and produces a valid JSON line.
func TestJSONLSinkValidEventAppendsLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")

	sink, err := NewPilotAuditEventJSONLSink(path, DefaultPilotAuditJSONLMaxSizeBytes)
	require.NoError(t, err)

	event := jsonlValidTestEvent()
	require.NoError(t, sink.Record(context.Background(), event))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Greater(t, len(data), 0, "file must be non-empty after valid event")

	var got PilotAuditEvent
	require.NoError(t, json.Unmarshal(data[:len(data)-1], &got), "line must be valid JSON")
	require.Equal(t, event.EventID, got.EventID)
}

func TestNewPilotAuditEventJSONLSinkRejectsInvalidConfig(t *testing.T) {
	_, err := NewPilotAuditEventJSONLSink("", DefaultPilotAuditJSONLMaxSizeBytes)
	require.Error(t, err)

	_, err = NewPilotAuditEventJSONLSink(filepath.Join(t.TempDir(), "audit.jsonl"), 0)
	require.Error(t, err)
}

// TestJSONLSinkInvalidEventDropped verifies that an invalid event does not
// alter the file (validator gate fires before any append).
func TestJSONLSinkInvalidEventDropped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")

	// Pre-create the file with known content so we can measure its size.
	require.NoError(t, os.WriteFile(path, []byte("existing\n"), 0o644))

	info, err := os.Stat(path)
	require.NoError(t, err)
	sizeBefore := info.Size()

	sink, err := NewPilotAuditEventJSONLSink(path, DefaultPilotAuditJSONLMaxSizeBytes)
	require.NoError(t, err)

	// Empty event is invalid (missing required fields).
	require.NoError(t, sink.Record(context.Background(), PilotAuditEvent{}))

	info, err = os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, sizeBefore, info.Size(), "file size must not change for invalid event")
}

// TestJSONLSinkRotation verifies that when a write would exceed maxSizeBytes
// the existing file is renamed and a fresh file is opened.
func TestJSONLSinkRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")

	// 256 bytes is small enough that a few events will trigger rotation.
	sink, err := NewPilotAuditEventJSONLSink(path, 256)
	require.NoError(t, err)

	event := jsonlValidTestEvent()
	// Write enough events to cross the threshold.
	for i := 0; i < 10; i++ {
		require.NoError(t, sink.Record(context.Background(), event))
	}

	// At least one rotated sibling must exist.
	matches, err := filepath.Glob(path + ".*")
	require.NoError(t, err)
	require.NotEmpty(t, matches, "expected at least one rotated file (e.g. audit.jsonl.<ts>)")

	// The primary file must contain fewer bytes than all 10 events combined,
	// proving rotation happened (not all events accumulated in one file).
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Less(t, info.Size(), int64(10*1024), "primary file should hold far fewer events than the total written")
}

// TestJSONLSinkConcurrency registers the sink via RegisterPilotAuditEventSink,
// then runs 64 goroutines x 32 dispatches each through DispatchPilotAuditEvent,
// and asserts 2048 valid JSON lines with no truncation.
func TestJSONLSinkConcurrency(t *testing.T) {
	resetPilotAuditSink(t)

	path := filepath.Join(t.TempDir(), "audit.jsonl")
	sink, err := NewPilotAuditEventJSONLSink(path, DefaultPilotAuditJSONLMaxSizeBytes)
	require.NoError(t, err)

	RegisterPilotAuditEventSink(sink)

	const workers = 64
	const perWorker = 32

	event := jsonlValidTestEvent()
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				_ = DispatchPilotAuditEvent(context.Background(), event)
			}
		}()
	}
	wg.Wait()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	// Split into non-empty lines and validate each as JSON.
	lines := splitLines(data)
	require.Equal(t, workers*perWorker, len(lines), "expected 2048 lines total")

	for i, line := range lines {
		var v map[string]any
		require.NoError(t, json.Unmarshal(line, &v), "line %d must be valid JSON", i)
	}
}

// splitLines splits newline-delimited bytes into non-empty line slices.
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
