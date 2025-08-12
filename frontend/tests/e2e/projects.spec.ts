import { test, expect } from '@playwright/test';

test('releases page route exists', async ({ page }) => {
	// Mock login first to satisfy route guard
	await page.goto('/login');
	await page.fill('input[name="email"]', 'test@example.com');
	await page.click('button:has-text("Send Link")');
	await expect(page).toHaveURL(/\//);
	await page.goto('/releases');
	await expect(page.getByRole('heading', { name: 'Releases' })).toBeVisible({ timeout: 15000 });
});
