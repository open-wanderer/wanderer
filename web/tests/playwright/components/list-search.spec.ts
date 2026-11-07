import { expect, test, type Page } from '../fixtures';
import { buildTrailModalFixture, mountTrailModalFixture } from './trail-modal-fixture';

const source = `
    <script>
        import ListSearchBar from '#lib/components/list/list_search_bar.svelte';
        import { currentUser } from '#lib/stores/user_store.ts';
        currentUser.set(window.listSearchAnonymous ? null : { id: 'tester', actor: 'tester' });
        let filter = $state({ q: '', author: '', public: true, shared: true, sort: 'created', sortOrder: '-' });
        let applications = $state(0);
    </script>
    <main><ListSearchBar bind:filter onapply={() => applications++} /></main>
    <output data-testid="filter">{JSON.stringify(filter)}</output>
    <output data-testid="applications">{applications}</output>
    <button data-testid="outside">Outside</button>
    <style>
        :global(body) { --input-background: 245,245,245; --input-border: 120,120,120; --input-border-focus: 30,80,120; --menu-background: 250,250,250; --content: 30,30,30; --secondary-hover: 220,225,230; }
        main { margin: 20px; max-width: 1000px; }
    </style>
`;
const input = (page: Page) => page.getByRole('searchbox', { name: 'search-list', exact: true });
const panel = (page: Page) => page.getByRole('region', { name: 'trail-search-filter-by', exact: true });
const reset = (page: Page) => page.getByRole('button', { name: 'trail-search-reset', exact: true });
const toggle = (page: Page) => page.locator('.search-heading button[aria-expanded]');
const filter = async (page: Page) => JSON.parse((await page.getByTestId('filter').textContent())!);

for (const dev of [true, false]) {
    test.describe(`list search in ${dev ? 'development' : 'production'}`, () => {
        let bundle: string;
        test.beforeAll(async () => { bundle = await buildTrailModalFixture(dev, source); });

        for (const anonymous of [false, true]) {
            test(`Escape clears collapsed text without opening the panel (anonymous=${anonymous})`, async ({ page }) => {
                await page.addInitScript(anonymous => { (window as any).listSearchAnonymous = anonymous; }, anonymous);
                await mountTrailModalFixture(page, bundle);
                await input(page).fill('Bern');
                await expect.poll(() => filter(page)).toMatchObject({ q: 'Bern' });
                if (!anonymous) {
                    await expect(panel(page)).toBeVisible();
                    await input(page).press('Escape');
                    await expect(input(page)).toHaveValue('Bern');
                    expect((await filter(page)).q).toBe('Bern');
                }
                await expect(panel(page)).toHaveCount(0);
                await input(page).press('Escape');
                await expect(input(page)).toHaveValue('');
                await expect.poll(() => filter(page)).toMatchObject({ q: '' });
                await expect(panel(page)).toHaveCount(0);
                const applications = await page.getByTestId('applications').textContent();
                await input(page).press('Escape');
                await expect(page.getByTestId('applications')).toHaveText(applications!);
                await expect(panel(page)).toHaveCount(0);
            });
        }

        for (const activation of ['pointer', 'keyboard'] as const) {
            test(`focuses the input only when closing with the keyboard: ${activation}`, async ({ page }) => {
                await mountTrailModalFixture(page, bundle);
                if (activation === 'keyboard') { await toggle(page).focus(); await toggle(page).press('Enter'); }
                else await toggle(page).click();
                await expect(panel(page)).toBeVisible();
                await expect(toggle(page)).toBeFocused();
                if (activation === 'keyboard') await toggle(page).press('Enter');
                else await toggle(page).click();
                await expect(panel(page)).toHaveCount(0);
                if (activation === 'keyboard') await expect(input(page)).toBeFocused();
                else await expect(toggle(page)).toBeFocused();
            });
        }

        test.describe('touch filter controls', () => {
            test.use({ hasTouch: true, viewport: { width: 420, height: 800 } });

            test('keeps focus on the filter toggle after opening and closing by tap', async ({ page }) => {
                await mountTrailModalFixture(page, bundle);
                await expect.poll(() => page.evaluate(() => innerWidth)).toBe(420);
                await toggle(page).evaluate(element => {
                    (window as any).filterToggleDetails = [];
                    element.addEventListener('click', event => (window as any).filterToggleDetails.push((event as MouseEvent).detail));
                });
                await toggle(page).tap();
                await expect(panel(page)).toBeVisible();
                await expect(toggle(page)).toBeFocused();
                await expect(input(page)).not.toBeFocused();
                await panel(page).locator('.filter-caption').tap();
                await toggle(page).tap();
                await expect(panel(page)).toHaveCount(0);
                await expect(toggle(page)).toBeFocused();
                await expect(input(page)).not.toBeFocused();
                expect(await page.evaluate(() => (window as any).filterToggleDetails)).toEqual([1, 1]);
            });
        });

        for (const expanded of [false, true]) {
            for (const activation of ['pointer', 'keyboard'] as const) {
                test(`hides an empty reset and restores filter focus after ${activation} reset with expanded=${expanded}`, async ({ page }) => {
                    await mountTrailModalFixture(page, bundle);
                    await expect(reset(page)).toHaveCount(0);
                    await input(page).click();
                    await expect(panel(page)).toBeVisible();
                    await expect(reset(page)).toHaveCount(0);
                    await input(page).fill('Bern');
                    await expect(reset(page)).toBeVisible();
                    await expect(reset(page).locator('i')).toHaveClass(/\bfa-filter-circle-xmark\b/);
                    await page.locator('.filter-chips').getByRole('button', { name: 'visibilty-status', exact: true }).click();
                    await page.getByRole('checkbox', { name: 'shared', exact: true }).uncheck();
                    await expect.poll(() => filter(page)).toMatchObject({ q: 'Bern', shared: false });
                    if (!expanded) {
                        await page.keyboard.press('Escape');
                        await expect(panel(page)).toHaveCount(0);
                    }
                    await expect(reset(page)).toBeVisible();
                    if (activation === 'keyboard') { await reset(page).focus(); await reset(page).press('Enter'); }
                    else await reset(page).click();
                    await expect(reset(page)).toHaveCount(0);
                    await expect(input(page)).toHaveValue('');
                    await expect.poll(() => filter(page)).toEqual({ q: '', author: '', public: true, shared: true, sort: 'created', sortOrder: '-' });
                    await expect(toggle(page)).toBeFocused();
                    await expect(toggle(page)).toHaveAttribute('aria-expanded', String(expanded));
                    if (expanded) {
                        await page.keyboard.press('Escape');
                        await expect(panel(page)).toHaveCount(0);
                        await expect(input(page)).toBeFocused();
                    }
                });
            }
        }

        for (const activation of ['pointer', 'keyboard'] as const) {
            test(`keeps anonymous text search closed and restores adjacent control focus after ${activation} reset`, async ({ page }) => {
                await page.addInitScript(() => { (window as any).listSearchAnonymous = true; });
                await mountTrailModalFixture(page, bundle);
                await expect(toggle(page)).toHaveCount(0);
                await expect(reset(page)).toHaveCount(0);
                if (activation === 'pointer') {
                    await page.setViewportSize({ width: 900, height: 600 });
                    await page.addStyleTag({ content: 'body { min-height: 1600px; }' });
                    await page.locator('main').evaluate(element => element.style.marginTop = '460px');
                }
                await input(page).click();
                await expect.poll(() => page.evaluate(() => scrollY)).toBe(0);
                await input(page).press('ArrowDown');
                await expect(panel(page)).toHaveCount(0);
                await expect(page.locator('.search-form')).not.toHaveClass(/\bexpanded\b/);
                await expect(input(page)).not.toHaveAttribute('aria-controls', /.+/);
                await input(page).fill('Bern');
                await expect.poll(() => filter(page)).toMatchObject({ q: 'Bern' });
                if (activation === 'keyboard') await input(page).press('Enter');
                await expect(reset(page)).toBeVisible();
                await expect(reset(page).locator('i')).toHaveClass(/\bfa-filter-circle-xmark\b/);
                if (activation === 'keyboard') { await reset(page).focus(); await reset(page).press('Enter'); }
                else await reset(page).click();
                await expect(reset(page)).toHaveCount(0);
                await expect(input(page)).toHaveValue('');
                await expect.poll(() => filter(page)).toMatchObject({ q: '', sort: 'created', sortOrder: '-' });
                await expect(page.locator('.result-controls .control-button').first()).toBeFocused();
                await expect(input(page)).not.toBeFocused();
                await page.keyboard.press('Escape');
                await expect(page.locator('.result-controls .control-button').first()).toBeFocused();
                await expect(panel(page)).toHaveCount(0);
                await expect(page.locator('.search-form')).not.toHaveClass(/\bexpanded\b/);
            });
        }
    });
}
