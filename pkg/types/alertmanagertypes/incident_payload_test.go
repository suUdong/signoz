package alertmanagertypes

import (
	"strings"
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

func TestSanitizeIncidentValueRedactsEmail(t *testing.T) {
	got := SanitizeIncidentValue("contact 김철수 chulsoo@example.co.kr immediately")
	if strings.Contains(got, "chulsoo@example.co.kr") {
		t.Fatalf("email not redacted: %q", got)
	}
}

func TestSanitizeIncidentValueRedactsKoreanMobile(t *testing.T) {
	got := SanitizeIncidentValue("notify 010-1234-5678 by 5pm")
	if strings.Contains(got, "010-1234-5678") {
		t.Fatalf("KR mobile not redacted: %q", got)
	}
}

func TestSanitizeIncidentValueRedactsLongUnmarkedSecret(t *testing.T) {
	raw := "abcdefghijklmnopqrstuvwxyz0123456789" // 36 chars, no marker
	got := SanitizeIncidentValue("token leaked: " + raw)
	if strings.Contains(got, raw) {
		t.Fatalf("long secret not redacted: %q", got)
	}
}

func TestSanitizeIncidentValuePreservesShortInnocuousText(t *testing.T) {
	got := SanitizeIncidentValue("short status: degraded for 3 minutes")
	if got != "short status: degraded for 3 minutes" {
		t.Fatalf("innocuous text mutated: %q", got)
	}
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
