import { expect, test } from '@playwright/test';
import { buildTrailSearchFixture, mountTrailSearchFixture } from './trail-search-fixture';

for (const dev of [true, false]) {
    test.describe(`sidebar Escape regressions (${dev ? 'development' : 'production'})`, () => {
        let bundle: string;
        test.beforeAll(async () => { bundle = await buildTrailSearchFixture(dev); });
        for (const side of ['left', 'right']) {
            test(`${side}: Escape hides the whole surface and refocus, click and ArrowDown reopen it`, async ({ page }) => {
                await mountTrailSearchFixture(page, bundle, { side: side as 'left' | 'right', layout: 'sidebar' });
                const input = page.getByRole('searchbox', { name: 'Search trails', exact: true });
                await input.fill('Bern');
                await expect(page.locator('.sidebar-suggestions')).toBeVisible();
                for (const reopen of ['click', 'focus', 'ArrowDown']) {
                    await input.press('Escape');
                    await expect(input).toHaveValue('Bern');
                    await expect(page.locator('.sidebar-suggestions')).toBeHidden();
                    if (reopen === 'click') await input.click();
                    else if (reopen === 'focus') { await page.getByTestId('outside').focus(); await input.focus(); }
                    else await input.press('ArrowDown');
                    await expect(page.locator('.sidebar-suggestions')).toBeVisible();
                    if (reopen === 'ArrowDown') await expect(page.locator('.direct-option').first()).toBeFocused();
                    await input.focus();
                }
                await input.press('Escape');
                await page.evaluate(() => (window as any).setFixtureLayout('overlay'));
                await expect(page.locator('.sidebar-panel')).toHaveCount(0);
                await expect(page.locator('.direct-option').first()).toBeVisible();
                await input.press('ArrowDown');
                await expect(page.locator('.direct-option').first()).toBeFocused();
            });
            test(`${side}: Escape dismisses status and captured filters without clearing text`, async ({ page }) => {
                await mountTrailSearchFixture(page, bundle, { side: side as 'left' | 'right', layout: 'sidebar', anonymous: true });
                const input = page.getByRole('searchbox', { name: 'Search trails', exact: true });
                await input.fill('Grat @an');
                await expect(page.locator('.sidebar-suggestions')).toContainText('Sign in');
                await input.press('Escape');
                await expect(input).toHaveValue('Grat @an');
                await expect(page.locator('.sidebar-suggestions')).toBeHidden();
                await mountTrailSearchFixture(page, bundle, { side: side as 'left' | 'right', layout: 'sidebar' });
                await input.fill('@an');
                await page.locator('.shortcut-option').first().click();
                await expect(page.locator('.captured-filters')).toBeVisible();
                await input.press('Escape');
                await expect(page.locator('.sidebar-suggestions')).toBeHidden();
                await expect(input).toHaveValue('');
            });
            test(`${side}: empty Escape never publishes and filter popups consume only their first Escape`, async ({ page }) => {
                await mountTrailSearchFixture(page, bundle, { side: side as 'left' | 'right', layout: 'sidebar' });
                const input = page.getByRole('searchbox', { name: 'Search trails', exact: true });
                await input.focus();
                const count = await page.evaluate(() => (window as any).fixturePublicationAttempts ?? 0);
                await input.press('Escape');
                await expect(input).toBeFocused();
                expect(await page.evaluate(() => (window as any).fixturePublicationAttempts ?? 0)).toBe(count);
                await expect(page.locator('.sidebar-panel')).toBeVisible();
                const place = page.getByRole('searchbox', { name: 'Place', exact: true });
                await place.fill('Bern');
                await expect(page.locator('.place-option').first()).toBeVisible();
                await page.locator('.place-option').first().focus();
                await page.keyboard.press('Escape');
                await expect(place).toHaveValue('Bern');
                await expect(place).toBeFocused();
                await expect(page.locator('.place-results')).toHaveCount(0);
                await page.getByRole('button', { name: 'More filters', exact: true }).click();
                const tags = page.locator('.filter-group:has(h3:text-is("Tags")) input');
                await tags.fill('alpine');
                await expect(page.locator('.filter-group:has(h3:text-is("Tags")) ul')).toBeVisible();
                await tags.press('Escape');
                await expect(tags).toHaveValue('alpine');
                await expect(page.locator('.filter-group:has(h3:text-is("Tags")) ul')).toBeHidden();
                const difficulty = page.locator('.difficulty-editor [aria-expanded]');
                await difficulty.click();
                await page.locator('.difficulty-editor button').first().focus();
                await page.keyboard.press('Escape');
                await expect(difficulty).toHaveAttribute('aria-expanded', 'false');
                await expect(difficulty).toBeFocused();
                const author = page.locator('.author-control input');
                await author.fill('anna');
                await expect(page.locator('.author-control ul')).toBeVisible();
                await author.press('Escape');
                await expect(author).toHaveValue('anna');
                await expect(page.locator('.author-control ul')).toHaveCount(0);
            });
        }
    });
}

for (const dev of [true, false]) {
    test.describe(`sidebar keyboard ownership (${dev ? 'development' : 'production'})`, () => {
        let bundle: string;
        test.beforeAll(async () => { bundle = await buildTrailSearchFixture(dev); });
        for (const side of ['left', 'right'] as const) {
            test(`${side}: author navigation never selects and the wrapper closes on exit`, async ({ page }) => {
                await mountTrailSearchFixture(page, bundle, { layout: 'sidebar', side });
                const column = page.locator('.fixture-columns aside');
                const more = page.getByRole('button', { name: 'More filters', exact: true });
                await more.click();
                const author = column.locator('.author-control input');
                await author.fill('anna');
                const options = column.locator('.author-control li');
                await expect(options.first()).toBeVisible();
                const before = await page.evaluate(() => (window as any).searchApplications.length);
                for (const key of ['Tab', 'ArrowDown', 'Shift+Tab', 'Shift']) {
                    await author.focus();
                    await author.click();
                    await options.nth(1).focus();
                    await page.keyboard.press(key);
                    await page.waitForTimeout(500);
                    expect(await page.evaluate(() => (window as any).appliedSearchFilter.author)).toBe('');
                    expect(await page.evaluate(() => (window as any).searchApplications.length)).toBe(before);
                }
                await author.click();
                await options.first().focus();
                await page.keyboard.press('Escape');
                await expect(author).toBeFocused();
                await expect(options).toHaveCount(0);
                for (const reopen of ['click', 'ArrowDown']) {
                    if (reopen === 'click') await author.click();
                    else await author.press('ArrowDown');
                    await expect(options.first()).toBeVisible();
                    await author.press('Escape');
                }
                await author.click();
                await options.first().focus();
                await page.getByTestId('outside').focus();
                await expect(options).toHaveCount(0);
                await author.click();
                await options.first().press('Enter');
                await expect.poll(() => page.evaluate(() => (window as any).appliedSearchFilter.author)).not.toBe('');
            });
            test(`${side}: only rendered place popups consume Escape; radius and loading retain native behavior`, async ({ page }) => {
                const fixture = await mountTrailSearchFixture(page, bundle, { layout: 'sidebar', side });
                let release!: () => void;
                fixture.gates.set('Pending', new Promise<void>(resolve => release = resolve));
                const place = page.getByRole('searchbox', { name: 'Place', exact: true });
                await place.fill('Pending');
                await expect.poll(() => fixture.requests.includes('Pending')).toBe(true);
                await place.press('Escape');
                await expect(place).toHaveValue('');
                release();
                await place.fill('Bern');
                await expect(page.locator('.place-option').first()).toBeVisible();
                await place.press('Escape');
                await place.press('ArrowDown');
                await expect(page.locator('.place-option').first()).toBeVisible();
                const radius = page.locator('.radius-control [role=slider]').first();
                await radius.focus();
                await page.keyboard.press('Escape');
                await expect(radius).toBeFocused();
                await expect(place).toHaveValue('Bern');
                const main = page.getByRole('searchbox', { name: 'Search trails', exact: true });
                await main.fill('Bern');
                await expect(page.locator('.sidebar-suggestions')).toBeVisible();
                await page.locator('.reset-button').focus();
                await page.keyboard.press('Escape');
                await expect(page.locator('.sidebar-suggestions')).toBeHidden();
                await expect(main).toHaveValue('Bern');
                await page.keyboard.press('Escape');
                await expect(page.locator('.reset-button')).toBeFocused();
                await expect(main).toHaveValue('Bern');
            });
        }
    });
}
