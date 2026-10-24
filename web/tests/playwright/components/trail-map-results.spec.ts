import { expect, test, type Page } from '@playwright/test';
import { fileURLToPath } from 'node:url';
import { buildTrailModalFixture, mountTrailModalFixture } from './trail-modal-fixture';

const componentPath = (name: string) => fileURLToPath(new URL(`../../../src/lib/components/${name}.svelte`, import.meta.url));

// Keep the real map-result component. Only the rendering children and the map
// engine are replaced; camera changes still drive its real viewport callbacks.
const stubs = {
    [componentPath('trail/trail_card')]: '<div>Trail</div>',
    [componentPath('base/skeleton_card')]: '<div>Loading</div>',
    [componentPath('empty_states/empty_state_search')]: '<div>No results</div>',
    [componentPath('trail/map_with_elevation_maplibre')]: `
        <script>
            import { onMount, tick } from 'svelte';
            let { map = $bindable(), oninit, onmoveend } = $props();
            class FakeMap {
                camera = { center: [0, 0], zoom: 3, bearing: 0, pitch: 0 };
                jumpTo(change) {
                    this.camera = { ...this.camera, ...change };
                    onmoveend();
                }
                resize() {}
                getCenter() { return { toArray: () => [...this.camera.center] }; }
                getZoom() { return this.camera.zoom; }
                getBearing() { return this.camera.bearing; }
                getPitch() { return this.camera.pitch; }
                getBounds() {
                    const [lng, lat] = this.camera.center;
                    return { getNorth: () => lat + .1, getSouth: () => lat - .1,
                        getEast: () => lng + .1, getWest: () => lng - .1 };
                }
                fitBounds(bounds) {
                    const [[west, south], [east, north]] = Array.isArray(bounds) ? bounds : bounds.toArray();
                    this.jumpTo({ center: [(west + east) / 2, (south + north) / 2], zoom: 12 });
                }
            }
            onMount(() => {
                map = new FakeMap();
                window.mapResultsFixture.camera = () => JSON.parse(JSON.stringify(map.camera));
                window.mapResultsFixture.pan = () => map.jumpTo({ center: [8, 47], zoom: 10, bearing: 12, pitch: 30 });
                // The real map also reports initialization asynchronously.
                void tick().then(() => oninit());
            });
            export function highlightCluster() {}
            export function unHighlightCluster() {}
        </script>
        <div data-testid="fake-map" style="height: 100%; min-height: 1px"></div>
    `,
};

const fixtureSource = `
    <script>
        import TrailMapResults from '$lib/components/trail/trail_map_results.svelte';
        let view = $state('map');
        let initialBounds = $state();
        let results;
        const initialCenter = { lat: 46, lon: 7 };
        window.mapResultsFixture = { events: [] };
        window.mapResultsFixture.setBounds = bounds => initialBounds = bounds;
        window.mapResultsFixture.focusPlace = place => results.focusPlace(place);
        function viewport(bounds, zoom, entering) {
            initialBounds = bounds;
            window.mapResultsFixture.events.push(JSON.parse(JSON.stringify({ bounds, zoom, entering })));
        }
    </script>
    <main>
        <nav>
            {#each ['map', 'cards', 'list', 'table'] as option}
                <button data-testid={'view-' + option} onclick={() => view = option}>{option}</button>
            {/each}
        </nav>
        <div hidden={view !== 'map'}>
            <TrailMapResults bind:this={results} trails={[]} mapTrails={[]} loading={false}
                active={view === 'map'} {initialBounds} {initialCenter}
                boundingBox={{ min_lat: 0, max_lat: 0, min_lon: 0, max_lon: 0 }}
                near={{ radius: 2000 }} onviewport={viewport}>
                {#snippet moreResults()}{/snippet}
            </TrailMapResults>
        </div>
    </main>
`;

const camera = (page: Page) => page.evaluate(() => (window as any).mapResultsFixture.camera());
const eventCount = (page: Page) => page.evaluate(() => (window as any).mapResultsFixture.events.length);
const movedCamera = { center: [8, 47], zoom: 10, bearing: 12, pitch: 30 };

async function openLinkedMap(page: Page, bundle: string) {
    await mountTrailModalFixture(page, bundle);
    await expect.poll(() => camera(page)).toMatchObject({ center: [7, 46], zoom: 14 });
    await expect.poll(() => eventCount(page)).toBeGreaterThan(0);
}

async function returnToMap(page: Page) {
    const before = await eventCount(page);
    await page.getByTestId('view-map').click();
    await expect.poll(() => eventCount(page)).toBeGreaterThan(before);
}

for (const dev of [true, false]) {
    test.describe(`map camera in ${dev ? 'development' : 'production'}`, () => {
        let bundle: string;
        test.beforeAll(async () => { bundle = await buildTrailModalFixture(dev, fixtureSource, stubs); });

        for (const view of ['cards', 'list', 'table']) {
            test(`uses the linked center once and retains a moved camera after ${view} view`, async ({ page }) => {
                await openLinkedMap(page, bundle);
                await page.evaluate(() => (window as any).mapResultsFixture.pan());
                expect(await camera(page)).toEqual(movedCamera);
                await page.getByTestId('view-' + view).click();
                await expect(page.getByTestId('fake-map')).toBeHidden();
                await returnToMap(page);
                expect(await camera(page)).toEqual(movedCamera);
                await expect(page.getByTestId('fake-map')).toBeVisible();
            });
        }

        test('retains the mobile results view and camera after leaving and returning', async ({ page }) => {
            await page.setViewportSize({ width: 640, height: 800 });
            await openLinkedMap(page, bundle);
            await page.evaluate(() => (window as any).mapResultsFixture.pan());
            await page.getByRole('button', { name: 'trail-map-show-results', exact: true }).click();
            await expect(page.getByTestId('fake-map')).toBeHidden();
            await page.getByTestId('view-list').click();
            await returnToMap(page);
            await expect(page.getByRole('button', { name: 'trail-map-show-map', exact: true })).toBeVisible();
            await expect(page.getByTestId('fake-map')).toBeHidden();
            expect(await camera(page)).toEqual(movedCamera);
        });

        test('restores explicitly changed bounds instead of the linked center', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await page.evaluate(() => (window as any).mapResultsFixture.pan());
            await page.getByTestId('view-list').click();
            await page.evaluate(() => (window as any).mapResultsFixture.setBounds({
                northEast: { lat: 50, lng: 12 }, southWest: { lat: 48, lng: 10 },
            }));
            await returnToMap(page);
            expect(await camera(page)).toMatchObject({ center: [11, 49], zoom: 12 });
        });

        test('keeps explicit place focus when returning to a map opened from a linked center', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await page.evaluate(() => (window as any).mapResultsFixture.focusPlace({ lat: 48, lon: 9, radius: 2000 }));
            const focused = await camera(page);
            expect(focused.center[0]).toBeCloseTo(9);
            expect(focused.center[1]).toBeCloseTo(48);
            await page.getByTestId('view-cards').click();
            await returnToMap(page);
            expect(await camera(page)).toEqual(focused);
        });
    });
}
