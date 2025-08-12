import { describe, expect, it } from 'vitest';
import { getApiBaseUrl } from '$lib/config.server';
import { apiClient } from './client';

describe('API client runtime config', () => {
	it('uses default when env not set and trims trailing slash', () => {
		const prev = process.env.FORGE_API_BASE_URL;
		delete process.env.FORGE_API_BASE_URL;
		expect(getApiBaseUrl()).toBe('http://127.0.0.1:5050');
		process.env.FORGE_API_BASE_URL = 'http://127.0.0.1:5050/';
		expect(getApiBaseUrl()).toBe('http://127.0.0.1:5050');
		if (prev !== undefined) process.env.FORGE_API_BASE_URL = prev;
	});

	it('attaches Authorization header when token getter returns a value', async () => {
		let seenAuth: string | null = null;
		const fakeFetch: typeof fetch = (input: any, init?: any) => {
			const req = input instanceof Request ? input : new Request(String(input), init);
			seenAuth = req.headers.get('Authorization');
			return Promise.resolve(
				new Response('{}', { status: 200, headers: { 'content-type': 'application/json' } })
			);
		};
		const client = apiClient('http://127.0.0.1:5050', () => 'abc', { fetch: fakeFetch });
		await client.GET('/healthz');
		expect(seenAuth).toBe('Bearer abc');
	});
});
