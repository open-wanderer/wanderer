import { expect, test, type Locator } from '@playwright/test';
import { buildTrailModalFixture, mountTrailModalFixture } from './trail-modal-fixture';

// Keep the real shared result link and presentations. Both result parents use
// this composition; the small parent only owns selection and counts link clicks.
const source = `
    <script>
        import TrailResultLink from '#lib/components/trail/trail_result_link.svelte';
        import TrailCard from '#lib/components/trail/trail_card.svelte';
        import TrailListItem from '#lib/components/trail/trail_list_item.svelte';
        let selected = $state([]);
        let list = $state(false);
        const trails = ['Ridge loop', 'Valley trail'].map((name, index) => ({ id: String(index), name,
            photos: [], tags: [], public: false, difficulty: 'easy', distance: 2400,
            duration: 3600, elevation_gain: 100, elevation_loss: 100 }));
        window.resultHoverLinkClicks = 0;
        function toggle(trail) {
            selected = selected.includes(trail.id) ? selected.filter(id => id !== trail.id) : [...selected, trail.id];
        }
    </script>
    <button onclick={() => { list = !list; selected = []; }}>Change presentation</button>
    {#each trails as trail}
        <TrailResultLink {trail} {list} selected={selected.includes(trail.id)} href={'/trail/' + trail.id}
            onclick={event => { event.preventDefault(); window.resultHoverLinkClicks++; if (selected.length) toggle(trail); }}
            onselect={() => toggle(trail)}>
            {#snippet children(hovered)}
                {#if list}<TrailListItem {trail} {hovered} selected={selected.includes(trail.id)} />
                {:else}<TrailCard {trail} {hovered} selected={selected.includes(trail.id)} />{/if}
            {/snippet}
        </TrailResultLink>
    {/each}
    <style>
        :global(body) { --background: 255,255,255; --content: 30,40,35; --secondary-hover: 235,240,235; }
        :global(.trail-result) { position: relative; width: 400px; margin-top: 20px; }
        :global(.trail-card) { height: 386px; }
        :global(.trail-card > div:first-child) { height: 192px; overflow: hidden; }
        :global(#header-img) { width: 100%; height: 100%; }
        :global(.trail-result li) { min-height: 160px; padding: 16px; }
        :global(.trail-result li > div:first-child) { display: none; }
        /* The fixture does not build Tailwind; retain its selection utility. */
        :global(.bg-secondary-hover) { background-color: rgba(var(--secondary-hover)); }
    </style>
`;

async function expectResting(result: Locator, display: string) {
    // Touch can report the click before CSS transitions start in the next frame.
    await result.page().evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    if (display === 'cards') await expect(result.locator('#header-img')).toHaveCSS('scale', 'none');
    else await expect(result.locator('li')).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)');
}

for (const dev of [true, false]) {
    test.describe(`result hover in ${dev ? 'development' : 'production'}`, () => {
        let bundle: string;
        test.beforeAll(async () => { bundle = await buildTrailModalFixture(dev, source); });

        for (const display of ['cards', 'list']) {
            test(`keeps ${display} hover over the checkbox and toggles through its whole circle`, async ({ page }) => {
                await mountTrailModalFixture(page, bundle);
                if (display === 'list') await page.getByRole('button', { name: 'Change presentation' }).click();
                const result = page.locator('.trail-result').first();
                await result.locator('a').hover();
                const checkbox = result.getByRole('checkbox', { name: 'Select Ridge loop', exact: true });
                await checkbox.hover();
                if (display === 'cards') await expect(result.locator('#header-img')).toHaveCSS('scale', '1.075');
                else await expect(result.locator('li')).toHaveCSS('background-color', 'rgb(235, 240, 235)');
                await expect(checkbox).toHaveCSS('cursor', 'pointer');
                const circle = result.locator('.selection');
                await expect(circle).toHaveCSS('cursor', 'pointer');
                const inputBox = (await checkbox.boundingBox())!;
                const circleBox = (await circle.boundingBox())!;
                expect(circleBox.x + 3).toBeLessThan(inputBox.x);
                await circle.click({ position: { x: 3, y: circleBox.height / 2 } });
                await expect(checkbox).toBeChecked();
                await circle.click({ position: { x: 3, y: circleBox.height / 2 } });
                await expect(checkbox).not.toBeChecked();
                expect(await page.evaluate(() => (window as any).resultHoverLinkClicks)).toBe(0);
            });

            for (const target of ['link', 'checkbox', 'circle']) {
                test(`clears ${display} hover after deselection through the ${target} while focus stays`, async ({ page }) => {
                    await mountTrailModalFixture(page, bundle);
                    const outside = page.getByRole('button', { name: 'Change presentation' });
                    if (display === 'list') await outside.click();
                    const result = page.locator('.trail-result').first();
                    const other = page.locator('.trail-result').nth(1);
                    await result.locator('a').hover();
                    const checkbox = result.getByRole('checkbox');
                    await checkbox.check();
                    await other.locator('a').hover();
                    await other.getByRole('checkbox').check();
                    if (target === 'link') await result.locator('a').click();
                    else if (target === 'checkbox') await checkbox.uncheck();
                    else await result.locator('.selection').click({ position: { x: 3, y: 16 } });
                    await outside.hover();
                    await expect(checkbox).not.toBeChecked();
                    await expect(other.getByRole('checkbox')).toBeChecked();
                    await expect(result.locator(target === 'link' ? 'a' : 'input')).toBeFocused();
                    await expectResting(result, display);
                    if (display === 'list') await expect(other.locator('li')).toHaveCSS('background-color', 'rgb(235, 240, 235)');
                    expect(await page.evaluate(() => (window as any).resultHoverLinkClicks)).toBe(target === 'link' ? 1 : 0);
                });
            }

            test(`does not leave ${display} hover after tapping in a touch context`, async ({ browser }) => {
                const context = await browser.newContext({ hasTouch: true, isMobile: true, viewport: { width: 600, height: 1000 } });
                try {
                    const page = await context.newPage();
                    await mountTrailModalFixture(page, bundle);
                    if (display === 'list') await page.getByRole('button', { name: 'Change presentation' }).tap();
                    expect(await page.evaluate(() => matchMedia('(hover: hover)').matches)).toBe(false);
                    const result = page.locator('.trail-result').first();
                    // A touch browser can synthesize mouse events. Exercise that path
                    // explicitly even if Chromium clears its emulated hover after a tap.
                    await result.locator('a').hover();
                    await expectResting(result, display);
                    await result.locator('a').tap();
                    await expectResting(result, display);
                    const checkbox = result.getByRole('checkbox');
                    await checkbox.tap();
                    await expect(checkbox).toBeChecked();
                    await checkbox.tap();
                    await expect(checkbox).not.toBeChecked();
                    await expectResting(result, display);
                    expect(await page.evaluate(() => (window as any).resultHoverLinkClicks)).toBe(1);
                } finally {
                    await context.close();
                }
            });
        }
    });
}
