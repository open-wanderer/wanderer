import { expect, test, type Page } from '@playwright/test';
import { buildMapPageFixture } from './map-page-fixture';
import { counts, mountEngine, waitForEngine } from './map-engine-fixture';
import { loadFontawesome } from './fontawesome-fixture';

const toggle = (page: Page) => page.locator('.enlarge-map');
const expanded = (page: Page) => page.locator('.map-page.expanded');
export async function assertExpandedGeometry(page: Page) {
    await expect.poll(() => page.locator('#trail-map').evaluate(node => {
        const r = node.getBoundingClientRect();
        return Math.max(Math.abs(r.x), Math.abs(r.y), Math.abs(r.width - innerWidth), Math.abs(r.height - innerHeight));
    })).toBeLessThanOrEqual(1);
    const pixels = await page.evaluate(() => [innerWidth * devicePixelRatio, innerHeight * devicePixelRatio]);
    await expect.poll(() => page.locator('#trail-map .maplibregl-canvas').evaluate((node: HTMLCanvasElement) => [node.width, node.height])).toEqual(pixels);
    expect(await page.evaluate(() => document.documentElement.scrollHeight <= innerHeight && document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    const header = page.locator('.map-search-header');
    await expect(header).toBeVisible();
    const h = (await header.boundingBox())!;
    const panels = [h];
    const list = page.locator('.trail-list');
    if (await list.isVisible()) {
        const l = (await list.boundingBox())!;
        expect(l.y).toBeGreaterThanOrEqual(h.y + h.height + 1);
        expect(l.y + l.height).toBeLessThan((await page.evaluate(() => innerHeight)) - 1);
        panels.push(l);
    }
    await expect(page.locator('.maplibregl-ctrl-bottom-right .maplibregl-ctrl-scale')).toBeVisible();
    for (const control of await page.locator('#trail-map .maplibregl-ctrl').all()) {
        if (!await control.isVisible()) continue;
        const c = (await control.boundingBox())!;
        for (const panel of panels) {
            const overlap = Math.min(panel.x + panel.width, c.x + c.width) - Math.max(panel.x, c.x) > 1
                && Math.min(panel.y + panel.height, c.y + c.height) - Math.max(panel.y, c.y) > 1;
            expect(overlap, `overlay ${JSON.stringify(panel)} intersects map control ${JSON.stringify(c)}`).toBe(false);
        }
    }
}
const camera = (page: Page) => page.evaluate(() => {
    const map = (window as any).realMap;
    return {center: map.getCenter().toArray(), zoom: map.getZoom(), bounds: map.getBounds().toArray()};
});
for (const dev of [true, false]) test.describe(`map enlarge (${dev ? 'development' : 'production'})`, () => {
    let bundle: string;
    test.beforeAll(async () => { bundle = await buildMapPageFixture(dev, true, true, true, true, true); });
    async function open(page: Page) {
        await mountEngine(page, bundle);
        await page.evaluate(() => {
            for (const text of ['Demo', 'ORIGIN']) { const hint = document.createElement('div'); hint.className = 'instance-hint'; hint.textContent = text; document.body.prepend(hint); }
            window.dispatchEvent(new Event('resize'));
        });
        await waitForEngine(page);
    }
    test('toggles chrome, geometry, name and the same focused control with exactly one viewport search', async ({ page }) => {
        await open(page);
        await expect(toggle(page)).toHaveAccessibleName('trail-map-enlarge');
        await expect(toggle(page).locator('i')).toHaveClass(/fa-maximize/);
        await expect(toggle(page)).not.toHaveAttribute('aria-pressed');
        const original = await page.locator('#trail-map').boundingBox();
        const before = await counts(page);
        await toggle(page).evaluate(node => (window as any).originalEnlarge = node);
        await toggle(page).click();
        await expect(expanded(page)).toHaveCount(1);
        await expect(toggle(page)).toBeFocused();
        await expect(toggle(page)).toHaveAccessibleName('trail-map-shrink');
        await expect(toggle(page).locator('i')).toHaveClass(/fa-minimize/);
        expect(await toggle(page).evaluate(node => node === (window as any).originalEnlarge)).toBe(true);
        for (const selector of ['body > nav', 'body > footer', '.instance-hint']) for (const node of await page.locator(selector).all()) await expect(node).toBeHidden();
        await assertExpandedGeometry(page); await waitForEngine(page);
        await expect.poll(() => counts(page).then(c => c.results)).toBe(before.results + 1);
        await page.waitForTimeout(450); expect((await counts(page)).results).toBe(before.results + 1);
        await toggle(page).click(); await expect(expanded(page)).toHaveCount(0);
        await expect(toggle(page)).toBeFocused(); await expect(toggle(page)).toHaveAccessibleName('trail-map-enlarge');
        await expect(toggle(page).locator('i')).toHaveClass(/fa-maximize/);
        await expect(page.locator('body > nav')).toBeVisible(); await expect(page.locator('body > footer')).toBeVisible();
        await expect.poll(() => page.locator('#trail-map').boundingBox().then(r => Math.abs(r!.height - original!.height))).toBeLessThan(2);
    });
    for (const focus of ['button', 'map', 'body']) test(`Escape ends mode from ${focus}`, async ({page}) => {
        await open(page); await toggle(page).click();
        if (focus === 'map') await page.locator('.maplibregl-canvas').focus();
        if (focus === 'body') await page.evaluate(() => (document.activeElement as HTMLElement).blur());
        await page.keyboard.press('Escape'); await expect(expanded(page)).toHaveCount(0);
    });
    test('fields, sort, view, search panel and trail menu retain their own Escape', async ({page}) => {
        await open(page); await toggle(page).click();
        const search = page.getByRole('searchbox'); await search.fill('ridge');
        await page.keyboard.press('Escape'); await expect(expanded(page)).toHaveCount(1);
        for (const suffix of ['sort', 'view']) {
            const control = page.locator(`.map-search-toolbar button[aria-controls$="-${suffix}"]`);
            await control.click(); await expect(control).toHaveAttribute('aria-expanded','true');
            await page.keyboard.press('Escape'); await expect(control).toHaveAttribute('aria-expanded','false'); await expect(expanded(page)).toHaveCount(1);
        }
        await page.getByRole('button', {name: 'trail-search-filter-by', exact: true}).click();
        await page.keyboard.press('Escape'); await expect(expanded(page)).toHaveCount(1);
        await page.locator('.trail-result').first().hover();
        await page.locator('.trail-result input[type=checkbox]').first().check();
        await page.locator('.selection-menu').click();
        await expect(page.locator('.map-page .dropdown .menu')).toBeVisible();
        await page.keyboard.press('Escape'); await expect(expanded(page)).toHaveCount(1);
        // Dropdown Escape itself is the separate dev fix, deliberately absent here.
    });
    test('modal consumes Escape without ending mode and toast remains usable', async ({page}) => {
        await open(page); await toggle(page).click();
        await page.evaluate(() => (window as any).fixture.openDialog());
        const input = page.locator('dialog[open] input'); await input.fill('Retained value');
        await page.locator('dialog[open] button').first().focus();
        await page.keyboard.press('Escape'); await expect(page.locator('dialog[open]')).toHaveCount(0); await expect(expanded(page)).toHaveCount(1);
        await page.evaluate(() => (window as any).fixture.toast());
        await expect(page.getByRole('status')).toContainText('Map toast');
        await page.getByRole('status').getByRole('button').click(); await expect(page.getByRole('status')).toBeEmpty();
    });
    test('snapshot restores only page state and unmount restores navbar', async ({page}) => {
        await open(page); await toggle(page).click();
        const saved = await page.evaluate(() => (window as any).fixture.capture()); expect(saved.expanded).toBe(true);
        await page.evaluate(saved => (window as any).fixture.remount(saved), saved);
        await expect(expanded(page)).toHaveCount(1); await assertExpandedGeometry(page);
        await page.evaluate(() => (window as any).fixture.destroy()); await expect(page.locator('body > nav')).toBeVisible();
    });
    test('390px map retains mode across list and map views', async ({page}) => {
        await page.setViewportSize({width:390,height:844}); await open(page); await toggle(page).click();
        await assertExpandedGeometry(page);
        await page.locator('.map-view-toggle').click(); await expect(expanded(page)).toHaveCount(0); await expect(page.locator('body > nav')).toBeVisible();
        await page.locator('.map-view-toggle').click(); await expect(expanded(page)).toHaveCount(1); await expect(toggle(page)).toBeVisible(); await assertExpandedGeometry(page);
    });
    test('free map area accepts a mouse drag', async ({page}) => {
        await open(page); await toggle(page).click(); await waitForEngine(page);
        const before = await camera(page);
        const size = page.viewportSize()!;
        await page.mouse.move(size.width - 210, size.height / 2);
        await page.mouse.down(); await page.mouse.move(size.width - 350, size.height / 2 + 70, {steps: 8}); await page.mouse.up();
        await expect.poll(() => camera(page).then(value => Math.abs(value.center[0] - before.center[0]))).toBeGreaterThan(.001);
        await waitForEngine(page);
    });
    test('wheel over floating results scrolls only the list', async ({page}) => {
        await open(page);
        await page.evaluate(() => (window as any).fixture.resultItems = Array.from({length:40}, (_,i) => ({id:'item-'+i, author:'tester', name:'Result '+i})));
        await toggle(page).click(); await expect(page.locator('.trail-result')).toHaveCount(40); await waitForEngine(page);
        const before = await camera(page), searches = await counts(page);
        const list = page.locator('.trail-results'), box = (await list.boundingBox())!;
        await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2); await page.mouse.wheel(0,360);
        await expect.poll(() => list.evaluate(node => node.scrollTop)).toBeGreaterThan(0);
        await page.waitForTimeout(450); expect(await camera(page)).toEqual(before); expect(await counts(page)).toEqual(searches);
    });
    test('floating list resize keeps the camera and searches once on release or keyboard change', async ({page}) => {
        await open(page);
        await page.evaluate(() => {
            const map = (window as any).realMap;
            map.addSource('credits', {type:'geojson', attribution:'© OpenMapTiles · © OpenStreetMap contributors', data:{type:'FeatureCollection',features:[]}});
            map.addLayer({id:'credits',type:'circle',source:'credits'});
        });
        await toggle(page).click(); await assertExpandedGeometry(page); await waitForEngine(page);
        const before = await camera(page), searches = await counts(page), map = await page.locator('#trail-map').boundingBox();
        const divider = page.locator('.list-resize'), width = Number(await divider.getAttribute('aria-valuenow'));
        const box = (await divider.boundingBox())!, x = box.x + box.width / 2, y = box.y + box.height * .75;
        await page.mouse.move(x,y); await page.mouse.down(); await page.mouse.move(x+96,y,{steps:5}); await page.mouse.up();
        await expect(divider).toHaveAttribute('aria-valuenow', String(width+96));
        await page.waitForTimeout(450);
        expect(await page.locator('#trail-map').boundingBox()).toEqual(map); expect(await camera(page)).toEqual(before); expect((await counts(page)).results).toBe(searches.results+1);
        await divider.press('End'); await assertExpandedGeometry(page);
        await page.waitForTimeout(450); expect(await camera(page)).toEqual(before); expect((await counts(page)).results).toBe(searches.results+2);
    });
    test('collapsed floating results remain below search and expand normally', async ({page}) => {
        await open(page); await toggle(page).click();
        await page.locator('.result-header .collapse-results').click();
        const restore = page.locator('.collapsed-actions .collapse-results'); await expect(restore).toBeVisible();
        const search = (await page.locator('.map-search-header').boundingBox())!, button = (await restore.boundingBox())!;
        expect(button.y).toBeGreaterThan(search.y + search.height);
        await restore.click(); await expect(page.locator('.trail-list')).toBeVisible(); await assertExpandedGeometry(page);
    });
    test('expanded search uses an overlay even with a saved push layout', async ({page}) => {
        await mountEngine(page,bundle,false,true); await toggle(page).click(); await waitForEngine(page);
        const before = await page.locator('#trail-map').boundingBox();
        await page.getByRole('button',{name:'trail-search-filter-by',exact:true}).click();
        await expect(page.locator('.search-panel')).toBeVisible();
        expect(await page.locator('.search-panel').evaluate(node => getComputedStyle(node).position)).toBe('absolute');
        expect(await page.locator('#trail-map').boundingBox()).toEqual(before);
    });
    test('an open result action dialog stays clickable after switching to phone width', async ({page}) => {
        await open(page); await toggle(page).click();
        await page.locator('.trail-result').first().hover();
        await page.locator('.trail-result input[type=checkbox]').first().check();
        await page.evaluate(() => (window as any).mapResultsFixture.openActionDialog());
        await page.setViewportSize({width:390,height:844});
        const input = page.getByRole('textbox',{name:'Action value',exact:true});
        await input.click({timeout:3000}); await input.fill('Phone dialog stays usable');
        await expect(input).toHaveValue('Phone dialog stays usable');
        await page.keyboard.press('Escape'); await expect(page.locator('dialog[open]')).toHaveCount(0); await expect(expanded(page)).toHaveCount(1);
    });
    test('floating merge status remains clickable in the expanded page', async ({page}) => {
        await open(page); await loadFontawesome(page); await toggle(page).click();
        await page.evaluate(() => (window as any).showMergeStatus());
        const retry = page.getByRole('button',{name:'trail-merge-retry',exact:true}); await expect(retry).toBeVisible();
        await retry.click({timeout:3000});
        await expect(page.getByRole('button',{name:'trail-merge-cancel',exact:true})).toBeVisible();
        await page.getByRole('button',{name:'trail-merge-cancel',exact:true}).click({timeout:3000});
        await page.getByRole('button',{name:'dismiss',exact:true}).click({timeout:3000});
        await expect(page.getByText('Merge source',{exact:true})).toHaveCount(0);
    });
    test('viewport errors keep a visible usable retry above the map', async ({page}) => {
        await open(page);
        const initial = await counts(page);
        await page.evaluate(() => (window as any).fixture.autoResolve = false); await toggle(page).click();
        await expect.poll(() => counts(page).then(value => value.results)).toBe(initial.results+1);
        await page.evaluate(() => (window as any).fixture.requests.filter((r:any)=>r.channel==='results').at(-1).reject(new Error('Offline')));
        const retry = page.getByRole('button',{name:'trail-search-retry',exact:true}); await expect(retry).toBeVisible();
        const before = await counts(page); await retry.click();
        await expect.poll(() => counts(page).then(value=>value.results)).toBe(before.results+1);
    });
    test('Escape cancels a pointer resize without ending enlargement', async ({page}) => {
        await open(page); await toggle(page).click(); await waitForEngine(page);
        const divider = page.locator('.list-resize');
        const width = Number(await divider.getAttribute('aria-valuenow'));
        const saved = await page.evaluate(() => localStorage.getItem('mapResultsLayout'));
        const box = (await divider.boundingBox())!;
        const x = box.x + box.width / 2, y = box.y + box.height * .75;
        await page.mouse.move(x, y); await page.mouse.down();
        await page.mouse.move(x + 96, y, {steps: 5});
        await expect(divider).toBeFocused();
        await expect(divider).toHaveAttribute('aria-valuenow', String(width + 96));
        await expect(page.locator('.map-results')).toHaveClass(/dragging/);
        await page.keyboard.press('Escape'); await page.mouse.up();
        await expect(divider).toHaveAttribute('aria-valuenow', String(width));
        await expect(page.locator('.map-results')).not.toHaveClass(/dragging/);
        expect(await page.evaluate(() => localStorage.getItem('mapResultsLayout'))).toBe(saved);
        await expect(expanded(page)).toHaveCount(1);
    });
    test('default prevented and composing Escape keep mode', async ({page}) => {
        await open(page); await toggle(page).click();
        await page.evaluate(() => window.dispatchEvent(new KeyboardEvent('keydown', {key: 'Escape', isComposing: true})));
        await expect(expanded(page)).toHaveCount(1);
        await expect(toggle(page)).toBeFocused();
        await toggle(page).evaluate(node => {
            node.addEventListener('keydown', event => {
                event.preventDefault();
                node.setAttribute('data-own-escape', 'true');
            }, {once: true});
        });
        await page.keyboard.press('Escape');
        await expect(toggle(page)).toHaveAttribute('data-own-escape', 'true');
        await expect(expanded(page)).toHaveCount(1);
    });
});
