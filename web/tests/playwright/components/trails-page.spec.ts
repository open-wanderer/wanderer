import { expect, test, type Page } from '@playwright/test';
import { buildTrailsPageFixture, mountTrailsPageFixture } from './trails-page-fixture';

const area = (offset = 0) => ({ northEast: { lat: 48 + offset, lng: 9 }, southWest: { lat: 46 + offset, lng: 7 } });
const result = (prefix: string, page = 1) => ({
    items: Array.from({ length: 48 * page }, (_, index) => ({ id: `${prefix}-${index}` })),
    page, totalPages: 5, totalHits: 240,
});
const mapResult = (name: string, outside = true) => ({ mapTrails: [{ id: name }], clusters: { type: 'FeatureCollection', features: [] }, hasOutsideBounds: outside });
const readResults = (page: Page) => page.getByTestId('results').textContent().then(text => JSON.parse(text!));
type PageRequest = { channel: string; filter: { q: string }; pages?: number; bounds?: ReturnType<typeof area>; aborted: boolean };
const requests = (page: Page): Promise<PageRequest[]> => page.evaluate(() => (window as any).fixture.requests.map((request: any) => ({
    channel: request.channel, ...request.body, aborted: request.signal?.aborted ?? false,
})));
const resolveRequest = (page: Page, index: number, data: object) => page.evaluate(({ index, data }) => (window as any).fixture.requests[index].resolve(data), { index, data });
const setView = (page: Page, view: string) => page.evaluate(view => (window as any).fixture.display(view), view);
const viewport = (page: Page, bounds: ReturnType<typeof area>, entering = false) => page.evaluate(({ bounds, entering }) => (window as any).fixture.viewport(bounds, 11, entering), { bounds, entering });

for (const dev of [true, false]) {
    test.describe(`trails page in ${dev ? 'development' : 'production'}`, () => {
        let bundle: string;
        test.beforeAll(async () => { bundle = await buildTrailsPageFixture(dev); });

        test('keeps cards, list and table views when applying a selected place', async ({ page }) => {
            await mountTrailsPageFixture(page, bundle);
            await resolveRequest(page, 0, result('initial'));
            for (const [index, view] of ['cards', 'list', 'table'].entries()) {
                await setView(page, view);
                const near = { lat: 47 + index, lon: 8, name: 'Selected place', radius: 3000 };
                await page.evaluate(near => (window as any).fixture.apply({ q: '', near }, true), near);
                await expect.poll(() => requests(page)).toHaveLength(index + 2);
                expect((await requests(page))[index + 1]).toMatchObject({ channel: 'results', filter: { q: '', near } });
                await resolveRequest(page, index + 1, result(view));
                await expect.poll(() => readResults(page)).toMatchObject({ display: view, filter: { q: '', near }, loading: false });
                expect(await page.evaluate(() => history.state.trailSearchView)).toBe(view);
                await expect(page.getByTestId('map')).toHaveCount(0);
            }
        });

        test('focuses the selected place in map view, including a repeated selection', async ({ page }) => {
            await mountTrailsPageFixture(page, bundle);
            await resolveRequest(page, 0, result('initial'));
            await setView(page, 'map');
            await expect(page.getByTestId('map')).toBeAttached();
            const near = { lat: 46.95, lon: 7.44, name: 'Bern', radius: 5000 };
            await page.evaluate(near => (window as any).fixture.apply({ q: '', near }, true), near);
            await expect.poll(() => page.evaluate(() => (window as any).fixture.focusedPlaces)).toEqual([near]);
            await page.evaluate(near => (window as any).fixture.apply({ q: '', near }, true), near);
            await expect.poll(() => page.evaluate(() => (window as any).fixture.focusedPlaces)).toEqual([near, near]);
            expect((await readResults(page)).display).toBe('map');
            expect(await page.evaluate(() => history.state.trailSearchView)).toBe('map');
        });

        test('ignores a late appended page after a newer search, including aborted replies', async ({ page }) => {
            await mountTrailsPageFixture(page, bundle);
            await resolveRequest(page, 0, result('initial'));
            await expect.poll(() => readResults(page)).toMatchObject({ loading: false });
            await page.getByRole('button', { name: 'trail-results-load-more', exact: true }).click();
            await page.evaluate(() => (window as any).fixture.apply({ q: 'new' }));
            await expect.poll(() => requests(page)).toHaveLength(3);
            expect((await requests(page))[1].aborted).toBe(true);
            await resolveRequest(page, 2, result('new'));
            await expect.poll(() => readResults(page)).toMatchObject({ trails: result('new').items, filter: { q: 'new' } });
            await resolveRequest(page, 1, result('old-page', 2));
            await expect.poll(() => readResults(page)).toMatchObject({ trails: result('new').items, pagination: { page: 1 }, loading: false });
            await expect(page.getByRole('alert')).toHaveCount(0);
        });

        test('flushes a pending viewport on exit and retains the loaded prefix until its replacement arrives', async ({ page }) => {
            await mountTrailsPageFixture(page, bundle);
            await resolveRequest(page, 0, result('initial'));
            await page.getByRole('button', { name: 'trail-results-load-more', exact: true }).click();
            await resolveRequest(page, 1, result('initial', 2));
            await expect.poll(() => readResults(page)).toMatchObject({ pagination: { page: 2 } });
            await page.clock.install();
            await setView(page, 'map');
            await expect(page.getByTestId('map')).toBeAttached();
            await viewport(page, area(), true);
            expect(await requests(page)).toHaveLength(2);
            await setView(page, 'list');
            await expect.poll(() => requests(page)).toHaveLength(3);
            expect((await requests(page))[2].channel).toBe('map');
            await page.clock.runFor(250);
            expect(await requests(page)).toHaveLength(3);
            await resolveRequest(page, 2, mapResult('map'));
            await expect.poll(() => requests(page)).toHaveLength(4);
            expect((await requests(page))[3]).toMatchObject({ channel: 'results', pages: 2, bounds: area() });
            expect((await readResults(page)).trails).toHaveLength(96);
            expect((await readResults(page)).loading).toBe(false);
            await resolveRequest(page, 3, result('scoped', 2));
            await expect.poll(() => readResults(page)).toMatchObject({ trails: result('scoped', 2).items, pagination: { page: 2 } });
        });

        test('ignores a late map reply when the camera returns to the same bounds', async ({ page }) => {
            await mountTrailsPageFixture(page, bundle);
            await resolveRequest(page, 0, result('initial'));
            await page.clock.install();
            await setView(page, 'map');
            await expect(page.getByTestId('map')).toBeAttached();
            await viewport(page, area(), true);
            await page.clock.runFor(201);
            await expect.poll(() => requests(page)).toHaveLength(2);
            await viewport(page, area(1));
            await viewport(page, area());
            await page.clock.runFor(201);
            await expect.poll(() => requests(page)).toHaveLength(3);
            expect((await requests(page))[1].aborted).toBe(true);
            await resolveRequest(page, 2, mapResult('current', false));
            await expect(page.getByTestId('map')).toContainText('current');
            await resolveRequest(page, 1, mapResult('obsolete'));
            await page.evaluate(() => new Promise(resolve => setTimeout(resolve, 0)));
            await expect(page.getByTestId('map')).toContainText('current');
            expect(await requests(page)).toHaveLength(3);
        });

        test('cancels the viewport debounce when applying criteria and on unmount', async ({ page }) => {
            await mountTrailsPageFixture(page, bundle);
            await resolveRequest(page, 0, result('initial'));
            await page.clock.install();
            await setView(page, 'map');
            await expect(page.getByTestId('map')).toBeAttached();
            await viewport(page, area(), true);
            await page.evaluate(() => (window as any).fixture.apply({ q: 'new' }));
            await expect.poll(() => requests(page)).toHaveLength(3);
            expect((await requests(page)).slice(1).map(request => request.filter.q)).toEqual(['new', 'new']);
            await page.clock.runFor(250);
            expect(await requests(page)).toHaveLength(3);
            await viewport(page, area(1));
            await page.evaluate(() => (window as any).fixture.destroy());
            await page.clock.runFor(250);
            expect(await requests(page)).toHaveLength(3);
            expect((await requests(page)).slice(1).every(request => request.aborted)).toBe(true);
        });

        test('restores viewport scope and the loaded prefix through history and snapshots', async ({ page }) => {
            await mountTrailsPageFixture(page, bundle);
            await resolveRequest(page, 0, result('initial'));
            await page.getByRole('button', { name: 'trail-results-load-more', exact: true }).click();
            await resolveRequest(page, 1, result('initial', 2));
            await setView(page, 'map');
            await expect(page.getByTestId('map')).toBeAttached();
            await viewport(page, area(), true);
            await expect.poll(() => requests(page)).toHaveLength(3);
            await resolveRequest(page, 2, mapResult('wide', false));
            await setView(page, 'list');
            const saved = await page.evaluate(() => (window as any).fixture.capture());
            await setView(page, 'map');
            await viewport(page, area(1));
            await expect.poll(() => requests(page)).toHaveLength(4);
            await resolveRequest(page, 3, mapResult('narrow'));
            await expect.poll(() => requests(page)).toHaveLength(5);
            await resolveRequest(page, 4, result('narrow'));
            await page.goBack();
            await expect.poll(() => requests(page)).toHaveLength(6);
            expect((await requests(page))[5]).toMatchObject({ channel: 'results', pages: 1 });
            expect((await requests(page))[5].bounds).toBeUndefined();
            await resolveRequest(page, 5, result('restored'));
            await page.evaluate(saved => (window as any).fixture.restore(saved), saved);
            await expect.poll(() => requests(page)).toHaveLength(7);
            expect((await requests(page))[6].bounds).toBeUndefined();
            await resolveRequest(page, 6, result('snapshot'));
            await expect.poll(() => readResults(page)).toMatchObject({ trails: result('snapshot').items, display: 'list', loading: false });
        });
    });
}
