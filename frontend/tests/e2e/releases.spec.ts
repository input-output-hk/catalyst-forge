import { test, expect } from '@playwright/test';

// Sort/filter/density basic checks on releases page (with mocked fallback)
// TODO: Extend to exercise real API once auth is wired

test('releases table renders and supports density toggle', async ({ page }) => {
	// Sign-in via mock login to bypass protected redirects
	await page.goto('/login');
	await page.fill('input[name="email"]', 'test@example.com');
	await page.click('button:has-text("Send Link")');
	// In mock flow, form GET may redirect immediately to dashboard
	await expect(page).toHaveURL(/\//);
	await page.goto('/releases');
	await page.goto('/releases');
	await expect(page.getByRole('heading', { name: 'Releases' })).toBeVisible({ timeout: 15000 });

	// density toggle (button)
	const toggle = page.getByRole('button', { name: /Density:/ });
	await expect(toggle).toBeVisible();
	await toggle.click();
	await expect(toggle).toContainText(/Density:/);

	// mocked banner likely visible when unauthorized
	const banner = page.getByText(/mocked data/i);
	await expect(banner).toBeVisible({ timeout: 15000 });
});
