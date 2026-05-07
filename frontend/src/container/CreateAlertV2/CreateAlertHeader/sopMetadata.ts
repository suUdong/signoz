import type { Labels } from 'types/api/alerts/def';

export const SOP_ID_LABEL = 'sop_id';

export type SopAnnotationFieldKey =
	| 'sop_binding_id'
	| 'sop_source'
	| 'sop_title'
	| 'sop_url'
	| 'sop_version';

export type SopAnnotationField = {
	key: SopAnnotationFieldKey;
	label: string;
	placeholder: string;
};

export const SOP_ANNOTATION_FIELDS: SopAnnotationField[] = [
	{
		key: 'sop_url',
		label: 'SOP URL',
		placeholder: 'https://kb.example/sop/SOP-PAY-001',
	},
	{
		key: 'sop_source',
		label: 'SOP source',
		placeholder: 'confluence, git, notion, manual',
	},
	{
		key: 'sop_title',
		label: 'SOP title',
		placeholder: 'Payment API 5xx response',
	},
	{
		key: 'sop_version',
		label: 'SOP version',
		placeholder: '2026-04-20.3',
	},
	{
		key: 'sop_binding_id',
		label: 'Binding ID',
		placeholder: 'payment-api-prod-critical',
	},
];

const SECRET_LIKE_PATTERNS = [
	/bearer\s+[a-z0-9._~+/=-]+/i,
	/(api[_-]?key|password|secret|token)\s*[:=]\s*\S+/i,
	/-----BEGIN [A-Z ]*PRIVATE KEY-----/,
	/\beyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{10,}\b/,
];

const SENSITIVE_URL_QUERY_KEYS = new Set([
	'access_token',
	'api_key',
	'apikey',
	'auth',
	'authorization',
	'bearer',
	'password',
	'secret',
	'token',
]);

function isHttpUrl(value: string): boolean {
	try {
		const url = new URL(value);

		return url.protocol === 'http:' || url.protocol === 'https:';
	} catch {
		return false;
	}
}

function hasUrlCredentials(value: string): boolean {
	try {
		const url = new URL(value);

		if (url.username || url.password) {
			return true;
		}

		return Array.from(url.searchParams.keys()).some((key) =>
			SENSITIVE_URL_QUERY_KEYS.has(key.trim().toLowerCase().replaceAll('-', '_')),
		);
	} catch {
		return false;
	}
}

export function hasSopBinding(labels: Labels, annotations: Labels): boolean {
	return Boolean(labels[SOP_ID_LABEL]?.trim() || annotations.sop_url?.trim());
}

export function getSopBindingStatus(
	labels: Labels,
	annotations: Labels,
): string {
	return hasSopBinding(labels, annotations)
		? 'SOP binding metadata is present'
		: 'SOP missing: add sop_id or sop_url before production use';
}

export function validateSopLabelValue(value: string | undefined): string[] {
	const trimmedValue = value?.trim();

	if (!trimmedValue) {
		return [];
	}

	const warnings: string[] = [];

	if (trimmedValue.length > 120) {
		warnings.push('Keep sop_id under 120 characters.');
	}

	if (SECRET_LIKE_PATTERNS.some((pattern) => pattern.test(trimmedValue))) {
		warnings.push(
			'Avoid secrets, tokens, or credentials in SOP metadata visible to alert viewers.',
		);
	}

	return warnings;
}

export function validateSopAnnotationValue(
	key: SopAnnotationFieldKey,
	value: string | undefined,
): string[] {
	const trimmedValue = value?.trim();

	if (!trimmedValue) {
		return [];
	}

	const warnings: string[] = [];

	if (key === 'sop_url') {
		if (!isHttpUrl(trimmedValue)) {
			warnings.push('Use an http:// or https:// SOP URL.');
		} else if (hasUrlCredentials(trimmedValue)) {
			warnings.push(
				'Do not put credentials in SOP URLs; use server-side SOP source credentials.',
			);
		}

		return warnings;
	}

	if (SECRET_LIKE_PATTERNS.some((pattern) => pattern.test(trimmedValue))) {
		warnings.push(
			'Avoid secrets, tokens, or credentials in SOP metadata visible to alert viewers.',
		);
	}

	return warnings;
}

export function validateSopAnnotations(
	annotations: Labels,
): Partial<Record<SopAnnotationFieldKey, string[]>> {
	return SOP_ANNOTATION_FIELDS.reduce<
		Partial<Record<SopAnnotationFieldKey, string[]>>
	>((warningsByKey, { key }) => {
		const warnings = validateSopAnnotationValue(key, annotations[key]);

		if (warnings.length) {
			warningsByKey[key] = warnings;
		}

		return warningsByKey;
	}, {});
}
