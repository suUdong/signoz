package ruletypes

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	PilotSOPSourceCatalogContractVersion      = "ds-apm.sop-source-catalog.v1"
	PilotSOPSourceHealthContractVersion       = "ds-apm.sop-source-health.v1"
	PilotAuditEventContractVersion            = "ds-apm.audit-event.v1"
	PilotServiceAccountProfileContractVersion = "ds-apm.service-account-profile.v1"
	PilotConfigurationContractVersion         = "ds-apm.pilot-configuration.v1"

	PilotSOPSourceKindURLRegistry     = "url_registry"
	PilotSOPSourceKindManagedMarkdown = "managed_markdown"
	PilotSOPSourceKindGitMarkdown     = "git_markdown"
	PilotSOPSourceKindConfluence      = "confluence"
	PilotSOPSourceKindNotion          = "notion"
	PilotSOPSourceKindSharePoint      = "sharepoint"
	PilotSOPSourceKindCustomConnector = "custom_connector"

	PilotSOPSourceAuthModeNone                     = "none"
	PilotSOPSourceAuthModePublicURL                = "public_url"
	PilotSOPSourceAuthModeServerSideServiceAccount = "server_side_service_account"
	PilotSOPSourceAuthModeServerSideOAuth          = "server_side_oauth"
	PilotSOPSourceAuthModeServerSideAPIKey         = "server_side_api_key"
	PilotSOPSourceAuthModeCustomSecretRef          = "custom_secret_ref"

	PilotSOPSourceStatusHealthy      = "healthy"
	PilotSOPSourceStatusDegraded     = "degraded"
	PilotSOPSourceStatusUnconfigured = "unconfigured"
	PilotSOPSourceStatusUnauthorized = "unauthorized"
	PilotSOPSourceStatusUnreachable  = "unreachable"
	PilotSOPSourceStatusStale        = "stale"
	PilotSOPSourceStatusDisabled     = "disabled"

	PilotCapabilityStatusHealthy      = "healthy"
	PilotCapabilityStatusDegraded     = "degraded"
	PilotCapabilityStatusDisabled     = "disabled"
	PilotCapabilityStatusUnauthorized = "unauthorized"
	PilotCapabilityStatusUnreachable  = "unreachable"
	PilotCapabilityStatusUnknown      = "unknown"

	PilotAuditEventTypeSOPSearch              = "sop.search"
	PilotAuditEventTypeSOPPreview             = "sop.preview"
	PilotAuditEventTypeSOPFetch               = "sop.fetch"
	PilotAuditEventTypeSOPHealthCheck         = "sop.health_check"
	PilotAuditEventTypeEvidenceCollectRequest = "evidence.collect_request"
	PilotAuditEventTypeEvidenceCollectResult  = "evidence.collect_result"
	PilotAuditEventTypeAISummaryRequest       = "ai.summary_request"
	PilotAuditEventTypeAISummaryResult        = "ai.summary_result"

	PilotAuditOutcomeAllowed  = "allowed"
	PilotAuditOutcomeDenied   = "denied"
	PilotAuditOutcomeRedacted = "redacted"
	PilotAuditOutcomeFailed   = "failed"
	PilotAuditOutcomeDeferred = "deferred"

	PilotAuditActorKindUser           = "user"
	PilotAuditActorKindSystem         = "system"
	PilotAuditActorKindServiceAccount = "service_account"

	PilotAuditModeRequired = "required"
	PilotAuditModeDeferred = "deferred"
	PilotAuditModeDisabled = "disabled"
)

const pilotBodyFetchDisabledUntilAuditContractEnabledWarning = "body_fetch_disabled_until_audit_contract_enabled"

type PilotSOPSourceCatalogResponse struct {
	ContractVersion string           `json:"contractVersion"`
	Sources         []PilotSOPSource `json:"sources"`
}

type PilotSOPSource struct {
	SourceID              string                     `json:"sourceId"`
	DisplayName           string                     `json:"displayName"`
	Kind                  string                     `json:"kind"`
	AuthMode              string                     `json:"authMode"`
	Status                string                     `json:"status"`
	LastHealthCheckAt     string                     `json:"lastHealthCheckAt,omitempty"`
	LastSyncAt            string                     `json:"lastSyncAt,omitempty"`
	Capabilities          PilotSOPSourceCapabilities `json:"capabilities"`
	ServiceAccountProfile string                     `json:"serviceAccountProfile,omitempty"`
	SecretRefVisible      bool                       `json:"secretRefVisible"`
	ConfiguredBy          string                     `json:"configuredBy,omitempty"`
	Warnings              []string                   `json:"warnings,omitempty"`
}

type PilotSOPSourceCapabilities struct {
	Search           *bool `json:"search"`
	Preview          *bool `json:"preview"`
	BodyFetch        *bool `json:"bodyFetch"`
	VersionSnapshots *bool `json:"versionSnapshots"`
	WebhookSync      *bool `json:"webhookSync,omitempty"`
}

type PilotSOPSourceHealthResponse struct {
	ContractVersion          string                         `json:"contractVersion"`
	SourceID                 string                         `json:"sourceId"`
	Status                   string                         `json:"status"`
	CheckedAt                string                         `json:"checkedAt"`
	CapabilityStatus         PilotSOPSourceCapabilityStatus `json:"capabilityStatus"`
	LastSuccessfulSyncAt     string                         `json:"lastSuccessfulSyncAt,omitempty"`
	SafeMessage              string                         `json:"safeMessage,omitempty"`
	RecommendedAction        string                         `json:"recommendedAction,omitempty"`
	CredentialDetailsVisible bool                           `json:"credentialDetailsVisible"`
	Warnings                 []string                       `json:"warnings,omitempty"`
}

type PilotSOPSourceCapabilityStatus struct {
	Search           string `json:"search"`
	Preview          string `json:"preview"`
	BodyFetch        string `json:"bodyFetch"`
	VersionSnapshots string `json:"versionSnapshots"`
	WebhookSync      string `json:"webhookSync,omitempty"`
}

type PilotAuditEvent struct {
	ContractVersion string                    `json:"contractVersion"`
	EventID         string                    `json:"eventId"`
	EventType       string                    `json:"eventType"`
	OccurredAt      string                    `json:"occurredAt"`
	Actor           PilotAuditActor           `json:"actor"`
	Tenant          PilotAuditTenant          `json:"tenant"`
	Resource        PilotAuditResource        `json:"resource"`
	Action          string                    `json:"action"`
	Outcome         string                    `json:"outcome"`
	Reason          string                    `json:"reason,omitempty"`
	RequestContext  PilotAuditRequestContext  `json:"requestContext"`
	SecurityContext PilotAuditSecurityContext `json:"securityContext"`
}

type PilotAuditActor struct {
	Kind        string `json:"kind"`
	ID          string `json:"id"`
	DisplayName string `json:"displayName,omitempty"`
}

type PilotAuditTenant struct {
	ProjectID   string `json:"projectId"`
	Environment string `json:"environment"`
}

type PilotAuditResource struct {
	Kind     string `json:"kind"`
	SourceID string `json:"sourceId,omitempty"`
	SOPID    string `json:"sopId,omitempty"`
	Version  string `json:"version,omitempty"`
}

type PilotAuditRequestContext struct {
	AlertRuleID string `json:"alertRuleId,omitempty"`
	IncidentID  string `json:"incidentId,omitempty"`
	ServiceName string `json:"serviceName,omitempty"`
	Severity    string `json:"severity,omitempty"`
}

type PilotAuditSecurityContext struct {
	ServiceAccountProfile  string `json:"serviceAccountProfile"`
	SecretRefVisible       bool   `json:"secretRefVisible"`
	BrowserCredentialsUsed bool   `json:"browserCredentialsUsed"`
	RedactionApplied       bool   `json:"redactionApplied"`
}

type PilotServiceAccountProfile struct {
	ContractVersion  string           `json:"contractVersion"`
	ProfileID        string           `json:"profileId"`
	DisplayName      string           `json:"displayName"`
	Purpose          string           `json:"purpose"`
	AllowedActions   []string         `json:"allowedActions"`
	TenantScope      PilotTenantScope `json:"tenantScope"`
	SecretRefVisible bool             `json:"secretRefVisible"`
	BrowserUsable    bool             `json:"browserUsable"`
	RotationPolicy   string           `json:"rotationPolicy"`
}

type PilotTenantScope struct {
	ProjectIDs   []string `json:"projectIds"`
	Environments []string `json:"environments"`
}

type PilotConfiguration struct {
	ContractVersion     string                   `json:"contractVersion"`
	ProjectID           string                   `json:"projectId"`
	Environment         string                   `json:"environment"`
	ServiceName         string                   `json:"serviceName"`
	SelectedSources     []PilotSelectedSource    `json:"selectedSources"`
	AllowedCapabilities PilotAllowedCapabilities `json:"allowedCapabilities"`
	AuditMode           string                   `json:"auditMode"`
	Enabled             bool                     `json:"enabled"`
	RolloutID           string                   `json:"rolloutId,omitempty"`
}

type PilotSelectedSource struct {
	SourceID              string `json:"sourceId"`
	ServiceAccountProfile string `json:"serviceAccountProfile"`
	Priority              *int   `json:"priority,omitempty"`
}

type PilotAllowedCapabilities struct {
	Search           *bool `json:"search"`
	Preview          *bool `json:"preview"`
	BodyFetch        *bool `json:"bodyFetch"`
	VersionSnapshots *bool `json:"versionSnapshots"`
}

func ValidatePilotSOPSourceCatalog(resp PilotSOPSourceCatalogResponse) error {
	var errs []error

	pilotRequireContractVersion(&errs, "contractVersion", resp.ContractVersion, PilotSOPSourceCatalogContractVersion)
	if len(resp.Sources) == 0 {
		errs = append(errs, fmt.Errorf("sources: must include at least one source"))
	}

	for i, source := range resp.Sources {
		path := fmt.Sprintf("sources[%d]", i)
		pilotRequireNonEmpty(&errs, path+".sourceId", source.SourceID)
		pilotRequireNonEmpty(&errs, path+".displayName", source.DisplayName)
		pilotRequireAllowed(&errs, path+".kind", source.Kind, allowedPilotSOPSourceKinds)
		pilotRequireAllowed(&errs, path+".authMode", source.AuthMode, allowedPilotSOPSourceAuthModes)
		pilotRequireAllowed(&errs, path+".status", source.Status, allowedPilotSOPSourceStatuses)
		pilotRequireCapabilityFlag(&errs, path+".capabilities.search", source.Capabilities.Search)
		pilotRequireCapabilityFlag(&errs, path+".capabilities.preview", source.Capabilities.Preview)
		pilotRequireCapabilityFlag(&errs, path+".capabilities.bodyFetch", source.Capabilities.BodyFetch)
		pilotRequireCapabilityFlag(&errs, path+".capabilities.versionSnapshots", source.Capabilities.VersionSnapshots)

		if source.SecretRefVisible {
			errs = append(errs, fmt.Errorf("%s.secretRefVisible: must be false for browser-visible catalog responses", path))
		}
		if pilotAuthModeRequiresServiceAccount(source.AuthMode) {
			pilotRequireNonEmpty(&errs, path+".serviceAccountProfile", source.ServiceAccountProfile)
		}

		pilotAppendSecretLikeStringErrors(&errs, path+".sourceId", source.SourceID)
		pilotAppendSecretLikeStringErrors(&errs, path+".displayName", source.DisplayName)
		pilotAppendSecretLikeStringErrors(&errs, path+".lastHealthCheckAt", source.LastHealthCheckAt)
		pilotAppendSecretLikeStringErrors(&errs, path+".lastSyncAt", source.LastSyncAt)
		pilotAppendSecretLikeStringErrors(&errs, path+".serviceAccountProfile", source.ServiceAccountProfile)
		pilotAppendSecretLikeStringErrors(&errs, path+".configuredBy", source.ConfiguredBy)
		for j, warning := range source.Warnings {
			pilotAppendSecretLikeStringErrors(&errs, fmt.Sprintf("%s.warnings[%d]", path, j), warning)
		}
	}

	return errors.Join(errs...)
}

func ValidatePilotSOPSourceHealth(resp PilotSOPSourceHealthResponse) error {
	var errs []error

	pilotRequireContractVersion(&errs, "contractVersion", resp.ContractVersion, PilotSOPSourceHealthContractVersion)
	pilotRequireNonEmpty(&errs, "sourceId", resp.SourceID)
	pilotRequireAllowed(&errs, "status", resp.Status, allowedPilotSOPSourceStatuses)
	pilotRequireNonEmpty(&errs, "checkedAt", resp.CheckedAt)
	pilotRequireAllowed(&errs, "capabilityStatus.search", resp.CapabilityStatus.Search, allowedPilotCapabilityStatuses)
	pilotRequireAllowed(&errs, "capabilityStatus.preview", resp.CapabilityStatus.Preview, allowedPilotCapabilityStatuses)
	pilotRequireAllowed(&errs, "capabilityStatus.bodyFetch", resp.CapabilityStatus.BodyFetch, allowedPilotCapabilityStatuses)
	pilotRequireAllowed(&errs, "capabilityStatus.versionSnapshots", resp.CapabilityStatus.VersionSnapshots, allowedPilotCapabilityStatuses)
	if strings.TrimSpace(resp.CapabilityStatus.WebhookSync) != "" {
		pilotRequireAllowed(&errs, "capabilityStatus.webhookSync", resp.CapabilityStatus.WebhookSync, allowedPilotCapabilityStatuses)
	}

	if resp.CredentialDetailsVisible {
		errs = append(errs, fmt.Errorf("credentialDetailsVisible: must be false for browser-visible health responses"))
	}
	if resp.Status != PilotSOPSourceStatusHealthy && resp.Status != PilotSOPSourceStatusDisabled {
		pilotRequireNonEmpty(&errs, "safeMessage", resp.SafeMessage)
		pilotRequireNonEmpty(&errs, "recommendedAction", resp.RecommendedAction)
	}

	pilotAppendSecretLikeStringErrors(&errs, "sourceId", resp.SourceID)
	pilotAppendSecretLikeStringErrors(&errs, "checkedAt", resp.CheckedAt)
	pilotAppendSecretLikeStringErrors(&errs, "lastSuccessfulSyncAt", resp.LastSuccessfulSyncAt)
	pilotAppendSecretLikeStringErrors(&errs, "safeMessage", resp.SafeMessage)
	pilotAppendSecretLikeStringErrors(&errs, "recommendedAction", resp.RecommendedAction)
	for i, warning := range resp.Warnings {
		pilotAppendSecretLikeStringErrors(&errs, fmt.Sprintf("warnings[%d]", i), warning)
	}

	return errors.Join(errs...)
}

func ValidatePilotAuditEvent(event PilotAuditEvent) error {
	var errs []error

	pilotRequireContractVersion(&errs, "contractVersion", event.ContractVersion, PilotAuditEventContractVersion)
	pilotRequireNonEmpty(&errs, "eventId", event.EventID)
	pilotRequireAllowed(&errs, "eventType", event.EventType, allowedPilotAuditEventTypes)
	pilotRequireNonEmpty(&errs, "occurredAt", event.OccurredAt)
	pilotRequireAllowed(&errs, "actor.kind", event.Actor.Kind, allowedPilotAuditActorKinds)
	pilotRequireNonEmpty(&errs, "actor.id", event.Actor.ID)
	pilotRequireNonEmpty(&errs, "tenant.projectId", event.Tenant.ProjectID)
	pilotRequireNonEmpty(&errs, "tenant.environment", event.Tenant.Environment)
	pilotRequireNonEmpty(&errs, "resource.kind", event.Resource.Kind)
	pilotRequireNonEmpty(&errs, "action", event.Action)
	pilotRequireAllowed(&errs, "outcome", event.Outcome, allowedPilotAuditOutcomes)
	pilotRequireNonEmpty(&errs, "requestContext.incidentId", event.RequestContext.IncidentID)
	pilotRequireNonEmpty(&errs, "requestContext.serviceName", event.RequestContext.ServiceName)
	pilotRequireNonEmpty(&errs, "securityContext.serviceAccountProfile", event.SecurityContext.ServiceAccountProfile)

	if strings.HasPrefix(event.EventType, "sop.") {
		pilotRequireNonEmpty(&errs, "resource.sourceId", event.Resource.SourceID)
	}
	if event.Outcome == PilotAuditOutcomeDenied || event.Outcome == PilotAuditOutcomeDeferred || event.Outcome == PilotAuditOutcomeFailed {
		pilotRequireNonEmpty(&errs, "reason", event.Reason)
	}
	if event.SecurityContext.SecretRefVisible {
		errs = append(errs, fmt.Errorf("securityContext.secretRefVisible: must be false for audit responses"))
	}
	if event.SecurityContext.BrowserCredentialsUsed {
		errs = append(errs, fmt.Errorf("securityContext.browserCredentialsUsed: must be false; browser credentials are not allowed"))
	}

	pilotAppendSecretLikeStringErrors(&errs, "eventId", event.EventID)
	pilotAppendSecretLikeStringErrors(&errs, "actor.id", event.Actor.ID)
	pilotAppendSecretLikeStringErrors(&errs, "actor.displayName", event.Actor.DisplayName)
	pilotAppendSecretLikeStringErrors(&errs, "tenant.projectId", event.Tenant.ProjectID)
	pilotAppendSecretLikeStringErrors(&errs, "tenant.environment", event.Tenant.Environment)
	pilotAppendSecretLikeStringErrors(&errs, "resource.sourceId", event.Resource.SourceID)
	pilotAppendSecretLikeStringErrors(&errs, "resource.sopId", event.Resource.SOPID)
	pilotAppendSecretLikeStringErrors(&errs, "resource.version", event.Resource.Version)
	pilotAppendSecretLikeStringErrors(&errs, "reason", event.Reason)
	pilotAppendSecretLikeStringErrors(&errs, "requestContext.alertRuleId", event.RequestContext.AlertRuleID)
	pilotAppendSecretLikeStringErrors(&errs, "requestContext.incidentId", event.RequestContext.IncidentID)
	pilotAppendSecretLikeStringErrors(&errs, "requestContext.serviceName", event.RequestContext.ServiceName)
	pilotAppendSecretLikeStringErrors(&errs, "requestContext.severity", event.RequestContext.Severity)
	pilotAppendSecretLikeStringErrors(&errs, "securityContext.serviceAccountProfile", event.SecurityContext.ServiceAccountProfile)

	return errors.Join(errs...)
}

func ValidatePilotServiceAccountProfile(profile PilotServiceAccountProfile) error {
	var errs []error

	pilotRequireContractVersion(&errs, "contractVersion", profile.ContractVersion, PilotServiceAccountProfileContractVersion)
	pilotRequireNonEmpty(&errs, "profileId", profile.ProfileID)
	pilotRequireNonEmpty(&errs, "displayName", profile.DisplayName)
	pilotRequireNonEmpty(&errs, "purpose", profile.Purpose)
	if len(profile.AllowedActions) == 0 {
		errs = append(errs, fmt.Errorf("allowedActions: must include at least one action"))
	}
	for i, action := range profile.AllowedActions {
		pilotRequireAllowed(&errs, fmt.Sprintf("allowedActions[%d]", i), action, allowedPilotAuditEventTypes)
		pilotAppendSecretLikeStringErrors(&errs, fmt.Sprintf("allowedActions[%d]", i), action)
	}
	validatePilotTenantScope(&errs, "tenantScope", profile.TenantScope)
	if profile.SecretRefVisible {
		errs = append(errs, fmt.Errorf("secretRefVisible: must be false for browser-visible service-account profiles"))
	}
	if profile.BrowserUsable {
		errs = append(errs, fmt.Errorf("browserUsable: must be false; service accounts are server-side only"))
	}
	pilotRequireNonEmpty(&errs, "rotationPolicy", profile.RotationPolicy)

	pilotAppendSecretLikeStringErrors(&errs, "profileId", profile.ProfileID)
	pilotAppendSecretLikeStringErrors(&errs, "displayName", profile.DisplayName)
	pilotAppendSecretLikeStringErrors(&errs, "purpose", profile.Purpose)
	pilotAppendSecretLikeStringErrors(&errs, "rotationPolicy", profile.RotationPolicy)

	return errors.Join(errs...)
}

func ValidatePilotConfiguration(config PilotConfiguration) error {
	var errs []error

	pilotRequireContractVersion(&errs, "contractVersion", config.ContractVersion, PilotConfigurationContractVersion)
	pilotRequireNonEmpty(&errs, "projectId", config.ProjectID)
	pilotRequireNonEmpty(&errs, "environment", config.Environment)
	pilotRequireNonEmpty(&errs, "serviceName", config.ServiceName)
	pilotRequireAllowed(&errs, "auditMode", config.AuditMode, allowedPilotAuditModes)
	pilotRequireCapabilityFlag(&errs, "allowedCapabilities.search", config.AllowedCapabilities.Search)
	pilotRequireCapabilityFlag(&errs, "allowedCapabilities.preview", config.AllowedCapabilities.Preview)
	pilotRequireCapabilityFlag(&errs, "allowedCapabilities.bodyFetch", config.AllowedCapabilities.BodyFetch)
	pilotRequireCapabilityFlag(&errs, "allowedCapabilities.versionSnapshots", config.AllowedCapabilities.VersionSnapshots)
	if len(config.SelectedSources) == 0 {
		errs = append(errs, fmt.Errorf("selectedSources: must include exactly one source or ordered source priorities"))
	}
	if len(config.SelectedSources) > 1 {
		for i, source := range config.SelectedSources {
			if source.Priority == nil {
				errs = append(errs, fmt.Errorf("selectedSources[%d].priority: field is required when multiple sources are selected", i))
			}
		}
	}
	for i, source := range config.SelectedSources {
		path := fmt.Sprintf("selectedSources[%d]", i)
		pilotRequireNonEmpty(&errs, path+".sourceId", source.SourceID)
		pilotRequireNonEmpty(&errs, path+".serviceAccountProfile", source.ServiceAccountProfile)
		pilotAppendSecretLikeStringErrors(&errs, path+".sourceId", source.SourceID)
		pilotAppendSecretLikeStringErrors(&errs, path+".serviceAccountProfile", source.ServiceAccountProfile)
	}

	pilotAppendSecretLikeStringErrors(&errs, "projectId", config.ProjectID)
	pilotAppendSecretLikeStringErrors(&errs, "environment", config.Environment)
	pilotAppendSecretLikeStringErrors(&errs, "serviceName", config.ServiceName)
	pilotAppendSecretLikeStringErrors(&errs, "rolloutId", config.RolloutID)

	return errors.Join(errs...)
}

var allowedPilotSOPSourceKinds = map[string]struct{}{
	PilotSOPSourceKindURLRegistry:     {},
	PilotSOPSourceKindManagedMarkdown: {},
	PilotSOPSourceKindGitMarkdown:     {},
	PilotSOPSourceKindConfluence:      {},
	PilotSOPSourceKindNotion:          {},
	PilotSOPSourceKindSharePoint:      {},
	PilotSOPSourceKindCustomConnector: {},
}

var allowedPilotSOPSourceAuthModes = map[string]struct{}{
	PilotSOPSourceAuthModeNone:                     {},
	PilotSOPSourceAuthModePublicURL:                {},
	PilotSOPSourceAuthModeServerSideServiceAccount: {},
	PilotSOPSourceAuthModeServerSideOAuth:          {},
	PilotSOPSourceAuthModeServerSideAPIKey:         {},
	PilotSOPSourceAuthModeCustomSecretRef:          {},
}

var allowedPilotSOPSourceStatuses = map[string]struct{}{
	PilotSOPSourceStatusHealthy:      {},
	PilotSOPSourceStatusDegraded:     {},
	PilotSOPSourceStatusUnconfigured: {},
	PilotSOPSourceStatusUnauthorized: {},
	PilotSOPSourceStatusUnreachable:  {},
	PilotSOPSourceStatusStale:        {},
	PilotSOPSourceStatusDisabled:     {},
}

var allowedPilotCapabilityStatuses = map[string]struct{}{
	PilotCapabilityStatusHealthy:      {},
	PilotCapabilityStatusDegraded:     {},
	PilotCapabilityStatusDisabled:     {},
	PilotCapabilityStatusUnauthorized: {},
	PilotCapabilityStatusUnreachable:  {},
	PilotCapabilityStatusUnknown:      {},
}

var allowedPilotAuditEventTypes = map[string]struct{}{
	PilotAuditEventTypeSOPSearch:              {},
	PilotAuditEventTypeSOPPreview:             {},
	PilotAuditEventTypeSOPFetch:               {},
	PilotAuditEventTypeSOPHealthCheck:         {},
	PilotAuditEventTypeEvidenceCollectRequest: {},
	PilotAuditEventTypeEvidenceCollectResult:  {},
	PilotAuditEventTypeAISummaryRequest:       {},
	PilotAuditEventTypeAISummaryResult:        {},
}

var allowedPilotAuditOutcomes = map[string]struct{}{
	PilotAuditOutcomeAllowed:  {},
	PilotAuditOutcomeDenied:   {},
	PilotAuditOutcomeRedacted: {},
	PilotAuditOutcomeFailed:   {},
	PilotAuditOutcomeDeferred: {},
}

var allowedPilotAuditActorKinds = map[string]struct{}{
	PilotAuditActorKindUser:           {},
	PilotAuditActorKindSystem:         {},
	PilotAuditActorKindServiceAccount: {},
}

var allowedPilotAuditModes = map[string]struct{}{
	PilotAuditModeRequired: {},
	PilotAuditModeDeferred: {},
	PilotAuditModeDisabled: {},
}

var pilotJWTLikePattern = regexp.MustCompile(`\b[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b`)

func pilotRequireContractVersion(errs *[]error, field string, got string, want string) {
	if strings.TrimSpace(got) != want {
		*errs = append(*errs, fmt.Errorf("%s: expected %q, got %q", field, want, got))
	}
}

func pilotRequireNonEmpty(errs *[]error, field string, value string) {
	if strings.TrimSpace(value) == "" {
		*errs = append(*errs, fmt.Errorf("%s: field is required", field))
	}
}

func pilotRequireAllowed(errs *[]error, field string, value string, allowed map[string]struct{}) {
	value = strings.TrimSpace(value)
	if value == "" {
		*errs = append(*errs, fmt.Errorf("%s: field is required", field))
		return
	}
	if _, ok := allowed[value]; !ok {
		*errs = append(*errs, fmt.Errorf("%s: unsupported value %q", field, value))
	}
}

func pilotRequireCapabilityFlag(errs *[]error, field string, value *bool) {
	if value == nil {
		*errs = append(*errs, fmt.Errorf("%s: field is required", field))
	}
}

func pilotAuthModeRequiresServiceAccount(authMode string) bool {
	switch authMode {
	case PilotSOPSourceAuthModeServerSideServiceAccount,
		PilotSOPSourceAuthModeServerSideOAuth,
		PilotSOPSourceAuthModeServerSideAPIKey,
		PilotSOPSourceAuthModeCustomSecretRef:
		return true
	default:
		return false
	}
}

func pilotAppendSecretLikeStringErrors(errs *[]error, field string, value string) {
	if hasPilotSecretLikeValue(value) {
		*errs = append(*errs, fmt.Errorf("%s: contains secret-like value", field))
	}
}

func hasPilotSecretLikeValue(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return false
	}

	for _, forbidden := range []string{
		"token=",
		"access_token",
		"client_secret",
		"api_key",
		"apikey",
		"password",
		"secret=",
		"bearer ",
	} {
		if strings.Contains(normalized, forbidden) {
			return true
		}
	}

	if strings.Contains(normalized, "-----begin") && strings.Contains(normalized, "private key-----") {
		return true
	}

	return pilotJWTLikePattern.MatchString(value)
}
