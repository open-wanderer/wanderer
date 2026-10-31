import { devices, expect, test } from '@playwright/test';
import { buildPointerFixture, mountPointerResults, clippedPoint, position, expectStablePress } from './map-focus-pointer-fixture';

test.use({
    viewport: devices['Pixel 7'].viewport,
    deviceScaleFactor: devices['Pixel 7'].deviceScaleFactor,
    userAgent: devices['Pixel 7'].userAgent,
    hasTouch: true,
    isMobile: true,
});
test.skip(({ browserName }) => browserName !== 'chromium', 'Pixel 7 mobile emulation uses Chromium');

for (const dev of [true, false]) test.describe(`map touch focus (${dev ? 'development' : 'production'})`, () => {
    let bundle: string;
    test.beforeAll(async () => { bundle = await buildPointerFixture(dev); });

    for (const display of ['cards', 'list'] as const) test(`${display}: a real tap in the clipped checkbox zone navigates without scrolling`, async ({ page }) => {
        const records = await mountPointerResults(page, bundle, display);
        const point = await clippedPoint(page, display);
        records.length = 0;
        const before = await position(page);
        await page.touchscreen.tap(point.x, point.y);
        await expect(page).toHaveURL(new RegExp(point.href + '$'));
        await expect(page.getByTestId('opened-trail')).toHaveText(point.href);
        await expectStablePress(records, before);
    });
});
