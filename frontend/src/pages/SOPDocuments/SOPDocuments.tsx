import { useCallback, useEffect, useMemo, useState } from 'react';
import { Alert, Button, Input, Select, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import SHA256 from 'crypto-js/sha256';
import {
	createSopDocument,
	listSopDocuments,
	previewSopDocumentBinding,
	SOP_DOCUMENT_CONTRACT_VERSION,
	type SopApprovalStatus,
	type SopBindingPreviewResult,
	type SopDocument,
	type SopDocumentSummary,
} from 'api/v2/rules/sopDocuments';

import './SOPDocuments.styles.scss';

type SopDocumentFormState = {
	sopId: string;
	title: string;
	version: string;
	sourceId: string;
	bodyMarkdown: string;
	displayUrl: string;
	ownerTeam: string;
	approvalStatus: SopApprovalStatus;
	projectIds: string;
	environments: string;
	tags: string;
	serviceAccountProfile: string;
};

const DEFAULT_FORM_STATE: SopDocumentFormState = {
	sopId: '',
	title: '',
	version: '',
	sourceId: 'src-managed-markdown-default',
	bodyMarkdown: '',
	displayUrl: '',
	ownerTeam: '',
	approvalStatus: 'approved',
	projectIds: 'customer-a',
	environments: 'prod',
	tags: '',
	serviceAccountProfile: 'managed-markdown-local',
};

const APPROVAL_STATUS_OPTIONS: { label: string; value: SopApprovalStatus }[] = [
	{ label: 'Approved', value: 'approved' },
	{ label: 'Draft', value: 'draft' },
	{ label: 'Deprecated', value: 'deprecated' },
	{ label: 'Disabled', value: 'disabled' },
];

function parseTags(value: string): string[] {
	return value
		.split(',')
		.map((tag) => tag.trim())
		.filter(Boolean);
}

function checksumForMarkdown(bodyMarkdown: string): string {
	return `sha256:${SHA256(bodyMarkdown).toString()}`;
}

function buildSopDocument(form: SopDocumentFormState): SopDocument {
	return {
		contractVersion: SOP_DOCUMENT_CONTRACT_VERSION,
		sopId: form.sopId.trim(),
		title: form.title.trim(),
		version: form.version.trim(),
		checksum: checksumForMarkdown(form.bodyMarkdown),
		source: {
			type: 'managed_markdown',
			sourceId: form.sourceId.trim(),
		},
		bodyMarkdown: form.bodyMarkdown,
		displayUrl: form.displayUrl.trim() || undefined,
		ownerTeam: form.ownerTeam.trim(),
		approvalStatus: form.approvalStatus,
		tenantScope: {
			projectIds: parseTags(form.projectIds),
			environments: parseTags(form.environments),
		},
		tags: parseTags(form.tags),
		updatedAt: new Date().toISOString(),
		securityContext: {
			serviceAccountProfile: form.serviceAccountProfile.trim(),
			secretRefVisible: false,
			browserCredentialsUsed: false,
			redactionApplied: true,
		},
	};
}

function getErrorMessage(error: unknown): string {
	if (typeof error === 'object' && error !== null && 'response' in error) {
		const response = (
			error as {
				response?: { data?: { error?: string; message?: string } | string };
			}
		).response;

		if (typeof response?.data === 'string') {
			return response.data;
		}
		return response?.data?.error || response?.data?.message || 'Request failed.';
	}

	return error instanceof Error ? error.message : 'Request failed.';
}

function isSubmitDisabled(form: SopDocumentFormState): boolean {
	return !(
		form.sopId.trim() &&
		form.title.trim() &&
		form.version.trim() &&
		form.sourceId.trim() &&
		form.bodyMarkdown.trim() &&
		form.ownerTeam.trim() &&
		form.projectIds.trim() &&
		form.environments.trim() &&
		form.serviceAccountProfile.trim()
	);
}

function SOPDocuments(): JSX.Element {
	const [documents, setDocuments] = useState<SopDocumentSummary[]>([]);
	const [form, setForm] = useState<SopDocumentFormState>(DEFAULT_FORM_STATE);
	const [bindingSopId, setBindingSopId] = useState('');
	const [bindingProjectId, setBindingProjectId] = useState('customer-a');
	const [bindingEnvironment, setBindingEnvironment] = useState('prod');
	const [bindingPreview, setBindingPreview] =
		useState<SopBindingPreviewResult>();
	const [isLoading, setIsLoading] = useState(false);
	const [isSaving, setIsSaving] = useState(false);
	const [isPreviewing, setIsPreviewing] = useState(false);
	const [message, setMessage] = useState('');
	const [error, setError] = useState('');

	const loadDocuments = useCallback(async (): Promise<void> => {
		setIsLoading(true);
		setError('');
		try {
			const response = await listSopDocuments();
			setDocuments(response.data.documents);
		} catch (requestError) {
			setError(getErrorMessage(requestError));
		} finally {
			setIsLoading(false);
		}
	}, []);

	useEffect(() => {
		void loadDocuments();
	}, [loadDocuments]);

	const handleFormFieldChange = useCallback(
		<Key extends keyof SopDocumentFormState>(
			key: Key,
			value: SopDocumentFormState[Key],
		): void => {
			setForm((prev) => ({ ...prev, [key]: value }));
			setMessage('');
			setError('');
		},
		[],
	);

	const handleCreateDocument = useCallback(async (): Promise<void> => {
		setIsSaving(true);
		setMessage('');
		setError('');
		try {
			const document = buildSopDocument(form);
			const response = await createSopDocument(document);
			setMessage(
				`Saved ${response.data.sopId} ${response.data.version} for SOP binding.`,
			);
			setForm(DEFAULT_FORM_STATE);
			await loadDocuments();
		} catch (requestError) {
			setError(getErrorMessage(requestError));
		} finally {
			setIsSaving(false);
		}
	}, [form, loadDocuments]);

	const handlePreviewBinding = useCallback(async (): Promise<void> => {
		setIsPreviewing(true);
		setBindingPreview(undefined);
		setMessage('');
		setError('');
		try {
			const response = await previewSopDocumentBinding({
				labels: {
					environment: bindingEnvironment.trim(),
					project_id: bindingProjectId.trim(),
					sop_id: bindingSopId.trim(),
				},
			});
			setBindingPreview(response.data);
		} catch (requestError) {
			setError(getErrorMessage(requestError));
		} finally {
			setIsPreviewing(false);
		}
	}, [bindingEnvironment, bindingProjectId, bindingSopId]);

	const columns = useMemo<ColumnsType<SopDocumentSummary>>(
		() => [
			{
				title: 'SOP ID',
				dataIndex: 'sopId',
				key: 'sopId',
			},
			{
				title: 'Title',
				dataIndex: 'title',
				key: 'title',
			},
			{
				title: 'Version',
				dataIndex: 'version',
				key: 'version',
				width: 160,
			},
			{
				title: 'Owner',
				dataIndex: 'ownerTeam',
				key: 'ownerTeam',
				width: 160,
			},
			{
				title: 'Status',
				dataIndex: 'approvalStatus',
				key: 'approvalStatus',
				width: 140,
				render: (status: SopApprovalStatus): JSX.Element => (
					<Tag color={status === 'approved' ? 'green' : 'default'}>{status}</Tag>
				),
			},
			{
				title: 'Tenant',
				dataIndex: 'tenantScope',
				key: 'tenantScope',
				render: (tenantScope: SopDocumentSummary['tenantScope']): JSX.Element => {
					const projectIds = tenantScope?.projectIds ?? [];
					const environments = tenantScope?.environments ?? [];
					return (
						<span>{`${projectIds.join(',')} / ${environments.join(',')}`}</span>
					);
				},
			},
		],
		[],
	);

	return (
		<div className="sop-documents-page">
			<header className="sop-documents-page__header">
				<h1>DS-APM SOP documents</h1>
				<p>
					Register managed Markdown SOPs that SigNoz alert rules can bind with
					<code>sop_id</code> and feed into SOP-grounded AI response strategy.
				</p>
			</header>

			{message && <Alert message={message} showIcon type="success" />}
			{error && <Alert message={error} showIcon type="error" />}

			<section className="sop-documents-page__section">
				<div className="sop-documents-page__section-header">
					<h2>Register managed Markdown SOP</h2>
					<p>
						The browser submits only managed Markdown content and public metadata.
						Connector credentials remain server-side.
					</p>
				</div>
				<div className="sop-documents-page__form-grid">
					<label htmlFor="sop-document-sop-id-input">
						<span>SOP ID</span>
						<Input
							data-testid="sop-document-sop-id"
							id="sop-document-sop-id-input"
							onChange={(event): void =>
								handleFormFieldChange('sopId', event.target.value)
							}
							placeholder="SOP-PAY-001"
							value={form.sopId}
						/>
					</label>
					<label htmlFor="sop-document-title-input">
						<span>Title</span>
						<Input
							data-testid="sop-document-title"
							id="sop-document-title-input"
							onChange={(event): void =>
								handleFormFieldChange('title', event.target.value)
							}
							placeholder="Payment API 5xx response"
							value={form.title}
						/>
					</label>
					<label htmlFor="sop-document-version-input">
						<span>Version</span>
						<Input
							data-testid="sop-document-version"
							id="sop-document-version-input"
							onChange={(event): void =>
								handleFormFieldChange('version', event.target.value)
							}
							placeholder="2026-05-12.1"
							value={form.version}
						/>
					</label>
					<label htmlFor="sop-document-owner-team-input">
						<span>Owner team</span>
						<Input
							data-testid="sop-document-owner-team"
							id="sop-document-owner-team-input"
							onChange={(event): void =>
								handleFormFieldChange('ownerTeam', event.target.value)
							}
							placeholder="payments"
							value={form.ownerTeam}
						/>
					</label>
					<label htmlFor="sop-document-approval-status-input">
						<span>Approval status</span>
						<Select
							id="sop-document-approval-status-input"
							options={APPROVAL_STATUS_OPTIONS}
							onChange={(value): void =>
								handleFormFieldChange('approvalStatus', value)
							}
							value={form.approvalStatus}
						/>
					</label>
					<label htmlFor="sop-document-source-id-input">
						<span>Source ID</span>
						<Input
							id="sop-document-source-id-input"
							onChange={(event): void =>
								handleFormFieldChange('sourceId', event.target.value)
							}
							value={form.sourceId}
						/>
					</label>
					<label htmlFor="sop-document-project-ids-input">
						<span>Project IDs</span>
						<Input
							data-testid="sop-document-project-ids"
							id="sop-document-project-ids-input"
							onChange={(event): void =>
								handleFormFieldChange('projectIds', event.target.value)
							}
							placeholder="customer-a"
							value={form.projectIds}
						/>
					</label>
					<label htmlFor="sop-document-environments-input">
						<span>Environments</span>
						<Input
							data-testid="sop-document-environments"
							id="sop-document-environments-input"
							onChange={(event): void =>
								handleFormFieldChange('environments', event.target.value)
							}
							placeholder="prod"
							value={form.environments}
						/>
					</label>
					<label htmlFor="sop-document-display-url-input">
						<span>Display URL</span>
						<Input
							id="sop-document-display-url-input"
							onChange={(event): void =>
								handleFormFieldChange('displayUrl', event.target.value)
							}
							placeholder="https://kb.example/sop/SOP-PAY-001"
							value={form.displayUrl}
						/>
					</label>
					<label htmlFor="sop-document-tags-input">
						<span>Tags</span>
						<Input
							id="sop-document-tags-input"
							onChange={(event): void =>
								handleFormFieldChange('tags', event.target.value)
							}
							placeholder="payment-api, critical"
							value={form.tags}
						/>
					</label>
					<label htmlFor="sop-document-service-account-profile-input">
						<span>Service account profile</span>
						<Input
							id="sop-document-service-account-profile-input"
							onChange={(event): void =>
								handleFormFieldChange('serviceAccountProfile', event.target.value)
							}
							value={form.serviceAccountProfile}
						/>
					</label>
					<label
						className="sop-documents-page__markdown"
						htmlFor="sop-document-body-markdown-input"
					>
						<span>Body Markdown</span>
						<Input.TextArea
							data-testid="sop-document-body-markdown"
							id="sop-document-body-markdown-input"
							onChange={(event): void =>
								handleFormFieldChange('bodyMarkdown', event.target.value)
							}
							placeholder={
								'# Payment API 5xx response\n\n1. Check payment success dashboard\n2. Inspect PG timeout logs'
							}
							rows={7}
							value={form.bodyMarkdown}
						/>
					</label>
				</div>
				<div className="sop-documents-page__actions">
					<Button
						data-testid="register-sop-document"
						disabled={isSubmitDisabled(form)}
						loading={isSaving}
						onClick={handleCreateDocument}
						type="primary"
					>
						Register SOP document
					</Button>
					<span>Checksum is generated from the Markdown body before submit.</span>
				</div>
			</section>

			<section className="sop-documents-page__section">
				<div className="sop-documents-page__section-header">
					<h2>Binding preview</h2>
					<p>Verify that an alert label resolves to an approved SOP document.</p>
				</div>
				<div className="sop-documents-page__binding">
					<Input
						data-testid="binding-sop-id"
						onChange={(event): void => setBindingSopId(event.target.value)}
						placeholder="SOP-PAY-001"
						value={bindingSopId}
					/>
					<Input
						data-testid="binding-project-id"
						onChange={(event): void => setBindingProjectId(event.target.value)}
						placeholder="customer-a"
						value={bindingProjectId}
					/>
					<Input
						data-testid="binding-environment"
						onChange={(event): void => setBindingEnvironment(event.target.value)}
						placeholder="prod"
						value={bindingEnvironment}
					/>
					<Button
						data-testid="preview-sop-binding"
						disabled={
							!bindingSopId.trim() ||
							!bindingProjectId.trim() ||
							!bindingEnvironment.trim()
						}
						loading={isPreviewing}
						onClick={handlePreviewBinding}
					>
						Preview binding
					</Button>
				</div>
				{bindingPreview && (
					<div className="sop-documents-page__binding-result">
						<Tag color={bindingPreview.status === 'bound' ? 'green' : 'orange'}>
							{bindingPreview.status}
						</Tag>
						<span>{bindingPreview.resolution}</span>
						<span>{bindingPreview.sopId || bindingSopId}</span>
						{bindingPreview.title && <span>{bindingPreview.title}</span>}
						{bindingPreview.version && <span>{bindingPreview.version}</span>}
						{bindingPreview.warnings?.map((warning) => (
							<span className="sop-documents-page__warning" key={warning}>
								{warning}
							</span>
						))}
					</div>
				)}
			</section>

			<section className="sop-documents-page__section">
				<div className="sop-documents-page__section-header">
					<h2>Registered documents</h2>
					<p>Latest registered documents available to SOP binding preview.</p>
				</div>
				<Table
					columns={columns}
					dataSource={documents}
					loading={isLoading}
					pagination={false}
					rowKey={(document): string => `${document.sopId}:${document.version}`}
					size="small"
				/>
			</section>
		</div>
	);
}

export default SOPDocuments;
