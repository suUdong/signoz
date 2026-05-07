import { Button, Input, Popover, Tooltip, Typography } from 'antd';
import { previewNotificationTemplate } from 'api/v2/rules/previewNotificationTemplate';
import { Info } from 'lucide-react';
import { useCallback, useState } from 'react';

import { useCreateAlertState } from '../context';
import {
	getUnknownIncidentTemplateVariables,
	INCIDENT_TEMPLATE_VARIABLES,
} from './incidentTemplateVariables';

function NotificationMessage(): JSX.Element {
	const { alertState, notificationSettings, setNotificationSettings } =
		useCreateAlertState();
	const [previewBody, setPreviewBody] = useState<string>('');
	const [previewMissingVars, setPreviewMissingVars] = useState<string[]>([]);
	const [previewError, setPreviewError] = useState<string>('');
	const [isPreviewLoading, setIsPreviewLoading] = useState(false);

	const unknownIncidentVariables = getUnknownIncidentTemplateVariables(
		notificationSettings.description,
	);

	const handlePreview = useCallback(async (): Promise<void> => {
		setPreviewError('');
		setIsPreviewLoading(true);

		try {
			const response = await previewNotificationTemplate({
				template: notificationSettings.description,
				labels: alertState.labels,
				annotations: alertState.annotations,
			});

			setPreviewBody(response.data.body);
			setPreviewMissingVars(response.data.missingVars || []);
		} catch {
			setPreviewBody('');
			setPreviewMissingVars([]);
			setPreviewError('Unable to preview this notification message.');
		} finally {
			setIsPreviewLoading(false);
		}
	}, [
		alertState.annotations,
		alertState.labels,
		notificationSettings.description,
	]);

	const handleDescriptionChange = useCallback(
		(value: string): void => {
			setPreviewBody('');
			setPreviewMissingVars([]);
			setPreviewError('');
			setNotificationSettings({
				type: 'SET_DESCRIPTION',
				payload: value,
			});
		},
		[setNotificationSettings],
	);

	const templateVariableContent = (
		<div className="template-variable-content">
			<Typography.Text strong>Incident template variables</Typography.Text>
			<Typography.Text className="template-variable-content-description">
				Use these DS-APM variables to include PM briefing, SI/SM routing, and SOP
				binding context in notifications.
			</Typography.Text>
			{INCIDENT_TEMPLATE_VARIABLES.map((item) => (
				<div className="template-variable-content-item" key={item.variable}>
					<code>{item.variable}</code>
					<Typography.Text>{item.description}</Typography.Text>
				</div>
			))}
		</div>
	);

	return (
		<div className="notification-message-container">
			<div className="notification-message-header">
				<div className="notification-message-header-content">
					<Typography.Text className="notification-message-header-title">
						Notification Message
						<Tooltip title="Customize the message content sent in alert notifications. Variables like $incident.impact_summary, $incident.next_action, $incident.service_name, and $incident.sop_id are replaced when the alert fires.">
							<Info size={16} />
						</Tooltip>
					</Typography.Text>
					<Typography.Text className="notification-message-header-description">
						Custom message content for alert notifications. Use template variables to
						include dynamic information.
					</Typography.Text>
				</div>
				<div className="notification-message-header-actions">
					<Popover content={templateVariableContent} trigger="click">
						<Button type="text">
							<Info size={12} />
							Variables
						</Button>
					</Popover>
					<Button
						disabled={!notificationSettings.description.trim()}
						loading={isPreviewLoading}
						onClick={handlePreview}
						type="text"
					>
						Preview
					</Button>
				</div>
			</div>
			<div className="notification-message-incident-hint">
				<Typography.Text>
					PM/SI-SM context is available with variables like{' '}
					<code>$incident.impact_summary</code>, <code>$incident.next_action</code>,
					<code>$incident.service_name</code>, and <code>$incident.sop_id</code>.
				</Typography.Text>
			</div>
			{unknownIncidentVariables.length > 0 && (
				<div className="notification-message-warning" role="alert">
					Unknown incident template variable
					{unknownIncidentVariables.length > 1 ? 's' : ''}:{' '}
					{unknownIncidentVariables.join(', ')}. Use the Variables list to pick a
					supported $incident.* field.
				</div>
			)}
			{previewError && (
				<div className="notification-message-warning" role="alert">
					{previewError}
				</div>
			)}
			{previewBody && (
				<div className="notification-message-preview">
					<Typography.Text className="notification-message-preview-title">
						Preview
					</Typography.Text>
					<pre>{previewBody}</pre>
					{previewMissingVars.length > 0 && (
						<div className="notification-message-warning" role="alert">
							Preview missing variables: {previewMissingVars.join(', ')}
						</div>
					)}
				</div>
			)}
			<Input.TextArea
				value={notificationSettings.description}
				onChange={(e): void => handleDescriptionChange(e.target.value)}
				placeholder="Enter notification message..."
			/>
		</div>
	);
}

export default NotificationMessage;
