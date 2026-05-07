import { useCallback, useMemo, useState } from 'react';
import { Button, Input } from '@signozhq/ui';
import logEvent from 'api/common/logEvent';
import { previewSop, type PreviewSopResult } from 'api/v2/rules/previewSop';
import classNames from 'classnames';
import { QueryParams } from 'constants/query';
import ROUTES from 'constants/routes';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import useUrlQuery from 'hooks/useUrlQuery';
import { RotateCcw } from 'lucide-react';
import type { Labels } from 'types/api/alerts/def';

import { useCreateAlertState } from '../context';
import {
	EVIDENCE_METADATA_FIELDS,
	validateEvidenceMetadata,
} from './evidenceMetadata';
import LabelsInput from './LabelsInput';
import {
	getMissingOperationalLabels,
	hasOperationalLabel,
	REQUIRED_OPERATIONAL_LABELS,
} from './operationalMetadata';
import {
	PM_BRIEFING_FIELDS,
	validatePmBriefingMetadata,
} from './pmBriefingMetadata';
import {
	getSopBindingStatus,
	hasSopBinding,
	SOP_ANNOTATION_FIELDS,
	SOP_ID_LABEL,
	validateSopAnnotations,
	validateSopLabelValue,
} from './sopMetadata';
import './styles.scss';

function CreateAlertHeader(): JSX.Element {
	const { alertState, setAlertState, isEditMode } = useCreateAlertState();
	const [sopPreview, setSopPreview] = useState<PreviewSopResult>();
	const [sopPreviewError, setSopPreviewError] = useState('');
	const [isSopPreviewLoading, setIsSopPreviewLoading] = useState(false);

	const { currentQuery } = useQueryBuilder();
	const { safeNavigate } = useSafeNavigate();
	const urlQuery = useUrlQuery();

	const groupByLabels = useMemo(() => {
		const labels = new Array<string>();
		currentQuery.builder.queryData.forEach((query) => {
			query.groupBy.forEach((groupBy) => {
				labels.push(groupBy.key);
			});
		});
		return labels;
	}, [currentQuery]);

	// If the label key is a group by label, then it is not allowed to be used as a label key
	const validateLabelsKey = useCallback(
		(key: string): string | null => {
			if (groupByLabels.includes(key)) {
				return `Cannot use ${key} as a key`;
			}
			return null;
		},
		[groupByLabels],
	);

	const handleSwitchToClassicExperience = useCallback(() => {
		void logEvent('Alert: Switch to classic experience button clicked', {});

		urlQuery.set(QueryParams.showClassicCreateAlertsPage, 'true');
		const url = `${ROUTES.ALERTS_NEW}?${urlQuery.toString()}`;
		safeNavigate(url, { replace: true });
	}, [safeNavigate, urlQuery]);

	const handleAnnotationChange = useCallback(
		(key: string, value: string): void => {
			const nextAnnotations = {
				...alertState.annotations,
			};

			if (value.trim()) {
				nextAnnotations[key] = value;
			} else {
				delete nextAnnotations[key];
			}

			setSopPreview(undefined);
			setSopPreviewError('');
			setAlertState({
				type: 'SET_ALERT_ANNOTATIONS',
				payload: nextAnnotations,
			});
		},
		[alertState.annotations, setAlertState],
	);

	const handleLabelChange = useCallback(
		(key: string, value: string): void => {
			const nextLabels = {
				...alertState.labels,
			};

			if (value.trim()) {
				nextLabels[key] = value;
			} else {
				delete nextLabels[key];
			}

			setSopPreview(undefined);
			setSopPreviewError('');
			setAlertState({
				type: 'SET_ALERT_LABELS',
				payload: nextLabels,
			});
		},
		[alertState.labels, setAlertState],
	);

	const handlePreviewSop = useCallback(async (): Promise<void> => {
		setSopPreviewError('');
		setIsSopPreviewLoading(true);

		try {
			const response = await previewSop({
				labels: alertState.labels,
				annotations: alertState.annotations,
			});

			setSopPreview(response.data);
		} catch {
			setSopPreview(undefined);
			setSopPreviewError('Unable to preview SOP source metadata.');
		} finally {
			setIsSopPreviewLoading(false);
		}
	}, [alertState.annotations, alertState.labels]);

	const pmBriefingWarnings = useMemo(
		() => validatePmBriefingMetadata(alertState.annotations),
		[alertState.annotations],
	);

	const evidenceMetadataWarnings = useMemo(
		() => validateEvidenceMetadata(alertState.annotations),
		[alertState.annotations],
	);

	const sopLabelWarnings = useMemo(
		() => validateSopLabelValue(alertState.labels[SOP_ID_LABEL]),
		[alertState.labels],
	);

	const sopAnnotationWarnings = useMemo(
		() => validateSopAnnotations(alertState.annotations),
		[alertState.annotations],
	);

	const missingOperationalLabels = useMemo(
		() => getMissingOperationalLabels(alertState.labels),
		[alertState.labels],
	);

	return (
		<div
			className={classNames('alert-header', { 'edit-alert-header': isEditMode })}
		>
			{!isEditMode && (
				<div className="alert-header__tab-bar">
					<div className="alert-header__tab">New Alert Rule</div>
					<Button
						prefix={<RotateCcw size={12} />}
						onClick={handleSwitchToClassicExperience}
						variant="solid"
						color="secondary"
						size="sm"
					>
						Switch to Classic Experience
					</Button>
				</div>
			)}
			<div className="alert-header__content">
				<Input
					type="text"
					value={alertState.name}
					onChange={(e): void =>
						setAlertState({ type: 'SET_ALERT_NAME', payload: e.target.value })
					}
					className="alert-header__input title"
					placeholder="Enter alert rule name"
					data-testid="alert-name-input"
				/>
				<LabelsInput
					labels={alertState.labels}
					onLabelsChange={(labels: Labels): void =>
						setAlertState({ type: 'SET_ALERT_LABELS', payload: labels })
					}
					validateLabelsKey={validateLabelsKey}
				/>
				<div
					className="operational-metadata"
					aria-label="SI/SM routing metadata completeness"
				>
					<div className="operational-metadata__header">
						<div className="operational-metadata__title">SI/SM routing metadata</div>
						<div className="operational-metadata__description">
							Recommended labels for APM routing, PM briefing, and vendor coordination.
						</div>
					</div>
					<div
						className={classNames('operational-metadata__status', {
							'operational-metadata__status--complete':
								missingOperationalLabels.length === 0,
						})}
						role="status"
					>
						{missingOperationalLabels.length
							? `Missing ${missingOperationalLabels.length} recommended labels`
							: 'All recommended SI/SM labels are present'}
					</div>
					<div className="operational-metadata__labels">
						{REQUIRED_OPERATIONAL_LABELS.map(({ description, key, label }) => {
							const isPresent = hasOperationalLabel(alertState.labels, key);

							return (
								<div
									className={classNames('operational-metadata__label', {
										'operational-metadata__label--present': isPresent,
									})}
									key={key}
									title={description}
								>
									<span className="operational-metadata__label-name">{label}</span>
									<span className="operational-metadata__label-key">{key}</span>
									<span className="operational-metadata__label-state">
										{isPresent ? 'Set' : 'Missing'}
									</span>
								</div>
							);
						})}
					</div>
				</div>
				<div className="sop-metadata" aria-label="SOP binding metadata">
					<div className="sop-metadata__header">
						<div className="sop-metadata__title">SOP binding</div>
						<div className="sop-metadata__description">
							Optional metadata used for direct SOP binding, preview links, and cockpit
							warnings.
						</div>
					</div>
					<div
						className={classNames('sop-metadata__status', {
							'sop-metadata__status--complete': hasSopBinding(
								alertState.labels,
								alertState.annotations,
							),
						})}
						role="status"
					>
						{getSopBindingStatus(alertState.labels, alertState.annotations)}
					</div>
					<div className="sop-metadata__grid">
						<label className="sop-metadata__field">
							<span className="sop-metadata__label">SOP ID</span>
							<Input
								aria-describedby={
									sopLabelWarnings.length ? 'sop-metadata-sop-id-warning' : undefined
								}
								aria-invalid={sopLabelWarnings.length > 0}
								className="sop-metadata__input"
								data-testid="sop-metadata-sop_id"
								onChange={(event): void =>
									handleLabelChange(SOP_ID_LABEL, event.target.value)
								}
								placeholder="SOP-PAY-001"
								type="text"
								value={alertState.labels[SOP_ID_LABEL] || ''}
							/>
							{sopLabelWarnings.map((warning, index) => (
								<span
									className="sop-metadata__warning"
									id={index === 0 ? 'sop-metadata-sop-id-warning' : undefined}
									key={warning}
									role="alert"
								>
									{warning}
								</span>
							))}
						</label>
						{SOP_ANNOTATION_FIELDS.map(({ key, label, placeholder }) => {
							const warnings = sopAnnotationWarnings[key] || [];
							const warningId = warnings.length
								? `sop-metadata-${key}-warning`
								: undefined;

							return (
								<label className="sop-metadata__field" key={key}>
									<span className="sop-metadata__label">{label}</span>
									<Input
										aria-describedby={warningId}
										aria-invalid={warnings.length > 0}
										className="sop-metadata__input"
										data-testid={`sop-metadata-${key}`}
										onChange={(event): void =>
											handleAnnotationChange(key, event.target.value)
										}
										placeholder={placeholder}
										type="text"
										value={alertState.annotations[key] || ''}
									/>
									{warnings.map((warning, index) => (
										<span
											className="sop-metadata__warning"
											id={index === 0 ? warningId : undefined}
											key={warning}
											role="alert"
										>
											{warning}
										</span>
									))}
								</label>
							);
						})}
					</div>
					<div className="sop-metadata__preview">
						<div className="sop-metadata__preview-header">
							<div>
								<div className="sop-metadata__preview-title">
									SOP source/search preview
								</div>
								<div className="sop-metadata__preview-description">
									Contract scaffold for resolving SOP metadata before a live source
									connector is enabled.
								</div>
							</div>
							<Button
								color="secondary"
								disabled={isSopPreviewLoading}
								onClick={handlePreviewSop}
								size="sm"
								variant="solid"
							>
								{isSopPreviewLoading ? 'Previewing…' : 'Preview SOP source'}
							</Button>
						</div>
						{sopPreviewError && (
							<div className="sop-metadata__warning" role="alert">
								{sopPreviewError}
							</div>
						)}
						{sopPreview && (
							<div
								className="sop-metadata__preview-grid"
								data-testid="sop-source-preview"
							>
								<div className="sop-metadata__preview-summary">
									<span className="sop-metadata__preview-summary-title">
										Review summary
									</span>
									<span className="sop-metadata__preview-summary-copy">
										{sopPreview.status === 'bound'
											? 'SOP metadata is ready for PM handoff review.'
											: 'Add SOP identity or source metadata before pilot use.'}
									</span>
									<div className="sop-metadata__preview-badges">
										<span className="sop-metadata__preview-badge">
											{sopPreview.access.browserCredentialsAllowed
												? 'Browser credentials allowed'
												: 'Browser credentials blocked'}
										</span>
										{sopPreview.access.requiresServerSideFetch && (
											<span className="sop-metadata__preview-badge">
												Server-side connector required
											</span>
										)}
										{sopPreview.access.auditEventRequired && (
											<span className="sop-metadata__preview-badge">
												Audit required before live fetch
											</span>
										)}
									</div>
								</div>
								<div>
									<span className="sop-metadata__preview-label">Contract</span>
									<span className="sop-metadata__preview-value">
										{sopPreview.contractVersion}
									</span>
								</div>
								<div>
									<span className="sop-metadata__preview-label">Status</span>
									<span
										className={classNames('sop-metadata__preview-value', {
											'sop-metadata__preview-value--complete':
												sopPreview.status === 'bound',
											'sop-metadata__preview-value--warning':
												sopPreview.status !== 'bound',
										})}
									>
										{sopPreview.status}
									</span>
								</div>
								<div>
									<span className="sop-metadata__preview-label">Source</span>
									<span className="sop-metadata__preview-value">
										{sopPreview.source.name}
									</span>
								</div>
								<div>
									<span className="sop-metadata__preview-label">Search</span>
									<span className="sop-metadata__preview-value">
										{sopPreview.search.query || 'No search terms yet'}
									</span>
								</div>
								<div>
									<span className="sop-metadata__preview-label">Preview</span>
									{sopPreview.preview.available && sopPreview.preview.url ? (
										<a
											className="sop-metadata__preview-link"
											href={sopPreview.preview.url}
											target="_blank"
											rel="noopener noreferrer"
										>
											{sopPreview.preview.displayUrl || sopPreview.preview.title}
										</a>
									) : (
										<span className="sop-metadata__preview-value">
											{sopPreview.preview.title || 'Preview unavailable'}
										</span>
									)}
								</div>
								<div>
									<span className="sop-metadata__preview-label">Auth boundary</span>
									<span className="sop-metadata__preview-value">
										{sopPreview.access.mode} · {sopPreview.access.credentialScope}
									</span>
								</div>
								<div>
									<span className="sop-metadata__preview-label">Service account</span>
									<span className="sop-metadata__preview-value">
										{sopPreview.access.recommendedServiceAccountProfile || 'Not required'}
									</span>
								</div>
								<div className="sop-metadata__preview-note">
									<span className="sop-metadata__preview-label">
										Browser credentials
									</span>
									<span className="sop-metadata__preview-value">
										{sopPreview.access.browserCredentialsAllowed
											? 'Allowed'
											: 'Never accepted'}
									</span>
								</div>
								<div className="sop-metadata__preview-note">
									<span className="sop-metadata__preview-label">Boundary note</span>
									<span className="sop-metadata__preview-value">
										{sopPreview.access.message}
									</span>
								</div>
								{sopPreview.warnings?.map((warning) => (
									<div
										className="sop-metadata__preview-warning"
										key={warning}
										role="alert"
									>
										{warning}
									</div>
								))}
							</div>
						)}
					</div>
				</div>
				<div
					className="pm-briefing-metadata"
					aria-label="PM-friendly incident briefing metadata"
				>
					<div className="pm-briefing-metadata__header">
						<div className="pm-briefing-metadata__title">PM incident briefing</div>
						<div className="pm-briefing-metadata__description">
							Optional annotations shown on alert details for SI/SM PM triage.
						</div>
					</div>
					<div className="pm-briefing-metadata__grid">
						{PM_BRIEFING_FIELDS.map(({ key, label, placeholder }) => {
							const warnings = pmBriefingWarnings[key] || [];
							const warningId = warnings.length
								? `pm-briefing-${key}-warning`
								: undefined;

							return (
								<label className="pm-briefing-metadata__field" key={key}>
									<span className="pm-briefing-metadata__label">{label}</span>
									<textarea
										aria-describedby={warningId}
										aria-invalid={warnings.length > 0}
										className="pm-briefing-metadata__textarea"
										value={alertState.annotations[key] || ''}
										onChange={(event): void =>
											handleAnnotationChange(key, event.target.value)
										}
										placeholder={placeholder}
										data-testid={`pm-briefing-${key}`}
										rows={2}
									/>
									{warnings.map((warning, index) => (
										<span
											className="pm-briefing-metadata__warning"
											id={index === 0 ? warningId : undefined}
											key={warning}
											role="alert"
										>
											{warning}
										</span>
									))}
								</label>
							);
						})}
					</div>
				</div>
				<div className="evidence-metadata" aria-label="AI evidence status metadata">
					<div className="evidence-metadata__header">
						<div className="evidence-metadata__title">AI/evidence status</div>
						<div className="evidence-metadata__description">
							Optional annotations shown on alert details and copied into the PM
							handoff.
						</div>
					</div>
					<div className="evidence-metadata__grid">
						{EVIDENCE_METADATA_FIELDS.map(({ key, label, placeholder }) => {
							const warnings = evidenceMetadataWarnings[key] || [];
							const warningId = warnings.length
								? `evidence-metadata-${key}-warning`
								: undefined;

							return (
								<label className="evidence-metadata__field" key={key}>
									<span className="evidence-metadata__label">{label}</span>
									<Input
										aria-describedby={warningId}
										aria-invalid={warnings.length > 0}
										className="evidence-metadata__input"
										data-testid={`evidence-metadata-${key}`}
										onChange={(event): void =>
											handleAnnotationChange(key, event.target.value)
										}
										placeholder={placeholder}
										type="text"
										value={alertState.annotations[key] || ''}
									/>
									{warnings.map((warning, index) => (
										<span
											className="evidence-metadata__warning"
											id={index === 0 ? warningId : undefined}
											key={warning}
											role="alert"
										>
											{warning}
										</span>
									))}
								</label>
							);
						})}
					</div>
				</div>
			</div>
		</div>
	);
}

export default CreateAlertHeader;
