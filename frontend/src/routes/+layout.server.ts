import type { LayoutServerLoad } from './$types';
import { redirect } from '@sveltejs/kit';
import { getApiBaseUrl } from '$lib/config.server';

export const load: LayoutServerLoad = async ({ locals, url }) => {
	const publicPaths = new Set([
		'/login',
		'/verify',
		'/bootstrap',
		'/auth/device',
		'/auth/device/approve',
		'/auth/callback'
	]);
	const isProtected = !publicPaths.has(url.pathname);
	if (isProtected && !locals.session) {
		throw redirect(302, '/login');
	}
	return {
		sessionEmail: locals.session?.email ?? null,
		pathname: url.pathname,
		apiBaseUrl: getApiBaseUrl()
	};
};
