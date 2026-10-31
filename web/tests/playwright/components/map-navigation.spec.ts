import { expect, test, devices } from '@playwright/test';
import { buildMapPageFixture, mountMapPageFixture } from './map-page-fixture';
import { setBrowserFullscreen } from '../browser_fullscreen';
for (const dev of [true, false]) {
    test.describe(`map navigation overlays (${dev ? 'development' : 'production'})`, () => {
        let bundle: string;
        test.beforeAll(async () => { bundle = await buildMapPageFixture(dev, true, false, true); });
        test.describe('touch surfaces', () => {
            test.use({ hasTouch: true, isMobile: true, userAgent: devices['Galaxy Tab S4 landscape'].userAgent, viewport: devices['Galaxy Tab S4 landscape'].viewport });
            for (const theme of ['light', 'dark']) {
                test(`map tools stay opaque after tapping in ${theme}`, async ({ page }) => {
                    await mountMapPageFixture(page, bundle);
                    expect(await page.evaluate(() => matchMedia('(hover: none)').matches)).toBe(true);
                    await page.evaluate(theme => { for (const node of [document.documentElement, document.body]) { node.classList.remove('light', 'dark'); node.classList.add(theme); } }, theme);
                    const cdp = await page.context().newCDPSession(page);
                    await cdp.send('DOM.enable');
                    await cdp.send('CSS.enable');
                    await page.locator('.result-header .expand-results').tap();
                    for (const selector of ['.overlay-move', '.fullscreen-search-toggle']) {
                        if (selector.includes('fullscreen')) await setBrowserFullscreen(page, true);
                        await expect(page.locator('.map-results')).toHaveAttribute('data-layout-ready', 'true');
                        await page.locator(selector).tap();
                        // Retained touch hover varies with a compositor repaint.
                        // Keep it explicit so this test always exercises that state.
                        const { root } = await cdp.send('DOM.getDocument');
                        const { nodeId } = await cdp.send('DOM.querySelector', { nodeId: root.nodeId, selector });
                        await cdp.send('CSS.forcePseudoState', { nodeId, forcedPseudoClasses: ['hover'] });
                        await page.locator(selector).evaluate(async element => {
                            getComputedStyle(element).backgroundColor;
                            await Promise.all(element.getAnimations().map(animation => animation.finished));
                        });
                        expect(await page.locator(selector).evaluate(element => ({ hover: element.matches(':hover'), color: getComputedStyle(element).backgroundColor }))).toMatchObject({ hover: true, color: theme === 'light' ? 'rgb(255, 255, 255)' : 'rgb(30, 32, 40)' });
                    }
                    await cdp.detach();
                });
            }
        });

        test('the real navigation drawer stays closable in fullscreen', async ({ page }) => {
            await page.setViewportSize({ width: 900, height: 800 });
            await mountMapPageFixture(page, bundle);
            await page.getByRole('button', { name: 'Toggle drawer', exact: true }).last().click();
            await expect(page.locator('.drawer-overlay')).toBeVisible();
            await setBrowserFullscreen(page, true);
            await expect(page.locator('.drawer-overlay')).toHaveJSProperty('inert', false);
            await expect(page.locator('.drawer-overlay')).toHaveAttribute('tabindex', '0');
            await page.locator('.drawer-overlay').evaluate(element => (element.nextElementSibling!.querySelector('button[aria-label="Toggle drawer"]') as HTMLButtonElement).click());
            await expect(page.locator('.drawer-overlay')).toHaveCount(0);
            await setBrowserFullscreen(page, false);
            await page.getByRole('button', { name: 'Toggle drawer', exact: true }).last().click();
            await setBrowserFullscreen(page, true);
            await page.locator('.drawer-overlay').click({ position: { x: 20, y: 100 } });
            await expect(page.locator('.drawer-overlay')).toHaveCount(0);
        });
        test('an open create menu stays keyboard accessible until closed', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await page.locator('.create-trigger').click();
            await expect(page.locator('.create-panel')).toBeVisible();
            await setBrowserFullscreen(page, true);
            await page.locator('.create-panel a').first().focus();
            await expect(page.locator('.create-panel a').first()).toBeFocused();
            await page.keyboard.press('Escape');
            await expect(page.locator('.create-panel')).toHaveCount(0);
        });
        for (const menu of ['create', 'user', 'notifications']) {
            test(`only the open ${menu} overlay stays focusable in fullscreen`, async ({ page }) => {
                await mountMapPageFixture(page, bundle, { backgroundNav: true });
                if (menu === 'create') await page.locator('.create-trigger').click();
                else await page.getByRole('button', { name: menu === 'user' ? 'Open user menu' : 'Toggle notification dropdown', exact: true }).click();
                await expect(page.locator('nav:has(#nav-bar-links) [data-fullscreen-overlay]')).toBeVisible();
                await setBrowserFullscreen(page, true);
                await expect(page.locator('#background-nav')).toHaveJSProperty('inert', true);
                await expect.poll(() => page.locator('nav:has(#nav-bar-links)').evaluate(nav =>
                    [...nav.querySelectorAll('a[href], button')].filter(element => !element.closest('[data-fullscreen-overlay]'))
                        .every(element => Boolean(element.closest('[inert]'))))).toBe(true);
                expect(await page.locator('nav:has(#nav-bar-links) [data-fullscreen-overlay]').evaluate(element => element.closest('[inert]') === null)).toBe(true);
                for (let i = 0; i < 15; i++) {
                    await page.keyboard.press('Tab');
                    expect(await page.evaluate(() => !document.activeElement?.closest('nav:has(#nav-bar-links)') || Boolean(document.activeElement.closest('[data-fullscreen-overlay]')))).toBe(true);
                }
            });
        }
        test('URL import opened in fullscreen receives synchronous modal focus', async ({ page }) => {
            await mountMapPageFixture(page, bundle);
            await page.locator('.create-trigger').click();
            await setBrowserFullscreen(page, true);
            await expect(page.locator('#url-import-modal')).toHaveJSProperty('inert', false);
            await page.locator('.create-panel button').last().click();
            const dialog = page.locator('dialog[open]');
            await expect(dialog).toBeVisible();
            expect(await dialog.evaluate(element => element.contains(document.activeElement))).toBe(true);
            await expect.poll(() => dialog.ariaSnapshot()).toContain('Close modal');
        });
        for (const when of ['before', 'during']) {
            test(`upload created ${when} fullscreen stays above the map and usable`, async ({ page }) => {
                await mountMapPageFixture(page, bundle);
                if (when === 'before') await page.evaluate(() => (window as any).fixture.addUpload());
                await page.getByTestId('fixture-search').focus();
                await setBrowserFullscreen(page, true);
                await expect(page.locator('.fullscreen-search-toggle')).toBeFocused();
                if (when === 'during') await page.evaluate(() => (window as any).fixture.addUpload());
                const cancel = page.getByRole('button', { name: 'Cancel upload', exact: true });
                await expect(cancel).toBeVisible();
                await expect.poll(() => cancel.evaluate(element => {
                    const r = element.getBoundingClientRect();
                    return element.contains(document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2)) && !element.closest('[inert]');
                })).toBe(true);
                await cancel.click();
                await expect(cancel).toHaveCount(0);
            });
        }
    });
}

for (const dev of [true, false]) test.describe(`upload drawer stacking (${dev ? 'development' : 'production'})`, () => {
    let bundle: string;
    test.beforeAll(async () => { bundle = await buildMapPageFixture(dev, true, false, true); });
    for (const width of [412, 900]) for (const fullscreen of [false, true]) {
        test(`${width}px, fullscreen=${fullscreen}: drawer actions stay above all upload states`, async ({ page }) => {
            await page.setViewportSize({ width, height: 915 });
            for (const status of ['running', 'minimized', 'done']) {
                await mountMapPageFixture(page, bundle);
                await setBrowserFullscreen(page, false);
                await page.evaluate(() => { sessionStorage.removeItem('fixture:loggedOut'); (window as any).fixture.addUpload(); });
                if (status === 'minimized') await page.getByRole('button', { name: 'Minimize', exact: true }).click();
                if (status === 'done') await page.evaluate(() => (window as any).fixture.completeUpload());
                await page.getByRole('button', { name: 'Toggle drawer', exact: true }).last().click();
                if (fullscreen) await setBrowserFullscreen(page, true);
                const upload = page.locator('.upload-dialog');
                expect(await upload.evaluate(node => Number(getComputedStyle(node).zIndex))).toBe(fullscreen ? 45 : 10);
                const panel = page.locator('.drawer-overlay + div');
                await expect(panel).toBeVisible();
                for (const selector of ['a:has(img[alt="avatar"])', 'a:has(p:text-is("tester"))', '[aria-label="Logout"]']) {
                    const control = panel.locator(selector);
                    await expect(control).toBeVisible();
                    expect(await control.evaluate(node => {
                        const r = node.getBoundingClientRect();
                        return node.contains(document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2));
                    })).toBe(true);
                }
                await panel.getByRole('button', { name: 'Logout', exact: true }).click();
                await page.waitForURL(url => url.pathname === '/');
                expect(await page.evaluate(() => sessionStorage.getItem('fixture:loggedOut'))).toBe('true');
                await expect(page.locator('.drawer-overlay')).toHaveCount(0);
            }
        });
    }
});
