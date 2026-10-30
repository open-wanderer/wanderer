import { expect, devices, type Page } from '@playwright/test';
import { mkdtemp, mkdir, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { windowTest as test, mount, counts, camera, bounds, settle, quietCounts, windowState, swipe } from './map-window-fixture';
import { buildMapPageFixture } from './map-page-fixture';
import { installBrowserFullscreen, setBrowserFullscreen } from '../browser_fullscreen';

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
    let bundle: string, nav: string, search: string;
    test.beforeAll(async () => {
        bundle = await buildMapPageFixture(dev, true, true);
        nav = await buildMapPageFixture(dev, true, true, true);
        search = await buildMapPageFixture(dev, true, true, false, true);
    });
    test('canvas follows single changes, twenty drag steps and Firefox F11 series', async ({ page }) => {
        await installBrowserFullscreen(page);
        await mount(page, bundle);
        await sampleCanvas(page);
        for (const [width, height] of [[1500, 950], [1100, 760]]) { await size(page, width, height); await settle(page); }
        for (let step = 1; step <= 20; step++) await size(page, 1100 + step * 20, 760 + step * 8);
        await settle(page);
        await setBrowserFullscreen(page, true);
        for (let step = 1; step <= 44; step++) {
            await size(page, 1500 + Math.round(step * 2), 920 + Math.round(step * 1.5));
            await page.waitForTimeout(16);
        }
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
    test('held pointer does not defer F11 canvas or swallow the next pan', async ({ page }) => {
        await mount(page, bundle);
        const r = (await page.locator('.maplibregl-canvas').boundingBox())!;
        await page.mouse.move(r.x + 100, r.y + 100);
        await page.mouse.down();
        await sampleCanvas(page);
        await windowState(page, true);
        await page.mouse.up();
        await canvasMatches(page);
        for (const fullscreen of [true, false]) {
            if (!fullscreen) await windowState(page, false);
            const before = await counts(page);
            await page.evaluate(() => (window as any).realMap.panBy([90, 20], { duration: 0 }));
            await changed(page, before);
        }
    });
    test('split handles and overview grip paint throughout camera motion', async ({ page }) => {
        await mount(page, bundle);
        for (const mode of ['split', 'overview']) {
            if (mode === 'overview') { await page.locator('.expand-results').click(); await settle(page); }
            const before = await counts(page);
            const handle = page.locator(mode === 'split' ? '.list-resize' : '.overlay-resize');
            const r = (await handle.boundingBox())!;
            await sampleCanvas(page);
            await page.mouse.move(r.x + r.width / 2, r.y + r.height / 2);
            // Begin motion immediately before the real resize handler in the
            // same pointer event, even when the test process is under load.
            await handle.evaluate(node => node.addEventListener('pointerdown', () => {
                (window as any).realMap.easeTo({ center: [8.05, 47.02], duration: 450 });
            }, { capture: true, once: true }));
            await page.mouse.down();
            await page.mouse.move(r.x + r.width / 2 - 90, r.y + r.height / 2 - 50, { steps: 8 });
            await page.mouse.up();
            await changed(page, before);
            await canvasMatches(page);
            expect((await camera(page)).center[0]).toBeCloseTo(8.05, 7);
            expect((await camera(page)).center[1]).toBeCloseTo(47.02, 7);
        }
    });
    test('push first drag retains its geographic pointer anchor and wheel notches complete', async ({ page }) => {
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
        // Three physical wheel notches, including MapLibre's 40ms detection
        // interval, must retain their complete zoom during a resize series.
        const runWheel = async (resize: boolean) => {
            await page.evaluate(() => (window as any).realMap.jumpTo({ center: [8, 47], zoom: 10 }));
            await quietCounts(page);
            await page.evaluate(async resize => {
                const map = (window as any).realMap, canvas = map.getCanvas(), r = canvas.getBoundingClientRect();
                if (resize) dispatchEvent(new Event('resize'));
                for (let step = 0; step < 3; step++) {
                    canvas.dispatchEvent(new WheelEvent('wheel', { bubbles: true, cancelable: true, deltaY: -120, clientX: r.x + r.width / 2, clientY: r.y + r.height / 2 }));
                    if (resize) { (document.querySelector('[data-search-root]') as HTMLElement).style.height = (60 + step * 8) + 'px'; dispatchEvent(new Event('resize')); }
                    await new Promise(resolve => setTimeout(resolve, 20));
                }
            }, resize);
            await quietCounts(page);
            return (await camera(page)).zoom;
        };
        const control = await runWheel(false);
        expect(await runWheel(true)).toBeCloseTo(control, 5);
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
    test('tracking survives window, header, fullscreen and phone rotation', async ({ page, context }) => {
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
        await windowState(page, true);
        await windowState(page, false);
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
    test('deferred mouse, keyboard, collapse and expansion publish the final camera once', async ({ page }) => {
        await mount(page, bundle);
        for (const action of ['key', 'collapse', 'expand', 'move']) {
            if (action === 'move') { await page.locator('.expand-results').click(); await settle(page); }
            const before = await counts(page);
            await page.evaluate(action => {
                const map = (window as any).realMap;
                map.easeTo({ center: [map.getCenter().lng + .03, 47.01], duration: 350 });
                if (action === 'key') document.querySelector('.list-resize')!.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }));
                if (action === 'collapse') (document.querySelector('.collapse-results') as HTMLElement).click();
                if (action === 'expand') (document.querySelector('.expand-results') as HTMLElement).click();
            }, action);
            if (action === 'move') {
                const r = (await page.locator('.overlay-move').boundingBox())!;
                await page.mouse.move(r.x + 10, r.y + 10); await page.mouse.down();
                await page.mouse.move(r.x - 80, r.y - 30, { steps: 5 }); await page.mouse.up();
            }
            await changed(page, before);
            const current = await camera(page);
            expect(current.center[1]).toBeCloseTo(47.01, 7);
            expect((await bounds(page)).northEast.lat).toBeGreaterThan(47.01);
            if (action === 'collapse') { await page.locator('.collapse-results').click(); await settle(page); }
            if (action === 'expand') { await page.locator('.expand-results').click(); await settle(page); }
        }
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
        const f11 = await counts(page);
        await windowState(page, true);
        expect(await quietCounts(page)).toMatchObject({ results: f11.results, publications: f11.publications, map: f11.map + 1, url: f11.url });
        await windowState(page, false);
        const r = (await page.locator('.reset-map-view').boundingBox())!;
        await session.send('Input.dispatchMouseEvent', { type: 'mousePressed', button: 'right', buttons: 2, clickCount: 1, x: r.x + 10, y: r.y + 10 });
        await session.send('Input.dispatchMouseEvent', { type: 'mouseMoved', button: 'none', buttons: 0, x: r.x - 30, y: r.y + 40 });
        let before = await counts(page);
        await size(page, 1100);
        await changed(page, before);
        for (const removal of ['blur', 'fullscreen', 'narrow', 'expand', 'overlay-close']) {
            await size(page, 1280);
            if (removal === 'overlay-close') { await page.locator('.expand-results').click(); await settle(page); }
            const h = (await page.locator(removal === 'overlay-close' ? '.overlay-move' : '.list-resize').boundingBox())!;
            await page.mouse.move(h.x + h.width / 2, h.y + h.height / 2); await page.mouse.down();
            await page.mouse.move(h.x - 30, h.y + h.height / 2);
            if (removal === 'blur') {
                await page.evaluate(() => dispatchEvent(new Event('blur')));
                await expect(page.locator('.map-results')).toHaveAttribute('data-layout-ready', 'true');
            }
            else if (removal === 'fullscreen') await windowState(page, true);
            else if (removal === 'narrow') await size(page, 700);
            else if (removal === 'overlay-close') {
                await page.locator('.overlay-close').press('Enter');
                await expect(page.locator('.map-results')).toHaveAttribute('data-layout-ready', 'true');
                await page.locator('.overlay-reopen').press('Enter');
            } else await page.locator('.expand-results').press('Enter');
            await page.mouse.up();
            if (removal === 'fullscreen') await windowState(page, false);
            await size(page, 1280);
            await settle(page);
            before = await counts(page);
            await page.evaluate(() => (window as any).realMap.panBy([70, 0], { duration: 0 }));
            await changed(page, before);
            if (removal === 'expand' || removal === 'overlay-close') { await page.locator('.expand-results').click(); await settle(page); }
        }
        await session.detach();
    });
    for (const privacy of ['brave', 'rfp']) test(`${privacy} screen dimensions never override fullscreen`, async ({ page }) => {
        await page.addInitScript(privacy => {
            const buckets = [[800, 600], [1280, 720], [1366, 768], [1536, 864], [1920, 1080], [2560, 1440], [3840, 2160]];
            const metric = (axis: number) => privacy === 'rfp' ? [innerWidth, innerHeight][axis]
                : (buckets.find(([w, h]) => w >= outerWidth && h >= outerHeight) ?? buckets.at(-1)!)[axis];
            Object.defineProperties(screen, { width: { get: () => metric(0) }, height: { get: () => metric(1) } });
            if (privacy === 'rfp') Object.defineProperties(window, { outerWidth: { get: () => innerWidth }, outerHeight: { get: () => innerHeight } });
        }, privacy);
        await mount(page, bundle);
        for (const enabled of [true, false]) {
            const before = await counts(page);
            await windowState(page, enabled);
            expect(await quietCounts(page)).toMatchObject({ results: before.results, publications: before.publications, map: before.map + 1, url: before.url });
        }
    });
    for (const menu of ['Open user menu', 'Toggle notification dropdown']) test(`${menu} yields Escape to a later window field handler`, async ({page}) => {
        await mount(page, nav);
        const trigger = page.getByRole('button', {name:menu,exact:true});
        await trigger.click();
        const opened = page.locator('nav .dropdown ul');
        await expect(opened).toBeVisible();
        const input = page.getByTestId('fixture-search');
        await input.focus();
        // Svelte's window handler is registered after both menu handlers.
        await page.evaluate(() => { (window as any).fixture.lateWindowField = true; });
        await page.keyboard.press('Escape');
        await expect(input).toBeFocused();
        await expect(input).toHaveValue('field handled Escape');
        await expect(opened).toBeVisible();
        await trigger.focus();
        await trigger.press('Escape');
        await expect(opened).toHaveCount(0);
        await expect(trigger).toBeFocused();
    });
    test('normal inner-only resize still publishes exactly once', async ({ page }) => {
        await mount(page, bundle);
        const before = await counts(page), session = await page.context().newCDPSession(page);
        await session.send('Emulation.setDeviceMetricsOverride', { width: 1280, height: 650, deviceScaleFactor: 1, mobile: false });
        await changed(page, before);
        await session.detach();
    });
    test('each fullscreen transition owns its delayed first resize', async ({ page }) => {
        await installBrowserFullscreen(page);
        await mount(page, bundle);
        const before = await counts(page);
        await page.evaluate(() => {
            (window as any).setTestBrowserFullscreen(true, true, false);
            setTimeout(() => {
                (window as any).setTestBrowserFullscreen(false, true, false);
                setTimeout(() => dispatchEvent(new Event('resize')), 200);
            }, 50);
        });
        expect(await quietCounts(page)).toMatchObject({ results: before.results, publications: before.publications, url: before.url });
    });
    test('each fullscreen session owns its trigger and quick user focus wins closing menus', async ({ page }) => {
        await mount(page, nav);
        await page.locator('.trail-result a').first().focus();
        await windowState(page, true); await windowState(page, false);
        await page.evaluate(() => { (document.activeElement as HTMLElement).blur(); document.querySelector('.trail-results')!.scrollTop = 0; });
        await windowState(page, true); await windowState(page, false);
        await expect(page.locator('[data-testid="fixture-search"]')).toBeFocused();
        expect(await page.locator('.trail-results').evaluate(node => node.scrollTop)).toBe(0);
        for (const menu of ['Open user menu', 'Toggle notification dropdown']) {
            await page.getByRole('button', { name: menu, exact: true }).click();
            await windowState(page, true);
            await page.keyboard.press('Escape');
            await expect(page.locator('nav [data-fullscreen-overlay]')).toHaveCount(0);
            await windowState(page, false);
            await page.getByRole('button', { name: menu, exact: true }).click();
            await windowState(page, true);
            await page.locator('.maplibregl-canvas').click({ position: { x: 300, y: 300 }, delay: 0 });
            await page.evaluate(() => new Promise(requestAnimationFrame));
            await expect(page.locator('.maplibregl-canvas')).toBeFocused();
            const before = await counts(page);
            await page.keyboard.press('ArrowRight');
            await changed(page, before);
            await windowState(page, false);
        }
    });
    test('a touch choice on the map wins the pending menu handoff', async ({ browser }) => {
        const context = await browser.newContext({ viewport: null, hasTouch: true });
        const page = await context.newPage();
        try {
            await mount(page, nav);
            await page.getByRole('button', { name: 'Open user menu', exact: true }).tap();
            await windowState(page, true);
            await page.locator('.maplibregl-canvas').tap({ position: { x: 300, y: 300 } });
            await page.evaluate(() => new Promise(requestAnimationFrame));
            await expect(page.locator('.maplibregl-canvas')).toBeFocused();
            const before = await counts(page);
            await page.keyboard.press('ArrowRight');
            await changed(page, before);
        } finally { await context.close(); }
    });
    test('motion hidden by rotation or list selection restores only published bounds', async ({ browser }) => {
        for (const action of ['rotation', 'list']) {
            const context = await browser.newContext({ ...devices['Pixel 7'] });
            const page = await context.newPage();
            try {
                await mount(page, bundle);
                if (action === 'rotation') {
                    await page.locator('.map-view-toggle').tap();
                    await page.setViewportSize({ width: 915, height: 412 });
                    await settle(page);
                }
                if (action === 'list') await page.evaluate(() => {
                    (window as any).fixture.afterFling = () => (document.querySelector('.map-view-toggle') as HTMLButtonElement).click();
                });
                await swipe(page, 1000, action === 'list');
                if (action === 'rotation') await page.setViewportSize({ width: 412, height: 915 });
                await expect(page.locator('.map-results')).toHaveAttribute('data-layout-ready', 'true');
                await page.waitForTimeout(650);
                const snapshot = await page.evaluate(() => (window as any).fixture.capture());
                const before = await counts(page);
                // A fling may finish before a physical rotation arrives. Both
                // valid outcomes agree with the published scope: an invalidated
                // camera, or a camera whose final movement was published.
                if (snapshot.camera) {
                    const params = new URLSearchParams(before.url);
                    expect(Math.abs(snapshot.camera.center[0] - (Number(params.get('tl_lon')) + Number(params.get('br_lon'))) / 2)).toBeLessThan(.005);
                    expect(Math.abs(snapshot.camera.center[1] - (Number(params.get('tl_lat')) + Number(params.get('br_lat'))) / 2)).toBeLessThan(.005);
                }
                await page.evaluate(value => (window as any).fixture.remount(value, 'popstate'), snapshot);
                expect(await quietCounts(page)).toMatchObject({ results: before.results + 1, publications: before.publications + 1 });
                const current = await camera(page), params = new URLSearchParams((await counts(page)).url);
                expect(Math.abs(current.center[0] - (Number(params.get('tl_lon')) + Number(params.get('br_lon'))) / 2)).toBeLessThan(.005);
                expect(Math.abs(current.center[1] - (Number(params.get('tl_lat')) + Number(params.get('br_lat'))) / 2)).toBeLessThan(.005);
            } finally { await context.close(); }
        }
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
