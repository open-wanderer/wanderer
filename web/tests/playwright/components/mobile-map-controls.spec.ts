import { expect, test, type Page } from '@playwright/test';
import { buildMapPageFixture } from './map-page-fixture';
import { mountEngine, waitForEngine } from './map-engine-fixture';
import { loadFontawesome } from './fontawesome-fixture';

const messages = { 'trail-map-show-results': 'Zu den Ergebnissen wechseln', 'trail-map-show-map': 'Zur Karte wechseln' };
async function openMap(page: Page, bundle: string, theme: string, push = false) {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.addInitScript(messages => { (window as any).fixtureMessages = messages; }, messages);
    await mountEngine(page, bundle, false, push);
    await loadFontawesome(page);
    await page.evaluate(theme => {
        for (const node of [document.documentElement, document.body]) { node.classList.remove('light', 'dark'); node.classList.add(theme); }
        const map = (window as any).realMap;
        map.addSource('credits', { type: 'geojson', attribution: 'Local map contributors · Test map attribution with readable text', data: { type: 'FeatureCollection', features: [] } });
        map.addLayer({ id: 'credits', type: 'circle', source: 'credits' });
    }, theme);
    const attribution = page.locator('.maplibregl-ctrl-attrib');
    await expect(attribution).toBeVisible();
    if (await attribution.getAttribute('open') !== null) await page.locator('.maplibregl-ctrl-attrib-button').click();
    await expect(attribution).not.toHaveAttribute('open');
}
async function attributionRow(page: Page) {
    const scale = page.locator('.maplibregl-ctrl-scale');
    const attribution = page.locator('.maplibregl-ctrl-attrib');
    await expect(scale).toBeVisible(); await expect(attribution).toBeVisible();
    const a = (await attribution.boundingBox())!, s = (await scale.boundingBox())!, map = (await page.locator('#trail-map').boundingBox())!;
    expect(a.x + a.width).toBeLessThanOrEqual(s.x);
    expect(Math.abs(a.y + a.height - s.y - s.height)).toBeLessThanOrEqual(1);
    expect(s.x + s.width).toBeGreaterThan(map.x + map.width - 16);
    expect(s.y + s.height).toBeGreaterThan(map.y + map.height - 16);
    expect(a.x).toBeGreaterThanOrEqual(map.x);
    expect(a.width).toBeGreaterThan(20);
}

for (const dev of [true, false]) test.describe(`mobile map controls (${dev ? 'development' : 'production'})`, () => {
    let bundle: string;
    test.beforeAll(async () => { bundle = await buildMapPageFixture(dev, true, true, false, true, true); });
    for (const theme of ['light', 'dark']) {
        test(`${theme}: scale and attribution share the bottom edge and move back at 768px`, async ({ page }) => {
            await openMap(page, bundle, theme);
            await expect(page.locator('.maplibregl-ctrl-bottom-right .maplibregl-ctrl-scale')).toHaveCount(1);
            await attributionRow(page);
            await page.locator('.maplibregl-ctrl-attrib-button').click();
            await expect(page.locator('.maplibregl-ctrl-attrib')).toHaveAttribute('open', '');
            await attributionRow(page);
            await expect(page.locator('.maplibregl-ctrl-attrib-inner')).toContainText('Local map contributors');
            expect(await page.locator('.maplibregl-ctrl-attrib-inner').evaluate(node => node.scrollWidth <= node.clientWidth)).toBe(true);
            await page.setViewportSize({ width: 768, height: 844 });
            await expect(page.locator('.maplibregl-ctrl-top-left .maplibregl-ctrl-scale')).toHaveCount(1);
            await page.setViewportSize({ width: 390, height: 844 });
            await expect(page.locator('.maplibregl-ctrl-bottom-right .maplibregl-ctrl-scale')).toHaveCount(1);
        });
        test(`${theme}: toolbar keeps an icon-only toggle in one place with focus, order and room for search`, async ({ page }) => {
            await openMap(page, bundle, theme);
            const toggle = page.locator('.map-view-toggle');
            const sort = page.locator('.map-search-toolbar button[aria-controls$="-sort"]');
            const view = page.locator('.map-search-toolbar button[aria-controls$="-view"]');
            await expect(sort).toBeHidden(); await expect(view).toBeHidden();
            await expect(toggle).toHaveAccessibleName(messages['trail-map-show-results']);
            await expect(toggle).toHaveAttribute('title', messages['trail-map-show-results']);
            await expect(toggle).not.toHaveAttribute('aria-pressed');
            await expect(toggle.locator('i')).toHaveClass(/fa-list-ul/);
            expect(await toggle.textContent()).toBe('');
            expect(await page.locator('#trail-map .map-view-toggle, .map-view-label').count()).toBe(0);
            const before = (await toggle.boundingBox())!, identity = await toggle.elementHandle();
            const mapInput = (await page.locator('[data-search-root] input[type="search"]').boundingBox())!;
            await toolbarOrder(page, false);
            await toggle.press('Enter');
            await expect(page.locator('.trail-results')).toBeVisible();
            await expect(toggle).toBeFocused();
            await expect(toggle).toHaveAccessibleName(messages['trail-map-show-map']);
            await expect(toggle).toHaveAttribute('title', messages['trail-map-show-map']);
            await expect(toggle.locator('i')).toHaveClass(/fa-map/);
            await expect(sort).toBeVisible(); await expect(view).toBeVisible();
            await toolbarOrder(page, true);
            await expect.poll(() => toggle.boundingBox()).toMatchObject(before);
            expect(await identity!.evaluate(node => node === document.querySelector('.map-view-toggle'))).toBe(true);
            expect((await page.locator('[data-search-root] input[type="search"]').boundingBox())!.width).toBeLessThan(mapInput.width);
            for (const button of [sort, view]) {
                expect((await button.boundingBox())!.width).toBeCloseTo(before.width, 0);
                expect((await button.boundingBox())!.height).toBeCloseTo(before.height, 0);
                expect(await button.evaluate(node => getComputedStyle(node).backgroundColor)).toBe(await toggle.evaluate(node => getComputedStyle(node).backgroundColor));
            }
            await toggle.press('Space'); await waitForEngine(page);
            await expect(toggle).toBeFocused(); await expect(sort).toBeHidden();
            await expect.poll(() => toggle.boundingBox()).toMatchObject(before);
            await page.setViewportSize({ width: 768, height: 844 });
            await expect(toggle).toBeHidden(); await expect(sort).toBeVisible(); await expect(view).toBeVisible();
            await page.setViewportSize({ width: 390, height: 844 });
            await expect(toggle).toBeVisible(); await expect(sort).toBeHidden();
        });
        test(`${theme}: push preference overlays map filters and follows both open-panel transitions`, async ({ page }) => {
            await openMap(page, bundle, theme, true);
            const toggle = page.locator('.map-view-toggle');
            const filter = page.locator('.search-heading button[aria-expanded]');
            const panel = page.locator('.search-panel');
            const before = (await page.locator('#trail-map').boundingBox())!;
            await filter.click(); await expect(panel).toBeVisible();
            await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject(before);
            await expect(page.locator('[data-search-root]')).not.toHaveClass(/in-flow/);
            // A click dispatched without moving focus exercises changing layout while still expanded.
            await toggle.evaluate(node => (node as HTMLButtonElement).click());
            await expect(panel).toBeVisible();
            await expect(page.locator('[data-search-root]')).toHaveClass(/in-flow/);
            const pushed = (await page.locator('.trail-list').boundingBox())!;
            expect(pushed.y).toBeGreaterThan(before.y + 100);
            await toggle.evaluate(node => (node as HTMLButtonElement).click());
            await expect(panel).toBeVisible();
            await expect(page.locator('[data-search-root]')).not.toHaveClass(/in-flow/);
            await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject(before);
            await filter.click(); await expect(panel).toHaveCount(0);
            await toggle.click(); await filter.click();
            await expect(panel).toBeVisible();
            expect((await page.locator('.trail-list').boundingBox())!.y).toBeGreaterThan(before.y + 100);
            // A real pointer switch dismisses the filter using its existing outside-click behavior.
            await toggle.click(); await expect(panel).toHaveCount(0);
            await expect(page.locator('#trail-map')).toBeVisible();
            await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject(before);
        });
    }
});

async function toolbarOrder(page: Page, list: boolean) {
    const input = page.locator('[data-search-root] input[type="search"]');
    const buttons = [page.locator('.search-heading button[aria-expanded]'),
        ...(list ? [page.locator('.map-search-toolbar button[aria-controls$="-sort"]'), page.locator('.map-search-toolbar button[aria-controls$="-view"]')] : []),
        page.locator('.map-view-toggle')];
    let previous = (await input.boundingBox())!;
    for (const button of buttons) {
        const box = (await button.boundingBox())!;
        expect(box.x).toBeGreaterThanOrEqual(previous.x + previous.width - 1);
        previous = box;
    }
    const toolbar = (await page.locator('.map-search-toolbar').boundingBox())!;
    expect(previous.x + previous.width).toBeLessThanOrEqual(toolbar.x + toolbar.width);
}
