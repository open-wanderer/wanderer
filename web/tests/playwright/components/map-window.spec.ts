import { expect } from '@playwright/test';
import { windowTest as test, camera, bounds, counts, quietCounts, settle, windowState, mount, clockedSwipe, recordInertiaTarget } from './map-window-fixture';
import { buildMapPageFixture } from './map-page-fixture';

for (const dev of [true, false]) {
    test.describe(`real window and map (${dev ? 'development' : 'production'})`, () => {
        let bundle: string;
        test.beforeAll(async () => { bundle = await buildMapPageFixture(dev, true, true); });

        for (const motion of ['ease', 'zoom', 'swipe']) {
            test(`a window resize during a running ${motion} publishes once at the final camera`, async ({ page }) => {
                await mount(page, bundle);
                const session = await page.context().newCDPSession(page);
                const {windowId} = await session.send('Browser.getWindowForTarget');
                const before = await quietCounts(page);
                const original = await camera(page);
                await page.clock.install();
                await page.clock.pauseAt(new Date(Date.now()+1000));
                await page.evaluate(()=>{(window as any).motionResizeEvents=0;addEventListener('resize',()=>++(window as any).motionResizeEvents);});
                if (motion === 'ease') await page.evaluate(() => (window as any).realMap.easeTo({center:[8.04,47.02],zoom:9,duration:1500}));
                else if (motion === 'zoom') await page.evaluate(() => (window as any).realMap.zoomIn({duration:1500}));
                else { await recordInertiaTarget(page); await clockedSwipe(page); }
                expect(await page.evaluate(()=>(window as any).realMap.isMoving())).toBe(true);
                await session.send('Browser.setWindowBounds',{windowId,bounds:{width:1100,height:800}});
                await expect.poll(()=>page.evaluate(()=>(window as any).motionResizeEvents)).toBeGreaterThan(0);
                // Node/CDP round trips cannot let the prerequisite animation finish
                // before the real resize arrives. Advance only after its handler ran.
                await page.clock.runFor(8000);
                expect(await page.evaluate(()=>(window as any).realMap.isMoving())).toBe(false);
                await settle(page); await session.detach();
                const after = await quietCounts(page);
                const reached = await camera(page);
                if (motion === 'ease') { expect(reached.center[0]).toBeCloseTo(8.04,8); expect(reached.center[1]).toBeCloseTo(47.02,8); expect(reached.zoom).toBeCloseTo(9,8); }
                else if (motion === 'zoom') expect(reached.zoom).toBeCloseTo(original.zoom+1,8);
                else {
                    const target = await page.evaluate(()=>(window as any).inertiaTarget);
                    expect(reached.center[0]).toBeCloseTo(target.center[0],6);
                    expect(reached.center[1]).toBeCloseTo(target.center[1],6);
                    expect(reached.zoom).toBeCloseTo(target.zoom,6);
                }
                expect(after.publications - before.publications).toBe(1);
                expect(after.results - before.results).toBe(1);
                const last = await page.evaluate(() => (window as any).fixture.publications.at(-1)[0]);
                const visible = await bounds(page);
                for (const corner of ['northEast', 'southWest'] as const) for (const axis of ['lat', 'lng'] as const) expect(last[corner][axis]).toBeCloseTo(visible[corner][axis], 10);
                const params = new URLSearchParams(after.url);
                for (const [key,value] of Object.entries({tl_lat:visible.northEast.lat,tl_lon:visible.northEast.lng,br_lat:visible.southWest.lat,br_lon:visible.southWest.lng})) expect(Number(params.get(key))).toBeCloseTo(value,10);
            });
        }

        test('a deferred window resize survives collapse during a fling', async ({page}) => {
            await mount(page,bundle);
            const before = await quietCounts(page);
            await page.clock.install();
            await page.clock.pauseAt(new Date(Date.now()+1000));
            await page.evaluate(()=>{(window as any).motionResizeEvents=0;addEventListener('resize',()=>++(window as any).motionResizeEvents);});
            await recordInertiaTarget(page);
            await clockedSwipe(page);
            expect(await page.evaluate(()=>(window as any).realMap.isMoving())).toBe(true);
            const session = await page.context().newCDPSession(page);
            const {windowId} = await session.send('Browser.getWindowForTarget');
            await session.send('Browser.setWindowBounds',{windowId,bounds:{width:1100,height:800}});
            await session.detach();
            await expect.poll(()=>page.evaluate(()=>(window as any).motionResizeEvents)).toBeGreaterThan(0);
            // Let the 200ms resize tail finish while inertia is still running.
            await page.clock.runFor(250);
            expect(await page.evaluate(()=>(window as any).realMap.isMoving())).toBe(true);
            const collapse = (await page.locator('.collapse-results').boundingBox())!;
            await page.mouse.click(collapse.x+collapse.width/2,collapse.y+collapse.height/2);
            await page.clock.runFor(8000);
            expect(await page.evaluate(()=>(window as any).realMap.isMoving())).toBe(false);
            const after = await quietCounts(page), reached = await camera(page);
            const target = await page.evaluate(()=>(window as any).inertiaTarget);
            expect(reached.center[0]).toBeCloseTo(target.center[0],6);
            expect(reached.center[1]).toBeCloseTo(target.center[1],6);
            expect(reached.zoom).toBeCloseTo(target.zoom,6);
            expect(after.publications-before.publications).toBe(1);
            expect(after.results-before.results).toBe(1);
            const visible = await bounds(page);
            const published = await page.evaluate(()=>(window as any).fixture.publications.at(-1)[0]);
            for (const corner of ['northEast','southWest'] as const) for (const axis of ['lat','lng'] as const) {
                expect(published[corner][axis]).toBeCloseTo(visible[corner][axis],10);
            }
            const params = new URLSearchParams(after.url);
            for (const [key,value] of Object.entries({tl_lat:visible.northEast.lat,tl_lon:visible.northEast.lng,br_lat:visible.southWest.lat,br_lon:visible.southWest.lng})) expect(Number(params.get(key))).toBeCloseTo(value,10);
        });

        test(`load after resize never restores the uninitialized camera`, async ({ page }) => {
            await mount(page, bundle, true);
            await page.evaluate(() => {
                const map = (window as any).realMap, resize = map._resizeInternal.bind(map);
                let finished = false;
                let resizes = 0;
                map._resizeInternal = (...args: any[]) => {
                    const result = resize(...args);
                    if (++resizes === 1 && !finished) { finished = true; (window as any).finishMapLoad(); }
                    return result;
                };
            });
            const session = await page.context().newCDPSession(page);
            const {windowId} = await session.send('Browser.getWindowForTarget');
            await session.send('Browser.setWindowBounds', {windowId,bounds:{width:1100,height:800}});
            await session.detach();
            await settle(page);
            const view = await camera(page);
            expect(view.center[0]).toBeCloseTo(8, 3);
            expect(view.center[1]).toBeCloseTo(47, 3);
            expect(view.zoom).toBeGreaterThan(5);
            expect((await counts(page)).url).not.toContain('tl_lat=90');
        });

        for (const scenario of ['control', 'fullscreen', 'leave-fullscreen']) {
            test(`snapshot remount retains center and zoom (${scenario})`, async ({ page }) => {
                await mount(page, bundle);
                await page.evaluate(() => (window as any).realMap.jumpTo({ center: [8, 47], zoom: 7.2, bearing: 12, pitch: 30 }));
                await settle(page);
                if (scenario !== 'control') await windowState(page, true);
                if (scenario === 'leave-fullscreen') {
                    await page.evaluate(() => (window as any).realMap.panBy([40, 30], { duration: 0 }));
                    await settle(page);
                    await windowState(page, false);
                }
                const original = await camera(page);
                await page.evaluate(async () => {
                    const value = (window as any).fixture.capture();
                    await (window as any).fixture.remount(value);
                });
                await settle(page);
                const restored = await camera(page);
                expect(restored.zoom).toBeCloseTo(original.zoom, 8);
                expect(restored.center[0]).toBeCloseTo(original.center[0], 8);
                expect(restored.center[1]).toBeCloseTo(original.center[1], 8);
            });
        }
    });
}
