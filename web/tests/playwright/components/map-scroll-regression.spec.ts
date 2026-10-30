import { expect, test, type Page } from '@playwright/test';
import { buildMapPageFixture } from './map-page-fixture';
import { mount, settle, windowState } from './map-window-fixture';
import { installBrowserFullscreen, setBrowserFullscreen } from '../browser_fullscreen';
import { discardHiddenScrollPosition } from './map-scroll-fixture';

// Use Chromium's native window in the repository configuration. Firefox is
// verified with the external configuration, without forcing a Chromium-only channel.
test.use({ viewport: null, deviceScaleFactor: undefined });

async function mountResults(page: Page, bundle: string) {
    await mount(page, bundle);
    await page.evaluate(() => {
        const fixture = (window as any).fixture;
        fixture.autoResolve = false;
        const push = fixture.requests.push.bind(fixture.requests);
        fixture.requests.push = (...requests: any[]) => {
            const length = push(...requests);
            for (const request of requests) request.resolve(request.channel === 'map'
                ? { mapTrails: [], clusters: { type: 'FeatureCollection', features: [] } }
                : { items: Array.from({ length: 40 }, (_, index) => ({ id: 'scroll-' + index, author: 'tester', name: 'Trail ' + index })),
                    page: request.body.page, totalPages: 2, totalHits: 80 });
            return length;
        };
        fixture.apply({ q: 'scroll results' });
    });
    await expect(page.locator('.trail-result')).toHaveCount(40);
    await page.addStyleTag({ content: '.trail-result a { min-height: 80px }' });
    await settle(page);
}

async function resizeWindow(page: Page, browserName: string) {
    if (browserName === 'chromium') {
        const session = await page.context().newCDPSession(page);
        const { windowId } = await session.send('Browser.getWindowForTarget');
        await session.send('Browser.setWindowBounds', { windowId, bounds: { width: 1240, height: 760 } });
        await session.detach();
    } else await page.setViewportSize({ width: 1240, height: 760 });
    await settle(page);
}

async function changeFullscreen(page: Page, browserName: string, fullscreen: boolean) {
    if (browserName === 'chromium') await windowState(page, fullscreen);
    else {
        await setBrowserFullscreen(page, fullscreen);
        await settle(page);
    }
}

async function activeResultIsVisible(page: Page) {
    return page.evaluate(() => {
        const list = document.querySelector<HTMLElement>('.trail-results')!;
        const focused = document.activeElement;
        if (!(focused instanceof HTMLElement) || !list.contains(focused)) return false;
        const bounds = list.getBoundingClientRect(), item = focused.getBoundingClientRect();
        return item.top >= bounds.top - 2 && item.bottom <= bounds.bottom + 2;
    });
}

async function wheel(page: Page, delta: number, times: number) {
    const box = (await page.locator('.trail-results').boundingBox())!;
    for (let index = 0; index < times; index++) {
        // Hovering the next card mutates its selection affordance before the
        // browser dispatches the next scroll event, as with a moving touchpad.
        await page.mouse.move(box.x + box.width / 2 + index % 2, box.y + box.height / 2 + index % 2);
        await page.mouse.wheel(0, delta);
        await page.waitForTimeout(80);
    }
}

for (const dev of [true, false]) {
    test.describe(`list scroll transitions (${dev ? 'development' : 'production'})`, () => {
        let bundle: string;
        test.beforeAll(async () => { bundle = await buildMapPageFixture(dev, true, true); });

        for (const layout of ['collapse', 'window']) {
            for (const interaction of ['wheel', 'Tab', 'Shift+Tab', 'deselect']) {
                test(`${interaction} remains visible after ${layout}`, async ({ page, browserName }) => {
                    await mountResults(page, bundle);
                    const list = page.locator('.trail-results');
                    if (layout === 'collapse') {
                        await page.locator('.result-header .collapse-results').click();
                        await expect(list).not.toBeVisible();
                        await page.locator('.collapse-results').click();
                        await expect(list).toBeVisible();
                    } else await resizeWindow(page, browserName);
                    await settle(page);
                    if (interaction === 'wheel') {
                        await wheel(page, 180, 8);
                        await expect.poll(() => list.evaluate(node => node.scrollTop)).toBe(1440);
                        await page.getByText('trail-results-load-more', { exact: true }).scrollIntoViewIfNeeded();
                        await page.getByText('trail-results-load-more', { exact: true }).click();
                        await expect(page.locator('.trail-result')).toHaveCount(40);
                        await expect.poll(() => page.evaluate(() => (window as any).fixture.requests.filter((request: any) => request.channel === 'results').at(-1).body.page)).toBe(2);
                    } else if (interaction === 'Tab' || interaction === 'Shift+Tab') {
                        const linksInList = page.locator('.trail-result a');
                        await (interaction === 'Tab' ? linksInList.first() : linksInList.nth(24)).focus();
                        let links = 0;
                        for (let stop = 0; stop < 24; stop++) {
                            await page.keyboard.press(interaction);
                            if (await page.evaluate(() => document.activeElement?.matches('.trail-result a'))) {
                                links++;
                                await expect.poll(() => activeResultIsVisible(page)).toBe(true);
                            }
                        }
                        expect(links).toBeGreaterThanOrEqual(12);
                    } else {
                        await page.locator('.trail-result').nth(1).hover();
                        await page.locator('.trail-result').nth(1).locator('input[type="checkbox"]').click();
                        await list.evaluate(node => node.scrollTop = 1200);
                        await expect.poll(() => list.evaluate(node => node.scrollTop)).toBe(1200);
                        const clear = page.getByLabel('trail-results-clear-selection');
                        await clear.focus();
                        await page.keyboard.press('Enter');
                        await expect(page.locator('.trail-result a').first()).toBeFocused();
                        await expect.poll(() => activeResultIsVisible(page)).toBe(true);
                    }
                });
            }
        }

        for (const interaction of ['wheel', 'Tab']) {
            test(`fullscreen returns a visible trigger after overlay ${interaction}`, async ({ page, browserName }) => {
                if (browserName === 'firefox') await installBrowserFullscreen(page);
                await mountResults(page, bundle);
                const trigger = page.locator('.trail-result a').nth(24);
                // Establish keyboard modality before the programmatic trigger
                // focus, independently of a previous native window's pointer.
                await page.keyboard.press('Tab');
                await trigger.focus();
                await expect.poll(() => activeResultIsVisible(page)).toBe(true);
                await changeFullscreen(page, browserName, true);
                await page.locator('.fullscreen-results-toggle').click();
                await expect(page.locator('.trail-results')).toBeVisible();
                if (interaction === 'wheel') await wheel(page, -180, 8);
                else {
                    await page.locator('.trail-result a').first().focus();
                    for (let index = 0; index < 4; index++) await page.keyboard.press('Tab');
                }
                await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
                await expect.poll(() => page.locator('.trail-results').evaluate(node => node.scrollTop)).toBeLessThan(1000);
                await changeFullscreen(page, browserName, false);
                await expect(trigger).toBeFocused();
                await expect.poll(() => activeResultIsVisible(page)).toBe(true);
            });
        }

        test('list transitions work without Element.checkVisibility', async ({ page, browserName }) => {
            await page.addInitScript(() => { delete (Element.prototype as any).checkVisibility; });
            const errors: string[] = [];
            page.on('pageerror', error => errors.push(error.message));
            await mountResults(page, bundle);
            const list = page.locator('.trail-results');
            await list.evaluate(node => node.scrollTop = 1440);
            await expect.poll(() => list.evaluate(node => node.scrollTop)).toBe(1440);
            await discardHiddenScrollPosition(page);
            await page.locator('.result-header .collapse-results').click();
            await expect(list).not.toBeVisible();
            await page.locator('.collapse-results').click();
            await expect(list).toBeVisible();
            await expect.poll(() => list.evaluate(node => node.scrollTop)).toBe(1440);
            await resizeWindow(page, browserName);
            expect(errors).toEqual([]);
        });
    });
}
