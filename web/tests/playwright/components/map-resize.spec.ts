import { expect, devices, type Page, type CDPSession } from '@playwright/test';
import { windowTest as test, mount, counts, bounds, settle, quietCounts, windowState } from './map-window-fixture';
import { buildMapPageFixture } from './map-page-fixture';

async function watchWindowLatency(page: Page) {
    await page.clock.install();
    await page.clock.pauseAt(new Date(Date.now() + 1000));
    return page.evaluate(() => {
        const w = window as any, fetch = window.fetch.bind(window), replace = history.replaceState.bind(history);
        const measured: { events: number[]; requests: { channel: string; at: number }[]; replacements: number[] } = w.windowLatency = { events: [], requests: [], replacements: [] };
        addEventListener('resize', () => measured.events.push(performance.now()));
        window.fetch = (input, options) => {
            if (input === 'results' || input === 'map') measured.requests.push({ channel: input, at: performance.now() });
            return fetch(input, options);
        };
        history.replaceState = (...args) => {
            measured.replacements.push(performance.now());
            return replace(...args);
        };
        return { width: innerWidth, height: innerHeight };
    });
}
async function measuredResize(page: Page, session: CDPSession, width: number, height: number) {
    const count = await page.evaluate(() => (window as any).windowLatency.events.length);
    await session.send('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: 1, mobile: false });
    await expect.poll(() => page.evaluate(() => (window as any).windowLatency.events.length)).toBe(count + 1);
    await page.clock.runFor(16);
}
async function expectWindowLatency(page: Page, before: Awaited<ReturnType<typeof counts>>, results = 1, publications = results) {
    const measured = await page.evaluate(() => (window as any).windowLatency);
    const end = measured.events.at(-1);
    expect(measured.events.length).toBeGreaterThan(0);
    expect(measured.requests.filter((r: any) => r.channel === 'results')).toHaveLength(results);
    expect(measured.requests.filter((r: any) => r.channel === 'map')).toHaveLength(1);
    for (const request of measured.requests) expect(request.at - end).toBeLessThanOrEqual(260);
    expect(measured.replacements.length).toBeLessThanOrEqual(1);
    for (const at of measured.replacements) expect(at - end).toBeLessThanOrEqual(260);
    const after = await counts(page);
    expect(after.results).toBe(before.results + results);
    expect(after.map).toBe(before.map + 1);
    expect(after.publications).toBe(before.publications + publications);
    if (results) {
        const visible = await bounds(page), published = await page.evaluate(() => (window as any).fixture.publications.at(-1)[0]);
        for (const corner of ['northEast', 'southWest'] as const) for (const axis of ['lat', 'lng'] as const) {
            expect(published[corner][axis]).toBeCloseTo(visible[corner][axis], 10);
        }
        const params = new URLSearchParams(after.url);
        for (const [key, value] of Object.entries({tl_lat: visible.northEast.lat, tl_lon: visible.northEast.lng, br_lat: visible.southWest.lat, br_lon: visible.southWest.lng})) {
            expect(Number(params.get(key))).toBeCloseTo(value, 10);
        }
    } else expect(after.url).toBe(before.url);
    await page.clock.runFor(800);
    expect(await counts(page)).toEqual(after);
}

async function size(page: Page, width: number, height = 800) {
    const session = await page.context().newCDPSession(page);
    const { windowId } = await session.send('Browser.getWindowForTarget');
    await session.send('Browser.setWindowBounds', { windowId, bounds: { width, height } });
    await expect.poll(() => page.evaluate(() => innerWidth)).toBe(width);
    await session.detach();
}
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
    expect(Number(params.get('tl_lat'))).toBeCloseTo(visible.northEast.lat, 10);
    expect(Number(params.get('br_lon'))).toBeCloseTo(visible.southWest.lng, 10);
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
    for (const plan of [{steps:20,gap:0,expected:1},{steps:4,gap:150,expected:1},{steps:2,gap:260,expected:2}]) {
        test(`${plan.steps} measured window events with ${plan.gap}ms gaps paint every frame and publish ${plan.expected} series`, async ({page}) => {
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
                for (let i=1; i<=plan.steps; i++) {
                    if (i>1 && plan.gap) await page.clock.runFor(plan.gap-16);
                    await session.send('Emulation.setDeviceMetricsOverride', {
                        width:initial.width+i*6,height:initial.height+i*3,deviceScaleFactor:1,mobile:false,
                    });
                    await expect.poll(() => page.evaluate(() => (window as any).resizeEvents.length)).toBe(i);
                    await page.clock.runFor(16);
                }
                await page.clock.runFor(800);
                const after = await quietCounts(page);
                expect(after.results).toBe(before.results+plan.expected);
                expect(after.publications).toBe(before.publications+plan.expected);
                expect(await page.evaluate(() => (window as any).replacements)).toBeLessThanOrEqual(plan.expected);
                const events = await page.evaluate(() => (window as any).resizeEvents as number[]);
                if (plan.gap===150) expect(Math.max(...events.slice(1).map((at,i)=>at-events[i]))).toBeLessThan(200);
                await canvasMatches(page);
            } finally { await session.detach(); }
        });
    }
    for (const plan of ['maximize', 'restore', 'series']) {
        test(`${plan} searches within 260ms of the final window event without another debounce`, async ({page}) => {
            await mount(page, bundle);
            const before = await quietCounts(page), initial = await watchWindowLatency(page);
            const session = await page.context().newCDPSession(page);
            try {
                const steps = plan === 'series' ? 4 : 1, direction = plan === 'restore' ? -1 : 1;
                for (let step = 1; step <= steps; step++) {
                    if (step > 1) await page.clock.runFor(134);
                    await measuredResize(page, session, initial.width + direction * step * 32, initial.height + direction * step * 16);
                }
                await page.clock.runFor(244);
                await expectWindowLatency(page, before);
            } finally { await session.detach(); }
        });
    }
    for (const movement of ['pan', 'zoom']) {
        test(`${movement} keeps the ordinary 200ms page debounce`, async ({page}) => {
            await mount(page, bundle);
            const before = await quietCounts(page);
            await watchWindowLatency(page);
            await page.evaluate(movement => {
                const map = (window as any).realMap;
                if (movement === 'pan') map.panBy([80, 20], {duration: 0});
                else map.zoomIn({duration: 0});
            }, movement);
            await page.clock.runFor(199);
            expect((await counts(page)).results).toBe(before.results);
            await page.clock.runFor(1);
            expect((await counts(page)).results).toBe(before.results + 1);
            expect((await counts(page)).map).toBe(before.map + 1);
            await page.clock.runFor(800);
            expect((await counts(page)).results).toBe(before.results + 1);
        });
    }
    test('a pending pan debounce is consumed by the completed window series', async ({page}) => {
        await mount(page, bundle);
        const before = await quietCounts(page), initial = await watchWindowLatency(page);
        await page.evaluate(() => (window as any).realMap.panBy([80, 20], {duration: 0}));
        await page.clock.runFor(80);
        await page.evaluate(() => { (window as any).windowLatency.replacements = []; });
        const session = await page.context().newCDPSession(page);
        try {
            await measuredResize(page, session, initial.width + 40, initial.height + 20);
            await page.clock.runFor(244);
            await expectWindowLatency(page, before, 1, 2);
        } finally { await session.detach(); }
    });
    test('F11 renders map data within 260ms without a second debounce', async ({page}) => {
        await mount(page, bundle);
        for (const fullscreen of [true, false]) {
            const before = await quietCounts(page);
            await watchWindowLatency(page);
            const session = await page.context().newCDPSession(page);
            try {
                const {windowId} = await session.send('Browser.getWindowForTarget');
                await session.send('Browser.setWindowBounds', {windowId, bounds: {windowState: fullscreen ? 'fullscreen' : 'normal'}});
                await expect.poll(() => page.evaluate(() => matchMedia('(display-mode: fullscreen)').matches)).toBe(fullscreen);
                await expect.poll(() => page.evaluate(() => (window as any).windowLatency.events.length)).toBeGreaterThan(0);
                await page.clock.runFor(260);
                await expectWindowLatency(page, before, 0);
            } finally { await session.detach(); }
            await page.clock.resume();
        }
    });
    test('a hidden mobile map retains a pending pan search through a window resize', async ({page}) => {
        await page.setViewportSize({width: 600, height: 800});
        await mount(page, bundle);
        const before = await quietCounts(page);
        await watchWindowLatency(page);
        await page.evaluate(() => (window as any).realMap.panBy([80, 20], {duration: 0}));
        const area = await page.evaluate(() => (window as any).fixture.publications.at(-1)[0]);
        const pannedUrl = (await counts(page)).url;
        await page.clock.runFor(80);
        await page.evaluate(() => (document.querySelector('.map-view-toggle') as HTMLButtonElement).click());
        await page.clock.runFor(16);
        await expect(page.locator('.map-results')).not.toHaveClass(/show-map/);
        const session = await page.context().newCDPSession(page);
        try {
            await measuredResize(page, session, 620, 780);
            await page.clock.runFor(1000);
            const after = await counts(page);
            expect(after).toEqual({...before, results: before.results + 1, map: before.map + 1, publications: before.publications + 1, url: pannedUrl});
            expect(await page.evaluate(() => (window as any).fixture.requests.filter((r: any) => r.channel === 'results').at(-1).body.bounds)).toEqual(area);
        } finally { await session.detach(); }
    });
    test('leaving fullscreen for a hidden mobile map retains a pending pan search', async ({page}) => {
        await page.setViewportSize({width: 600, height: 800});
        await mount(page, bundle);
        await page.locator('.map-view-toggle').click();
        await windowState(page, true);
        const before = await quietCounts(page);
        await watchWindowLatency(page);
        await page.evaluate(() => (window as any).realMap.panBy([80, 20], {duration: 0}));
        const area = await page.evaluate(() => (window as any).fixture.publications.at(-1)[0]);
        const pannedUrl = (await counts(page)).url;
        await page.clock.runFor(80);
        const session = await page.context().newCDPSession(page);
        try {
            const {windowId} = await session.send('Browser.getWindowForTarget');
            await session.send('Browser.setWindowBounds', {windowId, bounds: {windowState: 'normal'}});
            await expect.poll(() => page.evaluate(() => matchMedia('(display-mode: fullscreen)').matches)).toBe(false);
            await page.clock.runFor(16);
            await expect(page.locator('.map-page')).not.toHaveClass(/fullscreen/);
            expect((await counts(page)).results).toBe(before.results);
            await page.clock.runFor(1400);
            await expect(page.locator('.map-results')).not.toHaveClass(/fullscreen|show-map/);
            const after = await counts(page);
            expect(after).toEqual({...before, results: before.results + 1, map: before.map + 1, publications: before.publications + 1, url: pannedUrl});
            expect(await page.evaluate(() => (window as any).fixture.requests.filter((r: any) => r.channel === 'results').at(-1).body.bounds)).toEqual(area);
        } finally { await session.detach(); }
    });
    test('a visual viewport update before a window event still publishes once',async({page})=>{
        await mount(page,bundle);const before=await quietCounts(page);
        await page.evaluate(()=>{
            Object.defineProperty(window,'innerHeight',{configurable:true,value:650});
            Object.defineProperty(visualViewport,'height',{configurable:true,value:650});
            visualViewport!.dispatchEvent(new Event('resize'));
            requestAnimationFrame(()=>dispatchEvent(new Event('resize')));
        });
        await once(page,before);
    });
    for(const action of ['list-resize','overlay-move','collapse-results'])test(`${action} supersedes a window series without swallowing the next pan`,async({page})=>{
        await mount(page,bundle);
        if(action==='overlay-move'){await page.locator('.expand-results').click();await settle(page);}
        const before=await quietCounts(page);
        await page.evaluate(action=>{
            Object.defineProperty(window,'innerHeight',{configurable:true,value:650});
            Object.defineProperty(visualViewport,'height',{configurable:true,value:650});
            dispatchEvent(new Event('resize'));
            requestAnimationFrame(()=>{
                const target=document.querySelector('.'+action) as HTMLElement;
                if(action==='collapse-results') target.click();
                else {const r=target.getBoundingClientRect(),init={bubbles:true,pointerId:42,pointerType:'mouse',button:0,clientX:r.x+5,clientY:r.y+5};
                    target.setPointerCapture=()=>{};target.hasPointerCapture=()=>false;
                    target.dispatchEvent(new PointerEvent('pointerdown',init));
                    target.dispatchEvent(new PointerEvent('pointerup',init));}
            });
        },action);
        await once(page,before);
        const next=await counts(page);
        await page.evaluate(()=>(window as any).realMap.panBy([80,20],{duration:0}));
        await once(page,next);
    });
    test('browser zoom resizes the drawing buffer and publishes once',async({page})=>{
        await mount(page,bundle);const before=await quietCounts(page),session=await page.context().newCDPSession(page);
        await session.send('Emulation.setDeviceMetricsOverride',{width:1024,height:640,deviceScaleFactor:1.25,mobile:false});
        await once(page,before);
        const sizes=await page.evaluate(()=>{const m=(window as any).realMap,c=m.getCanvas(),r=m.getContainer();return{actual:[c.width,c.height],expected:[Math.floor(r.clientWidth*devicePixelRatio),Math.floor(r.clientHeight*devicePixelRatio)]};});
        expect(sizes.actual).toEqual(sizes.expected);await session.detach();
    });
    test('a held pointer keeps the canvas current through a window change',async({page})=>{
        await mount(page,bundle);const before=await quietCounts(page),r=(await page.locator('.maplibregl-canvas').boundingBox())!;
        await page.mouse.move(r.x+r.width/2,r.y+r.height/2);await page.mouse.down();await recordCanvas(page);
        await size(page,1100);await page.mouse.up();await once(page,before);await canvasMatches(page);
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
            await once(page,before);await canvasMatches(page);await cdp.detach();
        } finally {await context.close();}
    });

});
