import { page } from '@vitest/browser/context';
import { describe, expect, it } from 'vitest';
import { render } from 'vitest-browser-svelte';
import Page from './+page.svelte';
import TestProvider from '$lib/ui/TestProvider.svelte';

describe('/+page.svelte', () => {
	it.skip('should render h1', async () => {
		render(TestProvider, { slots: { default: Page } });
		const h1 = page.getByText('Welcome to Catalyst Forge');
		await expect.element(h1).toBeInTheDocument();
	});
});
