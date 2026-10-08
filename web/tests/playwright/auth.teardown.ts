import { test as teardown } from '@playwright/test';

teardown('delete user', async ({ page }) => {
    await page.goto('/settings/account', { waitUntil: 'networkidle' });
    await page.locator("#delete-account").click();
    
    // Wait for modal to appear
    const confirmButton = page.locator('dialog[open] button[name="delete"]');
    await confirmButton.waitFor({ state: 'visible' });
    await confirmButton.click();

    await page.waitForURL('/');
});
