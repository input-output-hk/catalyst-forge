import type { Actions, PageServerLoad } from './$types';
import { apiClient } from '$lib/api/client';

export const load: PageServerLoad = async () => {
	return {};
};

export const actions: Actions = {
	default: async ({ request, parent }) => {
		const form = await request.formData();
		const user_code = form.get('user_code')?.toString();
		if (!user_code) return { ok: false, error: 'missing_user_code' };
		const { apiBaseUrl } = await parent();
		const client = apiClient(apiBaseUrl);
		const { response } = await client.POST('/device/approve', { body: { user_code } });
		if (!response.ok) return { ok: false, error: 'approval_failed' };
		return { ok: true };
	}
};
