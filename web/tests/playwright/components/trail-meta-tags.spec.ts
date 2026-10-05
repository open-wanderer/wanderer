import { expect, test } from '@playwright/test';
import { fileURLToPath } from 'node:url';
import { buildTrailModalFixture, mountTrailModalFixture } from './trail-modal-fixture';

// Keep both real pages and MetaTags; replace unrelated map/panel presentation.
const sourceStubs = {
    [fileURLToPath(new URL('../../../src/lib/components/trail/trail_info_panel.svelte', import.meta.url))]: `
        <script>
            let { initTrail } = $props();
            let clicks = $state(0);
        </script>
        <button data-testid="panel-control" onclick={() => clicks++}>Open {initTrail.name}: {clicks}</button>
    `,
    [fileURLToPath(new URL('../../../src/lib/components/trail/map_with_elevation_maplibre.svelte', import.meta.url))]: `
        <script>let { markers = $bindable([]) } = $props();</script>
        <div data-testid="map-presentation"></div>
    `,
};

for (const dev of [true, false]) {
    for (const route of ['trail/view', 'map/trail']) {
        test.describe(`${route} metadata in ${dev ? 'development' : 'production'}`, () => {
            let bundle: string;
            test.beforeAll(async () => {
                bundle = await buildTrailModalFixture(dev, `
                    <script>
                        import Page from '$lib/../routes/${route}/[handle]/[id]/+page.svelte';
                        const tags = Object.freeze([
                            Object.freeze({ id: 'firsttag0000001', name: 'Berg Tour' }),
                            Object.freeze({ id: 'secondtag000001', name: 'See Ufer' }),
                            Object.freeze({ id: 'thirdtag0000001', name: 'Berg Tour' }),
                            Object.freeze({ id: 'fourthtag000001', name: '<b>&amp;</b>' }),
                        ]);
                        let data = $state({ trail: {
                            id: 'testtrail000001', name: 'Trail', photos: [], expand: { tags },
                        } });
                        window.inspectSourceTags = () => tags;
                        window.removeTagExpansion = () => { data.trail.expand = {}; };
                        window.setEmptyTags = () => { data.trail.expand = { tags: [] }; };
                    </script>
                    <Page {data} />
                `, sourceStubs);
            });

            test('deduplicates article names without dropping tag IDs or breaking interaction', async ({ page }) => {
                const errors: string[] = [];
                page.on('pageerror', error => errors.push(error.message));
                try {
                    await mountTrailModalFixture(page, bundle);
                } catch (error) {
                    if (errors.length) throw new Error(errors.join('\n'));
                    throw error;
                }
                const meta = page.locator('meta[property="article:tag"]');
                await expect(meta).toHaveCount(3);
                expect(await meta.evaluateAll(nodes => nodes.map(node => node.getAttribute('content')))).toEqual([
                    'Berg Tour', 'See Ufer', '<b>&amp;</b>',
                ]);
                await page.getByTestId('panel-control').click();
                await expect(page.getByTestId('panel-control')).toHaveText('Open Trail: 1');
                expect(await page.evaluate(() => (window as any).inspectSourceTags())).toEqual([
                    { id: 'firsttag0000001', name: 'Berg Tour' },
                    { id: 'secondtag000001', name: 'See Ufer' },
                    { id: 'thirdtag0000001', name: 'Berg Tour' },
                    { id: 'fourthtag000001', name: '<b>&amp;</b>' },
                ]);
                await page.evaluate(() => (window as any).removeTagExpansion());
                await expect(meta).toHaveCount(0);
                await page.evaluate(() => (window as any).setEmptyTags());
                await expect(meta).toHaveCount(0);
                expect(errors).toEqual([]);
            });
        });
    }
}
