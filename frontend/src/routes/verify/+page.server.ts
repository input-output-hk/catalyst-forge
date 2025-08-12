import type { PageServerLoad } from './$types';
import { fail } from '@sveltejs/kit';
import { apiClient } from '$lib/api/client';

export const load: PageServerLoad = async ({ url, parent }) => {
	const token = url.searchParams.get('token');
	if (!token) {
		return fail(400, { ok: false, message: 'Missing token' });
	}

	const { apiBaseUrl } = await parent();
	const client = apiClient(apiBaseUrl);
	const { data, error, response } = await client.GET('/verify', {
		params: { query: { token } }
	});

	if (error || !response.ok) {
		// Surface a friendly message; backend returns 400/401 for invalid/expired
		return { ok: false, message: 'Invalid or expired token' };
	}

	return { ok: true, message: 'Invite verified' };
};
