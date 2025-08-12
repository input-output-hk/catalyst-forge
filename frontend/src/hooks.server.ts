import type { Handle } from '@sveltejs/kit';

export const handle: Handle = async ({ event, resolve }) => {
	const email = event.cookies.get('cf_session');
	const rt = event.cookies.get('cforge_rt');
	// Treat presence of refresh cookie as authenticated for routing purposes
	if (email) {
		event.locals.session = { email };
	} else if (rt) {
		event.locals.session = { email: 'device' };
	} else {
		event.locals.session = null;
	}

	const response = await resolve(event, {
		filterSerializedResponseHeaders: (name) => name === 'content-type'
	});

	// Security headers
	response.headers.set('Referrer-Policy', 'no-referrer');
	response.headers.set('X-Content-Type-Options', 'nosniff');
	response.headers.set('X-Frame-Options', 'DENY');
	response.headers.set('Permissions-Policy', 'geolocation=()');
	return response;
};
