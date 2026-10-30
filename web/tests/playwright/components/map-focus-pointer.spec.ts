import { expect, test } from '@playwright/test';
import { buildPointerFixture, mountPointerResults, clippedPoint, pressAt, position, expectStablePress } from './map-focus-pointer-fixture';

test.use({ viewport: null, deviceScaleFactor: undefined });

for (const dev of [true, false]) test.describe(`map pointer focus (${dev ? 'development' : 'production'})`, () => {
    let bundle: string;
    test.beforeAll(async () => { bundle = await buildPointerFixture(dev); });

    for (const display of ['cards', 'list'] as const) {
        test(`${display}: a real click on the clipped checkbox zone navigates without scrolling`, async ({ page }) => {
            const records = await mountPointerResults(page, bundle, display);
            const point = await clippedPoint(page, display);
            records.length = 0;
            const before = await pressAt(page, point);
            await expect(page).toHaveURL(new RegExp(point.href + '$'));
            await expect(page.getByTestId('opened-trail')).toHaveText(point.href);
            await expectStablePress(records, before);
        });

        for (const method of ['middle', 'Control'] as const) test(`${display}: ${method} opens the clipped result in a real new tab without scrolling`, async ({ page }) => {
            const records = await mountPointerResults(page, bundle, display);
            const point = await clippedPoint(page, display);
            records.length = 0;
            const popup = page.context().waitForEvent('page');
            if (method === 'Control') await page.keyboard.down('Control');
            const before = await pressAt(page, point, method === 'middle' ? 'middle' : 'left');
            if (method === 'Control') await page.keyboard.up('Control');
            const opened = await popup;
            await expect(opened).toHaveURL(new RegExp(point.href + '$'));
            await expect(opened.getByTestId('opened-trail')).toHaveText(point.href);
            await expectStablePress(records, before, method === 'middle' ? 1 : 0);
            expect(await position(page)).toEqual(before);
            await opened.close();
        });

        test(`${display}: selecting a clipped result preserves the pointer target and both scroll positions`, async ({ page }) => {
            const records = await mountPointerResults(page, bundle, display);
            await page.locator('.trail-result').first().hover();
            await page.locator('.trail-result').first().locator('input').click();
            const point = await clippedPoint(page, display);
            records.length = 0;
            const before = await pressAt(page, point);
            await expect.poll(() => page.evaluate(() => (window as any).mapResultsFixture.selected())).toEqual(['pointer-0', 'pointer-6']);
            await expectStablePress(records, before);
            expect(await position(page)).toEqual(before);
        });

        test(`${display}: right press preserves the clipped link and its context-menu target`, async ({ page }) => {
            const records = await mountPointerResults(page, bundle, display);
            const point = await clippedPoint(page, display);
            records.length = 0;
            const before = await pressAt(page, point, 'right');
            await expectStablePress(records, before, 2);
            await expect.poll(() => records.filter(event => event.type === 'contextmenu').length).toBeGreaterThan(0);
            expect(records.find(event => event.type === 'contextmenu')!.trail).toBe(point.href);
            expect(await position(page)).toEqual(before);
        });
    }

    test('a checkbox in a bottom-clipped list row selects without moving the list or document', async ({ page }) => {
        const records = await mountPointerResults(page, bundle, 'list');
        const point = await clippedPoint(page, 'list', true);
        records.length = 0;
        const before = await pressAt(page, point);
        await expect.poll(() => page.evaluate(() => (window as any).mapResultsFixture.selected())).toEqual(['pointer-6']);
        await expectStablePress(records, before);
        expect(await position(page)).toEqual(before);
    });

    test('a clipped card click in a 1280x330 window does not scroll the document', async ({ page, browserName }) => {
        const session = browserName === 'chromium' ? await page.context().newCDPSession(page) : undefined;
        const windowId = session ? (await session.send('Browser.getWindowForTarget')).windowId : undefined;
        const original = session ? (await session.send('Browser.getWindowBounds', { windowId: windowId! })).bounds
            : await page.evaluate(() => ({ width: innerWidth, height: innerHeight }));
        const resize = async (bounds: { width?: number; height?: number }) => {
            if (session) await session.send('Browser.setWindowBounds', { windowId: windowId!, bounds });
            else await page.setViewportSize({ width: bounds.width!, height: bounds.height! });
        };
        await resize({ width: 1280, height: 330 });
        try {
            const records = await mountPointerResults(page, bundle);
            const point = await clippedPoint(page, 'cards');
            records.length = 0;
            expect(await page.evaluate(() => document.scrollingElement!.scrollHeight - innerHeight)).toBeGreaterThan(0);
            const before = await pressAt(page, point);
            await expect(page.getByTestId('opened-trail')).toHaveText(point.href);
            await expectStablePress(records, before);
        } finally {
            await resize(original);
            await session?.detach();
        }
    });
});
