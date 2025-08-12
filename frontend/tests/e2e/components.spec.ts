import { test, expect } from '@playwright/test';

test('components showcase interactions', async ({ page }) => {
	await page.goto('/components?openModal=1');
	await expect(page.getByRole('heading', { name: 'UI Primitives' })).toBeVisible();

	// Buttons visible
	await expect(page.getByRole('button', { name: 'Primary' })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Secondary' })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Ghost' })).toBeVisible();

	// Modal open/close with keyboard
	await expect(page.locator('.cf-modal__backdrop')).toBeVisible();
	await page.keyboard.press('Escape');
	await expect(page.locator('.cf-modal__backdrop')).toHaveCount(0);

	// Drawer open/close
	await page.getByTestId('open-drawer').click({ force: true });
	// Drawer isn't a dialog by role, check panel presence
	await expect(page.locator('.cf-drawer__panel')).toBeVisible();
	await page.getByRole('button', { name: 'Close' }).click();
	await expect(page.locator('.cf-drawer__panel')).toHaveCount(0);

	// Toast appears
	await page.getByRole('button', { name: 'Show Toast' }).click();
	await expect(page.locator('.cf-toast')).toBeVisible();
});
