import { test, expect } from '@playwright/test';

test('dashboard shows metrics and recent table', async ({ page }) => {
	await page.goto('/');
	await expect(page.getByRole('heading', { name: 'Welcome to Catalyst Forge' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Recent Activity' })).toBeVisible();
	await expect(page.locator('table tbody tr').first()).toBeVisible();
});
