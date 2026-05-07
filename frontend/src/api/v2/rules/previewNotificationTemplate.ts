import { GeneratedAPIInstance } from 'api/generatedAPIInstance';
import type { Labels } from 'types/api/alerts/def';

export type PreviewNotificationTemplateRequest = {
	template: string;
	labels?: Labels;
	annotations?: Labels;
	value?: string;
	threshold?: string;
};

export type PreviewNotificationTemplateResult = {
	body: string;
	missingVars?: string[];
};

type PreviewNotificationTemplateResponse = {
	data: PreviewNotificationTemplateResult;
	status: string;
};

export function previewNotificationTemplate(
	data: PreviewNotificationTemplateRequest,
): Promise<PreviewNotificationTemplateResponse> {
	return GeneratedAPIInstance<PreviewNotificationTemplateResponse>({
		url: '/api/v2/rules/notification_template/preview',
		method: 'POST',
		data,
	});
}
