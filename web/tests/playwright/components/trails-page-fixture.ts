import { readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { extname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { compile, compileModule } from 'svelte/compiler';
import { build } from 'vite';
import { expect, type Page } from '@playwright/test';

const root = fileURLToPath(new URL('../../../', import.meta.url));

// Keep the real list-page orchestration; resolve requests in any order,
// including after cancellation, without mounting a map or querying its viewport.
export async function buildTrailsPageFixture(dev: boolean) {
    const result = await build({
        root, configFile: false, logLevel: 'error',
        resolve: { conditions: ['browser', dev ? 'development' : 'production'] },
        plugins: [{
            name: 'trails-page-fixture', enforce: 'pre',
            resolveId(id) {
                if (id.startsWith('fixture:')) return `\0${id}`;
                if (id === '$app/env') return '\0fixture:environment';
                if (id === '$app/navigation') return '\0fixture:navigation';
                if (id === '$app/state') return '\0fixture:state.svelte.js';
                if (id === 'svelte-i18n') return '\0fixture:i18n';
                if (id.startsWith('#lib/')) {
                    const path = resolve(root, 'src/lib', id.slice(5));
                    return extname(id) ? path : [path + '.ts', path + '.js'].find(existsSync);
                }
            },
            async load(id) {
                if (id === '\0fixture:entry') return `
                    import { mount, unmount } from 'svelte';
                    import Page from ${JSON.stringify(resolve(root, 'src/routes/trails/+page.svelte'))};
                    window.fixture = { requests: [] };
                    window.fetch = (url, options) => new Promise((resolve, reject) => {
                        window.fixture.requests.push({ channel: url, body: JSON.parse(options.body), signal: options.signal,
                            resolve: data => resolve(Response.json(data)), reject });
                    });
                    const component = mount(Page, { target: document.body });
                    window.fixture.capture = () => component.snapshot.capture();
                    window.fixture.restore = value => component.snapshot.restore(value);
                    window.fixture.destroy = () => unmount(component);
                `;
                if (id === '\0fixture:environment') return 'export const browser = true;';
                if (id === '\0fixture:state.svelte.js') return compileModule(`
                    const params = new URL(location.href).searchParams;
                    export const page = $state({ url: new URL(location.href), state: {}, data: {
                        categories: [], subcategories: [], settings: {},
                        filter: { q: '', category: params.has('category') ? [params.get('category')] : [],
                            subcategory: params.has('subcategory') ? [params.get('subcategory')] : [], tags: [], difficulty: [0, 1, 2],
                            author: params.get('author') ?? '', public: true, private: true, shared: params.has('author') ? undefined : true,
                            near: { radius: 2000 }, distanceMin: 0, distanceMax: 50000, distanceLimit: 50000,
                            elevationGainMin: 0, elevationGainMax: 2000, elevationGainLimit: 2000,
                            elevationLossMin: 0, elevationLossMax: 2000, elevationLossLimit: 2000,
                            sort: 'relevance', sortOrder: '+' }
                    } });
                `, { filename: 'fixture-state.svelte.js', generate: 'client', dev }).js.code;
                if (id === '\0fixture:navigation') return `
                    export function beforeNavigate(callback) { window.fixture.beforeNavigate = callback; }
                `;
                if (id === '\0fixture:i18n') return `import { writable } from 'svelte/store'; export const _ = writable(key => key);`;
                if (id === resolve(root, 'src/lib/stores/trail_store.ts')) return `
                    async function query(channel, body, fetcher) {
                        const response = await fetcher(channel, { method: 'POST', body: JSON.stringify(body) });
                        if (!response.ok) throw new Error('Search failed');
                        return response.json();
                    }
                    export function trails_search_filter(filter, page, perPage, fetcher, bounds) {
                        return query('results', { filter, page, perPage, bounds }, fetcher);
                    }
                    export async function trails_search_filter_pages(filter, pages, perPage, fetcher) {
                        const result = await query('results', { filter, page: 1, perPage: Math.min(pages * perPage, 1000) }, fetcher);
                        return { ...result, page: Math.max(1, Math.ceil(result.items.length / perPage)), totalPages: Math.ceil(Math.min(result.totalHits, 1000) / perPage) };
                    }
                `;
                let component: string;
                if (id === resolve(root, 'src/lib/components/trail/trail_list.svelte')) component = `
                    <script>
                        let { filter = $bindable(), trails, pagination, display = $bindable(), loading,
                            onupdate, onresultsupdate, ondisplaychange, searchBar, moreResults, busy } = $props();
                        $effect(() => {
                            window.fixture.refreshResults = onresultsupdate;
                            window.fixture.display = mode => { display = mode; ondisplaychange(mode); };
                            window.fixture.sort = (sort, sortOrder) => { filter = { ...filter, sort, sortOrder }; onupdate(); };
                        });
                    </script>
                    {@render searchBar()}
                    <div class="result-content" aria-busy={busy ?? loading}>
                        <output data-testid="results">{JSON.stringify({ trails, pagination, filter, loading, display })}</output>
                        {#each trails as trail}<div style="height: 20px">{trail.id}</div>{/each}
                    </div>
                    {@render moreResults()}
                `;
                else if (id === resolve(root, 'src/lib/components/trail/trail_search_bar.svelte')) component = `
                    <script>
                        let { filter = $bindable(), onapply } = $props();
                        $effect(() => { window.fixture.apply = change => { filter = { ...filter, ...change }; onapply(filter); }; });
                        export function restoreFilter() {}
                    </script>
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

export async function mountTrailsPageFixture(page: Page, bundle: string, options: { query?: string; storage?: Record<string, unknown> } = {}) {
    await page.route('http://trails-page.test/**', route => route.fulfill({ contentType: 'text/html', body: '<!doctype html><html><body></body></html>' }));
    await page.goto(`http://trails-page.test/trails?${options.query ?? ''}`);
    await page.evaluate(storage => {
        for (const [key, value] of Object.entries(storage)) localStorage.setItem(key, typeof value === 'string' ? value : JSON.stringify(value));
    }, options.storage ?? {});
    await page.addScriptTag({ content: bundle });
    await expect.poll(() => page.evaluate(() => (window as any).fixture.requests.length)).toBe(1);
}
