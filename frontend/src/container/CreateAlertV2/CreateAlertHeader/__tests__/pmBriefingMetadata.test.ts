import {
	PM_BRIEFING_MAX_LENGTH,
	validatePmBriefingMetadata,
	validatePmBriefingValue,
} from '../pmBriefingMetadata';

describe('pmBriefingMetadata', () => {
	it('does not warn for empty or normal PM briefing values', () => {
		expect(validatePmBriefingValue(undefined)).toStrictEqual([]);
		expect(
			validatePmBriefingValue('Ask the vendor to inspect payment-api traces.'),
		).toStrictEqual([]);
	});

	it('warns when PM briefing metadata is too long', () => {
		const warnings = validatePmBriefingValue(
			'a'.repeat(PM_BRIEFING_MAX_LENGTH + 1),
		);

		expect(warnings).toStrictEqual([
			`Keep this under ${PM_BRIEFING_MAX_LENGTH} characters; link to longer notes instead.`,
		]);
	});

	it('warns when PM briefing metadata looks like a secret', () => {
		const warnings = validatePmBriefingValue(
			'Check with Authorization: Bearer abcdefghijklmnop',
		);

		expect(warnings).toStrictEqual([
			'Avoid secrets, tokens, or credentials in alert metadata visible to alert viewers.',
		]);
	});

	it('returns warnings keyed by PM briefing annotation', () => {
		expect(
			validatePmBriefingMetadata({
				customer_update: 'token=do-not-store-this',
				impact_summary: 'Checkout failures may affect customers.',
			}),
		).toStrictEqual({
			customer_update: [
				'Avoid secrets, tokens, or credentials in alert metadata visible to alert viewers.',
			],
		});
	});
});
