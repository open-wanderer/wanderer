import { expect, test, devices } from '@playwright/test';
import { buildMapPageFixture, mountMapPageFixture } from './map-page-fixture';
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
                    for (const selector of ['.overlay-move']) {

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
    });
}

for (const dev of [true, false]) test.describe(`upload drawer stacking (${dev ? 'development' : 'production'})`, () => {
    let bundle: string;
    test.beforeAll(async () => { bundle = await buildMapPageFixture(dev, true, false, true); });
    for (const width of [412, 900]) {
        test(`${width}px, drawer actions stay above all upload states`, async ({ page }) => {
            await page.setViewportSize({ width, height: 915 });
            for (const status of ['running', 'minimized', 'done']) {
                await mountMapPageFixture(page, bundle);

                await page.evaluate(() => { sessionStorage.removeItem('fixture:loggedOut'); (window as any).fixture.addUpload(); });
                if (status === 'minimized') await page.getByRole('button', { name: 'Minimize', exact: true }).click();
                if (status === 'done') await page.evaluate(() => (window as any).fixture.completeUpload());
                await page.getByRole('button', { name: 'Toggle drawer', exact: true }).last().click();

                const upload = page.locator('.fixed.bottom-4.right-4[role="presentation"]');
                expect(await upload.evaluate(node => Number(getComputedStyle(node).zIndex))).toBe(10);
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
