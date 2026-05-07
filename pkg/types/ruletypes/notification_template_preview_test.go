package ruletypes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreviewNotificationTemplate(t *testing.T) {
	got, err := PreviewNotificationTemplate(context.Background(), PreviewNotificationTemplateRequest{
		Template: "$incident.impact_summary Next: $incident.next_action SOP: $incident.sop_id <$incident.sop_url> Source: $incident.sop_source Value: {{$value}} Missing: $incident.bad_field",
		Labels: map[string]string{
			"project_id":   "customer-a",
			"service.name": "checkout-api",
			"severity":     "critical",
			"sop_id":       "SOP-PAY-001",
		},
		Annotations: map[string]string{
			"impact_summary": "Checkout latency can affect customer payments.",
			"next_action":    "Ask vendor to inspect slow traces.",
			"sop_source":     "confluence",
			"sop_url":        "https://runbooks.example.com/payment-latency",
		},
		Value: "230ms",
	})
	require.NoError(t, err)

	require.Equal(t, "Checkout latency can affect customer payments. Next: Ask vendor to inspect slow traces. SOP: SOP-PAY-001 <https://runbooks.example.com/payment-latency> Source: confluence Value: 230ms Missing: ", got.Body)
	require.Equal(t, []string{"$incident.bad_field"}, got.MissingVars)
}

func TestMissingIncidentTemplateVariables(t *testing.T) {
	require.Equal(t,
		[]string{"$incident.bad", "$incident.unknown"},
		MissingIncidentTemplateVariables("$incident.unknown $incident.next_action $incident.bad $incident.unknown"),
	)
	require.Empty(t, MissingIncidentTemplateVariables("$incident.impact_summary $incident.service_name $incident.sop_id $incident.sop_source $incident.sop_url"))
}
