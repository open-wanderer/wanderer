import { expect, test, type Page } from '@playwright/test';
import { buildTrailSearchFixture, mountTrailSearchFixture } from './trail-search-fixture';
import { trailSearchFilters, type TrailSearchFilterSections } from '../../../src/lib/models/trail_search_settings';

const input = (page: Page) => page.getByRole('searchbox', { name: 'Search trails', exact: true });
const panel = (page: Page) => page.getByRole('region', { name: 'Filter by', exact: true });
const applications = (page: Page) => page.evaluate(() => (window as any).searchApplications);
const appliedFilter = (page: Page) => page.evaluate(() => (window as any).appliedSearchFilter);

async function emulateMacPointerFocus(page: Page) {
    await page.addInitScript(() => {
        window.addEventListener('mousedown', event => {
            const target = event.target instanceof Element ? event.target.closest('button, a[href]') : null;
            if (event.defaultPrevented || !target || target.hasAttribute('tabindex')) return;
            event.preventDefault();
            if (document.activeElement instanceof HTMLElement) document.activeElement.blur();
        });
    });
}

for (const dev of [true, false]) {
    test.describe(`trail search in ${dev ? 'development' : 'production'}`, () => {
        let bundle: string;
        let errors: string[];
        test.beforeAll(async () => { bundle = await buildTrailSearchFixture(dev); });
        test.beforeEach(({ page }) => {
            errors = [];
            page.on('pageerror', error => errors.push(error.message));
        });
        test.afterEach(() => { expect(errors).toEqual([]); });

        test('uses direct map places without radius filters and enables three trail links only on the map', async ({ page }) => {
            const fixture = await mountTrailSearchFixture(page, bundle, { map: true });
            await input(page).click();
            await page.locator('.general-filters .filter-chip').filter({ hasText: /^Place/ }).click();
            await page.locator('.place-input input').fill('Existing');
            await page.locator('.place-option').first().click();
            await expect(page.locator('.selected-place')).toBeVisible();
            await input(page).fill('Ridge');
            const trails = page.locator('[data-kind="trails"] .direct-option');
            await expect(trails).toHaveCount(3);
            expect(fixture.trailRequests[0]).toMatchObject({ q: 'Ridge', options: { limit: 3, attributesToSearchOn: ['name'], filter: 'fixture-visibility' } });
            await expect(trails.first()).toHaveAttribute('href', '/map/trail/@alice/direct-0');
            await trails.first().focus();
            await trails.first().press('ArrowRight');
            await expect(trails.nth(1)).toBeFocused();
            await page.locator('[data-kind="places"] .direct-option').first().click();
            await expect(input(page)).toHaveValue('');
            expect(await appliedFilter(page)).toMatchObject({ near: { radius: 2000 } });
            expect((await appliedFilter(page)).near).not.toHaveProperty('lat');
            expect(await page.evaluate(() => (window as any).placeSelections)).toEqual([expect.objectContaining({ name: 'Ridge', lat: 47.37, lon: 8.54 })]);
        });

        test('keeps place and list suggestions stable while only trail criteria change', async ({ page }) => {
            const fixture = await mountTrailSearchFixture(page, bundle, { map: true });
            await input(page).fill('Ridge');
            await expect(page.locator('[data-kind="trails"] .direct-option')).toHaveCount(3);
            await expect(page.locator('[data-kind="places"] .direct-option')).toHaveCount(3);
            const placeRequests = fixture.requests.length;
            const listRequests = fixture.listRequests.length;
            let release!: () => void;
            fixture.trailGates.set(2, new Promise<void>(resolve => release = resolve));
            await page.locator('.activity-grid').getByRole('button', { name: 'Hiking', exact: true }).click();
            await expect.poll(() => fixture.trailRequests.length).toBe(2);
            await expect(page.locator('[data-kind="trails"] .direct-option')).toHaveCount(3);
            await expect(page.locator('[data-kind="places"] .direct-option')).toHaveCount(3);
            expect(fixture.requests).toHaveLength(placeRequests);
            expect(fixture.listRequests).toHaveLength(listRequests);
            release();
            await expect(page.locator('[data-kind="trails"] .direct-option').first()).toContainText('Refined trail');
        });

        for (const map of [false, true]) {
            for (const keyboard of [false, true]) {
                test(`returns input focus only for keyboard place selection: map=${map}, keyboard=${keyboard}`, async ({ page }) => {
                    await mountTrailSearchFixture(page, bundle, { map });
                    await input(page).fill('Bern');
                    const direct = page.locator('[data-kind="places"] .direct-option').first();
                    await expect(direct).toBeVisible();
                    if (keyboard) { await direct.focus(); await direct.press('Enter'); }
                    else await direct.click();
                    if (keyboard) await expect(input(page)).toBeFocused();
                    else await expect(input(page)).not.toBeFocused();
                    await expect(input(page)).toHaveValue('');
                });
            }
        }

        test('retries an HTTP author lookup failure after the next search interaction', async ({ page }) => {
            const fixture = await mountTrailSearchFixture(page, bundle);
            fixture.authorFailures.add('anna-local');
            await page.evaluate(() => (window as any).restoreFixtureFilter({ author: 'anna-local' }));
            await expect.poll(() => fixture.authorLookups.length).toBe(1);
            await expect(page.locator('[data-filter-key="author"]')).toHaveText('Author');
            await input(page).fill('retry');
            await expect(page.locator('[data-filter-key="author"]')).toHaveText('@anna');
            expect(fixture.authorLookups).toEqual(['anna-local', 'anna-local']);
        });

        test('keeps keyboard focus after selecting a place suggestion or an editor result', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            await input(page).fill('Bern');
            const direct = page.locator('[data-kind="places"] .direct-option').first();
            await direct.focus();
            await direct.press('Enter');
            await expect(input(page)).toBeFocused();
            await page.getByRole('button', { name: 'Clear all', exact: true }).click();
            await input(page).click();
            await page.locator('.general-filters .filter-chip').filter({ hasText: /^Place/ }).click();
            await page.locator('.place-input input').fill('Bern');
            const option = page.locator('.place-option').first();
            await expect(option).toBeVisible();
            await option.focus();
            await option.press('Enter');
            const change = page.getByRole('button', { name: 'Change place', exact: true });
            await expect(change).toBeFocused();
            await change.press('Enter');
            await expect(page.locator('.place-input input')).toBeFocused();
        });

        for (const keyboard of [false, true]) {
            test(`removes an editor place without reopening the keyboard (keyboard=${keyboard})`, async ({ page }) => {
                await mountTrailSearchFixture(page, bundle);
                await input(page).click();
                await page.locator('.general-filters .filter-chip').filter({ hasText: /^Place/ }).click();
                await page.locator('.place-input input').fill('Bern');
                await page.locator('.place-option').first().click();
                const clear = page.getByRole('button', { name: 'Remove place filter', exact: true });
                if (keyboard) { await clear.focus(); await clear.press('Enter'); }
                else await clear.click();
                if (keyboard) await expect(page.locator('.place-input input')).toBeFocused();
                else await expect(page.locator('.place-input input')).not.toBeFocused();
                await expect(panel(page)).toBeVisible();
                expect((await appliedFilter(page)).near).not.toHaveProperty('lat');
            });
        }

        test('does not retry unavailable actor lookups for anonymous visitors', async ({ page }) => {
            const fixture = await mountTrailSearchFixture(page, bundle, { anonymous: true, author: 'anna-local' });
            await input(page).click();
            await input(page).fill('anonymous');
            await expect.poll(() => appliedFilter(page)).toMatchObject({ q: 'anonymous' });
            await input(page).press('Escape');
            await input(page).click();
            await input(page).fill('second');
            await expect.poll(() => appliedFilter(page)).toMatchObject({ q: 'second' });
            expect(fixture.authorLookups).toEqual([]);
        });

        test.describe('macOS pointer focus model', () => {
            test.beforeEach(async ({ page }) => { await emulateMacPointerFocus(page); });

            for (const layout of ['sidebar', 'overlay', 'push'] as const) {
                test(`selects a place with the mouse in ${layout}`, async ({ page }) => {
                    await mountTrailSearchFixture(page, bundle, { layout });
                    await input(page).fill('St. Gallen');
                    const option = page.locator('[data-kind="places"] .direct-option').nth(1);
                    await expect(option).toBeVisible();
                    await expect(input(page)).toBeFocused();
                    await option.click();
                    await expect.poll(() => appliedFilter(page)).toMatchObject({ near: { name: 'St. Gallen', lat: 48.37, lon: 9.54 } });
                    if (layout === 'sidebar') {
                        await expect(input(page)).toBeFocused();
                        await expect(panel(page)).toBeVisible();
                    } else {
                        await expect(input(page)).not.toBeFocused();
                        await expect(panel(page)).toBeHidden();
                    }
                });

                test(`follows a list suggestion with the mouse in ${layout}`, async ({ page }) => {
                    const fixture = await mountTrailSearchFixture(page, bundle, { layout });
                    fixture.listHits.set('Bern', [{ id: 'list-one', name: 'Bern trails', author_name: 'anna' }]);
                    await input(page).fill('Bern');
                    const link = page.locator('[data-kind="lists"] .direct-option');
                    await expect(link).toBeVisible();
                    await expect(input(page)).toBeFocused();
                    await link.click();
                    await expect(page).toHaveURL('http://trail-search.test/lists/@anna/list-one');
                });

                for (const kind of ['author', 'tag'] as const) {
                    test(`captures the ${kind} shortcut with the mouse in ${layout}`, async ({ page }) => {
                        await mountTrailSearchFixture(page, bundle, { layout });
                        await input(page).fill(kind === 'author' ? 'Grat @anna' : 'Grat #alpin');
                        const option = page.locator('.shortcut-option').first();
                        await expect(option).toBeVisible();
                        await option.click();
                        await expect.poll(() => appliedFilter(page)).toMatchObject(kind === 'author' ? { author: 'anna-local' } : { tags: ['alpin'] });
                        await expect(input(page)).toHaveValue('Grat');
                        await expect(input(page)).toBeFocused();
                        await expect(page.locator('.captured-filter')).toHaveCount(1);
                    });

                    test(`removes a captured ${kind} shortcut with the mouse in ${layout}`, async ({ page }) => {
                        await mountTrailSearchFixture(page, bundle, { layout });
                        await input(page).fill(kind === 'author' ? '@anna' : '#alpin');
                        const option = page.locator('.shortcut-option').first();
                        await expect(option).toBeVisible();
                        await option.focus();
                        await option.press('Enter');
                        const chip = page.locator('.captured-filter');
                        await expect(chip).toBeVisible();
                        await expect(input(page)).toBeFocused();
                        await chip.click();
                        await expect.poll(() => appliedFilter(page)).toMatchObject(kind === 'author' ? { author: '' } : { tags: [] });
                        await expect(chip).toHaveCount(0);
                        if (layout === 'sidebar') await expect(input(page)).toBeFocused();
                        else await expect(input(page)).not.toBeFocused();
                    });
                }
            }
        });

        test('has no footer and clears text and filters from the collapsed input', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            const reset = page.getByRole('button', { name: 'Clear all', exact: true });
            await expect(reset).toBeHidden();
            await input(page).click();
            await expect(reset).toBeHidden();
            await input(page).fill('Grat');
            await expect(reset).toBeVisible();
            await page.locator('.activity-grid').getByRole('button', { name: 'Hiking', exact: true }).click();
            await input(page).press('Escape');
            await expect(panel(page)).toBeHidden();
            await expect(reset).toBeEnabled();
            await reset.click();
            await expect(reset).toBeHidden();
            await expect(page.getByRole('button', { name: 'Filter by', exact: true })).toBeFocused();
            await expect(input(page)).toHaveValue('');
            await expect(panel(page)).toBeHidden();
            expect(await appliedFilter(page)).toMatchObject({ q: '', category: [], sort: 'created', sortOrder: '+' });
            await input(page).click();
            await expect(page.locator('.search-footer, .result-total')).toHaveCount(0);
            const more = page.locator('.general-filters .filter-chips .advanced-toggle');
            await expect(more).toHaveText('More filters');
            await more.click();
            await expect(more).toHaveText('Fewer filters');
            await expect(page.locator('.advanced-filters')).toBeVisible();
            await more.click();
            await expect(page.locator('.advanced-filters')).toHaveCount(0);
        });

        for (const activation of ['pointer', 'keyboard'] as const) {
            for (const layout of ['overlay', 'sidebar'] as const) {
                const behavior = layout === 'sidebar' && activation === 'pointer' ? 'keeps the input unfocused' : 'restores reset focus';
                test(`${behavior} in the open ${layout} search after ${activation} activation`, async ({ page }) => {
                    await mountTrailSearchFixture(page, bundle, { layout });
                    await input(page).fill('Grat');
                    await page.locator('.activity-grid').getByRole('button', { name: 'Hiking', exact: true }).click();
                    const reset = page.getByRole('button', { name: 'Clear all', exact: true });
                    if (activation === 'keyboard') { await reset.focus(); await reset.press('Enter'); }
                    else await reset.click();
                    await expect(reset).toBeHidden();
                    await expect(input(page)).toHaveValue('');
                    await expect(panel(page)).toBeVisible();
                    expect(await appliedFilter(page)).toMatchObject({ q: '', category: [] });
                    if (layout === 'sidebar') {
                        if (activation === 'keyboard') await expect(input(page)).toBeFocused();
                        else await expect(input(page)).not.toBeFocused();
                        await expect(page.locator('.search-heading button[aria-expanded]')).toHaveCount(0);
                    } else {
                        await expect(page.getByRole('button', { name: 'Close search', exact: true })).toBeFocused();
                        await page.keyboard.press('Escape');
                        await expect(panel(page)).toBeHidden();
                    }
                });
            }
        }

        test('restores the filter button after clearing a collapsed search with the keyboard', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            await input(page).fill('Grat');
            await input(page).press('Escape');
            const reset = page.getByRole('button', { name: 'Clear all', exact: true });
            await reset.focus();
            await reset.press('Enter');
            await expect(reset).toBeHidden();
            await expect(panel(page)).toBeHidden();
            await expect(page.getByRole('button', { name: 'Filter by', exact: true })).toBeFocused();
            expect(await appliedFilter(page)).toMatchObject({ q: '', category: [] });
        });

        for (const layout of ['overlay', 'push'] as const) {
            for (const activation of ['pointer', 'keyboard'] as const) {
                test(`focuses the input only for keyboard filter toggles in ${layout}: ${activation}`, async ({ page }) => {
                    await mountTrailSearchFixture(page, bundle, { layout });
                    const toggle = page.locator('.search-heading button[aria-expanded]');
                    if (activation === 'keyboard') { await toggle.focus(); await toggle.press('Enter'); }
                    else await toggle.click();
                    await expect(panel(page)).toBeVisible();
                    if (activation === 'keyboard') await expect(input(page)).toBeFocused();
                    else await expect(toggle).toBeFocused();
                    if (activation === 'keyboard') { await toggle.focus(); await toggle.press('Enter'); }
                    else await toggle.click();
                    await expect(panel(page)).toBeHidden();
                    if (activation === 'keyboard') await expect(input(page)).toBeFocused();
                    else await expect(toggle).toBeFocused();
                });
            }
        }

        test.describe('touch filter controls', () => {
            test.use({ hasTouch: true, viewport: { width: 420, height: 800 } });

            for (const layout of ['overlay', 'push'] as const) {
                test(`keeps focus on the filter toggle after opening and closing by tap in ${layout}`, async ({ page }) => {
                    await mountTrailSearchFixture(page, bundle, { layout });
                    await expect.poll(() => page.evaluate(() => innerWidth)).toBe(420);
                    const toggle = page.locator('.search-heading button[aria-expanded]');
                    await toggle.evaluate(element => {
                        (window as any).filterToggleDetails = [];
                        element.addEventListener('click', event => (window as any).filterToggleDetails.push((event as MouseEvent).detail));
                    });
                    await toggle.tap();
                    await expect(panel(page)).toBeVisible();
                    await expect(toggle).toBeFocused();
                    await expect(input(page)).not.toBeFocused();
                    await panel(page).locator('.filter-caption').first().tap();
                    await toggle.tap();
                    await expect(panel(page)).toBeHidden();
                    await expect(toggle).toBeFocused();
                    await expect(input(page)).not.toBeFocused();
                    expect(await page.evaluate(() => (window as any).filterToggleDetails)).toEqual([1, 1]);
                });
            }

            for (const activation of ['tap', 'keyboard'] as const) {
                test(`focuses the input only for keyboard removal of a captured shortcut: ${activation}`, async ({ page }) => {
                    await mountTrailSearchFixture(page, bundle);
                    await expect.poll(() => page.evaluate(() => innerWidth)).toBe(420);
                    await input(page).fill('#alpin');
                    const option = page.locator('.shortcut-option').first();
                    await expect(option).toBeVisible();
                    await option.tap();
                    await expect(page.locator('.captured-filter')).toHaveText('#alpin');
                    await expect(input(page)).toBeFocused();
                    await page.locator('.activity-grid').getByRole('button', { name: 'Hiking', exact: true }).tap();
                    await expect(input(page)).not.toBeFocused();
                    const remove = page.getByRole('button', { name: 'Remove: #alpin', exact: true });
                    if (activation === 'keyboard') { await remove.focus(); await remove.press('Enter'); }
                    else await remove.tap();
                    await expect(remove).toHaveCount(0);
                    await expect.poll(() => appliedFilter(page)).toMatchObject({ tags: [], category: ['hiking'] });
                    if (activation === 'keyboard') await expect(input(page)).toBeFocused();
                    else await expect(input(page)).not.toBeFocused();
                    await expect(panel(page)).toBeVisible();
                });
            }

            test('clears sidebar criteria by tap without focusing the input', async ({ page }) => {
                await page.setViewportSize({ width: 1200, height: 800 });
                await mountTrailSearchFixture(page, bundle, { layout: 'sidebar' });
                await expect.poll(() => page.evaluate(() => innerWidth)).toBe(1200);
                await input(page).fill('Grat');
                await page.locator('.activity-grid').getByRole('button', { name: 'Hiking', exact: true }).tap();
                const reset = page.getByRole('button', { name: 'Clear all', exact: true });
                await reset.tap();
                await expect(reset).toBeHidden();
                await expect(input(page)).not.toBeFocused();
                await expect(input(page)).toHaveValue('');
                await expect(panel(page)).toBeVisible();
                expect(await appliedFilter(page)).toMatchObject({ q: '', category: [] });
            });
        });

        for (const layout of ['overlay', 'push'] as const) {
            test(layout + ' uses the appropriate positioning and keeps the close icon reachable', async ({ page }) => {
                await page.setViewportSize({ width: 390, height: 600 });
                await mountTrailSearchFixture(page, bundle, { layout });
                await input(page).click();
                const bounds = await panel(page).boundingBox();
                const main = await page.locator('main').boundingBox();
                if (layout === 'overlay') expect(main!.y + main!.height).toBeLessThan(bounds!.y + bounds!.height);
                else expect(main!.y + main!.height).toBeGreaterThanOrEqual(bounds!.y + bounds!.height);
                await expect(page.getByRole('button', { name: 'Close search', exact: true })).toBeInViewport();
                const body = await page.locator('.filter-body').evaluate(element => ({ client: element.clientWidth, scroll: element.scrollWidth }));
                expect(body.scroll).toBeLessThanOrEqual(body.client);
            });
        }

        for (const side of ['left', 'right']) {
            test(`sidebar ${side} Escape dismisses suggestions then clears text and retains filter focus`, async ({ page }) => {
                await mountTrailSearchFixture(page, bundle, { layout: 'sidebar' });
                await page.locator('main').evaluate((element, side) => { element.style.maxWidth = '1000px'; element.style.marginLeft = side === 'right' ? 'auto' : '0'; element.style.marginRight = side === 'left' ? 'auto' : '0'; }, side);
                await input(page).fill('Bern');
                await expect(page.locator('.direct-option').first()).toBeVisible();
                await input(page).press('Escape');
                await expect(page.locator('.direct-option').first()).toBeHidden();
                await expect(input(page)).toHaveValue('Bern');
                await expect(input(page)).toBeFocused();
                await input(page).press('Escape');
                await expect(input(page)).toHaveValue('');
                await expect.poll(() => appliedFilter(page)).toMatchObject({ q: '' });
                await expect(panel(page)).toBeVisible();
                await expect(input(page)).toBeFocused();
                await input(page).press('Escape');
                const place = page.getByRole('searchbox', { name: 'Place', exact: true });
                await place.fill('Zürich');
                await expect(page.locator('.place-option').first()).toBeVisible();
                await place.press('Escape');
                await expect(page.locator('.place-option')).toHaveCount(0);
                await expect(place).toHaveValue('Zürich');
                await expect(place).toBeFocused();
                await place.press('Escape');
                await expect(place).toHaveValue('');
                await place.fill('Bern');
                await expect(page.locator('.place-option').first()).toBeVisible();
                await page.getByRole('button', { name: 'More filters', exact: true }).click();
                const author = page.locator('.author-control input');
                await author.fill('anna');
                await expect(page.locator('.author-control ul')).toBeVisible();
                await author.press('Escape');
                await expect(page.locator('.author-control ul')).toHaveCount(0);
                await expect(author).toHaveValue('anna');
                await expect(author).toBeFocused();
                await author.press('Escape');
                await expect(author).toHaveValue('');
                await expect(panel(page)).toBeVisible();
            });
        }

        test('the sidebar stays open on Escape and outside clicks while edits still apply', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle, { layout: 'sidebar' });
            await expect(panel(page)).toBeVisible();
            await expect(page.getByRole('button', { name: 'Close search', exact: true })).toHaveCount(0);
            await expect(page.locator('.general-filters .sidebar-section')).toHaveCount(5);
            await expect(page.locator('.activity-grid')).toBeVisible();
            await expect(page.getByRole('searchbox', { name: 'Place', exact: true })).toBeVisible();
            await expect(page.getByRole('slider', { name: 'Distance · From', exact: true })).toBeVisible();
            await expect(page.locator('.difficulty-editor')).toBeVisible();
            await expect(page.locator('.advanced-filters')).toHaveCount(0);
            const more = page.getByRole('button', { name: 'More filters', exact: true });
            await more.click();
            await expect(more).toHaveAttribute('aria-expanded', 'true');
            await expect(page.locator('.advanced-filters .sidebar-section')).toHaveCount(7);
            await expect(page.getByLabel('After', { exact: true })).toBeVisible();
            await expect(page.locator('.author-control input')).toBeVisible();
            await page.evaluate(() => window.scrollTo(0, 0));
            await page.locator('.activity-grid').hover();
            await page.mouse.wheel(0, 300);
            await expect.poll(() => page.evaluate(() => window.scrollY)).toBeGreaterThan(0);
            expect(await page.locator('.filter-body').evaluate(element => element.scrollTop)).toBe(0);
            await more.click();
            await expect(more).toHaveAttribute('aria-expanded', 'false');
            await expect(page.locator('.advanced-filters')).toHaveCount(0);
            await input(page).fill('X');
            await expect.poll(() => appliedFilter(page)).toMatchObject({ q: 'X' });
            await input(page).press('Escape');
            await expect(panel(page)).toBeVisible();
            await page.getByTestId('outside').click();
            await expect(panel(page)).toBeVisible();
            expect(await appliedFilter(page)).toMatchObject({ q: '' });
        });

        test('restores criteria into the open sidebar without publishing its previous draft', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle, { layout: 'sidebar' });
            await page.clock.install();
            await input(page).fill('Pending text');
            await page.evaluate(() => (window as any).restoreFixtureFilter({ q: 'Restored text', category: ['walking'] }));
            await expect(input(page)).toHaveValue('Restored text');
            await expect(page.locator('.activity-grid').getByRole('button', { name: 'Walking', exact: true })).toHaveAttribute('aria-pressed', 'true');
            await page.clock.runFor(400);
            expect(await appliedFilter(page)).toMatchObject({ q: 'Restored text', category: ['walking'] });
            expect(await applications(page)).toEqual([]);
        });

        test('retains external sidebar sorting on outside clicks without publishing another search', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle, { layout: 'sidebar' });
            await expect(panel(page)).toBeVisible();
            await page.evaluate(() => (window as any).setFixtureSort('name', '-'));
            await expect.poll(() => appliedFilter(page)).toMatchObject({ sort: 'name', sortOrder: '-' });
            await page.getByTestId('outside').click();
            expect(await appliedFilter(page)).toMatchObject({ sort: 'name', sortOrder: '-' });
            expect(await applications(page)).toEqual([]);
            await expect(panel(page)).toBeVisible();
        });

        test('retains external sidebar sorting through pending live edits and reset', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle, { layout: 'sidebar' });
            await page.locator('.activity-grid').getByRole('button', { name: 'Hiking', exact: true }).click();
            await page.evaluate(() => (window as any).setFixtureSort('name', '-'));
            await expect.poll(() => appliedFilter(page)).toMatchObject({ category: ['hiking'], sort: 'name', sortOrder: '-' });
            expect((await applications(page)).at(-1).filter).toMatchObject({ category: ['hiking'], sort: 'name', sortOrder: '-' });

            await page.evaluate(() => (window as any).setFixtureSort('distance', '-'));
            await input(page).fill('Pending input');
            await page.getByRole('button', { name: 'Clear all', exact: true }).click();
            await expect.poll(() => appliedFilter(page)).toMatchObject({ q: '', category: [], sort: 'distance', sortOrder: '-' });
            expect((await applications(page)).at(-1).filter).toMatchObject({ q: '', category: [], sort: 'distance', sortOrder: '-' });
            await expect(panel(page)).toBeVisible();
        });

        test('retains external sidebar sorting when removing an applied filter chip', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle, { layout: 'sidebar' });
            await page.locator('.activity-grid').getByRole('button', { name: 'Hiking', exact: true }).click();
            const chip = page.locator('[data-filter-key="category:hiking"]');
            await expect(chip).toBeVisible();
            await page.evaluate(() => (window as any).setFixtureSort('name', '-'));
            await chip.click();
            expect(await appliedFilter(page)).toMatchObject({ category: [], sort: 'name', sortOrder: '-' });
            expect((await applications(page)).at(-1).filter).toMatchObject({ category: [], sort: 'name', sortOrder: '-' });
            await expect(panel(page)).toBeVisible();
        });

        test('updates text after a pause, keeps editors open and does not reapply on close', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            await input(page).fill('G');
            await input(page).fill('Gr');
            await input(page).fill('Grat');
            expect(await applications(page)).toEqual([]);
            await expect.poll(() => appliedFilter(page)).toMatchObject({ q: 'Grat' });
            expect(await applications(page)).toHaveLength(1);
            await expect(panel(page)).toBeVisible();
            await expect(input(page)).toBeFocused();
            await page.getByRole('button', { name: 'Close search', exact: true }).click();
            await expect(panel(page)).toBeHidden();
            expect(await applications(page)).toHaveLength(1);
        });

        test('applies consecutive filter edits without closing the mask or losing the current editor', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            await input(page).click();
            await page.locator('.activity-grid').getByRole('button', { name: 'Hiking', exact: true }).click();
            await expect.poll(() => appliedFilter(page)).toMatchObject({ category: ['hiking'] });
            await page.getByRole('button', { name: 'Alpine hiking', exact: true }).click();
            await expect.poll(() => appliedFilter(page)).toMatchObject({ subcategory: ['alpine'] });
            await page.getByRole('button', { name: 'Distance', exact: true }).click();
            const slider = page.getByRole('slider', { name: 'Distance · From', exact: true });
            await slider.press('ArrowRight');
            const minimum = Number(await slider.getAttribute('aria-valuenow'));
            await expect.poll(() => appliedFilter(page)).toMatchObject({ distanceMin: minimum, category: ['hiking'], subcategory: ['alpine'] });
            await expect(slider).toBeFocused();
            await expect(panel(page)).toBeVisible();
        });

        test('removing an applied chip retains text whose debounce is still pending', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            await input(page).click();
            await page.locator('.activity-grid').getByRole('button', { name: 'Hiking', exact: true }).click();
            await expect(page.locator('[data-filter-key="category:hiking"]')).toHaveCount(1);
            await input(page).fill('New search text');
            await page.locator('[data-filter-key="category:hiking"]').click();
            await expect(input(page)).toHaveValue('New search text');
            expect(await appliedFilter(page)).toMatchObject({ q: 'New search text', category: [] });
            await expect(panel(page)).toBeVisible();
        });

        for (const action of ['Escape', 'outside', 'Close']) {
            test('flushes pending text when closing via ' + action, async ({ page }) => {
                await mountTrailSearchFixture(page, bundle);
                await input(page).fill('Final input');
                if (action === 'outside') await page.getByTestId('outside').click();
                else if (action === 'Close') await page.getByRole('button', { name: 'Close search', exact: true }).click();
                else await input(page).press('Escape');
                await expect(panel(page)).toBeHidden();
                expect(await appliedFilter(page)).toMatchObject({ q: 'Final input' });
                await input(page).click();
                await expect(input(page)).toHaveValue('Final input');
            });
        }

        test('does not submit incomplete IME composition', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            await input(page).click();
            await input(page).dispatchEvent('compositionstart');
            await input(page).fill('途中');
            await input(page).dispatchEvent('keydown', { key: 'Escape', isComposing: true });
            await page.waitForTimeout(450);
            await expect(panel(page)).toBeVisible();
            expect(await applications(page)).toEqual([]);
            await input(page).dispatchEvent('compositionend');
            await expect.poll(() => appliedFilter(page)).toMatchObject({ q: '途中' });
        });

        test('captures an author only after selection and retains ordinary search text', async ({ page }) => {
            const fixture = await mountTrailSearchFixture(page, bundle);
            await input(page).fill('Grat @ann');
            const choices = page.locator('.shortcut-option');
            await expect(choices).toHaveCount(2);
            await choices.first().click();
            await expect(input(page)).toHaveValue('Grat');
            await expect(page.locator('.captured-filter')).toContainText('@anna');
            await expect(panel(page)).toBeVisible();
            await page.getByRole('button', { name: 'Close search', exact: true }).click();
            await expect.poll(() => applications(page)).not.toHaveLength(0);
            expect((await applications(page)).at(-1).filter).toMatchObject({ q: 'Grat', author: 'anna-local' });
            await expect(page.locator('.applied-filters')).toContainText('@anna');
            expect(fixture.authorRequests).toEqual(['ann']);
        });

        test('selects difficulties without closing the standard dropdown', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            await input(page).click();
            await page.locator('.filter-chip').filter({ hasText: 'Difficulty' }).click();
            const dropdown = page.locator('.difficulty-editor');
            const trigger = dropdown.getByRole('button', { name: 'Difficulty', exact: true });
            await trigger.focus();
            await trigger.press('Space');
            await expect(trigger).toHaveAttribute('aria-expanded', 'true');
            await dropdown.getByRole('button', { name: 'Easy', exact: true }).click();
            await dropdown.getByRole('button', { name: 'Moderate', exact: true }).click();
            await expect(dropdown.getByRole('button', { name: 'Easy', exact: true })).toHaveAttribute('aria-pressed', 'true');
            await expect(panel(page)).toBeVisible();
            await page.getByRole('button', { name: 'Close search', exact: true }).click();
            expect((await applications(page)).at(-1).filter.difficulty).toEqual([0, 1]);
            await page.locator('.applied-chip').click();
            expect((await applications(page)).at(-1).filter.difficulty).toEqual([0, 1, 2]);
            await input(page).click();
            await page.locator('.filter-chip').filter({ hasText: 'Difficulty' }).click();
            await expect(dropdown.getByText('Filter difficulty...', { exact: true })).toBeVisible();
        });

        test('finds and applies an author by plain name in the author editor', async ({ page }) => {
            const fixture = await mountTrailSearchFixture(page, bundle);
            await input(page).click();
            await page.getByRole('button', { name: 'More filters', exact: true }).click();
            await page.locator('.advanced-filters .filter-chip').filter({ hasText: 'Author' }).click();
            const author = page.getByRole('searchbox', { name: 'Author', exact: true });
            await author.fill('anna');
            const choices = page.locator('.advanced-filters .menu-item');
            await expect(choices).toHaveCount(2);
            await choices.first().click();
            await expect(page.locator('.author-chip')).toHaveText('@anna');
            expect(fixture.authorRequests).toEqual(['anna']);
            await expect(input(page)).toHaveValue('');
            await expect(panel(page)).toBeVisible();
            await page.getByRole('button', { name: 'Close search', exact: true }).click();
            expect((await applications(page)).at(-1).filter).toMatchObject({ author: 'anna-local', q: '' });
            await expect(page.locator('[data-filter-key="author"]')).toHaveText('@anna');
            expect(fixture.authorLookups).toEqual([]);
        });

        for (const [author, handle] of [['anna-local', '@anna'], ['anna-remote', '@anna@example.org']]) {
            test(`resolves the handle for a restored author filter: ${author}`, async ({ page }) => {
                const fixture = await mountTrailSearchFixture(page, bundle, { author });
                await expect(page.locator('[data-filter-key="author"]')).toHaveText(handle);
                await input(page).click();
                await page.getByRole('button', { name: /^More filters/ }).click();
                await page.locator('.advanced-filters .filter-chip').filter({ hasText: 'Author' }).click();
                await expect(page.locator('.author-chip')).toHaveText(handle);
                expect(fixture.authorLookups).toEqual([author]);
                await page.locator('.author-chip').click();
                await page.getByRole('button', { name: 'Close search', exact: true }).click();
                await expect(page.locator('[data-filter-key="author"]')).toHaveCount(0);
            });
        }

        test('keeps an unavailable author removable without showing its raw ID', async ({ page }) => {
            const fixture = await mountTrailSearchFixture(page, bundle, { author: 'unavailable-actor' });
            const chip = page.locator('[data-filter-key="author"]');
            await expect(chip).toHaveText('Author');
            await expect.poll(() => fixture.authorLookups.length).toBe(1);
            await input(page).fill('unrelated text');
            await expect.poll(() => applications(page).then(value => value.length)).toBeGreaterThan(0);
            await input(page).press('Escape');
            await input(page).click();
            expect(fixture.authorLookups).toEqual(['unavailable-actor']);
            await input(page).press('Escape');
            await chip.click();
            expect((await applications(page)).at(-1).filter.author).toBe('');
        });

        test('captures several existing tags and removes them individually', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            await input(page).fill('Grat #alp');
            const choices = page.locator('.shortcut-option');
            await expect(choices).toHaveCount(2);
            await choices.first().click();
            await input(page).fill('Grat #alpine');
            await expect(choices).toHaveCount(1);
            await choices.first().click();
            await expect(page.locator('.captured-filter')).toHaveCount(2);
            await page.getByRole('button', { name: 'Remove: #alpin', exact: true }).click();
            await expect(page.locator('.captured-filter')).toHaveCount(1);
            await page.getByRole('button', { name: 'Close search', exact: true }).click();
            await expect.poll(() => applications(page)).not.toHaveLength(0);
            expect((await applications(page)).at(-1).filter).toMatchObject({ q: 'Grat', tags: ['alpine hiking'] });
        });

        test('offers three place suggestions in a row and looks up list names separately', async ({ page }) => {
            await page.setViewportSize({ width: 1440, height: 1000 });
            const fixture = await mountTrailSearchFixture(page, bundle);
            await input(page).fill('St. Gallen');
            const choices = page.locator('.direct-option');
            await expect(choices).toHaveCount(3);
            await expect(page.locator('[data-kind="trails"], [data-kind="lists"]')).toHaveCount(0);
            const positions = await choices.evaluateAll(elements => elements.map(element => ({ x: element.getBoundingClientRect().x, y: element.getBoundingClientRect().y })));
            expect(positions[1].x).toBeGreaterThan(positions[0].x);
            expect(positions[2].x).toBeGreaterThan(positions[1].x);
            expect(positions[0].y).toBe(positions[2].y);
            expect(fixture.multiRequests).toEqual([]);
            expect(fixture.requests).toEqual(['St. Gallen']);
            await expect.poll(() => fixture.listRequests).toMatchObject([{ q: 'St. Gallen', options: { limit: 3, attributesToSearchOn: ['name'] } }]);
            await expect(panel(page)).toBeVisible();
        });

        test('offers at most three direct list links and keeps both suggestion groups horizontal', async ({ page }) => {
            await page.setViewportSize({ width: 1440, height: 1000 });
            const fixture = await mountTrailSearchFixture(page, bundle);
            fixture.listHits.set('Bern', [
                { id: 'list-one', name: 'Bern Highlights', author_name: 'anna' },
                { id: 'list-two', name: 'Bern Weekend', author_name: 'anna', domain: 'example.org' },
                { id: 'list-three', name: 'Bern Lakes', author_name: 'bea' },
                { id: 'list-four', name: 'Bern Extra', author_name: 'bea' },
            ]);
            await input(page).fill('Bern');
            const links = page.locator('[data-kind="lists"] .direct-option');
            await expect(links).toHaveCount(3);
            await expect(page.locator('[data-kind="places"] .direct-option')).toHaveCount(3);
            await expect(links.first()).toHaveAttribute('href', '/lists/@anna/list-one');
            await expect(links.nth(1)).toHaveAttribute('href', '/lists/@anna@example.org/list-two');
            const rows = await links.evaluateAll(elements => elements.map(element => element.getBoundingClientRect().y));
            expect(new Set(rows).size).toBe(1);
            await expect(panel(page)).toBeVisible();
            await expect(input(page)).toHaveValue('Bern');
            await links.first().click();
            await expect(page).toHaveURL('http://trail-search.test/lists/@anna/list-one');
        });

        test('list links remain usable while geocoding is pending and failed lists leave places usable', async ({ page }) => {
            const fixture = await mountTrailSearchFixture(page, bundle);
            let release!: () => void;
            fixture.gates.set('Bern', new Promise<void>(resolve => { release = resolve; }));
            fixture.listHits.set('Bern', [{ id: 'list-one', name: 'Bern Highlights', author_name: 'anna' }]);
            try {
                await input(page).fill('Bern');
                await expect(page.locator('[data-kind="lists"] a')).toHaveCount(1);
                await expect(page.locator('[data-kind="places"]')).toHaveCount(0);
                await input(page).press('Enter');
                await expect.poll(() => applications(page)).not.toHaveLength(0);
                expect((await applications(page)).at(-1).filter.q).toBe('Bern');
                release();
                fixture.listFailures.add('Zurich');
                await input(page).fill('Zurich');
                await expect.poll(() => fixture.listCompleted.includes('Zurich')).toBe(true);
                await expect(page.locator('[data-kind="places"] .direct-option')).toHaveCount(3);
                await expect(page.locator('[data-kind="lists"]')).toHaveCount(0);
                await expect(page.getByText('No results found', { exact: true })).toHaveCount(0);
            } finally { release(); }
        });

        test('late list responses cannot replace a newer query or author capture', async ({ page }) => {
            const fixture = await mountTrailSearchFixture(page, bundle);
            let release!: () => void;
            fixture.listGates.set('Bern', new Promise<void>(resolve => { release = resolve; }));
            fixture.listHits.set('Bern', [{ id: 'old-list', name: 'Bern Old', author_name: 'anna' }]);
            fixture.listHits.set('Zurich', [{ id: 'new-list', name: 'Zurich New', author_name: 'anna' }]);
            try {
                await input(page).fill('Bern');
                await expect.poll(() => fixture.listRequests.some(request => request.q === 'Bern')).toBe(true);
                await input(page).fill('Zurich');
                await expect(page.locator('[data-kind="lists"]')).toContainText('Zurich New');
                release();
                await expect.poll(() => fixture.listCompleted.includes('Bern')).toBe(true);
                await expect(page.locator('[data-kind="lists"]')).not.toContainText('Bern Old');
                const count = fixture.listRequests.length;
                await input(page).fill('@anna');
                await expect(page.locator('.shortcut-option')).toHaveCount(2);
                await expect(page.locator('[data-kind="lists"]')).toHaveCount(0);
                expect(fixture.listRequests).toHaveLength(count);
            } finally { release(); }
        });

        test('direct list links fit narrow windows and support keyboard focus and Escape', async ({ page }) => {
            await page.setViewportSize({ width: 320, height: 560 });
            const fixture = await mountTrailSearchFixture(page, bundle);
            fixture.empty.add('Bern');
            fixture.listHits.set('Bern', [1, 2, 3].map(index => ({ id: 'list-' + index, name: 'Bern with a very long list title ' + index, author_name: 'anna' })));
            await input(page).fill('Bern');
            const links = page.locator('[data-kind="lists"] .direct-option');
            await expect(links).toHaveCount(3);
            await expect(page.locator('[data-kind="places"]')).toHaveCount(0);
            const width = await page.locator('.filter-body').evaluate(element => ({ client: element.clientWidth, scroll: element.scrollWidth }));
            expect(width.scroll).toBeLessThanOrEqual(width.client);
            await expect(page.getByRole('button', { name: 'Close search', exact: true })).toBeInViewport();
            await input(page).press('ArrowUp');
            await expect(links.last()).toBeFocused();
            await links.last().press('ArrowLeft');
            await expect(links.nth(1)).toBeFocused();
            await page.keyboard.press('Escape');
            await expect(panel(page)).toBeHidden();
            expect(await appliedFilter(page)).toMatchObject({ q: 'Bern' });
        });

        test('offers author and tag choices next to each other and supports horizontal arrow keys', async ({ page }) => {
            await page.setViewportSize({ width: 1440, height: 1000 });
            await mountTrailSearchFixture(page, bundle);
            for (const query of ['@anna', '#']) {
                await input(page).fill(query);
                const choices = page.locator('.shortcut-option');
                await expect(choices).toHaveCount(query === '#' ? 3 : 2);
                const positions = await choices.evaluateAll(elements => elements.map(element => ({ x: element.getBoundingClientRect().x, y: element.getBoundingClientRect().y })));
                expect(positions[1].x).toBeGreaterThan(positions[0].x);
                expect(positions[0].y).toBe(positions.at(-1)!.y);
                await input(page).press('ArrowDown');
                await expect(choices.first()).toBeFocused();
                await choices.first().press('ArrowRight');
                await expect(choices.nth(1)).toBeFocused();
                await choices.nth(1).press('ArrowLeft');
                await expect(choices.first()).toBeFocused();
            }
        });

        for (const layout of ['overlay', 'push', 'sidebar'] as const) {
            test(`uses a direct place suggestion as an explicit filter in ${layout} layout`, async ({ page }) => {
                await mountTrailSearchFixture(page, bundle, { layout });
                await input(page).click();
                await page.locator('.activity-grid').getByRole('button', { name: 'Hiking', exact: true }).click();
                await page.evaluate(() => (window as any).setFixtureSort('name', '-'));
                await input(page).fill('St. Gallen');
                await page.locator('[data-kind="places"] .direct-option').nth(1).click();
                expect(await appliedFilter(page)).toMatchObject({
                    q: '', category: ['hiking'], sort: 'name', sortOrder: '-',
                    near: { name: 'St. Gallen', lat: 48.37, lon: 9.54, radius: 2000 },
                });
                expect((await applications(page)).at(-1).resetBounds).toBe(true);
                if (layout === 'sidebar') await expect(panel(page)).toBeVisible();
                else await expect(panel(page)).toBeHidden();
                await expect(page.locator('[data-filter-key="location"]')).toBeVisible();
                await page.locator('[data-filter-key="location"]').click();
                expect((await appliedFilter(page)).near).toEqual({ radius: 2000 });
            });
        }

        test('keeps text search usable while optional place lookup is pending or fails', async ({ page }) => {
            const fixture = await mountTrailSearchFixture(page, bundle);
            let release!: () => void;
            fixture.gates.set('Bern', new Promise<void>(resolve => { release = resolve; }));
            try {
                await input(page).fill('Bern');
                await expect.poll(() => fixture.requests.includes('Bern')).toBe(true);
                await expect(page.locator('.direct-suggestions')).toHaveCount(0);
                await input(page).press('Enter');
                await expect.poll(() => applications(page)).not.toHaveLength(0);
                expect((await applications(page)).at(-1).filter.q).toBe('Bern');
                release();
                fixture.failures.add('Zurich');
                await input(page).fill('Zurich');
                await expect.poll(() => fixture.completed.includes('Zurich')).toBe(true);
                await expect(page.locator('.direct-suggestions')).toHaveCount(0);
                await expect(page.getByText('No results found', { exact: true })).toHaveCount(0);
                await input(page).press('Enter');
                await expect.poll(() => appliedFilter(page)).toMatchObject({ q: 'Zurich' });
                expect((await applications(page)).at(-1).filter.q).toBe('Zurich');
            } finally { release(); }
        });

        test('hides empty place and author suggestions without a second empty-state message', async ({ page }) => {
            const fixture = await mountTrailSearchFixture(page, bundle);
            fixture.empty.add('NoPlace');
            await input(page).fill('NoPlace');
            await expect.poll(() => fixture.completed.includes('NoPlace')).toBe(true);
            await expect(page.locator('.direct-suggestions')).toHaveCount(0);
            await expect(page.getByText('No results found', { exact: true })).toHaveCount(0);
            await input(page).fill('@nobody');
            await expect.poll(() => fixture.authorRequests.includes('nobody')).toBe(true);
            await expect(page.locator('.shortcut-option')).toHaveCount(0);
            await expect(page.getByText('No results found', { exact: true })).toHaveCount(0);
            expect(fixture.multiRequests).toEqual([]);
        });

        test('supports keyboard selection of federated authors', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            await input(page).fill('Grat @anna@example.org');
            const choices = page.locator('.shortcut-option');
            await expect(choices).toHaveCount(2);
            await input(page).press('ArrowDown');
            await expect(choices.first()).toBeFocused();
            await choices.first().press('ArrowDown');
            await expect(choices.nth(1)).toBeFocused();
            await choices.nth(1).press('Enter');
            await expect(input(page)).toBeFocused();
            await expect(page.locator('.captured-filter')).toContainText('@anna@example.org');
            await input(page).press('Enter');
            await expect.poll(() => applications(page)).not.toHaveLength(0);
            expect((await applications(page)).at(-1).filter).toMatchObject({ q: 'Grat', author: 'anna-remote' });
        });

        test('keeps unconfirmed shortcuts and the removed in prefix literal', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            for (const query of ['@anna', '#alpin', 'in:Bern']) {
                await input(page).fill(query);
                if (query === 'in:Bern') await expect(page.locator('.shortcut-suggestions')).toHaveCount(0);
                else await expect(page.locator('.shortcut-option')).toHaveCount(2);
                await input(page).press('Enter');
                await expect(panel(page)).toBeHidden();
                expect((await applications(page)).at(-1).filter).toMatchObject({ q: query, author: '', tags: [], near: { radius: 2000 } });
            }
        });

        test('Escape closes the entire search from the input, shortcuts and direct suggestions', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            for (const query of ['@anna', '#alpin', 'Bern']) {
                await input(page).fill(query);
                if (query === '@anna') await expect(page.locator('.shortcut-option')).toHaveCount(2);
                else {
                    const choice = page.locator(query === 'Bern' ? '.direct-option' : '.shortcut-option').first();
                    await expect(choice).toBeVisible();
                    await input(page).press('ArrowDown');
                    await expect(choice).toBeFocused();
                }
                await page.keyboard.press('Escape');
                await expect(panel(page)).toBeHidden();
                await expect(input(page)).toBeFocused();
                await expect(input(page)).toHaveValue(query);
                expect((await appliedFilter(page)).q).toBe(query);
                await input(page).press('Escape');
                await expect(panel(page)).toBeHidden();
                await expect(input(page)).toHaveValue('');
                expect((await appliedFilter(page)).q).toBe('');
                const applications = await page.evaluate(() => (window as any).searchApplications.length);
                await input(page).press('Escape');
                expect(await page.evaluate(() => (window as any).searchApplications.length)).toBe(applications);
                await expect(panel(page)).toBeHidden();
            }
        });

        test('Escape closes the open search even after all search controls lose focus', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            await input(page).fill('Grat');
            await input(page).press('Enter');
            await expect.poll(() => applications(page)).not.toHaveLength(0);
            await input(page).fill('Latest input');
            const activity = page.locator('.activity-grid').getByRole('button', { name: 'Hiking', exact: true });
            await activity.click();
            await activity.evaluate((element: HTMLButtonElement) => element.blur());
            await expect(panel(page)).toBeVisible();
            expect(await page.evaluate(() => document.activeElement === document.body)).toBe(true);

            await page.keyboard.press('Escape');
            await expect(panel(page)).toBeHidden();
            await expect(input(page)).toBeFocused();
            await expect(input(page)).toHaveValue('Latest input');
            expect(await appliedFilter(page)).toMatchObject({ q: 'Latest input', category: ['hiking'] });

            await input(page).evaluate((element: HTMLInputElement) => element.blur());
            await page.keyboard.press('Escape');
            await expect(input(page)).not.toBeFocused();
            await expect(panel(page)).toBeHidden();
        });

        test('captures only the expression at the caret when editing the middle of a query', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            await input(page).fill('Grat @anna Schluss');
            await expect(page.locator('.shortcut-suggestions')).toHaveCount(0);
            await input(page).evaluate((element: HTMLInputElement) => {
                element.setSelectionRange(10, 10);
                element.dispatchEvent(new Event('select', { bubbles: true }));
            });
            await expect(page.locator('.shortcut-option')).toHaveCount(2);
            await page.locator('.shortcut-option').first().click();
            await expect(input(page)).toHaveValue('Grat Schluss');
            await page.getByRole('button', { name: 'Clear all', exact: true }).click();
            await expect(page.locator('.captured-filter')).toHaveCount(0);
            await expect(input(page)).toHaveValue('');
        });

        test('ignores a late place response after switching to a tag shortcut', async ({ page }) => {
            const fixture = await mountTrailSearchFixture(page, bundle);
            let release!: () => void;
            fixture.gates.set('Bern', new Promise<void>(resolve => { release = resolve; }));
            try {
                await input(page).fill('Bern');
                await expect.poll(() => fixture.requests.includes('Bern')).toBe(true);
                await input(page).fill('Grat #alp');
                await expect(page.locator('.shortcut-option')).toHaveCount(2);
                await page.locator('.shortcut-option').first().click();
                release();
                await expect.poll(() => fixture.completed.includes('Bern')).toBe(true);
                await expect(page.locator('.shortcut-suggestions')).toHaveCount(0);
                await expect(page.locator('.captured-filter')).toHaveText('#alpin');
                await input(page).press('Enter');
                await expect.poll(() => applications(page)).not.toHaveLength(0);
                expect((await applications(page)).at(-1).filter).toMatchObject({ q: 'Grat', tags: ['alpin'], near: { radius: 2000 } });
            } finally { release(); }
        });

        test('wraps place and author choices without horizontal overflow in a narrow short window', async ({ page }) => {
            await page.setViewportSize({ width: 320, height: 560 });
            await mountTrailSearchFixture(page, bundle);
            await input(page).fill('St. Gallen');
            const choices = page.locator('.direct-option');
            await expect(choices).toHaveCount(3);
            const bodyWidth = () => page.locator('.filter-body').evaluate(element => ({ client: element.clientWidth, scroll: element.scrollWidth }));
            const body = await bodyWidth();
            expect(body.scroll).toBeLessThanOrEqual(body.client);
            expect(await choices.first().evaluate(element => getComputedStyle(element).display)).toBe('inline-flex');
            await expect(page.getByRole('button', { name: 'Close search', exact: true })).toBeInViewport();
            await input(page).press('ArrowUp');
            await expect(choices.last()).toBeFocused();
            await choices.last().press('Enter');
            await expect(panel(page)).toBeHidden();
            expect(await appliedFilter(page)).toMatchObject({ q: '' });
            await input(page).fill('@anna@example.org');
            await expect(page.locator('.shortcut-option')).toHaveCount(2);
            const authorBody = await bodyWidth();
            expect(authorBody.scroll).toBeLessThanOrEqual(authorBody.client);
            await expect(page.getByRole('button', { name: 'Close search', exact: true })).toBeInViewport();
        });

        test('applies main-field text literally without automatically selecting places or activities', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            for (const query of ['Hiking', 'Bern', 'e', 'ing']) {
                await input(page).fill(query);
                await expect.poll(() => appliedFilter(page)).toMatchObject({ q: query });
                await input(page).press('Enter');
                await expect(panel(page)).toBeHidden();
                const applied = (await applications(page)).at(-1);
                expect(applied.filter.q).toBe(query);
                expect(applied.filter.category).toEqual([]);
                expect(applied.filter.near).toEqual({ radius: 2000 });
                expect(applied).not.toHaveProperty('context');
            }
        });

        test('refines only categories with subcategories and retains the other selected activities', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            await input(page).click();
            const activities = page.locator('.activity-grid');
            const walking = activities.getByRole('button', { name: 'Walking', exact: true });
            const hiking = activities.getByRole('button', { name: 'Hiking', exact: true });
            await walking.click();
            await expect(walking).toHaveAttribute('aria-pressed', 'true');
            await expect(page.locator('.refinement')).toHaveCount(0);
            await page.getByRole('button', { name: 'Close search', exact: true }).click();
            expect((await applications(page)).at(-1).filter.category).toEqual(['walking']);

            await input(page).click();
            await hiking.click();
            await expect(page.locator('.category-switch button')).toHaveCount(1);
            await expect(page.locator('.category-switch button')).toHaveText('Hiking');
            await page.getByRole('button', { name: 'Alpine hiking', exact: true }).click();
            await page.getByRole('button', { name: 'Close search', exact: true }).click();
            expect((await applications(page)).at(-1).filter.category).toEqual(['walking', 'hiking']);

            await input(page).click();
            await expect(page.getByRole('button', { name: 'Alpine hiking', exact: true })).toHaveAttribute('aria-pressed', 'true');
            await hiking.click();
            await expect(page.locator('.refinement')).toHaveCount(0);
            await expect(walking).toHaveAttribute('aria-pressed', 'true');
            await page.getByRole('button', { name: 'Close search', exact: true }).click();
            expect((await applications(page)).at(-1).filter).toMatchObject({ category: ['walking'], subcategory: [] });
        });

        test('combines text with an explicitly selected place and activity', async ({ page }) => {
            const fixture = await mountTrailSearchFixture(page, bundle);
            await input(page).fill('Grat');
            await page.getByRole('button', { name: 'Place', exact: true }).click();
            await page.getByRole('searchbox', { name: 'Place', exact: true }).fill('Zürich');
            await expect(page.locator('.place-option')).toHaveCount(5);
            await page.locator('.place-option').nth(1).click();
            await page.getByRole('slider', { name: 'Radius', exact: true }).press('ArrowRight');
            await expect(input(page)).toHaveValue('Grat');
            await page.getByRole('button', { name: /Categories/, exact: true }).click();
            await page.getByRole('button', { name: 'Hiking', exact: true }).click();
            await page.getByRole('button', { name: 'Alpine hiking', exact: true }).click();
            await page.getByRole('button', { name: 'Close search', exact: true }).click();
            await expect.poll(() => applications(page)).not.toHaveLength(0);
            const { filter } = (await applications(page)).at(-1);
            expect(filter.q).toBe('Grat');
            expect(filter.near).toMatchObject({ lat: 48.37, lon: 9.54, name: 'Zürich', radius: 2100 });
            expect(filter.category).toEqual(['hiking']);
            expect(filter.subcategory).toEqual(['alpine']);
            const activityChip = page.locator('.applied-chip').filter({ hasText: 'Hiking / Alpine hiking' });
            await expect(activityChip).toBeVisible();
            await expect(activityChip.locator('i').first()).toHaveClass(/\bfa-person-hiking\b/);
            await expect(input(page)).toHaveValue('Grat');
            await expect(page.locator('[data-filter-key="q"]')).toHaveCount(0);
            await expect(page.locator('.applied-chip').filter({ hasText: 'Zürich' }).locator('i').first()).toHaveClass(/\bfa-location-dot\b/);
            expect(fixture.requests).toContain('Zürich');
        });

        test('clears an unfinished place search when resetting all filters', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            await input(page).fill('Grat');
            await page.getByRole('button', { name: 'Place', exact: true }).click();
            const place = page.getByRole('searchbox', { name: 'Place', exact: true });
            await place.fill('Bern');
            await expect(page.locator('.place-option')).toHaveCount(5);
            const radius = page.getByRole('slider', { name: 'Radius', exact: true });
            await radius.press('ArrowRight');
            expect(Number(await radius.getAttribute('aria-valuenow'))).toBe(2100);
            await page.getByRole('button', { name: 'Clear all', exact: true }).click();
            expect(Number(await radius.getAttribute('aria-valuenow'))).toBe(2000);
            await expect(input(page)).toHaveValue('');
            await expect(place).toHaveValue('');
            await expect(page.locator('.place-option')).toHaveCount(0);
        });

        test('uses one distance track with two handles and preserves the full backend limit', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            await input(page).click();
            await page.getByRole('button', { name: 'Distance', exact: true }).click();
            await expect(page.getByRole('slider')).toHaveCount(2);
            await expect(page.locator('.noUi-target')).toHaveCount(1);
            await expect(page.locator('.range-values')).toHaveText('0 m 50.01 km+');
            expect(Number(await page.getByRole('slider', { name: 'Distance · To', exact: true }).getAttribute('aria-valuenow'))).toBe(50005);
            expect(await page.getByRole('search').evaluate((form: HTMLFormElement) => form.checkValidity()))
                .toBe(true);

            await page.getByRole('button', { name: 'Close search', exact: true }).click();
            const applied = await appliedFilter(page);
            expect(applied.distanceMin).toBe(0);
            expect(applied.distanceMax).toBe(50005);
            expect(applied.distanceLimit).toBe(50005);
            expect(await applications(page)).toEqual([]);
        });

        test('keeps compact filters inside a wide input and moves them below only when needed', async ({ page }) => {
            await page.setViewportSize({ width: 1200, height: 1000 });
            await mountTrailSearchFixture(page, bundle, { inline: true });
            await input(page).fill('Grat');
            await page.getByRole('button', { name: 'Hiking', exact: true }).click();
            await page.getByRole('button', { name: 'More filters', exact: true }).click();
            await page.locator('.advanced-filters').getByRole('button', { name: 'Elevation Gain', exact: true }).click();
            await page.getByRole('slider', { name: 'Elevation Gain · To', exact: true }).press('ArrowLeft');
            await page.getByRole('button', { name: 'Close search', exact: true }).click();
            const chips = page.locator('.applied-filters');
            const inside = async () => {
                const group = await chips.boundingBox();
                const heading = await page.locator('.search-heading').boundingBox();
                return Boolean(group && heading && group.y >= heading.y && group.y + group.height <= heading.y + heading.height);
            };
            await expect(page.locator('.applied-chip')).toHaveCount(2);
            await expect.poll(inside).toBe(true);
            const gain = page.locator('[data-filter-key="gain"]');
            await expect(gain).toContainText('≤');
            await expect(gain).not.toContainText('Elevation');
            await expect(gain).toHaveAttribute('title', /Elevation Gain/);
            await expect(gain.locator('i').first()).toHaveClass(/fa-arrow-trend-up/);
            await expect(page.locator('[data-filter-key="q"], .search-summary')).toHaveCount(0);
            const inputSpace = await input(page).evaluate(element => element.getBoundingClientRect().width - parseFloat(getComputedStyle(element).paddingRight));
            expect(inputSpace).toBeGreaterThanOrEqual(240);

            await page.setViewportSize({ width: 390, height: 700 });
            await expect.poll(inside).toBe(false);
            const group = await chips.boundingBox();
            const heading = await page.locator('.search-heading').boundingBox();
            expect(group!.y).toBeGreaterThanOrEqual(heading!.y + heading!.height);
            const width = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, viewport: innerWidth }));
            expect(width.page).toBeLessThanOrEqual(width.viewport);
            await gain.click();
            expect((await applications(page)).at(-1).filter.elevationGainMax).toBe(2000);
            await expect(page.locator('.applied-chip')).toHaveCount(1);

            await page.setViewportSize({ width: 1200, height: 1000 });
            await expect.poll(inside).toBe(true);
            await page.locator('.applied-chip').click();
            await expect(chips).toHaveCount(0);
            await expect(input(page)).toHaveValue('Grat');
            expect((await applications(page)).at(-1).filter).toMatchObject({ q: 'Grat', category: [] });
            expect(await input(page).evaluate(element => element.style.paddingRight)).toBe('');
        });

        test('keeps an explicitly selected place when closing and reopening the search', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            await input(page).fill('Grat');
            await expect.poll(() => appliedFilter(page)).toMatchObject({ q: 'Grat' });
            await page.getByRole('button', { name: 'Place', exact: true }).click();
            await page.getByRole('searchbox', { name: 'Place', exact: true }).fill('Zürich');
            await page.locator('.place-option').first().click();
            await expect.poll(() => appliedFilter(page)).toMatchObject({ q: 'Grat', near: { lat: 47.37, lon: 8.54 } });
            await expect(panel(page)).toBeVisible();
            await input(page).press('Escape');
            await input(page).click();
            await expect(page.locator('.selected-place')).toContainText('Zürich');
            expect(await appliedFilter(page)).toMatchObject({ q: 'Grat', near: { lat: 47.37, lon: 8.54 } });
        });

        test('discards pending place lookups when the search closes', async ({ page }) => {
            const fixture = await mountTrailSearchFixture(page, bundle);
            let release!: () => void;
            fixture.gates.set('Bern', new Promise<void>(resolve => { release = resolve; }));
            try {
                await input(page).click();
                await page.getByRole('button', { name: 'Place', exact: true }).click();
                await page.getByRole('searchbox', { name: 'Place', exact: true }).fill('Bern');
                await expect.poll(() => fixture.requests.includes('Bern')).toBe(true);
                await input(page).press('Escape');
                await expect(panel(page)).toBeHidden();
                await input(page).fill('Grat');
                release();
                await expect.poll(() => fixture.completed.includes('Bern')).toBe(true);
                await expect(page.locator('.selected-place')).toHaveCount(0);
                await expect(input(page)).toHaveValue('Grat');
                await expect(panel(page)).toBeVisible();
                await input(page).press('Enter');
                await expect.poll(() => applications(page)).not.toHaveLength(0);
                expect((await applications(page)).at(-1).filter.near).toEqual({ radius: 2000 });
            } finally { release(); }
        });

        test('appends more filters below the primary editor and keeps mobile actions visible', async ({ page }) => {
            await page.setViewportSize({ width: 390, height: 844 });
            await mountTrailSearchFixture(page, bundle);
            await input(page).click();
            await page.getByRole('button', { name: 'Distance', exact: true }).click();
            const lower = page.getByRole('slider', { name: 'Distance · From', exact: true });
            await lower.press('ArrowRight');
            const minimum = await lower.getAttribute('aria-valuenow');
            expect(Number(minimum)).toBeGreaterThan(0);
            await page.getByRole('button', { name: 'More filters', exact: true }).click();

            await expect(page.getByRole('group', { name: 'Distance', exact: true })).toBeVisible();
            await expect(lower).toHaveAttribute('aria-valuenow', minimum!);
            const editor = await page.locator('.general-filters .editor').boundingBox();
            const advanced = await page.locator('.advanced-filters').boundingBox();
            expect(advanced!.y).toBeGreaterThan(editor!.y + editor!.height - 1);
            await expect(page.getByRole('button', { name: 'Close search', exact: true })).toBeInViewport();
            await expect(page.locator('.search-footer')).toHaveCount(0);

            await page.getByRole('button', { name: 'Close search', exact: true }).click();
            await expect(panel(page)).toBeHidden();
            expect((await appliedFilter(page)).distanceMin).toBe(Number(minimum));
            await input(page).click();
            await expect(lower).toBeVisible();
            expect(Number(await lower.getAttribute('aria-valuenow'))).toBe(Number(minimum));
        });

        test('elevation handles apply independent intervals and follow draft resets', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle);
            await input(page).click();
            await page.getByRole('button', { name: 'More filters', exact: true }).click();
            const advanced = page.locator('.advanced-filters');
            const gain = page.getByRole('group', { name: 'Elevation Gain', exact: true });
            const loss = page.getByRole('group', { name: 'Elevation Loss', exact: true });
            const gainMin = gain.getByRole('slider', { name: /From/ });
            const gainMax = gain.getByRole('slider', { name: /To/ });
            const lossMax = loss.getByRole('slider', { name: /To/ });
            await advanced.getByRole('button', { name: 'Elevation Gain', exact: true }).click();
            await expect(gain.getByRole('slider')).toHaveCount(2);
            await expect(loss).toHaveCount(0);
            await expect(advanced.locator('.editor')).toHaveCount(1);
            await expect(advanced.locator('input[type="range"]')).toHaveCount(0);
            await gainMin.press('ArrowRight');
            await gainMax.press('ArrowLeft');
            await advanced.getByRole('button', { name: 'Elevation Loss', exact: true }).click();
            await expect(loss.getByRole('slider')).toHaveCount(2);
            await lossMax.press('ArrowLeft');
            await page.getByRole('button', { name: 'Clear all', exact: true }).click();
            expect(Number(await lossMax.getAttribute('aria-valuenow'))).toBe(2000);
            await advanced.getByRole('button', { name: 'Elevation Gain', exact: true }).click();
            expect(Number(await gainMin.getAttribute('aria-valuenow'))).toBe(0);
            expect(Number(await gainMax.getAttribute('aria-valuenow'))).toBe(2000);
            await gainMin.press('ArrowRight');
            await gainMax.press('ArrowLeft');
            await advanced.getByRole('button', { name: 'Elevation Loss', exact: true }).click();
            await lossMax.press('ArrowLeft');
            await page.getByRole('button', { name: 'Close search', exact: true }).click();
            await expect.poll(() => applications(page)).not.toHaveLength(0);
            const { filter } = (await applications(page)).at(-1);
            expect(filter.elevationGainMin).toBeGreaterThan(0);
            expect(filter.elevationGainMax).toBeLessThan(2000);
            expect(filter.elevationLossMin).toBe(0);
            expect(filter.elevationLossMax).toBeLessThan(2000);
        });

        test('moves filters in both directions while keeping their selections and advanced count', async ({ page }) => {
            await mountTrailSearchFixture(page, bundle, { sections: { activity: 'advanced', distance: 'advanced', gain: 'general' } });
            await input(page).click();
            await expect(page.locator('.general-filters .filter-chip:not(.advanced-toggle)')).toHaveText(['Place', 'Difficulty', 'Completion Status', 'Elevation Gain']);
            await expect(page.locator('.activity-grid')).toHaveCount(0);
            await page.getByRole('button', { name: 'Elevation Gain', exact: true }).click();
            await page.getByRole('slider', { name: 'Elevation Gain · From', exact: true }).press('ArrowRight');
            await page.getByRole('button', { name: 'More filters', exact: true }).click();
            const advanced = page.locator('.advanced-filters');
            await advanced.getByRole('button', { name: 'Hiking', exact: true }).click();
            await advanced.getByRole('button', { name: 'Alpine hiking', exact: true }).click();
            await expect(page.locator('.advanced-toggle')).toHaveText('Fewer filters · 1');
            await advanced.getByRole('button', { name: 'Distance', exact: true }).click();
            await advanced.getByRole('slider', { name: 'Distance · To', exact: true }).press('ArrowLeft');
            await expect(page.locator('.advanced-toggle')).toHaveText('Fewer filters · 2');
            await expect(advanced.getByRole('group', { name: 'Elevation Gain', exact: true })).toHaveCount(0);
            await page.getByRole('button', { name: 'Close search', exact: true }).click();
            await expect.poll(() => applications(page)).not.toHaveLength(0);
            const { filter } = (await applications(page)).at(-1);
            expect(filter.category).toEqual(['hiking']);
            expect(filter.subcategory).toEqual(['alpine']);
            expect(filter.elevationGainMin).toBeGreaterThan(0);
            expect(filter.distanceMax).toBeLessThan(filter.distanceLimit);
            await input(page).click();
            await expect(page.locator('.general-filters .filter-chip:not(.advanced-toggle)')).toHaveText(['Place', 'Difficulty', 'Completion Status', 'Elevation Gain']);
            await advanced.locator('.filter-chip').filter({ hasText: 'Hiking' }).click();
            await expect(advanced.getByRole('button', { name: 'Alpine hiking', exact: true })).toHaveAttribute('aria-pressed', 'true');
        });

        test('supports the independent place editor when every filter is advanced', async ({ page }) => {
            const sections = Object.fromEntries(trailSearchFilters.map(filter => [filter.id, 'advanced'])) as TrailSearchFilterSections;
            await mountTrailSearchFixture(page, bundle, { sections });
            await input(page).fill('Grat');
            await expect(page.locator('.filter-chip:not(.advanced-toggle)')).toHaveCount(0);
            await expect(page.locator('.editor')).toHaveCount(0);
            await page.getByRole('button', { name: 'More filters', exact: true }).click();
            await page.locator('.advanced-filters').getByRole('button', { name: 'Place', exact: true }).click();
            await page.getByRole('searchbox', { name: 'Place', exact: true }).fill('Zürich');
            await page.locator('.place-option').first().click();
            await expect(page.locator('.selected-place')).toContainText('Zürich');
            await page.getByRole('button', { name: 'Close search', exact: true }).click();
            await expect.poll(() => applications(page)).not.toHaveLength(0);
            expect((await applications(page)).at(-1).filter).toMatchObject({ q: 'Grat', near: { name: 'Zürich' } });
        });

        test('supports every filter in general search without an empty advanced section', async ({ page }) => {
            const sections = Object.fromEntries(trailSearchFilters.map(filter => [filter.id, 'general'])) as TrailSearchFilterSections;
            await mountTrailSearchFixture(page, bundle, { sections });
            await input(page).click();
            await expect(page.locator('.filter-chip')).toHaveCount(trailSearchFilters.length);
            await expect(page.locator('.advanced-toggle')).toHaveCount(0);
            await page.getByRole('button', { name: 'Date', exact: true }).click();
            await page.getByLabel('After', { exact: true }).fill('2025-01-01');
            await page.getByLabel('Before', { exact: true }).fill('2025-12-31');
            await page.getByRole('button', { name: 'Completion Status', exact: true }).click();
            await page.getByRole('radio', { name: 'Completed', exact: true }).check();
            await page.getByRole('button', { name: 'Close search', exact: true }).click();
            await expect.poll(() => applications(page)).not.toHaveLength(0);
            expect((await applications(page)).at(-1).filter).toMatchObject({ startDate: '2025-01-01', endDate: '2025-12-31', completed: true });
        });
    });
}
