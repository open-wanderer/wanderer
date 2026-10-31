import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { extname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { compile, compileModule } from 'svelte/compiler';
import { build } from 'vite';
import tailwindcss from '@tailwindcss/vite';
import { expect, type Page } from '@playwright/test';
import { mapResultStubs } from './trail-map-results-fixture';

const root = fileURLToPath(new URL('../../../', import.meta.url));


// Exercise the actual page orchestration. Rendering children and transport are
// controlled so replies can arrive out of order, even after their signal aborts.
export async function buildMapPageFixture(dev: boolean, realMapResults = false, realEngine = false, includeNav = false, realSearch = false) {
    // The high-concurrency stress run prebuilds these same bundles from its
    // immutable checkout. Normal component runs always compile fresh fixtures.
    const cache = process.env.PLAYWRIGHT_MAP_FIXTURE_CACHE;
    const cacheFile = cache && resolve(cache, [dev, realMapResults, realEngine, includeNav, realSearch].map(Number).join('-') + '.js');
    if (cacheFile && existsSync(cacheFile)) return readFile(cacheFile, 'utf8');
    const source = resolve(root, 'src/routes/map/+page.svelte');
    const dist = resolve(root, 'node_modules/maplibre-gl/dist/maplibre-gl.mjs');
    const engineSource = realEngine ? await readFile(dist, 'utf8') : '';
    const engineExports = engineSource.slice(engineSource.lastIndexOf('export{') + 7, engineSource.lastIndexOf('}')).split(',').map(part => part.split(' as ').pop()!.trim()).filter(Boolean);
    const mapComponent = resolve(root, 'src/lib/components/trail/map_with_elevation_maplibre.svelte');
    const result = await build({
        root, configFile: false, logLevel: 'error',
        resolve: { conditions: ['browser', dev ? 'development' : 'production'] },
        plugins: [...((includeNav || realEngine || realSearch) ? [tailwindcss()] : []), {
            name: 'map-page-fixture', enforce: 'pre',
            resolveId(id, importer) {
                if ((includeNav || realSearch) && id === '#lib/stores/user_store.ts') return '\0fixture:user';
                if (includeNav && id === '#lib/util/file_util.ts' && importer?.endsWith('nav_bar.svelte')) return '\0fixture:files';
                if (id.startsWith('fixture:')) return `\0${id}`;
                if (id === '$app/env') return '\0fixture:environment';
                if (id === '$app/navigation') return '\0fixture:navigation';
                if (id === '$app/state') return '\0fixture:state.svelte.js';
                if (id === 'svelte-i18n') return '\0fixture:i18n';
                if (realEngine && (id === 'maplibre-gl/dist/maplibre-gl.css' || id === '#lib/util/maplibre_worker.ts')) return '\0fixture:empty';
                if (id === 'maplibre-gl') return '\0fixture:maplibre';
                if (id.startsWith('#lib/')) {
                    const path = resolve(root, 'src/lib', id.slice(5));
                    return extname(id) ? path : [path + '.ts', path + '.js'].find(existsSync);
                }
            },
            async load(id) {
                if (id === '\0fixture:user') return `import { writable } from 'svelte/store'; export const currentUser = writable({ id: 'tester', actor: 'tester', username: 'tester' }); export function logout() { sessionStorage.setItem('fixture:loggedOut', 'true'); }`;
                if (id === '\0fixture:files') return 'export function getFileURL() { return undefined; }';
                if (id === resolve(root, 'src/lib/stores/upload_store.svelte.ts')) return compileModule('export const uploadStore = $state({ enqueuedUploads: [], completedUploads: [], uploading: false }); export async function processUploadQueue() {}', { filename: 'fixture-upload.svelte.js', generate: 'client', dev }).js.code;
                if (id === '\0fixture:empty') return 'export default {};';
                if (realEngine && id === '\0fixture:maplibre') return `const M = window.RealMapLibre; export default M; export const { ${engineExports.join(', ')} } = M;`;
                if ((realSearch || includeNav) && id === resolve(root, 'src/lib/stores/toast_store.svelte.ts')) return 'export const toastStore = {toasts: []}; export function show_toast() {} export function closeToast() {} export function pauseToast() {} export function resumeToast() {}';
                if (id === resolve(root, 'src/lib/stores/notification_store.ts')) return 'export async function notifications_index() { return {items: [], page: 1, totalPages: 1}; } export async function notifications_mark_as_seen() {}';
                if (id === '\0fixture:entry') return `
                    import { mount, unmount } from 'svelte';
                    import Page from ${JSON.stringify(source)};
                    ${includeNav ? `import NavBar from ${JSON.stringify(resolve(root, 'src/lib/components/nav_bar.svelte'))};
                    import Toast from ${JSON.stringify(resolve(root, 'src/lib/components/base/toast.svelte'))};
                    import UploadDialog from ${JSON.stringify(resolve(root, 'src/lib/components/settings/upload_dialog.svelte'))};
                    import { uploadStore } from ${JSON.stringify(resolve(root, 'src/lib/stores/upload_store.svelte.ts'))};` : ''}
                    ${includeNav || realEngine || realSearch ? `import ${JSON.stringify(resolve(root, 'src/css/app.css'))};
                    import ${JSON.stringify(resolve(root, 'src/css/components.css'))};
                    import ${JSON.stringify(resolve(root, 'src/css/theme.css'))};` : ''}
                    window.fixture = { requests: [], focusedPlaces: [], focusedLocations: [], focusedBounds: [], publications: [] };
                    window.mapResultsFixture = {};
                    window.mapInitial = { cameraGeometry: true };
                    const originalFetch = window.fetch.bind(window);
                    window.fetch = (url, options) => {
                        if (typeof url !== 'string' || !options?.body) return originalFetch(url, options);
                        return new Promise((resolve, reject) => {
                        window.fixture.requests.push({ channel: url, body: JSON.parse(options.body),
                            signal: options.signal, resolve: data => resolve(Response.json(data)), reject });
                        if (window.fixture.autoResolve) resolve(Response.json(url === 'map'
                            ? { mapTrails: [], clusters: { type: 'FeatureCollection', features: [] } }
                            : { items: [{ id: 'fixture-1', author: 'tester', name: 'Fixture trail' }], page: 1, totalPages: 1, totalHits: 1 }));
                    }); };
                    ${includeNav ? `document.documentElement.classList.add('light'); document.body.style.width = '100%'; mount(NavBar, { target: document.body, props: { user: { id: 'tester', username: 'tester', avatar: '' } } }); mount(Toast, { target: document.body }); mount(UploadDialog, { target: document.body }); window.fixture.addUpload = () => uploadStore.enqueuedUploads.push({ file: new File(['fixture'], 'fixture.gpx'), status: 'enqueued', progress: 0 }); window.fixture.completeUpload = () => uploadStore.completedUploads.push({ ...uploadStore.enqueuedUploads.shift(), status: 'success' });` : ''}
                    let component = mount(Page, { target: document.body });
                    window.fixture.remount = async (value, mode = 'sync') => {
                        await unmount(component);
                        if (mode === 'popstate') window.dispatchEvent(new PopStateEvent('popstate', { state: history.state }));
                        component = mount(Page, { target: document.body });
                        if (mode !== 'sync') await new Promise(requestAnimationFrame);
                        component.snapshot.restore(value);
                    };
                    window.fixture.capture = () => component.snapshot.capture();
                    window.fixture.restore = value => component.snapshot.restore(value);
                    window.fixture.destroy = () => unmount(component);
                `;
                if (id === '\0fixture:environment') return 'export const browser = true; export const dev = false; export const building = false; export const version = "fixture";';
                if (id === '\0fixture:maplibre') return 'export class LngLat { constructor(lng, lat) { this.lng = lng; this.lat = lat; } } export class LngLatBounds { constructor(sw, ne) { this.sw = sw; this.ne = ne; } toArray() { return [this.sw, this.ne]; } }';
                if (id === '\0fixture:state.svelte.js') return compileModule(`
                    export const page = $state({ url: new URL(location.href), state: history.state ?? {}, route: { id: '/map' }, data: {
                        categories: [], subcategories: [], settings: {}, boundingBox: {}, notifications: {items: [], page: 0, totalPages: 0},
                        filter: { q: '', category: [], subcategory: [], tags: [], difficulty: [0, 1, 2],
                            author: '', public: true, private: true, shared: true,
                            near: { radius: 2000 }, distanceMin: 0, distanceMax: 50000, distanceLimit: 50000,
                            elevationGainMin: 0, elevationGainMax: 2000, elevationGainLimit: 2000,
                            elevationLossMin: 0, elevationLossMax: 2000, elevationLossLimit: 2000,
                            sort: 'created', sortOrder: '-' }
                    } });
                    window.addEventListener('popstate', () => {
                        page.url = new URL(location.href);
                        page.state = history.state ?? {};
                    });
                `, { filename: 'fixture-state.svelte.js', generate: 'client', dev }).js.code;
                if (id === '\0fixture:navigation') return `
                    import { page } from 'fixture:state.svelte.js';
                    export function goto() {}
                    export function beforeNavigate() {}
                    export function pushState(url, state) {
                        history.pushState(state, '', url); page.state = state;
                    }
                    export function replaceState(url, state) {
                        history.replaceState(state, '', url); page.state = state;
                    }
                `;
                if (id === '\0fixture:i18n') return `
                    import { writable } from 'svelte/store';
                    export const _ = writable(key => key);
                    export const t = _, locale = writable('en'), json = writable(() => undefined);
                    export const locales = writable(['en']), isLoading = writable(false);
                    export const date = writable(String), time = writable(String), number = writable(String);
                    export function init() {} export function register() {} export function addMessages() {}
                    export function waitLocale() { return Promise.resolve(); } export function getLocaleFromNavigator() { return 'en'; }
                `;
                if (id === resolve(root, 'src/lib/stores/trail_store.ts')) return `
                    async function query(channel, body, fetcher) {
                        const response = await fetcher(channel, { method: 'POST', body: JSON.stringify(body) });
                        if (!response.ok) throw new Error('Search failed');
                        return response.json();
                    }
                    export function buildFilterText() { return "fixture"; }
                    export async function trails_upload() {}
                    export function trails_search_filter(filter, page, perPage, fetcher, bounds) {
                        return query('results', { filter, page, perPage, bounds }, fetcher);
                    }
                    export function trails_search_map_data(northEast, southWest, filter, zoom, fetcher = fetch) {
                        return query('map', { northEast, southWest, filter, zoom }, fetcher);
                    }
                `;
                let component: string;

                if (realMapResults && id in mapResultStubs && !(realEngine && id === mapComponent)) {
                    component = mapResultStubs[id as keyof typeof mapResultStubs];
                    return compile(component, { filename: id, generate: 'client', dev, css: 'injected' }).js.code;
                }
                if (realMapResults && id.endsWith('/trail_merge_dialog.svelte')) return compile('<div></div>', { filename: id, generate: 'client', dev }).js.code;
                if (!realSearch && id === resolve(root, 'src/lib/components/trail/trail_search_bar.svelte')) component = `
                    <script>
                        let { filter = $bindable(), onapply, onplaceselect, layout } = $props();
                        $effect(() => {
                            window.fixture.placeSuggestion = onplaceselect;
                            window.fixture.apply = (change, resetBounds = false) => { filter = { ...filter, ...change }; onapply(filter, resetBounds); };
                        });
                        export function restoreFilter() {}
                    </script>
                    <div data-search-root><input type="search" data-testid="fixture-search" aria-label="Fixture search" />
                    <button data-testid="fixture-trigger">Full screen trigger</button>
                    <output data-testid="search-filter">{JSON.stringify({ filter, layout })}</output></div>
                `;
                else if (id === resolve(root, 'src/lib/components/trail/trail_result_controls.svelte')) component = `
                    <script>
                        let { onsort, ondisplay, onsearchposition } = $props();
                        $effect(() => {
                            window.fixture.sort = onsort;
                            window.fixture.display = ondisplay;
                            window.fixture.position = onsearchposition;
                        });
                    </script>
                `;
                else if (!realMapResults && id === resolve(root, 'src/lib/components/trail/trail_map_results.svelte')) component = `
                    <script>
                        import { onMount } from 'svelte';
                        let { trails, mapTrails, clusters, loading, mapLoading, initialBounds, initialCenter, onviewport, onzoom, moreResults, display, onupdate } = $props();
                        $effect(() => { window.fixture.updateTrails = onupdate; window.fixture.viewport = onviewport; window.fixture.zoom = onzoom; });
                        onMount(() => onviewport(initialBounds ?? { northEast: { lat: 48, lng: 9 }, southWest: { lat: 46, lng: 7 } }, 11, true));
                        export function focusBounds(bounds) { window.fixture.focusedBounds.push(bounds); onviewport(bounds, 11, true); return true; }

                        export function refreshLayout() {}
                        export function captureView() { return undefined; }
                        export function focusCamera() {}
                        export function focusLocation(location) {
                            window.fixture.focusedLocations.push(location);
                            onviewport({ northEast: { lat: location.lat + .1, lng: location.lon + .1 }, southWest: { lat: location.lat - .1, lng: location.lon - .1 } }, 14);
                        }
                        export function focusPlace(place) {
                            window.fixture.focusedPlaces.push(JSON.parse(JSON.stringify(place)));
                            onviewport({ northEast: { lat: place.lat + .1, lng: place.lon + .1 }, southWest: { lat: place.lat - .1, lng: place.lon - .1 } }, 14);
                        }
                    </script>
                    <output data-testid="results">{JSON.stringify({ trails, loading, display })}</output>
                    <output data-testid="map">{JSON.stringify({ mapTrails, clusters, mapLoading })}</output>
                    {@render moreResults()}
                `;
                else if (id.endsWith('.svelte')) {
                    component = await readFile(id, 'utf8');
                    if (realEngine && id === mapComponent) {
                        component = component.replace('        map = new M.Map(finalMapOptions);', '        map = new M.Map(finalMapOptions); window.realMap = map;');
                        component = component.replace('            oninit?.(map!);', '            if (window.deferMapLoad) window.finishMapLoad = () => oninit?.(map!); else oninit?.(map!);');
                    }
                    if (realMapResults && id === source) component = component.replace('onzoom={onMapRender}', 'onzoom={(...args) => { (window.fixture.renders ??= []).push(args); onMapRender(...args); }}').replace('onviewport={onViewport}', 'onviewport={(...args) => { window.fixture.publications.push(args); onViewport(...args); }}');
                }
                else return;
                return compile(component, { filename: id, generate: 'client', dev, css: 'injected' }).js.code;
            },
        }],
        build: { write: false, minify: false, rolldownOptions: { input: 'fixture:entry', output: { format: 'iife' } } },
    });
    if (Array.isArray(result) || !('output' in result)) throw new Error('Expected a single bundle');
    const chunk = result.output.find(output => output.type === 'chunk');
    if (!chunk) throw new Error('Missing fixture bundle');
    const css = result.output.flatMap(output => output.type === 'asset' && output.fileName.endsWith('.css') ? [String(output.source)] : []).join('\n');
    const bundle = `document.head.appendChild(Object.assign(document.createElement('style'), { textContent: ${JSON.stringify(css)} }));\n${chunk.code}`;
    if (cacheFile) { await mkdir(cache!, { recursive: true }); await writeFile(cacheFile, bundle); }
    return bundle;
}

export async function mountMapPageFixture(page: Page, bundle: string, options: { query?: string; storage?: Record<string, unknown>; state?: Record<string, unknown> } = {}) {
    page.on('pageerror', error => { throw error; });

    await page.route('http://search-page.test/**', route => route.fulfill({ contentType: 'text/html', body: '<!doctype html><html><body></body></html>' }));
    await page.goto(`http://search-page.test/map?${options.query ?? ''}`);

    await page.evaluate(storage => {
        for (const [key, value] of Object.entries(storage)) localStorage.setItem(key, typeof value === 'string' ? value : JSON.stringify(value));
    }, options.storage ?? {});
    if (options.state) await page.evaluate(state => history.replaceState(state, ''), options.state);

    await page.addScriptTag({ content: bundle });
    await expect.poll(() => page.evaluate(() => (window as any).fixture.requests.length)).toBe(2);
}
