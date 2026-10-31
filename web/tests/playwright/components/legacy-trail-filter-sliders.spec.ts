import { readFile } from 'node:fs/promises';
import { expect, test } from '../fixtures';
import { buildTrailModalFixture, mountTrailModalFixture } from './trail-modal-fixture';

// Use the real legacy panel and sliders, with a bound filter as on /trails and /map.
const source = `
    <script>
        import TrailFilterPanel from '#lib/components/trail/trail_filter_panel.svelte';
        let filter = $state({
            q: '', category: [], subcategory: [], tags: [], difficulty: [],
            public: true, private: true, shared: true, near: { radius: 2000 },
            distanceMin: 0, distanceMax: 20000, distanceLimit: 20000,
            elevationGainMin: 0, elevationGainMax: 4000, elevationGainLimit: 4000,
            elevationLossMin: 0, elevationLossMax: 4000, elevationLossLimit: 4000,
            sort: 'created', sortOrder: '-',
        });
        let published = $state({});
    </script>
    <TrailFilterPanel categories={[]} bind:filter onupdate={value => published = { ...value }} />
    <output data-testid="filter">{JSON.stringify(filter)}</output>
    <output data-testid="published">{JSON.stringify(published)}</output>
`;

for (const dev of [true, false]) {
    test.describe(`legacy filter sliders in ${dev ? 'development' : 'production'}`, () => {
        let bundle: string;
        let css: string;
        test.beforeAll(async () => {
            bundle = await buildTrailModalFixture(dev, source);
            css = await readFile(new URL('../../../node_modules/nouislider/dist/nouislider.css', import.meta.url), 'utf8');
        });

        for (const [index, field, maximum] of [
            [0, 'distanceMin', 20000],
            [1, 'elevationGainMin', 4000],
            [2, 'elevationLossMin', 4000],
        ] as const) {
            test(`keeps the ${field} range fixed while dragging right and back to zero`, async ({ page }) => {
                await page.setViewportSize({ width: 900, height: 1600 });
                await mountTrailModalFixture(page, bundle);
                await page.addStyleTag({ content: css + '\n.trail-filter { width: 360px; margin: 32px; } .noUi-target { margin: 24px 0; }' });
                const sliders = page.locator('.noUi-target').filter({ has: page.locator('.noUi-handle-upper') });
                await expect(sliders).toHaveCount(3);
                const slider = sliders.nth(index);
                const lower = slider.locator('.noUi-handle-lower');
                await lower.scrollIntoViewIfNeeded();
                const track = (await slider.boundingBox())!;
                const start = (await lower.boundingBox())!;
                const read = async (id: string) => JSON.parse((await page.getByTestId(id).textContent())!)[field] as number;
                const frame = () => page.evaluate(() => new Promise(requestAnimationFrame));

                await page.mouse.move(start.x + start.width / 2, start.y + start.height / 2);
                await page.mouse.down();
                for (let step = 1; step <= 10; step++) {
                    await page.mouse.move(start.x + start.width / 2 + track.width * .25 * step / 10, start.y + start.height / 2);
                    await frame();
                }
                await page.mouse.up();
                await expect.poll(() => read('filter')).toBeGreaterThan(maximum * .2);
                await expect.poll(() => read('filter')).toBeLessThan(maximum * .3);
                await expect.poll(() => read('published')).toBeGreaterThan(maximum * .2);
                await expect.poll(() => read('published')).toBeLessThan(maximum * .3);

                const moved = (await lower.boundingBox())!;
                await page.mouse.move(moved.x + moved.width / 2, moved.y + moved.height / 2);
                await page.mouse.down();
                await page.mouse.move(track.x - 20, moved.y + moved.height / 2, { steps: 10 });
                await page.mouse.up();
                await expect.poll(() => read('filter')).toBe(0);
                await expect.poll(() => read('published')).toBe(0);
            });
        }
    });
}
