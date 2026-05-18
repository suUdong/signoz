package dlq

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestJSONLDeadLetterSinkAppendsAndRotates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dlq.jsonl")
	sink, err := NewJSONLDeadLetterSink(path, 64 /* tiny rotation for test */)
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()

	for i := 0; i < 5; i++ {
		e := &Entry{
			EventID:  fmt.Sprintf("evt-%d", i),
			Channel:  "slack",
			Payload:  []byte(`{"text":"hello world"}`),
			FailedAt: time.Now().UTC(),
			Reason:   "timeout",
		}
		if err := sink.Write(e); err != nil {
			t.Fatal(err)
		}
	}

	matches, err := filepath.Glob(filepath.Join(dir, "dlq.jsonl*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) < 2 {
		t.Fatalf("rotation expected, got files: %v", matches)
	}

	// Verify at least one entry is valid JSON with the expected schema
	var found bool
	for _, m := range matches {
		b, _ := os.ReadFile(m)
		for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
			if line == "" {
				continue
			}
			var got Entry
			if err := json.Unmarshal([]byte(line), &got); err == nil && got.EventID != "" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("no parseable entry across rotated files")
	}
}

func TestNewJSONLDeadLetterSinkRejectsInvalidConfig(t *testing.T) {
	if _, err := NewJSONLDeadLetterSink("", 1024); err == nil {
		t.Fatal("expected error for empty path")
	}
	if _, err := NewJSONLDeadLetterSink(filepath.Join(t.TempDir(), "dlq.jsonl"), 0); err == nil {
		t.Fatal("expected error for non-positive rotateBytes")
	}
}

func TestJSONLDeadLetterSinkConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dlq.jsonl")
	sink, err := NewJSONLDeadLetterSink(path, DefaultJSONLDeadLetterMaxSizeBytes)
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()

	const workers = 16
	const perWorker = 32

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func(w int) {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				e := &Entry{
					EventID:  fmt.Sprintf("evt-%d-%d", w, j),
					Channel:  "webhook",
					Payload:  []byte(`{"k":"v"}`),
					FailedAt: time.Now().UTC(),
					Reason:   "boom",
				}
				if err := sink.Write(e); err != nil {
					t.Errorf("write failed: %v", err)
					return
				}
			}
		}(w)
	}
	wg.Wait()

	// Every line across all (rotated + active) files must be valid JSON.
	matches, err := filepath.Glob(filepath.Join(dir, "dlq.jsonl*"))
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, m := range matches {
		b, err := os.ReadFile(m)
		if err != nil {
			t.Fatalf("read %s: %v", m, err)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
			if line == "" {
				continue
			}
			var got Entry
			if err := json.Unmarshal([]byte(line), &got); err != nil {
				t.Fatalf("invalid JSON line in %s: %v: %q", m, err, line)
			}
			total++
		}
	}
	if total != workers*perWorker {
		t.Fatalf("expected %d entries, got %d", workers*perWorker, total)
	}
}
