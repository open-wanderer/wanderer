import { expect, test, type Page } from '@playwright/test';
import { buildMapPageFixture } from './map-page-fixture';
import { counts, mountEngine, waitForEngine } from './map-engine-fixture';
import { dispatchFullscreenKey, installBrowserFullscreen, setBrowserFullscreen } from '../browser_fullscreen';
const expanded = (page: Page) => page.locator('.map-page.expanded');
for (const dev of [true, false]) test.describe(`map F11 (${dev ? 'development' : 'production'})`, () => {
    let bundle: string;
    test.beforeAll(async () => { bundle = await buildMapPageFixture(dev, true, true, true, true, true); });
    test.beforeEach(async ({page}) => { await installBrowserFullscreen(page); await mountEngine(page, bundle); });
    test('fullscreen change and resize couple the layout with one viewport search and no F11 interception', async ({page}) => {
        const before = await counts(page);
        expect(await dispatchFullscreenKey(page)).toBe(false);
        await setBrowserFullscreen(page, true); await expect(expanded(page)).toHaveCount(1); await waitForEngine(page);
        await expect.poll(() => counts(page).then(c => c.results)).toBe(before.results + 1);
        await page.waitForTimeout(450); expect((await counts(page)).results).toBe(before.results + 1);
        await setBrowserFullscreen(page, false); await expect(expanded(page)).toHaveCount(0);
        // Resize is the fallback notification, never a window-size heuristic.
        await page.evaluate(() => (window as any).setTestBrowserFullscreen(true, false, true)); await expect(expanded(page)).toHaveCount(1);
        await page.evaluate(() => (window as any).setTestBrowserFullscreen(false, false, true)); await expect(expanded(page)).toHaveCount(0);
        await page.setViewportSize({width:1000,height:900}); await expect(expanded(page)).toHaveCount(0);
    });
    test('manual enlargement survives leaving fullscreen', async ({page}) => {
        await page.locator('.enlarge-map').click(); await setBrowserFullscreen(page,true); await setBrowserFullscreen(page,false);
        await expect(expanded(page)).toHaveCount(1);
        expect(await page.evaluate(() => (window as any).fixture.capture().expanded)).toBe(true);
    });
    test('manual compression stays off during resize and after leaving fullscreen', async ({page}) => {
        await setBrowserFullscreen(page,true); await page.locator('.enlarge-map').click(); await expect(expanded(page)).toHaveCount(0);
        await page.setViewportSize({width:1100,height:900}); await expect(expanded(page)).toHaveCount(0);
        await setBrowserFullscreen(page,false); await expect(expanded(page)).toHaveCount(0);
    });
    test('Escape cannot end an automatically enabled mode', async ({page}) => {
        await setBrowserFullscreen(page,true); await page.locator('.maplibregl-canvas').focus(); await page.keyboard.press('Escape');
        await expect(expanded(page)).toHaveCount(1);
        expect(await page.evaluate(() => (window as any).fixture.capture().expanded)).toBe(false);
    });
    test('loading and restoring in fullscreen enable mode without persisting its automatic state', async ({page}) => {
        await setBrowserFullscreen(page,true); await page.locator('.enlarge-map').click();
        const saved = await page.evaluate(() => (window as any).fixture.capture()); expect(saved.expanded).toBe(false);
        await page.evaluate(saved => (window as any).fixture.remount(saved), saved); await expect(expanded(page)).toHaveCount(1);
        await mountEngine(page,bundle); await expect(expanded(page)).toHaveCount(1);
        expect(await page.evaluate(() => (window as any).fixture.capture().expanded)).toBe(false);
        await setBrowserFullscreen(page,false); await expect(expanded(page)).toHaveCount(0);
    });
    for (const focus of ['search','result']) test(`F11 retains the visible ${focus} focus and searches the whole map once`, async ({page}) => {
        const target = focus==='search' ? page.getByRole('searchbox') : page.locator('.trail-result a').first();
        await target.focus(); await expect(target).toBeFocused();
        await target.evaluate(node => (window as any).f11FocusedElement = node);
        const before = await counts(page);
        await setBrowserFullscreen(page,true); await waitForEngine(page);
        await expect(target).toBeVisible(); await expect(target).toBeFocused();
        expect(await target.evaluate(node => node === (window as any).f11FocusedElement && node.isConnected)).toBe(true);
        await expect.poll(() => counts(page).then(value=>value.results)).toBe(before.results+1);
        await page.waitForTimeout(450); expect((await counts(page)).results).toBe(before.results+1);
        expect(await page.locator('#trail-map').evaluate(node=>{const r=node.getBoundingClientRect();return [r.x,r.y,r.width,r.height];})).toEqual([0,0,1280,720]);
    });
    test('an open modal keeps focus, input and its normal Escape', async ({page}) => {
        await page.evaluate(() => (window as any).fixture.openDialog());
        const input = page.locator('dialog[open] input'); await input.fill('Unchanged input');
        await setBrowserFullscreen(page,true); await expect(input).toBeFocused(); await expect(input).toHaveValue('Unchanged input');
        await expect(input).toBeVisible(); await input.fill('Still usable');
        await page.keyboard.press('Escape'); await expect(page.locator('dialog[open]')).toHaveCount(0); await expect(expanded(page)).toHaveCount(1);
    });
});
