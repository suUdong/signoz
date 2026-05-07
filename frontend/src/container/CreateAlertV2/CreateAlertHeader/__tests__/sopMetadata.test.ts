import {
	getSopBindingStatus,
	hasSopBinding,
	validateSopAnnotations,
	validateSopAnnotationValue,
	validateSopLabelValue,
} from '../sopMetadata';

describe('sopMetadata', () => {
	it('detects SOP binding from sop_id labels or sop_url annotations', () => {
		expect(hasSopBinding({}, {})).toBe(false);
		expect(hasSopBinding({ sop_id: 'SOP-PAY-001' }, {})).toBe(true);
		expect(
			hasSopBinding({}, { sop_url: 'https://kb.example/sop/SOP-PAY-001' }),
		).toBe(true);
	});

	it('returns operator-facing SOP binding status text', () => {
		expect(getSopBindingStatus({}, {})).toBe(
			'SOP missing: add sop_id or sop_url before production use',
		);
		expect(getSopBindingStatus({ sop_id: 'SOP-PAY-001' }, {})).toBe(
			'SOP binding metadata is present',
		);
	});

	it('does not warn for empty or normal SOP metadata values', () => {
		expect(validateSopLabelValue(undefined)).toStrictEqual([]);
		expect(validateSopLabelValue('SOP-PAY-001')).toStrictEqual([]);
		expect(
			validateSopAnnotationValue('sop_url', 'https://kb.example/sop/SOP-PAY-001'),
		).toStrictEqual([]);
		expect(
			validateSopAnnotationValue('sop_version', '2026-04-20.3'),
		).toStrictEqual([]);
		expect(validateSopAnnotationValue('sop_source', 'confluence')).toStrictEqual(
			[],
		);
	});

	it('warns when SOP labels are too long or secret-like', () => {
		expect(validateSopLabelValue(`SOP-${'a'.repeat(121)}`)).toStrictEqual([
			'Keep sop_id under 120 characters.',
		]);
		expect(validateSopLabelValue('token=do-not-store-this')).toStrictEqual([
			'Avoid secrets, tokens, or credentials in SOP metadata visible to alert viewers.',
		]);
	});

	it('warns for unsafe SOP annotation values', () => {
		expect(
			validateSopAnnotationValue('sop_url', 'javascript:alert(1)'),
		).toStrictEqual(['Use an http:// or https:// SOP URL.']);
		expect(
			validateSopAnnotationValue(
				'sop_url',
				'https://kb.example/sop/SOP-PAY-001?token=hidden',
			),
		).toStrictEqual([
			'Do not put credentials in SOP URLs; use server-side SOP source credentials.',
		]);
		expect(
			validateSopAnnotationValue(
				'sop_url',
				'https://user:pass@kb.example/sop/SOP-PAY-001',
			),
		).toStrictEqual([
			'Do not put credentials in SOP URLs; use server-side SOP source credentials.',
		]);
		expect(
			validateSopAnnotationValue(
				'sop_title',
				'Authorization: Bearer abcdefghijklmnop',
			),
		).toStrictEqual([
			'Avoid secrets, tokens, or credentials in SOP metadata visible to alert viewers.',
		]);
	});

	it('returns warnings keyed by SOP annotation', () => {
		expect(
			validateSopAnnotations({
				sop_title: 'Payment API incident SOP',
				sop_url: 'ftp://kb.example/sop/SOP-PAY-001',
				sop_version: 'token=do-not-store-this',
			}),
		).toStrictEqual({
			sop_url: ['Use an http:// or https:// SOP URL.'],
			sop_version: [
				'Avoid secrets, tokens, or credentials in SOP metadata visible to alert viewers.',
			],
		});
	});
});
