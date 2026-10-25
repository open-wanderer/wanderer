import { expect, test } from '../../fixtures';

test('logs the user out', async ({ page }) => {
    await page.goto('/', { waitUntil: 'networkidle' });
    await page.getByLabel('Open user menu').click();
    await page.locator('.menu .menu-item').filter({ hasText: 'Logout' }).click();

    await page.waitForURL('/');

    const cookies = await page.context().cookies();
    expect(cookies.find((cookie) => cookie.name === 'pb_auth')).toBeFalsy();
});
