import { expect, test } from '@playwright/test';

test.use({ javaScriptEnabled: false });

test('server-rendered map toolbar has its final controls at phone and desktop widths without hydration', async ({ page }) => {
    for (const width of [390, 1280]) await test.step(`${width}px without JavaScript`, async () => {
        await page.setViewportSize({ width, height: 844 });
        await page.goto('/map?lat=46.95&lon=7.44');
        const toolbar = page.locator('.map-search-toolbar');
        await expect(toolbar.locator('.search-heading button[aria-expanded]')).toBeVisible();
        const toggle = toolbar.locator('.map-view-toggle');
        const sort = toolbar.locator('button[aria-controls$="-sort"]');
        const view = toolbar.locator('button[aria-controls$="-view"]');
        if (width < 768) {
            await expect(toggle).toBeVisible();
            await expect(sort).toBeHidden(); await expect(view).toBeHidden();
        } else {
            await expect(toggle).toBeHidden();
            await expect(sort).toBeVisible(); await expect(view).toBeVisible();
        }
    });
});
