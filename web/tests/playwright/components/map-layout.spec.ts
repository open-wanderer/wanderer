import { expect, devices, type Page } from '@playwright/test';
import { windowTest as test, camera, bounds, counts, settle, mount, swipe, quietCounts } from './map-window-fixture';
import { buildMapPageFixture } from './map-page-fixture';

async function size(page: Page, width: number, height = 800) {
    const cdp = await page.context().newCDPSession(page);
    const { windowId } = await cdp.send('Browser.getWindowForTarget');
    await cdp.send('Browser.setWindowBounds', { windowId, bounds: { width, height } });
    await expect.poll(() => page.evaluate(() => innerWidth)).toBe(width);
    await cdp.detach();
}
async function resolveRequests(page: Page) {
    await page.evaluate(() => {
        const f = (window as any).fixture;
        f.autoResolve = true;
        for (const r of f.requests) r.resolve(r.channel === 'map'
            ? { mapTrails: [], clusters: { type: 'FeatureCollection', features: [] } }
            : { items: [{ id: 'fixture-1', author: 'tester', name: 'Fixture trail' }], page: 1, totalPages: 1, totalHits: 1 });
    });
}
async function changedOnce(page: Page, before: Awaited<ReturnType<typeof counts>>) {
    expect(await quietCounts(page)).toMatchObject({ results: before.results + 1, map: before.map + 1, publications: before.publications + 1 });
    expect((await counts(page)).url).not.toBe(before.url);
}
async function pan(page: Page) {
    await page.evaluate(() => (window as any).realMap.panBy([80, 30], { duration: 0 }));
}

for (const dev of [true, false]) {
    test.describe(`map layout regressions (${dev ? 'development' : 'production'})`, () => {
        let bundle: string;
        let searchBundle: string;
        test.beforeAll(async () => {
            bundle = await buildMapPageFixture(dev, true, true);
            searchBundle = await buildMapPageFixture(dev, true, true, false, true);

        });

        for (const width of [390, 600]) for (const timing of ['popstate', 'hydrate']) {
            test(`hidden map before load has no snapshot camera (${width}, ${timing})`, async ({ page }) => {
                await size(page, width);
                await mount(page, bundle, true);
                await page.locator('.map-view-toggle').click();
                await resolveRequests(page);
                await page.evaluate(() => (window as any).finishMapLoad());
                expect(await quietCounts(page)).toMatchObject({ results: 1, publications: 1 });
                const saved = await page.evaluate(() => (window as any).fixture.capture());
                expect(saved.camera).toBeUndefined();
                const before = await counts(page);
                await page.evaluate(async ({ saved, timing }) => {
                    (window as any).deferMapLoad = false;
                    await (window as any).fixture.remount(saved, timing);
                }, { saved, timing });
                expect(await quietCounts(page)).toMatchObject({ results: before.results + 1, publications: before.publications + 1 });
                await settle(page);
                const restored = await camera(page);
                expect(restored.center[0]).toBeCloseTo(8, 3);
                expect(restored.center[1]).toBeCloseTo(47, 3);
                expect(restored.zoom).toBeGreaterThan(8);
                expect(Number(new URLSearchParams((await counts(page)).url).get('tl_lat'))).toBeCloseTo(47.1, 2);
            });
        }
        test('choosing mobile list after positioning retains its exact snapshot camera', async ({ page }) => {
            await size(page, 600);
            await mount(page, bundle);
            await page.evaluate(() => (window as any).realMap.jumpTo({ center: [8, 47], zoom: 10, bearing: 12, pitch: 30 }));
            await settle(page);
            const original = await camera(page);
            await page.locator('.map-view-toggle').click();
            expect(await page.evaluate(() => (window as any).fixture.capture().camera)).toEqual(original);
        });

        for (const delayed of [false, true]) {
            test(`mobile list becomes a live visible map on widening (delayed load=${delayed})`, async ({ page }) => {
                await size(page, 700);
                await mount(page, bundle, delayed);
                await page.locator('.map-view-toggle').click();
                if (delayed) { await resolveRequests(page); await page.evaluate(() => (window as any).finishMapLoad()); }
                expect(await quietCounts(page)).toMatchObject({ results: 1 });
                const hidden = await counts(page);
                await size(page, 1280);
                expect(await quietCounts(page)).toMatchObject({ results: hidden.results + 1 });
                await settle(page);
                for (const action of ['pan', 'zoom', 'reset', 'resize']) {
                    const before = await counts(page);
                    if (action === 'pan') await pan(page);
                    else if (action === 'zoom') await page.locator('.maplibregl-ctrl-zoom-in').click();
                    else if (action === 'reset') await page.locator('.reset-map-view').click();
                    else await size(page, 1100);
                    await changedOnce(page, before);
                }
            });
        }

        test('phone list selection keeps the rotated landscape map interactive', async ({ browser }) => {
            const context = await browser.newContext({ ...devices['Pixel 7'] });
            const page = await context.newPage();
            try {
                await mount(page, bundle);
                await page.locator('.map-view-toggle').tap();
                const hidden = await counts(page);
                await page.setViewportSize({ width: 915, height: 412 });
                expect(await quietCounts(page)).toMatchObject({ results: hidden.results + 1 });
                await settle(page);
                const before = await counts(page);
                const r = (await page.locator('.maplibregl-canvas').boundingBox())!;
                const cdp = await context.newCDPSession(page);
                const x = r.x + r.width / 2, y = r.y + r.height / 2;
                await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x, y }] });
                for (let step = 1; step <= 6; step++) {
                    await cdp.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x: x + step * 15, y }] });
                    await page.evaluate(() => new Promise(requestAnimationFrame));
                }
                await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
                await changedOnce(page, before);
                await cdp.detach();
            } finally { await context.close(); }
        });

        test('the first push drag keeps its geographic anchor without moving the panel', async ({ page }) => {
            await mount(page, searchBundle, false, true);
            await page.getByRole('searchbox').first().focus();
            await expect(page.locator('.search-panel')).toBeVisible();
            const before = await camera(page);
            const requests = await counts(page);
            const r = (await page.locator('.maplibregl-canvas').boundingBox())!;
            const y = r.y + r.height - 40;
            await page.mouse.move(r.x + r.width / 2, y);
            await page.mouse.down();
            await expect(page.locator('.search-panel')).toBeVisible();
            await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
            await page.mouse.move(r.x + r.width / 2 + 160, y, { steps: 8 });
            await page.mouse.up();
            expect(await quietCounts(page)).toMatchObject({ results: requests.results + 1, publications: requests.publications + 1 });
            await settle(page);
            expect(Math.abs((await camera(page)).center[0] - before.center[0])).toBeGreaterThan(.01);
        });
        test('the first phone touch drag keeps the push panel stable', async ({ browser }) => {
            const context = await browser.newContext({ ...devices['Pixel 7'] });
            const page = await context.newPage();
            try {
                await mount(page, searchBundle, false, true);
                await page.getByRole('searchbox').first().focus();
                await expect(page.locator('.search-panel')).toBeVisible();
                const before = await camera(page);
                const r = (await page.locator('.maplibregl-canvas').boundingBox())!;
                const cdp = await context.newCDPSession(page);
                const x = r.x + r.width / 2, y = Math.min(r.y + r.height - 30, 700);
                await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x, y }] });
                await expect(page.locator('.search-panel')).toBeVisible();
                await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
                for (let step = 1; step <= 6; step++) {
                    await cdp.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x: x + step * 15, y }] });
                    await page.evaluate(() => new Promise(requestAnimationFrame));
                }
                await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
                await settle(page);
                expect(Math.abs((await camera(page)).center[0] - before.center[0])).toBeGreaterThan(.01);
                await cdp.detach();
            } finally { await context.close(); }
        });

        test('header resize lets real swipe inertia reach its uninterrupted engine target', async ({ page }) => {
            await mount(page, bundle);
            await page.evaluate(() => {
                const map = (window as any).realMap, ease = map.easeTo.bind(map);
                map.easeTo = (options: any, data: any) => {
                    // Let a separate real engine compute the endpoint of this
                    // very gesture. Different frame rates can change velocity,
                    // so a second independently timed swipe is not a control.
                    const container = document.createElement('div');
                    container.style.cssText = `position:fixed;left:-10000px;width:${map.getContainer().clientWidth}px;height:${map.getContainer().clientHeight}px`;
                    document.body.append(container);
                    const control = new (window as any).RealMapLibre.Map({ container, interactive: false,
                        center: map.getCenter(), zoom: map.getZoom(), bearing: map.getBearing(), pitch: map.getPitch(),
                        style: { version: 8, sources: {}, layers: [] } });
                    control.easeTo({ ...options, duration: 0 });
                    (window as any).inertiaTarget = { center: control.getCenter().toArray(), zoom: control.getZoom() };
                    control.remove(); container.remove();
                    map.easeTo = ease;
                    return ease(options, data);
                };
            });
            const before = await counts(page);
            await swipe(page, 1000);
            await expect.poll(() => page.evaluate(() => Boolean((window as any).inertiaTarget))).toBe(true);
            expect(await page.evaluate(() => (window as any).realMap.isMoving())).toBe(true);
            await page.locator('[data-search-root]').evaluate(node => (node as HTMLElement).style.height = '120px');
            expect(await quietCounts(page)).toMatchObject({ results: before.results + 1, publications: before.publications + 1 });
            await settle(page);
            const target = await page.evaluate(() => (window as any).inertiaTarget);
            const result = await camera(page);
            expect(result.center[0]).toBeCloseTo(target.center[0], 6);
            expect(result.center[1]).toBeCloseTo(target.center[1], 6);
            expect(result.zoom).toBeCloseTo(target.zoom, 6);
        });

        for (const motion of ['zoom', 'double-click', 'keyboard', 'ease']) for (const source of ['header', 'visual-viewport']) {
            test(`${source} layout waits for a complete ${motion} animation`, async ({ page }) => {
                await mount(page, bundle);
                const start = { center: [8, 47], zoom: 10, bearing: 0, pitch: 0 };
                const run = async (resize: boolean, expectedZoom?: number) => {
                    await page.evaluate(start => (window as any).realMap.jumpTo(start), start);
                    await settle(page);
                    const before = await counts(page);
                    await page.evaluate(({ motion, resize, source }) => {
                        const map = (window as any).realMap;
                        if (motion === 'zoom') map.getContainer().querySelector('.maplibregl-ctrl-zoom-in').click();
                        else if (motion === 'double-click') map.getCanvas().dispatchEvent(new MouseEvent('dblclick', { bubbles: true, button: 0, clientX: 800, clientY: 400 }));
                        else if (motion === 'keyboard') {
                            map.getCanvas().focus();
                            map.getCanvas().dispatchEvent(new KeyboardEvent('keydown', { bubbles: true, key: 'ArrowRight', code: 'ArrowRight', keyCode: 39 }));
                        } else map.easeTo({ center: [8.1, 47.02], duration: 500 });
                        (window as any).fixture.animationStarted = map.isMoving();
                        if (resize) {
                            // Make the layout change in this same renderer task,
                            // while the animation is certainly still running.
                            (document.querySelector('[data-search-root]') as HTMLElement).style.height = '120px';
                            if (source === 'visual-viewport') visualViewport?.dispatchEvent(new Event('resize'));
                        }
                    }, { motion, resize, source });
                    expect(await page.evaluate(() => (window as any).fixture.animationStarted)).toBe(true);
                    await expect.poll(() => page.evaluate(() => (window as any).realMap.isMoving())).toBe(false);
                    if (expectedZoom !== undefined) expect((await camera(page)).zoom).toBeCloseTo(expectedZoom, 6);
                    expect(await quietCounts(page)).toMatchObject({ results: before.results + 1, publications: before.publications + 1 });
                    await settle(page);
                    return camera(page);
                };
                const control = await run(false);
                const resized = await run(true, control.zoom);
                expect(resized.zoom).toBeCloseTo(control.zoom, 6);
                expect(resized.center[0]).toBeCloseTo(control.center[0], 6);
                expect(resized.center[1]).toBeCloseTo(control.center[1], 6);
            });
        }
    });
}
