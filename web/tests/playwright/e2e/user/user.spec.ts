import { test, expect } from '@playwright/test';

test('logs the user out', async ({ page }) => {
    await page.goto('/', { waitUntil: 'networkidle' });
    await page.getByLabel('Open user menu').click();
    await page.locator(".menu .menu-item").filter({ hasText: "Logout" }).click();

    await page.waitForURL('/');
    
    const cookies = await page.context().cookies();
    const pbAuthCookie = cookies.find(cookie => cookie.name === 'pb_auth');
    expect(pbAuthCookie).toBeFalsy();
});
