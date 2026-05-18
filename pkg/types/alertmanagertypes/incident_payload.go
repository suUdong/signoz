package alertmanagertypes

import (
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/prometheus/alertmanager/template"
)

const RedactedIncidentValue = "[redacted]"

type IncidentField struct {
	Key   string
	Title string
	Value string
	Short bool
}

var incidentJWTLikePattern = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{10,}\b`)

var (
	incidentValueEmailPattern        = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)
	incidentValueKoreanMobilePattern = regexp.MustCompile(`(?:\+?82[-\s]?)?0?1[016789][-)\s]?\d{3,4}[-\s]?\d{4}`)
	incidentValueLongSecretPattern   = regexp.MustCompile(`\b[A-Za-z0-9_\-]{32,}\b`)
)

var incidentSensitiveURLKeys = map[string]struct{}{
	"access_token":  {},
	"api_key":       {},
	"apikey":        {},
	"auth":          {},
	"authorization": {},
	"bearer":        {},
	"client_secret": {},
	"password":      {},
	"secret":        {},
	"token":         {},
}

func BuildSafeIncidentInfo(labels, annotations template.KV) IncidentInfo {
	return SanitizeIncidentInfo(BuildIncidentInfo(labels, annotations))
}

func SanitizeIncidentInfo(info IncidentInfo) IncidentInfo {
	return IncidentInfo{
		ProjectID:        SanitizeIncidentValue(info.ProjectID),
		Environment:      SanitizeIncidentValue(info.Environment),
		ServiceName:      SanitizeIncidentValue(info.ServiceName),
		OwnerTeam:        SanitizeIncidentValue(info.OwnerTeam),
		Severity:         SanitizeIncidentValue(info.Severity),
		ImpactSummary:    SanitizeIncidentValue(info.ImpactSummary),
		NextAction:       SanitizeIncidentValue(info.NextAction),
		VendorRequest:    SanitizeIncidentValue(info.VendorRequest),
		CustomerUpdate:   SanitizeIncidentValue(info.CustomerUpdate),
		SopID:            SanitizeIncidentValue(info.SopID),
		SopURL:           SanitizeIncidentValue(info.SopURL),
		SopSource:        SanitizeIncidentValue(info.SopSource),
		SopTitle:         SanitizeIncidentValue(info.SopTitle),
		SopVersion:       SanitizeIncidentValue(info.SopVersion),
		SopBindingID:     SanitizeIncidentValue(info.SopBindingID),
		AIStrategyID:     SanitizeIncidentValue(info.AIStrategyID),
		AIStrategyStatus: SanitizeIncidentValue(info.AIStrategyStatus),
		AIHeadline:       SanitizeIncidentValue(info.AIHeadline),
		AIFirstActions:   SanitizeIncidentValue(info.AIFirstActions),
		AIConfidence:     SanitizeIncidentValue(info.AIConfidence),
		AILimitations:    SanitizeIncidentValue(info.AILimitations),
		AIEvidenceRefs:   SanitizeIncidentValue(info.AIEvidenceRefs),
	}
}

func SanitizeIncidentValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if sanitizedURL, ok := sanitizeIncidentURL(value); ok {
		value = sanitizedURL
	}
	if incidentValueLooksSecret(value) {
		return RedactedIncidentValue
	}

	value = redactIncidentPII(value)
	value = redactIncidentLongSecret(value)

	return value
}

func redactIncidentPII(value string) string {
	value = incidentValueEmailPattern.ReplaceAllString(value, "[redacted-email]")
	value = incidentValueKoreanMobilePattern.ReplaceAllString(value, "[redacted-phone]")
	return value
}

func redactIncidentLongSecret(value string) string {
	return incidentValueLongSecretPattern.ReplaceAllString(value, "[redacted-secret]")
}

func IncidentInfoFields(info IncidentInfo) []IncidentField {
	info = SanitizeIncidentInfo(info)
	fields := []IncidentField{
		{Key: "project_id", Title: "Project", Value: info.ProjectID, Short: true},
		{Key: "environment", Title: "Environment", Value: info.Environment, Short: true},
		{Key: "service_name", Title: "Service", Value: info.ServiceName, Short: true},
		{Key: "owner_team", Title: "Owner team", Value: info.OwnerTeam, Short: true},
		{Key: "severity", Title: "Severity", Value: info.Severity, Short: true},
		{Key: "impact_summary", Title: "Impact", Value: info.ImpactSummary},
		{Key: "next_action", Title: "Next action", Value: info.NextAction},
		{Key: "vendor_request", Title: "Vendor request", Value: info.VendorRequest},
		{Key: "customer_update", Title: "Customer update", Value: info.CustomerUpdate},
		{Key: "sop_id", Title: "SOP ID", Value: info.SopID, Short: true},
		{Key: "sop_title", Title: "SOP title", Value: info.SopTitle},
		{Key: "sop_version", Title: "SOP version", Value: info.SopVersion, Short: true},
		{Key: "sop_source", Title: "SOP source", Value: info.SopSource, Short: true},
		{Key: "sop_url", Title: "SOP URL", Value: info.SopURL},
		{Key: "sop_binding_id", Title: "SOP binding", Value: info.SopBindingID},
		{Key: "ai_strategy_id", Title: "AI strategy ID", Value: info.AIStrategyID},
		{Key: "ai_strategy_status", Title: "AI status", Value: info.AIStrategyStatus, Short: true},
		{Key: "ai_confidence", Title: "AI confidence", Value: info.AIConfidence, Short: true},
		{Key: "ai_headline", Title: "AI headline", Value: info.AIHeadline},
		{Key: "ai_first_actions", Title: "AI first actions", Value: info.AIFirstActions},
		{Key: "ai_limitations", Title: "AI limitations", Value: info.AILimitations},
		{Key: "ai_evidence_refs", Title: "AI evidence refs", Value: info.AIEvidenceRefs},
	}

	result := make([]IncidentField, 0, len(fields))
	for _, field := range fields {
		if strings.TrimSpace(field.Value) == "" {
			continue
		}
		result = append(result, field)
	}

	return result
}

func IncidentInfoDetails(info IncidentInfo) map[string]string {
	fields := IncidentInfoFields(info)
	if len(fields) == 0 {
		return nil
	}

	details := make(map[string]string, len(fields))
	for _, field := range fields {
		details[field.Key] = field.Value
	}

	return details
}

func sanitizeIncidentURL(value string) (string, bool) {
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", false
	}

	parsed.User = nil
	query := parsed.Query()
	for key := range query {
		if incidentURLKeyLooksSensitive(key) {
			query.Del(key)
		}
	}
	parsed.RawQuery = query.Encode()

	return parsed.String(), true
}

func incidentURLKeyLooksSensitive(key string) bool {
	key = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return '_'
	}, strings.TrimSpace(key))
	_, ok := incidentSensitiveURLKeys[key]

	return ok
}

func incidentValueLooksSecret(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return false
	}

	if strings.Contains(normalized, "bearer ") ||
		strings.Contains(normalized, "-----begin ") ||
		incidentJWTLikePattern.MatchString(value) {
		return true
	}
	for _, marker := range []string{
		"token=",
		"access_token",
		"client_secret",
		"api_key",
		"apikey",
		"password=",
		"secret=",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}

	return false
}
