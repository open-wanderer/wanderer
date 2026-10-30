import { expect, test, type Page } from '@playwright/test';
import { buildTrailsPageFixture, mountTrailsPageFixture } from './trails-page-fixture';

const result = (prefix: string, page = 1, count = 48) => ({
    items: Array.from({ length: count }, (_, index) => ({ id: `${prefix}-${index}` })), page, totalPages: 3, totalHits: 144,
});
const requests = (page: Page) => page.evaluate(() => (window as any).fixture.requests.map((request: any) => ({
    channel: request.channel, ...request.body, aborted: request.signal?.aborted ?? false,
})));
const resolve = (page: Page, index: number, data: unknown) => page.evaluate(({ index, data }) => (window as any).fixture.requests[index].resolve(data), { index, data });
const readResults = (page: Page) => page.getByTestId('results').textContent().then(text => JSON.parse(text!));
const initial = async (page: Page) => {
    await resolve(page, 0, result('initial'));
    await expect(page.locator('.result-content')).toHaveAttribute('aria-busy', 'false');
};

for (const dev of [true, false]) {
    test.describe(`trail list page in ${dev ? 'development' : 'production'}`, () => {
        let bundle: string;
        test.beforeAll(async () => { bundle = await buildTrailsPageFixture(dev); });
        test.beforeEach(({ page }) => { page.on('pageerror', error => { throw error; }); });

        test('starts a list search without a viewport or cluster query, and ignores map area parameters', async ({ page }) => {
            await mountTrailsPageFixture(page, bundle, { query: 'tl_lat=48&tl_lon=9&br_lat=46&br_lon=7' });
            expect(await requests(page)).toEqual([expect.objectContaining({ channel: 'results', page: 1, perPage: 48 })]);
            expect((await requests(page))[0]).not.toHaveProperty('bounds');
            expect((await requests(page))[0].filter).toMatchObject({ q: '', sort: 'relevance', sortOrder: '+' });
            await expect(page.locator('.result-content')).toHaveAttribute('aria-busy', 'true');
            await initial(page);
        });

        test('keeps a place as a radius filter in every result presentation', async ({ page }) => {
            await mountTrailsPageFixture(page, bundle);
            await initial(page);
            const near = { lat: 46.95, lon: 7.44, name: 'Bern', radius: 5000 };
            for (const display of ['cards', 'list', 'table']) {
                await page.evaluate(display => (window as any).fixture.display(display), display);
                await page.evaluate(near => (window as any).fixture.apply({ near }), near);
                expect((await requests(page)).at(-1)).toMatchObject({ channel: 'results', filter: { near } });
                expect((await readResults(page)).display).toBe(display);
            }
            expect((await requests(page)).every((request: any) => !request.bounds && request.channel === 'results')).toBe(true);
        });

        test('shows refresh progress and ignores a late appended page after new criteria', async ({ page }) => {
            await mountTrailsPageFixture(page, bundle);
            await initial(page);
            await page.getByRole('button', { name: 'trail-results-load-more', exact: true }).click();
            await page.evaluate(() => (window as any).fixture.apply({ q: 'new' }));
            await expect(page.locator('.result-content')).toHaveAttribute('aria-busy', 'true');
            expect((await requests(page))[1].aborted).toBe(true);
            expect((await readResults(page)).trails).toHaveLength(48);
            await expect(page.getByRole('button', { name: 'trail-results-load-more', exact: true })).toHaveCount(0);
            await resolve(page, 2, result('current'));
            await resolve(page, 1, result('obsolete', 2));
            await expect(page.locator('.result-content')).toHaveAttribute('aria-busy', 'false');
            expect((await readResults(page)).trails).toEqual(result('current').items);
        });

        test('retains the loaded prefix and scroll position after a bulk deletion on page two', async ({ page }) => {
            await mountTrailsPageFixture(page, bundle);
            await initial(page);
            await page.getByRole('button', { name: 'trail-results-load-more', exact: true }).click();
            await resolve(page, 1, result('second', 2));
            await expect.poll(() => readResults(page).then(value => value.trails.length)).toBe(96);
            await page.evaluate(() => window.scrollTo(0, 300));
            await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(300);
            await page.evaluate(() => { void (window as any).fixture.refreshResults(); });
            await expect.poll(() => requests(page).then(value => value.length)).toBe(3);
            expect((await requests(page))[2]).toMatchObject({ page: 1, perPage: 96 });
            const remaining = [...result('initial').items, ...result('second', 2).items.slice(1)];
            await resolve(page, 2, { items: remaining, page: 1, totalHits: 143, totalPages: 2 });
            await expect.poll(() => readResults(page).then(value => value.trails.length)).toBe(95);
            expect((await readResults(page)).pagination.page).toBe(2);
            await expect(page.locator('.result-content')).toHaveAttribute('aria-busy', 'false');
            expect(await page.evaluate(() => window.scrollY)).toBe(300);
        });

        test('aborts pending searches on unmount', async ({ page }) => {
            await mountTrailsPageFixture(page, bundle);
            await page.evaluate(() => (window as any).fixture.destroy());
            expect((await requests(page))[0].aborted).toBe(true);
            await resolve(page, 0, result('obsolete'));
            await expect(page.getByTestId('results')).toHaveCount(0);
        });

        test('shares the latest filter and sort with a new tab without allowing filter storage to override sort', async ({ page, context }) => {
            await mountTrailsPageFixture(page, bundle);
            await page.evaluate(() => {
                (window as any).fixture.apply({ q: 'saved', category: ['hiking'] });
                (window as any).fixture.sort('name', '-');
                const stored = JSON.parse(localStorage.getItem('trailListFilter')!);
                localStorage.setItem('trailListFilter', JSON.stringify({ ...stored, sort: 'distance', sortOrder: '+' }));
            });
            const other = await context.newPage();
            await mountTrailsPageFixture(other, bundle);
            expect((await requests(other))[0].filter).toMatchObject({ q: 'saved', category: ['hiking'], sort: 'name', sortOrder: '-' });
            await other.close();
        });

        for (const storage of [
            { sort: 'date', sort_order: '-', trailListFilter: { q: 'saved', sort: 'name', sortOrder: '+' } },
            { sort: 'date', sort_order: '+', trailSearchSort: 'name', trailSearchSortOrder: '-', trailListFilter: { q: 'saved', sort: 'distance' } },
        ]) {
            test(`restores the dedicated sort preferences with legacy fallback (${JSON.stringify(storage)})`, async ({ page }) => {
                await mountTrailsPageFixture(page, bundle, { storage });
                expect((await requests(page))[0].filter).toMatchObject({ q: 'saved', sort: 'trailSearchSort' in storage ? 'name' : 'date', sortOrder: '-' });
            });
        }

        test('gives linked author, category and subcategory precedence and ignores map area parameters', async ({ page }) => {
            await mountTrailsPageFixture(page, bundle, {
                query: 'author=linked-author&category=hiking&subcategory=alpine&tl_lat=48&tl_lon=9&br_lat=46&br_lon=7',
                storage: { trailListFilter: { q: 'stored', category: ['cycling'] } },
            });
            expect((await requests(page))[0]).toMatchObject({ filter: { q: '', author: 'linked-author', category: ['hiking'], subcategory: ['alpine'] } });
            expect((await requests(page))[0]).not.toHaveProperty('bounds');
        });

        test('clears filter storage on unrelated navigation while retaining sort for a new tab', async ({ page, context }) => {
            await mountTrailsPageFixture(page, bundle);
            await page.evaluate(() => {
                (window as any).fixture.apply({ q: 'saved' });
                (window as any).fixture.sort('name', '-');
                for (const pathname of ['/trails', '/trails/recent', '/map']) (window as any).fixture.beforeNavigate({ to: { url: new URL(pathname, location.href) } });
            });
            expect(await page.evaluate(() => JSON.parse(localStorage.getItem('trailListFilter')!).q)).toBe('saved');
            await page.evaluate(() => (window as any).fixture.beforeNavigate({ to: { url: new URL('/profile/@alice', location.href) } }));
            expect(await page.evaluate(() => localStorage.getItem('trailListFilter'))).toBeNull();
            const other = await context.newPage();
            await mountTrailsPageFixture(other, bundle);
            expect((await requests(other))[0].filter).toMatchObject({ q: '', sort: 'name', sortOrder: '-' });
            await other.close();
        });

        test('accepts current and legacy snapshots without resetting table presentation or the saved sort', async ({ page }) => {
            await mountTrailsPageFixture(page, bundle, { storage: { displayOption: 'table', trailSearchSort: 'name', trailSearchSortOrder: '-' } });
            await initial(page);
            const saved = await page.evaluate(() => (window as any).fixture.capture());
            expect(saved).toHaveProperty('filter');
            expect(saved).not.toHaveProperty('bounds');
            for (const snapshot of [saved, saved.filter]) {
                await page.evaluate(snapshot => (window as any).fixture.restore(snapshot), { ...snapshot, sort: 'created' });
                expect((await readResults(page)).display).toBe('table');
                expect((await requests(page)).at(-1).filter).toMatchObject({ sort: 'name', sortOrder: '-' });
            }
            expect(await page.evaluate(() => localStorage.getItem('displayOption'))).toBe('table');
        });

        test('persists the selected result presentation', async ({ page }) => {
            await mountTrailsPageFixture(page, bundle, { storage: { displayOption: 'table' } });
            expect((await readResults(page)).display).toBe('table');
            await page.evaluate(() => (window as any).fixture.display('list'));
            expect(await page.evaluate(() => localStorage.getItem('displayOption'))).toBe('list');
        });
    });
}
