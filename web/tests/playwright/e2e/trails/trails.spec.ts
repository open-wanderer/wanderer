import { expect, test, type Page } from '../../fixtures';
import { mockEmptyGeocoding } from '../search_helpers';
import { expectActiveNavigation } from '../navigation_helpers';

const isSearch = (url: string) => new URL(url).pathname === '/api/v1/search/trails';
const view = (page: Page) => page.locator('.search-toolbar .control-button[aria-controls$="-view"]');

// Keep the real loaders, controls, stores and routing. Search responses are
// deterministic so link parameters remain observable.
test.beforeEach(async ({ context }) => {
    await mockEmptyGeocoding(context);
    await context.route('**/api/v1/search/trails', route => {
        const { options } = route.request().postDataJSON();
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
