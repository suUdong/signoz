package ruletypes

import (
	"fmt"
	"strings"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
)

const (
	SOPTenantPolicyMissingLabelsWarning = "project_id and environment labels are required for SOP tenant policy"
	SOPTenantPolicyDeniedWarning        = "sop document is outside requested tenant scope"
)

func PilotTenantFromLabels(labels map[string]string) PilotAuditTenant {
	if labels == nil {
		return PilotAuditTenant{}
	}
	return PilotAuditTenant{
		ProjectID:   strings.TrimSpace(labels[alertmanagertypes.IncidentLabelProjectID]),
		Environment: strings.TrimSpace(labels[alertmanagertypes.IncidentLabelEnvironment]),
	}
}

func PilotTenantIsComplete(tenant PilotAuditTenant) bool {
	return strings.TrimSpace(tenant.ProjectID) != "" && strings.TrimSpace(tenant.Environment) != ""
}

func PilotTenantScopeAllows(scope PilotTenantScope, tenant PilotAuditTenant) bool {
	if !PilotTenantIsComplete(tenant) {
		return false
	}
	return pilotTenantScopeContains(scope.ProjectIDs, tenant.ProjectID) &&
		pilotTenantScopeContains(scope.Environments, tenant.Environment)
}

func normalizePilotTenantScope(scope PilotTenantScope) PilotTenantScope {
	return PilotTenantScope{
		ProjectIDs:   normalizeTenantScopeValues(scope.ProjectIDs),
		Environments: normalizeTenantScopeValues(scope.Environments),
	}
}

func validatePilotTenantScope(errs *[]error, path string, scope PilotTenantScope) {
	scope = normalizePilotTenantScope(scope)
	if len(scope.ProjectIDs) == 0 {
		*errs = append(*errs, fmt.Errorf("%s.projectIds: must include at least one project", path))
	}
	if len(scope.Environments) == 0 {
		*errs = append(*errs, fmt.Errorf("%s.environments: must include at least one environment", path))
	}
	for i, projectID := range scope.ProjectIDs {
		pilotRequireNonEmpty(errs, fmt.Sprintf("%s.projectIds[%d]", path, i), projectID)
		pilotAppendSecretLikeStringErrors(errs, fmt.Sprintf("%s.projectIds[%d]", path, i), projectID)
	}
	for i, environment := range scope.Environments {
		pilotRequireNonEmpty(errs, fmt.Sprintf("%s.environments[%d]", path, i), environment)
		pilotAppendSecretLikeStringErrors(errs, fmt.Sprintf("%s.environments[%d]", path, i), environment)
	}
}

func pilotTenantScopeContains(values []string, value string) bool {
	value = strings.TrimSpace(value)
	for _, candidate := range values {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || candidate == value {
			return true
		}
	}
	return false
}

func normalizeTenantScopeValues(values []string) []string {
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}
