export function decodeJwtClaims(token: string): Record<string, unknown> | null {
	if (!token) return null;
	try {
		const payload = token.split('.')[1];
		if (!payload) return null;
		const json =
			typeof atob === 'function'
				? atob(payload.replace(/-/g, '+').replace(/_/g, '/'))
				: Buffer.from(payload.replace(/-/g, '+').replace(/_/g, '/'), 'base64').toString('binary');
		return JSON.parse(
			decodeURIComponent(
				json
					.split('')
					.map(function (c) {
						return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
					})
					.join('')
			)
		);
	} catch {
		return null;
	}
}
