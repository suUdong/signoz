import {
	getMissingOperationalLabels,
	hasOperationalLabel,
	REQUIRED_OPERATIONAL_LABELS,
} from '../operationalMetadata';

describe('operationalMetadata', () => {
	it('reports every required SI/SM label when metadata is empty', () => {
		expect(getMissingOperationalLabels({})).toStrictEqual(
			REQUIRED_OPERATIONAL_LABELS,
		);
	});

	it('treats blank label values as missing', () => {
		expect(
			getMissingOperationalLabels({
				environment: ' ',
				owner_team: 'sm-payments',
				project_id: 'customer-a',
				'service.name': 'payment-api',
				severity: 'critical',
			}).map(({ key }) => key),
		).toStrictEqual(['environment']);
	});

	it('detects complete SI/SM routing labels', () => {
		const labels = {
			environment: 'prod',
			owner_team: 'sm-payments',
			project_id: 'customer-a',
			'service.name': 'payment-api',
			severity: 'critical',
		};

		expect(getMissingOperationalLabels(labels)).toStrictEqual([]);
		expect(hasOperationalLabel(labels, 'service.name')).toBe(true);
	});
});
