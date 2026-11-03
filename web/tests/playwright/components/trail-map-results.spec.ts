import { expect, test, type Page } from '@playwright/test';
import { mapResultStubs as stubs } from './trail-map-results-fixture';
import { buildTrailModalFixture, mountTrailModalFixture } from './trail-modal-fixture';

const fixtureSource = `
    <script>
        import TrailMapResults from '#lib/components/trail/trail_map_results.svelte';
        import { mergeStore, processMergeQueue } from '#lib/stores/trail_merge_store.svelte.ts';
        let resultDisplay = $state('cards');
        let trails = $state(Array.from({ length: 6 }, (_, i) => ({ id: 'fixture-' + i, author: 'tester', name: 'Fixture trail ' + i })));
        const initial = window.mapInitial ?? {};
        let initialBounds = $state(initial.bounds);
        let boundingBox = $state(initial.boundingBox ?? { min_lat: 0, max_lat: 0, min_lon: 0, max_lon: 0 });
        let settings = $state(initial.settings);
        let near = $state(initial.near ?? { radius: 2000 });
        let results;
        const initialCenter = 'center' in initial ? initial.center : { lat: 46, lon: 7 };
        window.mapResultsFixture = { events: [], updates: [] };
        Object.defineProperty(navigator, 'geolocation', { configurable: true, value: {
            getCurrentPosition(success, failure) {
                window.mapResultsFixture.geolocationRequests = (window.mapResultsFixture.geolocationRequests ?? 0) + 1;
                const complete = () => initial.geolocation
                    ? success({ coords: { latitude: initial.geolocation.lat, longitude: initial.geolocation.lon } })
                    : failure({ code: 1 });
                if (initial.delayGeolocation) window.mapResultsFixture.locate = complete;
                else complete();
            },
        } });
        window.mapResultsFixture.completeMerge = async () => {
            mergeStore.enqueuedMerges = [true, false].map((fails, index) => ({
                trailTarget: trails[0], trailSource: trails[index + 1], status: 'enqueued', progress: 0, settings: { delete: true },
                function: async () => { if (fails) throw new Error('Map source unavailable'); },
            }));
            await processMergeQueue();
            window.mapResultsFixture.merge();
        };
        window.mapResultsFixture.setTrails = next => trails = next;
        window.mapResultsFixture.configureDefault = config => {
            boundingBox = config.bounds;
            settings = config.settings;
            near = { radius: 2000, lat: 50, lon: 12, name: 'Keep this place filter' };
        };
        window.mapResultsFixture.display = mode => resultDisplay = mode;
        window.mapResultsFixture.focusPlace = place => results.focusPlace(place);

        function viewport(bounds, zoom) {
            window.mapResultsFixture.events.push(JSON.parse(JSON.stringify({ bounds, zoom })));
        }
    </script>
    <main>
        <button data-testid="outside">Outside</button>
        <TrailMapResults bind:this={results} {trails} onupdate={updated => window.mapResultsFixture.updates.push(updated ?? null)} mapTrails={[]} loading={false} display={resultDisplay}
            totalHits={trails.length} {initialBounds} {initialCenter} {boundingBox} {settings} {near} onviewport={viewport}>
            {#snippet toolbar()}<div data-testid="search-toolbar" style="height: 50px">Search</div>{/snippet}
            {#snippet moreResults()}{/snippet}
        </TrailMapResults>
    </main>
    <style>
        /* Match the app's Tailwind preflight without loading its entire CSS. */
        main :global(*) { box-sizing: border-box; }
        main :global(.sr-only) { border: 0; clip: rect(0, 0, 0, 0); height: 1px; margin: -1px; overflow: hidden; padding: 0; position: absolute; white-space: nowrap; width: 1px; }
    </style>
`;

const camera = (page: Page) => page.evaluate(() => (window as any).mapResultsFixture.camera());
const eventCount = (page: Page) => page.evaluate(() => (window as any).mapResultsFixture.events.length);
const lastEvent = (page: Page) => page.evaluate(() => (window as any).mapResultsFixture.events.at(-1));
const movedCamera = { center: [8, 47], zoom: 10, bearing: 12, pitch: 30 };

async function openLinkedMap(page: Page, bundle: string) {
    await mountTrailModalFixture(page, bundle);
    await expect.poll(() => camera(page)).toMatchObject({ center: [7, 46], zoom: 14 });
    await expect.poll(() => eventCount(page)).toBeGreaterThan(0);
    await expect(page.locator('#trail-map .reset-map-view')).toBeVisible();
}

for (const dev of [true, false]) {
    test.describe(`map results in ${dev ? 'development' : 'production'}`, () => {
        let bundle: string;
        test.beforeAll(async () => { bundle = await buildTrailModalFixture(dev, fixtureSource, stubs); });

        const box = { has_trails: true, min_lon: 4, max_lon: 8, min_lat: 42, max_lat: 46 };
        for (const scenario of [
            { name: 'URL bounds', config: { bounds: { northEast: { lat: 48, lng: 9 }, southWest: { lat: 46, lng: 7 } }, settings: { mapFocus: 'location', location: { lat: 49, lon: 10 } } }, expected: { center: [8, 47], zoom: 12 }, geolocation: 0 },
            { name: 'linked center', config: { near: { lat: 50, lon: 12, radius: 2000 } }, expected: { center: [7, 46], zoom: 14 }, geolocation: 0 },
            { name: 'place filter', config: { center: null, near: { lat: 50, lon: 12, radius: 2000 }, settings: { mapFocus: 'trails' }, boundingBox: box }, expected: { center: [12, 50], zoom: 12 }, geolocation: 0 },
            { name: 'trail overview', config: { center: null, settings: { mapFocus: 'trails', location: { lat: 49, lon: 10 } }, boundingBox: box }, expected: { center: [6, 44], zoom: 12 }, geolocation: 0 },
            { name: 'saved location', config: { center: null, settings: { mapFocus: 'location', location: { lat: 49, lon: 10 } }, boundingBox: box }, expected: { center: [10, 49], zoom: 12 }, geolocation: 0 },
            { name: 'browser location without a saved location', config: { center: null, settings: { mapFocus: 'location' }, boundingBox: box, geolocation: { lat: 51, lon: 13 } }, expected: { center: [13, 51], zoom: 12 }, geolocation: 1 },
            { name: 'browser location without trails', config: { center: null, settings: { mapFocus: 'trails' }, geolocation: { lat: 51, lon: 13 } }, expected: { center: [13, 51], zoom: 12 }, geolocation: 1 },
            { name: 'default camera after denied geolocation', config: { center: null, settings: { mapFocus: 'location' }, boundingBox: box }, expected: { center: [0, 0], zoom: 3 }, geolocation: 1 },
        ]) {
            test(`initializes from ${scenario.name} in priority order`, async ({ page }) => {
                await page.addInitScript(config => (window as any).mapInitial = config, scenario.config);
                await mountTrailModalFixture(page, bundle);
                await expect.poll(() => camera(page)).toMatchObject(scenario.expected);
                expect(await page.evaluate(() => (window as any).mapResultsFixture.geolocationRequests ?? 0)).toBe(scenario.geolocation);
            });
        }

        test('publishes one search area after initializing', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await page.waitForTimeout(100);
            expect(await eventCount(page)).toBe(1);
        });

        test('searches only the map area next to the results panel and below the search bar', async ({ page }) => {
            await openLinkedMap(page, bundle);
            const map = (await page.locator('#trail-map').boundingBox())!;
            const panel = (await page.locator('.results-panel').boundingBox())!;
            const toolbar = (await page.getByTestId('search-toolbar').boundingBox())!;
            // The fake map spans 0.2 degrees in each direction around its center.
            const { bounds } = await lastEvent(page);
            const west = 7 - .1 + .2 * (panel.x + panel.width + 16 - map.x) / map.width;
            const north = 46 + .1 - .2 * (toolbar.y + toolbar.height + 16 - map.y) / map.height;
            expect(bounds.southWest.lng).toBeCloseTo(west, 6);
            expect(bounds.northEast.lat).toBeCloseTo(north, 6);
            expect(bounds.northEast.lng).toBeCloseTo(7.1, 6);
            expect(bounds.southWest.lat).toBeCloseTo(45.9, 6);
            // Fits leave the covered parts out as well.
            await page.evaluate(() => (window as any).mapResultsFixture.focusPlace({ lat: 50, lon: 12, radius: 2000 }));
            const padding = await page.evaluate(() => (window as any).mapResultsFixture.fitPadding);
            expect(padding.left).toBeGreaterThan(panel.x + panel.width - map.x);
            expect(padding.top).toBeGreaterThan(toolbar.y + toolbar.height - map.y);
        });

        test('keeps the results panel visible without controls to resize it', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await expect(page.locator('.results-panel')).toBeVisible();
            await expect(page.getByRole('heading', { name: 'trail', level: 2 })).toBeVisible();
            await expect(page.locator('.result-count')).toHaveText('6');
            await expect(page.locator('.expand-results, [role="separator"]')).toHaveCount(0);
            await expect(page.locator('.map-view-toggle')).toBeHidden();
            const map = (await page.locator('#trail-map').boundingBox())!;
            const controls = (await page.locator('#trail-map .maplibregl-ctrl-top-right').boundingBox())!;
            const toolbar = (await page.getByTestId('search-toolbar').boundingBox())!;
            expect(map.width).toBe((await page.locator('.map-results').boundingBox())!.width);
            // The controls stand in a column next to the shortened search bar.
            expect(controls.x).toBeGreaterThanOrEqual(toolbar.x + toolbar.width);
            expect(controls.y).toBeLessThan(toolbar.y);
        });

        test('collapses the results panel to its header, searches the freed area and remembers the choice', async ({ page }) => {
            await openLinkedMap(page, bundle);
            const before = await lastEvent(page);
            const toggle = page.getByRole('button', { name: 'trail-map-collapse-list' });
            await expect(toggle).toHaveAttribute('aria-expanded', 'true');
            await toggle.click();
            const expand = page.getByRole('button', { name: 'trail-map-expand-list' });
            await expect(expand).toHaveAttribute('aria-expanded', 'false');
            await expect(page.getByRole('heading', { name: 'trail', level: 2 })).toBeVisible();
            await expect(page.locator('.result-count')).toHaveText('6');
            await expect(page.locator('.trail-results')).toBeHidden();
            const panel = (await page.locator('.results-panel').boundingBox())!;
            expect(panel.height).toBeLessThan(80);
            // The area beside the folded list now counts as visible.
            await expect.poll(() => lastEvent(page).then(event => event.bounds.southWest.lng)).toBeCloseTo(6.9, 6);
            expect(before.bounds.southWest.lng).toBeGreaterThan(6.9);
            await mountTrailModalFixture(page, bundle);
            await expect(page.getByRole('button', { name: 'trail-map-expand-list' })).toBeVisible();
            await expect(page.locator('.trail-results')).toBeHidden();
            await expect.poll(() => lastEvent(page).then(event => event?.bounds.southWest.lng)).toBeCloseTo(6.9, 6);
            await page.getByRole('button', { name: 'trail-map-expand-list' }).click();
            await expect(page.getByTestId('map-card')).toHaveCount(6);
            await expect.poll(() => lastEvent(page).then(event => event.bounds.southWest.lng)).toBeCloseTo(before.bounds.southWest.lng, 6);
        });

        test('shows a full-screen map on phones and toggles the list over it', async ({ page }) => {
            await page.setViewportSize({ width: 390, height: 800 });
            await openLinkedMap(page, bundle);
            const map = (await page.locator('#trail-map').boundingBox())!;
            expect(map.width).toBe((await page.locator('.map-results').boundingBox())!.width);
            await expect(page.locator('.results-panel')).toBeHidden();
            // Only the search bar covers part of the map, so the area spans its whole width.
            const { bounds } = await lastEvent(page);
            expect(bounds.southWest.lng).toBeCloseTo(6.9, 6);
            expect(bounds.northEast.lng).toBeCloseTo(7.1, 6);
            expect(bounds.northEast.lat).toBeLessThan(46.1);
            const toggle = page.locator('.map-view-toggle');
            await expect(page.locator('.collapse-results')).toBeHidden();
            await expect(toggle).toHaveText('list');
            await expect(toggle).toHaveAttribute('aria-expanded', 'false');
            const button = (await toggle.boundingBox())!;
            expect(Math.abs(button.x + button.width / 2 - (map.x + map.width / 2))).toBeLessThan(1);
            expect(map.y + map.height - (button.y + button.height)).toBeLessThan(40);
            const before = await eventCount(page);
            await toggle.click();
            await expect(page.locator('.results-panel')).toBeVisible();
            await expect(page.getByTestId('map-card')).toHaveCount(6);
            await expect(toggle).toHaveText('map');
            await expect(toggle).toHaveAttribute('aria-expanded', 'true');
            const toolbar = (await page.getByTestId('search-toolbar').boundingBox())!;
            const header = (await page.locator('.result-header').boundingBox())!;
            expect(header.y).toBeGreaterThanOrEqual(toolbar.y + toolbar.height);
            await expect(page.getByTestId('search-toolbar')).toBeVisible();
            await toggle.click();
            await expect(page.locator('.results-panel')).toBeHidden();
            expect(await eventCount(page)).toBe(before);
            expect(await camera(page)).toMatchObject({ center: [7, 46], zoom: 14 });
        });

        test('returns to the map on phones when a place is chosen from the list', async ({ page }) => {
            await page.setViewportSize({ width: 390, height: 800 });
            await openLinkedMap(page, bundle);
            await page.locator('.map-view-toggle').click();
            await expect(page.locator('.results-panel')).toBeVisible();
            await page.evaluate(() => (window as any).mapResultsFixture.focusPlace({ lat: 50, lon: 12, radius: 2000 }));
            await expect(page.locator('.results-panel')).toBeHidden();
            expect((await page.evaluate(() => (window as any).mapResultsFixture.fitPadding)).left).toBe(32);
        });

        test('keeps document-based height after scrolling and while the body is scroll locked', async ({ page }) => {
            await openLinkedMap(page, bundle);
            const height = await page.locator('.map-results').evaluate(element => element.getBoundingClientRect().height);
            await page.evaluate(() => { const spacer = document.createElement('div'); spacer.style.height = '1800px'; document.body.append(spacer); window.scrollTo(0, 300); });
            await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(300);
            await page.evaluate(() => window.dispatchEvent(new Event('resize')));
            await expect.poll(() => page.locator('.map-results').evaluate(element => element.getBoundingClientRect().height)).toBe(height);
            await page.evaluate(() => { document.body.style.position = 'fixed'; document.body.style.top = '-300px'; window.dispatchEvent(new Event('resize')); });
            await expect.poll(() => page.locator('.map-results').evaluate(element => element.getBoundingClientRect().height)).toBe(height);
        });

        for (const display of ['cards', 'list']) {
            test(`gives map ${display} links independent accessible names`, async ({ page }) => {
                await openLinkedMap(page, bundle);
                await page.evaluate(display => (window as any).mapResultsFixture.display(display), display);
                const link = page.locator('.result-items .trail-result > a').first();
                await link.focus();
                await expect(link).toHaveAccessibleName(/Fixture trail 0.*Trail details/);
                await expect(link).not.toHaveAccessibleName(/\bSelect\b|auswählen/);
                await expect(link.getByRole('checkbox')).toHaveCount(0);
                await page.keyboard.press('Tab');
                const checkbox = page.getByRole('checkbox', { name: 'Select Fixture trail 0', exact: true });
                await expect(checkbox).toBeFocused();
                await page.keyboard.press('Space');
                await expect(checkbox).toBeChecked();
                await checkbox.hover();
                await expect(link.locator('[data-hovered]')).toHaveAttribute('data-hovered', 'true');
                await page.getByTestId('outside').hover();
                await expect(link.locator('[data-hovered]')).toHaveAttribute('data-hovered', 'false');
            });
        }

        test('ignores delayed geolocation after the user moves the map', async ({ page }) => {
            await page.addInitScript(() => (window as any).mapInitial = { center: null, delayGeolocation: true, geolocation: { lat: 51, lon: 13 } });
            await mountTrailModalFixture(page, bundle);
            await page.evaluate(() => (window as any).mapResultsFixture.pan());
            await page.evaluate(() => (window as any).mapResultsFixture.locate());
            expect(await camera(page)).toEqual(movedCamera);
        });

        test('ignores delayed browser location during a user gesture', async ({ page }) => {
            await page.addInitScript(() => (window as any).mapInitial = { center: null, delayGeolocation: true, geolocation: { lat: 51, lon: 13 } });
            await mountTrailModalFixture(page, bundle);
            await expect.poll(() => eventCount(page)).toBeGreaterThan(0);
            await page.evaluate(() => { (window as any).mapResultsFixture.moving = true; (window as any).mapResultsFixture.locate(); });
            expect(await camera(page)).toMatchObject({ center: [0, 0], zoom: 3 });
        });

        test('accepts delayed browser location and searches there once', async ({ page }) => {
            await page.addInitScript(() => (window as any).mapInitial = { center: null, delayGeolocation: true, geolocation: { lat: 51, lon: 13 } });
            await mountTrailModalFixture(page, bundle);
            await expect.poll(() => eventCount(page)).toBe(1);
            await page.evaluate(() => (window as any).mapResultsFixture.locate());
            expect(await camera(page)).toMatchObject({ center: [13, 51], zoom: 12 });
            expect(await eventCount(page)).toBe(2);
            expect((await lastEvent(page)).zoom).toBe(12);
        });

        for (const scenario of [
            { name: 'all trails', settings: { mapFocus: 'trails' },
                bounds: { has_trails: true, min_lon: 4, max_lon: 8, min_lat: 42, max_lat: 46 },
                expected: { center: [6, 44], zoom: 12, bearing: 0, pitch: 0 } },
            { name: 'the saved location', settings: { mapFocus: 'location', location: { lat: 48, lon: 11 } },
                bounds: { has_trails: true, min_lon: 4, max_lon: 8, min_lat: 42, max_lat: 46 },
                expected: { center: [11, 48], zoom: 12, bearing: 0, pitch: 0 } },
            { name: 'a single trail at zero coordinates', settings: { mapFocus: 'trails' },
                bounds: { has_trails: true, min_lon: 0, max_lon: 0, min_lat: 0, max_lat: 0 },
                expected: { center: [0, 0], zoom: 12, bearing: 0, pitch: 0 } },
            { name: 'an empty index', settings: { mapFocus: 'trails' },
                bounds: { has_trails: false, min_lon: 0, max_lon: 0, min_lat: 0, max_lat: 0 },
                expected: { center: [0, 0], zoom: 3, bearing: 0, pitch: 0 } },
        ]) {
            test(`restores the default view for ${scenario.name}, ignoring linked and moved bounds and place filters`, async ({ page }) => {
                await openLinkedMap(page, bundle);
                await page.evaluate(config => (window as any).mapResultsFixture.configureDefault(config), scenario);
                await page.evaluate(() => (window as any).mapResultsFixture.pan());
                const before = await eventCount(page);
                const reset = page.locator('#trail-map .reset-map-view');
                const savedLocation = scenario.settings.mapFocus === 'location';
                await expect(reset).toHaveAttribute('aria-label', savedLocation ? 'trail-map-reset-location' : 'trail-map-reset-view');
                await expect(reset.locator('i')).toHaveClass(savedLocation ? /fa-location-dot/ : /fa-arrows-to-circle/);
                await reset.press('Enter');
                await expect.poll(() => camera(page)).toEqual(scenario.expected);
                expect(await eventCount(page)).toBe(before + 1);
                const buttons = page.locator('#trail-map .maplibregl-ctrl-top-right button');
                await expect(buttons.last()).toHaveClass(/reset-map-view/);
            });
        }

        test('changes result cards to list items while retaining the map and camera', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await expect(page.getByTestId('map-card').first()).toBeVisible();
            const before = await eventCount(page);
            await page.getByTestId('fake-map').evaluate(element => (window as any).retainedMap = element);
            await page.evaluate(() => (window as any).mapResultsFixture.display('list'));
            await expect(page.getByTestId('map-list-item').first()).toBeVisible();
            await expect(page.getByTestId('map-card')).toHaveCount(0);
            expect(await page.getByTestId('fake-map').evaluate(element => element === (window as any).retainedMap)).toBe(true);
            expect(await camera(page)).toMatchObject({ center: [7, 46], zoom: 14 });
            expect(await eventCount(page)).toBe(before);
        });

        test('selects several trails in cards and list without navigating or changing the map', async ({ page }) => {
            await openLinkedMap(page, bundle);
            const before = await eventCount(page);
            const url = page.url();
            await page.getByTestId('map-card').first().hover();
            await page.getByRole('checkbox', { name: 'Select Fixture trail 0', exact: true }).check();
            await page.getByTestId('map-card').nth(1).click();
            await expect(page.locator('.selection-count')).toHaveText('2 selected');
            expect(page.url()).toBe(url);
            for (const display of ['list', 'cards']) {
                await page.evaluate(display => (window as any).mapResultsFixture.display(display), display);
                await expect(page.locator('.trail-results input:checked')).toHaveCount(2);
            }
            await page.getByRole('checkbox', { name: 'trail-results-select-loaded' }).check();
            await expect(page.locator('.selection-count')).toHaveText('6 selected');
            await page.getByRole('button', { name: 'trail-results-clear-selection' }).click();
            await expect(page.locator('.selection-toolbar')).toHaveCount(0);
            await expect(page.locator('.result-items .trail-result > a').first()).toBeFocused();
            expect(await eventCount(page)).toBe(before);
        });

        test('keeps selection for appended and refreshed records and drops removed results', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await page.getByTestId('map-card').first().hover();
            await page.getByRole('checkbox', { name: 'Select Fixture trail 0', exact: true }).check();
            const replacement = Array.from({ length: 8 }, (_, i) => ({ id: 'fixture-' + i, author: 'tester', name: 'Updated trail ' + i }));
            await page.evaluate(trails => (window as any).mapResultsFixture.setTrails(trails), replacement);
            await expect(page.locator('.selection-count')).toHaveText('1 selected');
            await expect(page.getByRole('checkbox', { name: 'Select Updated trail 0', exact: true })).toBeChecked();
            await page.evaluate(trails => (window as any).mapResultsFixture.setTrails(trails), replacement.slice(1));
            await expect(page.locator('.selection-toolbar')).toHaveCount(0);
            await page.evaluate(trails => (window as any).mapResultsFixture.setTrails(trails), replacement);
            await page.getByTestId('map-card').first().hover();
            await expect(page.getByRole('checkbox', { name: 'Select Updated trail 0', exact: true })).not.toBeChecked();
        });

        test('forwards edits of selected trails and clears the selection', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await page.getByTestId('map-card').first().hover();
            await page.getByRole('checkbox', { name: 'Select Fixture trail 0', exact: true }).check();
            await expect(page.getByRole('button', { name: 'trail-results-selection-actions' })).toBeVisible();
            expect(await page.evaluate(() => (window as any).mapResultsFixture.selected())).toEqual(['fixture-0']);
            const updated = [{ id: 'fixture-0', difficulty: 'difficult' }];
            await page.evaluate(updated => (window as any).mapResultsFixture.update(updated), updated);
            await expect(page.locator('.selection-toolbar')).toHaveCount(0);
            expect(await page.evaluate(() => (window as any).mapResultsFixture.updates)).toEqual([updated]);
        });

        test('keeps the selection dialog open across window sizes', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await page.getByTestId('map-card').first().hover();
            await page.getByRole('checkbox', { name: 'Select Fixture trail 0', exact: true }).check();
            await page.evaluate(() => (window as any).mapResultsFixture.openActionDialog());
            const dialog = page.locator('dialog[open]');
            for (const size of [{ width: 600, height: 900 }, { width: 1280, height: 720 }]) {
                await page.setViewportSize(size);
                await expect(dialog).toBeVisible();
                await expect(dialog.getByRole('textbox')).toBeFocused();
                await expect(dialog.getByRole('textbox')).toHaveValue('Keep this value');
            }
        });

        test('keeps the selection checkbox focused when tabbing from a trail link and selects with Space', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await page.locator('.result-items .trail-result > a').first().focus();
            await page.keyboard.press('Tab');
            const checkbox = page.getByRole('checkbox', { name: 'Select Fixture trail 0', exact: true });
            await expect(checkbox).toBeFocused();
            await page.getByTestId('map-card').nth(1).hover();
            await expect(checkbox).toBeFocused();
            await page.keyboard.press('Space');
            await expect(page.locator('.selection-count')).toHaveText('1 selected');
        });

        test('keeps the real map merge panel and retry after clearing the selection', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await page.locator('.result-items .trail-result > a').first().focus();
            await page.getByRole('checkbox', { name: 'Select Fixture trail 0', exact: true }).check();
            await expect(page.locator('.selection-count')).toHaveText('1 selected');
            expect(await page.evaluate(() => (window as any).mapResultsFixture.showMergeDialog)).toBe(false);
            await page.evaluate(() => (window as any).mapResultsFixture.completeMerge());
            await expect(page.locator('.selection-toolbar')).toHaveCount(0);
            await expect(page.getByRole('button', { name: 'trail-merge-retry', exact: true })).toBeVisible();
            await expect(page.getByText('Map source unavailable', { exact: true })).toBeVisible();
        });
    });
}
