import { expect, test, type Page } from '@playwright/test';
import { buildMapPageFixture } from './map-page-fixture';
import { mountEngine, counts, waitForEngine } from './map-engine-fixture';

async function searchedArea(page: Page) {
    const actual = await page.evaluate(() => {
        const w = window as any, b = w.realMap.getBounds();
        return { bounds: { northEast: { lat: b.getNorth(), lng: b.getEast() }, southWest: { lat: b.getSouth(), lng: b.getWest() } },
            searched: w.fixture.requests.filter((r: any) => r.channel === 'results').at(-1).body.bounds,
            width: w.realMap.getCanvas().clientWidth, container: w.realMap.getContainer().clientWidth };
    });
    expect(actual.width).toBe(actual.container);
    for (const corner of ['northEast', 'southWest'] as const) for (const axis of ['lat', 'lng'] as const) expect(actual.searched[corner][axis]).toBeCloseTo(actual.bounds[corner][axis], 10);
    expect(Number(new URL(page.url()).searchParams.get('tl_lon'))).toBeCloseTo(actual.bounds.northEast.lng, 7);
}

for (const dev of [true, false]) test.describe(`automatic map sizing (${dev ? 'development' : 'production'})`, () => {
    let bundle: string;
    test.beforeAll(async () => { bundle = await buildMapPageFixture(dev, true, true); });

    test('splitter mouse and keyboard changes resize the map and debounce one visible-area search', async ({ page }) => {
        await mountEngine(page, bundle);
        const handle = page.locator('.list-resize');
        const width = Number(await handle.getAttribute('aria-valuenow'));
        const before = await counts(page), box = (await handle.boundingBox())!;
        await page.mouse.move(box.x + box.width / 2, box.y + 40);
        await page.mouse.down();
        await page.mouse.move(box.x + box.width / 2 + 80, box.y + 40, { steps: 5 });
        await page.mouse.up();
        await expect(handle).toHaveAttribute('aria-valuenow', String(width + 80));
        await waitForEngine(page);
        await page.waitForTimeout(250); // No second request after the debounce.
        expect((await counts(page)).results).toBe(before.results + 1);
        await searchedArea(page);
        const keyboard = await counts(page);
        await handle.press('ArrowRight');
        await expect(handle).toHaveAttribute('aria-valuenow', String(width + 96));
        await waitForEngine(page);
        expect((await counts(page)).results).toBe(keyboard.results + 1);
        await searchedArea(page);
    });

    test('geolocation tracking remains active after a viewport change', async ({ page, context }) => {
        await context.grantPermissions(['geolocation']);
        await context.setGeolocation({ latitude: 47, longitude: 8 });
        await mountEngine(page, bundle);
        await page.locator('.maplibregl-ctrl-geolocate').click();
        const tracking = page.locator('.maplibregl-ctrl-geolocate-active');
        await expect(tracking).toBeVisible();
        await waitForEngine(page);
        await page.setViewportSize({ width: 1000, height: 760 });
        await waitForEngine(page);
        await expect(tracking).toBeVisible();
        await context.setGeolocation({ latitude: 47.01, longitude: 8.02 });
        await expect.poll(() => page.evaluate(() => (window as any).realMap.getCenter().lng)).toBeCloseTo(8.02, 5);
        await expect(tracking).toBeVisible();
    });

    test('a modal viewport resize refreshes only map data and keeps URL and selection', async ({ page }) => {
        await mountEngine(page, bundle);
        await page.locator('.trail-result').first().hover();
        await page.locator('.trail-result input[type=checkbox]').first().check();
        await page.evaluate(() => { const dialog = document.createElement('dialog'); document.body.append(dialog); dialog.showModal(); });
        await waitForEngine(page);
        const before = await counts(page);
        await page.setViewportSize({ width: 1000, height: 760 });
        await waitForEngine(page);
        await page.waitForTimeout(250);
        expect(await counts(page)).toEqual({ results: before.results, map: before.map + 1, url: before.url });
        await expect(page.locator('.selection-count')).toHaveText('1 selected');
    });

    test('a mobile list becomes a live map on widening and searches after panning', async ({ page }) => {
        await page.setViewportSize({ width: 600, height: 760 });
        await mountEngine(page, bundle);
        await page.locator('.map-view-toggle').click();
        await expect(page.locator('#trail-map')).toBeHidden();
        await page.setViewportSize({ width: 1280, height: 760 });
        await expect(page.locator('.maplibregl-canvas')).toBeVisible();
        await waitForEngine(page);
        const before = await counts(page);
        await page.evaluate(() => (window as any).realMap.panBy([80, 0], { duration: 0 }));
        await waitForEngine(page);
        expect((await counts(page)).results).toBe(before.results + 1);
        expect((await counts(page)).url).not.toBe(before.url);
        await searchedArea(page);
    });
});
