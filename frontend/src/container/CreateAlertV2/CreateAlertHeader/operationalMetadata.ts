import type { Labels } from 'types/api/alerts/def';

export type RequiredOperationalLabel = {
	key: string;
	label: string;
	description: string;
};

export const REQUIRED_OPERATIONAL_LABELS: RequiredOperationalLabel[] = [
	{
		key: 'project_id',
		label: 'Project',
		description: 'Routes the incident to the right SI/SM project.',
	},
	{
		key: 'environment',
		label: 'Environment',
		description: 'Separates prod, staging, and customer-specific environments.',
	},
	{
		key: 'service.name',
		label: 'Service',
		description: 'Connects the alert to the OpenTelemetry service.',
	},
	{
		key: 'owner_team',
		label: 'Owner team',
		description: 'Identifies the first responder or vendor team.',
	},
	{
		key: 'severity',
		label: 'Severity',
		description: 'Drives urgency, notification routing, and customer messaging.',
	},
];

export function getMissingOperationalLabels(
	labels: Labels,
): RequiredOperationalLabel[] {
	return REQUIRED_OPERATIONAL_LABELS.filter(({ key }) => !labels[key]?.trim());
}

export function hasOperationalLabel(labels: Labels, key: string): boolean {
	return Boolean(labels[key]?.trim());
}
