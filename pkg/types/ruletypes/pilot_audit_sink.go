package ruletypes

import (
	"context"
	"sync"
)

// PilotAuditEventSink is the hook contract for delivering DS-APM pilot audit
// events to a downstream collector. The pilot scope treats delivery as
// best-effort: persistence can be wired by the operator surface, but retry
// policy remains outside the request path.
type PilotAuditEventSink interface {
	Record(ctx context.Context, event PilotAuditEvent) error
}

// NopPilotAuditEventSink is the default sink. It accepts any event and
// returns nil without persisting anything.
type NopPilotAuditEventSink struct{}

func (NopPilotAuditEventSink) Record(_ context.Context, _ PilotAuditEvent) error {
	return nil
}

var (
	pilotAuditSinkMu   sync.RWMutex
	pilotAuditSinkImpl PilotAuditEventSink = NopPilotAuditEventSink{}
)

// RegisterPilotAuditEventSink installs a sink. Passing nil resets to the
// no-op default. Safe for concurrent use.
func RegisterPilotAuditEventSink(sink PilotAuditEventSink) {
	pilotAuditSinkMu.Lock()
	defer pilotAuditSinkMu.Unlock()
	if sink == nil {
		pilotAuditSinkImpl = NopPilotAuditEventSink{}
		return
	}
	pilotAuditSinkImpl = sink
}

// CurrentPilotAuditEventSink returns the registered sink.
func CurrentPilotAuditEventSink() PilotAuditEventSink {
	pilotAuditSinkMu.RLock()
	defer pilotAuditSinkMu.RUnlock()
	return pilotAuditSinkImpl
}

// DispatchPilotAuditEvent forwards a validated audit event to the registered
// sink. Validation is the caller's responsibility. Pilot-scope callers treat
// sink errors as best-effort and do not fail the originating operation.
func DispatchPilotAuditEvent(ctx context.Context, event PilotAuditEvent) error {
	pilotAuditSinkMu.RLock()
	sink := pilotAuditSinkImpl
	pilotAuditSinkMu.RUnlock()
	return sink.Record(ctx, event)
}
