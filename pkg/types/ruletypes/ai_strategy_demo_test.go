package ruletypes

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	amtemplate "github.com/prometheus/alertmanager/template"
	"github.com/stretchr/testify/require"
)

type dsAISOPDemoSeed struct {
	SOPDocument  SOPDocument     `json:"sopDocument"`
	Alert        dsDemoAlert     `json:"alert"`
	EvidenceRefs []AIEvidenceRef `json:"evidenceRefs"`
}

type dsDemoAlert struct {
	IncidentID  string            `json:"incidentId"`
	Fingerprint string            `json:"fingerprint"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

func TestDemoSeedGeneratesSOPGroundedNotificationStrategy(t *testing.T) {
	seed := loadDSAISOPDemoSeed(t)

	require.NoError(t, ValidateSOPDocument(seed.SOPDocument))
	binding, err := PreviewSOPDocumentBinding([]SOPDocument{seed.SOPDocument}, SOPBindingPreviewRequest{
		Labels:      seed.Alert.Labels,
		Annotations: seed.Alert.Annotations,
	})
	require.NoError(t, err)
	require.Equal(t, SOPBindingStatusBound, binding.Status)
	require.Equal(t, SOPBindingResolutionExplicitLabel, binding.Resolution)
	require.Equal(t, seed.SOPDocument.SOPID, binding.SOPID)

	strategy, err := GenerateLocalAIStrategy(AIStrategyRequest{
		IncidentID:       seed.Alert.IncidentID,
		AlertFingerprint: seed.Alert.Fingerprint,
		Labels:           seed.Alert.Labels,
		Annotations:      seed.Alert.Annotations,
		SOPDocument:      seed.SOPDocument,
		EvidenceRefs:     seed.EvidenceRefs,
		GeneratedAt:      "2026-05-12T00:00:00Z",
	})
	require.NoError(t, err)
	require.Equal(t, AIStrategyStatusReady, strategy.Status)
	require.Equal(t, seed.SOPDocument.SOPID, strategy.SOPID)
	require.Contains(t, strategy.FirstActions[0].Text, "결제 성공률 dashboard")
	require.Equal(t, []string{"metric:error_rate:payment-api", "trace:pg-timeout:sample"}, strategy.Hypotheses[0].EvidenceRefs)

	annotations := mergeStringMaps(seed.Alert.Annotations, AIStrategyIncidentAnnotations(strategy))
	annotations[alertmanagertypes.IncidentAnnotationSopURL] = seed.SOPDocument.DisplayURL
	annotations[alertmanagertypes.IncidentAnnotationSopSource] = seed.SOPDocument.Source.SourceID
	annotations[alertmanagertypes.IncidentAnnotationSopTitle] = binding.Title
	annotations[alertmanagertypes.IncidentAnnotationSopVersion] = binding.Version
	annotations[alertmanagertypes.IncidentAnnotationSopBindingID] = binding.Resolution

	rendered := renderDSAISOPDemoNotification(t, seed.Alert.Labels, annotations)
	require.Contains(t, rendered, "AI ready/medium")
	require.Contains(t, rendered, "SOP SOP-PAY-001 Payment API 5xx response v2026-05-12.1")
	require.Contains(t, rendered, "payment-api critical 알림은 SOP SOP-PAY-001 기준")
	require.Contains(t, rendered, "결제 성공률 dashboard")
	require.Contains(t, rendered, "metric:error_rate:payment-api, trace:pg-timeout:sample")

	incident := alertmanagertypes.BuildIncidentInfo(toTemplateKV(seed.Alert.Labels), toTemplateKV(annotations))
	require.Equal(t, seed.Alert.Labels[alertmanagertypes.IncidentLabelServiceName], incident.ServiceName)
	require.Equal(t, seed.SOPDocument.SOPID, incident.SopID)
	require.Equal(t, seed.SOPDocument.Title, incident.SopTitle)
	require.Equal(t, strategy.StrategyID, incident.AIStrategyID)
	require.Equal(t, AIStrategyStatusReady, incident.AIStrategyStatus)
	require.Equal(t, strategy.Headline, incident.AIHeadline)
	require.Contains(t, incident.AIFirstActions, "결제 성공률 dashboard")
	require.Equal(t, "metric:error_rate:payment-api, trace:pg-timeout:sample", incident.AIEvidenceRefs)
}

func loadDSAISOPDemoSeed(t *testing.T) dsAISOPDemoSeed {
	t.Helper()

	raw, err := os.ReadFile("testdata/ds_ai_sop_demo_seed.json")
	require.NoError(t, err)

	var seed dsAISOPDemoSeed
	require.NoError(t, json.Unmarshal(raw, &seed))
	return seed
}

func renderDSAISOPDemoNotification(t *testing.T, labels map[string]string, annotations map[string]string) string {
	t.Helper()

	defs := "{{$labels := .Labels}}{{$value := .Value}}{{$threshold := .Threshold}}"
	data := AlertTemplateDataWithIncident(labels, annotations, "12", "5")
	expander := NewTemplateExpander(
		context.Background(),
		defs+"AI $incident.ai_strategy_status/$incident.ai_confidence\nSOP $incident.sop_id $incident.sop_title v$incident.sop_version\n$incident.ai_headline\n$incident.ai_first_actions\nEvidence: $incident.ai_evidence_refs",
		"ds-ai-sop-demo",
		data,
		nil,
	)
	rendered, err := expander.Expand()
	require.NoError(t, err)
	return rendered
}

func mergeStringMaps(base map[string]string, overlays ...map[string]string) map[string]string {
	merged := make(map[string]string, len(base))
	for key, value := range base {
		merged[key] = value
	}
	for _, overlay := range overlays {
		for key, value := range overlay {
			merged[key] = value
		}
	}

	return merged
}

func toTemplateKV(values map[string]string) amtemplate.KV {
	kv := make(amtemplate.KV, len(values))
	for key, value := range values {
		kv[key] = value
	}

	return kv
}
