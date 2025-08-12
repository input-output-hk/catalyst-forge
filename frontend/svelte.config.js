import adapter from '@sveltejs/adapter-auto';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	// Consult https://svelte.dev/docs/kit/integrations
	// for more information about preprocessors
	preprocess: vitePreprocess(),

	kit: {
		// adapter-auto only supports some environments, see https://svelte.dev/docs/kit/adapter-auto for a list.
		// If your environment is not supported, or you settled on a specific environment, switch out the adapter.
		// See https://svelte.dev/docs/kit/adapters for more information about adapters.
		adapter: adapter(),
		// Allow cross-origin POSTs in development to avoid CSRF false-positives when
		// using different hostnames (e.g., localhost vs 127.0.0.1) or ports.
		csrf: {
			checkOrigin: process.env.NODE_ENV === 'production'
		},
		csp: {
			mode: 'auto',
			directives: {
				'default-src': ['self'],
				'script-src': ['self'],
				'style-src': ['self', 'https:', 'unsafe-inline'],
				'img-src': ['self', 'https:', 'data:'],
				'connect-src': ['self', 'http:', 'https:'],
				'frame-ancestors': ['none']
			}
		}
	}
};

export default config;
