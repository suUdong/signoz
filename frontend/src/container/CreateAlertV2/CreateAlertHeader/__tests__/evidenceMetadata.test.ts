import {
	validateEvidenceMetadata,
	validateEvidenceMetadataValue,
} from '../evidenceMetadata';

describe('evidenceMetadata', () => {
	it('does not warn for empty or supported evidence metadata values', () => {
		expect(
			validateEvidenceMetadataValue('evidence_status', undefined),
		).toStrictEqual([]);
		expect(
			validateEvidenceMetadataValue('evidence_status', 'ready'),
		).toStrictEqual([]);
		expect(
			validateEvidenceMetadataValue('evidence_confidence', 'high'),
		).toStrictEqual([]);
		expect(
			validateEvidenceMetadataValue(
				'evidence_generated_at',
				'2026-04-26T14:06:00Z',
			),
		).toStrictEqual([]);
		expect(
			validateEvidenceMetadataValue(
				'evidence_url',
				'https://ds.example/incidents/INC-123/evidence',
			),
		).toStrictEqual([]);
	});

	it('warns for unsupported status, confidence, timestamp, and URL values', () => {
		expect(
			validateEvidenceMetadataValue('evidence_status', 'unknown'),
		).toStrictEqual([
			'Use one of: collecting, ready, summary_ready, failed, stale, unavailable.',
		]);
		expect(
			validateEvidenceMetadataValue('evidence_confidence', 'certain'),
		).toStrictEqual(['Use one of: high, medium, low.']);
		expect(
			validateEvidenceMetadataValue('evidence_generated_at', 'yesterday'),
		).toStrictEqual(['Use an ISO-8601 timestamp such as 2026-04-26T14:06:00Z.']);
		expect(
			validateEvidenceMetadataValue('evidence_url', 'javascript:alert(1)'),
		).toStrictEqual(['Use an http:// or https:// evidence URL.']);
	});

	it('warns when evidence metadata looks like a secret', () => {
		expect(
			validateEvidenceMetadataValue('ai_summary', 'token=do-not-store-this'),
		).toStrictEqual([
			'Avoid secrets, tokens, or credentials in evidence metadata visible to alert viewers.',
		]);
	});

	it('returns warnings keyed by evidence annotation', () => {
		expect(
			validateEvidenceMetadata({
				evidence_confidence: 'unknown',
				evidence_status: 'ready',
				evidence_url: 'ftp://evidence.example.com/report',
			}),
		).toStrictEqual({
			evidence_confidence: ['Use one of: high, medium, low.'],
			evidence_url: ['Use an http:// or https:// evidence URL.'],
		});
	});
});
