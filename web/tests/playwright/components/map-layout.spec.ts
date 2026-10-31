import { expect, devices, type Page } from '@playwright/test';
import { windowTest as test, camera, bounds, counts, settle, windowState, mount, swipe, holdFullscreenTimers, quietCounts } from './map-window-fixture';
import { buildMapPageFixture } from './map-page-fixture';
import { installBrowserFullscreen } from '../browser_fullscreen';

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
        let navBundle: string;
        test.beforeAll(async () => {
            bundle = await buildMapPageFixture(dev, true, true);
            searchBundle = await buildMapPageFixture(dev, true, true, false, true);
            navBundle = await buildMapPageFixture(dev, true, true, true);
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
        test('fullscreen reveals a never-positioned hidden map without changing list scope', async ({ page }) => {
            await size(page, 390);
            await mount(page, bundle, true);
            await page.locator('.map-view-toggle').click();
            await resolveRequests(page);
            await page.evaluate(() => (window as any).finishMapLoad());
            expect(await quietCounts(page)).toMatchObject({ results: 1, publications: 1 });
            const before = await counts(page);
            await windowState(page, true);
            const after = await counts(page);
            expect(after.results).toBe(before.results);
            expect(after.publications).toBe(before.publications);
            expect(after.url).toBe(before.url);
            expect((await camera(page)).center[0]).toBeCloseTo(8, 3);
            expect((await camera(page)).zoom).toBeGreaterThan(8);
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

        for (const control of ['.list-resize', '.overlay-move']) {
            test(`a ${control} click superseding resize releases the entire transaction`, async ({ page }) => {
                await mount(page, bundle);
                if (control === '.overlay-move') {
                    await page.locator('.result-header .expand-results').click(); await settle(page);
                    await page.locator('.overlay-resize').press('End'); await settle(page);
                }
                await page.evaluate(control => {
                    const raf = requestAnimationFrame;
                    const after = performance.now() + 100;
                    let armed = true;
                    window.requestAnimationFrame = callback => raf(time => {
                        if (armed && performance.now() >= after) {
                            armed = false;
                            const target = document.querySelector(control) as HTMLElement;
                            target.setPointerCapture = () => {};
                            target.hasPointerCapture = () => false;
                            target.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, pointerId: 14, button: 0 }));
                            target.dispatchEvent(new PointerEvent('pointerup', { bubbles: true, pointerId: 14, button: 0 }));
                            (window as any).interruptedSettle = true;
                            window.requestAnimationFrame = raf;
                        }
                        callback(time);
                    });
                    dispatchEvent(new Event('resize'));
                }, control);
                await expect.poll(() => page.evaluate(() => (window as any).interruptedSettle)).toBe(true);
                await expect(page.locator('.map-results')).toHaveAttribute('data-layout-ready', 'true');
                const before = await counts(page);
                await page.locator('[data-search-root]').evaluate(node => (node as HTMLElement).style.height = '320px');
                expect(await quietCounts(page)).toMatchObject({ map: before.map + 1, results: before.results });
                await pan(page);
                expect(await quietCounts(page)).toMatchObject({ results: before.results + 1 });
                await settle(page);
                const panned = await counts(page);
                await windowState(page, true);
                expect((await counts(page)).results).toBe(panned.results);
                expect((await counts(page)).url).toBe(panned.url);
                await windowState(page, false);
                const exited = await counts(page);
                await size(page, 800);
                await changedOnce(page, exited);
            });
        }

        for (const delay of [300, 400, 700]) {
            test(`fullscreen change still owns a first resize delayed ${delay} ms`, async ({ page }) => {
                await installBrowserFullscreen(page);
                await mount(page, bundle);
                await page.getByTestId('fixture-search').focus();
                for (const enabled of [true, false]) {
                    if (!enabled) { await pan(page); await settle(page); }
                    const before = await counts(page);
                    await page.evaluate(({ enabled, delay }) => {
                        (window as any).setTestBrowserFullscreen(enabled, true, false);
                    }, { enabled, delay });
                    if (enabled) await expect(page.locator('.map-page')).toHaveClass(/fullscreen/);
                    else await expect(page.locator('.map-page')).not.toHaveClass(/fullscreen/);
                    await page.waitForTimeout(delay);
                    await size(page, enabled ? 1600 : 1280, enabled ? 1000 : 800);
                    await settle(page);
                    await page.waitForTimeout(200); // Negative: no delayed second search/map request.
                    const after = await counts(page);
                    expect(after.results).toBe(before.results);
                    expect(after.url).toBe(before.url);
                    expect(after.map).toBe(before.map + 1);
                }
                await page.waitForTimeout(1100); // Outside the bounded transition interval.
                const before = await counts(page);
                await size(page, 1100);
                await changedOnce(page, before);
            });
        }
        test('retains an area requested during the fullscreen resize wait', async ({ page }) => {
            await installBrowserFullscreen(page);
            await mount(page, bundle);
            await page.evaluate(() => (window as any).realMap.jumpTo({ center: [8, 47], zoom: 10 }));
            await settle(page);
            await page.evaluate(() => (window as any).setTestBrowserFullscreen(true));
            await settle(page);
            const area = await page.evaluate(() => (window as any).fixture.capture().viewport);
            const beforeExit = await counts(page);
            await page.evaluate(() => (window as any).setTestBrowserFullscreen(false, true, false));
            await expect(page.locator('.map-page')).not.toHaveClass(/fullscreen/);
            const previousSize = await page.locator('#trail-map').boundingBox();
            await page.locator('.result-header .expand-results').click();
            await settle(page);
            const nextSize = await page.locator('#trail-map').boundingBox();
            expect([nextSize!.width,nextSize!.height]).not.toEqual([previousSize!.width,previousSize!.height]);
            const expected = await page.evaluate(area => (window as any).realMap.cameraForBounds([
                [area.southWest.lng, area.southWest.lat], [area.northEast.lng, area.northEast.lat],
            ]).zoom, area);
            expect((await camera(page)).zoom).toBeCloseTo(expected, 6);
            expect(await quietCounts(page)).toMatchObject({ results: beforeExit.results, publications: beforeExit.publications, url: beforeExit.url });
        });
        test('a late display-mode event does not stop motion started after the first resize', async ({ page }) => {
            await installBrowserFullscreen(page);
            await mount(page, bundle);
            const before = await counts(page);
            await page.evaluate(() => {
                addEventListener('resize', () => {
                    (window as any).realMap.easeTo({ center: [8.1, 47.02], zoom: 11, duration: 500 });
                    setTimeout(() => (window as any).setTestBrowserFullscreen(true, true, false), 50);
                }, { once: true });
            });
            await size(page, 1500, 1000);
            await expect(page.locator('.map-page')).toHaveClass(/fullscreen/);
            await expect.poll(() => page.evaluate(() => (window as any).realMap.isMoving())).toBe(false);
            expect((await camera(page)).zoom).toBeCloseTo(11, 6);
            expect((await camera(page)).center[0]).toBeCloseTo(8.1, 6);
            expect(await quietCounts(page)).toMatchObject({ results: before.results + 1, publications: before.publications + 1 });
            await settle(page);
        });

        test('fullscreen browser zoom publishes once, while an inner-only browser panel preserves scope', async ({ page }) => {
            await mount(page, bundle);
            await windowState(page, true);
            await page.waitForTimeout(1100);
            const before = await counts(page);
            const cdp = await page.context().newCDPSession(page);
            await cdp.send('Emulation.setDeviceMetricsOverride', { width: 1280, height: 800, deviceScaleFactor: 1.25, mobile: false });
            await changedOnce(page, before);
            const zoomed = await counts(page);
            await cdp.send('Emulation.setDeviceMetricsOverride', { width: 1280, height: 650, deviceScaleFactor: 1.25, mobile: false });
            expect(await quietCounts(page)).toMatchObject({ map: zoomed.map + 1 });
            await settle(page);
            expect((await counts(page)).results).toBe(zoomed.results);
            expect((await counts(page)).url).toBe(zoomed.url);
            const panel = await counts(page);
            await cdp.send('Emulation.setDeviceMetricsOverride', { width: 650, height: 1000, deviceScaleFactor: 1.25, mobile: false,
                screenOrientation: { type: 'portraitPrimary', angle: 0 } });
            await changedOnce(page, panel);
            await cdp.detach();
        });

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

        for (const selector of ['.result-header .collapse-results', '.result-header .expand-results', '.collapsed-actions .collapse-results', '.overlay-close', '.overlay-move']) {
            test(`fullscreen returns focus from a hidden ${selector} trigger to a visible equivalent`, async ({ page }) => {
                await mount(page, bundle);
                if (selector.startsWith('.collapsed')) await page.locator('.result-header .collapse-results').click();
                if (selector.startsWith('.overlay')) { await page.locator('.result-header .expand-results').click(); await settle(page); }
                await page.locator(selector).focus();
                await windowState(page, true);
                await windowState(page, false);
                expect(await page.evaluate(() => document.activeElement !== document.body && (document.activeElement as HTMLElement).checkVisibility({ visibilityProperty: true }))).toBe(true);
            });
        }
        test('mounting into existing browser fullscreen causes no focus handoff', async ({ page }) => {
            const cdp = await page.context().newCDPSession(page);
            const { windowId } = await cdp.send('Browser.getWindowForTarget');
            await cdp.send('Browser.setWindowBounds', { windowId, bounds: { windowState: 'fullscreen' } });
            await page.addInitScript(() => { (window as any).focusEvents = []; addEventListener('focusin', event => (window as any).focusEvents.push((event.target as Element).tagName)); });
            await mount(page, bundle);
            await page.waitForTimeout(1200); // Negative: delayed automatic focus must not occur.
            expect(await page.evaluate(() => (window as any).focusEvents)).toEqual([]);
            await cdp.detach();
        });
        test('real browser fullscreen keeps navigation overlays, modal focus and uploads usable', async ({ page }) => {
            await mount(page, navBundle);
            await page.locator('.create-trigger').click();
            await windowState(page, true);
            await expect(page.locator('nav:has(#nav-bar-links) [aria-label="Toggle theme"]')).toHaveCount(1);
            expect(await page.locator('nav:has(#nav-bar-links) [aria-label="Toggle theme"]').evaluate(node => Boolean(node.closest('[inert]')))).toBe(true);
            await page.locator('.create-panel button').last().click();
            const dialog = page.locator('dialog[open]');
            await expect(dialog.getByRole('button', { name: 'Close modal', exact: true })).toBeFocused();
            await expect.poll(() => dialog.ariaSnapshot()).toContain('Close modal');
            await dialog.getByRole('textbox').click();
            await page.evaluate(() => new Promise(requestAnimationFrame));
            // An overlay update must keep the handoff deferred; interacting
            // inside the dialog must not discard its target after closing.
            await dialog.evaluate(node => node.classList.add('focus-probe'));
            await page.evaluate(() => new Promise(requestAnimationFrame));
            await dialog.getByRole('button', { name: 'Close modal', exact: true }).click();
            await expect(page.locator('.fullscreen-search-toggle')).toBeFocused();
            await page.evaluate(() => (window as any).fixture.addUpload());
            const cancel = page.getByRole('button', { name: 'Cancel upload', exact: true });
            await expect(cancel).toBeVisible();
            await cancel.click();
            await expect(cancel).toHaveCount(0);
        });
        test('a selection dropdown hidden by fullscreen does not suppress focus handoff', async ({ page }) => {
            await mount(page, bundle);
            await page.locator('.trail-result').first().hover();
            await page.locator('.trail-result input[type=checkbox]').first().check();
            await page.locator('.selection-menu').click();
            await expect(page.locator('.selection-toolbar [data-fullscreen-overlay]')).toBeVisible();
            await windowState(page, true);
            await expect(page.locator('.selection-toolbar [data-fullscreen-overlay]')).toBeHidden();
            await expect(page.locator('.fullscreen-search-toggle')).toBeFocused();
            await windowState(page, false);
            expect(await page.evaluate(() => document.activeElement !== document.body && (document.activeElement as HTMLElement).checkVisibility({ visibilityProperty: true }))).toBe(true);
        });
        test('fullscreen restores focus to the main search when the filter-field trigger was removed', async ({ page }) => {
            await mount(page, searchBundle);
            await page.getByRole('searchbox').first().focus();
            await page.locator('.general-filters .filter-chip').filter({ hasText: 'trail-search-place' }).click();
            await page.locator('.place-input input').focus();
            await windowState(page, true);
            await expect(page.locator('.fullscreen-search-toggle')).toBeFocused();
            await windowState(page, false);
            await expect(page.locator('.search-heading input')).toBeFocused();
            await expect(page.locator('.search-form')).not.toHaveClass(/expanded/);
        });
        for (const key of ['Shift', 'Control', 'Alt', 'Meta', 'a']) {
            test(`${key} alone does not cancel focus handoff onto a visible control`, async ({ page }) => {
                await installBrowserFullscreen(page);
                await mount(page, bundle);
                await page.getByTestId('fixture-search').focus();
                for (const enabled of [true, false]) {
                    await holdFullscreenTimers(page);
                    await page.evaluate(enabled => (window as any).setTestBrowserFullscreen(enabled, true, false), enabled);
                    await page.keyboard.press(key);
                    await page.evaluate(() => (window as any).flushFullscreenTimers());
                    await settle(page);
                    await expect(page.locator(enabled ? '.fullscreen-search-toggle' : '[data-testid="fixture-search"]')).toBeFocused();
                }
            });
        }
        for (const action of ['Tab', 'click']) {
            test(`a real ${action} focus choice wins over the delayed fullscreen handoff`, async ({ page }) => {
                await installBrowserFullscreen(page);
                await mount(page, bundle);
                await page.getByTestId('fixture-search').focus();
                await holdFullscreenTimers(page);
                await page.evaluate(() => (window as any).setTestBrowserFullscreen(true, true, false));
                await expect(page.locator('.fullscreen-search-toggle')).toBeVisible();
                if (action === 'Tab') {
                    await page.locator('.fullscreen-search-toggle').focus();
                    await page.keyboard.press('Tab');
                } else await page.locator('.fullscreen-search-toggle').click();
                const chosen = await page.evaluate(() => {
                    const element = document.activeElement as HTMLElement;
                    element.dataset.focusChoice = 'true';
                    return element.tagName;
                });
                expect(chosen).not.toBe('BODY');
                expect(await page.locator('[data-focus-choice="true"]').evaluate(node => node.matches('.fullscreen-search-toggle'))).toBe(false);
                await page.evaluate(() => new Promise(requestAnimationFrame));
                await page.evaluate(() => (window as any).flushFullscreenTimers());
                await settle(page);
                await expect(page.locator('[data-focus-choice="true"]')).toBeFocused();
            });
        }
    });
}
