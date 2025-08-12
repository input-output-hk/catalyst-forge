import { test, expect } from '@playwright/test';

test('nav links navigate across routes', async ({ page }) => {
	await page.goto('/');
	await expect(page.getByRole('link', { name: 'Catalyst Forge' })).toBeVisible();
	await page.getByRole('link', { name: 'Projects' }).click();
	await expect(page.getByRole('heading', { name: 'Projects' })).toBeVisible();
	await page.getByRole('link', { name: 'Profile' }).click();
	await expect(page.getByRole('heading', { name: 'Profile' })).toBeVisible();
	await page.getByRole('link', { name: 'Login' }).click();
	await expect(page.getByRole('heading', { name: 'Login' })).toBeVisible();
});
