import createClient from 'openapi-fetch';
import type { paths } from './types';

/**
 * Browser-safe typed API client.
 * Provide the absolute baseUrl (e.g., from layout server data) to avoid importing server code.
 */
export function apiClient(
	baseUrl: string,
	getAccessToken?: () => string | undefined,
	opts?: { fetch?: typeof fetch }
) {
	const client = createClient<paths>({ baseUrl, fetch: opts?.fetch });
	client.use({
		onRequest({ request }) {
			const token = getAccessToken?.();
			if (token) request.headers.set('Authorization', `Bearer ${token}`);
		}
	});
	return client;
}
