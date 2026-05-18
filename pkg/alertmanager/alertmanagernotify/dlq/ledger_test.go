package dlq

import (
	"path/filepath"
	"testing"
)

func TestReplayLedgerIsIdempotent(t *testing.T) {
	led, err := NewReplayLedger(filepath.Join(t.TempDir(), "ledger.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer led.Close()
	if !led.MarkIfNew("evt-1") {
		t.Fatal("first mark should succeed")
	}
	if led.MarkIfNew("evt-1") {
		t.Fatal("second mark should report not new")
	}
	if !led.MarkIfNew("evt-2") {
		t.Fatal("different event should succeed")
	}
}

func TestReplayLedgerSurvivesReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ledger.jsonl")
	led1, err := NewReplayLedger(path)
	if err != nil {
		t.Fatal(err)
	}
	if !led1.MarkIfNew("evt-1") {
		t.Fatal("first mark should succeed")
	}
	if err := led1.Close(); err != nil {
		t.Fatal(err)
	}
	led2, err := NewReplayLedger(path)
	if err != nil {
		t.Fatal(err)
	}
	defer led2.Close()
	if led2.MarkIfNew("evt-1") {
		t.Fatal("reopened ledger should remember evt-1")
	}
	if !led2.MarkIfNew("evt-3") {
		t.Fatal("new event after reopen should succeed")
	}
}

func TestReplayLedgerRejectsEmptyEventID(t *testing.T) {
	led, err := NewReplayLedger(filepath.Join(t.TempDir(), "ledger.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer led.Close()
	if led.MarkIfNew("") {
		t.Fatal("empty event ID must not be marked new")
	}
}
