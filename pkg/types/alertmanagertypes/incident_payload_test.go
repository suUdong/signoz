package alertmanagertypes

import (
	"testing"

	"github.com/prometheus/alertmanager/template"
	"github.com/stretchr/testify/require"
)

func TestBuildSafeIncidentInfoRedactsSecretLikeValues(t *testing.T) {
	info := BuildSafeIncidentInfo(
		template.KV{
			IncidentLabelProjectID:   "customer-a",
			IncidentLabelEnvironment: "prod",
			IncidentLabelServiceName: "checkout-api",
			IncidentLabelSopID:       "SOP-PAY-001",
		},
		template.KV{
			IncidentAnnotationSopURL:           "https://runbooks.example.com/sop?token=hidden&view=public",
			IncidentAnnotationAIHeadline:       "bearer abcdefghijklmnopqrstuvwxyz",
			IncidentAnnotationAIFirstActions:   "Inspect PG timeout logs.",
			IncidentAnnotationAIStrategyStatus: "ready",
		},
	)

	require.Equal(t, "https://runbooks.example.com/sop?view=public", info.SopURL)
	require.Equal(t, RedactedIncidentValue, info.AIHeadline)
	require.Equal(t, "Inspect PG timeout logs.", info.AIFirstActions)

	fields := IncidentInfoFields(info)
	require.Contains(t, fields, IncidentField{
		Key:   "ai_headline",
		Title: "AI headline",
		Value: RedactedIncidentValue,
	})
	require.NotContains(t, IncidentInfoDetails(info)["sop_url"], "token=hidden")
}

func TestIncidentInfoFieldsOmitEmptyValues(t *testing.T) {
	fields := IncidentInfoFields(IncidentInfo{
		SopID:            "SOP-PAY-001",
		AIStrategyStatus: "quota_exhausted",
	})

	require.Equal(t, []IncidentField{
		{Key: "sop_id", Title: "SOP ID", Value: "SOP-PAY-001", Short: true},
		{Key: "ai_strategy_status", Title: "AI status", Value: "quota_exhausted", Short: true},
	}, fields)
}
