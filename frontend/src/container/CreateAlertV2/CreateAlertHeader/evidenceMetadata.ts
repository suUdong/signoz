import type { Labels } from 'types/api/alerts/def';

export type EvidenceMetadataFieldKey =
	| 'ai_confidence'
	| 'ai_evidence_refs'
	| 'ai_first_actions'
	| 'ai_headline'
	| 'ai_limitations'
	| 'ai_strategy_id'
	| 'ai_strategy_status'
	| 'ai_summary'
	| 'evidence_confidence'
	| 'evidence_generated_at'
	| 'evidence_status'
	| 'evidence_url';

export type EvidenceMetadataField = {
	key: EvidenceMetadataFieldKey;
	label: string;
	placeholder: string;
};

export const EVIDENCE_METADATA_FIELDS: EvidenceMetadataField[] = [
	{
		key: 'ai_strategy_status',
		label: 'AI strategy status',
		placeholder: 'ready, quota_exhausted, unavailable, timeout, sop_missing',
	},
	{
		key: 'ai_headline',
		label: 'AI headline',
		placeholder: 'SOP-grounded response strategy headline',
	},
	{
		key: 'ai_first_actions',
		label: 'AI first actions',
		placeholder: 'Human-approved first actions grounded in SOP/evidence',
	},
	{
		key: 'ai_confidence',
		label: 'AI confidence',
		placeholder: 'high, medium, low',
	},
	{
		key: 'ai_limitations',
		label: 'AI limitations',
		placeholder: 'Missing evidence, provider timeout, low confidence',
	},
	{
		key: 'ai_evidence_refs',
		label: 'AI evidence refs',
		placeholder: 'metric:error_rate:1, trace:error:1',
	},
	{
		key: 'ai_strategy_id',
		label: 'AI strategy ID',
		placeholder: 'AIS-20260512-0001',
	},
	{
		key: 'evidence_status',
		label: 'Evidence status',
		placeholder: 'ready, collecting, failed, stale, unavailable',
	},
	{
		key: 'evidence_generated_at',
		label: 'Generated at',
		placeholder: '2026-04-26T14:06:00Z',
	},
	{
		key: 'evidence_confidence',
		label: 'Confidence',
		placeholder: 'high, medium, low',
	},
	{
		key: 'evidence_url',
		label: 'Evidence URL',
		placeholder: 'https://ds.example/incidents/INC-123/evidence',
	},
	{
		key: 'ai_summary',
		label: 'AI summary',
		placeholder: 'Short evidence-backed summary. Link the full report above.',
	},
];

const EVIDENCE_STATUS_VALUES = new Set([
	'collecting',
	'failed',
	'ready',
	'stale',
	'summary_ready',
	'unavailable',
]);

const AI_STRATEGY_STATUS_VALUES = new Set([
	'blocked_by_policy',
	'evidence_unavailable',
	'low_confidence',
	'quota_exhausted',
	'ready',
	'sop_missing',
	'timeout',
	'unavailable',
]);

const EVIDENCE_CONFIDENCE_VALUES = new Set(['high', 'medium', 'low']);

const SECRET_LIKE_PATTERNS = [
	/bearer\s+[a-z0-9._~+/=-]+/i,
	/(api[_-]?key|password|secret|token)\s*[:=]\s*\S+/i,
	/-----BEGIN [A-Z ]*PRIVATE KEY-----/,
	/\beyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{10,}\b/,
];

function isHttpUrl(value: string): boolean {
	try {
		const url = new URL(value);

		return url.protocol === 'http:' || url.protocol === 'https:';
	} catch {
		return false;
	}
}

function isIsoLikeTimestamp(value: string): boolean {
	return !Number.isNaN(Date.parse(value));
}

export function validateEvidenceMetadataValue(
	key: EvidenceMetadataFieldKey,
	value: string | undefined,
): string[] {
	const trimmedValue = value?.trim();

	if (!trimmedValue) {
		return [];
	}

	const warnings: string[] = [];

	if (SECRET_LIKE_PATTERNS.some((pattern) => pattern.test(trimmedValue))) {
		warnings.push(
			'Avoid secrets, tokens, or credentials in evidence metadata visible to alert viewers.',
		);
	}

	if (key === 'evidence_url' && !isHttpUrl(trimmedValue)) {
		warnings.push('Use an http:// or https:// evidence URL.');
	}

	if (
		key === 'evidence_status' &&
		!EVIDENCE_STATUS_VALUES.has(trimmedValue.toLowerCase())
	) {
		warnings.push(
			'Use one of: collecting, ready, summary_ready, failed, stale, unavailable.',
		);
	}

	if (
		key === 'ai_strategy_status' &&
		!AI_STRATEGY_STATUS_VALUES.has(trimmedValue.toLowerCase())
	) {
		warnings.push(
			'Use one of: ready, unavailable, timeout, blocked_by_policy, quota_exhausted, sop_missing, evidence_unavailable, low_confidence.',
		);
	}

	if (
		(key === 'evidence_confidence' || key === 'ai_confidence') &&
		!EVIDENCE_CONFIDENCE_VALUES.has(trimmedValue.toLowerCase())
	) {
		warnings.push('Use one of: high, medium, low.');
	}

	if (key === 'evidence_generated_at' && !isIsoLikeTimestamp(trimmedValue)) {
		warnings.push('Use an ISO-8601 timestamp such as 2026-04-26T14:06:00Z.');
	}

	return warnings;
}

export function validateEvidenceMetadata(
	annotations: Labels,
): Partial<Record<EvidenceMetadataFieldKey, string[]>> {
	return EVIDENCE_METADATA_FIELDS.reduce<
		Partial<Record<EvidenceMetadataFieldKey, string[]>>
	>((warningsByKey, { key }) => {
		const warnings = validateEvidenceMetadataValue(key, annotations[key]);

		if (warnings.length) {
			warningsByKey[key] = warnings;
		}

		return warningsByKey;
	}, {});
}
