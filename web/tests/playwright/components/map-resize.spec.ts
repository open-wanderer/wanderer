import { expect, devices, type Page } from '@playwright/test';
import { windowTest as test, mount, counts, bounds, camera, settle, quietCounts, windowState } from './map-window-fixture';
import { buildMapPageFixture } from './map-page-fixture';

async function once(page: Page, before: Awaited<ReturnType<typeof counts>>, mapRequests?: number) {
    const after = await quietCounts(page);
    expect(after).toMatchObject({results: before.results + 1, publications: before.publications + 1});
    if (mapRequests !== undefined) expect(after.map).toBe(before.map + mapRequests);
    const published = await page.evaluate(() => (window as any).fixture.publications.at(-1)[0]);
    const visible = await bounds(page);
    for (const corner of ['northEast', 'southWest'] as const) for (const axis of ['lat', 'lng'] as const) {
        expect(published[corner][axis]).toBeCloseTo(visible[corner][axis], 10);
    }
    const params = new URLSearchParams((await counts(page)).url);
    for (const [key, value] of Object.entries({tl_lat: visible.northEast.lat, tl_lon: visible.northEast.lng, br_lat: visible.southWest.lat, br_lon: visible.southWest.lng})) {
        expect(Number(params.get(key))).toBeCloseTo(value, 10);
    }
}
async function recordCanvas(page: Page) {
    await page.evaluate(() => {
        const w = window as any, map = w.realMap;
        w.canvasErrors = [];
        const marker = document.createElement('div');
        marker.style.cssText = 'position:fixed;height:1px;pointer-events:none;opacity:0';
        document.body.append(marker);
        let flip = false, active = true;
        const observer = new ResizeObserver(() => {
            if (!active || !map.getContainer().checkVisibility()) return;
            const c = map.getCanvas(), r = map.getContainer();
            w.canvasErrors.push(Math.max(Math.abs(c.clientWidth-r.clientWidth),Math.abs(c.clientHeight-r.clientHeight)));
        });
        observer.observe(marker);
        const frame = () => { if(active){ marker.style.width = (flip = !flip) ? '1px' : '2px'; requestAnimationFrame(frame); } };
        requestAnimationFrame(frame);
        w.finishCanvas = () => { active = false; observer.disconnect(); marker.remove(); return w.canvasErrors; };
    });
}
async function canvasMatches(page: Page) {
    const errors = await page.evaluate(() => (window as any).finishCanvas());
    expect(errors.length).toBeGreaterThan(1);
    expect(Math.max(...errors)).toBeLessThanOrEqual(1);
}
for (const dev of [true, false]) test.describe(`normal window resize (${dev ? 'development' : 'production'})`, () => {
    let bundle: string;
    test.beforeAll(async () => { bundle = await buildMapPageFixture(dev, true, true); });
    test('F11 is an ordinary window resize and publishes exactly once in each direction', async ({page}) => {
        await mount(page, bundle);
        const original = await camera(page);
        for (const enabled of [true, false]) {
            const before = await quietCounts(page);
            await recordCanvas(page);
            await windowState(page, enabled);
            await once(page, before, 1);
            expect(await camera(page)).toEqual(original);
            await expect(page.locator('.map-page')).not.toHaveClass(/fullscreen/);
            await canvasMatches(page);
        }
    });
    test('4 measured window events with 150ms gaps paint every frame and publish 1 series', async ({page}) => {
        await mount(page, bundle);
        const before = await quietCounts(page);
        await recordCanvas(page);
        await page.clock.install();
        await page.clock.pauseAt(new Date(Date.now()+1000));
        const initial = await page.evaluate(() => {
            const w = window as any, replace = history.replaceState.bind(history);
            w.replacements = 0;
            w.resizeEvents = [];
            history.replaceState = (...args) => { ++w.replacements; return replace(...args); };
            addEventListener('resize', () => w.resizeEvents.push(performance.now()));
            return {width:innerWidth,height:innerHeight};
        });
        const session = await page.context().newCDPSession(page);
        try {
            for (let i=1; i<=4; i++) {
                if (i>1) await page.clock.runFor(150-16);
                await session.send('Emulation.setDeviceMetricsOverride', {
                    width:initial.width+i*6,height:initial.height+i*3,deviceScaleFactor:1,mobile:false,
                });
                await expect.poll(() => page.evaluate(() => (window as any).resizeEvents.length)).toBe(i);
                await page.clock.runFor(16);
            }
            await page.clock.runFor(800);
            const after = await quietCounts(page);
            expect(after.results).toBe(before.results+1);
            expect(after.publications).toBe(before.publications+1);
            expect(after.map).toBe(before.map+1);
            const visible = await bounds(page), published = await page.evaluate(() => (window as any).fixture.publications.at(-1)[0]);
            for (const corner of ['northEast', 'southWest'] as const) for (const axis of ['lat', 'lng'] as const) {
                expect(published[corner][axis]).toBeCloseTo(visible[corner][axis], 10);
            }
            const params = new URLSearchParams(after.url);
            for (const [key, value] of Object.entries({tl_lat: visible.northEast.lat, tl_lon: visible.northEast.lng, br_lat: visible.southWest.lat, br_lon: visible.southWest.lng})) {
                expect(Number(params.get(key))).toBeCloseTo(value, 10);
            }
            expect(await page.evaluate(() => (window as any).replacements)).toBeLessThanOrEqual(1);
            const events = await page.evaluate(() => (window as any).resizeEvents as number[]);
            expect(Math.max(...events.slice(1).map((at,i)=>at-events[i]))).toBeLessThan(200);
            await canvasMatches(page);
        } finally { await session.detach(); }
    });
    test('visual viewport and collapse supersede a window series without swallowing the next pan', async ({page}) => {
        await mount(page, bundle);
        const before = await quietCounts(page);
        await page.evaluate(() => {
            Object.defineProperty(window, 'innerHeight', {configurable: true, value: 650});
            Object.defineProperty(visualViewport, 'height', {configurable: true, value: 650});
            visualViewport!.dispatchEvent(new Event('resize'));
            requestAnimationFrame(() => dispatchEvent(new Event('resize')));
        });
        await once(page, before, 1);
        const resized = await counts(page);
        await page.evaluate(() => {
            Object.defineProperty(window, 'innerHeight', {configurable: true, value: 620});
            Object.defineProperty(visualViewport, 'height', {configurable: true, value: 620});
            dispatchEvent(new Event('resize'));
            requestAnimationFrame(() => (document.querySelector('.collapse-results') as HTMLButtonElement).click());
        });
        await once(page, resized, 1);
        const collapsed = await counts(page);
        await page.evaluate(() => {
            Object.defineProperty(window, 'innerHeight', {configurable: true, value: 590});
            Object.defineProperty(visualViewport, 'height', {configurable: true, value: 590});
            (window as any).realMap.panBy([80, 20], {duration: 0});
        });
        await once(page, collapsed, 1);
    });
    test('browser zoom resizes the drawing buffer and publishes once',async({page})=>{
        await mount(page,bundle);const before=await quietCounts(page),session=await page.context().newCDPSession(page);
        await session.send('Emulation.setDeviceMetricsOverride',{width:1024,height:640,deviceScaleFactor:1.25,mobile:false});
        await once(page,before);
        const sizes=await page.evaluate(()=>{const m=(window as any).realMap,c=m.getCanvas(),r=m.getContainer();return{actual:[c.width,c.height],expected:[Math.floor(r.clientWidth*devicePixelRatio),Math.floor(r.clientHeight*devicePixelRatio)]};});
        expect(sizes.actual).toEqual(sizes.expected);await session.detach();
    });
    test('rotating a phone with a held touch keeps the canvas current',async({browser})=>{
        const context=await browser.newContext({...devices['Pixel 7']});
        const page=await context.newPage();
        try {
            await mount(page,bundle);await page.addStyleTag({content:'output { display:none }'});await settle(page);
            const before=await quietCounts(page),cdp=await context.newCDPSession(page);
            const r=(await page.locator('.maplibregl-canvas').boundingBox())!;
            await cdp.send('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[{x:r.x+r.width/2,y:r.y+r.height/2}]});
            await recordCanvas(page);
            await page.setViewportSize({width:915,height:412});
            await cdp.send('Input.dispatchTouchEvent',{type:'touchEnd',touchPoints:[]});
            await once(page,before);await canvasMatches(page);
            const rotated = await quietCounts(page);
            await page.evaluate(() => (window as any).realMap.panBy([80, 20], {duration: 0}));
            await once(page, rotated, 1);
            await cdp.detach();
        } finally {await context.close();}
    });
});
