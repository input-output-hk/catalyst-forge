import type { RequestHandler } from '@sveltejs/kit';

export const GET: RequestHandler = async ({ cookies }) => {
	cookies.set('cf_session', '', { path: '/', maxAge: 0 });
	return new Response(null, { status: 302, headers: { Location: '/login' } });
};
