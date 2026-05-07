package ruletypes

import (
	"net/url"
	"strings"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
)

const (
	SOPPreviewContractVersion = "ds-apm.sop-preview.v1"

	SOPPreviewStatusBound      = "bound"
	SOPPreviewStatusInvalidURL = "invalid_url"
	SOPPreviewStatusMissing    = "missing"

	SOPPreviewAccessModeInvalidURLCredentials = "invalid_url_credentials"
	SOPPreviewAccessModeInvalidURL            = "invalid_url"
	SOPPreviewAccessModeMetadataOnly          = "metadata_only"
	SOPPreviewAccessModePublicURL             = "public_url"
	SOPPreviewAccessModeServerSideConnector   = "server_side_connector"

	SOPPreviewCredentialScopeManualPublicURL        = "manual_public_url"
	SOPPreviewCredentialScopeNone                   = "none"
	SOPPreviewCredentialScopeSourceConnectorSecrets = "source_connector_secret"

	SOPPreviewRecommendedServiceAccountProfile = "ds-sop-reader"

	sopPreviewURLCredentialWarning = "Do not put credentials in SOP URLs; store connector credentials server-side."
	sopPreviewURLSchemeWarning     = "Use an http:// or https:// SOP URL."
)

type PreviewSOPRequest struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type PreviewSOPResponse struct {
	ContractVersion string             `json:"contractVersion"`
	Status          string             `json:"status"`
	Source          SOPSourcePreview   `json:"source"`
	Binding         SOPBindingPreview  `json:"binding"`
	Search          SOPSearchPreview   `json:"search"`
	Preview         SOPDocumentPreview `json:"preview"`
	Access          SOPAccessPreview   `json:"access"`
	Warnings        []string           `json:"warnings,omitempty"`
}

type SOPSourcePreview struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type SOPBindingPreview struct {
	SOPID     string `json:"sopId,omitempty"`
	BindingID string `json:"bindingId,omitempty"`
	Version   string `json:"version,omitempty"`
	Title     string `json:"title,omitempty"`
}

type SOPSearchPreview struct {
	Query string   `json:"query"`
	Terms []string `json:"terms"`
}

type SOPDocumentPreview struct {
	Available  bool   `json:"available"`
	Title      string `json:"title,omitempty"`
	URL        string `json:"url,omitempty"`
	DisplayURL string `json:"displayUrl,omitempty"`
}

type SOPAccessPreview struct {
	Mode                             string `json:"mode"`
	RequiresServerSideFetch          bool   `json:"requiresServerSideFetch"`
	BrowserCredentialsAllowed        bool   `json:"browserCredentialsAllowed"`
	RecommendedServiceAccountProfile string `json:"recommendedServiceAccountProfile,omitempty"`
	CredentialScope                  string `json:"credentialScope"`
	AuditEventRequired               bool   `json:"auditEventRequired"`
	Message                          string `json:"message"`
}

func PreviewSOP(req PreviewSOPRequest) *PreviewSOPResponse {
	labels := req.Labels
	if labels == nil {
		labels = map[string]string{}
	}

	annotations := req.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}

	sopID := strings.TrimSpace(labels[alertmanagertypes.IncidentLabelSopID])
	sopURL := strings.TrimSpace(annotations[alertmanagertypes.IncidentAnnotationSopURL])
	sopSource := strings.TrimSpace(annotations[alertmanagertypes.IncidentAnnotationSopSource])
	sopTitle := strings.TrimSpace(annotations[alertmanagertypes.IncidentAnnotationSopTitle])
	sopVersion := strings.TrimSpace(annotations[alertmanagertypes.IncidentAnnotationSopVersion])
	sopBindingID := strings.TrimSpace(annotations[alertmanagertypes.IncidentAnnotationSopBindingID])

	source := SOPSourcePreview{
		Kind: "manual_metadata",
		Name: "Alert rule metadata",
	}
	if sopSource != "" {
		source.Kind = "configured_source"
		source.Name = sopSource
	}

	status := SOPPreviewStatusBound
	warnings := make([]string, 0)
	if sopID == "" && sopURL == "" {
		status = SOPPreviewStatusMissing
		warnings = append(warnings, "Add sop_id or sop_url before production use.")
	}

	documentPreview := SOPDocumentPreview{
		Title: firstNonEmpty(sopTitle, sopID, "SOP preview unavailable"),
	}
	urlWarning := ""
	if sopURL != "" {
		if displayURL, warning, ok := safeDisplayURL(sopURL); ok {
			documentPreview.Available = true
			documentPreview.URL = sopURL
			documentPreview.DisplayURL = displayURL
		} else {
			urlWarning = warning
			status = SOPPreviewStatusInvalidURL
			warnings = append(warnings, warning)
		}
	}

	terms := uniqueNonEmpty(
		sopID,
		sopBindingID,
		sopTitle,
		sopVersion,
		sopSource,
		strings.TrimSpace(labels[alertmanagertypes.IncidentLabelServiceName]),
		strings.TrimSpace(labels[alertmanagertypes.IncidentLabelEnvironment]),
		strings.TrimSpace(labels[alertmanagertypes.IncidentLabelSeverity]),
	)

	return &PreviewSOPResponse{
		ContractVersion: SOPPreviewContractVersion,
		Status:          status,
		Source:          source,
		Binding: SOPBindingPreview{
			SOPID:     sopID,
			BindingID: sopBindingID,
			Version:   sopVersion,
			Title:     sopTitle,
		},
		Search: SOPSearchPreview{
			Query: strings.Join(terms, " "),
			Terms: terms,
		},
		Preview:  documentPreview,
		Access:   previewSOPAccess(sopSource, sopURL, urlWarning),
		Warnings: warnings,
	}
}

func safeDisplayURL(rawURL string) (string, string, bool) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", sopPreviewURLSchemeWarning, false
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", sopPreviewURLSchemeWarning, false
	}

	if parsedURL.User != nil || hasSensitiveQueryParam(parsedURL.Query()) {
		return "", sopPreviewURLCredentialWarning, false
	}

	displayURL := parsedURL.Host + parsedURL.Path
	if displayURL == "" {
		return "", sopPreviewURLSchemeWarning, false
	}

	return displayURL, "", true
}

func previewSOPAccess(sopSource string, sopURL string, urlWarning string) SOPAccessPreview {
	access := SOPAccessPreview{
		Mode:                      SOPPreviewAccessModeMetadataOnly,
		BrowserCredentialsAllowed: false,
		CredentialScope:           SOPPreviewCredentialScopeNone,
		Message:                   "SOP preview is metadata-only until a source or URL is configured.",
	}

	switch {
	case urlWarning == sopPreviewURLCredentialWarning:
		access.Mode = SOPPreviewAccessModeInvalidURLCredentials
		access.RequiresServerSideFetch = true
		access.RecommendedServiceAccountProfile = SOPPreviewRecommendedServiceAccountProfile
		access.CredentialScope = SOPPreviewCredentialScopeSourceConnectorSecrets
		access.AuditEventRequired = true
		access.Message = "Credential-bearing SOP URLs are blocked; configure source credentials server-side."
	case urlWarning != "":
		access.Mode = SOPPreviewAccessModeInvalidURL
		access.Message = "SOP URL must be HTTP(S) before it can be previewed."
	case sopSource != "":
		access.Mode = SOPPreviewAccessModeServerSideConnector
		access.RequiresServerSideFetch = true
		access.RecommendedServiceAccountProfile = SOPPreviewRecommendedServiceAccountProfile
		access.CredentialScope = SOPPreviewCredentialScopeSourceConnectorSecrets
		access.AuditEventRequired = true
		access.Message = "Live SOP content must be fetched server-side with source connector credentials; browser credentials are never accepted."
	case sopURL != "":
		access.Mode = SOPPreviewAccessModePublicURL
		access.CredentialScope = SOPPreviewCredentialScopeManualPublicURL
		access.Message = "Browser can open the public SOP URL, but credentials must not be embedded in alert metadata."
	}

	return access
}

func hasSensitiveQueryParam(values url.Values) bool {
	for key := range values {
		if isSensitiveQueryKey(key) {
			return true
		}
	}

	return false
}

func isSensitiveQueryKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "-", "_")

	switch key {
	case "access_token", "api_key", "apikey", "auth", "authorization", "bearer", "password", "secret", "token":
		return true
	default:
		return false
	}
}

func uniqueNonEmpty(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return ""
}
