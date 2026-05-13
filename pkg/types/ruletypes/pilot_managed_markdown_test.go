package ruletypes

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPilotManagedMarkdownCatalogAndHealthValidate(t *testing.T) {
	source := validPilotManagedMarkdownSource()

	catalog, err := NewPilotManagedMarkdownCatalog([]PilotManagedMarkdownSource{source})
	require.NoError(t, err)
	require.Equal(t, PilotSOPSourceCatalogContractVersion, catalog.ContractVersion)
	require.Len(t, catalog.Sources, 1)
	require.Equal(t, PilotSOPSourceKindManagedMarkdown, catalog.Sources[0].Kind)
	require.Equal(t, PilotSOPSourceAuthModeServerSideServiceAccount, catalog.Sources[0].AuthMode)
	require.True(t, *catalog.Sources[0].Capabilities.BodyFetch)
	require.False(t, catalog.Sources[0].SecretRefVisible)

	health, err := NewPilotManagedMarkdownHealth(source, "2026-04-30T00:00:00Z")
	require.NoError(t, err)
	require.Equal(t, PilotSOPSourceHealthContractVersion, health.ContractVersion)
	require.Equal(t, PilotSOPSourceStatusHealthy, health.Status)
	require.Equal(t, PilotCapabilityStatusHealthy, health.CapabilityStatus.BodyFetch)
	require.False(t, health.CredentialDetailsVisible)
}

func TestNewPilotManagedMarkdownHealthReportsEmptySourceAsDisabled(t *testing.T) {
	source := validPilotManagedMarkdownSource()
	source.Documents = nil

	health, err := NewPilotManagedMarkdownHealth(source, "2026-04-30T00:00:00Z")
	require.NoError(t, err)
	require.Equal(t, PilotCapabilityStatusDisabled, health.CapabilityStatus.Search)
	require.Equal(t, PilotCapabilityStatusDisabled, health.CapabilityStatus.BodyFetch)
	require.Contains(t, health.SafeMessage, "no registered documents")
}

func TestFetchPilotManagedMarkdownSOPDeniesUntilAuditAccepted(t *testing.T) {
	resp, err := FetchPilotManagedMarkdownSOP(validPilotManagedMarkdownSource(), validPilotSOPFetchRequest(false))
	require.NoError(t, err)

	require.Equal(t, PilotSOPFetchStatusDenied, resp.Status)
	require.Empty(t, resp.BodyMarkdown)
	require.Equal(t, PilotAuditOutcomeDenied, resp.AuditEvent.Outcome)
	require.Equal(t, "live_fetch_blocked_until_audit_contract_accepted", resp.AuditEvent.Reason)
	require.Equal(t, []string{pilotBodyFetchDisabledUntilAuditContractEnabledWarning}, resp.Warnings)
	require.NoError(t, ValidatePilotSOPFetchResponse(resp))
}

func TestFetchPilotManagedMarkdownSOPReturnsBodyWithAllowedAudit(t *testing.T) {
	resp, err := FetchPilotManagedMarkdownSOP(validPilotManagedMarkdownSource(), validPilotSOPFetchRequest(true))
	require.NoError(t, err)

	require.Equal(t, PilotSOPFetchContractVersion, resp.ContractVersion)
	require.Equal(t, PilotSOPFetchStatusFetched, resp.Status)
	require.Equal(t, "SOP-PAY-001", resp.SOPID)
	require.Equal(t, "2026-04-20.3", resp.Version)
	require.Equal(t, "Payment API 5xx response", resp.Title)
	require.Contains(t, resp.BodyMarkdown, "Restart payment-api only after confirming queue drain.")
	require.Equal(t, PilotAuditOutcomeAllowed, resp.AuditEvent.Outcome)
	require.False(t, resp.SecurityContext.SecretRefVisible)
	require.False(t, resp.SecurityContext.BrowserCredentialsUsed)
	require.NoError(t, ValidatePilotSOPFetchResponse(resp))
}

func TestFetchPilotManagedMarkdownSOPDeniesCrossTenantFetch(t *testing.T) {
	req := validPilotSOPFetchRequest(true)
	req.Tenant.ProjectID = "customer-b"
	req.Tenant.Environment = "stage"

	resp, err := FetchPilotManagedMarkdownSOP(validPilotManagedMarkdownSource(), req)

	require.NoError(t, err)
	require.Equal(t, PilotSOPFetchStatusDenied, resp.Status)
	require.Equal(t, PilotAuditOutcomeDenied, resp.AuditEvent.Outcome)
	require.Equal(t, "tenant_scope_denied", resp.AuditEvent.Reason)
	require.Contains(t, resp.Warnings, SOPTenantPolicyDeniedWarning)
}

func TestFetchPilotManagedMarkdownSOPRejectsSecretLikeDocumentBody(t *testing.T) {
	source := validPilotManagedMarkdownSource()
	source.Documents[0].BodyMarkdown = "Run curl with access_token=hidden"

	resp, err := FetchPilotManagedMarkdownSOP(source, validPilotSOPFetchRequest(true))
	require.NoError(t, err)

	require.Equal(t, PilotSOPFetchStatusRedacted, resp.Status)
	require.Empty(t, resp.BodyMarkdown)
	require.Equal(t, PilotAuditOutcomeRedacted, resp.AuditEvent.Outcome)
	require.Equal(t, []string{"sop_document_contains_secret_like_value"}, resp.Warnings)
	require.NoError(t, ValidatePilotSOPFetchResponse(resp))
}

func TestFetchPilotManagedMarkdownSOPRejectsCredentialBearingDisplayURL(t *testing.T) {
	source := validPilotManagedMarkdownSource()
	source.Documents[0].DisplayURL = "https://kb.example/sop/SOP-PAY-001?token=hidden"

	resp, err := FetchPilotManagedMarkdownSOP(source, validPilotSOPFetchRequest(true))
	require.NoError(t, err)

	require.Equal(t, PilotSOPFetchStatusRedacted, resp.Status)
	require.Empty(t, resp.BodyMarkdown)
	require.Equal(t, []string{"sop_document_contains_secret_like_value"}, resp.Warnings)
	require.NoError(t, ValidatePilotSOPFetchResponse(resp))
}

func TestFetchPilotManagedMarkdownSOPReportsNotFoundAndUnavailableSources(t *testing.T) {
	notFoundReq := validPilotSOPFetchRequest(true)
	notFoundReq.SOPID = "SOP-UNKNOWN"
	notFoundResp, err := FetchPilotManagedMarkdownSOP(validPilotManagedMarkdownSource(), notFoundReq)
	require.NoError(t, err)
	require.Equal(t, PilotSOPFetchStatusNotFound, notFoundResp.Status)
	require.Equal(t, "sop_document_not_found", notFoundResp.AuditEvent.Reason)

	disabledSource := validPilotManagedMarkdownSource()
	disabledSource.Status = PilotSOPSourceStatusDisabled
	unavailableResp, err := FetchPilotManagedMarkdownSOP(disabledSource, validPilotSOPFetchRequest(true))
	require.NoError(t, err)
	require.Equal(t, PilotSOPFetchStatusSourceUnavailable, unavailableResp.Status)
	require.Equal(t, PilotAuditOutcomeFailed, unavailableResp.AuditEvent.Outcome)
	require.Equal(t, "source_not_available", unavailableResp.AuditEvent.Reason)
}

func TestValidatePilotSOPFetchResponseBlocksBodyOnDeniedResponses(t *testing.T) {
	resp, err := FetchPilotManagedMarkdownSOP(validPilotManagedMarkdownSource(), validPilotSOPFetchRequest(false))
	require.NoError(t, err)
	resp.BodyMarkdown = "should not be present"

	require.ErrorContains(t, ValidatePilotSOPFetchResponse(resp), "bodyMarkdown: must be empty unless status is fetched")
}

func TestFetchPilotManagedMarkdownSOPRedactsOversizedBody(t *testing.T) {
	source := validPilotManagedMarkdownSource()
	source.Documents[0].BodyMarkdown = strings.Repeat("a", PilotSOPFetchBodyMarkdownMaxBytes+1)

	resp, err := FetchPilotManagedMarkdownSOP(source, validPilotSOPFetchRequest(true))
	require.NoError(t, err)

	require.Equal(t, PilotSOPFetchStatusRedacted, resp.Status)
	require.Empty(t, resp.BodyMarkdown)
	require.Equal(t, []string{pilotSOPDocumentBodyExceedsMaxSizeWarning}, resp.Warnings)
	require.Equal(t, PilotAuditOutcomeRedacted, resp.AuditEvent.Outcome)
	require.Equal(t, pilotSOPDocumentBodyExceedsMaxSizeWarning, resp.AuditEvent.Reason)
	require.NoError(t, ValidatePilotSOPFetchResponse(resp))
}

func TestFetchPilotManagedMarkdownSOPRedactsHTMLPayload(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "doctype", body: "<!DOCTYPE html><html><body>SOP page</body></html>"},
		{name: "html-tag", body: "<html><body>SOP page</body></html>"},
		{name: "html-uppercase", body: "<HTML><body>ops doc</body></HTML>"},
		{name: "leading-whitespace", body: "  \n\t<!doctype html><html></html>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := validPilotManagedMarkdownSource()
			source.Documents[0].BodyMarkdown = tc.body

			resp, err := FetchPilotManagedMarkdownSOP(source, validPilotSOPFetchRequest(true))
			require.NoError(t, err)

			require.Equal(t, PilotSOPFetchStatusRedacted, resp.Status)
			require.Empty(t, resp.BodyMarkdown)
			require.Equal(t, []string{pilotSOPDocumentBodyNotMarkdownWarning}, resp.Warnings)
			require.Equal(t, pilotSOPDocumentBodyNotMarkdownWarning, resp.AuditEvent.Reason)
		})
	}
}

func TestFetchPilotManagedMarkdownSOPRedactsBinaryPayload(t *testing.T) {
	source := validPilotManagedMarkdownSource()
	source.Documents[0].BodyMarkdown = "Run book\x00 with embedded NUL"

	resp, err := FetchPilotManagedMarkdownSOP(source, validPilotSOPFetchRequest(true))
	require.NoError(t, err)

	require.Equal(t, PilotSOPFetchStatusRedacted, resp.Status)
	require.Equal(t, []string{pilotSOPDocumentBodyNotMarkdownWarning}, resp.Warnings)
}

func TestValidatePilotSOPFetchResponseRejectsCredentialFlags(t *testing.T) {
	t.Run("secretRefVisible", func(t *testing.T) {
		resp, err := FetchPilotManagedMarkdownSOP(validPilotManagedMarkdownSource(), validPilotSOPFetchRequest(true))
		require.NoError(t, err)
		resp.SecurityContext.SecretRefVisible = true

		require.ErrorContains(t, ValidatePilotSOPFetchResponse(resp), "securityContext.secretRefVisible")
	})

	t.Run("browserCredentialsUsed", func(t *testing.T) {
		resp, err := FetchPilotManagedMarkdownSOP(validPilotManagedMarkdownSource(), validPilotSOPFetchRequest(true))
		require.NoError(t, err)
		resp.SecurityContext.BrowserCredentialsUsed = true

		require.ErrorContains(t, ValidatePilotSOPFetchResponse(resp), "securityContext.browserCredentialsUsed")
	})
}

func TestValidatePilotSOPFetchResponseRejectsOversizedBody(t *testing.T) {
	resp, err := FetchPilotManagedMarkdownSOP(validPilotManagedMarkdownSource(), validPilotSOPFetchRequest(true))
	require.NoError(t, err)
	resp.BodyMarkdown = strings.Repeat("a", PilotSOPFetchBodyMarkdownMaxBytes+1)

	require.ErrorContains(t, ValidatePilotSOPFetchResponse(resp), "bodyMarkdown: exceeds max size")
}

func TestValidatePilotSOPFetchResponseRejectsNonMarkdownBody(t *testing.T) {
	resp, err := FetchPilotManagedMarkdownSOP(validPilotManagedMarkdownSource(), validPilotSOPFetchRequest(true))
	require.NoError(t, err)
	resp.BodyMarkdown = "<!DOCTYPE html><html></html>"

	require.ErrorContains(t, ValidatePilotSOPFetchResponse(resp), "bodyMarkdown: payload does not look like markdown")
}

func TestFetchPilotManagedMarkdownSOPAcceptsBodyAtExactMaxSize(t *testing.T) {
	source := validPilotManagedMarkdownSource()
	source.Documents[0].BodyMarkdown = strings.Repeat("a", PilotSOPFetchBodyMarkdownMaxBytes)

	resp, err := FetchPilotManagedMarkdownSOP(source, validPilotSOPFetchRequest(true))
	require.NoError(t, err)

	require.Equal(t, PilotSOPFetchStatusFetched, resp.Status)
	require.Len(t, resp.BodyMarkdown, PilotSOPFetchBodyMarkdownMaxBytes)
	require.Empty(t, resp.Warnings)
	require.NoError(t, ValidatePilotSOPFetchResponse(resp))
}

func TestFetchPilotManagedMarkdownSOPRedactsBinaryMagicPrefixes(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "pdf-magic", body: "%PDF-1.7\nrunbook content"},
		{name: "zip-magic", body: "PK\x03\x04 archived runbook"},
		{name: "pdf-with-leading-bom", body: "\ufeff%PDF-1.4 ..."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := validPilotManagedMarkdownSource()
			source.Documents[0].BodyMarkdown = tc.body

			resp, err := FetchPilotManagedMarkdownSOP(source, validPilotSOPFetchRequest(true))
			require.NoError(t, err)

			require.Equal(t, PilotSOPFetchStatusRedacted, resp.Status)
			require.Equal(t, []string{pilotSOPDocumentBodyNotMarkdownWarning}, resp.Warnings)
			require.Equal(t, pilotSOPDocumentBodyNotMarkdownWarning, resp.AuditEvent.Reason)
		})
	}
}

func TestFetchPilotManagedMarkdownSOPFailsValidationOnEmptyDocumentBody(t *testing.T) {
	source := validPilotManagedMarkdownSource()
	source.Documents[0].BodyMarkdown = ""

	resp, err := FetchPilotManagedMarkdownSOP(source, validPilotSOPFetchRequest(true))
	require.Error(t, err, "fetched response with empty bodyMarkdown must fail required-field validation")
	require.Contains(t, err.Error(), "bodyMarkdown: field is required")
	require.Equal(t, PilotSOPFetchStatusFetched, resp.Status)
}

func TestPilotBodyMarkdownLooksNonMarkdownAcceptsValidMarkdown(t *testing.T) {
	cases := []string{
		"",
		"# Heading\n\n- step 1\n- step 2",
		"Restart payment-api only after confirming queue drain.",
		"\ufeff# Heading after BOM",
		"```bash\nkubectl rollout restart deploy/payment-api\n```",
	}
	for _, body := range cases {
		require.False(t, pilotBodyMarkdownLooksNonMarkdown(body), "expected markdown-like for %q", body)
	}
}

func validPilotManagedMarkdownSource() PilotManagedMarkdownSource {
	return PilotManagedMarkdownSource{
		SourceID:              "src-managed-markdown-default",
		DisplayName:           "Managed Markdown SOP Registry",
		Status:                PilotSOPSourceStatusHealthy,
		LastHealthCheckAt:     "2026-04-30T00:00:00Z",
		LastSyncAt:            "2026-04-30T00:00:00Z",
		ServiceAccountProfile: "ds-sop-reader",
		TenantScope: PilotTenantScope{
			ProjectIDs:   []string{"customer-a"},
			Environments: []string{"prod"},
		},
		ConfiguredBy: "ds-admin",
		Documents: []PilotManagedMarkdownDocument{
			{
				SOPID:        "SOP-PAY-001",
				Version:      "2026-04-20.3",
				Title:        "Payment API 5xx response",
				BodyMarkdown: "Restart payment-api only after confirming queue drain.",
				DisplayURL:   "https://kb.example/sop/SOP-PAY-001",
				UpdatedAt:    "2026-04-20T00:00:00Z",
				Tags:         []string{"payment-api", "prod", "critical"},
			},
		},
	}
}

func validPilotSOPFetchRequest(auditAccepted bool) PilotSOPFetchRequest {
	return PilotSOPFetchRequest{
		SourceID:              "src-managed-markdown-default",
		SOPID:                 "SOP-PAY-001",
		Version:               "2026-04-20.3",
		OccurredAt:            "2026-04-30T00:00:00Z",
		AuditEventID:          "audit-20260430-000001",
		AuditMode:             PilotAuditModeRequired,
		AuditAccepted:         auditAccepted,
		ServiceAccountProfile: "ds-sop-reader",
		Actor: PilotAuditActor{
			Kind:        PilotAuditActorKindUser,
			ID:          "user-123",
			DisplayName: "PM Reviewer",
		},
		Tenant: PilotAuditTenant{
			ProjectID:   "customer-a",
			Environment: "prod",
		},
		RequestContext: PilotAuditRequestContext{
			AlertRuleID: "rule-payment-api-5xx",
			IncidentID:  "INC-20260430-001",
			ServiceName: "payment-api",
			Severity:    "critical",
		},
	}
}
