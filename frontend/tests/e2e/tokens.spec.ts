import { test, expect } from '@playwright/test';

test('tokens page displays color swatches and headings', async ({ page }) => {
	await page.goto('/tokens');
	await expect(page.getByRole('heading', { name: 'Brand Tokens' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Colors' })).toBeVisible();
	const swatches = page.locator('.swatch');
	const count = await swatches.count();
	expect(count).toBeGreaterThan(3);
});
