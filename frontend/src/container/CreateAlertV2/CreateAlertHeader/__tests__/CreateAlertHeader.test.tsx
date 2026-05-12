import { fireEvent, render, screen } from '@testing-library/react';
import { previewSop } from 'api/v2/rules/previewSop';
import { QueryParams } from 'constants/query';
import ROUTES from 'constants/routes';
import { defaultPostableAlertRuleV2 } from 'container/CreateAlertV2/constants';
import { getCreateAlertLocalStateFromAlertDef } from 'container/CreateAlertV2/utils';
import * as useSafeNavigateHook from 'hooks/useSafeNavigate';
import { AlertTypes } from 'types/api/alerts/alertTypes';

import * as rulesHook from '../../../../api/generated/services/rules';
import { CreateAlertProvider } from '../../context';
import CreateAlertHeader from '../CreateAlertHeader';

jest.mock('api/v2/rules/previewSop', () => ({
	previewSop: jest.fn(),
}));

const mockSafeNavigate = jest.fn();
jest.spyOn(useSafeNavigateHook, 'useSafeNavigate').mockReturnValue({
	safeNavigate: mockSafeNavigate,
});

jest.spyOn(rulesHook, 'useCreateRule').mockReturnValue({
	mutate: jest.fn(),
	isLoading: false,
} as unknown as ReturnType<typeof rulesHook.useCreateRule>);
jest.spyOn(rulesHook, 'useTestRule').mockReturnValue({
	mutate: jest.fn(),
	isLoading: false,
} as unknown as ReturnType<typeof rulesHook.useTestRule>);
jest.spyOn(rulesHook, 'useUpdateRuleByID').mockReturnValue({
	mutate: jest.fn(),
	isLoading: false,
} as unknown as ReturnType<typeof rulesHook.useUpdateRuleByID>);

jest.mock('uplot', () => {
	const paths = {
		spline: jest.fn(),
		bars: jest.fn(),
	};
	const uplotMock = jest.fn(() => ({
		paths,
	}));
	return {
		paths,
		default: uplotMock,
	};
});

jest.mock('react-router-dom', () => ({
	...jest.requireActual('react-router-dom'),
	useLocation: (): { search: string } => ({
		search: '',
	}),
}));

const ENTER_ALERT_RULE_NAME_PLACEHOLDER = 'Enter alert rule name';
const mockPreviewSop = previewSop as jest.MockedFunction<typeof previewSop>;

const renderCreateAlertHeader = (): ReturnType<typeof render> =>
	render(
		<CreateAlertProvider initialAlertType={AlertTypes.METRICS_BASED_ALERT}>
			<CreateAlertHeader />
		</CreateAlertProvider>,
	);

describe('CreateAlertHeader', () => {
	beforeEach(() => {
		jest.clearAllMocks();
		mockPreviewSop.mockResolvedValue({
			status: 'success',
			data: {
				access: {
					auditEventRequired: true,
					browserCredentialsAllowed: false,
					credentialScope: 'source_connector_secret',
					message:
						'Live SOP content must be fetched server-side with source connector credentials; browser credentials are never accepted.',
					mode: 'server_side_connector',
					recommendedServiceAccountProfile: 'ds-sop-reader',
					requiresServerSideFetch: true,
				},
				binding: {
					bindingId: 'payment-api-prod-critical',
					sopId: 'SOP-PAY-001',
					title: 'Payment API 5xx response',
					version: '2026-04-20.3',
				},
				contractVersion: 'ds-apm.sop-preview.v1',
				preview: {
					available: true,
					displayUrl: 'kb.example/sop/SOP-PAY-001',
					title: 'Payment API 5xx response',
					url: 'https://kb.example/sop/SOP-PAY-001?view=summary',
				},
				search: {
					query: 'SOP-PAY-001 payment-api-prod-critical Payment API 5xx response',
					terms: [
						'SOP-PAY-001',
						'payment-api-prod-critical',
						'Payment API 5xx response',
					],
				},
				source: {
					kind: 'configured_source',
					name: 'confluence',
				},
				status: 'bound',
				warnings: [],
			},
		});
	});

	it('renders the header with title', () => {
		renderCreateAlertHeader();
		expect(screen.getByText('New Alert Rule')).toBeInTheDocument();
	});

	it('renders name input with placeholder', () => {
		renderCreateAlertHeader();
		const nameInput = screen.getByPlaceholderText(
			ENTER_ALERT_RULE_NAME_PLACEHOLDER,
		);
		expect(nameInput).toBeInTheDocument();
	});

	it('renders LabelsInput component', () => {
		renderCreateAlertHeader();
		expect(screen.getByText('+ Add labels')).toBeInTheDocument();
	});

	it('shows missing SI/SM routing metadata labels for new alerts', () => {
		renderCreateAlertHeader();

		expect(screen.getByText('SI/SM routing metadata')).toBeInTheDocument();
		expect(screen.getByText('Missing 5 recommended labels')).toBeInTheDocument();
		expect(screen.getByText('service.name')).toBeInTheDocument();
		expect(screen.getByText('owner_team')).toBeInTheDocument();
	});

	it('shows complete SI/SM routing metadata when required labels are present', () => {
		render(
			<CreateAlertProvider
				isEditMode
				initialAlertType={AlertTypes.METRICS_BASED_ALERT}
				initialAlertState={getCreateAlertLocalStateFromAlertDef({
					...defaultPostableAlertRuleV2,
					labels: {
						environment: 'prod',
						owner_team: 'sm-payments',
						project_id: 'customer-a',
						'service.name': 'payment-api',
						severity: 'critical',
					},
				})}
			>
				<CreateAlertHeader />
			</CreateAlertProvider>,
		);

		expect(
			screen.getByText('All recommended SI/SM labels are present'),
		).toBeInTheDocument();
		expect(screen.getAllByText('Set')).toHaveLength(5);
	});

	it('renders SOP binding metadata and updates label plus annotations', () => {
		renderCreateAlertHeader();

		expect(screen.getByText('SOP binding')).toBeInTheDocument();
		expect(
			screen.getByText('SOP missing: add sop_id or sop_url before production use'),
		).toBeInTheDocument();

		const sopIdInput = screen.getByTestId('sop-metadata-sop_id');
		const sopUrlInput = screen.getByTestId('sop-metadata-sop_url');

		fireEvent.change(sopIdInput, {
			target: { value: 'SOP-PAY-001' },
		});
		fireEvent.change(sopUrlInput, {
			target: { value: 'https://kb.example/sop/SOP-PAY-001' },
		});

		expect(sopIdInput).toHaveValue('SOP-PAY-001');
		expect(sopUrlInput).toHaveValue('https://kb.example/sop/SOP-PAY-001');
		expect(
			screen.getByText('SOP binding metadata is present'),
		).toBeInTheDocument();
	});

	it('previews SOP source, search, and document metadata', async () => {
		renderCreateAlertHeader();

		const sopIdInput = screen.getByTestId('sop-metadata-sop_id');
		const sopUrlInput = screen.getByTestId('sop-metadata-sop_url');
		const sopSourceInput = screen.getByTestId('sop-metadata-sop_source');
		const sopTitleInput = screen.getByTestId('sop-metadata-sop_title');

		fireEvent.change(sopIdInput, {
			target: { value: 'SOP-PAY-001' },
		});
		fireEvent.change(sopUrlInput, {
			target: { value: 'https://kb.example/sop/SOP-PAY-001?view=summary' },
		});
		fireEvent.change(sopSourceInput, {
			target: { value: 'confluence' },
		});
		fireEvent.change(sopTitleInput, {
			target: { value: 'Payment API 5xx response' },
		});

		fireEvent.click(screen.getByRole('button', { name: 'Preview SOP source' }));

		expect(mockPreviewSop).toHaveBeenCalledWith({
			labels: {
				sop_id: 'SOP-PAY-001',
			},
			annotations: {
				sop_source: 'confluence',
				sop_title: 'Payment API 5xx response',
				sop_url: 'https://kb.example/sop/SOP-PAY-001?view=summary',
			},
		});
		await expect(
			screen.findByTestId('sop-source-preview'),
		).resolves.toBeInTheDocument();
		expect(screen.getByText('Review summary')).toBeInTheDocument();
		expect(
			screen.getByText('SOP metadata is ready for PM handoff review.'),
		).toBeInTheDocument();
		expect(screen.getByText('Browser credentials blocked')).toBeInTheDocument();
		expect(
			screen.getByText('Server-side connector required'),
		).toBeInTheDocument();
		expect(
			screen.getByText('Audit required before live fetch'),
		).toBeInTheDocument();
		expect(screen.getByText('ds-apm.sop-preview.v1')).toBeInTheDocument();
		expect(screen.getByText('bound')).toBeInTheDocument();
		expect(screen.getByText('confluence')).toBeInTheDocument();
		expect(
			screen.getByText(
				'SOP-PAY-001 payment-api-prod-critical Payment API 5xx response',
			),
		).toBeInTheDocument();
		const previewLink = screen.getByRole('link', {
			name: 'kb.example/sop/SOP-PAY-001',
		});
		expect(previewLink).toHaveAttribute(
			'href',
			'https://kb.example/sop/SOP-PAY-001?view=summary',
		);
		expect(screen.queryByText(/view=summary/)).not.toBeInTheDocument();
		expect(
			screen.getByText('server_side_connector · source_connector_secret'),
		).toBeInTheDocument();
		expect(screen.getByText('ds-sop-reader')).toBeInTheDocument();
		expect(screen.getByText('Never accepted')).toBeInTheDocument();
		expect(
			screen.getByText(
				'Live SOP content must be fetched server-side with source connector credentials; browser credentials are never accepted.',
			),
		).toBeInTheDocument();
	});

	it('warns when SOP metadata uses unsafe values', () => {
		renderCreateAlertHeader();

		const sopIdInput = screen.getByTestId('sop-metadata-sop_id');
		const sopUrlInput = screen.getByTestId('sop-metadata-sop_url');

		fireEvent.change(sopIdInput, {
			target: { value: `SOP-${'a'.repeat(121)}` },
		});
		fireEvent.change(sopUrlInput, {
			target: { value: 'javascript:alert(1)' },
		});

		expect(
			screen.getByText('Keep sop_id under 120 characters.'),
		).toBeInTheDocument();
		expect(
			screen.getByText('Use an http:// or https:// SOP URL.'),
		).toBeInTheDocument();
		expect(sopIdInput).toHaveAttribute('aria-invalid', 'true');
		expect(sopUrlInput).toHaveAttribute('aria-invalid', 'true');
	});

	it('warns when SOP URL includes browser-visible credentials', () => {
		renderCreateAlertHeader();

		const sopUrlInput = screen.getByTestId('sop-metadata-sop_url');

		fireEvent.change(sopUrlInput, {
			target: { value: 'https://user:pass@kb.example/sop/SOP-PAY-001' },
		});

		expect(
			screen.getByText(
				'Do not put credentials in SOP URLs; use server-side SOP source credentials.',
			),
		).toBeInTheDocument();
		expect(sopUrlInput).toHaveAttribute('aria-invalid', 'true');
	});

	it('renders and updates PM incident briefing metadata fields', () => {
		renderCreateAlertHeader();

		expect(screen.getByText('PM incident briefing')).toBeInTheDocument();
		const impactInput = screen.getByTestId('pm-briefing-impact_summary');
		const nextActionInput = screen.getByTestId('pm-briefing-next_action');

		fireEvent.change(impactInput, {
			target: { value: 'Checkout failures may affect customers.' },
		});
		fireEvent.change(nextActionInput, {
			target: { value: 'Ask vendor to check payment-api traces.' },
		});

		expect(impactInput).toHaveValue('Checkout failures may affect customers.');
		expect(nextActionInput).toHaveValue(
			'Ask vendor to check payment-api traces.',
		);
	});

	it('warns when PM incident briefing metadata contains secret-like values', () => {
		renderCreateAlertHeader();

		const customerUpdateInput = screen.getByTestId('pm-briefing-customer_update');

		fireEvent.change(customerUpdateInput, {
			target: { value: 'Customer update token=do-not-store-this' },
		});

		expect(
			screen.getByText(
				'Avoid secrets, tokens, or credentials in alert metadata visible to alert viewers.',
			),
		).toBeInTheDocument();
		expect(customerUpdateInput).toHaveAttribute('aria-invalid', 'true');
	});

	it('renders and updates AI evidence status metadata fields', () => {
		renderCreateAlertHeader();

		expect(screen.getByText('AI/evidence status')).toBeInTheDocument();
		const strategyStatusInput = screen.getByTestId(
			'evidence-metadata-ai_strategy_status',
		);
		const headlineInput = screen.getByTestId('evidence-metadata-ai_headline');
		const statusInput = screen.getByTestId('evidence-metadata-evidence_status');
		const generatedAtInput = screen.getByTestId(
			'evidence-metadata-evidence_generated_at',
		);

		fireEvent.change(strategyStatusInput, {
			target: { value: 'ready' },
		});
		fireEvent.change(headlineInput, {
			target: { value: 'SOP 기준 결제 지연 확인이 필요합니다.' },
		});
		fireEvent.change(statusInput, {
			target: { value: 'ready' },
		});
		fireEvent.change(generatedAtInput, {
			target: { value: '2026-04-26T14:06:00Z' },
		});

		expect(strategyStatusInput).toHaveValue('ready');
		expect(headlineInput).toHaveValue('SOP 기준 결제 지연 확인이 필요합니다.');
		expect(statusInput).toHaveValue('ready');
		expect(generatedAtInput).toHaveValue('2026-04-26T14:06:00Z');
	});

	it('warns when AI evidence metadata uses unsafe values', () => {
		renderCreateAlertHeader();

		const statusInput = screen.getByTestId('evidence-metadata-evidence_status');
		const strategyStatusInput = screen.getByTestId(
			'evidence-metadata-ai_strategy_status',
		);
		const evidenceUrlInput = screen.getByTestId('evidence-metadata-evidence_url');

		fireEvent.change(statusInput, {
			target: { value: 'unknown' },
		});
		fireEvent.change(strategyStatusInput, {
			target: { value: 'fabricated' },
		});
		fireEvent.change(evidenceUrlInput, {
			target: { value: 'javascript:alert(1)' },
		});

		expect(
			screen.getByText(
				'Use one of: collecting, ready, summary_ready, failed, stale, unavailable.',
			),
		).toBeInTheDocument();
		expect(
			screen.getByText('Use an http:// or https:// evidence URL.'),
		).toBeInTheDocument();
		expect(
			screen.getByText(
				'Use one of: ready, unavailable, timeout, blocked_by_policy, sop_missing, evidence_unavailable, low_confidence.',
			),
		).toBeInTheDocument();
		expect(statusInput).toHaveAttribute('aria-invalid', 'true');
		expect(strategyStatusInput).toHaveAttribute('aria-invalid', 'true');
		expect(evidenceUrlInput).toHaveAttribute('aria-invalid', 'true');
	});

	it('updates name when typing in name input', () => {
		renderCreateAlertHeader();
		const nameInput = screen.getByPlaceholderText(
			ENTER_ALERT_RULE_NAME_PLACEHOLDER,
		);

		fireEvent.change(nameInput, { target: { value: 'Test Alert' } });

		expect(nameInput).toHaveValue('Test Alert');
	});

	it('renders the header with title when isEditMode is true', () => {
		render(
			<CreateAlertProvider
				isEditMode
				initialAlertType={AlertTypes.METRICS_BASED_ALERT}
				initialAlertState={getCreateAlertLocalStateFromAlertDef(
					defaultPostableAlertRuleV2,
				)}
			>
				<CreateAlertHeader />
			</CreateAlertProvider>,
		);
		expect(screen.queryByText('New Alert Rule')).not.toBeInTheDocument();
		expect(
			screen.getByPlaceholderText(ENTER_ALERT_RULE_NAME_PLACEHOLDER),
		).toHaveValue('TEST_ALERT');
	});

	it('should navigate to classic experience when button is clicked', () => {
		renderCreateAlertHeader();
		const switchToClassicExperienceButton = screen.getByText(
			'Switch to Classic Experience',
		);
		expect(switchToClassicExperienceButton).toBeInTheDocument();
		fireEvent.click(switchToClassicExperienceButton);

		const params = new URLSearchParams();
		params.set(QueryParams.showClassicCreateAlertsPage, 'true');
		expect(mockSafeNavigate).toHaveBeenCalledWith(
			`${ROUTES.ALERTS_NEW}?${params.toString()}`,
			{ replace: true },
		);
	});

	it('should not render "switch to classic experience" button when isEditMode is true', () => {
		render(
			<CreateAlertProvider
				isEditMode
				initialAlertType={AlertTypes.METRICS_BASED_ALERT}
				initialAlertState={getCreateAlertLocalStateFromAlertDef(
					defaultPostableAlertRuleV2,
				)}
			>
				<CreateAlertHeader />
			</CreateAlertProvider>,
		);
		expect(
			screen.queryByText('Switch to Classic Experience'),
		).not.toBeInTheDocument();
	});
});
