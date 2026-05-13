import { Button } from 'antd';
import { Check, Copy } from 'lucide-react';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useCopyToClipboard } from 'react-use';
import type { Labels } from 'types/api/alerts/def';

import './AlertResponseContext.styles.scss';

type ResponseContextField = {
	key: string;
	label: string;
	copyLabel?: string;
	isUrl?: boolean;
	isLongText?: boolean;
	valueTone?: 'success' | 'warning';
};

type ResponseContextSection = {
	title: string;
	fields: ResponseContextField[];
};

const RESPONSE_CONTEXT_SECTIONS: ResponseContextSection[] = [
	{
		title: 'Incident briefing',
		fields: [
			{ key: 'impact_summary', label: 'Impact', isLongText: true },
			{ key: 'next_action', label: 'Next action', isLongText: true },
			{
				key: 'vendor_request',
				label: 'Vendor request',
				copyLabel: 'Copy vendor request',
				isLongText: true,
			},
			{
				key: 'customer_update',
				label: 'Customer update',
				copyLabel: 'Copy customer update',
				isLongText: true,
			},
		],
	},
	{
		title: 'Response context',
		fields: [
			{ key: 'runbook_url', label: 'Runbook', isUrl: true },
			{ key: 'owner', label: 'Owner' },
			{ key: 'escalation', label: 'Escalation' },
			{ key: 'context_url', label: 'Context', isUrl: true },
			{ key: 'ai_summary', label: 'AI summary', isLongText: true },
			{ key: 'evidence_url', label: 'Evidence', isUrl: true },
		],
	},
];

type AlertResponseContextProps = {
	alertName?: string;
	annotations?: Labels;
	labels?: Labels;
	strategyHistory?: AlertAIStrategyHistory;
};

type ResponseContextItem = ResponseContextField & { value: string };

type ResponseContextSectionWithItems = {
	title: string;
	items: ResponseContextItem[];
};

type AlertAIStrategyAction = {
	text?: string;
};

type AlertAIStrategyEvidenceRef = {
	refId?: string;
};

type AlertAIStrategySnapshot = {
	confidence?: string;
	evidenceRefs?: AlertAIStrategyEvidenceRef[];
	firstActions?: AlertAIStrategyAction[];
	headline?: string;
	limitations?: string[];
	status?: string;
	strategyId?: string;
};

export type AlertAIStrategyHistory = AlertAIStrategySnapshot & {
	strategy?: AlertAIStrategySnapshot;
};

const COPY_SUCCESS_RESET_MS = 1500;

const SECTION_COPY_LABELS: Record<string, string> = {
	'AI strategy': 'Copy AI strategy',
	'Evidence status': 'Copy evidence status',
	'Incident briefing': 'Copy briefing',
	'Response context': 'Copy context',
	'SOP status': 'Copy SOP status',
};

const COPIED_SECTION_PREFIX = 'section:';
const COPIED_ITEM_PREFIX = 'item:';
const COPIED_HANDOFF_TARGET = 'handoff:markdown';
const SENSITIVE_URL_QUERY_KEYS = new Set([
	'access_token',
	'api_key',
	'apikey',
	'auth',
	'authorization',
	'bearer',
	'password',
	'secret',
	'token',
]);

const HANDOFF_OPERATIONAL_FIELDS: ResponseContextField[] = [
	{ key: 'service.name', label: 'Service' },
	{ key: 'environment', label: 'Environment' },
	{ key: 'project_id', label: 'Project' },
	{ key: 'owner_team', label: 'Owner team' },
	{ key: 'severity', label: 'Severity' },
];

const EVIDENCE_TIMESTAMP_FIELDS: ResponseContextField[] = [
	{ key: 'evidence_generated_at', label: 'Generated' },
	{ key: 'evidence_collected_at', label: 'Collected' },
	{ key: 'evidence_updated_at', label: 'Updated' },
];

const SOP_METADATA_FIELDS: ResponseContextField[] = [
	{ key: 'sop_id', label: 'SOP ID', copyLabel: 'Copy SOP ID' },
	{ key: 'sop_source', label: 'Source' },
	{ key: 'sop_title', label: 'Title', isLongText: true },
	{ key: 'sop_version', label: 'Version' },
	{ key: 'sop_binding_id', label: 'Binding ID' },
	{
		key: 'sop_url',
		label: 'SOP preview',
		copyLabel: 'Copy SOP URL',
		isUrl: true,
	},
];

const AI_STRATEGY_FIELDS: ResponseContextField[] = [
	{ key: 'ai_strategy_id', label: 'Strategy ID' },
	{ key: 'ai_headline', label: 'Headline', isLongText: true },
	{
		key: 'ai_first_actions',
		label: 'First actions',
		copyLabel: 'Copy AI first actions',
		isLongText: true,
	},
	{ key: 'ai_confidence', label: 'Confidence' },
	{ key: 'ai_limitations', label: 'Limitations', isLongText: true },
	{ key: 'ai_evidence_refs', label: 'Evidence refs', isLongText: true },
];

function getSectionCopyText({
	items,
	title,
}: ResponseContextSectionWithItems): string {
	return [
		title,
		...items.map((item) => `${item.label}: ${getItemCopyValue(item)}`),
	].join('\n');
}

function normalizeHandoffValue(value: string): string {
	return value.replace(/\s+/g, ' ').trim();
}

function getCurrentUrl(): string | undefined {
	if (typeof window === 'undefined') {
		return undefined;
	}

	return window.location.href;
}

function getTrimmedValue(
	metadata: Labels | undefined,
	key: string,
): string | undefined {
	const value = metadata?.[key]?.trim();

	return value || undefined;
}

function getMetadataValue({
	annotations,
	key,
	labels,
}: AlertResponseContextProps & { key: string }): string | undefined {
	return getTrimmedValue(annotations, key) || getTrimmedValue(labels, key);
}

function joinNonEmptyValues(
	values: Array<string | undefined>,
): string | undefined {
	const joined = values
		.map((value) => value?.trim())
		.filter((value): value is string => Boolean(value))
		.join('\n');

	return joined || undefined;
}

function getAIStrategyHistorySnapshot(
	strategyHistory?: AlertAIStrategyHistory,
): AlertAIStrategySnapshot | undefined {
	return strategyHistory?.strategy || strategyHistory;
}

function getAIStrategyHistoryValue({
	key,
	strategyHistory,
}: Pick<AlertResponseContextProps, 'strategyHistory'> & {
	key: string;
}): string | undefined {
	const strategy = getAIStrategyHistorySnapshot(strategyHistory);

	if (!strategy) {
		return undefined;
	}

	switch (key) {
		case 'ai_strategy_id':
			return strategy.strategyId?.trim() || undefined;
		case 'ai_strategy_status':
			return strategy.status?.trim() || undefined;
		case 'ai_headline':
			return strategy.headline?.trim() || undefined;
		case 'ai_first_actions':
			return joinNonEmptyValues(
				strategy.firstActions?.map((action) => action.text) || [],
			);
		case 'ai_confidence':
			return strategy.confidence?.trim() || undefined;
		case 'ai_limitations':
			return joinNonEmptyValues(strategy.limitations || []);
		case 'ai_evidence_refs':
			return joinNonEmptyValues(
				strategy.evidenceRefs?.map((evidenceRef) => evidenceRef.refId) || [],
			)?.replace(/\n/g, ', ');
		default:
			return undefined;
	}
}

function getSafeHttpUrl(value: string): URL | undefined {
	try {
		const url = new URL(value);

		if (url.protocol === 'http:' || url.protocol === 'https:') {
			return url;
		}
	} catch {
		return undefined;
	}

	return undefined;
}

function isSensitiveUrlQueryKey(key: string): boolean {
	return SENSITIVE_URL_QUERY_KEYS.has(
		key.trim().toLowerCase().replaceAll('-', '_'),
	);
}

function getSanitizedHttpUrl(url: URL): URL {
	const sanitizedUrl = new URL(url.toString());

	sanitizedUrl.username = '';
	sanitizedUrl.password = '';
	Array.from(sanitizedUrl.searchParams.keys()).forEach((key) => {
		if (isSensitiveUrlQueryKey(key)) {
			sanitizedUrl.searchParams.delete(key);
		}
	});

	return sanitizedUrl;
}

function getUrlDisplayValue(url: URL): string {
	return `${url.host}${url.pathname === '/' ? '' : url.pathname}`;
}

function getItemCopyValue(item: ResponseContextItem): string {
	if (!item.isUrl) {
		return item.value;
	}

	const url = getSafeHttpUrl(item.value);
	if (!url) {
		return item.value;
	}

	return getSanitizedHttpUrl(url).toString();
}

function getFirstMetadataItem({
	annotations,
	fields,
	labels,
}: AlertResponseContextProps & {
	fields: ResponseContextField[];
}): ResponseContextItem | undefined {
	for (const field of fields) {
		const value = getMetadataValue({ annotations, key: field.key, labels });

		if (value) {
			return { ...field, value };
		}
	}

	return undefined;
}

function getEvidenceStatusValue({
	annotations,
	labels,
}: AlertResponseContextProps): string | undefined {
	return (
		getMetadataValue({ annotations, key: 'evidence_status', labels }) ||
		(getMetadataValue({ annotations, key: 'evidence_url', labels })
			? 'Ready'
			: undefined) ||
		(getMetadataValue({ annotations, key: 'ai_summary', labels })
			? 'Summary ready'
			: undefined)
	);
}

function getEvidenceStatusSection({
	annotations,
	labels,
}: AlertResponseContextProps): ResponseContextSectionWithItems | undefined {
	const statusValue = getEvidenceStatusValue({ annotations, labels });
	const timestampItem = getFirstMetadataItem({
		annotations,
		fields: EVIDENCE_TIMESTAMP_FIELDS,
		labels,
	});
	const confidenceValue = getMetadataValue({
		annotations,
		key: 'evidence_confidence',
		labels,
	});
	const items: ResponseContextItem[] = [
		statusValue
			? { key: 'evidence_status', label: 'Status', value: statusValue }
			: undefined,
		timestampItem,
		confidenceValue
			? {
					key: 'evidence_confidence',
					label: 'Confidence',
					value: confidenceValue,
				}
			: undefined,
	].filter((item): item is ResponseContextItem => Boolean(item));

	if (!items.length) {
		return undefined;
	}

	return {
		title: 'Evidence status',
		items,
	};
}

function hasSopBinding({
	annotations,
	labels,
}: AlertResponseContextProps): boolean {
	return Boolean(
		getMetadataValue({ annotations, key: 'sop_id', labels }) ||
		getMetadataValue({ annotations, key: 'sop_url', labels }),
	);
}

function getSopMetadataItems({
	annotations,
	labels,
}: AlertResponseContextProps): ResponseContextItem[] {
	return SOP_METADATA_FIELDS.map((field) => ({
		...field,
		value: getMetadataValue({ annotations, key: field.key, labels }),
	})).filter((field): field is ResponseContextItem => Boolean(field.value));
}

function getSopStatusSection({
	annotations,
	labels,
	shouldShowMissing,
}: AlertResponseContextProps & {
	shouldShowMissing: boolean;
}): ResponseContextSectionWithItems | undefined {
	const sopItems = getSopMetadataItems({ annotations, labels });
	const isBound = hasSopBinding({ annotations, labels });

	if (!isBound && !sopItems.length && !shouldShowMissing) {
		return undefined;
	}

	return {
		title: 'SOP status',
		items: [
			{
				key: 'sop_status',
				label: 'Status',
				value: isBound ? 'Bound' : 'Missing',
				valueTone: isBound ? 'success' : 'warning',
			},
			!isBound
				? {
						key: 'sop_action',
						label: 'Action',
						value: 'Add sop_id or sop_url to this alert rule.',
						isLongText: true,
						valueTone: 'warning',
					}
				: undefined,
			...sopItems,
		].filter((item): item is ResponseContextItem => Boolean(item)),
	};
}

function getAIStrategySection({
	annotations,
	labels,
	strategyHistory,
}: AlertResponseContextProps): ResponseContextSectionWithItems | undefined {
	const valueForAIField = (key: string): string | undefined =>
		strategyHistory
			? getAIStrategyHistoryValue({ key, strategyHistory })
			: getMetadataValue({ annotations, key, labels });
	const statusValue = valueForAIField('ai_strategy_status');
	const strategyItems = AI_STRATEGY_FIELDS.map((field) => ({
		...field,
		value: valueForAIField(field.key),
	})).filter((field): field is ResponseContextItem => Boolean(field.value));

	if (!statusValue && !strategyItems.length) {
		return undefined;
	}

	return {
		title: 'AI strategy',
		items: [
			statusValue
				? {
						key: 'ai_strategy_status',
						label: 'Status',
						value: statusValue,
						valueTone: statusValue === 'ready' ? 'success' : 'warning',
					}
				: undefined,
			...strategyItems,
		].filter((item): item is ResponseContextItem => Boolean(item)),
	};
}

function getSectionsWithItems({
	annotations,
	labels,
	strategyHistory,
}: AlertResponseContextProps): ResponseContextSectionWithItems[] {
	const sections = RESPONSE_CONTEXT_SECTIONS.map(({ fields, title }) => ({
		title,
		items: fields
			.map((field) => ({
				...field,
				value: getMetadataValue({ annotations, key: field.key, labels }),
			}))
			.filter((field): field is ResponseContextItem => Boolean(field.value)),
	})).filter(({ items }) => items.length > 0);
	const evidenceStatusSection = getEvidenceStatusSection({
		annotations,
		labels,
	});
	const aiStrategySection = getAIStrategySection({
		annotations,
		labels,
		strategyHistory,
	});
	const sopStatusSection = getSopStatusSection({
		annotations,
		labels,
		shouldShowMissing:
			sections.length > 0 ||
			Boolean(evidenceStatusSection) ||
			Boolean(aiStrategySection),
	});
	const resolvedSections = [
		...(sopStatusSection ? [sopStatusSection] : []),
		...(aiStrategySection ? [aiStrategySection] : []),
		...sections,
	];

	return evidenceStatusSection
		? [...resolvedSections, evidenceStatusSection]
		: resolvedSections;
}

function getOperationalContextItems(labels?: Labels): ResponseContextItem[] {
	return HANDOFF_OPERATIONAL_FIELDS.map((field) => ({
		...field,
		value: getTrimmedValue(labels, field.key),
	})).filter((field): field is ResponseContextItem => Boolean(field.value));
}

function formatMarkdownSection({
	items,
	title,
}: ResponseContextSectionWithItems): string[] {
	return [
		`## ${title}`,
		...items.map(
			(item) =>
				`- **${item.label}:** ${normalizeHandoffValue(getItemCopyValue(item))}`,
		),
	];
}

function getMarkdownHandoffText({
	alertName,
	labels,
	sections,
}: Pick<AlertResponseContextProps, 'alertName' | 'labels'> & {
	sections: ResponseContextSectionWithItems[];
}): string {
	const markdownSections = sections.flatMap((section) => [
		...formatMarkdownSection(section),
		'',
	]);
	const operationalItems = getOperationalContextItems(labels);

	if (operationalItems.length > 0) {
		markdownSections.push(
			...formatMarkdownSection({
				title: 'Operational context',
				items: operationalItems,
			}),
			'',
		);
	}

	const currentUrl = getCurrentUrl();
	if (currentUrl) {
		markdownSections.push(`Alert URL: ${currentUrl}`);
	}

	return [
		alertName ? `# Incident handoff: ${alertName}` : '# Incident handoff',
		'',
		...markdownSections,
	]
		.join('\n')
		.trim();
}

function AlertResponseContext({
	alertName,
	annotations,
	labels,
	strategyHistory,
}: AlertResponseContextProps): JSX.Element | null {
	const sections = getSectionsWithItems({
		annotations,
		labels,
		strategyHistory,
	});
	const [, copyToClipboard] = useCopyToClipboard();
	const [copiedTarget, setCopiedTarget] = useState<string>();
	const copiedResetTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

	useEffect(() => {
		return (): void => {
			if (copiedResetTimerRef.current) {
				clearTimeout(copiedResetTimerRef.current);
			}
		};
	}, []);

	const markCopied = useCallback((target: string): void => {
		setCopiedTarget(target);
		if (copiedResetTimerRef.current) {
			clearTimeout(copiedResetTimerRef.current);
		}

		copiedResetTimerRef.current = setTimeout(() => {
			setCopiedTarget(undefined);
			copiedResetTimerRef.current = null;
		}, COPY_SUCCESS_RESET_MS);
	}, []);

	const handleCopySection = useCallback(
		(section: ResponseContextSectionWithItems): void => {
			copyToClipboard(getSectionCopyText(section));
			markCopied(`${COPIED_SECTION_PREFIX}${section.title}`);
		},
		[copyToClipboard, markCopied],
	);

	const handleCopyItem = useCallback(
		(item: ResponseContextItem): void => {
			copyToClipboard(getItemCopyValue(item));
			const { key } = item;
			markCopied(`${COPIED_ITEM_PREFIX}${key}`);
		},
		[copyToClipboard, markCopied],
	);

	const handleCopyHandoff = useCallback((): void => {
		copyToClipboard(getMarkdownHandoffText({ alertName, labels, sections }));
		markCopied(COPIED_HANDOFF_TARGET);
	}, [alertName, copyToClipboard, labels, markCopied, sections]);

	if (!sections.length) {
		return null;
	}

	return (
		<section
			className="alert-response-context"
			aria-label="Incident response context"
		>
			<div className="alert-response-context__header">
				<div>
					<div className="alert-response-context__heading">PM handoff</div>
					<div className="alert-response-context__description">
						Copy-ready incident packet for PM, operator, vendor, and customer updates.
					</div>
				</div>
				<Button
					className="alert-response-context__copy"
					icon={
						copiedTarget === COPIED_HANDOFF_TARGET ? (
							<Check size={12} />
						) : (
							<Copy size={12} />
						)
					}
					onClick={handleCopyHandoff}
					size="small"
					type="text"
				>
					{copiedTarget === COPIED_HANDOFF_TARGET ? 'Copied' : 'Copy handoff'}
				</Button>
			</div>
			{sections.map((section) => (
				<div className="alert-response-context__section" key={section.title}>
					<div className="alert-response-context__section-header">
						<div className="alert-response-context__title">{section.title}</div>
						<Button
							className="alert-response-context__copy"
							icon={
								copiedTarget === `${COPIED_SECTION_PREFIX}${section.title}` ? (
									<Check size={12} />
								) : (
									<Copy size={12} />
								)
							}
							onClick={(): void => handleCopySection(section)}
							size="small"
							type="text"
						>
							{copiedTarget === `${COPIED_SECTION_PREFIX}${section.title}`
								? 'Copied'
								: SECTION_COPY_LABELS[section.title] || 'Copy'}
						</Button>
					</div>
					<div className="alert-response-context__items">
						{section.items.map((item) => {
							const { copyLabel, isLongText, isUrl, key, label, value } = item;
							const safeHttpUrl = isUrl ? getSafeHttpUrl(value) : undefined;
							const sanitizedHttpUrl = safeHttpUrl
								? getSanitizedHttpUrl(safeHttpUrl)
								: undefined;
							const isCopied = copiedTarget === `${COPIED_ITEM_PREFIX}${key}`;
							const valueClassName = [
								'alert-response-context__value',
								isLongText ? 'alert-response-context__value--long' : undefined,
								item.valueTone
									? `alert-response-context__value--${item.valueTone}`
									: undefined,
							]
								.filter(Boolean)
								.join(' ');

							return (
								<div className="alert-response-context__item" key={key}>
									<span className="alert-response-context__label">{label}</span>
									{safeHttpUrl && sanitizedHttpUrl ? (
										<a
											className={`${valueClassName} alert-response-context__value--link`}
											href={sanitizedHttpUrl.toString()}
											target="_blank"
											rel="noopener noreferrer"
										>
											{getUrlDisplayValue(safeHttpUrl)}
										</a>
									) : (
										<span className={valueClassName}>{value}</span>
									)}
									{copyLabel && (
										<Button
											aria-label={isCopied ? `Copied ${label}` : copyLabel}
											className="alert-response-context__item-copy"
											icon={isCopied ? <Check size={12} /> : <Copy size={12} />}
											onClick={(): void => handleCopyItem(item)}
											size="small"
											title={copyLabel}
											type="text"
										/>
									)}
								</div>
							);
						})}
					</div>
				</div>
			))}
		</section>
	);
}

export default AlertResponseContext;
