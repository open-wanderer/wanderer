import { readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { extname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { compile } from 'svelte/compiler';
import { build } from 'vite';
import { expect, type Page } from '@playwright/test';
import type { TrailSearchFilterSections } from '../../../src/lib/models/trail_search_settings';

const root = fileURLToPath(new URL('../../../', import.meta.url));

// Bundle the actual search, advanced filters, controls and place editor.
// Only application stores and provider calls are replaced.
export async function buildTrailSearchFixture(dev: boolean) {
    const result = await build({
        root, configFile: false, logLevel: 'error',
        resolve: { conditions: ['browser', dev ? 'development' : 'production'] },
        plugins: [{
            name: 'trail-search-fixture', enforce: 'pre',
            resolveId(id) {
                if (id.startsWith('fixture:')) return `\0${id}`;
                if (id === '$app/state') return '\0fixture:state';
                if (id === 'svelte-i18n') return '\0fixture:i18n';
                if (id.startsWith('$lib/')) {
                    const path = resolve(root, 'src/lib', id.slice(5));
                    return extname(id) ? path : [path + '.ts', path + '.js'].find(existsSync);
                }
            },
            async load(id) {
                if (id === '\0fixture:entry') return `
                    import { mount } from 'svelte';
                    import Fixture from 'fixture:search.svelte';
                    window.searchApplications = [];
                    mount(Fixture, { target: document.body });
                    document.body.dataset.searchFixtureReady = 'true';
                `;
                if (id === '\0fixture:state') return 'export const page = { data: { settings: { unit: "metric" } } };';
                if (id === '\0fixture:i18n') return `
                    import { writable } from 'svelte/store';
                    import translations from ${JSON.stringify(resolve(root, 'src/lib/i18n/locales/en.json'))};
                    export const locale = writable('en');
                    export const _ = writable((key, options) => (key === "trail" ? (options?.values?.n === 1 ? "Trail" : "Trails") : translations[key] ?? key).replace(/\\{(\\w+)\\}/g, (_, name) => options?.values?.[name] ?? name));
                `;
                if (id === resolve(root, 'src/lib/stores/category_preference_store.ts')) return `import { writable } from 'svelte/store'; export const categoryPreferences = writable([]);`;
                if (id === resolve(root, 'src/lib/stores/subcategory_preference_store.ts')) return `import { writable } from 'svelte/store'; export const subcategoryPreferences = writable([]);`;
                if (id === resolve(root, 'src/lib/stores/user_store.ts')) return `import { writable } from 'svelte/store'; export const currentUser = writable({ id: 'tester', actor: 'tester' });`;
                if (id === resolve(root, 'src/lib/stores/tag_store.ts')) return `
                    export async function tags_index(q, fetcher = fetch) {
                        const response = await fetcher('/fixture/tags?q=' + encodeURIComponent(q));
                        if (!response.ok) throw new Error('Tag lookup failed');
                        return response.json();
                    }
                `;
                if (id === resolve(root, 'src/lib/stores/toast_store.svelte')) return 'export function show_toast() {}';
                if (id === resolve(root, 'src/lib/stores/search_store.ts')) return `
                    export async function searchLocations(q, limit, fetcher = fetch) {
                        const response = await fetcher('/fixture/geocode?q=' + encodeURIComponent(q) + '&limit=' + limit);
                        if (!response.ok) throw new Error('Geocoding failed');
                        return response.json();
                    }
                    export async function searchLists(q, options, fetcher = fetch) {
                        const response = await fetcher('/api/v1/search/lists', { method: 'POST', body: JSON.stringify({ q, options }) });
                        if (!response.ok) throw new Error('List lookup failed');
                        return (await response.json()).hits;
                    }
                    export async function searchActorById(id, fetcher = fetch, signal) {
                        return (await fetcher('/fixture/author?id=' + encodeURIComponent(id), { signal })).json();
                    }
                    export async function searchActors(q, includeSelf, fetcher = fetch) {
                        return (await fetcher('/fixture/authors?q=' + encodeURIComponent(q))).json();
                    }
                `;
                let source: string;
                if (id === '\0fixture:search.svelte') source = `
                    <script>
                        import Search from '$lib/components/trail/trail_search_bar.svelte';
                        const categories = [
                            { id: 'hiking', name: 'Hiking', icon: 'fa-person-hiking' },
                            { id: 'walking', name: 'Walking', icon: 'fa-person-walking' },
                        ];
                        const subcategories = [{ id: 'alpine', category: 'hiking', name: 'Alpine hiking' }];
                        const defaults = { q: '', category: [], subcategory: [], tags: [], difficulty: [0, 1, 2],
                            author: '', public: true, private: true, shared: true, liked: false,
                            near: { radius: 2000 }, distanceMin: 0, distanceMax: 50005, distanceLimit: 50005,
                            elevationGainMin: 0, elevationGainMax: 2000, elevationGainLimit: 2000,
                            elevationLossMin: 0, elevationLossMax: 2000, elevationLossLimit: 2000,
                            sort: 'created', sortOrder: '+' };
                        let filter = $state({ ...structuredClone(defaults), author: window.searchFixtureAuthor ?? defaults.author });
                        window.setFixtureSort = (sort, sortOrder) => { filter = { ...filter, sort, sortOrder }; };
                        $effect(() => { window.appliedSearchFilter = JSON.parse(JSON.stringify(filter)); });
                        function applied(filter, resetBounds) {
                            window.searchApplications.push(JSON.parse(JSON.stringify({ filter, resetBounds })));
                        }
                    </script>
                    <main><Search {categories} {subcategories} {defaults} layout={window.searchFixtureLayout ?? "overlay"} bind:filter onapply={applied}
                        filterSections={window.searchFixtureSections}
                        inlineControls={Boolean(window.searchFixtureInline)}
                        bounds={window.searchFixtureBounds} /></main>
                    <button type="button" data-testid="outside">Outside</button>
                `;
                else if (id.endsWith('.svelte')) source = await readFile(id, 'utf8');
                else return;
                return compile(source, { filename: id, generate: 'client', dev, css: 'injected' }).js.code;
            },
        }],
        build: { write: false, minify: false, rolldownOptions: { input: 'fixture:entry', output: { format: 'iife' } } },
    });
    if (Array.isArray(result) || !('output' in result)) throw new Error('Expected a single bundle');
    const chunk = result.output.find(output => output.type === 'chunk');
    if (!chunk) throw new Error('Missing fixture bundle');
    const css = result.output.flatMap(output => output.type === 'asset' && output.fileName.endsWith('.css') ? [output.source] : [])
        .join('\n');
    return `document.head.appendChild(Object.assign(document.createElement('style'), { textContent: ${JSON.stringify(css)} }));\n${chunk.code}`;
}

export async function mountTrailSearchFixture(page: Page, bundle: string, bounds?: { northEast: { lat: number; lng: number }; southWest: { lat: number; lng: number } }, options: { inline?: boolean; sections?: TrailSearchFilterSections; author?: string; layout?: "overlay" | "push" | "sidebar" } = {}) {
    const state = { authorLookups: [] as string[], listRequests: [] as { q: string; options: Record<string, unknown> }[], listHits: new Map<string, { id: string; name: string; author_name: string; domain?: string }[]>(), listFailures: new Set<string>(), listCompleted: [] as string[], listGates: new Map<string, Promise<void>>(), requests: [] as string[], multiRequests: [] as string[], failures: new Set<string>(), empty: new Set<string>(), authorRequests: [] as string[], tagRequests: [] as string[], completed: [] as string[], gates: new Map<string, Promise<void>>() };
    await page.route('http://trail-search.test/**', async route => {
        const url = new URL(route.request().url());
        if (url.pathname === '/fixture/geocode') {
            const query = url.searchParams.get('q')!;
            state.requests.push(query);
            await state.gates.get(query);
            if (state.failures.has(query)) {
                await route.fulfill({ status: 503, json: { message: 'Unavailable' } });
                state.completed.push(query);
                return;
            }
            await route.fulfill({ json: (state.empty.has(query) ? [] : ['Switzerland', 'Germany', 'Austria', 'France', 'Italy']).map((description, index) => ({ name: query, description, lat: 47.37 + index, lon: 8.54 + index, category: 'place', type: 'city' })).slice(0, Number(url.searchParams.get('limit') ?? 5)) });
            state.completed.push(query);
        } else if (url.pathname === '/api/v1/search/lists') {
            const request = route.request().postDataJSON();
            state.listRequests.push(request);
            await state.listGates.get(request.q);
            await route.fulfill(state.listFailures.has(request.q)
                ? { status: 503, json: { message: 'Unavailable' } }
                : { json: { hits: (state.listHits.get(request.q) ?? []).slice(0, request.options.limit) } });
            state.listCompleted.push(request.q);
        } else if (url.pathname === '/api/v1/search/multi') {
            state.multiRequests.push(route.request().postData() ?? '');
            await route.fulfill({ json: { results: [] } });
        } else if (url.pathname === '/fixture/author') {
            const id = url.searchParams.get('id')!;
            state.authorLookups.push(id);
            await route.fulfill({ body: JSON.stringify(id === 'anna-local'
                ? { id, preferred_username: 'anna', is_local: true, domain: '' }
                : id === 'anna-remote' ? { id, preferred_username: 'anna', is_local: false, domain: 'example.org' } : null), contentType: 'application/json' });
        } else if (url.pathname === '/fixture/authors') {
            const query = url.searchParams.get('q')!;
            state.authorRequests.push(query);
            await state.gates.get(query);
            const authors = [
                { id: 'anna-local', username: 'Anna', preferred_username: 'anna', is_local: true, domain: '' },
                { id: 'anna-remote', username: 'Anna', preferred_username: 'anna', is_local: false, domain: 'example.org' },
            ];
            await route.fulfill({ json: authors.filter(author => !query || ('anna@example.org').includes(query)) });
        } else if (url.pathname === '/fixture/tags') {
            const query = url.searchParams.get('q')!;
            state.tagRequests.push(query);
            await state.gates.get(query);
            await route.fulfill({ json: { items: ['alpin', 'alpine hiking', 'forest'].filter(name => name.includes(query)).map(name => ({ name })) } });
        } else await route.fulfill({ contentType: 'text/html', body: `<!doctype html><html><head><style>
            :root { --content: 30,40,35; --background: 255,255,255; --input-background: 250,250,248;
                --menu-background: 255,255,255; --input-border: 210,215,210; --input-border-focus: 50,100,75;
                --separator: 220,225,220; --secondary-hover: 235,240,235; }
            * { box-sizing: border-box; } body { font-family: sans-serif; margin: 0; padding: 24px; }
            main { max-width: 900px; margin: 0 auto; } button { background: transparent; color: inherit; }
            #outside { margin-top: 650px; }
        </style></head><body></body></html>` });
    });
    await page.goto('http://trail-search.test/');
    if (bounds) await page.evaluate(value => { (window as any).searchFixtureBounds = value; }, bounds);
    if (options.inline) await page.evaluate(() => { (window as any).searchFixtureInline = true; });
    if (options.sections) await page.evaluate(value => { (window as any).searchFixtureSections = value; }, options.sections);
    await page.evaluate(({ author, layout }) => {
        (window as any).searchFixtureAuthor = author;
        (window as any).searchFixtureLayout = layout;
    }, options);
    await page.addScriptTag({ content: bundle });
    await expect(page.locator('body[data-search-fixture-ready="true"]')).toBeAttached();
    return state;
}
