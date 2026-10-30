import { expect, test, type Page } from '@playwright/test';
import { buildMapPageFixture } from './map-page-fixture';
import { mount, settle } from './map-window-fixture';

test.use({ viewport: null, deviceScaleFactor: undefined });

async function mountResults(page: Page, bundle: string) {
    await mount(page, bundle);
    await page.evaluate(() => {
        const fixture = (window as any).fixture;
        fixture.autoResolve = false;
        const push = fixture.requests.push.bind(fixture.requests);
        fixture.requests.push = (...requests: any[]) => {
            const count = push(...requests);
            for (const request of requests) request.resolve(request.channel === 'map'
                ? { mapTrails: [], clusters: { type: 'FeatureCollection', features: [] } }
                : { items: Array.from({ length: 40 }, (_, index) => ({ id: 'return-' + index, author: 'tester', name: 'Trail ' + index })),
                    page: 1, totalPages: 1, totalHits: 40 });
            return count;
        };
        fixture.apply({ q: 'focus return' });
    });
    await expect(page.locator('.trail-result')).toHaveCount(40);
    await page.addStyleTag({ content: '.trail-result a { min-height: 386px }' });
    await settle(page);
}

const position = (page: Page) => page.evaluate(() => ({
    list: document.querySelector<HTMLElement>('.trail-results')!.scrollTop,
    document: window.scrollY,
}));

async function focusResult(page: Page, method: string) {
    if (method === 'keyboard') {
        await page.locator('.result-header .collapse-results').focus();
        await page.keyboard.press('Tab');
        await expect(page.locator('.trail-result a').first()).toBeFocused();
        expect(await page.locator('.trail-result a').first().evaluate(node => node.matches(':focus-visible'))).toBe(true);
    } else {
        await page.locator('.trail-result').first().hover();
        await page.locator('.trail-result').first().locator('input').click();
        await expect(page.locator('.trail-result').first().locator('input')).toBeFocused();
    }
    // Scroll at the pane's padding edge, preserving focus in the list and
    // avoiding the map's existing hover popup taking focus from the result.
    const list = (await page.locator('.trail-results').boundingBox())!;
    await page.mouse.move(list.x + 2, list.y + list.height / 2);
    for (let step = 0; step < 8; step++) {
        await page.mouse.wheel(0, 180);
        await page.waitForTimeout(80);
    }
    await expect.poll(async () => (await position(page)).list).toBe(1440);
    const active = await page.evaluate(() => document.activeElement?.getBoundingClientRect().bottom);
    const top = await page.locator('.trail-results').evaluate(node => node.getBoundingClientRect().top);
    expect(active).toBeLessThan(top);
}

for (const dev of [true, false]) {
    test.describe(`window focus return (${dev ? 'development' : 'production'})`, () => {
        let bundle: string;
        test.beforeAll(async () => { bundle = await buildMapPageFixture(dev, true, true); });

        for (const focus of ['keyboard', 'checkbox']) for (const transition of ['tab', 'minimize']) {
            test(`${focus} focus keeps list position after ${transition}`, async ({ page, context, browserName }) => {
                test.skip(browserName !== 'chromium', 'Native focus-emulation and window-state controls use Chromium CDP.');
                await mountResults(page, bundle);
                const session = await context.newCDPSession(page);
                await session.send('Emulation.setFocusEmulationEnabled', { enabled: false });
                await page.bringToFront();
                await expect.poll(() => page.evaluate(() => document.hasFocus())).toBe(true);
                await focusResult(page, focus);
                const before = await position(page);
                // Prove the browser delivered the focus-in event that used to
                // cause the jump, rather than accepting an inert headless tab.
                await page.evaluate(() => {
                    (window as any).returnedFocus = [];
                    document.querySelector('.trail-results')!.addEventListener('focusin', event => {
                        (window as any).returnedFocus.push({ target: event.target === document.activeElement,
                            related: (event as FocusEvent).relatedTarget === null });
                    });
                });
                if (transition === 'tab') {
                    const other = await context.newPage();
                    await other.setContent('<button>Other tab</button>');
                    await other.bringToFront();
                    await expect.poll(() => page.evaluate(() => document.hasFocus())).toBe(false);
                    await page.bringToFront();
                    await expect.poll(() => page.evaluate(() => document.hasFocus())).toBe(true);
                    await other.close();
                } else {
                    const { windowId } = await session.send('Browser.getWindowForTarget');
                    await session.send('Browser.setWindowBounds', { windowId, bounds: { windowState: 'minimized' } });
                    await expect.poll(() => page.evaluate(() => document.hasFocus())).toBe(false);
                    await session.send('Browser.setWindowBounds', { windowId, bounds: { windowState: 'normal' } });
                    await page.bringToFront();
                    await expect.poll(() => page.evaluate(() => document.hasFocus())).toBe(true);
                }
                await expect.poll(() => page.evaluate(() => (window as any).returnedFocus.some((event: any) => event.target && event.related))).toBe(true);
                await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
                expect(await position(page)).toEqual(before);
                await session.detach();
            });
        }

        test('the first Tab from body still reveals its result completely', async ({ page }) => {
            await mountResults(page, bundle);
            await page.locator('.trail-results').evaluate(node => node.scrollTop = 1440);
            // Activate the document itself. Firefox can move focus to browser
            // chrome if blur() is called on an initially unfocused body.
            await page.mouse.click(4, 4);
            await expect.poll(() => page.evaluate(() => document.activeElement === document.body)).toBe(true);
            await page.evaluate(() => {
                // Keep the body's initial, unfocused state, so the first real
                // Tab has relatedTarget=null rather than a prior control.
                for (const node of document.querySelectorAll<HTMLElement>('button, input')) node.tabIndex = -1;
                document.querySelector('.trail-results')!.addEventListener('focusin', event => {
                    (window as any).firstTabRelatedIsNull = (event as FocusEvent).relatedTarget === null;
                }, { once: true });
            });
            await page.keyboard.press('Tab');
            expect(await page.evaluate(() => (window as any).firstTabRelatedIsNull)).toBe(true);
            // Firefox includes a scrollable region itself in its native Tab
            // sequence before its links; continue through that real stop.
            if (await page.locator('.trail-results').evaluate(node => node === document.activeElement)) await page.keyboard.press('Tab');
            await expect(page.locator('.trail-result a').first()).toBeFocused();
            await expect.poll(() => page.evaluate(() => {
                const list = document.querySelector('.trail-results')!.getBoundingClientRect();
                const target = document.activeElement!.getBoundingClientRect();
                return target.top >= list.top && target.bottom <= list.bottom;
            })).toBe(true);
        });
    });
}
