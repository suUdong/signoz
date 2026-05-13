import { fireEvent, render, screen, waitFor } from 'tests/test-utils';
import {
	createSopDocument,
	listSopDocuments,
	previewSopDocumentBinding,
} from 'api/v2/rules/sopDocuments';

import SOPDocuments from '../SOPDocuments';

jest.mock('api/v2/rules/sopDocuments', () => ({
	createSopDocument: jest.fn(),
	listSopDocuments: jest.fn(),
	previewSopDocumentBinding: jest.fn(),
	SOP_DOCUMENT_CONTRACT_VERSION: 'ds.sop_document.v1',
}));

const mockCreateSopDocument = createSopDocument as jest.MockedFunction<
	typeof createSopDocument
>;
const mockListSopDocuments = listSopDocuments as jest.MockedFunction<
	typeof listSopDocuments
>;
const mockPreviewSopDocumentBinding =
	previewSopDocumentBinding as jest.MockedFunction<
		typeof previewSopDocumentBinding
	>;

describe('SOPDocuments', () => {
	beforeEach(() => {
		jest.clearAllMocks();
		mockListSopDocuments.mockResolvedValue({
			status: 'success',
			data: {
				contractVersion: 'ds.sop_document_list.v1',
				documents: [
					{
						approvalStatus: 'approved',
						checksum: 'sha256:existing',
						contractVersion: 'ds.sop_document.v1',
						displayUrl: 'https://kb.example/sop/SOP-PAY-001',
						ownerTeam: 'payments',
						sopId: 'SOP-PAY-001',
						source: {
							sourceId: 'src-managed-markdown-default',
							type: 'managed_markdown',
						},
						tenantScope: {
							environments: ['prod'],
							projectIds: ['customer-a'],
						},
						tags: ['payment-api'],
						title: 'Payment API 5xx response',
						updatedAt: '2026-05-12T00:00:00Z',
						version: '2026-05-12.1',
					},
				],
			},
		});
		mockCreateSopDocument.mockImplementation(async (document) => ({
			status: 'success',
			data: document,
		}));
		mockPreviewSopDocumentBinding.mockResolvedValue({
			status: 'success',
			data: {
				contractVersion: 'ds.sop_binding.v1',
				resolution: 'explicit_label',
				sopId: 'SOP-PAY-001',
				status: 'bound',
				title: 'Payment API 5xx response',
				version: '2026-05-12.1',
			},
		});
	});

	it('lists, registers, and previews managed Markdown SOPs', async () => {
		render(<SOPDocuments />);

		await expect(
			screen.findByText('Payment API 5xx response'),
		).resolves.toBeInTheDocument();

		fireEvent.change(screen.getByTestId('sop-document-sop-id'), {
			target: { value: 'SOP-CHECKOUT-001' },
		});
		fireEvent.change(screen.getByTestId('sop-document-title'), {
			target: { value: 'Checkout latency response' },
		});
		fireEvent.change(screen.getByTestId('sop-document-version'), {
			target: { value: '2026-05-12.2' },
		});
		fireEvent.change(screen.getByTestId('sop-document-owner-team'), {
			target: { value: 'checkout' },
		});
		fireEvent.change(screen.getByTestId('sop-document-body-markdown'), {
			target: {
				value: '# Checkout latency response\n\n1. Check checkout latency dashboard',
			},
		});
		fireEvent.click(screen.getByTestId('register-sop-document'));

		await waitFor(() => expect(mockCreateSopDocument).toHaveBeenCalled());
		const submittedDocument = mockCreateSopDocument.mock.calls[0][0];
		expect(submittedDocument).toMatchObject({
			contractVersion: 'ds.sop_document.v1',
			sopId: 'SOP-CHECKOUT-001',
			title: 'Checkout latency response',
			version: '2026-05-12.2',
			ownerTeam: 'checkout',
			approvalStatus: 'approved',
			tenantScope: {
				environments: ['prod'],
				projectIds: ['customer-a'],
			},
			source: {
				sourceId: 'src-managed-markdown-default',
				type: 'managed_markdown',
			},
			securityContext: {
				browserCredentialsUsed: false,
				redactionApplied: true,
				secretRefVisible: false,
				serviceAccountProfile: 'managed-markdown-local',
			},
		});
		expect(submittedDocument.checksum).toMatch(/^sha256:[a-f0-9]{64}$/);

		fireEvent.change(screen.getByTestId('binding-sop-id'), {
			target: { value: 'SOP-PAY-001' },
		});
		fireEvent.click(screen.getByTestId('preview-sop-binding'));

		await waitFor(() =>
			expect(mockPreviewSopDocumentBinding).toHaveBeenCalledWith({
				labels: {
					environment: 'prod',
					project_id: 'customer-a',
					sop_id: 'SOP-PAY-001',
				},
			}),
		);
		await expect(
			screen.findByText('explicit_label'),
		).resolves.toBeInTheDocument();
		expect(screen.getByText('bound')).toBeInTheDocument();
	});
});
