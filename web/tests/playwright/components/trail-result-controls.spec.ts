import { expect, test, type Page } from '@playwright/test';
import { buildTrailResultControlsFixture, mountTrailResultControlsFixture } from './trail-result-controls-fixture';

const sortButton = (page: Page) => page.getByRole('button', { name: /^sort: / });
const viewButton = (page: Page) => page.getByRole('button', { name: /^trail-results-view: / });
const snapshot = (page: Page) => page.evaluate(() => (window as any).resultSnapshot);

for (const dev of [true, false]) {
    test.describe(`compact trail results in ${dev ? 'development' : 'production'}`, () => {
        let bundle: string;
        let errors: string[];
        test.beforeAll(async () => { bundle = await buildTrailResultControlsFixture(dev); });
        test.beforeEach(({ page }) => { errors = []; page.on('pageerror', error => errors.push(error.message)); });
        test.afterEach(() => { expect(errors).toEqual([]); });

        test('offers search layout choices and hides side positions on the map without refetching', async ({ page }) => {
            const requests = await mountTrailResultControlsFixture(page, bundle, { incremental: true, searchLayout: true });
            const initialRequests = requests.length;
            await viewButton(page).click();
            await page.getByRole('radio', { name: 'search-layout-left', exact: true }).check();
            await expect.poll(() => snapshot(page)).toMatchObject({ searchPosition: 'left' });
            await expect(page.getByRole('radio', { name: 'search-layout-overlay', exact: true })).toHaveCount(0);
            await page.getByRole('radio', { name: 'map', exact: true }).check();
            await expect(page.getByRole('radio', { name: 'search-layout-left', exact: true })).toHaveCount(0);
            await expect(page.getByRole('radio', { name: 'search-layout-right', exact: true })).toHaveCount(0);
            await page.getByRole('radio', { name: 'search-layout-push', exact: true }).check();
            await expect.poll(() => snapshot(page)).toMatchObject({ searchExpansion: 'push' });
            await page.getByRole('radio', { name: 'card', exact: true }).check();
            await expect(page.getByRole('radio', { name: 'search-layout-left', exact: true })).toBeChecked();
            await page.getByRole('radio', { name: 'search-layout-top', exact: true }).check();
            await expect(page.getByRole('radio', { name: 'search-layout-push', exact: true })).toBeChecked();
            await expect(page.locator('input[value="bottom"]')).toHaveCount(0);
            expect(requests).toHaveLength(initialRequests);
        });

        test('preserves direction on a criterion change and restores criterion and direction after reload', async ({ page }) => {
            const requests = await mountTrailResultControlsFixture(page, bundle, { order: '-' });
            await sortButton(page).click();
            await expect(page.getByRole('region', { name: 'sort', exact: true }).getByRole('radio')).toHaveCount(9);
            await page.getByRole('radio', { name: 'distance', exact: true }).check();
            await expect.poll(() => requests.at(-1)).toMatchObject({ sort: 'distance', order: '-' });
            const direction = page.getByRole('button', { name: /^trail-results-toggle-order:/ });
            await expect(page.locator('.sort-heading')).toContainText('trail-results-sort-by');
            await expect(page.locator('.direction-section')).toHaveCount(0);
            await direction.click();
            await expect.poll(() => requests.at(-1)).toMatchObject({ sort: 'distance', order: '+' });
            await expect(page.getByRole('region', { name: 'sort', exact: true })).toBeVisible();
            await expect(direction).toHaveAttribute('aria-label', 'trail-results-toggle-order: trail-results-distance-asc');
            await expect(direction.locator('i')).not.toHaveClass(/descending/);
            await direction.press('Enter');
            await expect.poll(() => requests.at(-1)).toMatchObject({ sort: 'distance', order: '-' });
            await expect(direction.locator('i')).toHaveClass(/descending/);
            await direction.press('Space');
            await expect.poll(() => requests.at(-1)).toMatchObject({ sort: 'distance', order: '+' });
            await direction.click();
            await expect.poll(() => requests.at(-1)).toMatchObject({ sort: 'distance', order: '-' });
            expect(await page.evaluate(() => [localStorage.getItem('sort'), localStorage.getItem('sort_order')]))
                .toEqual(['distance', '-']);

            await page.reload();
            await page.addScriptTag({ content: bundle });
            await expect(sortButton(page)).toHaveAttribute('aria-label', 'sort: trail-results-distance-desc');
            await expect.poll(() => snapshot(page)).toMatchObject({ filter: { sort: 'distance', sortOrder: '-' } });
        });

        test('uses real view and page-size callbacks without refetching the same-size table view', async ({ page }) => {
            const requests = await mountTrailResultControlsFixture(page, bundle);
            const before = requests.length;
            await viewButton(page).click();
            await page.getByRole('radio', { name: 'list', exact: true }).check();
            await expect(page.locator('[data-view="list"]')).toHaveCount(10);
            expect(requests.slice(before)).toEqual([{ page: 1, items: 10, sort: 'name', order: '+' }]);
            await page.getByTestId('outside').click();
            await page.getByRole('button', { name: '2', exact: true }).click();
            await expect.poll(() => requests.at(-1)).toMatchObject({ page: 2, items: 10 });

            const beforeTable = requests.length;
            await viewButton(page).click();
            await page.getByRole('radio', { name: 'table', exact: true }).check();
            await expect(page.getByTestId('table-view')).toBeVisible();
            await expect.poll(() => snapshot(page)).toMatchObject({ display: 'table', pagination: { page: 2, items: 10 } });
            expect(requests).toHaveLength(beforeTable);

            await page.getByRole('button', { name: '25', exact: true }).click();
            await expect(page.getByTestId('trail-id')).toHaveCount(25);
            expect(requests.slice(beforeTable)).toEqual([{ page: 1, items: 25, sort: 'name', order: '+' }]);
            expect(await page.evaluate(() => [localStorage.getItem('displayOption'), localStorage.getItem('paginationItems')]))
                .toEqual(['table', '25']);
        });

        test('controls per-view width and provides two mobile icon menus with Escape and outside dismissal', async ({ page }) => {
            await page.setViewportSize({ width: 390, height: 844 });
            await mountTrailResultControlsFixture(page, bundle, { sort: 'distance', widths: { cards: false, list: true } });
            await expect(page.locator('.result-buttons .control-button')).toHaveCount(2);
            for (const copy of await page.locator('.control-copy').all()) await expect(copy).toBeHidden();
            await viewButton(page).click();
            const width = page.getByRole('switch', { name: 'trail-results-wide' });
            await expect(width).toHaveAttribute('aria-checked', 'false');
            await width.click();
            await expect(width).toHaveAttribute('aria-checked', 'true');
            await page.getByRole('radio', { name: 'list', exact: true }).check();
            await expect(width).toHaveAttribute('aria-checked', 'true');
            await width.click();
            await expect(width).toHaveAttribute('aria-checked', 'false');
            await page.getByRole('radio', { name: 'card', exact: true }).check();
            await expect(width).toHaveAttribute('aria-checked', 'true');
            expect(await page.evaluate(() => JSON.parse(localStorage.getItem('trailWidthPreferences')!)))
                .toEqual({ cards: true, list: false });
            expect(await page.evaluate(() => (window as any).resultEvents.filter((event: any) => event.type === 'width')))
                .toEqual([{ type: 'width', display: 'cards', wide: true }, { type: 'width', display: 'list', wide: false }]);

            await width.press('Escape');
            await expect(page.getByRole('region', { name: 'trail-results-view', exact: true })).toBeHidden();
            await expect(viewButton(page)).toBeFocused();
            await sortButton(page).press('ArrowDown');
            const sortMenu = page.getByRole('region', { name: 'sort', exact: true });
            await expect(sortMenu).toBeVisible();
            await expect(sortMenu.getByRole('radio', { checked: true })).toHaveValue('distance');
            await expect(sortMenu.getByRole('radio', { checked: true })).toBeFocused();
            await page.getByTestId('outside').click();
            await expect(page.getByRole('region', { name: 'sort', exact: true })).toBeHidden();
        });

        test('offers relevance for a text query, hides its direction button and keeps explicit sorting separate from legacy lists', async ({ page }) => {
            const requests = await mountTrailResultControlsFixture(page, bundle, { incremental: true, query: 'Bern', sort: 'relevance' });
            await sortButton(page).click();
            const relevance = page.getByRole('radio', { name: 'trail-results-relevance', exact: true });
            await expect(relevance).toBeChecked();
            await expect(page.locator('.sort-direction-button')).toHaveCount(0);
            await page.getByRole('radio', { name: 'distance', exact: true }).check();
            await expect.poll(() => requests.at(-1)).toMatchObject({ sort: 'distance' });
            await expect(page.locator('.sort-direction-button')).toBeVisible();
            expect(await page.evaluate(() => localStorage.getItem('trailSearchSort'))).toBe('distance');
            expect(await page.evaluate(() => localStorage.getItem('sort'))).toBe('relevance');
            await relevance.check();
            await expect.poll(() => requests.at(-1)).toMatchObject({ sort: 'relevance' });
            await expect(page.locator('.sort-direction-button')).toHaveCount(0);
            await page.keyboard.press('Escape');
            await viewButton(page).click();
            await page.locator('.view-menu input[value="table"]').check();
            await expect(page.getByTestId('table-view')).toBeVisible();
        });

        test('shared search offers map and appends results without pagination or page-size controls', async ({ page }) => {
            const requests = await mountTrailResultControlsFixture(page, bundle, { incremental: true, items: 100 });
            await expect(page.getByRole('button', { name: 'Next page', exact: true })).toHaveCount(0);
            await page.getByRole('button', { name: 'Load more', exact: true }).click();
            await expect(page.getByTestId('trail-id')).toHaveCount(96);
            for (const mode of ['list', 'table', 'map', 'cards']) {
                await viewButton(page).click();
                await expect(page.locator('.size-section')).toHaveCount(0);
                await page.locator(`.view-menu input[value="${mode}"]`).check();
                await page.keyboard.press('Escape');
                if (mode === 'map') {
                    await expect(page.getByTestId('map-view')).toBeVisible();
                    const width = page.getByRole('switch', { name: 'trail-results-wide' });
                    await viewButton(page).click();
                    await expect(width).toBeVisible();
                    await width.click();
                    await expect(width).toHaveAttribute('aria-checked', 'true');
                    await page.keyboard.press('Escape');
                }
                else await expect(page.getByTestId('trail-id')).toHaveCount(96);
            }
            expect(requests).toHaveLength(2);
            await page.getByRole('button', { name: 'Load more', exact: true }).click();
            await expect(page.getByTestId('trail-id')).toHaveCount(120);
            await expect(page.getByRole('button', { name: 'Load more', exact: true })).toHaveCount(0);
            const ids = await page.getByTestId('trail-id').allTextContents();
            expect(new Set(ids).size).toBe(120);
            expect(await page.evaluate(() => JSON.parse(localStorage.getItem('trailWidthPreferences')!).map)).toBe(true);
            await viewButton(page).click();
            await page.locator('.view-menu input[value="map"]').check();
            await page.keyboard.press('Escape');
            await page.getByTestId('map-view').evaluate(node => { (window as any).retainedMap = node; });
            await viewButton(page).click();
            await page.locator('.view-menu input[value="list"]').check();
            await page.keyboard.press('Escape');
            await expect(page.getByTestId('map-view')).toBeHidden();
            await viewButton(page).click();
            await page.locator('.view-menu input[value="map"]').check();
            await page.keyboard.press('Escape');
            expect(await page.getByTestId('map-view').evaluate(node => node === (window as any).retainedMap)).toBe(true);
            expect(requests).toHaveLength(3);
        });
    });
}
