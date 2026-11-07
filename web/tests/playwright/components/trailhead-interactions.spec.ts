import { devices, expect, test, type Page } from '@playwright/test';
import { buildMapPageFixture } from './map-page-fixture';
import { mountEngine } from './map-engine-fixture';
import { loadFontawesome } from './fontawesome-fixture';
import { encodePolyline } from '../../../src/lib/util/polyline_util';

const line = [[7.92, 46.98], [8.08, 46.98]];
const polyline = encodePolyline(line.map(([lon, lat]) => [lat, lon]), 5);
async function open(page: Page, bundle: string) {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.addInitScript(() => { (window as any).fixtureSettings = { behavior: { showTrailStartMarker: true } }; });
    await mountEngine(page, bundle); await loadFontawesome(page);
    await expect.poll(() => page.evaluate(() => (window as any).realMap.loaded())).toBe(true);
}
const trail = (id: string, icon = 'fa-hiking') => ({
    id, name: `Trail ${id}`, author: 'tester', lat: 47.04, lon: 8, photos: [], polyline,
    distance: 2000, duration: 3600, elevation_gain: 200, elevation_loss: 200,
    expand: { category: { icon }, author: { id: 'tester', username: 'tester', actor: 'tester' } },
});
async function publish(page: Page, id: string, icon = 'fa-hiking') {
    await page.evaluate(({ id, trail }) => {
        const fixture = (window as any).fixture;
        fixture.autoResolve = false;
        fixture.apply({ q: id });
        const requests = fixture.requests.filter((r: any) => r.body.filter.q === id);
        for (const request of requests) request.resolve(request.channel === 'map'
            ? { mapTrails: [trail], clusters: { type: 'FeatureCollection', features: [{ type: 'Feature', properties: { id, point_count: 1, is_large: false, category_icon: trail.expand.category.icon }, geometry: { type: 'Point', coordinates: [8, 47.04] } }] } }
            : { items: [trail], page: 1, totalPages: 1, totalHits: 1 });
    }, { id, trail: trail(id, icon) });
}
async function ready(page: Page, id: string) {
    await expect.poll(() => page.evaluate(async id => {
        const map = (window as any).realMap;
        const clusters = await map.getSource('cluster-trails')?.getData();
        const preview = await map.getSource('preview')?.getData();
        return { cluster: clusters?.features[0]?.properties.id, preview: preview?.features[0]?.properties.trail,
            head: map.queryRenderedFeatures(map.project([8, 47.04]), { layers: ['unclustered-point'] }).length > 0,
            start: map.queryRenderedFeatures(map.project([7.92, 46.98]), { layers: ['preview-start-points'] }).length > 0,
            filter: map.getFilter('preview-start-points') };
    }, id)).toMatchObject({ cluster: id, preview: id, head: true, start: true });
}
// Pass the coordinate explicitly into browser functions (no closure is available there).
const lineStart = line[0];
async function point(page: Page, coordinate: number[]) {
    return page.evaluate(coordinate => {
        const map = (window as any).realMap, p = map.project(coordinate), box = map.getCanvas().getBoundingClientRect();
        return { x: box.x + p.x, y: box.y + p.y };
    }, coordinate);
}
for (const dev of [true, false]) test.describe(`trailhead interactions (${dev ? 'development' : 'production'})`, () => {
    let bundle: string;
    test.beforeAll(async () => { bundle = await buildMapPageFixture(dev, true, true); });
    test('touch opens trailhead, line and start-point popups without zoom', async ({ browser }) => {
        const context = await browser.newContext({ ...devices['Pixel 7'], viewport: { width: 390, height: 844 } });
        const touch = await context.newPage();
        try {
            await open(touch, bundle); await publish(touch, 'touch'); await ready(touch, 'touch');
            expect(await touch.evaluate(() => matchMedia('(hover: none)').matches)).toBe(true);
            const before = await touch.evaluate(() => (window as any).realMap.getZoom());
            for (const coordinate of [[8, 47.04], [8, 46.98], lineStart]) {
                await test.step(`touch popup at ${coordinate}`, async () => {
                const p = await point(touch, coordinate); await touch.touchscreen.tap(p.x, p.y);
                await expect(touch.locator('.maplibregl-popup'), `tap at ${coordinate}`).toContainText('Trail touch');
                expect(await touch.evaluate(() => (window as any).realMap.getZoom())).toBe(before);
                expect(await touch.evaluate(() => (window as any).realMap.isMoving())).toBe(false);
                await touch.locator('.maplibregl-popup-close-button').tap();
                });
            }
        } finally { await context.close(); }
    });
    test('mouse hover opens the popup and clicking a trailhead still zooms to 12', async ({ page }) => {
        await open(page, bundle); await publish(page, 'mouse'); await ready(page, 'mouse');
        const p = await point(page, [8, 47.04]); await page.mouse.move(p.x, p.y);
        await expect(page.locator('.maplibregl-popup')).toContainText('Trail mouse');
        await page.mouse.click(p.x, p.y);
        await expect.poll(() => page.evaluate(() => (window as any).realMap.getZoom())).toBe(12);
    });
    test('re-entering and following a thin preview line preserves its popup and opening position', async ({ page }) => {
        await open(page, bundle); await publish(page, 'line'); await ready(page, 'line');
        const p = await point(page, [7.925, 46.98]); await page.mouse.move(p.x, p.y);
        await expect(page.locator('.maplibregl-popup')).toContainText('Trail line');
        const popup = await page.locator('.maplibregl-popup').elementHandle();
        const position = await popup!.evaluate(node => node.style.transform);
        await page.mouse.move(p.x + 25, p.y + 9); await page.mouse.move(p.x + 35, p.y);
        const next = await point(page, [8.075, 46.98]);
        expect(next.x - p.x).toBeGreaterThan(60);
        expect(await page.evaluate(p => (window as any).realMap.queryRenderedFeatures(p, { layers: ['preview'] }).length, { x: next.x - (await page.locator('canvas').first().boundingBox())!.x, y: next.y - (await page.locator('canvas').first().boundingBox())!.y })).toBeGreaterThan(0);
        await page.mouse.move(next.x, next.y);
        expect(await popup!.evaluate(node => node.isConnected)).toBe(true);
        expect(await popup!.evaluate(node => node.style.transform)).toBe(position);
        await page.mouse.move(next.x, next.y + 100);
        await expect(page.locator('.maplibregl-popup')).toHaveCount(0);
        const start = await point(page, lineStart); await page.mouse.move(start.x, start.y);
        await expect(page.locator('.maplibregl-popup')).toContainText('Trail line');
    });
    test('late font work cannot replace newer cluster and preview symbols', async ({ page }) => {
        await open(page, bundle);
        await page.evaluate(() => {
            const load = document.fonts.load.bind(document.fonts);
            (window as any).fontReleases = [];
            document.fonts.load = (font, text) => text === '\uf6ec'
                ? new Promise(resolve => (window as any).fontReleases.push(() => load(font, text).catch(() => []).then(resolve))) : load(font, text);
        });
        await publish(page, 'old');
        await expect.poll(() => page.evaluate(() => (window as any).fontReleases.length)).toBeGreaterThanOrEqual(2);
        await publish(page, 'new', 'fa-bicycle'); await ready(page, 'new');
        await page.evaluate(async () => { await Promise.all((window as any).fontReleases.map((release: () => Promise<void>) => release())); await new Promise(requestAnimationFrame); });
        await ready(page, 'new');
        expect(await page.evaluate(async () => {
            const map = (window as any).realMap;
            return { head: map.getLayer('unclustered-point').type, start: map.getLayer('preview-start-points').type,
                image: map.hasImage('trailhead-fa-bicycle'), marker: (await map.getSource('preview')?.getData())?.features[0]?.properties.marker_icon };
        })).toEqual({ head: 'symbol', start: 'symbol', image: true, marker: 'trailhead-fa-bicycle' });
    });
});
