import { test, expect } from '@playwright/test';

test('home renders', async ({ page }) => {
	await page.goto('/');
	await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
});
