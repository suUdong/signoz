package ruletypes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidatePilotSOPSourceCatalogAcceptsValidResponse(t *testing.T) {
	require.NoError(t, ValidatePilotSOPSourceCatalog(validPilotSOPSourceCatalog()))
}

func TestValidatePilotSOPSourceCatalogRejectsUnsupportedKindAndAuthMode(t *testing.T) {
	unsupportedKind := validPilotSOPSourceCatalog()
	unsupportedKind.Sources[0].Kind = "unknown_vendor"
	require.ErrorContains(t, ValidatePilotSOPSourceCatalog(unsupportedKind), `sources[0].kind: unsupported value "unknown_vendor"`)

	unsupportedAuth := validPilotSOPSourceCatalog()
	unsupportedAuth.Sources[0].AuthMode = "browser_token"
	require.ErrorContains(t, ValidatePilotSOPSourceCatalog(unsupportedAuth), `sources[0].authMode: unsupported value "browser_token"`)
}

func TestValidatePilotSOPSourceCatalogBlocksSecretsAndIncompleteCapabilities(t *testing.T) {
	secretLeak := validPilotSOPSourceCatalog()
	secretLeak.Sources[0].DisplayName = "Managed Markdown token=hidden"
	require.ErrorContains(t, ValidatePilotSOPSourceCatalog(secretLeak), "sources[0].displayName: contains secret-like value")

	missingCapability := validPilotSOPSourceCatalog()
	missingCapability.Sources[0].Capabilities.BodyFetch = nil
	require.ErrorContains(t, ValidatePilotSOPSourceCatalog(missingCapability), "sources[0].capabilities.bodyFetch: field is required")

	visibleSecretRef := validPilotSOPSourceCatalog()
	visibleSecretRef.Sources[0].SecretRefVisible = true
	require.ErrorContains(t, ValidatePilotSOPSourceCatalog(visibleSecretRef), "sources[0].secretRefVisible: must be false")
}

func TestValidatePilotSOPSourceHealthAcceptsDegradedBoundaryResponse(t *testing.T) {
	require.NoError(t, ValidatePilotSOPSourceHealth(validPilotSOPSourceHealth()))
}

func TestValidatePilotSOPSourceHealthRejectsCredentialExposureAndUnknownStatus(t *testing.T) {
	visibleCredentials := validPilotSOPSourceHealth()
	visibleCredentials.CredentialDetailsVisible = true
	require.ErrorContains(t, ValidatePilotSOPSourceHealth(visibleCredentials), "credentialDetailsVisible: must be false")

	secretMessage := validPilotSOPSourceHealth()
	secretMessage.SafeMessage = "source returned password reset challenge"
	require.ErrorContains(t, ValidatePilotSOPSourceHealth(secretMessage), "safeMessage: contains secret-like value")

	unknownCapability := validPilotSOPSourceHealth()
	unknownCapability.CapabilityStatus.Preview = "partially_ok"
	require.ErrorContains(t, ValidatePilotSOPSourceHealth(unknownCapability), `capabilityStatus.preview: unsupported value "partially_ok"`)
}

func TestValidatePilotAuditEventAcceptsPreviewDeniedFetchAndDeferredEvidence(t *testing.T) {
	preview := validPilotAuditEvent()
	require.NoError(t, ValidatePilotAuditEvent(preview))

	deniedFetch := validPilotAuditEvent()
	deniedFetch.EventID = "audit-20260429-000002"
	deniedFetch.EventType = PilotAuditEventTypeSOPFetch
	deniedFetch.Action = "fetch"
	deniedFetch.Outcome = PilotAuditOutcomeDenied
	deniedFetch.Reason = "live_fetch_blocked_until_audit_contract_accepted"
	require.NoError(t, ValidatePilotAuditEvent(deniedFetch))

	deferredEvidence := validPilotAuditEvent()
	deferredEvidence.EventID = "audit-20260429-000003"
	deferredEvidence.EventType = PilotAuditEventTypeEvidenceCollectRequest
	deferredEvidence.Resource.Kind = "evidence_request"
	deferredEvidence.Resource.SourceID = ""
	deferredEvidence.Resource.SOPID = ""
	deferredEvidence.Action = "collect_request"
	deferredEvidence.Outcome = PilotAuditOutcomeDeferred
	deferredEvidence.Reason = "evidence_contract_lane_not_accepted"
	require.NoError(t, ValidatePilotAuditEvent(deferredEvidence))
}

func TestValidatePilotAuditEventRejectsMissingTenantBrowserCredentialsAndSecrets(t *testing.T) {
	missingTenant := validPilotAuditEvent()
	missingTenant.Tenant.ProjectID = ""
	require.ErrorContains(t, ValidatePilotAuditEvent(missingTenant), "tenant.projectId: field is required")

	browserCredentials := validPilotAuditEvent()
	browserCredentials.SecurityContext.BrowserCredentialsUsed = true
	require.ErrorContains(t, ValidatePilotAuditEvent(browserCredentials), "securityContext.browserCredentialsUsed: must be false")

	visibleSecretRef := validPilotAuditEvent()
	visibleSecretRef.SecurityContext.SecretRefVisible = true
	require.ErrorContains(t, ValidatePilotAuditEvent(visibleSecretRef), "securityContext.secretRefVisible: must be false")

	secretReason := validPilotAuditEvent()
	secretReason.Reason = "bearer abcdef"
	require.ErrorContains(t, ValidatePilotAuditEvent(secretReason), "reason: contains secret-like value")
}

func TestValidatePilotServiceAccountProfileAcceptsValidProfile(t *testing.T) {
	require.NoError(t, ValidatePilotServiceAccountProfile(validPilotServiceAccountProfile()))
}

func TestValidatePilotServiceAccountProfileRejectsBrowserUseUnscopedAndMissingRotation(t *testing.T) {
	browserUsable := validPilotServiceAccountProfile()
	browserUsable.BrowserUsable = true
	require.ErrorContains(t, ValidatePilotServiceAccountProfile(browserUsable), "browserUsable: must be false")

	visibleSecretRef := validPilotServiceAccountProfile()
	visibleSecretRef.SecretRefVisible = true
	require.ErrorContains(t, ValidatePilotServiceAccountProfile(visibleSecretRef), "secretRefVisible: must be false")

	missingRotation := validPilotServiceAccountProfile()
	missingRotation.RotationPolicy = ""
	require.ErrorContains(t, ValidatePilotServiceAccountProfile(missingRotation), "rotationPolicy: field is required")

	unscoped := validPilotServiceAccountProfile()
	unscoped.TenantScope.ProjectIDs = nil
	require.ErrorContains(t, ValidatePilotServiceAccountProfile(unscoped), "tenantScope.projectIds: must include at least one project")
}

func TestValidatePilotConfigurationAcceptsSingleSourceAndDisabledFlag(t *testing.T) {
	config := validPilotConfiguration()
	require.NoError(t, ValidatePilotConfiguration(config))

	config.Enabled = false
	require.NoError(t, ValidatePilotConfiguration(config))
}

func TestValidatePilotConfigurationRequiresPriorityForMultipleSources(t *testing.T) {
	config := validPilotConfiguration()
	config.SelectedSources = append(config.SelectedSources, PilotSelectedSource{
		SourceID:              "src-confluence-secondary",
		ServiceAccountProfile: "ds-sop-reader",
	})

	require.ErrorContains(t, ValidatePilotConfiguration(config), "selectedSources[0].priority: field is required")
	require.ErrorContains(t, ValidatePilotConfiguration(config), "selectedSources[1].priority: field is required")

	firstPriority := 10
	secondPriority := 20
	config.SelectedSources[0].Priority = &firstPriority
	config.SelectedSources[1].Priority = &secondPriority
	require.NoError(t, ValidatePilotConfiguration(config))
}

func TestPilotSecretLikeValueDetectionKeepsSymbolicNamesButBlocksCredentialShapes(t *testing.T) {
	require.False(t, hasPilotSecretLikeValue("source_connector_secret"))
	require.False(t, hasPilotSecretLikeValue("custom_secret_ref"))
	require.False(t, hasPilotSecretLikeValue("ds-sop-reader"))

	require.True(t, hasPilotSecretLikeValue("https://kb.example/sop?token=hidden"))
	require.True(t, hasPilotSecretLikeValue("client_secret=s3cr3t"))
	require.True(t, hasPilotSecretLikeValue("Authorization: Bearer abcdef"))
	require.True(t, hasPilotSecretLikeValue("-----BEGIN PRIVATE KEY-----"))
	require.True(t, hasPilotSecretLikeValue("eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJwaWxvdCJ9.dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"))
}

func validPilotSOPSourceCatalog() PilotSOPSourceCatalogResponse {
	return PilotSOPSourceCatalogResponse{
		ContractVersion: PilotSOPSourceCatalogContractVersion,
		Sources: []PilotSOPSource{
			{
				SourceID:          "src-managed-markdown-default",
				DisplayName:       "Managed Markdown SOP Registry",
				Kind:              PilotSOPSourceKindManagedMarkdown,
				AuthMode:          PilotSOPSourceAuthModeServerSideServiceAccount,
				Status:            PilotSOPSourceStatusHealthy,
				LastHealthCheckAt: "2026-04-29T00:00:00Z",
				LastSyncAt:        "2026-04-29T00:00:00Z",
				Capabilities: PilotSOPSourceCapabilities{
					Search:           pilotBool(true),
					Preview:          pilotBool(true),
					BodyFetch:        pilotBool(false),
					VersionSnapshots: pilotBool(true),
					WebhookSync:      pilotBool(false),
				},
				ServiceAccountProfile: "ds-sop-reader",
				SecretRefVisible:      false,
				ConfiguredBy:          "ds-admin",
				Warnings:              []string{},
			},
		},
	}
}

func validPilotSOPSourceHealth() PilotSOPSourceHealthResponse {
	return PilotSOPSourceHealthResponse{
		ContractVersion: PilotSOPSourceHealthContractVersion,
		SourceID:        "src-managed-markdown-default",
		Status:          PilotSOPSourceStatusDegraded,
		CheckedAt:       "2026-04-29T00:00:00Z",
		CapabilityStatus: PilotSOPSourceCapabilityStatus{
			Search:           PilotCapabilityStatusHealthy,
			Preview:          PilotCapabilityStatusHealthy,
			BodyFetch:        PilotCapabilityStatusDisabled,
			VersionSnapshots: PilotCapabilityStatusDegraded,
		},
		LastSuccessfulSyncAt:     "2026-04-28T23:30:00Z",
		SafeMessage:              "Source is reachable, but version snapshot sync is delayed.",
		RecommendedAction:        "Check server-side source sync worker and audit queue.",
		CredentialDetailsVisible: false,
		Warnings: []string{
			pilotBodyFetchDisabledUntilAuditContractEnabledWarning,
		},
	}
}

func validPilotAuditEvent() PilotAuditEvent {
	return PilotAuditEvent{
		ContractVersion: PilotAuditEventContractVersion,
		EventID:         "audit-20260429-000001",
		EventType:       PilotAuditEventTypeSOPPreview,
		OccurredAt:      "2026-04-29T00:00:00Z",
		Actor: PilotAuditActor{
			Kind:        PilotAuditActorKindUser,
			ID:          "user-123",
			DisplayName: "PM Reviewer",
		},
		Tenant: PilotAuditTenant{
			ProjectID:   "customer-a",
			Environment: "prod",
		},
		Resource: PilotAuditResource{
			Kind:     "sop_source",
			SourceID: "src-managed-markdown-default",
			SOPID:    "SOP-PAY-001",
			Version:  "2026-04-20.3",
		},
		Action:  "preview",
		Outcome: PilotAuditOutcomeAllowed,
		Reason:  "metadata_preview_only",
		RequestContext: PilotAuditRequestContext{
			AlertRuleID: "rule-payment-api-5xx",
			IncidentID:  "INC-20260429-001",
			ServiceName: "payment-api",
			Severity:    "critical",
		},
		SecurityContext: PilotAuditSecurityContext{
			ServiceAccountProfile:  "ds-sop-reader",
			SecretRefVisible:       false,
			BrowserCredentialsUsed: false,
			RedactionApplied:       true,
		},
	}
}

func validPilotServiceAccountProfile() PilotServiceAccountProfile {
	return PilotServiceAccountProfile{
		ContractVersion: PilotServiceAccountProfileContractVersion,
		ProfileID:       "ds-sop-reader",
		DisplayName:     "DS SOP Reader",
		Purpose:         "Read-only server-side SOP source access for pilot previews and fetches.",
		AllowedActions: []string{
			PilotAuditEventTypeSOPSearch,
			PilotAuditEventTypeSOPPreview,
			PilotAuditEventTypeSOPFetch,
		},
		TenantScope: PilotTenantScope{
			ProjectIDs: []string{
				"customer-a",
			},
			Environments: []string{
				"prod",
				"staging",
			},
		},
		SecretRefVisible: false,
		BrowserUsable:    false,
		RotationPolicy:   "customer_policy_or_90d_default",
	}
}

func validPilotConfiguration() PilotConfiguration {
	return PilotConfiguration{
		ContractVersion: PilotConfigurationContractVersion,
		ProjectID:       "customer-a",
		Environment:     "prod",
		ServiceName:     "payment-api",
		SelectedSources: []PilotSelectedSource{
			{
				SourceID:              "src-managed-markdown-default",
				ServiceAccountProfile: "ds-sop-reader",
			},
		},
		AllowedCapabilities: PilotAllowedCapabilities{
			Search:           pilotBool(true),
			Preview:          pilotBool(true),
			BodyFetch:        pilotBool(false),
			VersionSnapshots: pilotBool(true),
		},
		AuditMode: PilotAuditModeRequired,
		Enabled:   true,
		RolloutID: "pilot-customer-a-payment-api",
	}
}

func pilotBool(value bool) *bool {
	return &value
}
