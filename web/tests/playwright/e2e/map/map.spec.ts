import { devices } from '@playwright/test';
import { expectActiveNavigation } from '../navigation_helpers';
import { expect, test, type Page } from '../../fixtures';
import { mockEmptyGeocoding } from '../search_helpers';

const searchInput = (page: Page) => page.locator('[data-search-root] input[type="search"]').first();
const viewButton = (page: Page) => page.locator('.map-search-toolbar .control-button[aria-controls$="-view"]');
const isTrailSearch = (url: string) => new URL(url).pathname === '/api/v1/search/trails';

// Keep real routing, snapshots, MapLibre and search controls.
// A local empty basemap and empty search replies keep these tests independent of
// external tile servers and the trails created by other suites.
test.beforeEach(async ({ context }) => {

    await mockEmptyGeocoding(context);
    await context.route('**/styles/ofm.json', route => route.fulfill({ json: {
        version: 8, sources: {}, layers: [{ id: 'background', type: 'background', paint: { 'background-color': '#dae5df' } }],
    } }));
    await context.route('**/api/v1/search/trails', route => {
        const { options } = route.request().postDataJSON();
        return route.fulfill({ json: { hits: [], totalHits: 0, totalPages: 0, page: options.page, hitsPerPage: options.hitsPerPage } });
    });
    await context.route('**/api/v1/search/trails/cluster', route => route.fulfill({ json: {
        type: 'FeatureCollection', features: [], totalHits: 0,
    } }));
});

async function openMap(page: Page) {
    const loaded = page.waitForResponse(response => isTrailSearch(response.url()));
    await page.goto('/map?lat=46.95&lon=7.44');
    await loaded;
    await expect(page.locator('.map-results .maplibregl-canvas')).toBeVisible();
}
async function settledMapLayout(page: Page) {
    await expect(page.locator('#trail-map .maplibregl-ctrl-top-right .reset-map-view')).toBeVisible();
    await expect(page.locator('.map-results')).toHaveAttribute('data-layout-ready', 'true');
    await expect.poll(() => page.locator('#trail-map').evaluate(element => {
        const height = (corner: string) => element.querySelector<HTMLElement>('.maplibregl-ctrl-' + corner)?.offsetHeight ?? 0;
        const minimum = Math.max(160, Math.ceil(Math.max(height('top-left') + height('bottom-left'), height('top-right') + height('bottom-right')) + 14));
        const applied = parseFloat((element.closest<HTMLElement>('.map-results')!).style.getPropertyValue('--overlay-min-height'));
        return applied === minimum;
    })).toBe(true);
}

async function enterSearch(page: Page, query: string) {
    const searched = page.waitForRequest(request => isTrailSearch(request.url()) && request.postDataJSON()?.q === query);
    await searchInput(page).fill(query);
    await searched;
    await expect(page.locator('[data-search-root] .search-panel')).toBeVisible();
    await searchInput(page).press('Escape');
    await expect(page.locator('[data-search-root] .search-panel')).toHaveCount(0);
}

async function mapPixel(page: Page) {
    const map = (await page.locator('#trail-map').boundingBox())!;
    const screenshot = (await page.screenshot()).toString('base64');
    return page.evaluate(async ({ screenshot, x, y }) => {
        const image = new Image();
        image.src = 'data:image/png;base64,' + screenshot;
        await image.decode();
        const canvas = document.createElement('canvas');
        canvas.width = image.width;
        canvas.height = image.height;
        const context = canvas.getContext('2d')!;
        context.drawImage(image, 0, 0);
        return [...context.getImageData(x, y, 1, 1).data];
    }, { screenshot, x: Math.round(map.x + map.width / 2), y: Math.round(map.y + map.height / 2) });
}

async function expectUniqueSelectionIDs(page: Page) {
    const checkboxes = page.locator('.trail-results input[type="checkbox"]');
    expect(await checkboxes.count()).toBeGreaterThanOrEqual(2);
    const ids = await checkboxes.evaluateAll(elements => elements.map(element => element.id).filter(Boolean));
    expect(new Set(ids).size).toBe(ids.length);
}

test('keeps separate trail and map navigation entries without redirecting the list page', async ({ page }) => {
    const response = await page.request.get('/trails', { maxRedirects: 0 });
    expect(response.status()).toBe(200);
    expect(response.headers().location).toBeUndefined();
    await page.goto('/trails');
    const nav = page.locator('#nav-bar-links');
    await expectActiveNavigation(page, '/trails');
    await page.reload();
    await expectActiveNavigation(page, '/trails');
    await expect(page.locator('.result-content')).toHaveAttribute('aria-busy', 'false');
    await expect(nav.locator('a')).toHaveCount(4);
    await expect(nav.locator('a').nth(1)).toHaveAttribute('href', '/trails');
    await expect(nav.locator('a').nth(2)).toHaveAttribute('href', '/map');
    await expect(nav.getByRole('link', { name: 'Trails', exact: true })).toHaveAttribute('href', '/trails');
    const mapLink = nav.getByRole('link', { name: 'Map', exact: true });
    await expect(mapLink).toHaveAttribute('href', '/map');
    await expect(page.locator('.maplibregl-canvas')).toHaveCount(0);
    await expect.poll(async () => {
        const indicator = (await nav.locator(':scope > div').boundingBox())!;
        const active = (await nav.locator('a[href="/trails"]').boundingBox())!;
        return Math.round(indicator.x - active.x);
    }).toBe(-8);
    await page.goto('/map');
    await expectActiveNavigation(page, '/map');
    await page.reload();
    await expectActiveNavigation(page, '/map');
    await nav.getByRole('link', { name: 'Trails', exact: true }).click();
    await expectActiveNavigation(page, '/trails');
    await mapLink.click();
    await expect(page.locator('.maplibregl-canvas')).toBeVisible();
    await expect(page).toHaveTitle('Map | wanderer');
    await expect.poll(async () => {
        const indicator = (await nav.locator(':scope > div').boundingBox())!;
        const active = (await mapLink.boundingBox())!;
        return Math.round(indicator.x - active.x);
    }).toBe(-8);
    await nav.getByRole('link', { name: 'Trails', exact: true }).click();
    await expect(page).toHaveURL('/trails');
    await expect.poll(async () => {
        const indicator = (await nav.locator(':scope > div').boundingBox())!;
        const active = (await nav.locator('a[href="/trails"]').boundingBox())!;
        return Math.round(indicator.x - active.x);
    }).toBe(-8);
    await expect(page.locator('.maplibregl-canvas')).toHaveCount(0);
});

test('keeps trail filters tab-local across navigation and reload', async ({ page, context }) => {
    await openMap(page);
    await enterSearch(page, 'Current tab');
    expect(await page.evaluate(() => localStorage.getItem('trailListFilter'))).toBeNull();
    await page.locator('#nav-bar-links a[href="/"]').click();
    await expect(page).toHaveURL('/');
    await page.goBack();
    await expect(searchInput(page)).toHaveValue('Current tab');
    await expect(page.locator('#trail-map[aria-busy="false"] .maplibregl-canvas')).toBeVisible();
    await page.reload();
    await expect(searchInput(page)).toHaveValue('Current tab');
    const other = await context.newPage();
    await openMap(other);
    await expect(searchInput(other)).toHaveValue('');
    await enterSearch(other, 'Other tab');
    await page.reload();
    await expect(searchInput(page)).toHaveValue('Current tab');
    await other.reload();
    await expect(searchInput(other)).toHaveValue('Other tab');
    await other.close();
});

test('restores the fresh default map view without resetting search and retains it after reload', async ({ page, context }) => {
    const area = (url: string) => {
        const params = new URL(url).searchParams;
        return ['tl_lat', 'tl_lon', 'br_lat', 'br_lon'].map(key => params.has(key) ? Math.round(Number(params.get(key)) * 1e6) / 1e6 : null);
    };
    const fresh = await context.newPage();
    await fresh.goto('/map');
    await expect(fresh.locator('#trail-map .reset-map-view')).toBeVisible();
    await expect.poll(() => area(fresh.url()).every(value => value !== null)).toBe(true);
    const expected = area(fresh.url());
    await fresh.close();
    await openMap(page);
    await enterSearch(page, 'Keep this search');
    await page.locator('#trail-map .maplibregl-ctrl-zoom-in').click();
    const map = (await page.locator('#trail-map').boundingBox())!;
    await page.mouse.move(map.x + map.width / 2, map.y + map.height / 2);
    await page.mouse.down();
    await page.mouse.move(map.x + map.width / 2 + 100, map.y + map.height / 2 + 50, { steps: 4 });
    await page.mouse.up();
    expect(area(page.url())).not.toEqual(expected);
    const search = page.waitForRequest(request => isTrailSearch(request.url()) && request.postDataJSON().q === 'Keep this search');
    await page.getByRole('button', { name: 'Restore map overview', exact: true }).press('Enter');
    await search;
    await expect.poll(() => area(page.url())).toEqual(expected);
    await expect(searchInput(page)).toHaveValue('Keep this search');
    const buttons = page.locator('#trail-map .maplibregl-ctrl-top-right button');
    await expect(buttons.last()).toHaveClass(/reset-map-view/);
    await page.reload();
    await expect(searchInput(page)).toHaveValue('Keep this search');
    await expect.poll(() => area(page.url())).toEqual(expected);
});

test('offers cards, list and table in the map view menu and remembers the choice', async ({ page, context }) => {
    let searches = 0;
    await context.route('**/api/v1/search/trails', route => {
        ++searches;
        const { options } = route.request().postDataJSON();
        return route.fulfill({ json: {
            hits: [{ id: 'map-view-trail', name: 'Map view fixture', author: 'fixtureauthor00', author_name: 'Fixture',
                author_avatar: '/favicon.png', public: true, completed: false, tags: [],
                created: 1710000000, date: 1710000000, distance: 2400, duration: 3600,
                elevation_gain: 100, elevation_loss: 100, difficulty: 0, location: 'Bern',
                _geo: { lat: 46.95, lng: 7.44 } }],
            totalHits: 1, totalPages: 1, page: options.page, hitsPerPage: options.hitsPerPage,
        } });
    });
    await openMap(page);
    await expect(page.locator('.trail-list .trail-card')).toBeVisible();
    const loaded = searches;
    await page.locator('.maplibregl-canvas').evaluate(element => (window as any).retainedMapCanvas = element);
    await viewButton(page).click();
    const menu = page.locator('.view-menu');
    await expect(menu.locator('a')).toHaveCount(0);
    await expect(menu.locator('input[value="cards"]')).toBeChecked();
    await expect(menu.locator('input[value="map"]')).toHaveCount(0);
    await expect(menu.locator('input[value="top"]')).toHaveCount(0);
    await expect(menu.locator('input[value="left"], input[value="right"]')).toHaveCount(0);
    await menu.locator('input[value="list"]').check();
    await expect(page.locator('.trail-list .result-items .trail-result > a > li')).toBeVisible();
    await expect(page.locator('.trail-list .trail-card')).toHaveCount(0);
    expect(await page.locator('.maplibregl-canvas').evaluate(element => element === (window as any).retainedMapCanvas)).toBe(true);
    await page.waitForTimeout(300); // Negative check spans the viewport debounce.
    expect(searches).toBe(loaded);
    await page.keyboard.press('Escape');
    await page.getByRole('separator').press('Home');
    await expect.poll(() => page.locator('.trail-results').evaluate(element => element.scrollWidth <= element.clientWidth)).toBe(true);
    await page.reload();
    await expect(page.locator('.trail-list .result-items .trail-result > a > li')).toBeVisible();
    await viewButton(page).click();
    await expect(page.locator('.view-menu input[value="list"]')).toBeChecked();
    await page.locator('.view-menu input[value="cards"]').check();
    await page.keyboard.press('Escape');
    await expect(page.locator('.trail-list .trail-card')).toBeVisible();
    await expect.poll(() => page.locator('.trail-results').evaluate(element => element.scrollWidth <= element.clientWidth)).toBe(true);
    await viewButton(page).click();
    await page.locator('.view-menu input[value="table"]').check();
    await page.keyboard.press('Escape');
    await expect(page.locator('.trail-list table')).toBeVisible();
    await expect(page.locator('.trail-list tbody tr')).toHaveCount(1);
    await expect(page.locator('.trail-list input[type="checkbox"]')).toHaveCount(2);
    const ascending = page.waitForResponse(response => isTrailSearch(response.url()) && response.request().postDataJSON()?.options.sort?.[0] === 'name:asc');
    await page.getByRole('columnheader', { name: /^Name/ }).click();
    await ascending;
    const descending = page.waitForResponse(response => isTrailSearch(response.url()) && response.request().postDataJSON()?.options.sort?.[0] === 'name:desc');
    await page.getByRole('columnheader', { name: /^Name/ }).click();
    await descending;
    await page.reload();
    await expect(page.locator('.trail-list table')).toBeVisible();
    await viewButton(page).click();
    await expect(page.locator('.view-menu input[value="table"]')).toBeChecked();
});

test('restores multi-select, loaded-result selection and bulk edits in every result view', async ({ page, context }) => {
    await context.addInitScript(() => localStorage.setItem('mapResultsLayout', JSON.stringify({ expanded: true })));
    const authCookie = (await context.cookies()).find(cookie => cookie.name === 'pb_auth')!;
    const actor = JSON.parse(decodeURIComponent(authCookie.value)).record.actor;
    expect(Boolean(actor)).toBe(true);
    let empty = false;
    let searches = 0;
    await context.route('**/api/v1/search/trails', route => {
        ++searches;
        const { options } = route.request().postDataJSON();
        const start = options.page === 1 ? 0 : 2;
        return route.fulfill({ json: {
            hits: empty ? [] : Array.from({ length: 2 }, (_, offset) => ({
                id: 'selection-' + (start + offset), name: 'Selection trail ' + (start + offset),
                author: actor, author_name: 'Test', author_avatar: '/favicon.png',
                public: true, completed: false, tags: [], difficulty: 0,
                created: 1710000000, date: 1710000000, distance: 2400, duration: 3600,
                elevation_gain: 100, elevation_loss: 100, location: 'Bern',
                _geo: { lat: 46.95, lng: 7.44 },
            })),
            totalHits: empty ? 0 : 4, totalPages: empty ? 0 : 2, page: options.page, hitsPerPage: options.hitsPerPage,
        } });
    });
    // Hovering also preloads the linked detail route through SvelteKit.
    await context.route(/\/api\/v1\/trail\/selection-\d+\?/, route => route.fulfill({ json: {
        id: new URL(route.request().url()).pathname.split('/').at(-1),
        name: 'Selection trail', author: actor, photos: [], tags: [], public: true,
        completed: false, difficulty: 'easy', expand: {},
    } }));
    const updates: string[] = [];
    await context.route('**/api/v1/trail/form/selection-*', route => {
        const id = new URL(route.request().url()).pathname.split('/').at(-1)!;
        expect(route.request().postData()).toContain('difficult');
        updates.push(id);
        return route.fulfill({ json: { id, difficulty: 'difficult', author: actor, photos: [], tags: [] } });
    });
    await openMap(page);
    const cards = page.locator('.trail-list .trail-card');
    await cards.first().hover();
    await expect(cards.first().locator('..').locator('..').getByRole('checkbox')).toHaveAttribute('aria-label', 'Select Selection trail 0');
    await cards.first().locator('..').locator('..').getByRole('checkbox').check();
    await cards.nth(1).getByText('Selection trail 1', { exact: true }).click();
    await expect(page.locator('.selection-count')).toHaveText('2 selected');
    await expect(page.locator('.result-items .trail-result > a').first()).toHaveAccessibleName(/Selection trail 0.*Bern/);
    await expect(page.locator('.result-items .trail-result > a').first()).not.toHaveAccessibleName(/\bSelect\b|auswählen/);
    await expectUniqueSelectionIDs(page);
    expect(new URL(page.url()).pathname).toBe('/map');
    await page.getByRole('button', { name: 'Load more', exact: true }).click();
    await expect(cards).toHaveCount(4);
    await expect(page.locator('.selection-count')).toHaveText('2 selected');
    await viewButton(page).click();
    await page.locator('.view-menu input[value="list"]').check();
    await page.keyboard.press('Escape');
    await expect(page.locator('.trail-results input:checked')).toHaveCount(2);
    await expect(page.locator('.result-items .trail-result > a').first()).not.toHaveAccessibleName(/\bSelect\b|auswählen/);
    await expectUniqueSelectionIDs(page);
    await expect(page.locator('.trail-results input:checked').first()).toHaveAccessibleName('Select Selection trail 0');
    await expect(page.locator('.result-items .trail-result > a').first()).toHaveAccessibleName(/Selection trail 0.*Bern/);
    const firstResult = page.locator('.result-items .trail-result').first();
    await firstResult.getByRole('checkbox').uncheck();
    await page.locator('#trail-map').hover();
    await expect(firstResult.locator('li')).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)');
    await expect(page.locator('.result-items li').nth(1)).not.toHaveCSS('background-color', 'rgba(0, 0, 0, 0)');
    await page.getByRole('button', { name: 'Clear selection', exact: true }).click();
    await expect(firstResult.locator('a')).toBeFocused();
    await page.locator('#trail-map').hover();
    for (const row of await page.locator('.result-items li').all()) {
        await expect(row).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)');
    }
    await firstResult.getByRole('checkbox').check();
    await page.locator('.result-items .trail-result > a').nth(1).click();
    await expect(page.locator('.selection-count')).toHaveText('2 selected');
    await viewButton(page).click();
    await page.locator('.view-menu input[value="table"]').check();
    await page.keyboard.press('Escape');
    await expect(page.locator('tbody input:checked')).toHaveCount(2);
    await expect(page.locator('tbody input[type="checkbox"]').first()).toHaveAttribute('aria-label', 'Select Selection trail 0');
    await expect(page.locator('thead input[type="checkbox"]')).toHaveAttribute('aria-label', 'Select all loaded trails');
    await page.locator('thead input[type="checkbox"]').check();
    await expect(page.locator('.selection-count')).toHaveText('4 selected');
    await page.locator('tbody tr').first().click();
    await expect(page.locator('.selection-count')).toHaveText('3 selected');
    await page.getByRole('button', { name: 'Clear selection', exact: true }).click();
    await expect(page.locator('.selection-toolbar')).toHaveCount(0);
    await page.locator('thead input[type="checkbox"]').check();
    await viewButton(page).click();
    await page.locator('.view-menu input[value="cards"]').check();
    await page.keyboard.press('Escape');
    if (await page.locator('.map-results').evaluate(element => element.classList.contains('results-expanded'))) {
        await page.locator('.overview-actions .expand-results').click();
    }
    await expect(page.locator('.map-results')).not.toHaveClass(/results-expanded/);
    await settledMapLayout(page);
    const originalMap = (await page.locator('#trail-map').boundingBox())!;
    const loaded = searches;
    await page.getByRole('button', { name: 'Actions for selected trails' }).click();
    await page.locator('.selection-toolbar .menu').getByText('Adjust', { exact: true }).click();
    const dialog = page.locator('dialog[open]');
    await expect(dialog).toBeVisible();
    const selected = await page.locator('.selection-count').textContent();
    const lockedArea = new URL(page.url()).search;
    const dialogPosition = await dialog.boundingBox();
    // Some browser/OS shortcuts cannot be cancelled. The dialog still owns
    // focus and its search area when the browser changes state underneath it.

    await settledMapLayout(page);
    await expect(dialog).toBeVisible();
    expect(await dialog.boundingBox()).toEqual(dialogPosition);
    expect(await dialog.evaluate(element => element.contains(document.activeElement))).toBe(true);

    await settledMapLayout(page);
    expect(await dialog.boundingBox()).toEqual(dialogPosition);
    await page.locator('.map-results-container').evaluate(element => element.style.width = 'calc(100% + 20px)');
    await expect.poll(() => page.locator('#trail-map').boundingBox().then(box => box!.width)).toBeGreaterThan(originalMap.width);
    await settledMapLayout(page);
    await expect(dialog).toBeVisible();
    await expect(page.locator('.selection-count')).toHaveText(selected!);
    expect(new URL(page.url()).search).toBe(lockedArea);
    await page.waitForTimeout(300); // Negative check must outlast the 200 ms viewport debounce.
    expect(searches).toBe(loaded);
    await page.locator('.map-results-container').evaluate(element => element.style.width = '');
    await dialog.locator('label').filter({ hasText: 'Difficulty' }).locator('input').check();
    await dialog.locator('select').selectOption('difficult');
    await dialog.locator('.btn-primary').click();
    await expect(page.locator('dialog[open]')).toHaveCount(0);
    await expect(page.locator('.selection-toolbar')).toHaveCount(0);
    expect(updates.sort()).toEqual(['selection-0', 'selection-1', 'selection-2', 'selection-3']);
    await expect(cards.filter({ hasText: 'Difficult' })).toHaveCount(4);
    await expect(cards.first().locator('xpath=..')).toHaveAttribute('href', '/map/trail/@Test/selection-0');
    await page.waitForTimeout(300); // The index may still contain pre-edit records.
    expect(searches).toBe(loaded);
    await cards.first().hover();
    await cards.first().locator('..').locator('..').getByRole('checkbox').check();

    await expect(page.locator('.selection-count')).toHaveText('1 selected');
    empty = true;
    await enterSearch(page, 'No matching trails');
    await expect(page.locator('.selection-toolbar')).toHaveCount(0);
    await viewButton(page).click();
    await page.locator('.view-menu input[value="table"]').check();
    await page.keyboard.press('Escape');
    await expect(page.locator('thead input[type="checkbox"]')).not.toBeChecked();
});

test('keeps map search above the results despite a shared sidebar preference', async ({ page, context }) => {
    await context.addInitScript(() => localStorage.setItem('searchLayout', JSON.stringify({ position: 'left', expansion: 'push' })));
    await openMap(page);
    await enterSearch(page, 'Keep filters');
    await expect(page.locator('.filter-sidebar')).toHaveCount(0);
    await expect(page.locator('.map-search-toolbar [data-search-root]')).toHaveClass(/in-flow/);
    await viewButton(page).click();
    await expect(page.locator('.view-menu input[value="left"], .view-menu input[value="right"]')).toHaveCount(0);
    await page.keyboard.press('Escape');


    await expect(searchInput(page)).toHaveValue('Keep filters');
    expect(await page.evaluate(() => JSON.parse(localStorage.getItem('searchLayout')!))).toMatchObject({ position: 'left' });
});

test('keeps the map painted while resizing split results and searches only after release', async ({ page }) => {
    let searches = 0;
    page.on('request', request => { if (isTrailSearch(request.url())) ++searches; });
    await openMap(page);
    const loaded = searches;
    const area = new URL(page.url()).search;
    await page.evaluate(() => {
        const map = document.querySelector<HTMLElement>('#trail-map')!;
        const canvas = map.querySelector<HTMLCanvasElement>('.maplibregl-canvas')!;
        const gl = canvas.getContext('webgl2')!;
        let width = map.clientWidth;
        let height = map.clientHeight;
        (window as any).splitResizeFrames = [];
        // Observe after the component's resize handler, before browser paint.
        // A screenshot alone misses a blank frame repainted on the next RAF.
        new ResizeObserver(([entry]) => {
            if (entry.contentRect.width === width && entry.contentRect.height === height) return;
            width = entry.contentRect.width;
            height = entry.contentRect.height;
            const pixel = new Uint8Array(4);
            gl.readPixels(Math.floor(gl.drawingBufferWidth / 2), Math.floor(gl.drawingBufferHeight / 2), 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, pixel);
            (window as any).splitResizeFrames.push([...pixel]);
        }).observe(map);
    });
    const divider = page.getByRole('separator');
    const edge = (await divider.boundingBox())!;
    const width = Number(await divider.getAttribute('aria-valuenow'));
    const x = edge.x + edge.width / 2;
    const y = edge.y + edge.height / 2;
    await page.mouse.move(x, y);
    await page.mouse.down();
    for (const delta of [48, 96, 144, 96]) {
        const frames = await page.evaluate(() => (window as any).splitResizeFrames.length);
        await page.mouse.move(x + delta, y, { steps: 3 });
        await expect(divider).toHaveAttribute('aria-valuenow', String(width + delta));
        await expect.poll(() => page.evaluate(() => (window as any).splitResizeFrames.length)).toBeGreaterThan(frames);
        expect(await page.evaluate(() => (window as any).splitResizeFrames.at(-1))).toEqual([218, 229, 223, 255]);
        await page.waitForTimeout(300); // Negative check spans the viewport debounce.
        expect(searches).toBe(loaded);
        expect(new URL(page.url()).search).toBe(area);
    }
    const searched = page.waitForRequest(request => isTrailSearch(request.url()));
    await page.mouse.up();
    await searched;
    expect(await mapPixel(page)).toEqual([218, 229, 223, 255]);
});

test.describe('small-map rendering at fractional pixel density', () => {
    test.use({ deviceScaleFactor: 1.5 });

    test('fits the final canvas size and requests clusters at its zoom without changing results', async ({ page, context }) => {
        let searches = 0;
        const clusterZooms: number[] = [];
        page.on('request', request => { if (isTrailSearch(request.url())) ++searches; });
        await context.route('**/api/v1/search/trails/cluster', route => {
            clusterZooms.push(route.request().postDataJSON().zoom);
            return route.fulfill({ json: { type: 'FeatureCollection', features: [], totalHits: 0 } });
        });
        await openMap(page);
        await expect.poll(() => clusterZooms.at(-1)).toBe(14);
        const loaded = searches;
        const area = new URL(page.url()).search;
        const params = new URLSearchParams(area);
        const mercatorY = (latitude: number) => Math.log(Math.tan(Math.PI / 4 + latitude * Math.PI / 360));
        const widthAtZoomZero = (Number(params.get('tl_lon')) - Number(params.get('br_lon'))) / 360 * 512;
        const heightAtZoomZero = (mercatorY(Number(params.get('tl_lat'))) - mercatorY(Number(params.get('br_lat')))) / (2 * Math.PI) * 512;
        await page.locator('.expand-results').click();
        const canvas = page.locator('#trail-map .maplibregl-canvas');
        const size = await canvas.evaluate(element => {
            const canvas = element as HTMLCanvasElement;
            return { width: canvas.clientWidth, height: canvas.clientHeight, pixels: [canvas.width, canvas.height], dpr: devicePixelRatio };
        });
        expect(size.pixels).toEqual([Math.floor(size.width * size.dpr), Math.floor(size.height * size.dpr)]);
        const expectedZoom = Math.log2(Math.min(size.width / widthAtZoomZero, size.height / heightAtZoomZero));
        await expect.poll(() => clusterZooms.at(-1)).toBeCloseTo(expectedZoom, 5);
        await page.waitForTimeout(300); // Negative check spans the viewport debounce.
        expect(searches).toBe(loaded);
        expect(new URL(page.url()).search).toBe(area);
    });
});

test('expands results into columns with an interactive small map and restores the split', async ({ page, context }) => {
    let searches = 0;
    // Hover also preloads the linked detail page through SvelteKit.
    await context.route(/\/api\/v1\/trail\/grid-trail-\d+\?/, route => route.fulfill({ json: {
        id: new URL(route.request().url()).pathname.split('/').at(-1),
        name: 'Panoramaweg', author: 'fixtureauthor00', photos: [], tags: [],
        public: true, completed: false, difficulty: 'easy', expand: {},
    } }));
    await context.route('**/api/v1/search/trails', route => {
        ++searches;
        const { options } = route.request().postDataJSON();
        return route.fulfill({ json: {
            hits: Array.from({ length: 8 }, (_, i) => ({ id: 'grid-trail-' + i, name: 'Panoramaweg ' + i,
                author: 'fixtureauthor00', public: true, completed: false, tags: [],
                created: 1710000000, date: 1710000000, distance: 2400, duration: 3600,
                elevation_gain: 100, elevation_loss: 100, difficulty: 0, location: 'Bern',
                _geo: { lat: 46.95, lng: 7.44 } })),
            totalHits: 8, totalPages: 1, page: options.page, hitsPerPage: options.hitsPerPage,
        } });
    });
    await openMap(page);
    await expect(page.locator('.trail-list .trail-card')).toHaveCount(8);
    await expect(page.locator('.result-header .collapse-results')).toBeVisible();
    await expect(page.locator('.result-header .expand-results')).toBeVisible();
    await expect(page.locator('.map-divider')).toHaveCount(0);
    const header = await page.locator('.result-header').boundingBox();
    await page.locator('.trail-results').evaluate(element => { element.scrollTop = 200; });
    expect(await page.locator('.result-header').boundingBox()).toEqual(header);
    expect(await page.locator('.trail-results').evaluate(element => element.scrollTop)).toBeGreaterThan(0);
    await page.locator('.trail-results').evaluate(element => { element.scrollTop = 0; });
    await page.locator('.trail-list .trail-card').first().hover();
    await expect(page.locator('#trail-map .maplibregl-popup')).toBeVisible();
    const loaded = searches;
    const originalArea = new URL(page.url()).search;
    await page.locator('.maplibregl-canvas').evaluate(element => (window as any).retainedMapCanvas = element);
    await page.locator('.expand-results').click();
    await expect(page.locator('.map-results')).toHaveClass(/results-expanded/);
    await expect(page.locator('.result-header')).toHaveCount(0);
    await expect(page.locator('.collapse-results')).toHaveCount(0);
    await expect(page.locator('#trail-map .reset-map-view')).toBeInViewport({ ratio: 1 });
    await expect(page.locator('#trail-map .maplibregl-popup')).toHaveCount(0);
    await page.locator('.trail-list .trail-card').first().hover();
    await expect(page.locator('#trail-map .maplibregl-popup')).toHaveCount(0);
    await expect(page.locator('#trail-map .trail-hover-marker')).toBeVisible();
    await searchInput(page).hover();
    await expect(page.locator('#trail-map .trail-hover-marker')).toHaveCount(0);
    await expect(page.getByRole('separator')).toBeHidden();
    const layout = await page.locator('.map-results').evaluate(element => {
        const list = element.querySelector('.trail-list')!.getBoundingClientRect();
        const map = element.querySelector('#trail-map')!.getBoundingClientRect();
        const cards = [...element.querySelectorAll('.trail-card')].map(card => card.getBoundingClientRect());
        return { width: element.clientWidth, listWidth: list.width, mapWidth: map.width, mapHeight: map.height,
            columns: new Set(cards.map(card => card.x)).size, maxCardWidth: Math.max(...cards.map(card => card.width)) };
    });
    expect(layout.listWidth).toBe(layout.width);
    expect(layout.columns).toBeGreaterThan(1);
    expect(layout.maxCardWidth).toBeLessThanOrEqual(576);
    expect(layout.mapWidth).toBeLessThan(layout.width / 2);
    expect(layout.mapHeight).toBeGreaterThanOrEqual(160);
    const overview = (await page.locator('#trail-map').boundingBox())!;
    const restore = (await page.locator('.map-pane .overview-actions .expand-results').boundingBox())!;
    expect(restore.x).toBeGreaterThanOrEqual(overview.x);
    expect(restore.x + restore.width).toBeLessThanOrEqual(overview.x + overview.width);
    expect(Math.abs(restore.y + restore.height / 2 - overview.y)).toBeLessThanOrEqual(1);
    await settledMapLayout(page);
    await page.waitForTimeout(300); // A layout change must not schedule a later list search.
    expect(searches).toBe(loaded);
    expect(new URL(page.url()).search).toBe(originalArea);
    expect(await page.locator('.maplibregl-canvas').evaluate(element => element === (window as any).retainedMapCanvas)).toBe(true);
    const original = (await page.locator('#trail-map').boundingBox())!;
    const handle = page.locator('.overlay-resize');
    const grip = (await handle.boundingBox())!;
    await page.mouse.move(grip.x + 16, grip.y + 16);
    await page.mouse.down();
    await page.mouse.move(grip.x - 84, grip.y - 44, { steps: 4 });
    await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ width: original.width + 100, height: original.height + 60 });
    expect(await mapPixel(page)).toEqual([218, 229, 223, 255]); // Basemap stays painted during the drag.
    await page.mouse.up();
    await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ width: original.width + 100, height: original.height + 60 });
    await settledMapLayout(page);
    await page.waitForTimeout(300); // A layout change must not schedule a later list search.
    expect(searches).toBe(loaded);
    await handle.press('ArrowLeft');
    await handle.press('ArrowUp');
    await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ width: original.width + 116, height: original.height + 76 });
    await page.locator('#trail-map .maplibregl-ctrl-zoom-in').click();
    await expect.poll(() => searches).toBeGreaterThan(loaded);
    await page.locator('.expand-results').click();
    await expect(page.locator('.map-results')).not.toHaveClass(/results-expanded/);
    await expect(page.getByRole('separator')).toHaveAttribute('aria-valuenow', '400');
    await page.locator('.expand-results').click();
    await page.reload();
    await expect(page.locator('.map-results')).toHaveClass(/results-expanded/);
    await expect(page.locator('.maplibregl-canvas')).toBeVisible();
    await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ width: original.width + 116, height: original.height + 76 });


    await expect(page.locator('.map-results')).toHaveClass(/results-expanded/);
    await page.locator('.expand-results').click();
    await page.locator('.trail-list .trail-card').first().hover();
    await expect(page.locator('#trail-map .maplibregl-popup')).toBeVisible();
});

test('moves, hides and restores the overview without changing the search area', async ({ page }) => {
    let searches = 0;
    page.on('request', request => { if (isTrailSearch(request.url())) ++searches; });
    await openMap(page);
    await page.locator('.expand-results').click();
    await expect(page.locator('.overlay-close')).toBeVisible();
    await settledMapLayout(page);
    const loaded = searches;
    const area = new URL(page.url()).search;
    const original = (await page.locator('#trail-map').boundingBox())!;
    await page.locator('.maplibregl-canvas').evaluate(element => (window as any).retainedMapCanvas = element);
    const move = (await page.locator('.overlay-move').boundingBox())!;
    await page.mouse.move(move.x + move.width / 2, move.y + move.height / 2);
    await page.mouse.down();
    await page.mouse.move(move.x + move.width / 2 - 120, move.y + move.height / 2 - 60, { steps: 4 });
    await page.mouse.up();
    await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ x: original.x - 120, y: original.y - 60 });
    for (const button of await page.locator('.overview-actions button').all()) {
        const box = (await button.boundingBox())!;
        expect(Math.abs(box.y + box.height / 2 - original.y + 60)).toBeLessThanOrEqual(1);
        expect(box).toMatchObject({ width: 32, height: 32 });
        await expect(button).toBeInViewport({ ratio: 1 });
    }
    await page.locator('.overlay-close').click();
    await expect(page.locator('#trail-map')).toBeHidden();
    await expect(page.locator('.overlay-reopen')).toBeFocused();
    const screen = page.viewportSize()!;
    expect(await page.locator('.overlay-reopen').boundingBox()).toMatchObject({
        x: screen.width - 88, y: screen.height - 88, width: 64, height: 64,
    });
    await page.locator('#trail-map .maplibregl-ctrl-zoom-in').evaluate((button: HTMLButtonElement) => button.focus());
    await expect(page.locator('.overlay-reopen')).toBeFocused();
    await expect(page.locator('.map-results')).toHaveClass(/results-expanded/);
    await page.locator('.overlay-reopen').click();
    await expect(page.locator('#trail-map')).toBeVisible();
    expect(await mapPixel(page)).toEqual([218, 229, 223, 255]);
    expect(await page.locator('.maplibregl-canvas').evaluate(element => element === (window as any).retainedMapCanvas)).toBe(true);
    await settledMapLayout(page);
    await page.waitForTimeout(300); // A layout change must not schedule a later list search.
    expect(searches).toBe(loaded);
    expect(new URL(page.url()).search).toBe(area);
    await page.locator('.overlay-close').click();
    await page.reload();
    await expect(page.locator('.overlay-reopen')).toBeVisible();
    await expect(page.locator('#trail-map')).toBeHidden();

    await expect(page.locator('.overlay-reopen')).toBeVisible();
    await page.locator('.overlay-reopen').click();
    await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ x: original.x - 120, y: original.y - 60 });
    await page.locator('.overview-actions .expand-results').click();
    await expect(page.locator('.map-results')).not.toHaveClass(/results-expanded/);
    await expect(page.locator('.result-header')).toBeVisible();
});

test('keeps all overview controls visible at the minimum size, with reset last', async ({ page, context }) => {
    await context.addInitScript(() => localStorage.setItem('mapResultsLayout', JSON.stringify({
        expanded: true, overlay: { width: 256, height: 160 },
    })));
    await openMap(page);
    const controlsFit = () => page.locator('#trail-map').evaluate(element => {
        const map = element.getBoundingClientRect();
        const buttons = [...element.querySelectorAll<HTMLButtonElement>('.maplibregl-ctrl-top-right button')].filter(button => button.getClientRects().length);
        const attribution = element.querySelector('.maplibregl-ctrl-bottom-right')?.getBoundingClientRect();
        return {
            inside: Boolean(attribution) && buttons.length >= 5 && buttons.every(button => {
                const rect = button.getBoundingClientRect();
                return rect.top >= map.top && rect.left >= map.left && rect.right <= map.right && rect.bottom < attribution!.top;
            }),
            resetLast: buttons.at(-1)?.classList.contains('reset-map-view') ?? false,
        };
    });
    await expect.poll(controlsFit).toEqual({ inside: true, resetLast: true });
    const handle = page.locator('.overlay-resize');
    await handle.press('Home');
    const minimum = (await page.locator('#trail-map').boundingBox())!;
    expect(minimum.width).toBe(256);
    expect(minimum.height).toBeGreaterThan(160);
    const grip = (await handle.boundingBox())!;
    await page.mouse.move(grip.x + 12, grip.y + 12);
    await page.mouse.down();
    await page.mouse.move(grip.x + 12, grip.y + 160, { steps: 4 });
    await page.mouse.up();
    await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ height: minimum.height });
    await expect.poll(controlsFit).toEqual({ inside: true, resetLast: true });
    await page.setViewportSize({ width: 1000, height: 420 });
    await handle.press('Home');
    await expect.poll(controlsFit).toEqual({ inside: true, resetLast: true });
    await page.reload();
    await expect.poll(controlsFit).toEqual({ inside: true, resetLast: true });
});

for (const granted of [true, false]) {
    test(`uses ${granted ? 'browser location' : 'the default camera after location denial'} without a saved location`, async ({ page, context }) => {
        const auth = (await context.cookies()).find(cookie => cookie.name === 'pb_auth')!;
        const userId = JSON.parse(decodeURIComponent(auth.value)).record.id;
        const user = await (await page.request.get(`/api/v1/user/${userId}?expand=settings_via_user`)).json();
        const settings = user.expand.settings_via_user[0];
        const endpoint = `/api/v1/settings/${settings.id}`;
        const updated = await page.request.post(endpoint, { data: { mapFocus: 'location', location: null } });
        expect(updated.ok()).toBe(true);
        try {
            await context.grantPermissions(granted ? ['geolocation'] : []);
            if (granted) await context.setGeolocation({ latitude: 49, longitude: 10 });
            await page.goto('/map');
            await expect(page.locator('.maplibregl-canvas')).toBeVisible();
            await expect.poll(() => new URL(page.url()).searchParams.has('tl_lat')).toBe(true);
            const position = async () => {
                const params = new URL(page.url()).searchParams;
                return [Number(params.get('tl_lon')) + Number(params.get('br_lon')), Number(params.get('tl_lat')) + Number(params.get('br_lat'))];
            };
            await expect.poll(position).toEqual(granted ? [expect.closeTo(20, 3), expect.closeTo(98, 2)] : [expect.closeTo(0, 3), expect.closeTo(0, 3)]);
            const params = new URL(page.url()).searchParams;
            const width = await page.locator('.maplibregl-canvas').evaluate(canvas => canvas.clientWidth);
            const longitudeSpan = Number(params.get('tl_lon')) - Number(params.get('br_lon'));
            expect(Math.log2(width / (longitudeSpan / 360 * 512))).toBeCloseTo(granted ? 12 : 1, 4);
        } finally {
            const restored = await page.request.post(endpoint, { data: { mapFocus: settings.mapFocus, location: settings.location ?? null } });
            expect(restored.ok()).toBe(true);
        }
    });
}

test('offers trails outside the viewport and centers a suggested place without a radius filter', async ({ page, context }) => {
    const requests: any[] = [];
    await context.route('**/api/v1/search/trails', route => {
        const body = route.request().postDataJSON();
        requests.push(body);
        const direct = body.options.limit === 3;
        return route.fulfill({ json: {
            hits: direct ? Array.from({ length: 3 }, (_, index) => ({ id: 'outside-' + index, name: 'Ridge ' + index, author_name: 'alice', domain: 'remote.example' })) : [],
            totalHits: 0, totalPages: 0, page: body.options.page ?? 1, hitsPerPage: body.options.hitsPerPage ?? 3,
        } });
    });
    await context.route('**/api/v1/geocoding/search**', route => route.fulfill({ json: {
        type: 'FeatureCollection', features: [{ type: 'Feature', geometry: { type: 'Point', coordinates: [10, 49] }, properties: {
            name: 'Ridge', display_name: 'Ridge, Germany', category: 'place', type: 'city', addresstype: 'city', address: { city: 'Ridge', country: 'Germany' },
        } }],
    } }));
    await openMap(page);
    await searchInput(page).click();
    await page.locator('.general-filters .filter-chip').filter({ hasText: /^Place/ }).click();
    await page.locator('.place-input input').fill('Existing');
    await page.locator('.place-option').first().click();
    await expect.poll(() => requests.some(request => request.options.filter?.includes('_geoRadius'))).toBe(true);
    await searchInput(page).fill('Ridge');
    const links = page.locator('[data-kind="trails"] .direct-option');
    await expect(links).toHaveCount(3);
    await expect(links.first()).toHaveAttribute('href', '/map/trail/@alice@remote.example/outside-0');
    const direct = requests.find(request => request.options.limit === 3);
    await expect.poll(() => requests.find(request => request.q === 'Ridge' && request.options.page)?.options.filter).toBeTruthy();
    const list = requests.find(request => request.q === 'Ridge' && request.options.page);
    expect(list.options.filter.startsWith(direct.options.filter)).toBe(true);
    expect(direct.options.filter).not.toContain('max_lat >=');
    expect(direct.options.attributesToSearchOn).toEqual(['name']);
    const placeSearch = page.waitForRequest(request => isTrailSearch(request.url()) &&
        request.postDataJSON().q === '' && Boolean(request.postDataJSON().options.page) &&
        !request.postDataJSON().options.filter.includes('_geoRadius'));
    await page.locator('[data-kind="places"] .direct-option').first().click();
    await placeSearch;
    await expect(searchInput(page)).toHaveValue('');
    await expect.poll(() => {
        const params = new URL(page.url()).searchParams;
        return Number(params.get('tl_lon')) + Number(params.get('br_lon'));
    }).toBeCloseTo(20, 4);
    const params = new URL(page.url()).searchParams;
    const width = await page.locator('.maplibregl-canvas').evaluate(canvas => canvas.clientWidth);
    expect(Math.log2(width / ((Number(params.get('tl_lon')) - Number(params.get('br_lon'))) / 360 * 512))).toBeCloseTo(14, 4);
    await expect(page.locator('[data-filter-key="location"]')).toHaveCount(0);
    expect(requests.at(-1).options.filter).not.toContain('_geoRadius');
});

test('holds selected list results when push search changes map height but queries a wider window', async ({ page, context }) => {
    await context.addInitScript(() => localStorage.setItem('searchLayout', JSON.stringify({ position: 'top', expansion: 'push' })));
    const listQueries: unknown[] = [];
    const markerQueries: unknown[] = [];
    await context.route('**/api/v1/search/trails', route => {
        const body = route.request().postDataJSON();
        listQueries.push(body);
        return route.fulfill({ json: {
            hits: [0, 1].map(index => ({ id: 'push-' + index, name: 'Push trail ' + index, author: 'fixtureauthor00', author_name: 'Fixture',
                public: true, completed: false, tags: [], created: 1710000000, date: 1710000000, distance: 2400, duration: 3600,
                elevation_gain: 100, elevation_loss: 100, difficulty: 0, location: 'Bern', _geo: { lat: 47, lng: 7.45 } })),
            totalHits: 2, totalPages: 1, page: 1, hitsPerPage: 48,
        } });
    });
    await context.route('**/api/v1/search/trails/cluster', route => {
        markerQueries.push(route.request().postDataJSON());
        return route.fulfill({ json: { type: 'FeatureCollection', features: [], totalHits: 0 } });
    });
    await context.route(/\/api\/v1\/trail\/push-\d+\?/, route => route.fulfill({ json: {
        id: new URL(route.request().url()).pathname.split('/').at(-1), name: 'Push trail', author: 'fixtureauthor00',
        photos: [], tags: [], public: true, completed: false, difficulty: 'easy', expand: {},
    } }));
    await openMap(page);
    await settledMapLayout(page);
    await page.locator('.trail-card').first().hover();
    await page.getByRole('checkbox', { name: 'Select Push trail 0', exact: true }).check();
    await page.locator('.trail-card').nth(1).getByText('Push trail 1', { exact: true }).click();
    await expect(page.locator('.selection-count')).toHaveText('2 selected');
    const count = listQueries.length;
    const area = new URL(page.url()).search;
    const markers = markerQueries.length;
    await searchInput(page).click();
    await expect(page.getByRole('region', { name: 'Filter by', exact: true })).toBeVisible();
    await expect.poll(() => markerQueries.length).toBeGreaterThan(markers);
    await page.waitForTimeout(300); // A height-only change must outlast the list debounce without a query.
    expect(listQueries).toHaveLength(count);
    expect(new URL(page.url()).search).toBe(area);
    await expect(page.locator('.selection-count')).toHaveText('2 selected');
    await page.setViewportSize({ width: 1440, height: 900 });
    await expect.poll(() => listQueries.length).toBeGreaterThan(count);
});

for (const theme of ['light', 'dark']) {
    test(`map overlay buttons have an opaque contrasting surface and a focus ring in ${theme} mode`, async ({ page }) => {
        await openMap(page);
        await page.evaluate(theme => {
            for (const element of [document.documentElement, document.body]) {
                element.classList.remove('light', 'dark'); element.classList.add(theme);
            }
        }, theme);
        const check = async (selector: string) => {
            const button = page.locator(selector);
            await expect(button).toBeVisible();
            await page.mouse.move(0, 0);
            await button.focus();
            await expect.poll(() => button.evaluate(element => {
                const style = getComputedStyle(element);
                const channels = (color: string) => color.match(/[\d.]+/g)!.map(Number);
                const background = channels(style.backgroundColor);
                const foreground = channels(getComputedStyle(element.querySelector('i') ?? element).color);
                const luminance = (rgb: number[]) => rgb.slice(0, 3).map(c => c / 255).map(c => c <= .04045 ? c / 12.92 : ((c + .055) / 1.055) ** 2.4).reduce((sum, c, i) => sum + c * [.2126, .7152, .0722][i], 0);
                const a = luminance(background), b = luminance(foreground);
                return {
                    opaque: background.length === 3 || background[3] === 1,
                    contrast: (Math.max(a, b) + .05) / (Math.min(a, b) + .05) >= 3,
                    focus: /0px 0px 0px 4px/.test(style.boxShadow),
                    shadow: style.filter.includes('drop-shadow'),
                };
            })).toEqual({ opaque: true, contrast: true, focus: true, shadow: true });
        };


        await page.locator('.result-header .collapse-results').click();
        await check('.collapsed-actions .collapse-results');
        await page.locator('.collapsed-actions .collapse-results').click();
        await page.locator('.result-header .expand-results').click();
        await settledMapLayout(page);
        for (const selector of ['.overview-actions .expand-results', '.overlay-close', '.overlay-move']) await check(selector);
        await page.locator('.overlay-close').click();
        await check('.overlay-reopen');
    });
}

test.describe('touch map tools', () => {
    test.use({ hasTouch: true, isMobile: true, userAgent: devices['Galaxy Tab S4 landscape'].userAgent, viewport: devices['Galaxy Tab S4 landscape'].viewport });
    for (const theme of ['light', 'dark']) {
        test(`keeps map tool surfaces after touch in ${theme} mode`, async ({ page }) => {
            await openMap(page);
            expect(await page.evaluate(() => matchMedia('(hover: none)').matches)).toBe(true);
            await page.evaluate(theme => { for (const node of [document.documentElement, document.body]) { node.classList.remove('light', 'dark'); node.classList.add(theme); } }, theme);
            const opaque = async (selector: string) => {
                await page.locator(selector).tap();
                await page.locator(selector).evaluate(async element => {
                    getComputedStyle(element).backgroundColor;
                    await Promise.all(element.getAnimations().map(animation => animation.finished));
                });
                await expect.poll(() => page.locator(selector).evaluate(element => getComputedStyle(element).backgroundColor))
                    .toBe(theme === 'light' ? 'rgb(255, 255, 255)' : 'rgb(30, 32, 40)');
            };
            await page.locator('.result-header .expand-results').tap();
            await settledMapLayout(page);
            await opaque('.overlay-move');

        });
    }
});

test('keeps the camera after opening a trail, Back and reload', async ({ page, context }) => {
    const trail = { id: 'snapshot-trail', name: 'Snapshot trail', author: 'fixtureauthor00', public: true,
        completed: false, tags: [], photos: [], difficulty: 0, expand: {}, distance: 2400, duration: 3600,
        elevation_gain: 100, elevation_loss: 100, created: 1710000000, date: 1710000000,
        _geo: { lat: 46.95, lng: 7.44 } };
    await context.route('**/api/v1/search/trails', route => route.fulfill({ json: {
        hits: [trail], totalHits: 1, totalPages: 1, page: 1, hitsPerPage: 48,
    } }));
    await context.route(/\/api\/v1\/trail\/snapshot-trail\?/, route => route.fulfill({ json: { ...trail, difficulty: 'easy', created: '2026-10-01T12:00:00Z', date: '2026-10-01T12:00:00Z' } }));
    const readCamera = () => page.evaluate(() => {
        // Inspect the real router snapshot, captured from the live map. Calling
        // beforeunload exercises SvelteKit's normal persistence path without unloading.
        window.dispatchEvent(new Event('beforeunload', { cancelable: true }));
        const snapshots = JSON.parse(sessionStorage.getItem('sveltekit:snapshot') ?? '{}');
        const metadata = history.state['sveltekit:metadata'];
        const index = metadata?.navigationIndex;
        const current = index === undefined ? Object.values(snapshots).at(-1) : snapshots[index];
        return (current as any[])?.find(value => value?.camera)?.camera;
    });
    await openMap(page);

    await settledMapLayout(page);
    const original = await readCamera();
    expect(original).toBeTruthy();
    await page.locator('.trail-result a').first().click();
    await expect(page).toHaveURL(/\/map\/trail\//);
    await page.goBack();
    await settledMapLayout(page);
    await expect.poll(readCamera).toMatchObject({ zoom: expect.closeTo(original.zoom, 8), center: original.center });
    await page.reload();
    await settledMapLayout(page);
    await expect.poll(readCamera).toMatchObject({ zoom: expect.closeTo(original.zoom, 8), center: original.center });
});


test('phone map switches between results and a painted map without horizontal scrolling', async ({page}) => {
    await page.setViewportSize({width:390,height:844});
    await openMap(page);
    await settledMapLayout(page);
    expect((await mapPixel(page)).slice(0,3)).toEqual([218,229,223]);
    await page.locator('.map-view-toggle').click();
    await expect(page.locator('#trail-map')).toBeHidden();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    expect(await page.locator('.trail-results').evaluate(node => node.scrollWidth <= node.clientWidth)).toBe(true);
    await page.locator('.map-view-toggle').click();
    await expect(page.locator('#trail-map')).toBeVisible();
    await settledMapLayout(page);
    expect((await mapPixel(page)).slice(0,3)).toEqual([218,229,223]);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
});
