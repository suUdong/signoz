export type IncidentTemplateVariable = {
	variable: string;
	description: string;
};

export const INCIDENT_TEMPLATE_VARIABLES: IncidentTemplateVariable[] = [
	{
		variable: '$incident.project_id',
		description: 'SI/SM project or customer identifier used for routing.',
	},
	{
		variable: '$incident.environment',
		description: 'Environment such as prod, staging, or customer-specific env.',
	},
	{
		variable: '$incident.service_name',
		description: 'OpenTelemetry service connected to this APM alert.',
	},
	{
		variable: '$incident.owner_team',
		description: 'First responder, service owner, or vendor team.',
	},
	{
		variable: '$incident.severity',
		description: 'Alert severity used for urgency and escalation.',
	},
	{
		variable: '$incident.impact_summary',
		description: 'PM-friendly impact summary from alert annotations.',
	},
	{
		variable: '$incident.next_action',
		description: 'Recommended first action for the PM or responder.',
	},
	{
		variable: '$incident.vendor_request',
		description: 'Question or evidence request for partner/vendor developers.',
	},
	{
		variable: '$incident.customer_update',
		description: 'Plain-language customer update draft.',
	},
	{
		variable: '$incident.sop_id',
		description: 'Stable SOP identifier bound to this alert rule.',
	},
	{
		variable: '$incident.sop_url',
		description: 'SOP preview/deep-link URL for responders.',
	},
	{
		variable: '$incident.sop_source',
		description:
			'SOP source such as Confluence, Git, Notion, or manual metadata.',
	},
	{
		variable: '$incident.sop_title',
		description: 'Human-readable SOP title for the notification.',
	},
	{
		variable: '$incident.sop_version',
		description: 'SOP version used for operational drift checks.',
	},
	{
		variable: '$incident.sop_binding_id',
		description: 'Environment/service/severity-specific SOP binding reference.',
	},
	{
		variable: '$incident.ai_strategy_id',
		description: 'Generated AI response strategy identifier.',
	},
	{
		variable: '$incident.ai_strategy_status',
		description: 'AI strategy state such as ready, timeout, or sop_missing.',
	},
	{
		variable: '$incident.ai_headline',
		description: 'SOP-grounded strategy headline for responders.',
	},
	{
		variable: '$incident.ai_first_actions',
		description:
			'Human-approved first actions grounded in SOP steps or evidence.',
	},
	{
		variable: '$incident.ai_confidence',
		description: 'AI strategy confidence: high, medium, or low.',
	},
	{
		variable: '$incident.ai_limitations',
		description: 'Known limits such as missing evidence or provider timeout.',
	},
	{
		variable: '$incident.ai_evidence_refs',
		description: 'Evidence reference IDs cited by the AI strategy.',
	},
];

const INCIDENT_VARIABLE_PATTERN = /\$incident\.[A-Za-z0-9_.-]+/g;

const KNOWN_INCIDENT_VARIABLES = new Set(
	INCIDENT_TEMPLATE_VARIABLES.map(({ variable }) => variable),
);

export function getUnknownIncidentTemplateVariables(
	template: string,
): string[] {
	const matches = template.match(INCIDENT_VARIABLE_PATTERN) || [];
	const unknown = matches.filter(
		(variable) => !KNOWN_INCIDENT_VARIABLES.has(variable),
	);

	return Array.from(new Set(unknown)).sort();
}
