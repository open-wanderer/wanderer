import { expect, type Page } from '@playwright/test';

export async function expectActiveNavigation(page: Page, path: string) {
    const nav = page.locator('#nav-bar-links');
    await expect.poll(async () => {
        const indicator = await nav.locator(':scope > div').boundingBox();
        const active = await nav.locator(`a[href="${path}"]`).boundingBox();
        return Boolean(indicator && active && indicator.width > 0
            && Math.abs(indicator.x - active.x + 8) < 1
            && Math.abs(indicator.width - active.width - 16) < 1);
    }).toBe(true);
}
