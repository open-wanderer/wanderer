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
    await expect(page.locator('#trail-map')).toHaveAttribute('aria-busy', 'false');
}
// The search area leaves out the part of the map under the results panel.
const uncoveredWidth = (page: Page) => page.evaluate(() => {
    const map = document.querySelector('#trail-map')!.getBoundingClientRect();
    const panel = document.querySelector('.results-panel')!.getBoundingClientRect();
    return map.right - Math.max(map.left, panel.right + 16);
});

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

test('offers cards and list in the map view menu and remembers the choice', async ({ page, context }) => {
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
    await expect(page.locator('.results-panel .trail-card')).toBeVisible();
    const loaded = searches;
    await page.locator('.maplibregl-canvas').evaluate(element => (window as any).retainedMapCanvas = element);
    await viewButton(page).click();
    const menu = page.locator('.view-menu');
    await expect(menu.locator('a')).toHaveCount(0);
    await expect(menu.locator('input[value="cards"]')).toBeChecked();
    await expect(menu.locator('input[value="map"]')).toHaveCount(0);
    await expect(menu.locator('input[value="top"]')).toHaveCount(0);
    await expect(menu.locator('input[value="left"], input[value="right"]')).toHaveCount(0);
    await expect(menu.locator('input[value="table"], input[value="push"]')).toHaveCount(0);
    await menu.locator('input[value="list"]').check();
    await expect(page.locator('.results-panel .result-items .trail-result > a > li')).toBeVisible();
    await expect(page.locator('.results-panel .trail-card')).toHaveCount(0);
    expect(await page.locator('.maplibregl-canvas').evaluate(element => element === (window as any).retainedMapCanvas)).toBe(true);
    await page.waitForTimeout(300); // Negative check spans the viewport debounce.
    expect(searches).toBe(loaded);
    await page.keyboard.press('Escape');
    await expect.poll(() => page.locator('.trail-results').evaluate(element => element.scrollWidth <= element.clientWidth)).toBe(true);
    await page.reload();
    await expect(page.locator('.results-panel .result-items .trail-result > a > li')).toBeVisible();
    await viewButton(page).click();
    await expect(page.locator('.view-menu input[value="list"]')).toBeChecked();
    await page.locator('.view-menu input[value="cards"]').check();
    await page.keyboard.press('Escape');
    await expect(page.locator('.results-panel .trail-card')).toBeVisible();
    await expect.poll(() => page.locator('.trail-results').evaluate(element => element.scrollWidth <= element.clientWidth)).toBe(true);
});

test('restores multi-select, loaded-result selection and bulk edits in cards and list', async ({ page, context }) => {
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
    const cards = page.locator('.results-panel .trail-card');
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
    await page.getByRole('checkbox', { name: 'Select all loaded trails' }).check();
    await expect(page.locator('.selection-count')).toHaveText('4 selected');
    await viewButton(page).click();
    await page.locator('.view-menu input[value="cards"]').check();
    await page.keyboard.press('Escape');
    await settledMapLayout(page);
    const loaded = searches;
    await page.getByRole('button', { name: 'Actions for selected trails' }).click();
    await page.locator('.selection-toolbar .menu').getByText('Adjust', { exact: true }).click();
    const dialog = page.locator('dialog[open]');
    await expect(dialog).toBeVisible();
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
});

test('keeps map search over the map despite a shared sidebar and push preference', async ({ page, context }) => {
    await context.addInitScript(() => localStorage.setItem('searchLayout', JSON.stringify({ position: 'left', expansion: 'push' })));
    await openMap(page);
    await enterSearch(page, 'Keep filters');
    await expect(page.locator('.filter-sidebar')).toHaveCount(0);
    await expect(page.locator('.map-search-toolbar [data-search-root]')).not.toHaveClass(/in-flow/);
    await viewButton(page).click();
    await expect(page.locator('.view-menu input[value="left"], .view-menu input[value="right"], .view-menu input[value="push"]')).toHaveCount(0);
    await page.keyboard.press('Escape');
    await expect(searchInput(page)).toHaveValue('Keep filters');
    expect(await page.evaluate(() => JSON.parse(localStorage.getItem('searchLayout')!))).toMatchObject({ position: 'left', expansion: 'push' });
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
            const [lon, lat] = granted ? [10, 49] : [0, 0];
            const contains = () => {
                const params = new URL(page.url()).searchParams;
                return Number(params.get('br_lon')) < lon && lon < Number(params.get('tl_lon'))
                    && Number(params.get('br_lat')) < lat && lat < Number(params.get('tl_lat'));
            };
            await expect.poll(contains).toBe(true);
            const params = new URL(page.url()).searchParams;
            const longitudeSpan = Number(params.get('tl_lon')) - Number(params.get('br_lon'));
            expect(Math.log2(await uncoveredWidth(page) / (longitudeSpan / 360 * 512))).toBeCloseTo(granted ? 12 : 1, 2);
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
        return Number(params.get('br_lon')) < 10 && 10 < Number(params.get('tl_lon')) && Number(params.get('br_lat')) < 49 && 49 < Number(params.get('tl_lat'));
    }).toBe(true);
    const params = new URL(page.url()).searchParams;
    expect(Math.log2(await uncoveredWidth(page) / ((Number(params.get('tl_lon')) - Number(params.get('br_lon'))) / 360 * 512))).toBeCloseTo(14, 2);
    await expect(page.locator('[data-filter-key="location"]')).toHaveCount(0);
    expect(requests.at(-1).options.filter).not.toContain('_geoRadius');
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


test('phone shows a full-screen painted map and toggles the list without horizontal scrolling', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await openMap(page);
    await settledMapLayout(page);
    expect((await mapPixel(page)).slice(0, 3)).toEqual([218, 229, 223]);
    await expect(page.locator('.results-panel')).toBeHidden();
    const toggle = page.locator('.map-view-toggle');
    await expect(toggle).toHaveText('List');
    await toggle.click();
    await expect(page.locator('.results-panel')).toBeVisible();
    await expect(toggle).toHaveText('Map');
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    expect(await page.locator('.trail-results').evaluate(node => node.scrollWidth <= node.clientWidth)).toBe(true);
    await toggle.click();
    await expect(page.locator('.results-panel')).toBeHidden();
    await settledMapLayout(page);
    expect((await mapPixel(page)).slice(0, 3)).toEqual([218, 229, 223]);
});

test('floats the results panel and search bar over a full-width map', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await openMap(page);
    await settledMapLayout(page);
    const map = (await page.locator('#trail-map').boundingBox())!;
    const panel = (await page.locator('.results-panel').boundingBox())!;
    const search = (await page.locator('.map-search-toolbar').boundingBox())!;
    expect(map.width).toBe(await page.evaluate(() => document.documentElement.clientWidth));
    expect(panel.x).toBeGreaterThan(map.x);
    expect(panel.y + panel.height).toBeLessThan(map.y + map.height);
    expect(search.x).toBeGreaterThan(panel.x + panel.width);
    await expect(page.locator('.expand-results, [role="separator"]')).toHaveCount(0);
    await page.getByRole('button', { name: 'Collapse trail list' }).click();
    await expect(page.locator('.trail-results')).toBeHidden();
    await expect(page.locator('.results-panel h2')).toBeVisible();
    await page.getByRole('button', { name: 'Expand trail list' }).click();
    await expect(page.locator('.trail-results')).toBeVisible();
    await expect(page.locator('.map-view-toggle')).toBeHidden();
    const controls = (await page.locator('#trail-map .maplibregl-ctrl-top-right .maplibregl-ctrl').first().boundingBox())!;
    expect(controls.x).toBeGreaterThanOrEqual(search.x + search.width);
    expect(Math.abs(controls.y - search.y)).toBeLessThan(2);
    const scale = (await page.locator('#trail-map .maplibregl-ctrl-bottom-right .maplibregl-ctrl-scale').boundingBox())!;
    const attribution = (await page.locator('#trail-map .maplibregl-ctrl-attrib').boundingBox())!;
    expect(scale.y + scale.height).toBeLessThanOrEqual(attribution.y);
    const params = new URL(page.url()).searchParams;
    const longitudeSpan = Number(params.get('tl_lon')) - Number(params.get('br_lon'));
    const pixels = longitudeSpan / 360 * 512 * 2 ** 14;
    expect(Math.abs(pixels - await uncoveredWidth(page))).toBeLessThan(2);
});
