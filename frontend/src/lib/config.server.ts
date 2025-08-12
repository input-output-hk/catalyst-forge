/**
 * Returns the API base URL from a runtime environment variable, with a safe default.
 * Prefer FORGE_API_BASE_URL to enable true runtime configuration in container/K8s.
 * Falls back to 127.0.0.1:5050 for local docker-compose.
 */
export function getApiBaseUrl(): string {
	const fromEnv = process.env.FORGE_API_BASE_URL || process.env.VITE_API_URL;
	const base = (fromEnv ?? 'http://127.0.0.1:5050').trim();
	return base.endsWith('/') ? base.slice(0, -1) : base;
}

/**
 * Returns the bootstrap email used for the /auth/bootstrap endpoint.
 * Prefer FORGE_BOOTSTRAP_EMAIL; fallback to NEXT_PUBLIC_BOOTSTRAP_EMAIL/VITE_BOOTSTRAP_EMAIL for dev.
 */
export function getBootstrapEmail(): string | null {
	const fromEnv = process.env.FORGE_BOOTSTRAP_EMAIL || process.env.NEXT_PUBLIC_BOOTSTRAP_EMAIL || process.env.VITE_BOOTSTRAP_EMAIL;
	return (fromEnv ?? '').trim() || null;
}
