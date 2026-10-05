import { expect, test } from '@playwright/test';
import { buildTrailModalFixture, mountTrailModalFixture } from './trail-modal-fixture';

// Exercise a production rich-text sink with browser DOMPurify, rather than
// relying only on string assertions from the server's DOM implementation.
const fixture = `
    <script>
        import Card from '$lib/components/waypoint/waypoint_card.svelte';
        let waypoint = $state({
            id: 'cached-waypoint', name: 'Cached waypoint', lat: 46, lon: 8,
            photos: [], description: window.cachedRichText
        });
        window.updateRichText = value => { waypoint.description = value; };
    </script>
    <section data-testid="rich-text"><Card {waypoint} /></section>
`;

let bundle: string;
test.beforeAll(async () => { bundle = await buildTrailModalFixture(false, fixture); });

test('sanitizes cached HTML and later updates in the actual waypoint card', async ({ page }) => {
    const errors: string[] = [];
    page.on('pageerror', error => errors.push(error.message));
    await page.addInitScript(() => {
        (window as any).unsafeRenderCalls = 0;
        (window as any).recordUnsafeRender = () => { (window as any).unsafeRenderCalls++; };
        (window as any).cachedRichText = '<p onclick="recordUnsafeRender()">Grüezi <strong>山 🚲</strong></p>' +
            '<img src="/missing-image" onerror="recordUnsafeRender()">' +
            '<a class="mention" href="https://example.test/profile">@friend</a>' +
            '<a href="javascript:recordUnsafeRender()">Unsafe link</a>';
    });
    await mountTrailModalFixture(page, bundle);
    const card = page.getByTestId('rich-text');
    await expect(card.locator('strong')).toHaveText('山 🚲');
    await expect(card.locator('a.mention')).toHaveAttribute('href', 'https://example.test/profile');
    await expect(card.locator('[onclick], [onerror], a[href^="javascript:"]')).toHaveCount(0);
    await card.getByText('Grüezi', { exact: false }).click();
    await card.getByText('Unsafe link').click();

    await page.evaluate(() => (window as any).updateRichText(
        '<svg><g onload="recordUnsafeRender()"></g></svg>' +
        '<iframe srcdoc="<script>recordUnsafeRender()</script>"></iframe>' +
        '<p>Updated <em>safe text</em> &amp; friends</p>'
    ));
    await expect(card.locator('em')).toHaveText('safe text');
    await expect(card.locator('svg, iframe, script, [onload]')).toHaveCount(0);
    await expect(card).toContainText('Updated safe text & friends');
    expect(await page.evaluate(() => (window as any).unsafeRenderCalls)).toBe(0);
    expect(errors).toEqual([]);
});
