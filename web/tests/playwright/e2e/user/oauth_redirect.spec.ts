import { expect, test } from '../../fixtures';

test('a forged app state hands nothing to the app', async ({ page }) => {
    await page.goto('/login/redirect?code=abc&state=wanderer-app.e30.AAAA');

    await expect(page.getByText('app_login_refused')).toBeVisible();
    await expect(page.locator('main a')).toHaveCount(0);
});

test('the old app marker hands nothing to the app', async ({ page }) => {
    await page.goto('/login/redirect?code=abc&state=abc123.wanderer-app');

    await expect(page.getByText('missing_provider')).toBeVisible();
    await expect(page.locator('main a')).toHaveCount(0);
});

test('error_uri becomes a link only for http(s)', async ({ page }) => {
    await page.goto(`/login/redirect?error=x&error_uri=${encodeURIComponent('javascript:alert(1)')}`);
    await expect(page.getByRole('heading', { name: 'x' })).toBeVisible();
    await expect(page.locator('main a')).toHaveCount(0);

    await page.goto(`/login/redirect?error=x&error_uri=${encodeURIComponent('https://idp.example/help')}`);
    await expect(page.locator('main a')).toHaveAttribute('href', 'https://idp.example/help');
});
