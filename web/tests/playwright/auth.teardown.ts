import { test as teardown } from '@playwright/test';

teardown('delete user', async ({ page }) => {
    await page.goto('/settings/account', { waitUntil: 'networkidle' });
    await page.locator("#delete-account").click();
    
    // Wait for modal to appear
    await page.locator("#confirm").waitFor({ state: 'visible' });
    await page.locator("#confirm").click();

    await page.waitForURL('/');
});