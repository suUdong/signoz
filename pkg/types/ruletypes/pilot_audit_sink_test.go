package ruletypes

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

type recordingPilotAuditSink struct {
	mu     sync.Mutex
	events []PilotAuditEvent
	err    error
}

func (s *recordingPilotAuditSink) Record(_ context.Context, event PilotAuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return s.err
}

func (s *recordingPilotAuditSink) Events() []PilotAuditEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]PilotAuditEvent, len(s.events))
	copy(cp, s.events)
	return cp
}

func TestPilotAuditSinkDefaultsToNop(t *testing.T) {
	resetPilotAuditSink(t)

	sink := CurrentPilotAuditEventSink()
	_, ok := sink.(NopPilotAuditEventSink)
	require.True(t, ok, "default sink should be NopPilotAuditEventSink, got %T", sink)

	require.NoError(t, DispatchPilotAuditEvent(context.Background(), pilotAuditSinkTestEvent()))
}

func TestPilotAuditSinkRegisterAndDispatch(t *testing.T) {
	resetPilotAuditSink(t)

	recorder := &recordingPilotAuditSink{}
	RegisterPilotAuditEventSink(recorder)

	require.Same(t, recorder, CurrentPilotAuditEventSink())

	event := pilotAuditSinkTestEvent()
	require.NoError(t, DispatchPilotAuditEvent(context.Background(), event))

	got := recorder.Events()
	require.Len(t, got, 1)
	require.Equal(t, event.EventID, got[0].EventID)
	require.Equal(t, event.Outcome, got[0].Outcome)
}

func TestPilotAuditSinkPropagatesError(t *testing.T) {
	resetPilotAuditSink(t)

	wantErr := errors.New("sink down")
	RegisterPilotAuditEventSink(&recordingPilotAuditSink{err: wantErr})

	gotErr := DispatchPilotAuditEvent(context.Background(), pilotAuditSinkTestEvent())
	require.ErrorIs(t, gotErr, wantErr)
}

func TestPilotAuditSinkNilResetsToNop(t *testing.T) {
	resetPilotAuditSink(t)

	RegisterPilotAuditEventSink(&recordingPilotAuditSink{})
	RegisterPilotAuditEventSink(nil)

	_, ok := CurrentPilotAuditEventSink().(NopPilotAuditEventSink)
	require.True(t, ok, "passing nil should reset to NopPilotAuditEventSink")
}

func TestPilotAuditSinkIsConcurrencySafe(t *testing.T) {
	resetPilotAuditSink(t)

	recorder := &recordingPilotAuditSink{}
	RegisterPilotAuditEventSink(recorder)

	const workers = 16
	const perWorker = 32

	var counter atomic.Int64
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				if err := DispatchPilotAuditEvent(context.Background(), pilotAuditSinkTestEvent()); err == nil {
					counter.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	require.Equal(t, int64(workers*perWorker), counter.Load())
	require.Len(t, recorder.Events(), workers*perWorker)
}

func resetPilotAuditSink(t *testing.T) {
	t.Helper()
	RegisterPilotAuditEventSink(nil)
	t.Cleanup(func() { RegisterPilotAuditEventSink(nil) })
}

func pilotAuditSinkTestEvent() PilotAuditEvent {
	return PilotAuditEvent{
		ContractVersion: PilotAuditEventContractVersion,
		EventID:         "audit-test-001",
		EventType:       PilotAuditEventTypeSOPFetch,
		OccurredAt:      "2026-05-02T00:00:00Z",
		Actor: PilotAuditActor{
			Kind: PilotAuditActorKindUser,
			ID:   "tester",
		},
		Tenant: PilotAuditTenant{
			ProjectID:   "customer-a",
			Environment: "prod",
		},
		Resource: PilotAuditResource{
			Kind:     "sop_source",
			SourceID: "src-managed-markdown-default",
			SOPID:    "SOP-PAY-001",
		},
		Action:  "fetch",
		Outcome: PilotAuditOutcomeAllowed,
		SecurityContext: PilotAuditSecurityContext{
			ServiceAccountProfile: "ds-sop-reader",
			RedactionApplied:      true,
		},
	}
}
