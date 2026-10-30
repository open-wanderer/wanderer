import { expect, test } from '@playwright/test';
import { buildListsPageFixture, mountListsPageFixture } from './lists-page-fixture';

for (const dev of [true, false]) {
    test.describe(`list page in ${dev ? 'development' : 'production'}`, () => {
        let bundle: string;
        test.beforeAll(async () => { bundle = await buildListsPageFixture(dev); });
        test.beforeEach(({ page }) => { page.on("pageerror", error => { throw error; }); });

        test('dims only replacing results and leaves incremental loading undimmed', async ({ page }) => {
            await mountListsPageFixture(page, bundle);
            await page.evaluate(() => (window as any).fixture.apply({ q: 'new' }));
            await expect(page.locator('.list-results')).toHaveAttribute('aria-busy', 'true');
            await expect(page.locator('.list-card-results')).toHaveCSS('opacity', '0.55');
            await expect(page.getByTestId('list-card')).toHaveCount(30);
            await expect(page.locator('main')).not.toHaveAttribute('aria-busy');
            await page.evaluate(() => (window as any).fixture.requests[0].resolve({
                items: Array.from({ length: 30 }, (_, i) => ({ id: 'new-' + i, name: 'New list ' + i })), page: 1, totalPages: 3,
            }));
            await expect(page.locator('.list-results')).toHaveAttribute('aria-busy', 'false');
            await expect(page.locator('.list-card-results')).toHaveCSS('opacity', '1');
            await page.locator('#list-container').evaluate(element => { element.scrollTop = element.scrollHeight; });
            await expect.poll(() => page.evaluate(() => (window as any).fixture.requests.length)).toBe(2);
            await expect(page.locator('.list-results')).toHaveAttribute('aria-busy', 'false');
            await expect(page.locator('.list-card-results')).toHaveCSS('opacity', '1');
        });

        test('keeps an opened list fully opaque while search results refresh', async ({ page }) => {
            await mountListsPageFixture(page, bundle, true);
            await page.getByRole('button', { name: 'Open trail' }).click();
            await page.getByRole('button', { name: 'Back to list' }).click();
            await expect(page.locator('.list-results')).toHaveAttribute('aria-busy', 'false');
            await expect(page.locator('.list-list')).toHaveCSS('opacity', '1');
            await expect.poll(() => page.getByTestId('list-panel').evaluate(element => {
                let opacity = 1;
                for (let current: Element | null = element; current; current = current.parentElement) {
                    opacity *= Number(getComputedStyle(current).opacity);
                    if (current.matches('main')) break;
                }
                return opacity;
            })).toBe(1);
        });

        test('keeps the same result height while a fixed-body dialog locks the scrolled page', async ({ page }) => {
            await mountListsPageFixture(page, bundle);
            const height = await page.locator('.list-results').evaluate(element => element.getBoundingClientRect().height);
            await page.evaluate(() => {
                document.body.style.position = 'fixed';
                document.body.style.top = '-300px';
                document.body.style.width = '100%';
                window.dispatchEvent(new Event('resize'));
            });
            await expect.poll(() => page.locator('.list-results').evaluate(element => element.getBoundingClientRect().height)).toBe(height);
        });

        test('keeps result height relative to its document position after scrolling', async ({ page }) => {
            await mountListsPageFixture(page, bundle);
            const height = await page.locator('.list-results').evaluate(element => element.getBoundingClientRect().height);
            await page.evaluate(() => {
                const spacer = document.createElement('div'); spacer.style.height = '1800px'; document.body.append(spacer);
                window.scrollTo(0, 300);
            });
            await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(300);
            await page.evaluate(() => window.dispatchEvent(new Event('resize')));
            await expect.poll(() => page.locator('.list-results').evaluate(element => element.getBoundingClientRect().height)).toBe(height);
        });
    });
}
