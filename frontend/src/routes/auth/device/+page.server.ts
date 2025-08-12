import type { Actions, PageServerLoad } from './$types';
import { apiClient } from '$lib/api/client';

export const load: PageServerLoad = async () => {
	return {};
};

export const actions: Actions = {
	init: async ({ request, parent }) => {
		const form = await request.formData();
		const name = form.get('name')?.toString();
		const platform = form.get('platform')?.toString();
		const fingerprint = form.get('fingerprint')?.toString();
		const { apiBaseUrl } = await parent();
		const client = apiClient(apiBaseUrl);
		const { data, error, response } = await client.POST('/device/init', {
			body: { name, platform, fingerprint }
		});
		if (error || !response.ok || !data)
			return { init: null, error: 'Failed to start device authorization' };
		return { init: data };
	},
	poll: async ({ request, cookies, parent }) => {
		const form = await request.formData();
		const device_code = form.get('device_code')?.toString();
		if (!device_code) return { pending: false, error: 'missing_device_code' };

		const { apiBaseUrl } = await parent();
		const client = apiClient(apiBaseUrl);
		const { data, response } = await client.POST('/device/token', {
			body: { device_code }
		});

		// Handle server-indicated pacing and error states per swagger
		if (response.status === 429 && data?.error === 'slow_down') {
			return { pending: true, slowDown: true };
		}
		if (response.status === 401 && data?.error === 'authorization_pending') {
			return { pending: true };
		}
		if (response.status === 401 && data?.error === 'expired_token') {
			return { pending: false, error: 'expired_token' };
		}
		if (response.status === 401 && data?.error === 'access_denied') {
			return { pending: false, error: 'access_denied' };
		}

		if (response.ok && data?.access) {
			cookies.set('cf_session', data.access, {
				httpOnly: true,
				path: '/',
				sameSite: 'lax',
				secure: process.env.NODE_ENV !== 'development',
				maxAge: 60 * 60 * 8
			});
			return { done: true };
		}

		return { pending: false, error: 'unexpected_response' };
	}
};
