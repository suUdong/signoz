package ruletypes

import (
	"context"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/stretchr/testify/require"
)

func TestPreviewNotificationTemplate(t *testing.T) {
	got, err := PreviewNotificationTemplate(context.Background(), PreviewNotificationTemplateRequest{
		Template: "$incident.impact_summary Next: $incident.next_action SOP: $incident.sop_id <$incident.sop_url> Source: $incident.sop_source AI: $incident.ai_strategy_status/$incident.ai_confidence $incident.ai_headline Actions: $incident.ai_first_actions Limits: $incident.ai_limitations Evidence: $incident.ai_evidence_refs Value: {{$value}} Missing: $incident.bad_field",
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
			alertmanagertypes.IncidentAnnotationAIStrategyStatus: "ready",
			alertmanagertypes.IncidentAnnotationAIHeadline:       "SOP 기준 결제 지연 확인이 필요합니다.",
			alertmanagertypes.IncidentAnnotationAIFirstActions:   "PG timeout 로그를 확인",
			alertmanagertypes.IncidentAnnotationAIConfidence:     "medium",
			alertmanagertypes.IncidentAnnotationAILimitations:    "최근 배포 정보는 연결되지 않음",
			alertmanagertypes.IncidentAnnotationAIEvidenceRefs:   "metric:error_rate:1",
		},
		Value: "230ms",
	})
	require.NoError(t, err)

	require.Equal(t, "Checkout latency can affect customer payments. Next: Ask vendor to inspect slow traces. SOP: SOP-PAY-001 <https://runbooks.example.com/payment-latency> Source: confluence AI: ready/medium SOP 기준 결제 지연 확인이 필요합니다. Actions: PG timeout 로그를 확인 Limits: 최근 배포 정보는 연결되지 않음 Evidence: metric:error_rate:1 Value: 230ms Missing: ", got.Body)
	require.Equal(t, []string{"$incident.bad_field"}, got.MissingVars)
}

func TestPreviewNotificationTemplateFailsOpenWhenAIQuotaExhausted(t *testing.T) {
	req := validAIStrategyRequest()
	req.Controls.QuotaLimit = 1
	req.Controls.QuotaUsed = 1
	strategy, err := GenerateLocalAIStrategy(req)
	require.NoError(t, err)

	annotations := map[string]string{
		"impact_summary": "Checkout 5xx can affect customer payments.",
		"next_action":    "Use SOP-PAY-001 step 1 to inspect the payment dashboard.",
		"sop_source":     "managed-markdown",
		"sop_title":      "Payment API 5xx response",
		"sop_url":        "https://runbooks.example.com/sop-pay-001",
	}
	for key, value := range AIStrategyIncidentAnnotations(strategy) {
		annotations[key] = value
	}

	got, err := PreviewNotificationTemplate(context.Background(), PreviewNotificationTemplateRequest{
		Template: "SOP: $incident.sop_id <$incident.sop_url> Next: $incident.next_action AI: $incident.ai_strategy_status Limits: $incident.ai_limitations Actions: $incident.ai_first_actions",
		Labels: map[string]string{
			"project_id":   "customer-a",
			"environment":  "prod",
			"service.name": "checkout-api",
			"severity":     "critical",
			"sop_id":       "SOP-PAY-001",
		},
		Annotations: annotations,
	})
	require.NoError(t, err)

	require.Equal(
		t,
		"SOP: SOP-PAY-001 <https://runbooks.example.com/sop-pay-001> Next: Use SOP-PAY-001 step 1 to inspect the payment dashboard. AI: quota_exhausted Limits: AI strategy quota is exhausted for this period. Actions: ",
		got.Body,
	)
	require.Empty(t, got.MissingVars)
}

func TestMissingIncidentTemplateVariables(t *testing.T) {
	require.Equal(t,
		[]string{"$incident.bad", "$incident.unknown"},
		MissingIncidentTemplateVariables("$incident.unknown $incident.next_action $incident.bad $incident.unknown"),
	)
	require.Empty(t, MissingIncidentTemplateVariables("$incident.impact_summary $incident.service_name $incident.sop_id $incident.sop_source $incident.sop_url $incident.ai_strategy_status $incident.ai_headline $incident.ai_first_actions $incident.ai_confidence $incident.ai_limitations $incident.ai_evidence_refs"))
}
