import { expect, test, type Page } from '@playwright/test';
import { buildMapPageFixture } from './map-page-fixture';
import { counts, mountEngine, waitForEngine } from './map-engine-fixture';

// Expected edges come from the painted DOM, independently of the app's insets.
async function geometry(page: Page) {
    return page.evaluate(() => {
        const map = (window as any).realMap;
        const area = map.getContainer().getBoundingClientRect();
        const toolbar = document.querySelector('.map-search-toolbar')!.getBoundingClientRect();
        const list = document.querySelector('.trail-list')?.getBoundingClientRect();
        const expanded = Boolean(document.querySelector('.map-page.expanded'));
        const left = expanded && innerWidth >= 768 && list?.width ? list.right - area.left + 16 : 0;
        const top = expanded ? toolbar.bottom - area.top + 16 : 0;
        const request = (window as any).fixture.requests.filter((r: any) => r.channel === 'map').at(-1).body;
        const point = (value: any) => { const p = map.project([value.lng,value.lat]); return [p.x,p.y]; };
        const results = (window as any).fixture.requests.filter((r: any) => r.channel === 'results').at(-1).body.bounds;
        return { left, top, right: map.getContainer().clientWidth, bottom: map.getContainer().clientHeight,
            ne: point(request.northEast), sw: point(request.southWest),
            resultsNE: point(results.northEast), resultsSW: point(results.southWest),
            bounds: {northEast: request.northEast, southWest: request.southWest},
            camera: {center: map.getCenter().toArray(), zoom: map.getZoom()}, padding: map.getPadding() };
    });
}
async function assertArea(page: Page) {
    await waitForEngine(page);
    const g = await geometry(page);
    for (const [actual, expected] of [[g.ne, [g.right,g.top]], [g.sw,[g.left,g.bottom]],
        [g.resultsNE,[g.right,g.top]], [g.resultsSW,[g.left,g.bottom]]]) {
        expect(Math.abs(actual[0]-expected[0]), 'published longitude edge in pixels').toBeLessThanOrEqual(1);
        expect(Math.abs(actual[1]-expected[1]), 'published latitude edge in pixels').toBeLessThanOrEqual(1);
    }
    expect(g.padding).toEqual({left:0,top:0,right:0,bottom:0});
    return g;
}
async function oneSearch(page: Page, action: () => Promise<unknown>) {
    const before = await counts(page);
    await action();
    await expect.poll(() => counts(page).then(c => c.results)).toBe(before.results+1);
    await waitForEngine(page);
    await page.waitForTimeout(400);
    const after = await counts(page);
    expect(after.results).toBe(before.results+1);
    expect(after.map).toBe(before.map+1);
}
async function centered(page: Page, location = {lat:47,lon:8}) {
    const g = await assertArea(page);
    const point = await page.evaluate(p => { const pixel = (window as any).realMap.project([p.lon,p.lat]); return [pixel.x,pixel.y]; }, location);
    expect(Math.abs(point[0]-(g.left+g.right)/2)).toBeLessThanOrEqual(2);
    expect(Math.abs(point[1]-(g.top+g.bottom)/2)).toBeLessThanOrEqual(2);
}
async function contained(page: Page, coordinates: number[][], margin: number) {
    const g = await assertArea(page);
    const pixels = await page.evaluate(points => points.map(p => { const pixel = (window as any).realMap.project(p); return [pixel.x,pixel.y]; }), coordinates);
    for (const [x,y] of pixels) {
        expect(x).toBeGreaterThanOrEqual(g.left+margin-1); expect(x).toBeLessThanOrEqual(g.right-margin+1);
        expect(y).toBeGreaterThanOrEqual(g.top+margin-1); expect(y).toBeLessThanOrEqual(g.bottom-margin+1);
    }
}
for (const dev of [true,false]) test.describe(`visible map area (${dev ? 'development' : 'production'})`, () => {
    let bundle: string;
    test.use({viewport:{width:1280,height:720}});
    test.beforeAll(async () => { bundle = await buildMapPageFixture(dev,true,true,true,true,true); });
    async function open(page: Page) {
        await mountEngine(page,bundle);
        await oneSearch(page, () => page.locator('.enlarge-map').click());
        return assertArea(page);
    }
    test('publishes the free rectangle and keeps the normal full rectangle and camera', async ({page}) => {
        await mountEngine(page,bundle);
        const normal = await assertArea(page); expect(normal.left).toBe(0); expect(normal.top).toBe(0);
        await oneSearch(page, () => page.locator('.enlarge-map').click());
        const expanded = await assertArea(page); expect(expanded.left).toBeGreaterThan(400); expect(expanded.top).toBeGreaterThan(50);
        expect(expanded.camera).toEqual(normal.camera);
        await oneSearch(page, () => page.locator('.enlarge-map').click());
        expect((await assertArea(page)).camera).toEqual(normal.camera);
    });
    test('collapse and expand each publish exactly one new search', async ({page}) => {
        const initial = await open(page);
        await oneSearch(page, () => page.locator('.result-header .collapse-results').click());
        const collapsed = await assertArea(page); expect(collapsed.left).toBe(0); expect(collapsed.camera).toEqual(initial.camera);
        await oneSearch(page, () => page.locator('.collapsed-actions .collapse-results').click());
        expect((await assertArea(page)).left).toBe(initial.left);
    });
    test('drag publishes only on release and keyboard changes use the page debounce', async ({page}) => {
        const initial = await open(page), before = await counts(page);
        const divider = page.locator('.list-resize'), box = (await divider.boundingBox())!;
        const x = box.x+box.width/2, y = box.y+box.height*.75;
        await page.mouse.move(x,y); await page.mouse.down(); await page.mouse.move(x+96,y,{steps:8});
        await page.waitForTimeout(450); expect(await counts(page)).toEqual(before);
        await oneSearch(page, () => page.mouse.up());
        const dragged = await assertArea(page); expect(dragged.left).toBe(initial.left+96); expect(dragged.camera).toEqual(initial.camera);
        await oneSearch(page, async () => { await divider.focus(); await page.keyboard.press('ArrowRight'); await page.keyboard.press('ArrowRight'); await page.keyboard.press('End'); });
        const widest = await assertArea(page); expect(widest.left).toBeGreaterThan(dragged.left); expect(widest.camera).toEqual(initial.camera);
    });
    test('filter overlay and toolbar height do not search; the next publication measures the toolbar alone', async ({page}) => {
        const initial = await open(page), before = await counts(page);
        await page.getByRole('button',{name:'trail-search-filter-by',exact:true}).click();
        await expect(page.locator('.search-panel')).toBeVisible();
        await page.waitForTimeout(450); expect(await counts(page)).toEqual(before); await assertArea(page);
        await page.keyboard.press('Escape');
        await page.locator('.map-search-toolbar').evaluate(node => (node as HTMLElement).style.minHeight = '96px');
        await page.waitForTimeout(450); expect(await counts(page)).toEqual(before);
        expect((await geometry(page)).bounds).toEqual(initial.bounds);
        await oneSearch(page, () => page.locator('.result-header .collapse-results').click());
        expect((await assertArea(page)).top).toBeGreaterThan(initial.top);
    });
    test('place radius and explicit bounds fit entirely inside the free rectangle', async ({page}) => {
        await open(page);
        const near = {lat:47,lon:8,radius:10000,name:'Radius place'};
        await oneSearch(page, () => page.evaluate(value => (window as any).fixture.applyNear(value), near));
        // Sample the full geodesic circle, rather than just the target center.
        const points = Array.from({length:72}, (_,i) => {
            const bearing=i*Math.PI/36, d=near.radius/6371008.8, lat=near.lat*Math.PI/180, lon=near.lon*Math.PI/180;
            const y=Math.asin(Math.sin(lat)*Math.cos(d)+Math.cos(lat)*Math.sin(d)*Math.cos(bearing));
            const x=lon+Math.atan2(Math.sin(bearing)*Math.sin(d)*Math.cos(lat),Math.cos(d)-Math.sin(lat)*Math.sin(y));
            return [x*180/Math.PI,y*180/Math.PI];
        });
        await contained(page,points,32);
        await oneSearch(page, () => page.evaluate(() => (window as any).fixture.focusArea({southWest:{lat:46,lng:7},northEast:{lat:47,lng:9}})));
        await contained(page,[[7,46],[9,47]],0);
    });
    test('location selection stays centered with a normal and maximally wide list', async ({page}) => {
        await open(page);
        await oneSearch(page, () => page.evaluate(() => (window as any).fixture.focusLocation({lat:47,lon:8})));
        await centered(page);
        await oneSearch(page, () => page.locator('.list-resize').press('End'));
        await oneSearch(page, () => page.evaluate(() => (window as any).fixture.focusLocation({lat:47,lon:8})));
        await centered(page);
    });
    for (const mode of ['saved','geolocation','overview'] as const) test(`${mode} reset uses the free rectangle, also at maximum list width`, async ({page,context}) => {
        if (mode==='geolocation') { await context.grantPermissions(['geolocation']); await context.setGeolocation({latitude:47,longitude:8}); }
        await page.addInitScript(value => {
            (window as any).fixtureSettings = value==='saved' ? {mapFocus:'location',location:{lat:47,lon:8}} : {mapFocus:value==='overview'?'trails':'location'};
            (window as any).fixtureBoundingBox = {has_trails:true,min_lon:7,max_lon:9,min_lat:46,max_lat:47};
        },mode);
        await open(page);
        for (const widest of [false,true]) {
            if (widest) await oneSearch(page, () => page.locator('.list-resize').press('End'));
            await page.getByRole('button',{name:mode==='saved'?'trail-map-reset-location':'trail-map-reset-view',exact:true}).click();
            if (mode==='overview') {
                await expect.poll(() => geometry(page).then(g => g.camera.zoom)).toBeGreaterThan(0);
                await contained(page,[[7,46],[9,47]],32);
            } else {
                await expect.poll(() => page.evaluate(() => Math.abs((window as any).realMap.getZoom()-12))).toBeLessThan(1e-8);
                await centered(page);
            }
        }
    });
    for (const mode of ['pending','initial'] as const) test(`${mode} location centers when the map initializes in expanded mode`, async ({page}) => {
        await mountEngine(page,bundle,true,false,'?lat=47&lon=8');
        await page.evaluate(() => (window as any).fixture.restore({...((window as any).fixture.capture()),expanded:true,viewport:undefined,camera:undefined}));
        await expect(page.locator('.map-page.expanded')).toBeVisible();
        if (mode==='pending') await page.evaluate(() => (window as any).fixture.focusLocation({lat:47,lon:8}));
        await page.evaluate(() => { (window as any).fixture.autoResolve=true; (window as any).finishMapLoad(); });
        await centered(page);
    });
    test('snapshot remount keeps the exact camera, area and a single search', async ({page}) => {
        const initial = await open(page);
        for (const mode of ['popstate','sync']) {
            await oneSearch(page, () => page.evaluate(mode => (window as any).fixture.remount((window as any).fixture.capture(),mode),mode));
            const restored = await assertArea(page); expect(restored.camera).toEqual(initial.camera);
            expect(restored.ne).toEqual(initial.ne); expect(restored.sw).toEqual(initial.sw);
        }
    });
    test('390px map mode only excludes the toolbar', async ({page}) => {
        await page.setViewportSize({width:390,height:844}); await open(page);
        const g=await assertArea(page); expect(g.left).toBe(0); expect(g.top).toBeGreaterThan(50);
    });
});
