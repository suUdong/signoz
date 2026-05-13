import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import AlertResponseContext from './AlertResponseContext';

const mockCopyToClipboard = jest.fn();

jest.mock('react-use', () => ({
	useCopyToClipboard: (): [unknown, jest.Mock] => [null, mockCopyToClipboard],
}));

describe('AlertResponseContext', () => {
	beforeEach(() => {
		jest.clearAllMocks();
		window.history.pushState({}, '', '/');
	});

	it('does not render when supported response metadata is absent', () => {
		const { container } = render(
			<AlertResponseContext
				annotations={{ description: 'regular alert description' }}
				labels={{ severity: 'critical' }}
			/>,
		);

		expect(screen.queryByText('Response context')).not.toBeInTheDocument();
		expect(screen.queryByText('Incident briefing')).not.toBeInTheDocument();
		expect(container).toBeEmptyDOMElement();
	});

	it('renders PM-friendly incident briefing metadata', () => {
		render(
			<AlertResponseContext
				annotations={{
					customer_update:
						'We are investigating increased payment API errors and will share the next update in 15 minutes.',
					impact_summary: 'Payment approval failures may affect checkout.',
				}}
				labels={{
					next_action:
						'Ask the vendor to check the latest deployment and PG timeout logs.',
					vendor_request: 'Provide suspected cause, mitigation, and ETA.',
				}}
			/>,
		);

		expect(screen.getByText('Incident briefing')).toBeInTheDocument();
		expect(screen.getByText('PM handoff')).toBeInTheDocument();
		expect(
			screen.getByText(
				'Copy-ready incident packet for PM, operator, vendor, and customer updates.',
			),
		).toBeInTheDocument();
		expect(
			screen.getByRole('button', { name: /Copy handoff/ }),
		).toBeInTheDocument();
		expect(screen.getByText('Impact')).toBeInTheDocument();
		expect(
			screen.getByText('Payment approval failures may affect checkout.'),
		).toBeInTheDocument();
		expect(screen.getByText('Next action')).toBeInTheDocument();
		expect(
			screen.getByText(
				'Ask the vendor to check the latest deployment and PG timeout logs.',
			),
		).toBeInTheDocument();
		expect(screen.getByText('Vendor request')).toBeInTheDocument();
		expect(
			screen.getByText('Provide suspected cause, mitigation, and ETA.'),
		).toBeInTheDocument();
		expect(screen.getByText('Customer update')).toBeInTheDocument();
		expect(
			screen.getByText(
				'We are investigating increased payment API errors and will share the next update in 15 minutes.',
			),
		).toBeInTheDocument();
		expect(
			screen.getByRole('button', { name: 'Copy vendor request' }),
		).toBeInTheDocument();
		expect(
			screen.getByRole('button', { name: 'Copy customer update' }),
		).toBeInTheDocument();
		expect(
			screen.getByRole('button', { name: /Copy briefing/ }),
		).toBeInTheDocument();
	});

	it('copies the PM-friendly incident briefing for handoff', async () => {
		const user = userEvent.setup();
		render(
			<AlertResponseContext
				annotations={{
					impact_summary: 'Payment approval failures may affect checkout.',
					next_action: 'Ask vendor to check payment API traces.',
				}}
				labels={{
					customer_update: 'We are investigating payment API errors.',
				}}
			/>,
		);

		await user.click(screen.getByRole('button', { name: /Copy briefing/ }));

		expect(mockCopyToClipboard).toHaveBeenCalledWith(
			[
				'Incident briefing',
				'Impact: Payment approval failures may affect checkout.',
				'Next action: Ask vendor to check payment API traces.',
				'Customer update: We are investigating payment API errors.',
			].join('\n'),
		);
	});

	it('copies a Markdown handoff with briefing, operational context, and alert URL', async () => {
		const user = userEvent.setup();
		window.history.pushState({}, '', '/alerts/rules/checkout-latency');
		render(
			<AlertResponseContext
				alertName="Checkout latency high"
				annotations={{
					impact_summary: 'Checkout latency can affect payments.',
					vendor_request: 'Share suspected cause and mitigation ETA.',
				}}
				labels={{
					environment: 'prod',
					owner_team: 'payments',
					service_name: 'legacy-name-should-not-be-used',
					'service.name': 'checkout-api',
					severity: 'critical',
				}}
			/>,
		);

		await user.click(screen.getByRole('button', { name: /Copy handoff/ }));

		expect(mockCopyToClipboard).toHaveBeenCalledWith(
			[
				'# Incident handoff: Checkout latency high',
				'',
				'## SOP status',
				'- **Status:** Missing',
				'- **Action:** Add sop_id or sop_url to this alert rule.',
				'',
				'## Incident briefing',
				'- **Impact:** Checkout latency can affect payments.',
				'- **Vendor request:** Share suspected cause and mitigation ETA.',
				'',
				'## Operational context',
				'- **Service:** checkout-api',
				'- **Environment:** prod',
				'- **Owner team:** payments',
				'- **Severity:** critical',
				'',
				'Alert URL: http://localhost/alerts/rules/checkout-latency',
			].join('\n'),
		);
		expect(screen.getByRole('button', { name: /Copied/ })).toBeInTheDocument();
	});

	it('copies individual vendor and customer handoff snippets', async () => {
		const user = userEvent.setup();
		render(
			<AlertResponseContext
				annotations={{
					customer_update: 'We are investigating payment API errors.',
					vendor_request: 'Provide suspected cause, mitigation, and ETA.',
				}}
			/>,
		);

		await user.click(screen.getByRole('button', { name: 'Copy vendor request' }));
		expect(mockCopyToClipboard).toHaveBeenLastCalledWith(
			'Provide suspected cause, mitigation, and ETA.',
		);
		expect(
			screen.getByRole('button', { name: 'Copied Vendor request' }),
		).toBeInTheDocument();

		await user.click(
			screen.getByRole('button', { name: 'Copy customer update' }),
		);
		expect(mockCopyToClipboard).toHaveBeenLastCalledWith(
			'We are investigating payment API errors.',
		);
		expect(
			screen.getByRole('button', { name: 'Copied Customer update' }),
		).toBeInTheDocument();
	});

	it('renders supported response metadata from annotations and labels', () => {
		render(
			<AlertResponseContext
				annotations={{
					ai_summary: 'Payment latency has crossed the SLO burn threshold.',
					sop_url: 'https://runbooks.example.com/payment-latency?token=hidden',
				}}
				labels={{
					escalation: '#payments-oncall',
					owner: 'payments-team',
				}}
			/>,
		);

		expect(screen.getByText('Response context')).toBeInTheDocument();
		expect(screen.getByText('Owner')).toBeInTheDocument();
		expect(screen.getByText('payments-team')).toBeInTheDocument();
		expect(screen.getByText('Escalation')).toBeInTheDocument();
		expect(screen.getByText('#payments-oncall')).toBeInTheDocument();
		expect(screen.getByText('AI summary')).toBeInTheDocument();
		expect(
			screen.getByText('Payment latency has crossed the SLO burn threshold.'),
		).toBeInTheDocument();

		const sopLink = screen.getByRole('link', {
			name: 'runbooks.example.com/payment-latency',
		});
		expect(sopLink).toHaveAttribute(
			'href',
			'https://runbooks.example.com/payment-latency',
		);
		expect(sopLink).toHaveAttribute('rel', 'noopener noreferrer');
		expect(screen.queryByText(/token=hidden/)).not.toBeInTheDocument();
		expect(
			screen.getByRole('button', { name: /Copy context/ }),
		).toBeInTheDocument();
	});

	it('renders bound SOP status with safe preview and copy actions', () => {
		render(
			<AlertResponseContext
				annotations={{
					sop_binding_id: 'payment-api-prod-critical',
					sop_source: 'confluence',
					sop_title: 'Payment API 5xx response',
					sop_url: 'https://runbooks.example.com/payment-latency?token=hidden',
					sop_version: '2026-04-20.3',
				}}
				labels={{
					sop_id: 'SOP-PAY-001',
				}}
			/>,
		);

		expect(screen.getByText('SOP status')).toBeInTheDocument();
		expect(screen.getByText('Status')).toBeInTheDocument();
		expect(screen.getByText('Bound')).toBeInTheDocument();
		expect(screen.getByText('SOP ID')).toBeInTheDocument();
		expect(screen.getByText('SOP-PAY-001')).toBeInTheDocument();
		expect(screen.getByText('Source')).toBeInTheDocument();
		expect(screen.getByText('confluence')).toBeInTheDocument();
		expect(screen.getByText('Title')).toBeInTheDocument();
		expect(screen.getByText('Payment API 5xx response')).toBeInTheDocument();
		expect(screen.getByText('Version')).toBeInTheDocument();
		expect(screen.getByText('2026-04-20.3')).toBeInTheDocument();
		expect(screen.getByText('Binding ID')).toBeInTheDocument();
		expect(screen.getByText('payment-api-prod-critical')).toBeInTheDocument();

		const sopLink = screen.getByRole('link', {
			name: 'runbooks.example.com/payment-latency',
		});
		expect(sopLink).toHaveAttribute(
			'href',
			'https://runbooks.example.com/payment-latency',
		);
		expect(screen.queryByText(/token=hidden/)).not.toBeInTheDocument();
		expect(
			screen.getByRole('button', { name: /Copy SOP status/ }),
		).toBeInTheDocument();
		expect(
			screen.getByRole('button', { name: 'Copy SOP URL' }),
		).toBeInTheDocument();
	});

	it('renders missing SOP status when other handoff context is present', () => {
		render(
			<AlertResponseContext
				annotations={{
					impact_summary: 'Checkout failures may affect customers.',
				}}
			/>,
		);

		expect(screen.getByText('SOP status')).toBeInTheDocument();
		expect(screen.getByText('Missing')).toBeInTheDocument();
		expect(
			screen.getByText('Add sop_id or sop_url to this alert rule.'),
		).toBeInTheDocument();
		expect(
			screen.getByRole('button', { name: /Copy SOP status/ }),
		).toBeInTheDocument();
	});

	it('renders SOP-grounded AI strategy metadata and copy actions', async () => {
		const user = userEvent.setup();
		render(
			<AlertResponseContext
				annotations={{
					ai_confidence: 'medium',
					ai_evidence_refs: 'metric:error_rate:1, trace:error:1',
					ai_first_actions: 'SOP-PAY-001 1단계에 따라 PG timeout 로그를 확인',
					ai_headline: 'SOP 기준 결제 지연 확인이 필요합니다.',
					ai_limitations: '최근 배포 정보는 연결되지 않음',
					ai_strategy_id: 'AIS-20260512-0001',
					ai_strategy_status: 'ready',
				}}
				labels={{
					sop_id: 'SOP-PAY-001',
				}}
			/>,
		);

		expect(screen.getByText('AI strategy')).toBeInTheDocument();
		expect(screen.getByText('ready')).toBeInTheDocument();
		expect(
			screen.getByText('SOP 기준 결제 지연 확인이 필요합니다.'),
		).toBeInTheDocument();
		expect(
			screen.getByText('SOP-PAY-001 1단계에 따라 PG timeout 로그를 확인'),
		).toBeInTheDocument();
		expect(screen.getByText('medium')).toBeInTheDocument();
		expect(
			screen.getByText('metric:error_rate:1, trace:error:1'),
		).toBeInTheDocument();
		expect(
			screen.getByRole('button', { name: /Copy AI strategy/ }),
		).toBeInTheDocument();

		await user.click(
			screen.getByRole('button', { name: 'Copy AI first actions' }),
		);
		expect(mockCopyToClipboard).toHaveBeenLastCalledWith(
			'SOP-PAY-001 1단계에 따라 PG timeout 로그를 확인',
		);
	});

	it('renders AI fallback status without fabricated first actions', () => {
		render(
			<AlertResponseContext
				annotations={{
					ai_limitations: 'No evidence refs were available.',
					ai_strategy_status: 'evidence_unavailable',
				}}
				labels={{
					sop_id: 'SOP-PAY-001',
				}}
			/>,
		);

		expect(screen.getByText('AI strategy')).toBeInTheDocument();
		expect(screen.getByText('evidence_unavailable')).toBeInTheDocument();
		expect(
			screen.getByText('No evidence refs were available.'),
		).toBeInTheDocument();
		expect(screen.queryByText('First actions')).not.toBeInTheDocument();
	});

	it('renders persisted AI strategy history before stale annotations', () => {
		render(
			<AlertResponseContext
				annotations={{
					ai_first_actions: 'stale action from annotation',
					ai_strategy_status: 'ready',
				}}
				labels={{
					sop_id: 'SOP-PAY-001',
				}}
				strategyHistory={{
					strategy: {
						confidence: 'low',
						headline: 'AI 사용량 한도에 도달하여 SOP 기본 알림만 전송합니다.',
						limitations: ['AI strategy quota is exhausted for this period.'],
						status: 'quota_exhausted',
						strategyId: 'AIS-20260513-0002',
					},
				}}
			/>,
		);

		expect(screen.getByText('AI strategy')).toBeInTheDocument();
		expect(screen.getByText('quota_exhausted')).toBeInTheDocument();
		expect(
			screen.getByText('AI 사용량 한도에 도달하여 SOP 기본 알림만 전송합니다.'),
		).toBeInTheDocument();
		expect(
			screen.getByText('AI strategy quota is exhausted for this period.'),
		).toBeInTheDocument();
		expect(screen.queryByText('ready')).not.toBeInTheDocument();
		expect(
			screen.queryByText('stale action from annotation'),
		).not.toBeInTheDocument();
	});

	it('copies SOP status section and individual SOP URL', async () => {
		const user = userEvent.setup();
		render(
			<AlertResponseContext
				annotations={{
					sop_title: 'Payment API 5xx response',
					sop_url: 'https://runbooks.example.com/payment-latency',
				}}
				labels={{
					sop_id: 'SOP-PAY-001',
				}}
			/>,
		);

		await user.click(screen.getByRole('button', { name: /Copy SOP status/ }));

		expect(mockCopyToClipboard).toHaveBeenLastCalledWith(
			[
				'SOP status',
				'Status: Bound',
				'SOP ID: SOP-PAY-001',
				'Title: Payment API 5xx response',
				'SOP preview: https://runbooks.example.com/payment-latency',
			].join('\n'),
		);

		await user.click(screen.getByRole('button', { name: 'Copy SOP URL' }));
		expect(mockCopyToClipboard).toHaveBeenLastCalledWith(
			'https://runbooks.example.com/payment-latency',
		);
	});

	it('renders evidence status and freshness metadata', () => {
		render(
			<AlertResponseContext
				annotations={{
					evidence_confidence: 'high',
					evidence_generated_at: '2026-04-26T14:06:00Z',
					evidence_status: 'ready',
					evidence_url: 'https://evidence.example.com/incidents/INC-123',
				}}
			/>,
		);

		expect(screen.getByText('Evidence status')).toBeInTheDocument();
		expect(screen.getAllByText('Status')).toHaveLength(2);
		expect(screen.getByText('ready')).toBeInTheDocument();
		expect(screen.getByText('Generated')).toBeInTheDocument();
		expect(screen.getByText('2026-04-26T14:06:00Z')).toBeInTheDocument();
		expect(screen.getByText('Confidence')).toBeInTheDocument();
		expect(screen.getByText('high')).toBeInTheDocument();
		expect(
			screen.getByRole('button', { name: /Copy evidence status/ }),
		).toBeInTheDocument();
	});

	it('infers evidence status when only evidence link is present', () => {
		render(
			<AlertResponseContext
				annotations={{
					evidence_url: 'https://evidence.example.com/incidents/INC-123',
				}}
			/>,
		);

		expect(screen.getByText('Evidence status')).toBeInTheDocument();
		expect(screen.getByText('Ready')).toBeInTheDocument();
	});

	it('prefers annotation metadata over labels when both provide a supported key', () => {
		render(
			<AlertResponseContext
				annotations={{ owner: 'incident-commander' }}
				labels={{ owner: 'service-team' }}
			/>,
		);

		expect(screen.getByText('incident-commander')).toBeInTheDocument();
		expect(screen.queryByText('service-team')).not.toBeInTheDocument();
	});

	it('renders unsafe URL metadata as text instead of a link', () => {
		render(
			<AlertResponseContext annotations={{ sop_url: 'javascript:alert(1)' }} />,
		);

		expect(screen.getByText('javascript:alert(1)')).toBeInTheDocument();
		expect(
			screen.queryByRole('link', { name: 'javascript:alert(1)' }),
		).not.toBeInTheDocument();
	});
});
