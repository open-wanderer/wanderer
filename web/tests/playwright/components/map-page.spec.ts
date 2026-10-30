import { expect, test, type Page } from '@playwright/test';
import { buildMapPageFixture, mountMapPageFixture } from './map-page-fixture';
import { dispatchFullscreenKey, setBrowserFullscreen } from '../browser_fullscreen';

const area = (offset = 0) => ({ northEast: { lat: 48 + offset, lng: 9 }, southWest: { lat: 46 + offset, lng: 7 } });
const result = (prefix: string, page = 1) => ({
    items: Array.from({ length: 48 }, (_, index) => ({ id: `${prefix}-${index}` })), page, totalPages: 5, totalHits: 240,
});
const markers = (name: string) => ({ mapTrails: [{ id: name }], clusters: { type: 'FeatureCollection', features: [] } });
const requests = (page: Page) => page.evaluate(() => (window as any).fixture.requests.map((request: any) => ({
    channel: request.channel, ...request.body, aborted: request.signal?.aborted ?? false,
})));
const resolve = (page: Page, index: number, data: object) => page.evaluate(({ index, data }) => (window as any).fixture.requests[index].resolve(data), { index, data });
const viewport = (page: Page, bounds: ReturnType<typeof area>) => page.evaluate(bounds => (window as any).fixture.viewport(bounds, 11), bounds);
const readResults = (page: Page) => page.getByTestId('results').textContent().then(text => JSON.parse(text!));
const initial = async (page: Page) => { await resolve(page, 0, result('initial')); await resolve(page, 1, markers('initial')); await expect.poll(() => readResults(page)).toMatchObject({ loading: false }); };

for (const dev of [true, false]) {
    test.describe(`map page in ${dev ? 'development' : 'production'}`, () => {
        let bundle: string;
        test.beforeAll(async () => { bundle = await buildMapPageFixture(dev); });
        test.beforeEach(({ page }) => { page.on('pageerror', error => { throw error; }); });

        test('uses the map title and accepts a handed-over filter without shared filter storage', async ({ page }) => {
            await mountMapPageFixture(page, bundle, { state: { mapSearchFilter: { q: 'ridge', category: ['hiking'], sort: 'name', sortOrder: '-' } } });
            await expect(page).toHaveTitle('map | wanderer');
            expect((await requests(page))[0].filter).toMatchObject({ q: 'ridge', category: ['hiking'], sort: 'name', sortOrder: '-' });
            const saved = await page.evaluate(() => (window as any).fixture.capture());
            expect(saved.filter.q).toBe('ridge');
            expect(await page.evaluate(() => localStorage.getItem('trailListFilter'))).toBeNull();
        });

        test('does not rewrite map area parameters for camera rounding noise', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await initial(page);
            await viewport(page, area(1));
            await expect(page).toHaveURL(/tl_lat=49/);
            const url = page.url();
            await viewport(page, area(1 + 1e-10));
            expect(page.url()).toBe(url);
            await viewport(page, area(2));
            await expect(page).toHaveURL(/tl_lat=50/);
        });

        test('accepts a legacy filter snapshot and retains URL bounds when the snapshot has no viewport', async ({ page }) => {
            await mountMapPageFixture(page, bundle, { query: 'tl_lat=48&tl_lon=9&br_lat=46&br_lon=7' });
            await initial(page);
            const filter = (await requests(page))[0].filter;
            await page.evaluate(filter => (window as any).fixture.restore({ ...filter, q: 'legacy' }), filter);
            await expect.poll(() => requests(page).then(value => value.length)).toBe(4);
            expect((await requests(page))[2]).toMatchObject({ filter: { q: 'legacy' }, bounds: area() });
            expect(await page.evaluate(() => (window as any).fixture.focusedBounds)).toEqual([area()]);
            await page.evaluate(filter => (window as any).fixture.restore({ filter: { ...filter, q: 'wrapped' } }), filter);
            await expect.poll(() => requests(page).then(value => value.length)).toBe(6);
            expect((await requests(page))[4]).toMatchObject({ filter: { q: 'wrapped' }, bounds: area() });
        });

        test('centers a direct place suggestion without adding a radius filter', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await initial(page);
            await page.clock.install();
            const location = { lat: 49, lon: 10, name: 'Direct place' };
            await page.evaluate(location => (window as any).fixture.placeSuggestion(location), location);
            await page.clock.runFor(201);
            await expect.poll(() => requests(page).then(value => value.length)).toBe(4);
            expect((await requests(page))[2]).toMatchObject({ bounds: { northEast: { lat: 49.1, lng: 10.1 }, southWest: { lat: 48.9, lng: 9.9 } }, filter: { near: { radius: 2000 } } });
            expect((await requests(page))[2].filter.near).not.toHaveProperty('lat');
            expect((await requests(page))[2].filter.near).not.toHaveProperty('lon');
            expect((await requests(page))[3]).toMatchObject({ zoom: 14 });
            expect(await page.evaluate(() => (window as any).fixture.focusedLocations)).toEqual([location]);
        });

        test('marks only results busy while a replacement search retains old hits', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await initial(page);
            await page.evaluate(() => (window as any).fixture.apply({ q: 'replacement' }));
            await expect(page.locator('.map-results-container')).toHaveAttribute('aria-busy', 'true');
            expect((await readResults(page)).trails).toEqual(result('initial').items);
            await expect(page.locator('.map-page')).not.toHaveAttribute('aria-busy');
            await resolve(page, 2, result('replacement'));
            await expect(page.locator('.map-results-container')).toHaveAttribute('aria-busy', 'false');
        });

        test('clears refresh progress when returning to the loaded area before a replacement finishes', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await initial(page);
            await page.clock.install();
            await viewport(page, area(1));
            await page.clock.runFor(201);
            await expect(page.locator('.map-results-container')).toHaveAttribute('aria-busy', 'true');
            await viewport(page, area());
            await expect(page.locator('.map-results-container')).toHaveAttribute('aria-busy', 'false');
            await expect(page.getByRole('button', { name: 'trail-results-load-more', exact: true })).toBeEnabled();
            expect((await requests(page))[2].aborted).toBe(true);
            await resolve(page, 2, result('stale'));
            expect((await readResults(page)).trails).toEqual(result('initial').items);
        });

        test('keeps replacing results busy across consecutive viewport searches', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await initial(page);
            await page.clock.install({ time: 1000 });
            await page.clock.pauseAt(2000);
            await viewport(page, area(1));
            await page.clock.runFor(201);
            await expect(page.locator('.map-results-container')).toHaveAttribute('aria-busy', 'true');
            await viewport(page, area(2));
            await expect(page.locator('.map-results-container')).toHaveAttribute('aria-busy', 'true');
            await page.clock.runFor(201);
            await expect(page.locator('.map-results-container')).toHaveAttribute('aria-busy', 'true');
            await resolve(page, 4, result('latest'));
            await expect(page.locator('.map-results-container')).toHaveAttribute('aria-busy', 'false');
        });

        test('refreshes only markers for header or visual viewport height changes', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await initial(page);
            const bounds = area(2);
            await page.evaluate(bounds => (window as any).fixture.zoom(11, bounds), bounds);
            await expect.poll(() => requests(page).then(value => value.length)).toBe(3);
            expect((await requests(page))[2]).toMatchObject({ channel: 'map', northEast: bounds.northEast, southWest: bounds.southWest });
            expect((await requests(page))[0].aborted).toBe(false);
            expect((await readResults(page)).trails).toEqual(result('initial').items);
            await page.waitForTimeout(250); // Negative check spans the viewport debounce.
            expect(await requests(page)).toHaveLength(3);
            await viewport(page, area(3));
            await expect.poll(() => requests(page).then(value => value.length)).toBe(5);
            expect((await requests(page))[3]).toMatchObject({ channel: 'results', bounds: area(3) });
        });

        test('debounces consecutive marker-only layout changes without replacing list results', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await initial(page);
            await page.clock.install({ time: 1000 });
            await page.clock.pauseAt(2000);
            await page.evaluate(bounds => (window as any).fixture.zoom(11, bounds), area(1));
            await page.clock.runFor(100);
            await page.evaluate(bounds => (window as any).fixture.zoom(11, bounds), area(2));
            await page.clock.runFor(100);
            expect(await requests(page)).toHaveLength(2);
            await page.clock.runFor(101);
            await expect.poll(() => requests(page).then(value => value.length)).toBe(3);
            expect((await requests(page))[2]).toMatchObject({ channel: 'map', northEast: area(2).northEast });
            expect((await readResults(page)).trails).toEqual(result('initial').items);
        });

        test('retains an explicitly unset sharing filter in handed-over criteria and snapshots', async ({ page }) => {
            await mountMapPageFixture(page, bundle, { state: { mapSearchFilter: { author: 'linked-author' } } });
            expect((await requests(page))[0].filter).not.toHaveProperty('shared');
            await initial(page);
            const snapshot = await page.evaluate(() => (window as any).fixture.capture());
            await page.evaluate(snapshot => (window as any).fixture.restore(snapshot), snapshot);
            expect((await requests(page)).at(-1).filter).not.toHaveProperty('shared');
        });

        test('leaves F11 to the browser except while a modal dialog is open', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await initial(page);
            await page.evaluate(() => {
                Element.prototype.requestFullscreen = () => { throw new Error('Unexpected document fullscreen request'); };
                document.exitFullscreen = () => { throw new Error('Unexpected document fullscreen exit'); };
            });
            expect(await dispatchFullscreenKey(page)).toBe(false);
            await expect(page.locator('.map-page')).not.toHaveClass(/fullscreen/);
            await page.evaluate(() => { const dialog = document.createElement('dialog'); document.body.append(dialog); dialog.showModal(); });
            expect(await dispatchFullscreenKey(page)).toBe(true);
            await expect(page.locator('.map-page')).not.toHaveClass(/fullscreen/);
            await page.evaluate(() => document.querySelector('dialog')!.close());
            expect(await dispatchFullscreenKey(page)).toBe(false);
            expect(await requests(page)).toHaveLength(2);
        });

        test('follows browser fullscreen through the display-mode media query', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await initial(page);
            const snapshot = await page.evaluate(() => (window as any).fixture.capture());
            await setBrowserFullscreen(page, true);
            await expect(page.locator('.map-page')).toHaveClass(/fullscreen/);
            expect(await page.evaluate(() => document.fullscreenElement)).toBeNull();
            expect(await dispatchFullscreenKey(page, 'Escape')).toBe(false);
            await setBrowserFullscreen(page, false);
            await expect(page.locator('.map-page')).not.toHaveClass(/fullscreen/);
            expect(await page.evaluate(() => (window as any).fixture.capture())).toEqual(snapshot);
            expect(await requests(page)).toHaveLength(2);
        });

        for (const allDimensions of [false, true]) {
            test(`screen-sized window is not fullscreen without display-mode (all dimensions=${allDimensions})`, async ({ page }) => {
                await page.addInitScript(allDimensions => {
                    Object.defineProperties(window, {
                        outerWidth: { configurable: true, get: () => screen.width },
                        outerHeight: { configurable: true, get: () => screen.height },
                        ...(allDimensions ? {
                            innerWidth: { configurable: true, get: () => screen.width },
                            innerHeight: { configurable: true, get: () => screen.height },
                        } : {}),
                    });
                    if (allDimensions) Object.defineProperties(screen, {
                        availWidth: { configurable: true, get: () => screen.width },
                        availHeight: { configurable: true, get: () => screen.height },
                    });
                }, allDimensions);
                await mountMapPageFixture(page, bundle, { backgroundNav: true });
                await initial(page);
                await expect(page.locator('.map-page')).not.toHaveClass(/fullscreen/);
                await expect(page.locator('.map-search-toolbar')).toBeVisible();
                await page.evaluate(() => window.dispatchEvent(new Event('resize')));
                await expect(page.locator('.map-page')).not.toHaveClass(/fullscreen/);
                await expect(page.locator('#background-nav')).toHaveJSProperty('inert', false);
            });
        }

        test('keeps external modals and toast live regions usable in fullscreen', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await initial(page);
            await page.evaluate(() => {
                const nav = document.createElement('nav');
                nav.id = 'fixture-nav';
                nav.innerHTML = '<button>Navigation</button><dialog><input aria-label="URL import" value="Keep URL" /></dialog>';
                const toast = document.createElement('div');
                toast.id = 'fixture-toast';
                toast.innerHTML = '<div role="status" aria-live="polite"><button>Toast action</button></div>';
                toast.style.cssText = 'position:fixed;bottom:0;right:0;z-index:1000';
                document.body.append(nav, toast);
                nav.querySelector('dialog')!.showModal();
            });
            await setBrowserFullscreen(page, true);
            await expect(page.getByRole('textbox', { name: 'URL import' })).toBeVisible();
            await expect(page.getByRole('textbox', { name: 'URL import' })).toBeFocused();
            await expect(page.locator('#fixture-nav')).toHaveJSProperty('inert', false);
            await expect(page.locator('#fixture-toast')).toHaveJSProperty('inert', false);
            await page.getByRole('textbox', { name: 'URL import' }).fill('Still usable');
            await page.locator('dialog').evaluate(element => (element as HTMLDialogElement).close());
            expect(await page.locator('#fixture-nav > button').evaluate(element => Boolean(element.closest('[inert]')))).toBe(true);
            expect(await page.locator('#fixture-nav dialog').evaluate(element => element.closest('[inert]'))).toBeNull();
            await expect(page.locator('#fixture-toast [role=status]')).toHaveAttribute('aria-live', 'polite');
            await page.getByRole('button', { name: 'Toast action' }).click();
            await setBrowserFullscreen(page, false);
            await expect(page.locator('#fixture-nav')).toHaveJSProperty('inert', false);
        });

        test('rechecks display-mode on resize without a media change event', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await initial(page);
            for (const enabled of [true, false]) {
                await page.evaluate(enabled => (window as any).setTestBrowserFullscreen(enabled, false, true), enabled);
                if (enabled) await expect(page.locator('.map-page')).toHaveClass(/fullscreen/);
                else await expect(page.locator('.map-page')).not.toHaveClass(/fullscreen/);
            }
        });

        test('direct sibling dialogs and live regions remain interactive in fullscreen', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await initial(page);
            await page.evaluate(() => {
                const main = document.querySelector('.map-page')!;
                const dialog = document.createElement('dialog');
                dialog.innerHTML = '<input aria-label="Direct dialog" />';
                const live = document.createElement('div');
                live.id = 'direct-live'; live.role = 'status'; live.setAttribute('aria-live', 'polite');
                live.innerHTML = '<button>Direct live action</button>';
                live.style.cssText = 'position:fixed;bottom:0;z-index:1001';
                main.before(dialog, live); dialog.showModal();
            });
            await setBrowserFullscreen(page, true);
            await expect(page.getByRole('textbox', { name: 'Direct dialog' })).toBeFocused();
            await expect(page.locator('dialog')).toHaveJSProperty('inert', false);
            await expect(page.locator('#direct-live')).toHaveJSProperty('inert', false);
            await page.locator('dialog').evaluate(element => (element as HTMLDialogElement).close());
            await page.getByRole('button', { name: 'Direct live action' }).click();
        });

        test('recognizes browser fullscreen when the page is first mounted', async ({ page }) => {
            await mountMapPageFixture(page, bundle, { fullscreen: true });
            await initial(page);
            await expect(page.locator('.map-page')).toHaveClass(/fullscreen/);
            await expect(page.locator('.map-search-toolbar')).toBeHidden();
            await setBrowserFullscreen(page, false);
            await expect(page.locator('.map-page')).not.toHaveClass(/fullscreen/);
            await expect(page.locator('.map-search-toolbar')).toBeVisible();
            expect(await requests(page)).toHaveLength(2);
        });

        test('uses only the saved expansion mode while ignoring stored trail criteria', async ({ page }) => {
            const stored = { q: 'trail-only', category: ['cycling'] };
            await mountMapPageFixture(page, bundle, { storage: { trailListFilter: stored, searchLayout: { position: 'left', expansion: 'push' } } });
            expect((await requests(page))[0]).toMatchObject({ channel: 'results', filter: { q: '', category: [], sort: 'created', sortOrder: '-' }, bounds: area(), page: 1 });
            expect((await requests(page))[1]).toMatchObject({ channel: 'map', northEast: area().northEast, southWest: area().southWest });
            await expect(page.getByTestId('search-filter')).toContainText('"layout":"push"');
            await page.evaluate(() => (window as any).fixture.apply({ q: 'map-only' }));
            expect(await page.evaluate(() => JSON.parse(localStorage.getItem('trailListFilter')!))).toEqual(stored);
            expect((await requests(page)).slice(2).map((r: any) => r.filter.q)).toEqual(['map-only', 'map-only']);
        });

        test('keeps search at the top despite a saved side position', async ({ page }) => {
            await mountMapPageFixture(page, bundle, { storage: { searchLayout: { position: 'right', expansion: 'push' } } });
            await initial(page);
            await expect(page.getByTestId('search-filter')).toContainText('"layout":"push"');
            expect(await page.evaluate(() => typeof (window as any).fixture.position)).toBe('undefined');
            await expect(page.locator('.filter-sidebar')).toHaveCount(0);
            expect(await requests(page)).toHaveLength(2);
            expect(await page.evaluate(() => JSON.parse(localStorage.getItem('searchLayout')!))).toEqual({ position: 'right', expansion: 'push' });
        });

        test('switches the result presentation without publishing filters or refetching', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await initial(page);
            await page.evaluate(() => (window as any).fixture.display('list'));
            await expect.poll(() => readResults(page)).toMatchObject({ display: 'list', trails: result('initial').items });
            expect(await requests(page)).toHaveLength(2);
            expect(await page.evaluate(() => localStorage.getItem('mapResultsDisplay'))).toBe('list');
            await page.evaluate(() => (window as any).fixture.display('table'));
            await expect.poll(() => readResults(page)).toMatchObject({ display: 'table', trails: result('initial').items });
            expect(await requests(page)).toHaveLength(2);
            expect(await page.evaluate(() => localStorage.getItem('mapResultsDisplay'))).toBe('table');
            expect(await page.evaluate(() => localStorage.getItem('trailListFilter'))).toBeNull();
        });

        test('keeps successful bulk edits without a stale refetch and refreshes both channels after other mutations', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            const trail = { id: 'initial-0', name: 'Before', expand: { author: { id: 'author' }, category: { id: 'old' } } };
            await resolve(page, 0, { ...result('initial'), items: [trail] });
            await resolve(page, 1, { ...markers('initial'), mapTrails: [trail] });
            await expect.poll(() => readResults(page)).toMatchObject({ loading: false });
            await page.evaluate(() => (window as any).fixture.updateTrails([{ id: 'initial-0', difficulty: 'difficult', expand: { category: { id: 'new' } } }]));
            const expected = { ...trail, difficulty: 'difficult', expand: { author: { id: 'author' }, category: { id: 'new' } } };
            await expect.poll(() => readResults(page)).toMatchObject({ trails: [expected] });
            await expect(page.getByTestId('map')).toContainText('"category":{"id":"new"}');
            expect(await requests(page)).toHaveLength(2);
            await page.evaluate(() => (window as any).fixture.updateTrails());
            await expect.poll(() => requests(page)).toHaveLength(4);
            expect((await requests(page)).slice(2).map((request: any) => request.channel)).toEqual(['results', 'map']);
        });

        test('uses overlay search in fullscreen and restores scrolling and the saved push layout on exit', async ({ page }) => {
            await mountMapPageFixture(page, bundle, { storage: { searchLayout: { position: 'top', expansion: 'push' } } });
            await initial(page);
            await expect(page.locator('.fullscreen-logo')).toHaveCount(0);
            await setBrowserFullscreen(page, true);
            await expect(page.locator('.map-page')).toHaveClass(/fullscreen/);
            await expect(page.locator('.fullscreen-logo')).toBeVisible();
            await expect(page.locator('.fullscreen-logo')).toHaveAttribute('href', '/');
            await expect(page.locator('.map-search-toolbar')).toBeHidden();
            await expect(page.locator('.fullscreen-search-toggle')).toHaveAttribute('aria-expanded', 'false');
            await page.locator('.fullscreen-search-toggle').click();
            await expect(page.locator('.map-search-toolbar')).toBeVisible();
            await expect(page.locator('.fullscreen-search-toggle')).toHaveAttribute('aria-expanded', 'true');
            await expect(page.getByTestId('search-filter')).toContainText('"layout":"overlay"');
            expect(await page.evaluate(() => document.body.style.overflow)).toBe('hidden');
            await page.evaluate(() => (window as any).fixture.menuOpen = true);
            await page.keyboard.press('Escape');
            await expect(page.locator('.map-page')).toHaveClass(/fullscreen/);
            await page.keyboard.press('Escape');
            await expect(page.locator('.map-page')).toHaveClass(/fullscreen/);
            await setBrowserFullscreen(page, false);
            await expect(page.locator('.map-page')).not.toHaveClass(/fullscreen/);
            await expect(page.locator('.fullscreen-logo')).toHaveCount(0);
            await expect(page.getByTestId('search-filter')).toContainText('"layout":"push"');
            expect(await page.evaluate(() => document.body.style.overflow)).toBe('');
            expect(await requests(page)).toHaveLength(2);
            expect(await dispatchFullscreenKey(page)).toBe(false);
            await setBrowserFullscreen(page, true);
            await expect(page.locator('.map-page')).toHaveClass(/fullscreen/);
            await expect(page.locator('.map-search-toolbar')).toBeHidden();
            await setBrowserFullscreen(page, false);
            await expect(page.locator('.map-page')).not.toHaveClass(/fullscreen/);
            await setBrowserFullscreen(page, true);
            await expect(page.locator('.map-page')).toHaveClass(/fullscreen/);
            await page.evaluate(() => (window as any).fixture.destroy());
            expect(await page.evaluate(() => document.body.style.overflow)).toBe('');
        });

        test('keeps search above results after fullscreen and viewport changes', async ({ page }) => {
            await mountMapPageFixture(page, bundle, { storage: { searchLayout: { position: 'right', expansion: 'push' } } });
            await initial(page);
            await setBrowserFullscreen(page, true);
            await expect(page.getByTestId('search-filter')).toContainText('"layout":"overlay"');
            await setBrowserFullscreen(page, false);
            for (const width of [800, 1280]) {
                await page.setViewportSize({ width, height: 800 });
                await expect(page.getByTestId('search-filter')).toContainText('"layout":"push"');
                await expect(page.locator('.filter-sidebar')).toHaveCount(0);
            }
            expect(await requests(page)).toHaveLength(2);
            expect(await page.evaluate(() => JSON.parse(localStorage.getItem('searchLayout')!))).toEqual({ position: 'right', expansion: 'push' });
        });

        test('resets to page one for a new viewport and retains visible results while replacing them', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await initial(page);
            await page.getByRole('button', { name: 'trail-results-load-more', exact: true }).click();
            await resolve(page, 2, result('next', 2));
            await expect.poll(() => readResults(page).then(value => value.trails.length)).toBe(96);
            await page.clock.install();
            await viewport(page, area(1));
            await page.clock.runFor(201);
            await expect.poll(() => requests(page)).toHaveLength(5);
            expect((await requests(page))[3]).toMatchObject({ channel: 'results', page: 1, bounds: area(1) });
            expect((await requests(page))[3]).not.toHaveProperty('pages');
            expect((await readResults(page)).trails).toHaveLength(96);
            await resolve(page, 3, result('scoped'));
            await expect.poll(() => readResults(page)).toMatchObject({ trails: result('scoped').items });
        });

        test('refreshes clusters for the layout zoom without replacing loaded result pages or the search area', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await initial(page);
            await page.getByRole('button', { name: 'trail-results-load-more', exact: true }).click();
            await resolve(page, 2, result('next', 2));
            await expect.poll(() => readResults(page).then(value => value.trails.length)).toBe(96);
            await page.clock.install();
            await page.evaluate(() => (window as any).fixture.zoom(8));
            await page.clock.runFor(201);
            await expect.poll(() => requests(page)).toHaveLength(4);
            expect((await requests(page))[3]).toMatchObject({ channel: 'map', zoom: 8, northEast: area().northEast, southWest: area().southWest });
            expect((await readResults(page)).trails).toHaveLength(96);
            await page.evaluate(() => (window as any).fixture.zoom(11));
            expect((await requests(page))[3].aborted).toBe(true);
            await resolve(page, 3, markers('obsolete-zoom'));
            await page.clock.runFor(201);
            await expect(page.getByTestId('map')).not.toContainText('obsolete-zoom');
            expect(await requests(page)).toHaveLength(4);
            expect((await readResults(page)).trails).toHaveLength(96);
        });

        test('ignores late viewport replies even when returning to the same bounds', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await page.clock.install();
            await viewport(page, area(1));
            await viewport(page, area());
            await page.clock.runFor(201);
            await expect.poll(() => requests(page)).toHaveLength(4);
            expect((await requests(page)).slice(0, 2).every((r: any) => r.aborted)).toBe(true);
            await resolve(page, 2, result('current'));
            await resolve(page, 3, markers('current'));
            await resolve(page, 0, result('obsolete'));
            await resolve(page, 1, markers('obsolete'));
            await expect.poll(() => readResults(page)).toMatchObject({ trails: result('current').items });
            await expect(page.getByTestId('map')).toContainText('current');
            await expect(page.getByTestId('map')).not.toContainText('obsolete');
        });

        test('cancels viewport debounce on a filter change and cancels all work on unmount', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await initial(page);
            await page.clock.install();
            await viewport(page, area(1));
            await page.evaluate(() => (window as any).fixture.apply({ q: 'new' }));
            await expect.poll(() => requests(page)).toHaveLength(4);
            expect((await requests(page)).slice(2).map((r: any) => r.filter.q)).toEqual(['new', 'new']);
            await page.clock.runFor(250);
            expect(await requests(page)).toHaveLength(4);
            await viewport(page, area(2));
            await page.evaluate(() => (window as any).fixture.destroy());
            await page.clock.runFor(250);
            expect(await requests(page)).toHaveLength(4);
            expect((await requests(page)).slice(2).every((r: any) => r.aborted)).toBe(true);
        });

        test('focuses explicit places including a repeated selection without navigating away', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await initial(page);
            const near = { lat: 46.95, lon: 7.44, name: 'Bern', radius: 5000 };
            await page.evaluate(near => (window as any).fixture.apply({ q: '', near }, true), near);
            await page.evaluate(near => (window as any).fixture.apply({ q: '', near }, true), near);
            await expect.poll(() => page.evaluate(() => (window as any).fixture.focusedPlaces)).toEqual([near, near]);
            expect(new URL(page.url()).pathname).toBe('/map');
        });

        test('restores map criteria and viewport from a snapshot without sharing them through storage', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await initial(page);
            await page.evaluate(() => (window as any).fixture.apply({ q: 'saved' }));
            await viewport(page, area(1));
            const saved = await page.evaluate(() => (window as any).fixture.capture());
            await page.evaluate(() => (window as any).fixture.apply({ q: 'changed' }));
            await page.evaluate(saved => (window as any).fixture.restore(saved), saved);
            await expect.poll(() => page.getByTestId('search-filter').textContent()).toContain('"q":"saved"');
            await expect.poll(() => page.evaluate(() => (window as any).fixture.focusedBounds)).toEqual([area(1)]);
            await expect.poll(() => requests(page).then(value => value.filter((r: any) => !r.aborted).slice(-2).map((r: any) => r.filter.q))).toEqual(['saved', 'saved']);
            expect(await page.evaluate(() => localStorage.getItem('trailListFilter'))).toBeNull();
        });


    });
}

// Use the real page and real TrailMapResults, not the request-only rendering
// stub above. setViewportSize dispatches the actual window resize; no manual
// invocation of either component's resize handler is involved.
for (const dev of [true, false]) {
    test.describe(`map page with real map results in ${dev ? 'development' : 'production'}`, () => {
        let bundle: string;
        test.beforeAll(async () => { bundle = await buildMapPageFixture(dev, true); });
        test.beforeEach(async ({ page }) => {
            page.on('pageerror', error => { throw error; });
        });
        for (const lock of ['modal', 'scroll']) {
            test(`page window resize under ${lock} lock refreshes only map data`, async ({ page }) => {
                await mountMapPageFixture(page, bundle);
                await resolve(page, 0, { items: [{ id: 'fixture-0', author: 'tester', name: 'Fixture trail' }], page: 1, totalPages: 1, totalHits: 1 });
                await resolve(page, 1, { mapTrails: [], clusters: { type: 'FeatureCollection', features: [] } });
                await page.getByTestId('map-card').first().hover();
                await page.locator('.selection input').check();
                if (lock === 'modal') await page.evaluate(() => (window as any).mapResultsFixture.openActionDialog());
                else await page.evaluate(() => { document.body.style.position = 'fixed'; document.body.style.width = '100%'; });
                const before = await page.evaluate(() => ({ publications: (window as any).fixture.publications.length, url: location.search, results: (window as any).fixture.requests.filter((r: any) => r.channel === 'results').length }));
                await page.setViewportSize({ width: 1450, height: 900 });
                await page.waitForTimeout(500);
                expect(await page.evaluate(() => ({ publications: (window as any).fixture.publications.length, url: location.search, results: (window as any).fixture.requests.filter((r: any) => r.channel === 'results').length }))).toEqual(before);
                expect(await page.evaluate(() => (window as any).fixture.requests.filter((r: any) => r.channel === 'map').length)).toBeGreaterThan(1);
                expect(await page.evaluate(() => (window as any).mapResultsFixture.selected())).toEqual(['fixture-0']);
                if (lock === 'modal') await expect(page.locator('dialog[open]')).toBeVisible();
            });
        }

        for (const order of ['media-before', 'media-after', 'media-50ms-after', 'resize-delayed'] as const) {
            test(`keeps the exact camera and publishes no viewport or search on F11 (${order})`, async ({ page }) => {
                await mountMapPageFixture(page, bundle);
                await resolve(page, 0, { items: [], page: 1, totalPages: 1, totalHits: 0 });
                await resolve(page, 1, { mapTrails: [], clusters: { type: 'FeatureCollection', features: [] } });
                const expected = { center: [8, 47], zoom: 10, bearing: 20, pitch: 30 };
                await page.evaluate(camera => {
                    (window as any).mapResultsFixture.setCamera(camera);
                    (window as any).mapResultsFixture.emitResizeMoves = true;
                }, expected);
                await page.waitForTimeout(350);
                const before = await page.evaluate(() => ({
                    requests: (window as any).fixture.requests.filter((r: any) => r.channel === 'results').length,
                    publications: (window as any).fixture.publications.length,
                    url: location.href,
                }));
                const media = async (enabled: boolean) => page.evaluate(enabled => {
                    (window as any).setTestBrowserFullscreen(enabled, true, false);
                }, enabled);
                for (let cycle = 0; cycle < 2; cycle++) {
                    for (const enabled of [true, false]) {
                        const mapsBefore = await page.evaluate(() => (window as any).fixture.requests.filter((r: any) => r.channel === 'map').length);
                        const size = enabled ? { width: 1500, height: 1000 } : { width: 1280, height: 720 };
                        if (order === 'media-before' || order === 'resize-delayed') {
                            await media(enabled);
                            if (order === 'resize-delayed') await page.waitForTimeout(180);
                            await page.setViewportSize(size);
                        } else {
                            if (order === 'media-50ms-after') await page.evaluate(enabled => {
                                window.addEventListener('resize', () => setTimeout(() => (window as any).setTestBrowserFullscreen(enabled, true, false), 50), { once: true });
                            }, enabled);
                            await page.setViewportSize(size);
                            if (order !== 'media-50ms-after') await media(enabled);
                        }
                        await expect(page.locator('.map-page')).toHaveClass(enabled ? /fullscreen/ : /^(?!.*fullscreen)/);
                        await expect(page.locator('.map-results')).toHaveAttribute('data-layout-ready', 'true');
                        await page.waitForTimeout(650);
                        expect(await page.evaluate(() => (window as any).fixture.requests.filter((r: any) => r.channel === 'map').length)).toBe(mapsBefore + 1);
                        const mapRequest = await page.evaluate(() => (window as any).fixture.requests.filter((r: any) => r.channel === 'map').at(-1).body);
                        const bounds = await page.evaluate(() => (window as any).mapResultsFixture.renderedBounds());
                        for (const corner of ['northEast', 'southWest'] as const) for (const axis of ['lat', 'lng'] as const) expect(mapRequest[corner][axis]).toBeCloseTo(bounds[corner][axis], 10);
                        expect(await page.evaluate(() => (window as any).mapResultsFixture.camera())).toEqual(expected);
                        expect(await page.evaluate(() => ({
                            requests: (window as any).fixture.requests.filter((r: any) => r.channel === 'results').length,
                            publications: (window as any).fixture.publications.length,
                            url: location.href,
                        }))).toEqual(before);
                    }
                }
            });
        }
    });
}
