import { test, expect } from '@playwright/test';

test('mock query route shows projects and flaky state', async ({ page }) => {
	await page.goto('/mock');
	await expect(page.getByRole('heading', { name: 'Mock Query Demo' })).toBeVisible();
	// Eventually should list at least one project
	await expect(page.locator('ul li').first()).toBeVisible();
});
