package ruletypes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTemplateExpander(t *testing.T) {
	defs := "{{$labels := .Labels}}{{$value := .Value}}{{$threshold := .Threshold}}"
	data := AlertTemplateData(map[string]string{"service.name": "my-service"}, "100", "200")
	expander := NewTemplateExpander(context.Background(), defs+"test $service.name", "test", data, nil)
	result, err := expander.Expand()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, "test my-service", result)
}

func TestTemplateExpander_WithThreshold(t *testing.T) {
	defs := "{{$labels := .Labels}}{{$value := .Value}}{{$threshold := .Threshold}}"
	data := AlertTemplateData(map[string]string{"service.name": "my-service"}, "200", "100")
	expander := NewTemplateExpander(context.Background(), defs+"test $service.name exceeds {{$threshold}} and observed at {{$value}}", "test", data, nil)
	result, err := expander.Expand()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, "test my-service exceeds 100 and observed at 200", result)
}

func TestTemplateExpanderOldVariableSyntax(t *testing.T) {
	defs := "{{$labels := .Labels}}{{$value := .Value}}{{$threshold := .Threshold}}"
	data := AlertTemplateData(map[string]string{"service.name": "my-service"}, "200", "100")
	expander := NewTemplateExpander(context.Background(), defs+"test {{.Labels.service_name}} exceeds {{$threshold}} and observed at {{$value}}", "test", data, nil)
	result, err := expander.Expand()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, "test my-service exceeds 100 and observed at 200", result)
}

func TestTemplateExpander_WithAlreadyNormalizedKey(t *testing.T) {
	defs := "{{$labels := .Labels}}{{$value := .Value}}{{$threshold := .Threshold}}"
	data := AlertTemplateData(map[string]string{"service_name": "my-service"}, "200", "100")
	expander := NewTemplateExpander(context.Background(), defs+"test {{.Labels.service_name}} exceeds {{$threshold}} and observed at {{$value}}", "test", data, nil)
	result, err := expander.Expand()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, "test my-service exceeds 100 and observed at 200", result)
}

func TestTemplateExpander_WithMissingKey(t *testing.T) {
	defs := "{{$labels := .Labels}}{{$value := .Value}}{{$threshold := .Threshold}}"
	data := AlertTemplateData(map[string]string{"service_name": "my-service"}, "200", "100")
	expander := NewTemplateExpander(context.Background(), defs+"test {{.Labels.missing_key}} exceeds {{$threshold}} and observed at {{$value}}", "test", data, nil)
	result, err := expander.Expand()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, "test  exceeds 100 and observed at 200", result)
}

func TestTemplateExpander_WithLablesDotSyntax(t *testing.T) {
	defs := "{{$labels := .Labels}}{{$value := .Value}}{{$threshold := .Threshold}}"
	data := AlertTemplateData(map[string]string{"service.name": "my-service"}, "200", "100")
	expander := NewTemplateExpander(context.Background(), defs+"test {{.Labels.service.name}} exceeds {{$threshold}} and observed at {{$value}}", "test", data, nil)
	result, err := expander.Expand()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, "test my-service exceeds 100 and observed at 200", result)
}

func TestTemplateExpander_WithVariableSyntax(t *testing.T) {
	defs := "{{$labels := .Labels}}{{$value := .Value}}{{$threshold := .Threshold}}"
	data := AlertTemplateData(map[string]string{"service.name": "my-service"}, "200", "100")
	expander := NewTemplateExpander(context.Background(), defs+"test {{$service.name}} exceeds {{$threshold}} and observed at {{$value}}", "test", data, nil)
	result, err := expander.Expand()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, "test my-service exceeds 100 and observed at 200", result)
}

func TestTemplateExpander_WithIncidentSyntax(t *testing.T) {
	defs := "{{$labels := .Labels}}{{$value := .Value}}{{$threshold := .Threshold}}"
	data := AlertTemplateDataWithIncident(
		map[string]string{
			"project_id":   "customer-a",
			"service.name": "checkout-api",
			"owner_team":   "sm-payments",
			"severity":     "critical",
			"sop_id":       "SOP-PAY-001",
		},
		map[string]string{
			"impact_summary":     "Checkout latency can affect customer payments.",
			"next_action":        "Ask vendor to inspect slow traces.",
			"sop_source":         "confluence",
			"sop_title":          "Payment API 5xx response",
			"sop_url":            "https://runbooks.example.com/payment-latency",
			"ai_strategy_status": "ready",
			"ai_headline":        "SOP 기준 결제 지연 확인이 필요합니다.",
			"ai_first_actions":   "PG timeout 로그를 확인",
			"ai_confidence":      "medium",
		},
		"200",
		"100",
	)
	expander := NewTemplateExpander(
		context.Background(),
		defs+"[$incident.project_id][$incident.service_name][$incident.sop_id] $incident.impact_summary Next: {{$incident.next_action}} SOP: $incident.sop_title <$incident.sop_url> Source: $incident.sop_source AI: $incident.ai_strategy_status/$incident.ai_confidence $incident.ai_headline Actions: $incident.ai_first_actions",
		"test",
		data,
		nil,
	)
	result, err := expander.Expand()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, "[customer-a][checkout-api][SOP-PAY-001] Checkout latency can affect customer payments. Next: Ask vendor to inspect slow traces. SOP: Payment API 5xx response <https://runbooks.example.com/payment-latency> Source: confluence AI: ready/medium SOP 기준 결제 지연 확인이 필요합니다. Actions: PG timeout 로그를 확인", result)
}
