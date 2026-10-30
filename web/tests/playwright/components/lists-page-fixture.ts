import { readFile } from 'node:fs/promises';
import { extname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { compile } from 'svelte/compiler';
import { build } from 'vite';
import { expect, type Page } from '@playwright/test';

const root = fileURLToPath(new URL('../../../', import.meta.url));

// Exercise the real list page's request, loading and layout lifecycle.
export async function buildListsPageFixture(dev: boolean) {
    const result = await build({
        root, configFile: false, logLevel: 'error',
        resolve: { conditions: ['browser', dev ? 'development' : 'production'] },
        plugins: [{
            name: 'lists-page-fixture', enforce: 'pre',
            resolveId(id) {
                if (id.startsWith('fixture:')) return `\0${id}`;
                if (id === '$app/state') return '\0fixture:state';
                if (id === '$app/navigation') return '\0fixture:navigation';
                if (id === '$app/env') return '\0fixture:environment';
                if (id === 'svelte-i18n') return '\0fixture:i18n';
                if (id.startsWith('#lib/')) return resolve(root, 'src/lib', id.slice(5) + (extname(id) ? '' : '.ts'));
            },
            async load(id) {
                if (id === '\0fixture:entry') return `
                    import { mount } from 'svelte';
                    import Page from ${JSON.stringify(resolve(root, 'src/routes/lists/[[handle]]/[[id]]/+page.svelte'))};
                    import { page, data } from 'fixture:state';
                    window.fixture = { requests: [] };
                    window.fetch = (url, options) => new Promise(resolve => window.fixture.requests.push({
                        body: JSON.parse(options.body), resolve: value => resolve(Response.json(value))
                    }));
                    mount(Page, { target: document.body, props: { data } });
                `;
                if (id === '\0fixture:state') return `
                    export const data = { lists: { items: Array.from({ length: 30 }, (_, i) => ({ id: 'initial-' + i, name: 'Initial list ' + i })), page: 1, totalPages: 3 } };
                    export const page = { params: window.fixtureSelected ? { handle: "tester", id: "initial-0" } : {}, data: { filter: { q: '', author: '', public: true, private: true, shared: true, sort: 'name', sortOrder: '+' } } };
                `;
                if (id === '\0fixture:navigation') return 'export function goto() {}';
                if (id === '\0fixture:environment') return 'export const browser = true;';
                if (id === '\0fixture:i18n') return `import { writable } from 'svelte/store'; export const _ = writable(key => key);`;
                if (id === resolve(root, 'src/lib/stores/list_store.ts')) return `
                    export async function lists_search_filter(filter, page, _size, fetcher = fetch) {
                        return (await fetcher('/list-results', { method: 'POST', body: JSON.stringify({ filter, page }) })).json();
                    }
                    export function lists_show() {} export function lists_delete() {}
                `;
                if (id === resolve(root, 'src/lib/stores/trail_store.ts')) return 'export async function trails_show() { return { id: "trail", author: "tester", expand: {} }; }';
                let source: string;
                if (id === resolve(root, 'src/lib/components/list/list_search_bar.svelte')) source = `
                    <script>
                        let { filter = $bindable(), onapply } = $props();
                        window.fixture.apply = change => { filter = { ...filter, ...change }; onapply(filter); };
                    </script>
                    <div data-search-root style="height: 64px">Search lists</div>
                `;
                else if (id === resolve(root, 'src/lib/components/list/list_card.svelte')) source = '<script>let { list } = $props();</script><div data-testid="list-card" style="height: 72px">{list.name}</div>';
                else if (id === resolve(root, 'src/lib/components/trail/map_with_elevation_maplibre.svelte')) source = '<script>let { map = $bindable() } = $props(); map = { resize() {}, flyTo() {} }; export function unHighlightTrail() {}</script><div>Map</div>';
                else if (id === resolve(root, 'src/lib/components/list/list_panel.svelte')) source = '<script>let { onclick } = $props();</script><div data-testid="list-panel">Selected list<button onclick={() => onclick({ trail: { id: "trail", expand: { author: { preferred_username: "tester" } } } })}>Open trail</button></div>';
                else if (id === resolve(root, 'src/lib/components/trail/trail_info_panel.svelte')) source = '<script>let { onback } = $props();</script><button onclick={onback}>Back to list</button>';
                else if (id.endsWith('.svelte') && !id.includes('/routes/')) source = '<div></div>';
                else if (id.endsWith('.svelte')) source = await readFile(id, 'utf8');
                else return;
                return compile(source, { filename: id, generate: 'client', dev, css: 'injected' }).js.code;
            },
        }],
        build: { write: false, minify: false, rolldownOptions: { input: 'fixture:entry', output: { format: 'iife' } } },
    });
    if (Array.isArray(result) || !('output' in result)) throw new Error('Expected one bundle');
    const chunk = result.output.find(output => output.type === 'chunk');
    if (!chunk) throw new Error('Missing list fixture');
    return chunk.code;
}

export async function mountListsPageFixture(page: Page, bundle: string, selected = false) {
    await page.route('http://lists-page.test/**', route => route.fulfill({ contentType: 'text/html', body: `<!doctype html><html><head><style>
        * { box-sizing: border-box; } body { margin: 0; } main { display: grid; gap: 12px; } #list-container { overflow-y: auto; }
    </style></head><body></body></html>` }));
    await page.goto('http://lists-page.test/');
    await page.evaluate(selected => (window as any).fixtureSelected = selected, selected);
    await page.addScriptTag({ content: bundle });
    await expect(page.getByTestId(selected ? 'list-panel' : 'list-card')).toHaveCount(selected ? 1 : 30);
    await expect(page.locator('.list-results')).toHaveAttribute('aria-busy', 'false');
}
