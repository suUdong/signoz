import { GeneratedAPIInstance } from 'api/generatedAPIInstance';
import type { Labels } from 'types/api/alerts/def';

export type PreviewSopRequest = {
	labels?: Labels;
	annotations?: Labels;
};

export type PreviewSopResult = {
	contractVersion: string;
	status: 'bound' | 'invalid_url' | 'missing';
	source: {
		kind: string;
		name: string;
	};
	binding: {
		sopId?: string;
		bindingId?: string;
		version?: string;
		title?: string;
	};
	search: {
		query: string;
		terms: string[];
	};
	preview: {
		available: boolean;
		title?: string;
		url?: string;
		displayUrl?: string;
	};
	access: {
		mode:
			| 'invalid_url'
			| 'invalid_url_credentials'
			| 'metadata_only'
			| 'public_url'
			| 'server_side_connector';
		requiresServerSideFetch: boolean;
		browserCredentialsAllowed: boolean;
		recommendedServiceAccountProfile?: string;
		credentialScope: string;
		auditEventRequired: boolean;
		message: string;
	};
	warnings?: string[];
};

type PreviewSopResponse = {
	data: PreviewSopResult;
	status: string;
};

export function previewSop(
	data: PreviewSopRequest,
): Promise<PreviewSopResponse> {
	return GeneratedAPIInstance<PreviewSopResponse>({
		url: '/api/v2/rules/sop/preview',
		method: 'POST',
		data,
	});
}
