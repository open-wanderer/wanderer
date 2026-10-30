import { expect, type Page } from '@playwright/test';
import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
const dist = fileURLToPath(new URL('../../../node_modules/maplibre-gl/dist/', import.meta.url));
export const counts = (page: Page) => page.evaluate(() => ({
    results: (window as any).fixture.requests.filter((r: any) => r.channel === 'results').length,
    map: (window as any).fixture.requests.filter((r: any) => r.channel === 'map').length,
    url: location.search,
}));
export async function waitForEngine(page: Page) {
    await expect.poll(() => page.evaluate(() => {
        const map = (window as any).realMap;
        const request = (window as any).fixture.requests.filter((r: any) => r.channel === 'map').at(-1)?.body;
        if (!map || map.isMoving() || !request) return false;
        if (!map.getContainer().getClientRects().length) return true;
        const b = map.getBounds();
        return Math.abs(request.northEast.lat - b.getNorth()) < 1e-8 && Math.abs(request.northEast.lng - b.getEast()) < 1e-8
            && Math.abs(request.southWest.lat - b.getSouth()) < 1e-8 && Math.abs(request.southWest.lng - b.getWest()) < 1e-8;
    })).toBe(true);
}
export async function mountEngine(page: Page, bundle: string, delayed = false, push = false) {
    page.on('pageerror', error => { throw error; });
    await page.route('https://search-page.test/**', async route => {
        const path = new URL(route.request().url()).pathname;
        if (path.startsWith('/maplibre/')) {
            const name = path.split('/').at(-1)!;
            return route.fulfill({ contentType: name.endsWith('.css') ? 'text/css' : 'text/javascript', body: await readFile(dist + name, 'utf8') });
        }
        if (path.startsWith('/styles/')) return route.fulfill({ json: { version: 8, sources: {}, layers: [{ id: 'background', type: 'background', paint: { 'background-color': '#dae5df' } }] } });
        return route.fulfill({ contentType: 'text/html', body: '<html><head><meta name="viewport" content="width=device-width, initial-scale=1"><link rel="stylesheet" href="/maplibre/maplibre-gl.css"></head><body><script type="module">import * as M from "/maplibre/maplibre-gl.mjs"; window.RealMapLibre = M;</script></body></html>' });
    });
    await page.route(url => url.hostname !== 'search-page.test', route => route.abort());
    await page.goto('https://search-page.test/map?tl_lat=47.1&tl_lon=8.1&br_lat=46.9&br_lon=7.9');
    await expect.poll(() => page.evaluate(() => Boolean((window as any).RealMapLibre))).toBe(true);
    if (push) await page.evaluate(() => localStorage.setItem('searchLayout', JSON.stringify({ position: 'top', expansion: 'push' })));
    if (delayed) await page.evaluate(() => (window as any).deferMapLoad = true);
    await page.addScriptTag({ content: bundle });
    if (delayed) await expect.poll(() => page.evaluate(() => Boolean((window as any).finishMapLoad))).toBe(true);
    else {
        await expect.poll(() => page.evaluate(() => (window as any).fixture.publications.length)).toBeGreaterThan(0);
        await page.evaluate(() => {
            (window as any).fixture.autoResolve = true;
            for (const request of (window as any).fixture.requests) request.resolve(request.channel === 'map'
                ? { mapTrails: [], clusters: { type: 'FeatureCollection', features: [] } }
                : { items: [{ id: 'fixture-1', author: 'tester', name: 'Fixture trail' }], page: 1, totalPages: 1, totalHits: 1 });
        });
        await waitForEngine(page);
    }
}

