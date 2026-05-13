package ruletypes

import (
	"errors"
	"fmt"
	"strings"
)

const (
	PilotSOPFetchContractVersion = "ds-apm.sop-fetch.v1"

	PilotSOPFetchStatusFetched           = "fetched"
	PilotSOPFetchStatusDenied            = "denied"
	PilotSOPFetchStatusNotFound          = "not_found"
	PilotSOPFetchStatusSourceUnavailable = "source_unavailable"
	PilotSOPFetchStatusRedacted          = "redacted"

	PilotSOPFetchBodyMarkdownMaxBytes = 256 * 1024

	pilotSOPDocumentBodyExceedsMaxSizeWarning = "sop_document_body_exceeds_max_size"
	pilotSOPDocumentBodyNotMarkdownWarning    = "sop_document_body_not_markdown_like"
)

type PilotManagedMarkdownSource struct {
	SourceID              string                         `json:"sourceId"`
	DisplayName           string                         `json:"displayName"`
	Status                string                         `json:"status"`
	LastHealthCheckAt     string                         `json:"lastHealthCheckAt,omitempty"`
	LastSyncAt            string                         `json:"lastSyncAt,omitempty"`
	ServiceAccountProfile string                         `json:"serviceAccountProfile"`
	TenantScope           PilotTenantScope               `json:"tenantScope"`
	ConfiguredBy          string                         `json:"configuredBy,omitempty"`
	Documents             []PilotManagedMarkdownDocument `json:"documents,omitempty"`
	Warnings              []string                       `json:"warnings,omitempty"`
}

type PilotManagedMarkdownDocument struct {
	SOPID        string   `json:"sopId"`
	Version      string   `json:"version,omitempty"`
	Title        string   `json:"title"`
	BodyMarkdown string   `json:"bodyMarkdown"`
	DisplayURL   string   `json:"displayUrl,omitempty"`
	UpdatedAt    string   `json:"updatedAt,omitempty"`
	Tags         []string `json:"tags,omitempty"`
}

type PilotManagedMarkdownSOPFetchRequest struct {
	Source PilotManagedMarkdownSource `json:"source"`
	Fetch  PilotSOPFetchRequest       `json:"fetch"`
}

type PilotSOPFetchRequest struct {
	SourceID              string                   `json:"sourceId"`
	SOPID                 string                   `json:"sopId"`
	Version               string                   `json:"version,omitempty"`
	OccurredAt            string                   `json:"occurredAt"`
	AuditEventID          string                   `json:"auditEventId,omitempty"`
	AuditMode             string                   `json:"auditMode"`
	AuditAccepted         bool                     `json:"auditAccepted"`
	Actor                 PilotAuditActor          `json:"actor"`
	Tenant                PilotAuditTenant         `json:"tenant"`
	RequestContext        PilotAuditRequestContext `json:"requestContext"`
	ServiceAccountProfile string                   `json:"serviceAccountProfile"`
}

type PilotSOPFetchResponse struct {
	ContractVersion string                    `json:"contractVersion"`
	Status          string                    `json:"status"`
	SourceID        string                    `json:"sourceId"`
	SOPID           string                    `json:"sopId"`
	Version         string                    `json:"version,omitempty"`
	Title           string                    `json:"title,omitempty"`
	DisplayURL      string                    `json:"displayUrl,omitempty"`
	BodyMarkdown    string                    `json:"bodyMarkdown,omitempty"`
	AuditEvent      PilotAuditEvent           `json:"auditEvent"`
	SecurityContext PilotAuditSecurityContext `json:"securityContext"`
	Warnings        []string                  `json:"warnings,omitempty"`
}

func NewPilotManagedMarkdownCatalog(sources []PilotManagedMarkdownSource) (PilotSOPSourceCatalogResponse, error) {
	resp := PilotSOPSourceCatalogResponse{
		ContractVersion: PilotSOPSourceCatalogContractVersion,
		Sources:         make([]PilotSOPSource, 0, len(sources)),
	}

	for _, source := range sources {
		resp.Sources = append(resp.Sources, PilotSOPSource{
			SourceID:          strings.TrimSpace(source.SourceID),
			DisplayName:       strings.TrimSpace(source.DisplayName),
			Kind:              PilotSOPSourceKindManagedMarkdown,
			AuthMode:          PilotSOPSourceAuthModeServerSideServiceAccount,
			Status:            firstNonEmpty(source.Status, PilotSOPSourceStatusUnconfigured),
			LastHealthCheckAt: strings.TrimSpace(source.LastHealthCheckAt),
			LastSyncAt:        strings.TrimSpace(source.LastSyncAt),
			Capabilities: PilotSOPSourceCapabilities{
				Search:           pilotBoolValue(true),
				Preview:          pilotBoolValue(true),
				BodyFetch:        pilotBoolValue(true),
				VersionSnapshots: pilotBoolValue(true),
				WebhookSync:      pilotBoolValue(false),
			},
			ServiceAccountProfile: strings.TrimSpace(source.ServiceAccountProfile),
			SecretRefVisible:      false,
			ConfiguredBy:          strings.TrimSpace(source.ConfiguredBy),
			Warnings:              source.Warnings,
		})
	}

	return resp, ValidatePilotSOPSourceCatalog(resp)
}

func NewPilotManagedMarkdownHealth(source PilotManagedMarkdownSource, checkedAt string) (PilotSOPSourceHealthResponse, error) {
	status := firstNonEmpty(source.Status, PilotSOPSourceStatusUnconfigured)
	hasDocuments := len(source.Documents) > 0
	resp := PilotSOPSourceHealthResponse{
		ContractVersion: PilotSOPSourceHealthContractVersion,
		SourceID:        strings.TrimSpace(source.SourceID),
		Status:          status,
		CheckedAt:       strings.TrimSpace(checkedAt),
		CapabilityStatus: PilotSOPSourceCapabilityStatus{
			Search:           pilotCapabilityStatusForSource(status, hasDocuments),
			Preview:          pilotCapabilityStatusForSource(status, hasDocuments),
			BodyFetch:        pilotCapabilityStatusForSource(status, hasDocuments),
			VersionSnapshots: pilotCapabilityStatusForSource(status, hasDocuments),
		},
		LastSuccessfulSyncAt:     strings.TrimSpace(source.LastSyncAt),
		CredentialDetailsVisible: false,
		Warnings:                 source.Warnings,
	}

	if status != PilotSOPSourceStatusHealthy {
		resp.SafeMessage = "Managed Markdown SOP source is not fully healthy."
		resp.RecommendedAction = "Check managed markdown source registration and audit-enabled fetch configuration."
	}
	if !hasDocuments {
		resp.SafeMessage = "Managed Markdown SOP source has no registered documents."
		resp.RecommendedAction = "Register at least one managed markdown SOP before enabling pilot fetch."
	}

	return resp, ValidatePilotSOPSourceHealth(resp)
}

func FetchPilotManagedMarkdownSOP(source PilotManagedMarkdownSource, req PilotSOPFetchRequest) (PilotSOPFetchResponse, error) {
	req = normalizePilotSOPFetchRequest(source, req)
	securityContext := PilotAuditSecurityContext{
		ServiceAccountProfile:  firstNonEmpty(req.ServiceAccountProfile, source.ServiceAccountProfile),
		SecretRefVisible:       false,
		BrowserCredentialsUsed: false,
		RedactionApplied:       true,
	}

	if source.Status != PilotSOPSourceStatusHealthy && source.Status != PilotSOPSourceStatusDegraded {
		return pilotSOPFetchDenied(req, securityContext, PilotSOPFetchStatusSourceUnavailable, PilotAuditOutcomeFailed, "source_not_available", nil)
	}
	if req.AuditMode != PilotAuditModeRequired || !req.AuditAccepted {
		return pilotSOPFetchDenied(req, securityContext, PilotSOPFetchStatusDenied, PilotAuditOutcomeDenied, "live_fetch_blocked_until_audit_contract_accepted", []string{
			pilotBodyFetchDisabledUntilAuditContractEnabledWarning,
		})
	}
	if !PilotTenantScopeAllows(source.TenantScope, req.Tenant) {
		return pilotSOPFetchDenied(req, securityContext, PilotSOPFetchStatusDenied, PilotAuditOutcomeDenied, "tenant_scope_denied", []string{
			SOPTenantPolicyDeniedWarning,
		})
	}

	doc, ok := findPilotManagedMarkdownDocument(source.Documents, req.SOPID, req.Version)
	if !ok {
		return pilotSOPFetchDenied(req, securityContext, PilotSOPFetchStatusNotFound, PilotAuditOutcomeFailed, "sop_document_not_found", nil)
	}
	if warning := pilotManagedMarkdownDocumentWarning(doc); warning != "" {
		return pilotSOPFetchDenied(req, securityContext, PilotSOPFetchStatusRedacted, PilotAuditOutcomeRedacted, warning, []string{warning})
	}

	auditEvent := pilotSOPFetchAuditEvent(req, securityContext, PilotAuditOutcomeAllowed, "managed_markdown_fetch_allowed")
	resp := PilotSOPFetchResponse{
		ContractVersion: PilotSOPFetchContractVersion,
		Status:          PilotSOPFetchStatusFetched,
		SourceID:        req.SourceID,
		SOPID:           req.SOPID,
		Version:         doc.Version,
		Title:           doc.Title,
		DisplayURL:      doc.DisplayURL,
		BodyMarkdown:    doc.BodyMarkdown,
		AuditEvent:      auditEvent,
		SecurityContext: securityContext,
	}

	return resp, ValidatePilotSOPFetchResponse(resp)
}

func ValidatePilotSOPFetchResponse(resp PilotSOPFetchResponse) error {
	var errs []error

	pilotRequireContractVersion(&errs, "contractVersion", resp.ContractVersion, PilotSOPFetchContractVersion)
	pilotRequireAllowed(&errs, "status", resp.Status, allowedPilotSOPFetchStatuses)
	pilotRequireNonEmpty(&errs, "sourceId", resp.SourceID)
	pilotRequireNonEmpty(&errs, "sopId", resp.SOPID)
	if resp.Status == PilotSOPFetchStatusFetched {
		pilotRequireNonEmpty(&errs, "bodyMarkdown", resp.BodyMarkdown)
		if len(resp.BodyMarkdown) > PilotSOPFetchBodyMarkdownMaxBytes {
			errs = append(errs, fmt.Errorf("bodyMarkdown: exceeds max size of %d bytes", PilotSOPFetchBodyMarkdownMaxBytes))
		}
		if pilotBodyMarkdownLooksNonMarkdown(resp.BodyMarkdown) {
			errs = append(errs, fmt.Errorf("bodyMarkdown: payload does not look like markdown"))
		}
	}
	if resp.SecurityContext.SecretRefVisible {
		errs = append(errs, fmt.Errorf("securityContext.secretRefVisible: must be false for SOP fetch responses"))
	}
	if resp.SecurityContext.BrowserCredentialsUsed {
		errs = append(errs, fmt.Errorf("securityContext.browserCredentialsUsed: must be false for SOP fetch responses"))
	}
	if resp.Status != PilotSOPFetchStatusFetched && strings.TrimSpace(resp.BodyMarkdown) != "" {
		errs = append(errs, fmt.Errorf("bodyMarkdown: must be empty unless status is fetched"))
	}

	pilotAppendSecretLikeStringErrors(&errs, "sourceId", resp.SourceID)
	pilotAppendSecretLikeStringErrors(&errs, "sopId", resp.SOPID)
	pilotAppendSecretLikeStringErrors(&errs, "version", resp.Version)
	pilotAppendSecretLikeStringErrors(&errs, "title", resp.Title)
	pilotAppendSecretLikeStringErrors(&errs, "displayUrl", resp.DisplayURL)
	pilotAppendSecretLikeStringErrors(&errs, "bodyMarkdown", resp.BodyMarkdown)
	for i, warning := range resp.Warnings {
		pilotAppendSecretLikeStringErrors(&errs, fmt.Sprintf("warnings[%d]", i), warning)
	}
	if err := ValidatePilotAuditEvent(resp.AuditEvent); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

var allowedPilotSOPFetchStatuses = map[string]struct{}{
	PilotSOPFetchStatusFetched:           {},
	PilotSOPFetchStatusDenied:            {},
	PilotSOPFetchStatusNotFound:          {},
	PilotSOPFetchStatusSourceUnavailable: {},
	PilotSOPFetchStatusRedacted:          {},
}

func normalizePilotSOPFetchRequest(source PilotManagedMarkdownSource, req PilotSOPFetchRequest) PilotSOPFetchRequest {
	req.SourceID = strings.TrimSpace(firstNonEmpty(req.SourceID, source.SourceID))
	req.SOPID = strings.TrimSpace(req.SOPID)
	req.Version = strings.TrimSpace(req.Version)
	req.AuditMode = strings.TrimSpace(firstNonEmpty(req.AuditMode, PilotAuditModeRequired))
	req.AuditEventID = strings.TrimSpace(firstNonEmpty(
		req.AuditEventID,
		fmt.Sprintf("audit-%s-%s-fetch", req.SourceID, req.SOPID),
	))
	req.ServiceAccountProfile = strings.TrimSpace(firstNonEmpty(req.ServiceAccountProfile, source.ServiceAccountProfile))
	return req
}

func pilotSOPFetchDenied(
	req PilotSOPFetchRequest,
	securityContext PilotAuditSecurityContext,
	status string,
	outcome string,
	reason string,
	warnings []string,
) (PilotSOPFetchResponse, error) {
	auditEvent := pilotSOPFetchAuditEvent(req, securityContext, outcome, reason)
	resp := PilotSOPFetchResponse{
		ContractVersion: PilotSOPFetchContractVersion,
		Status:          status,
		SourceID:        req.SourceID,
		SOPID:           req.SOPID,
		AuditEvent:      auditEvent,
		SecurityContext: securityContext,
		Warnings:        warnings,
	}

	return resp, ValidatePilotSOPFetchResponse(resp)
}

func pilotSOPFetchAuditEvent(req PilotSOPFetchRequest, securityContext PilotAuditSecurityContext, outcome string, reason string) PilotAuditEvent {
	return PilotAuditEvent{
		ContractVersion: PilotAuditEventContractVersion,
		EventID:         req.AuditEventID,
		EventType:       PilotAuditEventTypeSOPFetch,
		OccurredAt:      req.OccurredAt,
		Actor:           req.Actor,
		Tenant:          req.Tenant,
		Resource: PilotAuditResource{
			Kind:     "sop_source",
			SourceID: req.SourceID,
			SOPID:    req.SOPID,
			Version:  req.Version,
		},
		Action:          "fetch",
		Outcome:         outcome,
		Reason:          reason,
		RequestContext:  req.RequestContext,
		SecurityContext: securityContext,
	}
}

func findPilotManagedMarkdownDocument(docs []PilotManagedMarkdownDocument, sopID string, version string) (PilotManagedMarkdownDocument, bool) {
	sopID = strings.TrimSpace(sopID)
	version = strings.TrimSpace(version)
	for _, doc := range docs {
		if strings.TrimSpace(doc.SOPID) != sopID {
			continue
		}
		if version != "" && strings.TrimSpace(doc.Version) != version {
			continue
		}

		return doc, true
	}

	return PilotManagedMarkdownDocument{}, false
}

func pilotManagedMarkdownDocumentWarning(doc PilotManagedMarkdownDocument) string {
	if hasPilotSecretLikeValue(doc.Title) || hasPilotSecretLikeValue(doc.BodyMarkdown) || hasPilotSecretLikeValue(doc.DisplayURL) {
		return "sop_document_contains_secret_like_value"
	}
	if len(doc.BodyMarkdown) > PilotSOPFetchBodyMarkdownMaxBytes {
		return pilotSOPDocumentBodyExceedsMaxSizeWarning
	}
	if pilotBodyMarkdownLooksNonMarkdown(doc.BodyMarkdown) {
		return pilotSOPDocumentBodyNotMarkdownWarning
	}
	if strings.TrimSpace(doc.DisplayURL) == "" {
		return ""
	}
	if _, warning, ok := safeDisplayURL(doc.DisplayURL); !ok {
		return warning
	}

	return ""
}

var pilotNonMarkdownMagicPrefixes = []string{
	"<!doctype html",
	"<html",
	"%pdf-",
	"pk\x03\x04",
}

func pilotBodyMarkdownLooksNonMarkdown(body string) bool {
	if strings.IndexByte(body, 0x00) >= 0 {
		return true
	}
	leading := strings.ToLower(strings.TrimLeft(strings.TrimPrefix(body, "\ufeff"), " \t\r\n"))
	for _, prefix := range pilotNonMarkdownMagicPrefixes {
		if strings.HasPrefix(leading, prefix) {
			return true
		}
	}

	return false
}

func pilotCapabilityStatusForSource(status string, hasDocuments bool) string {
	if !hasDocuments {
		return PilotCapabilityStatusDisabled
	}

	switch status {
	case PilotSOPSourceStatusHealthy:
		return PilotCapabilityStatusHealthy
	case PilotSOPSourceStatusDegraded:
		return PilotCapabilityStatusDegraded
	case PilotSOPSourceStatusUnauthorized:
		return PilotCapabilityStatusUnauthorized
	case PilotSOPSourceStatusUnreachable:
		return PilotCapabilityStatusUnreachable
	case PilotSOPSourceStatusDisabled, PilotSOPSourceStatusUnconfigured:
		return PilotCapabilityStatusDisabled
	default:
		return PilotCapabilityStatusUnknown
	}
}

func pilotBoolValue(value bool) *bool {
	return &value
}
