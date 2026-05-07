package ruletypes

import (
	"testing"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/stretchr/testify/require"
)

func TestPreviewSOPBuildsSourceSearchAndSafePreview(t *testing.T) {
	got := PreviewSOP(PreviewSOPRequest{
		Labels: map[string]string{
			alertmanagertypes.IncidentLabelSopID:       "SOP-PAY-001",
			alertmanagertypes.IncidentLabelServiceName: "payment-api",
			alertmanagertypes.IncidentLabelEnvironment: "prod",
			alertmanagertypes.IncidentLabelSeverity:    "critical",
		},
		Annotations: map[string]string{
			alertmanagertypes.IncidentAnnotationSopURL:       "https://kb.example/sop/SOP-PAY-001?view=summary",
			alertmanagertypes.IncidentAnnotationSopSource:    "confluence",
			alertmanagertypes.IncidentAnnotationSopTitle:     "Payment API 5xx response",
			alertmanagertypes.IncidentAnnotationSopVersion:   "2026-04-20.3",
			alertmanagertypes.IncidentAnnotationSopBindingID: "payment-api-prod-critical",
		},
	})

	require.Equal(t, SOPPreviewContractVersion, got.ContractVersion)
	require.Equal(t, SOPPreviewStatusBound, got.Status)
	require.Equal(t, SOPSourcePreview{Kind: "configured_source", Name: "confluence"}, got.Source)
	require.Equal(t, SOPBindingPreview{
		SOPID:     "SOP-PAY-001",
		BindingID: "payment-api-prod-critical",
		Version:   "2026-04-20.3",
		Title:     "Payment API 5xx response",
	}, got.Binding)
	require.Equal(t, []string{
		"SOP-PAY-001",
		"payment-api-prod-critical",
		"Payment API 5xx response",
		"2026-04-20.3",
		"confluence",
		"payment-api",
		"prod",
		"critical",
	}, got.Search.Terms)
	require.Equal(t, "SOP-PAY-001 payment-api-prod-critical Payment API 5xx response 2026-04-20.3 confluence payment-api prod critical", got.Search.Query)
	require.Equal(t, SOPDocumentPreview{
		Available:  true,
		Title:      "Payment API 5xx response",
		URL:        "https://kb.example/sop/SOP-PAY-001?view=summary",
		DisplayURL: "kb.example/sop/SOP-PAY-001",
	}, got.Preview)
	require.Equal(t, SOPAccessPreview{
		Mode:                             SOPPreviewAccessModeServerSideConnector,
		RequiresServerSideFetch:          true,
		BrowserCredentialsAllowed:        false,
		RecommendedServiceAccountProfile: SOPPreviewRecommendedServiceAccountProfile,
		CredentialScope:                  SOPPreviewCredentialScopeSourceConnectorSecrets,
		AuditEventRequired:               true,
		Message:                          "Live SOP content must be fetched server-side with source connector credentials; browser credentials are never accepted.",
	}, got.Access)
	require.Empty(t, got.Warnings)
}

func TestPreviewSOPReportsMissingAndInvalidURL(t *testing.T) {
	missing := PreviewSOP(PreviewSOPRequest{})
	require.Equal(t, SOPPreviewStatusMissing, missing.Status)
	require.Equal(t, SOPSourcePreview{Kind: "manual_metadata", Name: "Alert rule metadata"}, missing.Source)
	require.Equal(t, SOPAccessPreview{
		Mode:                      SOPPreviewAccessModeMetadataOnly,
		BrowserCredentialsAllowed: false,
		CredentialScope:           SOPPreviewCredentialScopeNone,
		Message:                   "SOP preview is metadata-only until a source or URL is configured.",
	}, missing.Access)
	require.Equal(t, []string{"Add sop_id or sop_url before production use."}, missing.Warnings)

	invalidURL := PreviewSOP(PreviewSOPRequest{
		Annotations: map[string]string{
			alertmanagertypes.IncidentAnnotationSopURL: "javascript:alert(1)",
		},
	})
	require.Equal(t, SOPPreviewStatusInvalidURL, invalidURL.Status)
	require.False(t, invalidURL.Preview.Available)
	require.Empty(t, invalidURL.Preview.URL)
	require.Equal(t, SOPAccessPreview{
		Mode:                      SOPPreviewAccessModeInvalidURL,
		BrowserCredentialsAllowed: false,
		CredentialScope:           SOPPreviewCredentialScopeNone,
		Message:                   "SOP URL must be HTTP(S) before it can be previewed.",
	}, invalidURL.Access)
	require.Equal(t, []string{sopPreviewURLSchemeWarning}, invalidURL.Warnings)
}

func TestPreviewSOPBlocksCredentialBearingURLs(t *testing.T) {
	withQueryCredential := PreviewSOP(PreviewSOPRequest{
		Annotations: map[string]string{
			alertmanagertypes.IncidentAnnotationSopURL: "https://kb.example/sop/SOP-PAY-001?token=hidden",
		},
	})
	require.Equal(t, SOPPreviewStatusInvalidURL, withQueryCredential.Status)
	require.False(t, withQueryCredential.Preview.Available)
	require.Empty(t, withQueryCredential.Preview.URL)
	require.Equal(t, SOPAccessPreview{
		Mode:                             SOPPreviewAccessModeInvalidURLCredentials,
		RequiresServerSideFetch:          true,
		BrowserCredentialsAllowed:        false,
		RecommendedServiceAccountProfile: SOPPreviewRecommendedServiceAccountProfile,
		CredentialScope:                  SOPPreviewCredentialScopeSourceConnectorSecrets,
		AuditEventRequired:               true,
		Message:                          "Credential-bearing SOP URLs are blocked; configure source credentials server-side.",
	}, withQueryCredential.Access)
	require.Equal(t, []string{sopPreviewURLCredentialWarning}, withQueryCredential.Warnings)

	withUserCredential := PreviewSOP(PreviewSOPRequest{
		Annotations: map[string]string{
			alertmanagertypes.IncidentAnnotationSopURL: "https://user:pass@kb.example/sop/SOP-PAY-001",
		},
	})
	require.Equal(t, SOPPreviewStatusInvalidURL, withUserCredential.Status)
	require.False(t, withUserCredential.Preview.Available)
	require.Empty(t, withUserCredential.Preview.URL)
	require.Equal(t, []string{sopPreviewURLCredentialWarning}, withUserCredential.Warnings)
}
