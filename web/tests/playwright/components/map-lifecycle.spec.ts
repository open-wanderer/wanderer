import { expect, devices, type Page } from '@playwright/test';
import { mkdtemp, mkdir, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { windowTest as test, mount, counts, camera, bounds, settle, quietCounts, swipe } from './map-window-fixture';
import { buildMapPageFixture } from './map-page-fixture';

async function size(page: Page, width: number, height = 800) {
    const session = await page.context().newCDPSession(page);
    const { windowId } = await session.send('Browser.getWindowForTarget');
    await session.send('Browser.setWindowBounds', { windowId, bounds: { width, height } });
    await expect.poll(() => page.evaluate(() => innerWidth)).toBe(width);
    await session.detach();
}
async function changed(page: Page, before: Awaited<ReturnType<typeof counts>>) {
    expect(await quietCounts(page)).toMatchObject({ results: before.results + 1, map: before.map + 1, publications: before.publications + 1 });
    expect((await counts(page)).url).not.toBe(before.url);
}
async function sampleCanvas(page: Page) {
    await page.evaluate(() => {
        const w = window as any, map = w.realMap;
        const marker = document.createElement('div');
        marker.style.cssText = 'position:fixed;left:0;top:0;height:1px;pointer-events:none;opacity:0';
        document.body.append(marker);
        const samples: number[] = [];
        let running = true, flip = false;
        const observer = new ResizeObserver(() => {
            const c = map.getCanvas().getBoundingClientRect(), r = map.getContainer().getBoundingClientRect();
            if (running && r.width && r.height) samples.push(Math.max(Math.abs(c.width - r.width), Math.abs(c.height - r.height)));
        });
        observer.observe(marker);
        const frame = () => { if (running) { marker.style.width = (flip = !flip) ? '1px' : '2px'; requestAnimationFrame(frame); } };
        requestAnimationFrame(frame);
        w.stopCanvasSamples = () => { running = false; observer.disconnect(); marker.remove(); return samples; };
    });
}
async function canvasMatches(page: Page) {
    const samples = await page.evaluate(() => (window as any).stopCanvasSamples());
    expect(samples.length).toBeGreaterThan(1);
    expect(Math.max(...samples), 'canvas must fill the container on every painted frame').toBeLessThanOrEqual(1);
}

for (const dev of [true, false]) test.describe(`map lifecycle (${dev ? 'development' : 'production'})`, () => {
    test.describe.configure({ timeout: 60000 });
    let bundle: string, search: string;
    test.beforeAll(async () => {
        bundle = await buildMapPageFixture(dev, true, true);
        search = await buildMapPageFixture(dev, true, true, false, true);
    });
    test('canvas follows single changes', async ({ page }) => {

        await mount(page, bundle);
        await sampleCanvas(page);
        for (const [width, height] of [[1500, 950], [1100, 760]]) { await size(page, width, height); await settle(page); }

        await quietCounts(page);
        await canvasMatches(page);
        // A real compositor screenshot also detects an unpainted drawing buffer.
        const screenshot = await page.locator('.maplibregl-canvas').screenshot();
        const pixel = await page.evaluate(async base64 => {
            const image = new Image(); image.src = 'data:image/png;base64,' + base64; await image.decode();
            const canvas = document.createElement('canvas'); canvas.width = image.width; canvas.height = image.height;
            const context = canvas.getContext('2d')!; context.drawImage(image, 0, 0);
            return [...context.getImageData(image.width - 12, image.height - 12, 1, 1).data].slice(0, 3);
        }, screenshot.toString('base64'));
        expect(pixel).toEqual([218, 229, 223]);
    });
    test('split handles and overview grip paint throughout camera motion', async ({ page }) => {
        await mount(page, bundle);
        for (const mode of ['split', 'overview', 'overlay-move']) {
            if (mode === 'overview') { await page.locator('.expand-results').click(); await settle(page); }
            const before = await counts(page);
            const handle = page.locator(mode === 'split' ? '.list-resize' : mode === 'overview' ? '.overlay-resize' : '.overlay-move');
            const target = mode === 'overlay-move' ? [(await camera(page)).center[0] + .03, 47.01] : [8.05, 47.02];
            const r = (await handle.boundingBox())!;
            await sampleCanvas(page);
            await page.mouse.move(r.x + r.width / 2, r.y + r.height / 2);
            // Begin motion immediately before the real resize handler in the
            // same pointer event, even when the test process is under load.
            await handle.evaluate((node, target) => node.addEventListener('pointerdown', () => {
                (window as any).realMap.easeTo({ center: target, duration: 450 });
            }, { capture: true, once: true }), target);
            await page.mouse.down();
            await page.mouse.move(r.x + r.width / 2 - 90, r.y + r.height / 2 - 50, { steps: 8 });
            await page.mouse.up();
            await changed(page, before);
            await canvasMatches(page);
            expect((await camera(page)).center[0]).toBeCloseTo(target[0], 7);
            expect((await camera(page)).center[1]).toBeCloseTo(target[1], 7);
            if (mode === 'overlay-move') expect((await bounds(page)).northEast.lat).toBeGreaterThan(47.01);
        }
    });
    test('push first drag retains its geographic pointer anchor', async ({ page }) => {
        await mount(page, search, false, true);
        await page.getByRole('searchbox').first().focus();
        await expect(page.locator('.search-panel')).toBeVisible();
        await page.evaluate(() => { const map = (window as any).realMap; map.dragPan.disable(); map.dragPan.enable({ maxSpeed: 0 }); });
        const r = (await page.locator('.maplibregl-canvas').boundingBox())!;
        const x = r.x + r.width / 2, y = r.y + r.height - 40;
        const anchor = await page.evaluate(({ x, y }) => {
            const map = (window as any).realMap, r = map.getCanvas().getBoundingClientRect();
            return map.unproject([x - r.x, y - r.y]).toArray();
        }, { x, y });
        await sampleCanvas(page);
        await page.mouse.move(x, y); await page.mouse.down();
        for (let step = 1; step <= 8; step++) {
            await page.mouse.move(x + step * 15, y);
            await page.evaluate(() => new Promise(requestAnimationFrame));
            const error = await page.evaluate(({ anchor, x, y }) => {
                const map = (window as any).realMap, r = map.getCanvas().getBoundingClientRect(), point = map.project(anchor);
                return Math.hypot(point.x + r.x - x, point.y + r.y - y);
            }, { anchor, x: x + step * 15, y });
            expect(error, 'grabbed geographic point remains under the pointer').toBeLessThanOrEqual(2);
        }
        await page.mouse.up(); await settle(page);
        await canvasMatches(page);
    });
    test('phone zoom and closing keyboard paint every frame and reach full zoom', async ({ browser }) => {
        const context = await browser.newContext({ ...devices['Pixel 7'] });
        const page = await context.newPage();
        try {
            await mount(page, bundle);
            await page.evaluate(() => {
                const state = { height: 600 };
                (window as any).keyboardViewport = state;
                Object.defineProperty(visualViewport, 'height', { get: () => state.height });
                visualViewport!.dispatchEvent(new Event('resize'));
            });
            await settle(page);
            await sampleCanvas(page);
            const before = (await camera(page)).zoom;
            await page.evaluate(() => {
                const map = (window as any).realMap;
                map.getContainer().querySelector('.maplibregl-ctrl-zoom-in').click();
                (window as any).keyboardViewport.height = 915;
                visualViewport!.dispatchEvent(new Event('resize'));
            });
            await quietCounts(page);
            await canvasMatches(page);
            expect((await camera(page)).zoom).toBeCloseTo(before + 1, 6);
        } finally { await context.close(); }
    });
    test('tracking survives window, header and phone rotation', async ({ page, context }) => {
        await context.grantPermissions(['geolocation']);
        await context.setGeolocation({ latitude: 47.3, longitude: 8.5 });
        await mount(page, bundle);
        await page.locator('.maplibregl-ctrl-geolocate').click();
        const active = page.locator('.maplibregl-ctrl-geolocate-active');
        await expect(active).toBeVisible();
        await expect.poll(() => page.evaluate(() => (window as any).realMap.isMoving())).toBe(false);
        await size(page, 1100);
        await page.locator('[data-search-root]').evaluate(node => (node as HTMLElement).style.height = '120px');
        await settle(page);
        const cdp = await context.newCDPSession(page);
        for (const [width, height] of [[412, 915], [915, 412]]) {
            await cdp.send('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: 1, mobile: false });
            await settle(page);
            await expect(active).toBeVisible();
        }
        await context.setGeolocation({ latitude: 47.05, longitude: 8.06 });
        await expect.poll(async () => (await camera(page)).center[0]).toBeCloseTo(8.06, 5);
        await expect(active).toBeVisible();
        await cdp.detach();
    });
    test('lost context-menu release, blur and removed handles never leave a layout locked', async ({ page }) => {
        await mount(page, bundle);
        const session = await page.context().newCDPSession(page);
        await page.evaluate(() => new (window as any).RealMapLibre.Popup().setLngLat((window as any).realMap.getCenter())
            .setHTML('<a id="context-trail" href="/map/trail/@tester/fixture-1">Fixture trail</a>').addTo((window as any).realMap));
        const link = (await page.locator('#context-trail').boundingBox())!;
        await session.send('Input.dispatchMouseEvent', { type: 'mousePressed', button: 'right', buttons: 2, clickCount: 1, x: link.x + 4, y: link.y + 4 });
        await session.send('Input.dispatchMouseEvent', { type: 'mouseMoved', button: 'none', buttons: 0, x: link.x - 30, y: link.y + 40 });
        const popup = await counts(page);
        await size(page, 1200);
        await changed(page, popup);
        const keyboard = await counts(page);
        await page.locator('.maplibregl-canvas').focus();
        await page.keyboard.press('ArrowRight');
        await changed(page, keyboard);
        for (const removal of ['blur', 'expand']) {
            await size(page, 1280);
            const h = (await page.locator('.list-resize').boundingBox())!;
            await page.mouse.move(h.x + h.width / 2, h.y + h.height / 2); await page.mouse.down();
            await page.mouse.move(h.x - 30, h.y + h.height / 2);
            if (removal === 'blur') {
                await page.evaluate(() => dispatchEvent(new Event('blur')));
                await expect(page.locator('.map-results')).toHaveAttribute('data-layout-ready', 'true');
            }

            else await page.locator('.expand-results').press('Enter');
            await page.mouse.up();

            await size(page, 1280);
            await settle(page);
            const before = await counts(page);
            await page.evaluate(() => (window as any).realMap.panBy([70, 0], { duration: 0 }));
            await changed(page, before);
            if (removal === 'expand') { await page.locator('.expand-results').click(); await settle(page); }
        }
        await session.detach();
    });
    test('hiding a moving map discards its unpublished snapshot camera', async ({browser}) => {
        const context = await browser.newContext({...devices['Pixel 7']});
        const page = await context.newPage();
        try {
            await mount(page, bundle);
            await page.evaluate(() => {
                const w = window as any;
                w.fixture.afterFling = () => {
                    w.fixture.movingWhenHidden = w.realMap.isMoving();
                    (document.querySelector('.map-view-toggle') as HTMLButtonElement).click();
                };
            });
            await swipe(page, 100, true);
            expect(await page.evaluate(() => (window as any).fixture.movingWhenHidden)).toBe(true);
            await expect(page.locator('#trail-map')).toBeHidden();
            await page.waitForTimeout(650);
            expect(await page.evaluate(() => (window as any).fixture.capture().camera)).toBeUndefined();
        } finally { await context.close(); }
    });
    test('fractional mobile boundary has no visible silent map', async ({ playwright }) => {
        const profile = await mkdtemp(join(tmpdir(), 'wanderer-fractional-zoom-'));
        await mkdir(join(profile, 'Default'));
        await writeFile(join(profile, 'Default', 'Preferences'), JSON.stringify({ partition: { default_zoom_level: { x: Math.log(1.25) / Math.log(1.2) } } }));
        const context = await playwright.chromium.launchPersistentContext(profile, { channel: 'chromium', viewport: null,
            args: ['--window-size=1280,800', '--screen-info={1600x1000}'] });
        const page = context.pages()[0];
        try {
            await mount(page, bundle);
            const session = await context.newCDPSession(page);
            const { windowId } = await session.send('Browser.getWindowForTarget');
            await session.send('Browser.setWindowBounds', { windowId, bounds: { width: 959, height: 1000 } });
            await expect.poll(() => page.evaluate(() => visualViewport!.width)).toBeLessThan(768);
            await expect(page.locator('.map-view-toggle')).toBeVisible();
            const width = await page.evaluate(() => visualViewport!.width);
            expect(width).toBeGreaterThan(767);
            expect(width).toBeLessThan(768);
            await page.locator('.map-view-toggle').click();
            await expect(page.locator('#trail-map')).toBeHidden();
            await expect(page.locator('.map-view-toggle')).toBeVisible();
            await page.locator('.map-view-toggle').click();
            await settle(page);
            const before = await counts(page);
            await page.evaluate(() => (window as any).realMap.panBy([60, 0], { duration: 0 }));
            await changed(page, before);
            await session.detach();
        } finally { await context.close(); await rm(profile, { recursive: true, force: true }); }
    });
});
