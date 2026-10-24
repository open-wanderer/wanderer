import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { basename, extname, resolve } from 'node:path';
import { compile } from 'svelte/compiler';
import { build } from 'vite';
import { expect, type Page } from '@playwright/test';

const root = fileURLToPath(new URL('../../../', import.meta.url));

// Keep the actual TrailList, result controls and pagination; replace only trail
// presentations and the parent data request so each callback can be observed.
export async function buildTrailResultControlsFixture(dev: boolean) {
    const result = await build({
        root, configFile: false, logLevel: 'error',
        resolve: { conditions: ['browser', dev ? 'development' : 'production'] },
        plugins: [{
            name: 'trail-result-controls-fixture', enforce: 'pre',
            resolveId(id) {
                if (id.startsWith('fixture:')) return `\0${id}`;
                if (id === 'svelte-i18n') return '\0fixture:i18n';
                if (id.startsWith('$lib/')) return resolve(root, 'src/lib', id.slice(5) + (extname(id) ? '' : '.ts'));
            },
            async load(id) {
                if (id === '\0fixture:entry') return `
                    import { mount } from 'svelte';
                    import Fixture from 'fixture:results.svelte';
                    window.resultEvents = [];
                    mount(Fixture, { target: document.body });
                    document.body.dataset.resultFixtureReady = 'true';
                `;
                if (id === '\0fixture:i18n') return `export const _ = { subscribe(fn) { fn(key => key); return () => {}; } };`;
                let source: string;
                if (id === '\0fixture:results.svelte') source = `
                    <script>
                        import { onMount } from 'svelte';
                        import TrailList from '$lib/components/trail/trail_list.svelte';
                        const incremental = localStorage.getItem('fixtureIncremental') === 'true';
                        let filter = $state({ q: localStorage.getItem('fixtureQuery') ?? '', sort: localStorage.getItem('fixtureSort') ?? 'name', sortOrder: '+' });
                        let trails = $state([]);
                        let pagination = $state({ page: incremental ? 1 : 3, totalPages: 10, items: incremental ? 48 : 12 });
                        let loading = $state(true);
                        let display = $state('cards');
                        let wide = $state(false);
                        let searchPosition = $state('top');
                        let searchExpansion = $state('overlay');
                        const searchLayout = localStorage.getItem('fixtureSearchLayout') === 'true';
                        let widths = $state(JSON.parse(localStorage.getItem('trailWidthPreferences') ?? '{}'));
                        $effect(() => { window.resultSnapshot = JSON.parse(JSON.stringify({ filter, pagination, display, wide, searchPosition, searchExpansion })); });
                        function record(type, extra = {}) { window.resultEvents.push(JSON.parse(JSON.stringify({ type, ...extra }))); }
                        async function load(page, items) {
                            loading = true;
                            const query = new URLSearchParams({ page, items, sort: filter.sort, order: filter.sortOrder });
                            const response = await (await fetch('/fixture/trails?' + query)).json();
                            pagination = { page, totalPages: response.totalPages, items };
                            trails = incremental && page > 1 ? [...trails, ...response.items] : response.items;
                            loading = false;
                        }
                        function update() { record('sort', { sort: filter.sort, order: filter.sortOrder }); load(pagination.page, pagination.items); }
                        function paginate(page, items) { record('page', { page, items }); load(page, items); }
                        function displayChanged(value) { display = value; wide = widths[value] ?? false; record('display', { display: value }); }
                        function widthChanged() {
                            wide = !wide;
                            widths = { ...widths, [display]: wide };
                            localStorage.setItem('trailWidthPreferences', JSON.stringify(widths));
                            record('width', { display, wide });
                        }
                        onMount(() => { if (incremental) load(1, 48); });
                    </script>
                    <main><TrailList bind:filter bind:trails bind:pagination bind:display {loading} compactControls
                        managed={incremental} {incremental} mapView={incremental ? mapResults : undefined} moreResults={incremental ? moreResults : undefined}
                        fullWidth={wide} onwidthtoggle={widthChanged} onupdate={update} onpagination={paginate} ondisplaychange={displayChanged}
                        searchPosition={display === 'map' ? 'top' : searchPosition} {searchExpansion}
                        onsearchposition={searchLayout && display !== 'map' ? value => searchPosition = value : undefined}
                        onsearchexpansion={searchLayout && (searchPosition === 'top' || display === 'map') ? value => searchExpansion = value : undefined} /></main>
                    {#snippet mapResults()}<div data-testid="map-view">Map</div>{/snippet}
                    {#snippet moreResults()}{#if pagination.page < pagination.totalPages}<button onclick={() => load(pagination.page + 1, 48)}>Load more</button>{/if}{/snippet}
                    <button type="button" data-testid="outside">Outside</button>
                `;
                else if (!id.endsWith('.svelte')) return;
                else switch (basename(id)) {
                    case 'trail_card.svelte':
                    case 'trail_list_item.svelte':
                        source = `<script>let { trail } = $props();</script><span data-testid="trail-id" data-view="${basename(id) === 'trail_card.svelte' ? 'cards' : 'list'}">{trail.id}</span>`;
                        break;
                    case 'trail_table.svelte':
                        source = '<script>let { trails } = $props();</script><div data-testid="table-view">{#each trails ?? [] as trail}<span data-testid="trail-id">{trail.id}</span>{/each}</div>';
                        break;
                    case 'skeleton_card.svelte':
                    case 'skeleton_list_item.svelte': source = '<span data-testid="loading"></span>'; break;
                    case 'empty_state_search.svelte': source = '<span data-testid="empty"></span>'; break;
                    case 'trail_dropdown.svelte': source = '<span data-testid="selection-actions"></span>'; break;
                    default: source = await readFile(id, 'utf8');
                }
                return compile(source, { filename: id, generate: 'client', dev, css: 'injected' }).js.code;
            },
        }],
        build: { write: false, minify: false, rolldownOptions: { input: 'fixture:entry', output: { format: 'iife' } } },
    });
    if (Array.isArray(result) || !('output' in result)) throw new Error('Expected a single bundle');
    const chunk = result.output.find(output => output.type === 'chunk');
    if (!chunk) throw new Error('Missing fixture bundle');
    return chunk.code;
}

export async function mountTrailResultControlsFixture(page: Page, bundle: string, options: {
    display?: string; items?: number; sort?: string; order?: '+' | '-'; widths?: Record<string, boolean>; incremental?: boolean; query?: string; searchLayout?: boolean;
} = {}) {
    const requests: { page: number; items: number; sort: string; order: string }[] = [];
    await page.route('http://trail-results.test/**', async route => {
        const url = new URL(route.request().url());
        if (url.pathname === '/fixture/trails') {
            const page = Number(url.searchParams.get('page'));
            const items = Number(url.searchParams.get('items'));
            requests.push({ page, items, sort: url.searchParams.get('sort')!, order: url.searchParams.get('order')! });
            await route.fulfill({ json: { totalPages: Math.ceil(120 / items), items: Array.from({ length: Math.min(items, 120 - (page - 1) * items) }, (_, index) => ({ id: String((page - 1) * items + index + 1), author: 'tester' })) } });
        } else await route.fulfill({ contentType: 'text/html', body: `<!doctype html><html><head><style>
            :root { --content: 30,40,35; --input-background: 250,250,248; --menu-background: 255,255,255;
                --input-border: 210,215,210; --input-border-focus: 50,100,75; --separator: 220,225,220; --secondary-hover: 235,240,235; }
            * { box-sizing: border-box; } body { font-family: sans-serif; margin: 0; padding: 24px; }
            main { max-width: 900px; margin: 0 auto; } button { background: transparent; color: inherit; }
            [data-testid="outside"] { display: block; margin-top: 30px; }
        </style></head><body></body></html>` });
    });
    await page.goto('http://trail-results.test/');
    await page.evaluate(({ display, items, sort, order, widths, incremental, query, searchLayout }) => {
        localStorage.clear();
        localStorage.setItem('displayOption', display ?? 'cards');
        localStorage.setItem('paginationItems', String(items ?? 12));
        localStorage.setItem('sort', sort ?? 'name');
        localStorage.setItem('sort_order', order ?? '+');
        localStorage.setItem('trailWidthPreferences', JSON.stringify(widths ?? {}));
        localStorage.setItem('fixtureSearchLayout', String(searchLayout ?? false));
        localStorage.setItem('fixtureQuery', query ?? '');
        localStorage.setItem('fixtureSort', sort ?? 'name');
        localStorage.setItem('fixtureIncremental', String(incremental ?? false));
    }, options);
    await page.addScriptTag({ content: bundle });
    await expect(page.getByTestId('trail-id')).toHaveCount(options.incremental ? 48 : options.items ?? 12);
    return requests;
}
