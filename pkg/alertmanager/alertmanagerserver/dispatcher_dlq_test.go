package alertmanagerserver

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/alertmanager/alertmanagernotify/dlq"
	"github.com/SigNoz/signoz/pkg/alertmanager/nfmanager/nfmanagertest"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/valuer"

	"github.com/prometheus/alertmanager/dispatch"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/provider/mem"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/prometheus/common/promslog"

	"github.com/stretchr/testify/require"
)

// fakeDLQSink is an in-memory dlq.Sink used to assert that the dispatcher
// records terminal stage failures.
type fakeDLQSink struct {
	mu      sync.Mutex
	entries []*dlq.Entry
}

func (f *fakeDLQSink) Write(e *dlq.Entry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	// copy to defend against accidental mutation by the caller
	cp := *e
	f.entries = append(f.entries, &cp)
	return nil
}

func (f *fakeDLQSink) snapshot() []*dlq.Entry {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*dlq.Entry, len(f.entries))
	copy(out, f.entries)
	return out
}

// failingStage is a notify.Stage that always returns an error so that the
// dispatcher's terminal-failure branch executes.
type failingStage struct {
	err error
}

func (s *failingStage) Exec(ctx context.Context, _ *slog.Logger, alerts ...*alertmanagertypes.Alert) (context.Context, []*alertmanagertypes.Alert, error) {
	return ctx, alerts, s.err
}

func TestDispatcherWritesDLQOnTerminalFailure(t *testing.T) {
	logger := promslog.NewNopLogger()
	marker := alertmanagertypes.NewMarker(prometheus.NewRegistry())

	alertsProvider, err := mem.NewAlerts(context.Background(), marker, time.Hour, 0, nil, logger, prometheus.NewRegistry(), nil)
	require.NoError(t, err)
	defer alertsProvider.Close()

	timeout := func(d time.Duration) time.Duration { return d }
	metrics := NewDispatcherMetrics(false, prometheus.NewRegistry())
	nfManager := nfmanagertest.NewMock()

	orgID := "test-org"
	// ruleId-HighLatency is one of the few expression literals the mock
	// notification manager knows how to evaluate (see evaluateExpr).
	ruleID := "ruleId-HighLatency"
	receiver := "broken-receiver"

	nfManager.SetMockConfig(orgID, ruleID, &alertmanagertypes.NotificationConfig{
		Renotify: alertmanagertypes.ReNotificationConfig{
			RenotifyInterval: time.Hour,
		},
		NotificationGroup: map[model.LabelName]struct{}{
			"ruleId": {},
		},
	})
	// Register a route whose expression the mock recognizes, pointing at
	// our failing receiver so Match() returns [receiver].
	nfManager.SetMockRoute(orgID, &alertmanagertypes.RoutePolicy{
		Identifiable: types.Identifiable{ID: valuer.GenerateUUID()},
		Expression:   `ruleId == "ruleId-HighLatency"`,
		Name:         ruleID,
		Enabled:      true,
		OrgID:        orgID,
		Channels:     []string{receiver},
	})

	// route is the top-level routing tree; the per-receiver route used
	// for delivery is created lazily inside processAlert.
	route := &dispatch.Route{
		RouteOpts: dispatch.RouteOpts{
			Receiver:      receiver,
			GroupWait:     time.Millisecond,
			GroupInterval: time.Millisecond,
			GroupBy:       map[model.LabelName]struct{}{"ruleId": {}},
		},
	}

	stage := &failingStage{err: errors.New("downstream notify exploded")}
	sink := &fakeDLQSink{}

	dispatcher := NewDispatcher(alertsProvider, route, stage, marker, timeout, nil, logger, metrics, nfManager, orgID, sink)
	go dispatcher.Run()
	defer dispatcher.Stop()

	a := newAlert(model.LabelSet{
		"ruleId":    model.LabelValue(ruleID),
		"alertname": "boom",
	})
	require.NoError(t, alertsProvider.Put(context.Background(), a))

	// Wait for the failing stage to execute and the dispatcher to record
	// the DLQ entry. The aggrGroup flush is driven by the 1ms timers above.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(sink.snapshot()) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	got := sink.snapshot()
	require.NotEmpty(t, got, "expected at least one DLQ entry on terminal failure")

	e := got[0]
	require.NotEmpty(t, e.EventID, "EventID must be populated from the alert fingerprint")
	require.Equal(t, receiver, e.Channel, "Channel must match the receiver name from the notify context")
	require.Contains(t, e.Reason, "downstream notify exploded", "Reason must capture the stage error")
	require.NotZero(t, e.FailedAt, "FailedAt must be stamped")
	require.NotEmpty(t, e.Payload, "Payload must contain the marshalled alerts batch")
}

func TestDispatcherNilDLQSinkIsSafe(t *testing.T) {
	// A nil sink must not panic and must not change existing behavior.
	logger := promslog.NewNopLogger()
	marker := alertmanagertypes.NewMarker(prometheus.NewRegistry())

	alertsProvider, err := mem.NewAlerts(context.Background(), marker, time.Hour, 0, nil, logger, prometheus.NewRegistry(), nil)
	require.NoError(t, err)
	defer alertsProvider.Close()

	timeout := func(d time.Duration) time.Duration { return d }
	metrics := NewDispatcherMetrics(false, prometheus.NewRegistry())
	nfManager := nfmanagertest.NewMock()

	stage := &failingStage{err: errors.New("boom")}

	dispatcher := NewDispatcher(alertsProvider, nil, stage, marker, timeout, nil, logger, metrics, nfManager, "test-org", nil)
	go dispatcher.Run()
	dispatcher.Stop()
}

// ensure failingStage satisfies notify.Stage
var _ notify.Stage = (*failingStage)(nil)
