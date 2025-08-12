import { test, expect } from '@playwright/test';

test('mock login and logout flow', async ({ page }) => {
	await page.goto('/login');
	await page.getByLabel('Email').fill('user@example.com');
	await page.getByRole('button', { name: 'Send Link' }).click();
	await expect(page).toHaveURL(/\/?$/);
	// header shows email and logout
	await expect(page.getByText('user@example.com')).toBeVisible();
	await page.getByRole('link', { name: 'Logout' }).click();
	await expect(page).toHaveURL(/\/login$/);
});
