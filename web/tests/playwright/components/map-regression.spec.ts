import { expect, devices, type Page, type CDPSession } from '@playwright/test';
import { windowTest as test, mount, quietCounts, bounds, clockedSwipe } from './map-window-fixture';
import { buildMapPageFixture } from './map-page-fixture';

async function publishedOnce(page: Page, before: Awaited<ReturnType<typeof quietCounts>>) {
    const after = await quietCounts(page);
    expect(after.results).toBe(before.results + 1);
    expect(after.publications).toBe(before.publications + 1);
    const area = await bounds(page);
    const published = await page.evaluate(() => (window as any).fixture.publications.at(-1)[0]);
    const params = new URLSearchParams(after.url);
    for (const corner of ['northEast', 'southWest'] as const) for (const axis of ['lat', 'lng'] as const) {
        expect(published[corner][axis]).toBeCloseTo(area[corner][axis], 10);
    }
    expect(Number(params.get('tl_lat'))).toBeCloseTo(area.northEast.lat, 10);
    expect(Number(params.get('br_lon'))).toBeCloseTo(area.southWest.lng, 10);
}

for (const dev of [true, false]) test.describe(`map transaction regressions (${dev ? 'development' : 'production'})`, () => {
    let bundle: string;
    test.beforeAll(async () => { bundle = await buildMapPageFixture(dev, true, true); });

    for (const motion of ['zoom', 'inertia']) for (const layout of ['collapse', 'splitter', 'keyboard']) {
        test(`${motion} with ${layout} publishes its final area despite unheld mouse movement`, async ({ page }) => {
            await mount(page, bundle);
            const before = await quietCounts(page);
            const grip = (await page.locator('.list-resize').boundingBox())!;
            const collapse = (await page.locator('.collapse-results').boundingBox())!;
            const zoom = (await page.locator('.maplibregl-ctrl-zoom-in').boundingBox())!;
            // Keep the prerequisite motion alive across Node round trips under
            // CPU load. The layout action must land before its moveend.
            await page.clock.install();
            await page.clock.pauseAt(new Date(Date.now() + 1000));
            const originalZoom = await page.evaluate(() => (window as any).realMap.getZoom());
            if (motion === 'zoom') {
                await page.evaluate(() => {
                    const map = (window as any).realMap, zoomIn = map.zoomIn.bind(map);
                    map.zoomIn = (options: any, data: any) => zoomIn({ ...options, duration: 1200 }, data);
                });
                await page.mouse.click(zoom.x + zoom.width / 2, zoom.y + zoom.height / 2);
                await page.clock.runFor(16);
            } else await clockedSwipe(page);
            expect(await page.evaluate(() => (window as any).realMap.isMoving())).toBe(true);
            if (layout === 'collapse') await page.mouse.click(collapse.x + collapse.width / 2, collapse.y + collapse.height / 2);
            else if (layout === 'keyboard') await page.locator('.list-resize').press('Shift+ArrowLeft');
            else {
                await page.mouse.move(grip.x + grip.width / 2, grip.y + grip.height / 2);
                await page.mouse.down();
                await page.mouse.move(grip.x - 80, grip.y + grip.height / 2, { steps: 4 });
                await page.mouse.up();
            }
            let elapsed = 0;
            while (await page.evaluate(() => (window as any).realMap.isMoving())) {
                if (elapsed > 10000) throw new Error('Camera motion did not finish');
                await page.evaluate(() => dispatchEvent(new MouseEvent('mousemove', { buttons: 0, clientX: 20, clientY: 20 })));
                await page.clock.runFor(16);
                elapsed += 16;
            }
            for (let i = 0; i < 3; i++) {
                await page.evaluate(i => dispatchEvent(new MouseEvent('mousemove', { buttons: 0, clientX: 20 + i, clientY: 20 })), i);
                await page.clock.runFor(16);
            }
            await page.clock.runFor(1000);
            if (motion === 'zoom') expect(await page.evaluate(() => (window as any).realMap.getZoom())).toBeCloseTo(originalZoom + 1, 8);
            await publishedOnce(page, before);
        });
    }

    for (const pointer of ['mouse', 'touch']) for (const trigger of ['blur', 'hidden']) {
        test(`a held ${pointer} on ${trigger} lets camera easing reach its target exactly once`, async ({ browser, page: desktop }) => {
            const context = pointer === 'touch' ? await browser.newContext({ ...devices['Pixel 7'], viewport: { width: 1000, height: 700 } }) : undefined;
            const page = context ? await context.newPage() : desktop;
            let cdp: CDPSession | undefined;
            try {
                await mount(page, bundle);
                const before = await quietCounts(page);
                const box = (await page.locator('.maplibregl-canvas').boundingBox())!;
                const x = box.x + box.width / 2, y = box.y + box.height / 2;
                if (pointer === 'mouse') { await page.mouse.move(x, y); await page.mouse.down(); }
                else {
                    cdp = await page.context().newCDPSession(page);
                    await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x, y }] });
                }
                await page.evaluate(() => {
                    const map = (window as any).realMap;
                    (window as any).motionEnds = 0;
                    map.on('moveend', () => ++(window as any).motionEnds);
                    map.easeTo({ center: [8.05, 47.03], duration: 900 });
                });
                await expect.poll(() => page.evaluate(() => (window as any).realMap.getCenter().lng)).toBeGreaterThan(8.005);
                expect(await page.evaluate(() => (window as any).realMap.isMoving())).toBe(true);
                expect(await page.evaluate(() => {
                    const map = (window as any).realMap;
                    return [map.dragPan, map.touchZoomRotate, map.dragRotate, map.touchPitch].some(handler => handler.isActive());
                })).toBe(false);
                if (trigger === 'blur') await page.evaluate(() => dispatchEvent(new Event('blur')));
                else await page.evaluate(() => { Object.defineProperty(document, 'hidden', { configurable: true, value: true }); document.dispatchEvent(new Event('visibilitychange')); });
                await expect.poll(() => page.evaluate(() => (window as any).realMap.isMoving())).toBe(false);
                const center = await page.evaluate(() => (window as any).realMap.getCenter().toArray());
                expect(center[0]).toBeCloseTo(8.05, 9);
                expect(center[1]).toBeCloseTo(47.03, 9);
                expect(await page.evaluate(() => (window as any).motionEnds)).toBe(1);
                if (pointer === 'mouse') await page.mouse.up();
                else await cdp!.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
                if (trigger === 'hidden') await page.evaluate(() => { delete (document as any).hidden; document.dispatchEvent(new Event('visibilitychange')); });
                await publishedOnce(page, before);
            } finally { await cdp?.detach(); await context?.close(); }
        });
    }
});
