import type { Actions, PageServerLoad } from './$types';
import { apiClient } from '$lib/api/client';

export const load: PageServerLoad = async ({ cookies, parent, fetch }) => {
	const { apiBaseUrl } = await parent();
	const token = cookies.get('cf_session') ?? '';
	const client = apiClient(apiBaseUrl, () => token, { fetch });

	// Attempt to fetch JWKS for convenience; ignore failures in unauth state
	let jwks: Record<string, unknown> | null = null;
	try {
		const res = await client.GET('/.well-known/jwks.json');
		jwks = (res.data as any) ?? null;
	} catch {
		jwks = null;
	}

	// Temporary sessions list mock until real tokens endpoints are wired
	const sessions = [
		{
			id: 'current',
			user_agent: 'This device',
			created_at: new Date().toISOString(),
			active: true
		}
	];

	return {
		token,
		jwksUrl: `${apiBaseUrl}/.well-known/jwks.json`,
		jwks,
		sessions,
		mocked: true // TODO: Replace with real session list once auth is wired
	};
};

export const actions: Actions = {
	// TODO: Replace with real POST /tokens/refresh once available
	refresh: async ({ request, parent, cookies, fetch }) => {
		const { apiBaseUrl } = await parent();
		const token = cookies.get('cf_session') ?? '';
		try {
			const res = await fetch(`${apiBaseUrl}/tokens/refresh`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: token ? `Bearer ${token}` : ''
				}
			});
			if (res.ok) {
				const json = await res.json().catch(() => ({}));
				return { ok: true, data: json };
			}
			return { ok: false, error: `HTTP ${res.status}` };
		} catch (e: any) {
			return { ok: false, error: e?.message ?? 'network error', mocked: true };
		}
	},
	// TODO: Replace with real POST /tokens/revoke once available
	revoke: async ({ request, parent, cookies, fetch }) => {
		const form = await request.formData();
		const sessionId = String(form.get('session_id') ?? '');
		const { apiBaseUrl } = await parent();
		const token = cookies.get('cf_session') ?? '';
		try {
			const res = await fetch(`${apiBaseUrl}/tokens/revoke`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: token ? `Bearer ${token}` : ''
				},
				body: JSON.stringify({ session_id: sessionId })
			});
			if (res.ok) return { ok: true };
			return { ok: false, error: `HTTP ${res.status}` };
		} catch (e: any) {
			return { ok: false, error: e?.message ?? 'network error', mocked: true };
		}
	}
};
