import { discardHiddenScrollPosition } from './map-scroll-fixture';
import { expect } from '@playwright/test';
import { windowTest as test, camera, bounds, counts, quietCounts, settle, windowState, mount, swipe } from './map-window-fixture';
import { installBrowserFullscreen } from '../browser_fullscreen';
import { buildMapPageFixture } from './map-page-fixture';

for (const dev of [true, false]) {
    test.describe(`real window and map (${dev ? 'development' : 'production'})`, () => {
        let bundle: string;
        test.beforeAll(async () => { bundle = await buildMapPageFixture(dev, true, true); });

        test('hidden results retain their position through a native F11 cycle',async({page})=>{
            await mount(page,bundle);
            await page.addStyleTag({content:'.trail-result { min-height:3000px }'});
            const list=page.locator('.trail-results');
            await list.evaluate(node=>node.scrollTop=1440);
            const before=await list.evaluate(node=>node.scrollTop);
            expect(before).toBeGreaterThan(1000);
            await discardHiddenScrollPosition(page);
            await page.locator('.collapse-results').click();
            await expect(list).not.toBeVisible();
            await windowState(page,true);
            await windowState(page,false);
            await page.locator('.collapse-results').click();
            await expect(list).toBeVisible();
            await expect.poll(async()=>Math.abs(await list.evaluate(node=>node.scrollTop)-before)).toBeLessThanOrEqual(2);
        });

        test('F11 keeps camera and list scope, loads the exact rendered area and hands focus back', async ({ page }) => {
            await mount(page, bundle);
            await page.evaluate(() => (window as any).realMap.jumpTo({ center: [8, 47], zoom: 7.2, bearing: 20, pitch: 30 }));
            await settle(page);
            for (const target of ['[data-testid="fixture-search"]', '.trail-result a', '.list-resize']) {
                await page.locator(target).focus();
                const before = await counts(page), original = await camera(page);
                for (const fullscreen of [true, false]) {
                    const previous = await counts(page);
                    await windowState(page, fullscreen);
                    const after = await quietCounts(page);
                    expect(after.results).toBe(before.results);
                    expect(after.publications).toBe(before.publications);
                    expect(after.url).toBe(before.url);
                    expect(after.map, JSON.stringify(await page.evaluate(() => ({ width: innerWidth, height: innerHeight, root: document.querySelector('.map-page')?.className, map: document.querySelector('#trail-map')?.getBoundingClientRect().toJSON(), renders: (window as any).fixture.renders })))).toBe(previous.map + 1);
                    expect(await camera(page)).toEqual(original);
                    const request = await page.evaluate(() => (window as any).fixture.requests.filter((r: any) => r.channel === 'map').at(-1).body);
                    const visible = await bounds(page);
                    for (const corner of ['northEast', 'southWest'] as const) for (const axis of ['lat', 'lng'] as const) expect(request[corner][axis]).toBeCloseTo(visible[corner][axis], 10);
                    await expect(page.locator(fullscreen ? '.fullscreen-search-toggle' : target)).toBeFocused();
                }
            }
        });

        for (const delay of [0, 16, 50]) {
            test(`real engine focus survives resize ${delay} ms after change`, async ({ page }) => {
                await installBrowserFullscreen(page);
                await mount(page, bundle);
                for (const target of ['[data-testid="fixture-search"]', '.trail-result a', '.list-resize']) {
                    await page.locator(target).focus();
                    for (const enabled of [true, false]) {
                        await page.evaluate(({ enabled, delay }) => {
                            (window as any).setTestBrowserFullscreen(enabled, true, false);
                            setTimeout(() => window.dispatchEvent(new Event('resize')), delay);
                        }, { enabled, delay });
                        await settle(page);
                        await expect(page.locator(enabled ? '.fullscreen-search-toggle' : target)).toBeFocused();
                    }
                }
            });
        }

        for (const motion of ['ease', 'zoom', 'swipe']) {
            test(`F11 finishes a running ${motion} gesture once at the displayed camera`, async ({ page }) => {
                await mount(page, bundle);
                await expect(async () => {
                    if (motion === 'ease') await page.evaluate(() => (window as any).realMap.easeTo({ center: [8.04, 47.02], zoom: 9, duration: 1500 }));
                    else if (motion === 'zoom') await page.evaluate(() => (window as any).realMap.zoomIn({ duration: 1500 }));
                    else await swipe(page);
                    expect(await page.evaluate(() => (window as any).realMap.isMoving())).toBe(true);
                }).toPass();
                const before = await counts(page);
                await windowState(page, true);
                const after = await quietCounts(page);
                expect(after.publications - before.publications).toBe(1);
                expect(after.results - before.results).toBe(1);
                const last = await page.evaluate(() => (window as any).fixture.publications.at(-1)[0]);
                const visible = await bounds(page);
                for (const corner of ['northEast', 'southWest'] as const) for (const axis of ['lat', 'lng'] as const) expect(last[corner][axis]).toBeCloseTo(visible[corner][axis], 10);
            });
        }

        for (const transition of ['fullscreen', 'resize']) {
            test(`load after ${transition} never restores the uninitialized camera`, async ({ page }) => {
                await mount(page, bundle, true);
                await page.evaluate(transition => {
                    const map = (window as any).realMap, resize = map._resizeInternal.bind(map);
                    let finished = false;
                    let resizes = 0;
                    map._resizeInternal = (...args: any[]) => {
                        const result = resize(...args);
                        if (++resizes === 2 && !finished) { finished = true; (window as any).finishMapLoad(); }
                        return result;
                    };
                    if (transition === 'resize') window.dispatchEvent(new Event('resize'));
                }, transition);
                if (transition === 'fullscreen') await windowState(page, true);
                await settle(page);
                const view = await camera(page);
                expect(view.center[0]).toBeCloseTo(8, 3);
                expect(view.center[1]).toBeCloseTo(47, 3);
                expect(view.zoom).toBeGreaterThan(5);
                expect((await counts(page)).url).not.toContain('tl_lat=90');
            });
        }

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
