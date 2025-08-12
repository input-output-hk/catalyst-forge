import { redirect, type Actions, fail } from '@sveltejs/kit';

export const actions: Actions = {};

export async function load({ url, cookies }) {
	const token = url.searchParams.get('token');
	const email = url.searchParams.get('email');
	if (!token || !email) {
		return fail(400, { message: 'missing token or email' });
	}

	cookies.set('cf_session', email, {
		httpOnly: true,
		path: '/',
		sameSite: 'lax',
		secure: process.env.NODE_ENV !== 'development',
		maxAge: 60 * 60 * 8
	});

	throw redirect(302, '/');
}
