import { readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { extname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { compile, compileModule } from 'svelte/compiler';
import { build } from 'vite';
import { expect, type Page } from '@playwright/test';

const root = fileURLToPath(new URL('../../../', import.meta.url));
const source = resolve(root, 'src/routes/trails/+page.svelte');

// Exercise the actual page orchestration. Rendering children and transport are
// controlled so replies can arrive out of order, even after their signal aborts.
export async function buildTrailsPageFixture(dev: boolean) {
    const result = await build({
        root, configFile: false, logLevel: 'error',
        resolve: { conditions: ['browser', dev ? 'development' : 'production'] },
        plugins: [{
            name: 'trails-page-fixture', enforce: 'pre',
            resolveId(id) {
                if (id.startsWith('fixture:')) return `\0${id}`;
                if (id === '$app/environment') return '\0fixture:environment';
                if (id === '$app/navigation') return '\0fixture:navigation';
                if (id === '$app/state') return '\0fixture:state.svelte.js';
                if (id === 'svelte-i18n') return '\0fixture:i18n';
                if (id === 'maplibre-gl') return '\0fixture:maplibre';
                if (id.startsWith('$lib/')) {
                    const path = resolve(root, 'src/lib', id.slice(5));
                    return extname(id) ? path : [path + '.ts', path + '.js'].find(existsSync);
                }
            },
            async load(id) {
                if (id === '\0fixture:entry') return `
                    import { mount, unmount } from 'svelte';
                    import Page from ${JSON.stringify(source)};
                    window.fixture = { requests: [], focusedPlaces: [] };
                    window.fetch = (url, options) => new Promise((resolve, reject) => {
                        window.fixture.requests.push({ channel: url, body: JSON.parse(options.body),
                            signal: options.signal, resolve: data => resolve(Response.json(data)), reject });
                    });
                    const component = mount(Page, { target: document.body });
                    window.fixture.capture = () => component.snapshot.capture();
                    window.fixture.restore = value => component.snapshot.restore(value);
                    window.fixture.destroy = () => unmount(component);
                `;
                if (id === '\0fixture:environment') return 'export const browser = true;';
                if (id === '\0fixture:maplibre') return 'export class LngLat { constructor(lng, lat) { this.lng = lng; this.lat = lat; } }';
                if (id === '\0fixture:state.svelte.js') return compileModule(`
                    export const page = $state({ url: new URL(location.href), state: {}, data: {
                        categories: [], subcategories: [], settings: {}, boundingBox: {},
                        filter: { q: '', category: [], subcategory: [], tags: [], difficulty: [0, 1, 2],
                            author: '', public: true, private: true, shared: true,
                            near: { radius: 2000 }, distanceMin: 0, distanceMax: 50000, distanceLimit: 50000,
                            elevationGainMin: 0, elevationGainMax: 2000, elevationGainLimit: 2000,
                            elevationLossMin: 0, elevationLossMax: 2000, elevationLossLimit: 2000,
                            sort: 'relevance', sortOrder: '+' }
                    } });
                    window.addEventListener('popstate', () => {
                        page.url = new URL(location.href);
                        page.state = history.state ?? {};
                    });
                `, { filename: 'fixture-state.svelte.js', generate: 'client', dev }).js.code;
                if (id === '\0fixture:navigation') return `
                    import { page } from 'fixture:state.svelte.js';
                    export function beforeNavigate() {}
                    export function pushState(url, state) {
                        history.pushState(state, '', url); page.url = new URL(location.href); page.state = state;
                    }
                    export function replaceState(url, state) {
                        history.replaceState(state, '', url); page.url = new URL(location.href); page.state = state;
                    }
                `;
                if (id === '\0fixture:i18n') return `
                    import { writable } from 'svelte/store';
                    export const _ = writable(key => key);
                `;
                if (id === resolve(root, 'src/lib/stores/trail_store.ts')) return `
                    async function query(channel, body, fetcher) {
                        const response = await fetcher(channel, { method: 'POST', body: JSON.stringify(body) });
                        if (!response.ok) throw new Error('Search failed');
                        return response.json();
                    }
                    export function trails_search_filter_pages(filter, pages, perPage, fetcher, bounds) {
                        return query('results', { filter, pages, perPage, bounds }, fetcher);
                    }
                    export function trails_search_filter(filter, page, perPage, fetcher, bounds) {
                        return query('results', { filter, page, perPage, bounds }, fetcher);
                    }
                    export function trails_search_map_data(northEast, southWest, filter, zoom, fetcher = fetch) {
                        return query('map', { northEast, southWest, filter, zoom }, fetcher);
                    }
                `;
                let component: string;
                if (id === resolve(root, 'src/lib/components/trail/trail_list.svelte')) component = `
                    <script>
                        let { filter = $bindable(), trails, pagination, display = $bindable(), loading,
                            onupdate, ondisplaychange, searchBar, mapView, moreResults } = $props();
                        let visited = $state(false);
                        $effect(() => { if (display === 'map') visited = true; });
                        $effect(() => {
                            window.fixture.display = mode => { display = mode; ondisplaychange(mode); };
                            window.fixture.sort = (sort, sortOrder) => { filter = { ...filter, sort, sortOrder }; onupdate(); };
                        });
                    </script>
                    {@render searchBar()}
                    <output data-testid="results">{JSON.stringify({ trails, pagination, filter, loading, display })}</output>
                    {#if visited}<div hidden={display !== 'map'}>{@render mapView()}</div>{/if}
                    {@render moreResults()}
                `;
                else if (id === resolve(root, 'src/lib/components/trail/trail_search_bar.svelte')) component = `
                    <script>
                        let { filter = $bindable(), onapply, onclearbounds } = $props();
                        $effect(() => {
                            window.fixture.apply = (change, resetBounds = false) => { filter = { ...filter, ...change }; onapply(filter, resetBounds); };
                            window.fixture.clearBounds = () => onclearbounds?.();
                        });
                    </script>
                `;
                else if (id === resolve(root, 'src/lib/components/trail/trail_map_results.svelte')) component = `
                    <script>
                        let { mapTrails, clusters, mapLoading, onviewport } = $props();
                        $effect(() => { window.fixture.viewport = onviewport; });
                        export function focusBounds(bounds) { onviewport(bounds, 11, true); }
                        export function focusPlace(place) { window.fixture.focusedPlaces.push(JSON.parse(JSON.stringify(place))); }
                    </script>
                    <output data-testid="map">{JSON.stringify({ mapTrails, clusters, mapLoading })}</output>
                `;
                else if (id.endsWith('.svelte')) component = await readFile(id, 'utf8');
                else return;
                return compile(component, { filename: id, generate: 'client', dev, css: 'injected' }).js.code;
            },
        }],
        build: { write: false, minify: false, rolldownOptions: { input: 'fixture:entry', output: { format: 'iife' } } },
    });
    if (Array.isArray(result) || !('output' in result)) throw new Error('Expected a single bundle');
    const chunk = result.output.find(output => output.type === 'chunk');
    if (!chunk) throw new Error('Missing fixture bundle');
    return chunk.code;
}

export async function mountTrailsPageFixture(page: Page, bundle: string) {
    await page.route('http://trails-page.test/**', route => route.fulfill({ contentType: 'text/html', body: '<!doctype html><html><body></body></html>' }));
    await page.goto('http://trails-page.test/trails?view=list');
    await page.addScriptTag({ content: bundle });
    await expect.poll(() => page.evaluate(() => (window as any).fixture.requests.length)).toBe(1);
    await expect.poll(() => page.evaluate(() => history.state?.trailSearchView)).toBe('list');
}
