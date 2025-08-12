import { test, expect } from '@playwright/test';

// Smoke of device flow UI only; backend calls are mocked/unauthorized in local dev
// TODO: Convert to full happy-path when device auth is wired

test('device auth page renders (smoke)', async ({ page }) => {
	await page.goto('/auth/device');
	// Page title or intro copy
	await expect(page.getByRole('heading', { name: /Approve a Device/i })).toBeVisible({
		timeout: 15000
	});

	// Submit the init form
	// Presence of Start button is enough for now
	await expect(page.getByRole('button', { name: /Start/i })).toBeVisible();
});
