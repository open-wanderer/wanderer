import { expect, test, type Page } from '../../fixtures';
import { mockEmptyGeocoding } from '../search_helpers';
import { expectActiveNavigation } from '../navigation_helpers';

const isSearch = (url: string) => new URL(url).pathname === '/api/v1/search/trails';
const input = (page: Page) => page.locator('[data-search-root] input[type="search"]').first();
const view = (page: Page) => page.locator('.search-toolbar .control-button[aria-controls$="-view"]');

// Keep the real loaders, controls, stores and routing. Search responses are
// deterministic so link parameters and the list-to-map handoff are observable.
test.beforeEach(async ({ context }) => {
    await mockEmptyGeocoding(context);
    await context.route('**/styles/ofm.json', route => route.fulfill({ json: {
        version: 8, sources: {}, layers: [{ id: 'background', type: 'background', paint: { 'background-color': '#dae5df' } }],
    } }));
    await context.route('**/api/v1/search/trails/cluster', route => route.fulfill({ json: { type: 'FeatureCollection', features: [], totalHits: 0 } }));
    await context.route('**/api/v1/search/trails', route => {
        const { options } = route.request().postDataJSON();
        if (options.limit === 0) return route.fulfill({ json: { facetStats: {
            min_lat: { min: 46.9 }, max_lat: { max: 47.1 }, min_lon: { min: 7.3 }, max_lon: { max: 7.6 },
        } } });
        return route.fulfill({ json: {
            hits: [{ id: 'list-fixture', name: 'Ridge loop', author: 'fixtureauthor00', author_name: 'Fixture',
                author_avatar: '/favicon.png', public: true, completed: false, tags: [],
                created: 1710000000, date: 1710000000, distance: 2400, duration: 3600,
                elevation_gain: 100, elevation_loss: 100, difficulty: 0, location: 'Bern', _geo: { lat: 47, lng: 7.45 } }],
            totalHits: 1, totalPages: 1, page: options.page ?? 1, hitsPerPage: options.hitsPerPage ?? 3,
        } });
    });
    await context.route(/\/api\/v1\/trail\/list-fixture\?/, route => route.fulfill({ json: {
        id: 'list-fixture', name: 'Ridge loop', author: 'fixtureauthor00', photos: [], tags: [], public: true,
        completed: false, difficulty: 'easy', expand: {},
    } }));
});

async function openTrails(page: Page, query = '') {
    const response = page.waitForResponse(response => isSearch(response.url()));
    await page.goto('/trails' + query);
    await response;
    await expect(page.locator('.result-content')).toHaveAttribute('aria-busy', 'false');
}

test('marks Trails active after a direct visit and reload', async ({ page }) => {
    await openTrails(page);
    await expectActiveNavigation(page, '/trails');
    await page.reload();
    await expectActiveNavigation(page, '/trails');
});

test('shows a trail at 390 pixels and keeps its normal detail link', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 800 });
    await openTrails(page);
    await expect(page.locator('.trail-card')).toBeVisible();
    await expect(page.locator('#trails .trail-result > a').first()).toHaveAttribute('href', '/trail/view/@Fixture/list-fixture');
    await expect(page.locator('.maplibregl-canvas')).toHaveCount(0);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
});

test('loads author and category links without applying an old map area', async ({ page }) => {
    const categories = await (await page.request.get('/api/v1/category?perPage=-1')).json();
    const category = categories.items[0];
    expect(category.id).toBeTruthy();
    const request = page.waitForRequest(request => isSearch(request.url()));
    await openTrails(page, `?author=linked-author&category=${category.id}&tl_lat=48&tl_lon=9&br_lat=46&br_lon=7`);
    const body = (await request).postDataJSON();
    expect(body.options.filter).toContain('author = linked-author');
    expect(body.options.filter).toContain(category.id);
    expect(body.options.filter).not.toContain('max_lat >=');
    expect(body.options.filter).not.toContain('min_lon <=');
});

test('retains table presentation when returning through browser history', async ({ page }) => {
    await openTrails(page);
    await view(page).click();
    await page.locator('.view-menu input[value="table"]').check();
    await page.keyboard.press('Escape');
    await expect(page.locator('#trails table')).toBeVisible();
    await page.locator('#nav-bar-links a[href="/"]').click();
    await expect(page).toHaveURL('/');
    await page.goBack();
    await expect(page.locator('#trails table')).toBeVisible();
    expect(await page.evaluate(() => localStorage.getItem('displayOption'))).toBe('table');
});

test('shows effective creation sorting and reverses it on the first table-header click', async ({ page }) => {
    await openTrails(page);
    await view(page).click();
    await page.locator('.view-menu input[value="table"]').check();
    await page.keyboard.press('Escape');
    const header = page.locator('#trails th').filter({ hasText: 'Creation date' });
    await expect(header.locator('.fa-arrow-up')).toBeVisible();
    const changed = page.waitForRequest(request => isSearch(request.url()) && request.postDataJSON().options.sort?.[0] === 'created:desc');
    await header.click();
    await changed;
    await expect(header.locator('.fa-arrow-up')).toHaveClass(/rotated/);
});

test('opens matching trails on the map and retains the handed-over filter after reload', async ({ page }) => {
    await openTrails(page);
    const searched = page.waitForResponse(response => isSearch(response.url()) && response.request().postDataJSON().q === 'ridge');
    await input(page).fill('ridge');
    await searched;
    await input(page).press('Escape');
    const firstMapRequest = page.waitForRequest(request => isSearch(request.url()) && request.postDataJSON().options.filter.includes('max_lat >='));
    await page.getByRole('button', { name: 'Show on map', exact: true }).click();
    await expect(page).toHaveURL(url => url.pathname === '/map' && url.searchParams.has('tl_lat'));
    await expect(page.locator('.maplibregl-canvas')).toBeVisible();
    await expect(input(page)).toHaveValue('ridge');
    await expect(page.locator('.trail-list')).toContainText('Ridge loop');
    const mapQuery = (await firstMapRequest).postDataJSON();
    expect(mapQuery.q).toBe('ridge');
    expect(mapQuery.options.filter).toContain('min_lat <=');
    expect(mapQuery.options.filter).toContain('max_lon >=');
    expect(mapQuery.options.filter).toContain('min_lon <=');
    const url = new URL(page.url());
    expect(Number(url.searchParams.get('tl_lat'))).toBeGreaterThan(47.1);
    expect(Number(url.searchParams.get('br_lat'))).toBeLessThan(46.9);
    expect(Number(url.searchParams.get('tl_lat'))).toBeLessThan(47.5);
    expect(Number(url.searchParams.get('br_lat'))).toBeGreaterThan(46.5);
    expect(Number(url.searchParams.get('tl_lon'))).toBeGreaterThan(7.6);
    expect(Number(url.searchParams.get('tl_lon'))).toBeLessThan(8);
    expect(Number(url.searchParams.get('br_lon'))).toBeLessThan(7.3);
    expect(Number(url.searchParams.get('br_lon'))).toBeGreaterThan(6.9);
    await page.reload();
    await expect(input(page)).toHaveValue('ridge');
    await expect(page.locator('.trail-list')).toContainText('Ridge loop');
});

test('offers the map handoff in the mobile view menu without a third input button', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 800 });
    await openTrails(page);
    await expect(page.locator('.show-on-map')).toBeHidden();
    await expect(page.locator('.search-toolbar .control-button')).toHaveCount(2);
    await view(page).click();
    await expect(page.locator('.view-menu .show-on-map-mobile')).toBeVisible();
    await page.getByRole('button', { name: 'Show on map', exact: true }).click();
    await expect(page).toHaveURL(url => url.pathname === '/map');
});

test.describe('failed mobile map handoff', () => {
    // This regression deliberately returns 503. Keep its expected network
    // diagnostic scoped to this test and endpoint; all other errors still fail.
    test.use({ browserErrorAllowlist: [{
        source: 'console.error',
        message: /^Failed to load resource: the server responded with a status of 503 \(Service Unavailable\)$/,
        url: /\/api\/v1\/search\/trails$/,
        issue: 'https://github.com/open-wanderer/wanderer/pull/1404',
    }] });
    test('keeps mobile handoff visibly busy and closes its menu on failure', async ({ page, context }) => {
        await page.setViewportSize({ width: 390, height: 800 });
        let release!: () => void;
        let gate = new Promise<void>(resolve => release = resolve);
        await context.route('**/api/v1/search/trails', async route => {
            if (route.request().postDataJSON().options.limit !== 0) return route.fallback();
            await gate;
            await route.fulfill({ status: 503, json: { message: 'Unavailable' } });
        });
        await openTrails(page);
        await view(page).click();
        const action = page.locator('.show-on-map-mobile');
        await action.click();
        await expect(action).toBeDisabled();
        await expect(action).toHaveCSS('opacity', '0.45');
        release();
        await expect(page.locator('.view-menu')).toBeHidden();
        await expect(page.getByRole('alert').filter({ hasText: 'Could not open these trails' })).toBeVisible();
        const retry = page.getByRole('button', { name: 'Try again', exact: true });
        await expect(retry).toBeVisible();
        const announcement = await page.getByRole('alert').filter({ hasText: 'Could not open these trails' }).elementHandle();
        gate = new Promise<void>(resolve => release = resolve);
        const retried = page.waitForRequest(request => request.url().endsWith('/api/v1/search/trails') && request.postDataJSON().options.limit === 0);
        await retry.click();
        await retried;
        await expect(retry).toHaveAttribute('aria-busy', 'true');
        await expect(retry.locator('.fa-spinner')).toBeVisible();
        await expect(retry).toHaveCSS('opacity', '0.45');
        release();
        await expect.poll(() => announcement!.evaluate(element => element.isConnected)).toBe(false);
        await expect(page.getByRole('alert').filter({ hasText: 'Could not open these trails' })).toBeVisible();
        await expect(retry).toBeFocused();
    });
});
