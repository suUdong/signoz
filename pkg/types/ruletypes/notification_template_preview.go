package ruletypes

import (
	"context"
	"regexp"
	"sort"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
)

type PreviewNotificationTemplateRequest struct {
	Template    string            `json:"template"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Value       string            `json:"value,omitempty"`
	Threshold   string            `json:"threshold,omitempty"`
}

type PreviewNotificationTemplateResponse struct {
	Body        string   `json:"body"`
	MissingVars []string `json:"missingVars,omitempty"`
}

var incidentTemplateVariablePattern = regexp.MustCompile(`\$incident\.([A-Za-z0-9_]+)`)

var knownIncidentTemplateFields = map[string]struct{}{
	"project_id":         {},
	"environment":        {},
	"service_name":       {},
	"owner_team":         {},
	"severity":           {},
	"impact_summary":     {},
	"next_action":        {},
	"vendor_request":     {},
	"customer_update":    {},
	"sop_id":             {},
	"sop_url":            {},
	"sop_source":         {},
	"sop_title":          {},
	"sop_version":        {},
	"sop_binding_id":     {},
	"ai_strategy_id":     {},
	"ai_strategy_status": {},
	"ai_headline":        {},
	"ai_first_actions":   {},
	"ai_confidence":      {},
	"ai_limitations":     {},
	"ai_evidence_refs":   {},
}

func PreviewNotificationTemplate(
	ctx context.Context,
	req PreviewNotificationTemplateRequest,
) (*PreviewNotificationTemplateResponse, error) {
	value := req.Value
	if value == "" {
		value = "0"
	}

	threshold := req.Threshold
	if threshold == "" {
		threshold = "0"
	}

	data := AlertTemplateDataWithIncident(req.Labels, req.Annotations, value, threshold)
	defs := "{{$labels := .Labels}}{{$value := .Value}}{{$threshold := .Threshold}}"
	expander := NewTemplateExpander(
		ctx,
		defs+req.Template,
		"__notification_template_preview",
		data,
		nil,
	)

	body, err := expander.Expand()
	if err != nil {
		return nil, err
	}

	return &PreviewNotificationTemplateResponse{
		Body:        body,
		MissingVars: MissingIncidentTemplateVariables(req.Template),
	}, nil
}

func MissingIncidentTemplateVariables(template string) []string {
	matches := incidentTemplateVariablePattern.FindAllStringSubmatch(template, -1)
	if len(matches) == 0 {
		return nil
	}

	missing := make(map[string]struct{})
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}

		if _, ok := knownIncidentTemplateFields[match[1]]; ok {
			continue
		}

		missing[alertmanagertypes.IncidentTemplateVariablePrefix+match[1]] = struct{}{}
	}

	if len(missing) == 0 {
		return nil
	}

	result := make([]string, 0, len(missing))
	for variable := range missing {
		result = append(result, variable)
	}
	sort.Strings(result)

	return result
}
