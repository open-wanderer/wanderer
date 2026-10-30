import { expect, test as base, type Page } from '@playwright/test';
import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

// New headless Chromium exposes native window state and display-mode.
// Fixture defaults belong to extend(), so importing shared helpers cannot
// register test.use() options in an unrelated spec or a reused worker.
export const windowTest = base.extend({
    channel: 'chromium',
    launchOptions: {args:['--window-size=1280,800','--screen-info={1600x1000}']},
    viewport: null,
    deviceScaleFactor: undefined,
    context: async ({browser}, use) => {
        const context = await browser.newContext({viewport:null});
        await use(context);
        await context.close();
    },
});
const dist = fileURLToPath(new URL('../../../node_modules/maplibre-gl/dist/', import.meta.url));
export const camera = (page: Page) => page.evaluate(() => {
    const map = (window as any).realMap;
    return { center: map.getCenter().toArray(), zoom: map.getZoom(), bearing: map.getBearing(), pitch: map.getPitch() };
});
export const bounds = (page: Page) => page.evaluate(() => {
    const b = (window as any).realMap.getBounds();
    return { northEast: { lat: b.getNorth(), lng: b.getEast() }, southWest: { lat: b.getSouth(), lng: b.getWest() } };
});
export const counts = (page: Page) => page.evaluate(() => ({
    results: (window as any).fixture.requests.filter((r: any) => r.channel === 'results').length,
    map: (window as any).fixture.requests.filter((r: any) => r.channel === 'map').length,
    publications: (window as any).fixture.publications.length,
    url: location.search,
}));
export async function settle(page: Page) {
    await expect(page.locator('.map-results')).toHaveAttribute('data-layout-ready', 'true');
    await expect.poll(() => page.evaluate(() => {
        const map = (window as any).realMap;
        const request = (window as any).fixture.requests.filter((r: any) => r.channel === 'map').at(-1)?.body;
        if (!map || map.isMoving() || !request) return false;
        const hidden = !map.getContainer().getClientRects().length;
        const published = (window as any).fixture.publications.at(-1)?.[0];
        if (hidden && published) return Math.abs(request.northEast.lat - published.northEast.lat) < 1e-8
            && Math.abs(request.southWest.lng - published.southWest.lng) < 1e-8;
        const b = map.getBounds();
        return Math.abs(request.northEast.lat - b.getNorth()) < 1e-8 && Math.abs(request.northEast.lng - b.getEast()) < 1e-8
            && Math.abs(request.southWest.lat - b.getSouth()) < 1e-8 && Math.abs(request.southWest.lng - b.getWest()) < 1e-8;
    })).toBe(true);
}
export async function windowState(page: Page, fullscreen: boolean) {
    const session = await page.context().newCDPSession(page);
    const { windowId } = await session.send('Browser.getWindowForTarget');
    const previous = await page.evaluate(() => [innerWidth, innerHeight]);
    await session.send('Browser.setWindowBounds', { windowId, bounds: { windowState: fullscreen ? 'fullscreen' : 'normal' } });
    await expect.poll(() => page.evaluate(() => [innerWidth, innerHeight])).not.toEqual(previous);
    await expect(page.locator('.map-page')).not.toHaveClass(/fullscreen/);
    await settle(page);
    await session.detach();
}
export async function mount(page: Page, bundle: string, delayed = false, push = false) {
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
        await settle(page);
    }
}


export async function recordInertiaTarget(page: Page) {
    await page.evaluate(() => {
        const w = window as any, map = w.realMap, ease = map.easeTo.bind(map);
        map.easeTo = (options: any, data: any) => {
            // Compute this exact gesture's uninterrupted endpoint in a second
            // real engine. A separately timed swipe has a different velocity.
            const container = document.createElement('div');
            container.style.cssText = `position:fixed;left:-10000px;width:${map.getContainer().clientWidth}px;height:${map.getContainer().clientHeight}px`;
            document.body.append(container);
            const control = new w.RealMapLibre.Map({container,interactive:false,
                center:map.getCenter(),zoom:map.getZoom(),bearing:map.getBearing(),pitch:map.getPitch(),
                style:{version:8,sources:{},layers:[]}});
            control.easeTo({...options,duration:0});
            w.inertiaTarget = {center:control.getCenter().toArray(),zoom:control.getZoom()};
            control.remove();container.remove();map.easeTo=ease;
            return ease(options,data);
        };
    });
}

export async function clockedSwipe(page: Page) {
    await page.evaluate(() => {
        const map = (window as any).realMap;
        map.dragPan.disable();
        map.dragPan.enable({ deceleration: 500, maxSpeed: 1000 });
        const canvas = map.getCanvas(), box = canvas.getBoundingClientRect();
        (window as any).regressionSwipe = { x: box.x + box.width / 2, y: box.y + box.height / 2 };
        const { x, y } = (window as any).regressionSwipe;
        canvas.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, button: 0, buttons: 1, clientX: x, clientY: y }));
    });
    for (let step = 1; step <= 5; step++) {
        await page.evaluate(step => {
            const { x, y } = (window as any).regressionSwipe;
            document.dispatchEvent(new MouseEvent('mousemove', { bubbles: true, buttons: 1, clientX: x + step * 36, clientY: y + step * 8 }));
        }, step);
        await page.clock.runFor(16);
    }
    await page.evaluate(() => {
        const { x, y } = (window as any).regressionSwipe;
        (window as any).realMap.getCanvas().dispatchEvent(new MouseEvent('mouseup', { bubbles: true, button: 0, clientX: x + 180, clientY: y + 40 }));
    });
    await page.clock.runFor(16);
}

export async function swipe(page: Page, deceleration = 100, afterRelease = false) {
    await page.evaluate(async ({ deceleration, afterRelease }) => {
        const map = (window as any).realMap;
        map.dragPan.disable();
        map.dragPan.enable({ deceleration, maxSpeed: 1000 });
        const canvas = map.getCanvas(), r = canvas.getBoundingClientRect();
        const x = r.x + r.width / 2, y = r.y + r.height / 2;
        canvas.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, button: 0, buttons: 1, clientX: x, clientY: y }));
        for (let step = 1; step <= 5; step++) {
            document.dispatchEvent(new MouseEvent('mousemove', { bubbles: true, buttons: 1, clientX: x + step * 36, clientY: y + step * 8 }));
            if (step < 5) await new Promise(requestAnimationFrame);
        }
        // Release in the renderer, without a Node round trip that can outlive
        // MapLibre's inertia buffer under CPU load.
        canvas.dispatchEvent(new MouseEvent('mouseup', { bubbles: true, button: 0, clientX: x + 180, clientY: y + 40 }));
        await new Promise(requestAnimationFrame);
        if (afterRelease) (window as any).fixture.afterFling?.();
    }, { deceleration, afterRelease });
}

// Exact counts are observed after the entire publication/render debounce chain,
// not at the first transient +1. This also catches a delayed second request.
export async function quietCounts(page: Page) {
    await settle(page);
    await page.waitForTimeout(650);
    return counts(page);
}
