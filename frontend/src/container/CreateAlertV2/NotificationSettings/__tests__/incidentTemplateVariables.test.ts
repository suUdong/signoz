import {
	getUnknownIncidentTemplateVariables,
	INCIDENT_TEMPLATE_VARIABLES,
} from '../incidentTemplateVariables';

describe('incidentTemplateVariables', () => {
	it('lists PM briefing and SI/SM routing variables', () => {
		expect(
			INCIDENT_TEMPLATE_VARIABLES.map(({ variable }) => variable),
		).toStrictEqual(
			expect.arrayContaining([
				'$incident.project_id',
				'$incident.environment',
				'$incident.service_name',
				'$incident.owner_team',
				'$incident.severity',
				'$incident.impact_summary',
				'$incident.next_action',
				'$incident.vendor_request',
				'$incident.customer_update',
				'$incident.sop_id',
				'$incident.sop_url',
				'$incident.sop_source',
				'$incident.sop_title',
				'$incident.sop_version',
				'$incident.sop_binding_id',
				'$incident.ai_strategy_id',
				'$incident.ai_strategy_status',
				'$incident.ai_headline',
				'$incident.ai_first_actions',
				'$incident.ai_confidence',
				'$incident.ai_limitations',
				'$incident.ai_evidence_refs',
			]),
		);
	});

	it('returns unknown incident variables once in sorted order', () => {
		expect(
			getUnknownIncidentTemplateVariables(
				'$incident.unknown $incident.next_action $incident.bad $incident.unknown',
			),
		).toStrictEqual(['$incident.bad', '$incident.unknown']);
	});

	it('ignores supported incident variables', () => {
		expect(
			getUnknownIncidentTemplateVariables(
				'Impact: $incident.impact_summary Next: $incident.next_action SOP: $incident.sop_id <$incident.sop_url> Source: $incident.sop_source AI: $incident.ai_strategy_status $incident.ai_headline $incident.ai_first_actions $incident.ai_limitations',
			),
		).toStrictEqual([]);
	});
});
