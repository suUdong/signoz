package alertmanagertypes

import "github.com/prometheus/alertmanager/template"

const (
	IncidentTemplateVariablePrefix = "$incident."

	IncidentLabelProjectID   = "project_id"
	IncidentLabelEnvironment = "environment"
	IncidentLabelServiceName = "service.name"
	IncidentLabelOwnerTeam   = "owner_team"
	IncidentLabelSeverity    = "severity"
	IncidentLabelSopID       = "sop_id"

	IncidentAnnotationImpactSummary  = "impact_summary"
	IncidentAnnotationNextAction     = "next_action"
	IncidentAnnotationVendorRequest  = "vendor_request"
	IncidentAnnotationCustomerUpdate = "customer_update"
	IncidentAnnotationSopURL         = "sop_url"
	IncidentAnnotationSopSource      = "sop_source"
	IncidentAnnotationSopTitle       = "sop_title"
	IncidentAnnotationSopVersion     = "sop_version"
	IncidentAnnotationSopBindingID   = "sop_binding_id"
)

// IncidentInfo is the PM-friendly DS-APM/SI-SM incident context extracted
// from alert labels and annotations. It is intentionally derived from the
// existing Alertmanager payload so notifications and webhooks can use the
// metadata without introducing a separate persistence model.
type IncidentInfo struct {
	ProjectID      string `json:"projectId,omitempty" mapstructure:"project_id"`
	Environment    string `json:"environment,omitempty" mapstructure:"environment"`
	ServiceName    string `json:"serviceName,omitempty" mapstructure:"service_name"`
	OwnerTeam      string `json:"ownerTeam,omitempty" mapstructure:"owner_team"`
	Severity       string `json:"severity,omitempty" mapstructure:"severity"`
	ImpactSummary  string `json:"impactSummary,omitempty" mapstructure:"impact_summary"`
	NextAction     string `json:"nextAction,omitempty" mapstructure:"next_action"`
	VendorRequest  string `json:"vendorRequest,omitempty" mapstructure:"vendor_request"`
	CustomerUpdate string `json:"customerUpdate,omitempty" mapstructure:"customer_update"`
	SopID          string `json:"sopId,omitempty" mapstructure:"sop_id"`
	SopURL         string `json:"sopUrl,omitempty" mapstructure:"sop_url"`
	SopSource      string `json:"sopSource,omitempty" mapstructure:"sop_source"`
	SopTitle       string `json:"sopTitle,omitempty" mapstructure:"sop_title"`
	SopVersion     string `json:"sopVersion,omitempty" mapstructure:"sop_version"`
	SopBindingID   string `json:"sopBindingId,omitempty" mapstructure:"sop_binding_id"`
}

// BuildIncidentInfo maps the recommended DS-APM operational labels and PM
// briefing annotations into a structured notification payload. Callers should
// pass public annotations only.
func BuildIncidentInfo(labels, annotations template.KV) IncidentInfo {
	return IncidentInfo{
		ProjectID:      labels[IncidentLabelProjectID],
		Environment:    labels[IncidentLabelEnvironment],
		ServiceName:    labels[IncidentLabelServiceName],
		OwnerTeam:      labels[IncidentLabelOwnerTeam],
		Severity:       labels[IncidentLabelSeverity],
		ImpactSummary:  annotations[IncidentAnnotationImpactSummary],
		NextAction:     annotations[IncidentAnnotationNextAction],
		VendorRequest:  annotations[IncidentAnnotationVendorRequest],
		CustomerUpdate: annotations[IncidentAnnotationCustomerUpdate],
		SopID:          labels[IncidentLabelSopID],
		SopURL:         annotations[IncidentAnnotationSopURL],
		SopSource:      annotations[IncidentAnnotationSopSource],
		SopTitle:       annotations[IncidentAnnotationSopTitle],
		SopVersion:     annotations[IncidentAnnotationSopVersion],
		SopBindingID:   annotations[IncidentAnnotationSopBindingID],
	}
}

// IsZero reports whether the incident context has no usable SI/SM or PM
// briefing metadata. It lets integrations omit the structured block when
// legacy alerts carry none of the recommended keys.
func (i IncidentInfo) IsZero() bool {
	return i.ProjectID == "" &&
		i.Environment == "" &&
		i.ServiceName == "" &&
		i.OwnerTeam == "" &&
		i.Severity == "" &&
		i.ImpactSummary == "" &&
		i.NextAction == "" &&
		i.VendorRequest == "" &&
		i.CustomerUpdate == "" &&
		i.SopID == "" &&
		i.SopURL == "" &&
		i.SopSource == "" &&
		i.SopTitle == "" &&
		i.SopVersion == "" &&
		i.SopBindingID == ""
}
