import type { Labels } from 'types/api/alerts/def';

export type PmBriefingFieldKey =
	| 'impact_summary'
	| 'next_action'
	| 'vendor_request'
	| 'customer_update';

export type PmBriefingField = {
	key: PmBriefingFieldKey;
	label: string;
	placeholder: string;
};

export const PM_BRIEFING_MAX_LENGTH = 280;

export const PM_BRIEFING_FIELDS: PmBriefingField[] = [
	{
		key: 'impact_summary',
		label: 'Impact',
		placeholder: 'Who or what is affected? Example: Checkout failures may rise.',
	},
	{
		key: 'next_action',
		label: 'Next action',
		placeholder:
			'What should the PM do first? Example: Ask vendor to inspect traces.',
	},
	{
		key: 'vendor_request',
		label: 'Vendor request',
		placeholder:
			'What should the vendor developer answer? Example: cause, mitigation, ETA.',
	},
	{
		key: 'customer_update',
		label: 'Customer update',
		placeholder: 'Plain-language customer update draft for the first response.',
	},
];

const SECRET_LIKE_PATTERNS = [
	/bearer\s+[a-z0-9._~+/=-]+/i,
	/(api[_-]?key|password|secret|token)\s*[:=]\s*\S+/i,
	/-----BEGIN [A-Z ]*PRIVATE KEY-----/,
	/\beyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{10,}\b/,
];

export function validatePmBriefingValue(value: string | undefined): string[] {
	const trimmedValue = value?.trim();

	if (!trimmedValue) {
		return [];
	}

	const warnings: string[] = [];

	if (trimmedValue.length > PM_BRIEFING_MAX_LENGTH) {
		warnings.push(
			`Keep this under ${PM_BRIEFING_MAX_LENGTH} characters; link to longer notes instead.`,
		);
	}

	if (SECRET_LIKE_PATTERNS.some((pattern) => pattern.test(trimmedValue))) {
		warnings.push(
			'Avoid secrets, tokens, or credentials in alert metadata visible to alert viewers.',
		);
	}

	return warnings;
}

export function validatePmBriefingMetadata(
	annotations: Labels,
): Partial<Record<PmBriefingFieldKey, string[]>> {
	return PM_BRIEFING_FIELDS.reduce<
		Partial<Record<PmBriefingFieldKey, string[]>>
	>((warningsByKey, { key }) => {
		const warnings = validatePmBriefingValue(annotations[key]);

		if (warnings.length) {
			warningsByKey[key] = warnings;
		}

		return warningsByKey;
	}, {});
}
