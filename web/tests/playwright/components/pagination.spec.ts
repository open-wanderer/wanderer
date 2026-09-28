import { expect, test, type Page } from '@playwright/test';
import { buildPaginationFixture, mountPaginationFixture } from './pagination-fixture';

function displaySelect(page: Page) {
    return page.locator('select').filter({ has: page.locator('option[value="cards"]') });
}

async function expectTrails(page: Page, first: number, count: number) {
    await expect(page.getByTestId('trail-id')).toHaveText(
        Array.from({ length: count }, (_, index) => String(first + index)),
    );
}

for (const dev of [true, false]) {
    test.describe(dev ? 'development' : 'production', () => {
        let bundle: string;
        const runtimeErrors: string[] = [];

        test.beforeAll(async () => {
            bundle = await buildPaginationFixture(dev);
        });

        test.beforeEach(async ({ page }) => {
            runtimeErrors.length = 0;
            page.on('pageerror', error => runtimeErrors.push(error.message));
            page.on('console', message => {
                // The independent filter ownership warning is outside this fix.
                if (/ownership_invalid_(mutation|binding)/.test(message.text()) && message.text().includes('pagination')) {
                    runtimeErrors.push(message.text());
                }
            });
        });

        test.afterEach(() => {
            expect(runtimeErrors).toEqual([]);
        });

        test('updates profile page count and loading when changing items per page', async ({ page }) => {
            const { requests, response } = await mountPaginationFixture(page, bundle);
            await expect(page.getByRole('button', { name: '10', exact: true }).first()).toBeVisible();

            let release!: () => void;
            response.gate = new Promise<void>(resolve => { release = resolve; });
            try {
                await page.locator('select').last().selectOption('96');
                await expect.poll(() => requests.at(-1)).toEqual({ page: 1, items: 96 });
                await expect(page.getByTestId('loading')).toHaveCount(96);
                await expect(page.getByTestId('trail-id')).toHaveCount(0);
            } finally {
                release();
            }
            await expectTrails(page, 1, 96);
            await expect(page.getByRole('button', { name: '10', exact: true })).toHaveCount(0);
            await expect(page.getByRole('button', { name: '3', exact: true })).toHaveCount(0);

            await page.getByRole('button', { name: '2', exact: true }).first().click();
            await expectTrails(page, 97, 24);
            expect(requests.at(-1)).toEqual({ page: 2, items: 96 });
        });

        test('refreshes profile pagination metadata after a filter update', async ({ page }) => {
            const { response } = await mountPaginationFixture(page, bundle);
            response.total = 24;
            await page.getByRole('button', { name: 'Change sort order' }).click();
            await expect(page.getByRole('button', { name: '10', exact: true })).toHaveCount(0);
            await page.getByRole('button', { name: '2', exact: true }).first().click();
            await expectTrails(page, 13, 12);
            await expect(page.getByRole('button', { name: '3', exact: true })).toHaveCount(0);
        });

        for (const [from, oldSize, to, newSize] of [
            ['list', 25, 'cards', 24],
            ['cards', 24, 'list', 25],
        ] as const) {
            test(`reloads page one when switching ${from} ${oldSize} to ${to} ${newSize}`, async ({ page }) => {
                const { requests } = await mountPaginationFixture(page, bundle, from, oldSize);
                await page.getByRole('button', { name: '2', exact: true }).first().click();
                await expectTrails(page, oldSize + 1, oldSize);
                const beforeSwitch = requests.length;

                await displaySelect(page).selectOption(to);
                await expectTrails(page, 1, newSize);
                expect(requests.slice(beforeSwitch)).toEqual([{ page: 1, items: newSize }]);
                expect(await page.evaluate(() => localStorage.getItem('paginationItems'))).toBe(String(newSize));
                const firstPage = await page.getByTestId('trail-id').allTextContents();

                await page.getByRole('button', { name: '2', exact: true }).first().click();
                await expectTrails(page, newSize + 1, newSize);
                const secondPage = await page.getByTestId('trail-id').allTextContents();
                expect([...firstPage, ...secondPage]).toEqual(
                    Array.from({ length: newSize * 2 }, (_, index) => String(index + 1)),
                );
            });
        }

        test('keeps the current page when switching list and table with the same size', async ({ page }) => {
            const { requests } = await mountPaginationFixture(page, bundle, 'list', 25);
            await page.getByRole('button', { name: '2', exact: true }).first().click();
            await expectTrails(page, 26, 25);
            const beforeSwitch = requests.length;

            await displaySelect(page).selectOption('table');
            await expectTrails(page, 26, 25);
            await displaySelect(page).selectOption('list');
            await expectTrails(page, 26, 25);
            expect(requests).toHaveLength(beforeSwitch);
        });
    });
}
