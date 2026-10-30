import { discardHiddenScrollPosition } from './map-scroll-fixture';
import { expect, test, type Page } from '@playwright/test';
import { mapResultStubs as stubs } from './trail-map-results-fixture';
import { buildTrailModalFixture, mountTrailModalFixture } from './trail-modal-fixture';

const fixtureSource = `
    <script>
        import TrailMapResults from '#lib/components/trail/trail_map_results.svelte';
        import { mergeStore, processMergeQueue } from '#lib/stores/trail_merge_store.svelte.ts';
        let view = $state('map');
        let resultDisplay = $state('cards');
        let trails = $state(Array.from({ length: 6 }, (_, i) => ({ id: 'fixture-' + i, author: 'tester', name: 'Fixture trail ' + i })));
        const initial = window.mapInitial ?? {};
        let initialBounds = $state(initial.bounds);
        let boundingBox = $state(initial.boundingBox ?? { min_lat: 0, max_lat: 0, min_lon: 0, max_lon: 0 });
        let settings = $state(initial.settings);
        let near = $state(initial.near ?? { radius: 2000 });
        let results;
        let headerHeight = $state(0);
        const initialCenter = 'center' in initial ? initial.center : { lat: 46, lon: 7 };
        window.mapResultsFixture = { events: [], updates: [], zooms: [], renders: [], header: value => headerHeight = value };
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
        <div data-search-root style:height={headerHeight + 'px'}></div>
        <div hidden={view !== 'map'}>
            <TrailMapResults bind:this={results} {trails} onupdate={updated => window.mapResultsFixture.updates.push(updated ?? null)} mapTrails={[]} loading={false} display={resultDisplay}
                active={view === 'map'} {initialBounds} {initialCenter}
                {boundingBox} {settings} {near} onviewport={viewport} onzoom={(zoom, bounds) => { window.mapResultsFixture.zooms.push(zoom); window.mapResultsFixture.renders.push({ zoom, bounds }); }}>
                {#snippet moreResults()}{/snippet}
            </TrailMapResults>
        </div>
    </main>
    <style>
        /* Match the app's Tailwind preflight without loading its entire CSS. */
        main :global(*) { box-sizing: border-box; }
        main :global(.sr-only) { border: 0; clip: rect(0, 0, 0, 0); height: 1px; margin: -1px; overflow: hidden; padding: 0; position: absolute; white-space: nowrap; width: 1px; }


    </style>
`;

const camera = (page: Page) => page.evaluate(() => (window as any).mapResultsFixture.camera());
const eventCount = (page: Page) => page.evaluate(() => (window as any).mapResultsFixture.events.length);
const movedCamera = { center: [8, 47], zoom: 10, bearing: 12, pitch: 30 };

async function toggleExpandedResults(page: Page) {
    const before = await page.evaluate(() => (window as any).mapResultsFixture.zooms.length);
    await page.locator('.expand-results').click();
    await expect.poll(() => page.evaluate(() => (window as any).mapResultsFixture.zooms.length)).toBeGreaterThan(before);
}

async function openLinkedMap(page: Page, bundle: string) {
    await mountTrailModalFixture(page, bundle);
    await expect.poll(() => camera(page)).toMatchObject({ center: [7, 46], zoom: 14 });
    await expect.poll(() => eventCount(page)).toBeGreaterThan(0);
    await expect(page.locator('#trail-map .reset-map-view')).toBeVisible();
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

        for (const available of [true, false]) for (const control of ['collapse','mobile']) test(`${control} restores the visible list position after hiding${available ? '' : ' without checkVisibility'}`, async ({page}) => {
            if (!available) await page.addInitScript(() => { delete (Element.prototype as any).checkVisibility; });
            const errors: string[] = [];
            page.on('pageerror', error => errors.push(error.message));
            await openLinkedMap(page,bundle);
            if(control==='mobile') {
                await page.setViewportSize({width:600,height:900});
                await page.locator('.map-view-toggle').click();
            }
            await page.evaluate(() => (window as any).mapResultsFixture.setTrails(Array.from({length:40},(_,i)=>({id:'scroll-'+i,author:'tester',name:'Trail '+i}))));
            await page.addStyleTag({content:'.trail-result { min-height:180px }'});
            const list=page.locator('.trail-results');
            await expect(list.locator('.trail-result')).toHaveCount(40);
            await list.evaluate(node=>node.scrollTop=1440);
            const before=await list.evaluate(node=>node.scrollTop);
            expect(before).toBeGreaterThan(1000);
            await discardHiddenScrollPosition(page);
            const toggle=page.locator(control==='mobile'?'.map-view-toggle':'.collapse-results');
            await toggle.click();
            await expect(list).not.toBeVisible();
            await toggle.click();
            await expect(list).toBeVisible();
            await expect.poll(async()=>Math.abs(await list.evaluate(node=>node.scrollTop)-before)).toBeLessThanOrEqual(2);
            expect(errors).toEqual([]);
        });

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

        for (const pitch of [0, 30]) {
            test(`retains a 20 degree camera after two mobile list/map cycles at pitch ${pitch}`, async ({ page }) => {
                await page.setViewportSize({ width: 390, height: 800 });
                await page.addInitScript(() => (window as any).mapInitial = { cameraGeometry: true });
                await openLinkedMap(page, bundle);
                const expected = { center: [8, 47], zoom: 10, bearing: 20, pitch };
                await page.evaluate(change => (window as any).mapResultsFixture.setCamera(change), expected);
                for (let i = 0; i < 2; i++) {
                    await page.locator('.map-view-toggle').click();
                    await expect(page.getByTestId('fake-map')).toBeHidden();
                    await page.locator('.map-view-toggle').click();
                    await expect(page.getByTestId('fake-map')).toBeVisible();
                    await expect.poll(() => camera(page)).toEqual(expected);
                }
            });
        }

        test('restores rotated layout cameras by pixel ratio without repeated bounds fitting', async ({ page }) => {
            await page.addInitScript(() => (window as any).mapInitial = { cameraGeometry: true });
            await openLinkedMap(page, bundle);
            await page.evaluate(() => (window as any).mapResultsFixture.setCamera({ center: [8, 47], zoom: 10, bearing: 20, pitch: 30 }));
            const original = (await page.locator('#trail-map').boundingBox())!;
            for (let i = 0; i < 2; i++) {
                await toggleExpandedResults(page);
                const small = (await page.locator('#trail-map').boundingBox())!;
                expect((await camera(page)).zoom).toBeCloseTo(10 + Math.log2(Math.min(small.width / original.width, small.height / original.height)), 1);
                await toggleExpandedResults(page);
                await expect.poll(() => camera(page).then(value => value.zoom)).toBeCloseTo(10, 5);
            }
        });

        for (const bearing of [0, 20]) {
            test(`moves an overview at ${bearing} degrees without a bounds fit or camera change`, async ({ page }) => {
                await page.addInitScript(() => (window as any).mapInitial = { cameraGeometry: true });
                await openLinkedMap(page, bundle);
                await toggleExpandedResults(page);
                await page.evaluate(bearing => (window as any).mapResultsFixture.setCamera({ center: [8, 47], zoom: 10, bearing, pitch: bearing ? 30 : 0 }), bearing);
                const before = await page.evaluate(() => (window as any).mapResultsFixture.fitCount ?? 0);
                const handle = (await page.locator('.overlay-move').boundingBox())!;
                await page.mouse.move(handle.x + 16, handle.y + 16);
                await page.mouse.down();
                await page.mouse.move(handle.x - 40, handle.y - 30);
                await page.mouse.up();
                await expect(page.locator('.map-results')).toHaveAttribute('data-layout-ready', 'true');
                expect(await page.evaluate(() => (window as any).mapResultsFixture.fitCount ?? 0)).toBe(before);
                expect(await camera(page)).toEqual({ center: [8, 47], zoom: 10, bearing, pitch: bearing ? 30 : 0 });
            });
        }

        for (const expanded of [false, true]) {
            test(`holds list scope and selection for search header and visual viewport height changes (expanded=${expanded})`, async ({ page }) => {
                await page.addInitScript(() => (window as any).mapInitial = { cameraGeometry: true });
                await openLinkedMap(page, bundle);
                if (expanded) await toggleExpandedResults(page);
                await page.getByTestId('map-card').first().hover();
                await page.getByRole('checkbox', { name: 'Select Fixture trail 0', exact: true }).check();
                const before = await eventCount(page);
                let renders = await page.evaluate(() => (window as any).mapResultsFixture.renders.length);
                await page.evaluate(() => (window as any).mapResultsFixture.header(200));
                await expect.poll(() => page.evaluate(() => (window as any).mapResultsFixture.renders.length)).toBeGreaterThan(renders);
                expect(await eventCount(page)).toBe(before);
                const scopes = await page.evaluate(() => ({
                    list: (window as any).mapResultsFixture.events.at(-1).bounds,
                    map: (window as any).mapResultsFixture.renders.at(-1).bounds,
                }));
                expect(scopes.map).not.toEqual(scopes.list);
                await expect(page.locator('.selection-count')).toHaveText('1 selected');
                if (expanded) {
                    renders = await page.evaluate(() => (window as any).mapResultsFixture.renders.length);
                    await page.evaluate(() => (window as any).mapResultsFixture.header(0));
                    await expect.poll(() => page.evaluate(() => (window as any).mapResultsFixture.renders.length)).toBeGreaterThan(renders);
                }
                renders = await page.evaluate(() => (window as any).mapResultsFixture.renders.length);
                await page.evaluate(() => {
                    Object.defineProperty(window.visualViewport, 'height', { configurable: true, value: innerHeight - 260 });
                    window.visualViewport!.dispatchEvent(new Event('resize'));
                });
                await expect.poll(() => page.evaluate(() => (window as any).mapResultsFixture.renders.length)).toBeGreaterThan(renders);
                expect(await eventCount(page)).toBe(before);
                await page.setViewportSize({ width: 1400, height: 800 });
                await expect.poll(() => eventCount(page)).toBeGreaterThan(before);
            });
        }

        for (const display of ['cards', 'list']) {
            test(`gives map ${display} links independent accessible names`, async ({ page }) => {
                await openLinkedMap(page, bundle);
                await page.evaluate(display => (window as any).mapResultsFixture.display(display), display);
                const link = page.locator('.result-items .trail-result > a').first();
                await link.focus();
                await expect(link).toHaveAccessibleName(/Fixture trail 0.*Trail details/);
                await expect(link).not.toHaveAccessibleName(/\bSelect\b|auswählen/);
                await expect(link).not.toHaveAttribute('aria-label');
                await expect(link).not.toHaveAttribute('aria-labelledby');
                await expect(link.getByRole('heading', { name: 'Fixture trail 0', level: 4, exact: true })).toBeVisible();
                await expect(link.getByRole('checkbox')).toHaveCount(0);
                await page.keyboard.press('Tab');
                const checkbox = page.getByRole('checkbox', { name: 'Select Fixture trail 0', exact: true });
                await expect(checkbox).toBeFocused();
                await page.keyboard.press('Space');
                await expect(checkbox).toBeChecked();
                await checkbox.hover();
                await expect(link.locator('[data-hovered]')).toHaveAttribute('data-hovered', 'true');
                await page.getByTestId('view-map').hover();
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

        test('reports the initial area after mobile results were opened before map initialization', async ({ page }) => {
            await page.setViewportSize({ width: 390, height: 600 });
            await page.addInitScript(() => (window as any).mapInitial = { delayInitialization: true });
            await mountTrailModalFixture(page, bundle);
            await page.locator('.map-view-toggle').click();
            await expect(page.getByTestId('fake-map')).toBeHidden();
            await page.evaluate(() => (window as any).mapResultsFixture.initialize());
            await expect.poll(() => eventCount(page)).toBeGreaterThan(0);
            await expect(page.locator('.trail-list')).toBeVisible();
            expect(await camera(page)).toMatchObject({ center: [7, 46], zoom: 14 });
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

        test('retains a keyboard checkbox across document focus loss', async ({ page }) => {
            await openLinkedMap(page, bundle);
            const link = page.locator('.result-items .trail-result > a').first();
            await link.focus();
            await page.keyboard.press('Tab');
            const checkbox = link.locator('..').getByRole('checkbox');
            await expect(checkbox).toBeFocused();
            await page.evaluate(() => {
                const hasFocus = document.hasFocus.bind(document);
                document.hasFocus = () => false;
                document.activeElement!.dispatchEvent(new FocusEvent('focusout', { bubbles: true, relatedTarget: null }));
                document.hasFocus = hasFocus;
            });
            await expect(checkbox).toBeFocused();
            await page.keyboard.press('Space');
            await expect(checkbox).toBeChecked();
        });

        test('keeps the camera, search area and selection when a modal container resize emits moveend', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await page.getByTestId('map-card').first().hover();
            await page.getByRole('checkbox', { name: 'Select Fixture trail 0', exact: true }).check();
            await page.evaluate(() => (window as any).mapResultsFixture.emitResizeMoves = true);
            await page.evaluate(() => { const dialog = document.createElement('dialog'); document.body.append(dialog); dialog.showModal(); });
            const before = await eventCount(page);
            const beforeCamera = await camera(page);
            for (const width of ['90%', '100%']) {
                const zooms = await page.evaluate(() => (window as any).mapResultsFixture.zooms.length);
                await page.locator('main').evaluate((element, width) => element.style.width = width, width);
                await expect.poll(() => page.evaluate(() => (window as any).mapResultsFixture.zooms.length)).toBeGreaterThan(zooms);
                expect(await eventCount(page)).toBe(before);
                expect(await camera(page)).toEqual(beforeCamera);
                await expect(page.locator('.selection-count')).toHaveText('1 selected');
            }
        });

        test('keeps bearing and pitch when expanding results and collapsing them from the overview', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await page.evaluate(() => (window as any).mapResultsFixture.pan());
            await toggleExpandedResults(page);
            expect(await camera(page)).toMatchObject({ bearing: 12, pitch: 30 });
            await page.locator('.overview-actions .expand-results').click();
            await expect(page.locator('.map-results')).not.toHaveClass(/results-expanded/);
            expect(await camera(page)).toMatchObject({ bearing: 12, pitch: 30 });
        });

        test('reports the newly visible area on automatic resize without changing the camera', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await page.evaluate(() => { (window as any).mapResultsFixture.sizedBounds = true; (window as any).mapResultsFixture.pan(); });
            const before = await eventCount(page);
            const previous = await page.evaluate(() => (window as any).mapResultsFixture.events.at(-1).bounds);
            await page.locator('main').evaluate(element => element.style.width = '75%');
            await expect.poll(() => eventCount(page)).toBeGreaterThan(before);
            expect(await camera(page)).toEqual(movedCamera);
            expect(await page.evaluate(() => (window as any).mapResultsFixture.events.at(-1).bounds)).not.toEqual(previous);
        });

        test('reports rendered overview bounds after abrupt window resizing', async ({ page }) => {
            await page.addInitScript(() => (window as any).mapInitial = { cameraGeometry: true });
            await openLinkedMap(page, bundle);
            await toggleExpandedResults(page);
            for (const size of [{ width: 1000, height: 360 }, { width: 1400, height: 900 }]) {
                const before = await eventCount(page);
                await page.setViewportSize(size);
                await expect.poll(() => eventCount(page)).toBeGreaterThan(before);
                await expect(page.locator('.map-results')).toHaveAttribute('data-layout-ready', 'true');
                // Let window/visualViewport observers finish before comparing
                // the last list notification with the actual rendered camera.
                await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
                const bounds = await page.evaluate(() => {
                    const state = (window as any).mapResultsFixture;
                    return { list: state.events.at(-1).bounds, rendered: state.renderedBounds() };
                });
                for (const corner of ['northEast', 'southWest'] as const) {
                    expect(bounds.list[corner].lat).toBeCloseTo(bounds.rendered[corner].lat, 10);
                    expect(bounds.list[corner].lng).toBeCloseTo(bounds.rendered[corner].lng, 10);
                }
            }
        });

        test('accepts delayed browser location while mobile results are visible', async ({ page }) => {
            await page.setViewportSize({ width: 390, height: 600 });
            await page.addInitScript(() => (window as any).mapInitial = { center: null, delayGeolocation: true, geolocation: { lat: 51, lon: 13 } });
            await mountTrailModalFixture(page, bundle);
            await expect.poll(() => eventCount(page)).toBeGreaterThan(0);
            await page.locator('.map-view-toggle').click();
            await expect(page.getByTestId('fake-map')).toBeHidden();
            await page.evaluate(() => (window as any).mapResultsFixture.locate());
            await expect.poll(() => page.evaluate(() => (window as any).mapResultsFixture.events.at(-1))).toMatchObject({ bounds: { northEast: { lat: 51.1, lng: expect.closeTo(13.1, 6) } }, zoom: 12 });
            await page.locator('.map-view-toggle').click();
            await expect(page.getByTestId('fake-map')).toBeVisible();
            expect(await camera(page)).toMatchObject({ center: [13, 51], zoom: 12 });
        });

        test('ignores delayed browser location during a user gesture', async ({ page }) => {
            await page.addInitScript(() => (window as any).mapInitial = { center: null, delayGeolocation: true, geolocation: { lat: 51, lon: 13 } });
            await mountTrailModalFixture(page, bundle);
            await expect.poll(() => eventCount(page)).toBeGreaterThan(0);
            await page.evaluate(() => { (window as any).mapResultsFixture.moving = true; (window as any).mapResultsFixture.locate(); });
            expect(await camera(page)).toMatchObject({ center: [0, 0], zoom: 3 });
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

        test('restores the default view from the small overview', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await page.evaluate(() => (window as any).mapResultsFixture.configureDefault({
                bounds: { has_trails: false, min_lon: 0, max_lon: 0, min_lat: 0, max_lat: 0 },
                settings: { mapFocus: 'location', location: { lon: 11, lat: 48 } },
            }));
            await toggleExpandedResults(page);
            for (const mode of ['overview']) {
                await page.evaluate(() => (window as any).mapResultsFixture.pan());
                await page.locator('#trail-map .reset-map-view').click();
                await expect.poll(() => camera(page)).toEqual({ center: [11, 48], zoom: 12, bearing: 0, pitch: 0 });
            }
        });

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
            const toggle = page.getByRole('button', { name: 'trail-map-hide-results' });
            await expect(page.locator('.result-header .collapse-results')).toBeVisible();
            await expect(page.locator('.result-header .expand-results')).toBeVisible();
            await expect(page.locator('.map-divider')).toHaveCount(0);
            await expect(page.locator('.trail-list .list-resize')).toBeVisible();
            const target = (await toggle.boundingBox())!;
            expect(target.width).toBeGreaterThanOrEqual(44);
            expect(target.height).toBeGreaterThanOrEqual(44);
            const root = (await page.locator('.map-results').boundingBox())!;
            expect(target.y - root.y).toBeLessThan(32);
            await expect(toggle.locator('i')).toHaveCount(1);
            await page.evaluate(() => (window as any).mapResultsFixture.display('table'));
            await expect(page.getByTestId('map-table')).toBeVisible();
            expect(await eventCount(page)).toBe(before);
        });

        test('selects several trails across cards, list and table without navigating or changing the map', async ({ page }) => {
            await openLinkedMap(page, bundle);
            const before = await eventCount(page);
            const url = page.url();
            await page.getByTestId('map-card').first().hover();
            await page.getByRole('checkbox', { name: 'Select Fixture trail 0', exact: true }).check();
            await page.getByTestId('map-card').nth(1).click();
            await expect(page.locator('.selection-count')).toHaveText('2 selected');
            expect(page.url()).toBe(url);
            for (const display of ['list', 'table', 'cards']) {
                await page.evaluate(display => (window as any).mapResultsFixture.display(display), display);
                await expect(page.locator('.trail-results input:checked')).toHaveCount(2);
            }
            await page.getByRole('checkbox', { name: 'trail-results-select-loaded' }).check();
            await expect(page.locator('.selection-count')).toHaveText('6 selected');
            await page.getByRole('button', { name: 'trail-results-clear-selection' }).click();
            await expect(page.locator('.selection-toolbar')).toHaveCount(0);
            await expect(page.locator('.result-items .trail-result > a').first()).toBeFocused();
            await page.getByTestId('view-map').hover();
            for (const item of await page.getByTestId('map-card').all()) {
                await expect(item).toHaveAttribute('data-hovered', 'false');
            }
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

        test('retains multi-select across layouts and forwards edits', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await page.getByTestId('map-card').first().hover();
            await page.getByRole('checkbox', { name: 'Select Fixture trail 0', exact: true }).check();
            await toggleExpandedResults(page);
            await expect(page.getByRole('button', { name: 'trail-results-selection-actions' })).toBeVisible();
            expect(await page.evaluate(() => (window as any).mapResultsFixture.selected())).toEqual(['fixture-0']);

            await expect(page.getByRole('button', { name: 'trail-results-selection-actions' })).toBeVisible();
            await expect(page.locator('.selection-count')).toHaveText('1 selected');
            const updated = [{ id: 'fixture-0', difficulty: 'difficult' }];
            await page.evaluate(updated => (window as any).mapResultsFixture.update(updated), updated);
            await expect(page.locator('.selection-toolbar')).toHaveCount(0);
            expect(await page.evaluate(() => (window as any).mapResultsFixture.updates)).toEqual([updated]);
        });

        for (const lock of ['modal', 'scroll']) {
            test(`window resize preserves selected trails under ${lock} lock`, async ({ page }) => {
                await openLinkedMap(page, bundle);
                await page.getByTestId('map-card').first().hover();
                await page.getByRole('checkbox', { name: 'Select Fixture trail 0', exact: true }).check();
                if (lock === 'modal') await page.evaluate(() => (window as any).mapResultsFixture.openActionDialog());
                else await page.evaluate(() => { document.body.style.position = 'fixed'; document.body.style.width = '100%'; });
                const before = await eventCount(page);
                const renders = await page.evaluate(() => (window as any).mapResultsFixture.renders.length);
                await page.setViewportSize({ width: 1400, height: 900 });
                await expect(page.locator('.map-results')).toHaveAttribute('data-layout-ready', 'true');
                await expect.poll(() => page.evaluate(() => (window as any).mapResultsFixture.renders.length)).toBeGreaterThan(renders);
                expect(await eventCount(page)).toBe(before);
                expect(await page.evaluate(() => (window as any).mapResultsFixture.selected())).toEqual(['fixture-0']);
                if (lock === 'modal') await expect(page.locator('dialog[open]')).toBeVisible();
            });
        }

        test('keeps the selection dialog visible across mobile and collapsed results layouts', async ({ page }) => {
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
            await dialog.evaluate(element => (element as HTMLDialogElement).close());
            await page.locator('.collapse-results').click();
            await page.evaluate(() => (window as any).mapResultsFixture.openActionDialog());
            await expect(dialog).toBeVisible();
            await expect(dialog.getByRole('textbox')).toBeFocused();
            await dialog.evaluate(element => (element as HTMLDialogElement).close());
        });

        test('expands results into a responsive grid, retaining the map and search area', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await page.evaluate(() => (window as any).mapResultsFixture.emitResizeMoves = true);
            const before = await eventCount(page);
            await page.getByTestId('fake-map').evaluate(element => (window as any).retainedMap = element);
            await toggleExpandedResults(page);
            const root = page.locator('.map-results');
            await expect(root).toHaveClass(/results-expanded/);
            await expect(page.locator('.result-header')).toHaveCount(0);
            await expect(page.locator('.collapse-results')).toHaveCount(0);
            await expect(page.locator('.map-pane .overview-actions .expand-results')).toBeVisible();
            await expect(page.locator('.expand-results i')).toHaveClass(/fa-expand/);
            await expect.poll(() => page.evaluate(() => (window as any).mapResultsFixture.zooms.at(-1))).toBe(12);
            expect(await page.evaluate(() => (window as any).mapResultsFixture.fittedSize)).toEqual(
                await page.getByTestId('fake-map').evaluate(element => ({ width: element.clientWidth, height: element.clientHeight })));
            await expect(page.getByRole('separator')).toBeHidden();
            const layout = await root.evaluate(element => {
                const list = element.querySelector('.trail-list')!.getBoundingClientRect();
                const map = element.querySelector('#trail-map')!.getBoundingClientRect();
                const cards = [...element.querySelectorAll('.result-items .trail-result > a')].map(card => card.getBoundingClientRect());
                return { width: element.clientWidth, listWidth: list.width, mapWidth: map.width,
                    columns: new Set(cards.map(card => card.x)).size, maxCardWidth: Math.max(...cards.map(card => card.width)) };
            });
            expect(layout.listWidth).toBe(layout.width);
            expect(layout.mapWidth).toBeLessThan(layout.width / 2);
            expect(layout.columns).toBeGreaterThan(1);
            expect(layout.maxCardWidth).toBeLessThanOrEqual(576);
            expect(await page.getByTestId('fake-map').evaluate(element => element === (window as any).retainedMap)).toBe(true);
            expect(await eventCount(page)).toBe(before);
            await toggleExpandedResults(page);
            await expect(root).not.toHaveClass(/results-expanded/);
            await expect(page.getByRole('separator')).toHaveAttribute('aria-valuenow', '400');
            await toggleExpandedResults(page);
            await page.reload();
            await page.addScriptTag({ content: bundle });
            await expect(root).toHaveClass(/results-expanded/);
            await expect(page.getByTestId('fake-map')).toBeVisible();
            await expect(page.locator('.collapse-results')).toHaveCount(0);
            await toggleExpandedResults(page);
            await page.getByRole('button', { name: 'trail-map-hide-results', exact: true }).click();
            await expect(page.locator('.trail-list')).toBeHidden();
            await expect(root).not.toHaveClass(/results-expanded/);
        });

        test('resizes the small map by dragging and keyboard, persists it and cancels a drag with Escape', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await toggleExpandedResults(page);
            const handle = page.getByRole('button', { name: 'trail-map-resize-overlay', exact: true });
            await expect(handle).toBeVisible();
            await expect(handle.locator('i')).toHaveCount(0);
            expect(await handle.evaluate(element => getComputedStyle(element).backgroundColor)).toBe('rgba(0, 0, 0, 0)');
            await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
            await page.evaluate(() => (window as any).mapResultsFixture.emitResizeMoves = true);
            const before = await eventCount(page);
            const original = (await page.locator('#trail-map').boundingBox())!;
            const grip = (await handle.boundingBox())!;
            await page.mouse.move(grip.x + grip.width / 2, grip.y + grip.height / 2);
            await page.mouse.down();
            await page.mouse.move(grip.x + grip.width / 2 - 100, grip.y + grip.height / 2 - 60, { steps: 4 });
            await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ width: original.width + 100, height: original.height + 60 });
            await expect.poll(() => page.evaluate(() => (window as any).mapResultsFixture.redraws ?? 0)).toBeGreaterThan(0);
            expect(await eventCount(page)).toBe(before);
            await page.mouse.up();
            await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
            expect(await eventCount(page)).toBe(before);
            const saved = await page.evaluate(() => JSON.parse(localStorage.getItem('mapResultsLayout')!));
            expect(saved.overlay).toEqual({ width: original.width + 100, height: original.height + 60 });
            await page.reload();
            await page.addScriptTag({ content: bundle });
            await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject(saved.overlay);
            await handle.press('Home');
            await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ width: 256, height: 160 });
            await handle.press('ArrowLeft');
            await handle.press('Shift+ArrowUp');
            await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ width: 272, height: 224 });
            const stored = await page.evaluate(() => localStorage.getItem('mapResultsLayout'));
            const next = (await handle.boundingBox())!;
            await page.mouse.move(next.x + 16, next.y + 16);
            await page.mouse.down();
            await page.mouse.move(next.x - 50, next.y - 40);
            await handle.press('Escape');
            await page.mouse.up();
            await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ width: 272, height: 224 });
            expect(await page.evaluate(() => localStorage.getItem('mapResultsLayout'))).toBe(stored);
            await handle.press('End');
            const bounds = (await page.locator('#trail-map').boundingBox())!;
            const root = (await page.locator('.map-results').boundingBox())!;
            expect(bounds.width).toBeLessThanOrEqual(root.width - 64);
            expect(bounds.height).toBeLessThanOrEqual(root.height - 112);
        });

        test('closes and reopens the overview without losing the map, including after reload', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await toggleExpandedResults(page);
            await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
            const before = await eventCount(page);
            await page.getByTestId('fake-map').evaluate(element => (window as any).retainedMap = element);
            await expect(page.locator('.overview-actions button')).toHaveCount(3);
            const map = (await page.locator('#trail-map').boundingBox())!;
            for (const button of await page.locator('.overview-actions button').all()) {
                const box = (await button.boundingBox())!;
                expect(Math.abs(box.y + box.height / 2 - map.y)).toBeLessThanOrEqual(1);
                expect(box).toMatchObject({ width: 32, height: 32 });
                expect(box.x).toBeGreaterThan(map.x + map.width / 2);
            }
            await page.locator('.overlay-close').click();
            await expect(page.getByTestId('fake-map')).toBeHidden();
            await expect(page.locator('.overlay-reopen')).toBeFocused();
            await expect(page.locator('.overview-actions button')).toHaveCount(0);
            expect(await page.evaluate(({ x, y }) => document.elementFromPoint(x, y)?.closest('.map-pane') === null,
                { x: map.x + map.width / 2, y: map.y + map.height / 2 })).toBe(true);
            await page.locator('.overlay-reopen').click();
            await expect(page.getByTestId('fake-map')).toBeVisible();
            await expect(page.locator('.overlay-close')).toBeFocused();
            expect(await page.getByTestId('fake-map').evaluate(element => element === (window as any).retainedMap)).toBe(true);
            expect(await eventCount(page)).toBe(before);
            await page.locator('.overlay-close').click();
            await page.reload();
            await page.addScriptTag({ content: bundle });
            await expect(page.locator('.overlay-reopen')).toBeVisible();
            await expect(page.getByTestId('fake-map')).toBeHidden();

            await expect(page.locator('.overlay-reopen')).toBeVisible();
            await page.locator('.overlay-reopen').click();
            await expect(page.getByTestId('fake-map')).toBeVisible();
        });

        test('moves the overview within its bounds, persists its position and cancels with Escape', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await toggleExpandedResults(page);
            const move = page.locator('.overlay-move');
            const original = (await page.locator('#trail-map').boundingBox())!;
            const before = await eventCount(page);
            const initialCamera = await camera(page);
            const button = (await move.boundingBox())!;
            await page.mouse.move(button.x + button.width / 2, button.y + button.height / 2);
            await page.mouse.down();
            await page.mouse.move(button.x + button.width / 2 - 100, button.y + button.height / 2 - 60, { steps: 4 });
            await page.mouse.up();
            await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ x: original.x - 100, y: original.y - 60, width: original.width, height: original.height });
            expect(await eventCount(page)).toBe(before);
            expect(await camera(page)).toEqual(initialCamera);
            await page.reload();
            await page.addScriptTag({ content: bundle });
            await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ x: original.x - 100, y: original.y - 60 });
            await move.press('ArrowLeft');
            await move.press('Shift+ArrowUp');
            await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ x: original.x - 116, y: original.y - 124 });
            await move.press('End');
            const root = (await page.locator('.map-results').boundingBox())!;
            await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ x: root.x, y: root.y });
            await move.press('Home');
            await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ x: original.x, y: original.y });
            const saved = await page.evaluate(() => localStorage.getItem('mapResultsLayout'));
            const next = (await move.boundingBox())!;
            await page.mouse.move(next.x + next.width / 2, next.y + next.height / 2);
            await page.mouse.down();
            await page.mouse.move(next.x + next.width / 2 - 60, next.y + next.height / 2 - 40);
            await move.press('Escape');
            await page.mouse.up();
            await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ x: original.x, y: original.y });
            expect(await page.evaluate(() => localStorage.getItem('mapResultsLayout'))).toBe(saved);
        });

        test('keeps the overview large enough for changing controls, including in a short window', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await toggleExpandedResults(page);
            await page.locator('.maplibregl-ctrl-top-right').evaluate(element => {
                const extra = document.createElement('div');
                extra.style.height = '260px';
                element.prepend(extra);
            });
            await page.locator('.maplibregl-ctrl-bottom-right').evaluate(element => { (element as HTMLElement).style.height = '32px'; });
            const minimum = await page.locator('.maplibregl-ctrl-top-right').evaluate(element => element.getBoundingClientRect().height + 32 + 14);
            const handle = page.locator('.overlay-resize');
            await handle.press('Home');
            await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ width: 256, height: minimum });
            await handle.press('ArrowDown');
            await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ height: minimum });
            await page.setViewportSize({ width: 1000, height: 360 });
            await expect(page.locator('.map-results')).toHaveAttribute('data-layout-ready', 'true');
            // Resize events and observers can arrive after setViewportSize resolves.
            // Wait for the layout and callback count to stay unchanged across frames.
            await page.evaluate(() => new Promise<void>(resolve => {
                let previous = '', stable = 0;
                function frame() {
                    const root = document.querySelector<HTMLElement>('.map-results')!;
                    const map = document.querySelector('#trail-map')!;
                    const box = map.getBoundingClientRect();
                    const current = JSON.stringify([(window as any).mapResultsFixture.events.length, box.width, box.height]);
                    stable = current === previous && root.dataset.layoutReady === 'true' ? stable + 1 : 0;
                    previous = current;
                    if (stable >= 3) resolve();
                    else requestAnimationFrame(frame);
                }
                requestAnimationFrame(frame);
            }));
            await page.waitForTimeout(650);
            const root = (await page.locator('.map-results').boundingBox())!;
            expect(root.height).toBeGreaterThanOrEqual(minimum + 112);
            const before = await eventCount(page);
            const grip = (await handle.boundingBox())!;
            await page.mouse.move(grip.x + 12, grip.y + 12);
            await page.mouse.down();
            await page.mouse.move(grip.x + 12, grip.y + 160);
            await page.mouse.up();
            await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ height: minimum });
            expect(await eventCount(page)).toBe(before);
        });

        test('defers a late control minimum until an overview move finishes', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await toggleExpandedResults(page);
            const original = (await page.locator('#trail-map').boundingBox())!;
            const move = (await page.locator('.overlay-move').boundingBox())!;
            await page.mouse.move(move.x + move.width / 2, move.y + move.height / 2);
            await page.mouse.down();
            await page.locator('.maplibregl-ctrl-top-right').evaluate(element => {
                const extra = document.createElement('div'); extra.style.height = '280px'; element.prepend(extra);
            });
            await page.mouse.move(move.x + move.width / 2 - 60, move.y + move.height / 2 - 40, { steps: 4 });
            await expect.poll(() => page.locator('#trail-map').boundingBox()).toMatchObject({ x: original.x - 60, y: original.y - 40, height: original.height });
            await page.mouse.up();
            await expect(page.locator('.map-results')).toHaveAttribute('data-layout-ready', 'true');
            await expect.poll(() => page.locator('#trail-map').boundingBox().then(box => box!.height)).toBeGreaterThan(original.height);
        });

        test('drags the splitter within its limits and reports the viewport only after release', async ({ page }) => {
            await openLinkedMap(page, bundle);
            await page.evaluate(() => (window as any).mapResultsFixture.emitResizeMoves = true);
            const divider = page.getByRole('separator', { name: 'trail-map-resize-results' });
            const startWidth = Number(await divider.getAttribute('aria-valuenow'));
            const box = (await divider.boundingBox())!;
            const before = await eventCount(page);
            await page.mouse.move(box.x + box.width / 2, box.y + box.height * .75);
            await page.mouse.down();
            await page.mouse.move(box.x + box.width / 2 + 120, box.y + box.height * .75, { steps: 4 });
            await expect(divider).toHaveAttribute('aria-valuenow', String(startWidth + 120));
            await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
            expect(await eventCount(page)).toBe(before);
            await page.mouse.up();
            await expect.poll(() => eventCount(page)).toBeGreaterThan(before);
            expect(await page.evaluate(() => JSON.parse(localStorage.getItem('mapResultsLayout')!))).toMatchObject({ width: startWidth + 120, collapsed: false });
            await divider.press('Home');
            await expect(divider).toHaveAttribute('aria-valuenow', '280');
            const left = (await divider.boundingBox())!;
            await page.mouse.move(left.x + left.width / 2, left.y + left.height * .75);
            await page.mouse.down();
            await page.mouse.move(0, left.y + left.height * .75);
            await page.mouse.up();
            await expect(divider).toHaveAttribute('aria-valuenow', '280');
            await divider.press('End');
            expect(await divider.getAttribute('aria-valuenow')).toBe(await divider.getAttribute('aria-valuemax'));
            const mapWidth = await page.locator('#trail-map').evaluate(element => element.clientWidth);
            expect(mapWidth).toBeGreaterThanOrEqual(318);
        });

        test('supports keyboard sizing, collapse and restoring both preferences after reload', async ({ page }) => {
            await openLinkedMap(page, bundle);
            const divider = page.getByRole('separator', { name: 'trail-map-resize-results' });
            await divider.press('Home');
            await divider.press('ArrowRight');
            await expect(divider).toHaveAttribute('aria-valuenow', '296');
            await divider.press('Shift+ArrowRight');
            await expect(divider).toHaveAttribute('aria-valuenow', '360');
            await divider.press('ArrowLeft');
            await expect(divider).toHaveAttribute('aria-valuenow', '344');
            await page.reload();
            await page.addScriptTag({ content: bundle });
            await expect(divider).toHaveAttribute('aria-valuenow', '344');
            await divider.press('Enter');
            await expect(divider).toBeHidden();
            await expect(page.getByRole('button', { name: 'trail-map-restore-results' })).toBeVisible();
            await expect(page.locator('.trail-list')).toBeHidden();
            await page.reload();
            await page.addScriptTag({ content: bundle });
            await expect(divider).toBeHidden();
            await page.getByRole('button', { name: 'trail-map-restore-results' }).click();
            await expect(divider).toHaveAttribute('aria-valuenow', '344');
            await expect(page.locator('.trail-list')).toBeVisible();
        });

        test('cancels a drag with Escape without changing the saved width', async ({ page }) => {
            await openLinkedMap(page, bundle);
            const divider = page.getByRole('separator', { name: 'trail-map-resize-results' });
            await divider.press('Home');
            const saved = await page.evaluate(() => localStorage.getItem('mapResultsLayout'));
            const box = (await divider.boundingBox())!;
            await page.mouse.move(box.x + box.width / 2, box.y + box.height * .75);
            await page.mouse.down();
            await page.mouse.move(box.x + box.width / 2 + 100, box.y + box.height * .75);
            await expect(divider).toHaveAttribute('aria-valuenow', '380');
            await divider.press('Escape');
            await page.mouse.up();
            await expect(divider).toHaveAttribute('aria-valuenow', '280');
            expect(await page.evaluate(() => localStorage.getItem('mapResultsLayout'))).toBe(saved);
        });

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
